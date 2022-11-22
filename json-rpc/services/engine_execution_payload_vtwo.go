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

type ExecutionPayloadV2Args struct {
	ParentHash    eth.Hash              `json:"parentHash"`
	FeeRecipient  eth.Address           `json:"feeRecipient"`
	StateRoot     eth.Hash              `json:"stateRoot"`
	ReceiptsRoot  eth.Hash              `json:"receiptsRoot"`
	LogsBloom     eth.Hex               `json:"logsBloom"`
	PrevRandao    eth.Hash              `json:"prevRandao"`
	BlockNumber   eth.Uint64            `json:"blockNumber"`
	GasLimit      eth.Uint64            `json:"gasLimit"`
	GasUsed       eth.Uint64            `json:"gasUsed"`
	Timestamp     eth.Uint64            `json:"timestamp"`
	ExtraData     eth.Hash              `json:"extraData"`
	BaseFeePerGas BigInt                `json:"baseFeePerGas"`
	BlockHash     eth.Hash              `json:"blockHash"`
	Transactions  []eth.TransactionType `json:"transactions"`
	Withdrawals   []WithdrawalV1Args    `json:"withdrawals"`
}

func (e *EngineService) ExecutePayLoadV2(r *http.Request, args *ExecutionPayloadV2Args, reply *eth.Hex) error {
	ctx := r.Context()
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Info("engine execute payload v2 args:", zap.Reflect("args", args))
	zlogger.Info("engine execute payload v2 reply:", zap.Reflect("reply", reply))

	return nil
}

func (e *ExecutionPayloadV2Args) Validate(requestInfo *rpc.RequestInfo) error {
	return nil
}
