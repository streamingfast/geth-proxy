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
	"github.com/streamingfast/eth-go"
	"go.uber.org/zap"
	"net/http"

	"github.com/streamingfast/logging"
)

type ChainIdArgs struct {
}

func (e *EthService) ChainId(r *http.Request, args *ChainIdArgs, reply *eth.Uint64) error {
	ctx := r.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Info("chain Id:", zap.Reflect("args", args))

	*reply = 5
	return nil
}

func (e *ChainIdArgs) Validate(requestInfo *rpc.RequestInfo) error {
	return nil
}
