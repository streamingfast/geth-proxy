package evmexecutor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/eth-go"
)

var ErrNotFound = errors.New("not found")

type StateProvider interface {
	FetchAccount(ctx context.Context, addr common.Address, atBlock bstream.BlockRef) (*Account, error)
	FetchBlockHeaderByHash(ctx context.Context, hash eth.Hash) (*types.Header, error)
	FetchBlockHeaderByNumber(ctx context.Context, number uint64) (*types.Header, error)
	FetchLatestBlockHeader(ctx context.Context) (*types.Header, error)

	// FetchStorage retrieves the contract's storage key for contract at `addr` for storage key `key` at a given block. If the
	// key is not found in the storage, the implementation must returns `ErrNotFound`` error.
	FetchStorage(ctx context.Context, addr common.Address, key common.Hash, atBlock bstream.BlockRef) (common.Hash, error)
}

func NewStateProvider(ctx context.Context, dsn string) (StateProvider, error) {
	if strings.HasPrefix(dsn, JSONRPCScheme) {
		return NewJSONRPCStateProviderFromDSN(ctx, dsn)
	}

	if strings.HasPrefix(dsn, StateDBScheme) {
		return NewStateDBProviderFromDSN(ctx, dsn)
	}

	return nil, fmt.Errorf("DSN %q is invalid, accepting only %s:// and %s:// schemes", dsn, JSONRPCScheme, StateDBScheme)
}
