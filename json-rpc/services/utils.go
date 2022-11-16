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
	"errors"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/streamingfast/derr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// isUserError determines if the error we receive was most probably a problem
// on user's side.
//
// Here some errors that are flagged as error:
// - gRPC Canceled call (most probably canceled because user's closed the connection)
func isUserError(err error) bool {
	return isGrpcErrorWithCode(err, codes.Canceled)
}

var deterministicErrs = []error{
	vm.ErrOutOfGas,
	vm.ErrCodeStoreOutOfGas,
	vm.ErrDepth,
	vm.ErrInsufficientBalance,
	vm.ErrContractAddressCollision,
	vm.ErrExecutionReverted,
	vm.ErrMaxCodeSizeExceeded,
	vm.ErrInvalidJump,
	vm.ErrWriteProtection,
	vm.ErrReturnDataOutOfBounds,
	vm.ErrGasUintOverflow,
	vm.ErrInvalidCode,
	vm.ErrNonceUintOverflow,
}

// isUserError determines if the error we receive was most probably a problem
// on user's side.
//
// Here some errors that are flagged as error:
// - gRPC Canceled call (most probably canceled because user's closed the connection)
func isEVMDetermisticError(err error) bool {
	return derr.Find(err, func(candidate error) bool {
		for _, deterministicErr := range deterministicErrs {
			if candidate == deterministicErr {
				return true
			}
		}

		return false
	}) != nil
}

func isGrpcErrorWithCode(err error, code codes.Code) bool {
	var status (interface{ GRPCStatus() *status.Status })
	if errors.As(err, &status) && status.GRPCStatus().Code() == code {
		return true
	}

	return false
}
