// Copyright 2021 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jsonrpc

import (
	"context"
	"fmt"
	"github.com/gorilla/rpc/v2"
	"net/http"
	"time"

	"github.com/emiliocramer/lighthouse-geth-proxy/json-rpc/services"
	"github.com/gorilla/mux"
	"github.com/streamingfast/dhttp"
	"github.com/streamingfast/logging"
	"github.com/streamingfast/shutter"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Server struct {
	*shutter.Shutter

	httpServer     *http.Server
	httpListenAddr string
	mux            *mux.Router
}

func NewServer(
	httpListenAddr string,
	isReady func() bool,
	serviceHandlers []services.ServiceHandler,
) (*Server, error) {
	router := mux.NewRouter()
	srv := &Server{
		Shutter:        shutter.New(),
		httpListenAddr: httpListenAddr,
		mux:            router,
	}

	metricsRouter := router.PathPrefix("/").Subrouter()
	coreRouter := router.PathPrefix("/").Subrouter()

	// Health endpoints
	metricsRouter.Path("/healthz").HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if !isReady() || srv.IsTerminating() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		_, err := w.Write([]byte("ok"))
		if err != nil {
			return
		}
	})

	// Midddleware
	coreRouter.Use(dhttp.NewAddLoggerToContextMiddleware(zlog))
	coreRouter.Use(dhttp.NewLogRequestMiddleware(zlog))
	coreRouter.Use(dhttp.NewOpenCensusMiddleware())
	coreRouter.Use(dhttp.NewAddTraceIDHeaderMiddleware(zlog))

	rpcRouter := coreRouter.PathPrefix("/").Subrouter()
	rpcRouter.Use(forceContentTypeApplicationJSON)

	rpc.MethodSeparator = "_"
	rpcServer := rpc.NewServer()
	rpcServer.RegisterCodec(services.NewEthereumCodec(), "application/json")
	rpcServer.RegisterBeforeFunc(func(i *rpc.RequestInfo, args interface{}) {
		logRequest("incoming request", zapcore.DebugLevel, i, zap.Any("args", args))
	})
	rpcServer.RegisterInterceptFunc(createRequestInterceptor)
	rpcServer.RegisterValidateRequestFunc(ValidateRequest)
	rpcServer.RegisterAfterFunc(afterRequestInterceptor)

	for _, service := range serviceHandlers {
		namespace := service.Namespace()
		err := rpcServer.RegisterService(service, namespace)
		if err != nil {
			return nil, fmt.Errorf("registering service %q (service handler of type %T): %w", namespace, service, err)
		}
	}

	// The ingress forwards the full path `/call` to us, it does not strip the paths so we need to handle it directly ourself
	rpcRouter.Path("/call").Methods("POST").Handler(rpcServer)
	rpcRouter.Path("/call/{token}").Methods("POST").Handler(rpcServer)

	// The ingress forwards the full path `/json-rpc` to us, it does not strip the paths so we need to handle it directly ourself
	rpcRouter.Path("/json-rpc").Methods("POST").Handler(rpcServer)
	rpcRouter.Path("/json-rpc/{token}").Methods("POST").Handler(rpcServer)

	rpcRouter.Path("/").Methods("POST").Handler(rpcServer)
	rpcRouter.Path("/{token}").Methods("POST").Handler(rpcServer)

	srv.OnTerminating(func(_ error) {
		zlog.Info("gracefully shutting down http server, draining connections")
		if srv.httpServer != nil {
			zlog.Info("allowing server to gracefully shuts down without interrupting any active connections")
			// FIXME: Should we use graceful shutdown delay - X seconds instead?
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			srv.httpServer.Shutdown(ctx)
		}
	})

	return srv, nil
}

func (s *Server) Serve() {
	zlog.Info("listening & serving HTTP content", zap.String("http_listen_addr", s.httpListenAddr))
	errorLogger, err := zap.NewStdLogAt(zlog, zap.ErrorLevel)
	if err != nil {
		s.Shutdown(fmt.Errorf("unable to create error logger: %w", err))
		return
	}

	s.httpServer = &http.Server{
		Addr:     s.httpListenAddr,
		Handler:  s.mux,
		ErrorLog: errorLogger,
	}

	err = s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		s.Shutdown(fmt.Errorf("failed listening http %q: %w", s.httpListenAddr, err))
	}

	zlog.Info("server terminated")
}

func createRequestInterceptor(i *rpc.RequestInfo) *http.Request {
	logRequest("incoming request parsing", zapcore.DebugLevel, i)
	i.Request.Method = i.Method // puts the method in http request, what a cool way to pass it down to the handler
	return i.Request
}

func afterRequestInterceptor(i *rpc.RequestInfo) {
	logRequest("after request", zapcore.DebugLevel, i, zap.Error(i.Error), zap.Int("status_code", i.StatusCode))
}

func logRequest(msg string, level zapcore.Level, i *rpc.RequestInfo, extraFields ...zap.Field) {
	logger := logging.Logger(i.Request.Context(), zlog)
	if ce := logger.Check(level, msg); ce != nil {
		ce.Write(append([]zap.Field{zap.String("method", i.Method)}, extraFields...)...)
	}
}

func forceContentTypeApplicationJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("content-type") != "application/json" {
			r.Header.Set("content-type", "application/json")
		}
		next.ServeHTTP(w, r)
	})
}
