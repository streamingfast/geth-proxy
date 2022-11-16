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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/eth-go/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestMarshalCallArgs(t *testing.T) {
	codec := NewEthereumCodec()

	type args struct {
		call CallArgs
	}
	tests := []struct {
		name      string
		in        string
		want      CallArgs
		assertion require.ErrorAssertionFunc
	}{
		{
			"call args with block latest",
			`{"params":[{"data":"0x","to":"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"},"latest"],"jsonrpc":"2.0"}`,
			CallArgs{
				Object: callParams{
					To:   eth.MustNewAddress("0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"),
					Data: eth.Hex{},
				},
				BlockRef: rpc.LatestBlock,
			},
			require.NoError,
		},
		{
			"call args with block num",
			`{"params":[{"data":"0x","to":"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"},"0xab"],"jsonrpc":"2.0"}`,
			CallArgs{
				Object: callParams{
					To:   eth.MustNewAddress("0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"),
					Data: eth.Hex{},
				},
				BlockRef: rpc.BlockNumber(171),
			},
			require.NoError,
		},
		{
			"call args with block hash",
			`{"params":[{"data":"0x","to":"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"},{"blockHash":"0x8274eae990cb45d881ab73c10f851fb406501922843b02fa066abde3403d25ab"}],"jsonrpc":"2.0"}`,
			CallArgs{
				Object: callParams{
					To:   eth.MustNewAddress("0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"),
					Data: eth.Hex{},
				},
				BlockRef: rpc.BlockHash("0x8274eae990cb45d881ab73c10f851fb406501922843b02fa066abde3403d25ab"),
			},
			require.NoError,
		},
		{
			"batch requests",
			`[{"params":[{"data":"0x","to":"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"},"latest"],"jsonrpc":"2.0"},
				{"params":[{"data":"0x","to":"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"},"latest"],"jsonrpc":"2.0"}]`,
			CallArgs{
				Object: callParams{
					To:   eth.MustNewAddress("0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"),
					Data: eth.Hex{},
				},
				BlockRef: rpc.LatestBlock,
			},
			require.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := codec.NewRequest(NewJSONRPCRequest(tt.in))

			var call CallArgs
			err := request.ReadRequest(0, &call)

			tt.assertion(t, err)
			assert.Equal(t, tt.want.BlockRef, call.BlockRef)
		})
	}
}

func TestBatch(t *testing.T) {
	codec := NewEthereumCodec()

	type args struct {
		call CallArgs
	}
	tests := []struct {
		name      string
		in        string
		numOfReqs int
		want      CallArgs
		assertion require.ErrorAssertionFunc
	}{
		{
			"Batch of one request in an array",
			`[{"params":[{"data":"0x","to":"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"},"latest"],"jsonrpc":"2.0"}]`,
			1,
			CallArgs{
				Object: callParams{
					To:   eth.MustNewAddress("0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"),
					Data: eth.Hex{},
				},
				BlockRef: rpc.LatestBlock,
			},
			require.NoError,
		},
		{
			"Batch of two requests in the same array",
			`[{"params":[{"data":"0x","to":"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"},"latest"],"jsonrpc":"2.0"},
				{"params":[{"data":"0x","to":"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"},"latest"],"jsonrpc":"2.0"}]`,
			2,
			CallArgs{
				Object: callParams{
					To:   eth.MustNewAddress("0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"),
					Data: eth.Hex{},
				},
				BlockRef: rpc.LatestBlock,
			},
			require.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := codec.NewRequest(NewJSONRPCRequest(tt.in))

			var call CallArgs
			for i := 0; i < tt.numOfReqs; i++ {
				err := request.ReadRequest(i, &call)
				tt.assertion(t, err)
				assert.Equal(t, tt.want.BlockRef, call.BlockRef)
			}
		})
	}
}

type StructFields []interface{}

func (f *StructFields) UnmarshalJSON(data []byte) error {
	result := gjson.ParseBytes(data)
	if !result.IsArray() {
		return fmt.Errorf("expected array but got %s", result.Type)
	}

	elementResults := result.Array()
	if len(elementResults) != len(*f) {
		return fmt.Errorf("input array has %d elements but we are trying to unserialize in only %d struct fields", len(elementResults), len(*f))
	}

	for i, elementResult := range elementResults {
		reference := (*f)[i]
		err := json.Unmarshal([]byte(elementResult.Raw), reference)
		if err != nil {
			return fmt.Errorf("unable to marshal JSON into field %d of type %T: %w", i, reference, err)
		}
	}

	return nil
}

func NewJSONRPCRequest(body string) *http.Request {
	return httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
}
