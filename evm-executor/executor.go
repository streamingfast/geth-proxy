package evmexecutor

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/deepmind"
	"github.com/holiman/uint256"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/eth-go/rpc"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

type CallExecutor interface {
	BlockByNumber(ctx context.Context, atBlock bstream.BlockRef) (*rpc.Block, error)
	BlockByHash(ctx context.Context, atBlock bstream.BlockRef) (*rpc.Block, error)
	ExecuteCall(ctx context.Context, callParams rpc.CallParams, atBlock bstream.BlockRef) (returnData []byte, gasUsed uint64, err error)
}

func NewCallExecutor(
	ctx context.Context,
	chainConfig *ChainConfig,
	gasCap uint64,
	stateProviderDSN string,
	timeout time.Duration,
) (CallExecutor, error) {
	stateProvider, err := NewStateProvider(ctx, stateProviderDSN)
	if err != nil {
		return nil, fmt.Errorf("new state provider: %w", err)
	}

	return &callExecutor{
		chainConfig:   chainConfig,
		gasCap:        gasCap,
		stateProvider: stateProvider,
		timeout:       timeout,
	}, nil
}

type callExecutor struct {
	chainConfig *ChainConfig
	// gasCap represents the maximum amount of Gas an EVM call can consume before
	// returning with an out of gas error
	gasCap        uint64
	stateProvider StateProvider
	timeout       time.Duration
}

func (e *callExecutor) ExecuteCall(ctx context.Context, callParams rpc.CallParams, atBlock bstream.BlockRef) (returnData []byte, gasUsed uint64, err error) {
	logger := logging.Logger(ctx, zlog)
	logger.Debug("executing call", zap.Stringer("at", atBlock), zap.Reflect("params", callParams))

	startTime := time.Now()
	defer func() {
		logger.Debug("execute call completed", zap.Duration("elapsed", time.Since(startTime)))
	}()

	// For now we do not accumulate anything related to deep mind
	dmContext := deepmind.NoOpContext

	var from, to *common.Address
	if len(callParams.From) > 0 {
		address := common.BytesToAddress(callParams.From)
		from = &address
	}

	if len(callParams.To) > 0 {
		address := common.BytesToAddress(callParams.To)
		to = &address
	}

	data, err := methodCallDataToBytes(callParams.Data)
	if err != nil {
		return nil, 0, fmt.Errorf("call params data: %w", err)
	}

	transactionArgs := TransactionArgs{
		From:     from,
		To:       to,
		GasPrice: (*hexutil.Big)(callParams.GasPrice),
		Value:    (*hexutil.Big)(callParams.Value),
		// FIXME: Should we provide this value?
		Nonce: nil,
		Input: (*hexutil.Bytes)(&data),
	}

	// The `transactionArgs.Gas`` must be set only when user provided value is non-zero, in the zero
	// case, `transactionArgs.Gas` must be nil to be populated properly in `doCall`
	if callParams.GasLimit != 0 {
		transactionArgs.Gas = (*hexutil.Uint64)(&callParams.GasLimit)
	}

	blockHeader, err := fetchBlockHeader(ctx, atBlock, e.stateProvider)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch block header: %w", err)
	}

	executionResult, err := doCall(ctx, transactionArgs, blockHeader, e.chainConfig, e.stateProvider, e.timeout, e.gasCap, dmContext)
	if err != nil {
		return nil, 0, fmt.Errorf("execution failed: %w", err)
	}

	if executionResult.Err != nil {
		// We must not wrap the error here, because it's going to be serialized as-is for the response
		return nil, 0, executionResult.Err
	}

	returnData = executionResult.ReturnData
	gasUsed = executionResult.UsedGas
	err = nil

	return
}

func methodCallDataToBytes(callData interface{}) ([]byte, error) {
	switch v := callData.(type) {
	case []byte:
		return v, nil

	case eth.Hex:
		return v, nil

	case *eth.MethodCall:
		return v.Encode()

	default:
		return nil, fmt.Errorf("only supporting 'callParams.Data' of type '[]byte', 'eth.Hex' and '*eth.MethodCall', got %T", callData)
	}
}

func (e *callExecutor) BlockByNumber(ctx context.Context, atBlock bstream.BlockRef) (*rpc.Block, error) {
	blockHeader, err := fetchBlockHeader(ctx, atBlock, e.stateProvider)
	if err != nil {
		return nil, fmt.Errorf("fetch block header: %w", err)
	}

	difficulty, overflow := uint256.FromBig(blockHeader.Difficulty)
	if overflow {
		return nil, fmt.Errorf("difficulty is above 256 bits, refusing to return block")
	}

	var baseFee *uint256.Int
	if blockHeader.BaseFee != nil {
		baseFee, overflow = uint256.FromBig(blockHeader.BaseFee)
		if overflow {
			return nil, fmt.Errorf("baseFee is above 256 bits, refusing to return block")
		}
	}

	return &rpc.Block{
		Number:           eth.Uint64(blockHeader.Number.Uint64()),
		Hash:             eth.Hash(blockHeader.Hash().Bytes()),
		ParentHash:       eth.Hash(blockHeader.ParentHash[:]),
		Timestamp:        eth.Timestamp(time.Unix(int64(blockHeader.Time), 0).UTC()),
		StateRoot:        eth.Hash(blockHeader.Root[:]),
		TransactionsRoot: eth.Hash(blockHeader.TxHash[:]),
		ReceiptsRoot:     eth.Hash(blockHeader.ReceiptHash[:]),
		MixHash:          eth.Hash(blockHeader.MixDigest[:]),
		GasLimit:         eth.Uint64(blockHeader.GasLimit),
		GasUsed:          eth.Uint64(blockHeader.GasUsed),
		Difficulty:       (*eth.Uint256)(difficulty),
		Miner:            eth.Address(blockHeader.Coinbase[:]),
		Nonce:            eth.FixedUint64(blockHeader.Nonce.Uint64()),
		LogsBloom:        eth.Hex(blockHeader.Bloom[:]),
		ExtraData:        eth.Hex(blockHeader.Extra),
		BaseFeePerGas:    (*eth.Uint256)(baseFee),
		UnclesSHA3:       eth.Hash(blockHeader.UncleHash[:]),
		Transactions:     rpc.NewBlockTransactions(),

		// The BlockHeader don't have those:
		// - BlockSize
		// - Transactions
		// - TotalDifficulty
		// - Uncles
	}, nil
}

func (e *callExecutor) BlockByHash(ctx context.Context, atBlock bstream.BlockRef) (*rpc.Block, error) {
	blockHeader, err := fetchBlockHeader(ctx, atBlock, e.stateProvider)
	if err != nil {
		return nil, fmt.Errorf("fetch block header: %w", err)
	}

	difficulty, overflow := uint256.FromBig(blockHeader.Difficulty)
	if overflow {
		return nil, fmt.Errorf("difficulty is above 256 bits, refusing to return block")
	}

	var baseFee *uint256.Int
	if blockHeader.BaseFee != nil {
		baseFee, overflow = uint256.FromBig(blockHeader.BaseFee)
		if overflow {
			return nil, fmt.Errorf("baseFee is above 256 bits, refusing to return block")
		}
	}

	return &rpc.Block{
		Number:           eth.Uint64(blockHeader.Number.Uint64()),
		Hash:             eth.Hash(blockHeader.Hash().Bytes()),
		ParentHash:       eth.Hash(blockHeader.ParentHash[:]),
		Timestamp:        eth.Timestamp(time.Unix(int64(blockHeader.Time), 0).UTC()),
		StateRoot:        eth.Hash(blockHeader.Root[:]),
		TransactionsRoot: eth.Hash(blockHeader.TxHash[:]),
		ReceiptsRoot:     eth.Hash(blockHeader.ReceiptHash[:]),
		MixHash:          eth.Hash(blockHeader.MixDigest[:]),
		GasLimit:         eth.Uint64(blockHeader.GasLimit),
		GasUsed:          eth.Uint64(blockHeader.GasUsed),
		Difficulty:       (*eth.Uint256)(difficulty),
		Miner:            eth.Address(blockHeader.Coinbase[:]),
		Nonce:            eth.FixedUint64(blockHeader.Nonce.Uint64()),
		LogsBloom:        eth.Hex(blockHeader.Bloom[:]),
		ExtraData:        eth.Hex(blockHeader.Extra),
		BaseFeePerGas:    (*eth.Uint256)(baseFee),
		UnclesSHA3:       eth.Hash(blockHeader.UncleHash[:]),
		Transactions:     rpc.NewBlockTransactions(),

		// The BlockHeader don't have those:
		// - BlockSize
		// - Transactions
		// - TotalDifficulty
		// - Uncles
	}, nil
}
