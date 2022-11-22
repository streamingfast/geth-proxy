package main

import (
	"fmt"
	"go.uber.org/zap"
	"time"

	jsonrpc "github.com/emiliocramer/lighthouse-geth-proxy/json-rpc"
	"github.com/emiliocramer/lighthouse-geth-proxy/json-rpc/services"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/derr"
)

func init() {
	rootCmd.AddCommand(ServeJSONRPCCommand)

	ServeJSONRPCCommand.Flags().String("listen-addr-beacon", ":8080", "The port that should be listened too for incoming JSON-RPC requests")
}

var ServeJSONRPCCommand = &cobra.Command{
	Use:   "serve",
	Short: "Starts the JSON-RPC server",
	RunE:  serveJSONRPCE,
}

func serveJSONRPCE(cmd *cobra.Command, args []string) error {
	listenAddrBeacon := viper.GetString("serve-listen-addr-beacon")

	zlog.Info("starting server", zap.String("listen_addr", listenAddrBeacon))

	server, err := jsonrpc.NewServer(
		listenAddrBeacon,
		func() bool { return true },
		[]services.ServiceHandler{
			services.NewEngineService(),
			services.NewEthService(),
		},
	)

	if err != nil {
		return fmt.Errorf("creating json rpc server: %w", err)
	}

	go server.Serve()

	zlog.Info("waiting for server to terminate")

	select {
	case <-derr.SetupSignalHandler(0 * time.Second):
		server.Shutdown(nil)
		zlog.Info("signal handler terminated app, waiting for it to complete")
		<-server.Terminated()
	case <-server.Terminated():
		return server.Err()
	}

	return nil
}
