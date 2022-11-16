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

type WithdrawalV1Args struct {
	Index          eth.Uint64  `json:"index"`
	ValidatorIndex eth.Uint64  `json:"validatorIndex"`
	Address        eth.Address `json:"address"`
	Amount         eth.Hex     `json:"amount"`
}

func (e *EngineService) WithdrawalV1(r *http.Request, args *WithdrawalV1Args, reply *eth.Hex) error {
	ctx := r.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Info("engine withdrawal v1:", zap.Reflect("args", args))

	*reply = eth.MustNewHex("ab")
	return nil
}

func (e *WithdrawalV1Args) Validate(requestInfo *rpc.RequestInfo) error {
	return nil
}
