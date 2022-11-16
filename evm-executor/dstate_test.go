package evmexecutor

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/deepmind"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/eth-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDStateDB_GetCommittedState(t *testing.T) {
	type fields struct {
		atBlock  bstream.BlockRef
		provider StateProvider
	}

	type args struct {
		address common.Address
		key     common.Hash
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   common.Hash
		panics error
	}{
		{
			"empty hash when storage not found, directly not found error",
			fields{
				atBlock: bstream.NewBlockRef("aa", 1),
				provider: (&MockStateProvider{}).SetFetchStorage(func(ctx context.Context, addr common.Address, key common.Hash, atBlock bstream.BlockRef) (common.Hash, error) {
					return common.Hash{}, ErrNotFound
				}),
			},
			args{
				address: common.Address{},
				key:     common.Hash{},
			},
			EmptyHash,
			nil,
		},
		{
			"empty hash when storage not found, wrapped not found error",
			fields{
				atBlock: bstream.NewBlockRef("aa", 1),
				provider: (&MockStateProvider{}).SetFetchStorage(func(ctx context.Context, addr common.Address, key common.Hash, atBlock bstream.BlockRef) (common.Hash, error) {
					return common.Hash{}, fmt.Errorf("wrapped error: %w", ErrNotFound)
				}),
			},
			args{
				address: common.Address{},
				key:     common.Hash{},
			},
			EmptyHash,
			nil,
		},
		{
			"panics when storage returns any error outside of not found",
			fields{
				atBlock: bstream.NewBlockRef("aa", 1),
				provider: (&MockStateProvider{}).SetFetchStorage(func(ctx context.Context, addr common.Address, key common.Hash, atBlock bstream.BlockRef) (common.Hash, error) {
					return common.Hash{}, fmt.Errorf("random error")
				}),
			},
			args{
				address: common.Address{},
				key:     common.Hash{},
			},
			EmptyHash,
			fmt.Errorf("cannot fetch storage: random error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewDStateDB(context.Background(), tt.fields.provider, tt.fields.atBlock, deepmind.NoOpContext)
			db.logger = zlog

			if tt.panics != nil {
				require.PanicsWithError(t, tt.panics.Error(), func() {
					db.GetCommittedState(tt.args.address, tt.args.key)
				})
			} else {
				assert.Equal(t, tt.want, db.GetCommittedState(tt.args.address, tt.args.key))
			}
		})
	}
}

func TestDStateDB_GetCommittedState_InitialStorageCachingWorks(t *testing.T) {
	fetchStorageCount := 0
	provider := (&MockStateProvider{}).SetFetchStorage(func(context.Context, common.Address, common.Hash, bstream.BlockRef) (common.Hash, error) {
		fetchStorageCount++
		return common.HexToHash("cc"), nil
	})

	db := NewDStateDB(context.Background(), provider, bstream.NewBlockRef("aa", 1), deepmind.NoOpContext)
	db.logger = zlog

	// New call to "bb/ff"
	result1 := db.GetCommittedState(common.HexToAddress("bb"), common.HexToHash("ff"))

	// Cached call "bb/ff"
	result2 := db.GetCommittedState(common.HexToAddress("bb"), common.HexToHash("ff"))

	// New call "bb/ee" on existing cached account "bb"
	result3 := db.GetCommittedState(common.HexToAddress("bb"), common.HexToHash("ee"))

	assert.Equal(t, common.HexToHash("cc"), result1)
	assert.Equal(t, common.HexToHash("cc"), result2)
	assert.Equal(t, common.HexToHash("cc"), result3)
	assert.Equal(t, 2, fetchStorageCount)
}

var _ StateProvider = (*MockStateProvider)(nil)

type MockStateProvider struct {
	fetchAccount             func(ctx context.Context, addr common.Address, atBlock bstream.BlockRef) (*Account, error)
	fetchBlockHeaderByHash   func(ctx context.Context, hash eth.Hash) (*types.Header, error)
	fetchBlockHeaderByNumber func(ctx context.Context, number uint64) (*types.Header, error)
	fetchLatestBlockHeader   func(ctx context.Context) (*types.Header, error)
	fetchStorage             func(ctx context.Context, addr common.Address, key common.Hash, atBlock bstream.BlockRef) (common.Hash, error)
}

func (p *MockStateProvider) SetFetchAccount(fun func(ctx context.Context, addr common.Address, atBlock bstream.BlockRef) (*Account, error)) *MockStateProvider {
	p.fetchAccount = fun
	return p
}

func (p *MockStateProvider) SetFetchBlockHeaderByHash(fun func(ctx context.Context, hash eth.Hash) (*types.Header, error)) *MockStateProvider {
	p.fetchBlockHeaderByHash = fun
	return p
}

func (p *MockStateProvider) SetFetchBlockHeaderByNumber(fun func(ctx context.Context, number uint64) (*types.Header, error)) *MockStateProvider {
	p.fetchBlockHeaderByNumber = fun
	return p
}

func (p *MockStateProvider) SetFetchLatestBlockHeader(fun func(ctx context.Context) (*types.Header, error)) *MockStateProvider {
	p.fetchLatestBlockHeader = fun
	return p
}

func (p *MockStateProvider) SetFetchStorage(fun func(ctx context.Context, addr common.Address, key common.Hash, atBlock bstream.BlockRef) (common.Hash, error)) *MockStateProvider {
	p.fetchStorage = fun
	return p
}

func (p *MockStateProvider) FetchAccount(ctx context.Context, addr common.Address, atBlock bstream.BlockRef) (*Account, error) {
	if p.fetchAccount != nil {
		return p.fetchAccount(ctx, addr, atBlock)
	}

	panic(`mock state provider do not have an implementation for "fetchAccount" but the method was called, provide a mock implementation for it`)
}

func (p *MockStateProvider) FetchBlockHeaderByHash(ctx context.Context, hash eth.Hash) (*types.Header, error) {
	if p.fetchBlockHeaderByHash != nil {
		return p.fetchBlockHeaderByHash(ctx, hash)
	}

	panic(`mock state provider do not have an implementation for "fetchBlockHeaderByHash" but the method was called, provide a mock implementation for it`)
}

func (p *MockStateProvider) FetchBlockHeaderByNumber(ctx context.Context, number uint64) (*types.Header, error) {
	if p.fetchBlockHeaderByNumber != nil {
		return p.fetchBlockHeaderByNumber(ctx, number)
	}

	panic(`mock state provider do not have an implementation for "fetchBlockHeaderByNumber" but the method was called, provide a mock implementation for it`)
}

func (p *MockStateProvider) FetchLatestBlockHeader(ctx context.Context) (*types.Header, error) {
	if p.fetchLatestBlockHeader != nil {
		return p.fetchLatestBlockHeader(ctx)
	}

	panic(`mock state provider do not have an implementation for "fetchLatestBlockHeader" but the method was called, provide a mock implementation for it`)
}

func (p *MockStateProvider) FetchStorage(ctx context.Context, addr common.Address, key common.Hash, atBlock bstream.BlockRef) (common.Hash, error) {
	if p.fetchStorage != nil {
		return p.fetchStorage(ctx, addr, key, atBlock)
	}

	panic(`mock state provider do not have an implementation for "fetchStorage" but the method was called, provide a mock implementation for it`)
}
