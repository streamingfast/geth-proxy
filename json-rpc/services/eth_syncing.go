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
	"github.com/gorilla/rpc/v2"
	"go.uber.org/zap"
	"net/http"

	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/logging"
)

type SyncingArgs struct {
}

type SyncingValidResp struct {
	startingBlock eth.Uint64
	currentBlock  eth.Uint64
	highestBlock  eth.Uint64
}

func (e *EthService) Syncing(r *http.Request, args *SyncingArgs, reply *SyncingValidResp) error {
	ctx := r.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Info("Syncing args:", zap.Reflect("args", args))
	zlogger.Info("Syncing reply:", zap.Reflect("reply", reply))

	return nil
}

func (e *SyncingArgs) Validate(requestInfo *rpc.RequestInfo) error {
	return nil
}
