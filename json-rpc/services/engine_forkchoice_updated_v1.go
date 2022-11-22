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
	"github.com/streamingfast/eth-go"
	"go.uber.org/zap"
	"net/http"

	"github.com/streamingfast/logging"
)

type ForkchoiceUpdatedV1Args struct {
	forkchoiceState   ForkchoiceStateV1Args
	payloadAttributes PayloadAttributesV1Args
}

type ForkChoiceUpdatedV1Status string

const (
	ForkchoiceStatusValid   ForkChoiceUpdatedV1Status = "VALID"
	ForkchoiceStatusInvalid ForkChoiceUpdatedV1Status = "INVALID"
	ForkchoiceStatusSyncing ForkChoiceUpdatedV1Status = "SYNCING"
)

type ForkchoiceUpdatedV1Reply struct {
	forkchoiceStatus ForkChoiceUpdatedV1Status
	payloadId        eth.Hash
}

func (e *EngineService) ForkchoiceUpdatedV1(r *http.Request, args *ForkchoiceUpdatedV1Args, reply *ForkchoiceUpdatedV1Reply) error {
	ctx := r.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Info("Forkchoice Updated v1 args:", zap.Reflect("args", args))
	zlogger.Info("Forkchoice Updated v1 reply:", zap.Reflect("reply", reply))

	return nil
}
