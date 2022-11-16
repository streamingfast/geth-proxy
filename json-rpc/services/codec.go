package services

import (
	"fmt"
	"io"

	"github.com/gorilla/rpc/v2/json2"
	"github.com/streamingfast/eth-go/rpc"
)

// NewEthereumCodec defines a `json2.Codec` that handles Ethereum rules like the output format
// which is all in hexadecimal form for Ethereum types.
//
// The whole file is in `services` package because it's used in tests and also used in the
// parent `jsonrpc` package which uses also the codec. By keeping it in `services`, we avoid
// a cycle between `jsonrpc` -- requires --> `services` -- requires (via eth_call_test.go) --> `jsonrpc`.
func NewEthereumCodec() *json2.Codec {
	return json2.NewCustomCodec(json2.WithJSONEncoderFactory(EthereumJSONRPCEncoder))
}

type json2EncoderFunc func(v interface{}) error

func (f json2EncoderFunc) Encode(v interface{}) error {
	return f(v)
}

func EthereumJSONRPCEncoder(w io.Writer) json2.JSONEncoder {
	return json2EncoderFunc(func(v interface{}) error {
		out, err := rpc.MarshalJSONRPC(v)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}

		byteWritten := 0
		buffer := out
		for byteWritten < len(out) {
			writeByteCount, err := w.Write(buffer)
			if err != nil {
				return fmt.Errorf("write: %w", err)
			}

			byteWritten += writeByteCount
			buffer = buffer[writeByteCount:]
		}

		return nil
	})
}
