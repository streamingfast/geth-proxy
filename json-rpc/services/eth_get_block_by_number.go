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

package services

import (
	"net/http"

	"github.com/emiliocramer/lighthouse-geth-proxy/config"
	"github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json2"
	ethrpc "github.com/streamingfast/eth-go/rpc"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func (e *EthService) GetBlockByNumber(r *http.Request, args *GetBlockByNumberArgs, reply *ethrpc.Block) error {
	ctx := r.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Debug("eth get block by number", zap.Reflect("args", args))
	zlogger.Debug("args block ref", zap.Reflect("blockref", args.BlockRef))
	zlogger.Debug("args block include transaction", zap.Reflect("includeTransaction", args.IncludeTransactions))

	if args.BlockRef.IsPending() {
		return &json2.Error{
			Code:    json2.E_BAD_PARAMS,
			Message: "'blockNrOrHash' param value 'pending' is not accepted",
			Data:    nil,
		}
	}

	block, err := e.evmExecutor.BlockByNumber(ctx, config.ToBstreamBlockRef(args.BlockRef))
	if err != nil {
		zlogger.Error("block by number call failed", zap.Error(err))
		return &json2.Error{Code: json2.E_SERVER, Message: err.Error()}
	}

	if tracer.Enabled() {
		zlog.Debug("execute get block by number succeeded")
	}

	*reply = *block
	return nil
}

type GetBlockByNumberArgs struct {
	BlockRef            *ethrpc.BlockRef `json:"blockNrOrHash"`
	IncludeTransactions bool             `json:"includeTransactions"`
}

func (a *GetBlockByNumberArgs) Validate(requestInfo *rpc.RequestInfo) error {
	return nil
}
