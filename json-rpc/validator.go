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

package jsonrpc

import (
	"fmt"

	"github.com/gorilla/rpc/v2"
	"go.uber.org/zap/zapcore"
)

type Validateable interface {
	Validate(requestInfo *rpc.RequestInfo) error
}

func ValidateRequest(r *rpc.RequestInfo, args interface{}) error {
	logRequest("validate", zapcore.DebugLevel, r)

	if _, ok := args.(Validateable); !ok {
		return fmt.Errorf("args of type %T must implement dweb3.Validateable interface", args)
	}

	return nil
}
