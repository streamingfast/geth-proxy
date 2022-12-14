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

type ForkchoiceStateV1Args struct {
	HeadBlockHash      eth.Hash `json:"headBlockHash"`
	SafeBlockHash      eth.Hash `json:"SafeBlockHash"`
	FinalizedBlockHash eth.Hash `json:"finalizedBlockHash"`
}

func (e *EngineService) ForkchoiceStateV1(r *http.Request, args *ForkchoiceStateV1Args, reply *eth.Hex) error {
	ctx := r.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Info("engine forkchoice state v1 args:", zap.Reflect("args", args))
	zlogger.Info("engine forkchoice state v1 reply:", zap.Reflect("reply", reply))

	return nil
}

func (e *ForkchoiceStateV1Args) Validate(requestInfo *rpc.RequestInfo) error {
	return nil
}
