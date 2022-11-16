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
	"math/big"
	"strconv"
)

// ServiceHandler is an abstraction that all of our Ethereum JSON-RPC handler implements
// and the sole method on this interface is the `Namespace()` which is the prefix of method
// this service is handling. For example `net_version` is in the namespace `net` which `eth_call`
// is in the `eth` namespace.
type ServiceHandler interface {
	// Namespace returns the namespace this service handler accepts for processing.
	Namespace() string
}

type BigInt big.Int

type NetworkID uint64

// MarshalJSONRPC ensures we serialize using the networkd id in `string` type just like
// Alchemy is doing. Without this, since we use `rpc.MarshalJSONRPC`, it by default serializes
// to an hex encoded number.
func (r *NetworkID) MarshalJSONRPC() ([]byte, error) {
	return []byte(`"` + strconv.FormatUint(uint64(*r), 10) + `"`), nil
}
