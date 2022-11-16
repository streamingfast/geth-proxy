// Copyright 2021 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the = License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"fmt"
	"net/http"

	"github.com/gorilla/rpc/v2"
	ethrpc "github.com/streamingfast/eth-go/rpc"
)

func (e *EthService) GetBlockByHash(r *http.Request, _ *GetBlockByHashArgs, _ *ethrpc.Block) error {
	return fmt.Errorf("EVM Executor does not support eth_getBlockByHash method, you are probably trying to use EVM Executor as full-blown JSON-RPC provider which is unsupported")
}

type GetBlockByHashArgs struct {
	BlockRef            *ethrpc.BlockRef `json:"blockNrOrHash"`
	IncludeTransactions bool             `json:"includeTransactions"`
}

func (a *GetBlockByHashArgs) Validate(requestInfo *rpc.RequestInfo) error {
	return nil
}
