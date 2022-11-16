package evmexecutor

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func isGrpcErrorWithCode(err error, code codes.Code) bool {
	var status (interface{ GRPCStatus() *status.Status })
	if errors.As(err, &status) && status.GRPCStatus().Code() == code {
		return true
	}

	return false
}
