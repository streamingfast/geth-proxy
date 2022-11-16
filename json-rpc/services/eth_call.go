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
	"fmt"
	"github.com/gorilla/rpc/v2"
	"github.com/streamingfast/eth-go"
	"math/big"
	"net/http"

	evmexecutor "github.com/emiliocramer/lighthouse-geth-proxy/evm-executor"
	"github.com/gorilla/rpc/v2/json2"
	"github.com/streamingfast/bstream"
	ethrpc "github.com/streamingfast/eth-go/rpc"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

type callParams struct {
	From     *eth.Address `json:"from"`
	To       eth.Address  `json:"to"`
	GasLimit eth.Uint64   `json:"gas"`
	GasPrice *BigInt      `json:"gasPrice"`
	Value    *BigInt      `json:"value"`
	Data     eth.Hex      `json:"data"`
}

func (p *callParams) toRpcParams() (out ethrpc.CallParams) {
	out.To = p.To

	if p.From != nil {
		out.From = *p.From
	}

	out.Value = (*big.Int)(p.Value)
	out.GasLimit = uint64(p.GasLimit)
	out.GasPrice = (*big.Int)(p.GasPrice)
	out.Data = p.Data

	return
}

type CallArgs struct {
	Object   callParams       `json:"object"`
	BlockRef *ethrpc.BlockRef `json:"blockNrOrHash"`
}

func (e *EthService) Call(r *http.Request, args *CallArgs, reply *eth.Hex) error {

	if args.BlockRef.IsPending() {
		return &json2.Error{
			Code:    json2.E_BAD_PARAMS,
			Message: "'blockNrOrHash' param value 'pending' is not accepted",
			Data:    nil,
		}
	}

	ctx := r.Context()

	blockRef := toBstreamBlockRef(args.BlockRef)
	zlogger := logging.Logger(ctx, zlog).With(zap.String("to", args.Object.To.Pretty()), zap.String("data", args.Object.Data.String()), zap.Stringer("block_ref", blockRef))
	zlogger.Debug("eth call", zap.Reflect("args", args))

	returnedData, gasUsed, err := e.evmExecutor.ExecuteCall(ctx, args.Object.toRpcParams(), blockRef)
	if err != nil {
		level := zap.ErrorLevel
		if isUserError(err) || isEVMDetermisticError(err) {
			level = zap.DebugLevel
		}

		zlogger.Check(level, "execute call failed").Write(zap.Error(err))
		return &json2.Error{Code: json2.E_SERVER, Message: err.Error()}
	}

	zlogger.Info("execute call succeeed", zap.Stringer("returned_data", eth.Hex(returnedData)), zap.Uint64("gas_used", gasUsed))

	*reply = returnedData
	return nil
}

func (a *CallArgs) Validate(requestInfo *rpc.RequestInfo) error {
	return nil
}

func toBstreamBlockRef(ref *ethrpc.BlockRef) bstream.BlockRef {
	if ref.IsLatest() {
		return evmexecutor.LatestBlockRef
	}

	if ref.IsEarliest() {
		return evmexecutor.EarliestBlockRef
	}

	if ref.IsPending() {
		panic(fmt.Errorf("block ref is pending but it's not accepted for conversion to bstream.BlockRef"))
	}

	if hash, ok := ref.BlockHash(); ok {
		return bstream.NewBlockRef(hash.String(), 0)
	}

	blockNumber, ok := ref.BlockNumber()
	if !ok {
		panic(fmt.Errorf("block ref is not a block number but was expected to be so"))
	}

	return bstream.NewBlockRef("", blockNumber)
}
