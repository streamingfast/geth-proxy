package evmexecutor

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"runtime/debug"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/deepmind"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/eth-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/codes"
)

var dmUnsetTrxHash = common.Hash{}

// doCall is inspired from https://github.com/streamingfast/go-ethereum/blob/21e9fa4c18367c9bc1cbd4a052d443594221440b/internal/ethapi/api.go#L895.
func doCall(
	ctx context.Context,
	args TransactionArgs,
	header *types.Header,
	chainConfig *ChainConfig,
	provider StateProvider,
	timeout time.Duration,
	globalGasCap uint64,
	dmContext *deepmind.Context,
) (result *core.ExecutionResult, err error) {
	startTime := time.Now()
	defer func() {
		zlog.Debug("evm execution completed", zap.Duration("elapsed", time.Since(startTime)))

		recoveredErr := recover()
		if recoveredErr != nil {
			switch t := recoveredErr.(type) {
			case error:
				err = t
			case string:
				err = errors.New(t)
			default:
				err = fmt.Errorf("unknown error occurred: %s", recoveredErr)
			}

			// Let's log only error that happened that were not due to a context.Cancelled
			if !errors.Is(err, context.Canceled) && !isGrpcErrorWithCode(err, codes.Canceled) {
				fields := []zap.Field{zap.Stringer("block", bstream.NewBlockRef(header.Hash().String(), header.Number.Uint64())), zap.Error(err)}
				if zlog.Core().Enabled(zap.DebugLevel) {
					fields = append(fields, zap.String("stack", string(debug.Stack())))
				}

				zlog.Info("recovered error from panic", fields...)
			}
		}
	}()

	zlog.Debug("evm execution", zap.Reflect("args", args), zap.Reflect("header", header))

	state := NewDStateDB(ctx, provider, newGethBlockRef(header.Hash(), header.Number.Uint64()), dmContext)

	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()

	// Get a new instance of the EVM.
	msg, err := args.ToMessage(globalGasCap, header.BaseFee)
	if err != nil {
		return nil, err
	}

	if tracer.Enabled() {
		zlog.Debug("call message", zap.Object("msg", (*DebugMessage)(&msg)))
	}

	evm, vmError, err := getEVM(ctx, msg, state, header, &vm.Config{NoBaseFee: true}, chainConfig, dmContext)
	if err != nil {
		return nil, err
	}

	// Wait for the context to be done and cancel the evm. Even if the
	// EVM has finished, cancelling may be done (repeatedly)
	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()

	if dmContext.Enabled() {
		dmContext.StartTransactionRaw(
			dmUnsetTrxHash,
			msg.To(),
			msg.Value(),
			nil, nil, nil,
			msg.Gas(),
			msg.GasPrice(),
			msg.Nonce(),
			msg.Data(),
			// FIXME: This does not respect firehose deep mind protocol, we would need to retrieve the
			// information somewhere ... (the four fields below this comment are concerned)
			nil,
			nil,
			nil,
			0,
		)
		dmContext.RecordTrxFrom(msg.From())
	}

	// Execute the message.
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	result, err = core.ApplyMessage(evm, msg, gp)
	if err := vmError(); err != nil {
		return nil, err
	}

	// If the timer caused an abort, return an appropriate error message
	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted (timeout = %v)", timeout)
	}

	if dmContext.Enabled() {
		var failed bool
		var gasUsed uint64
		if result != nil {
			failed = result.Failed()
			gasUsed = result.UsedGas
		}

		config := evm.ChainConfig()

		// FIXME: Should we aim here to get out the state root from what happened through the EVM execution.
		var root []byte
		if config.IsByzantium(header.Number) {
			// state.Finalise(true)
		} else {
			// root = state.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
		}

		// FIXME: The ApplyMessage can in speculative mode error out with some errors that in sync mode would
		//        not have been possible. Like ErrNonceTooHight or ErrNonceTooLow. Those error will be in
		//        `err` value but there is no real way to return it right now. You will get a failed transaction
		//        without any call and that's it.

		// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
		// based on the eip phase, we're passing whether the root touch-delete accounts.
		receipt := types.NewReceipt(root, failed, gasUsed)
		receipt.TxHash = dmUnsetTrxHash
		receipt.GasUsed = gasUsed
		// if the transaction created a contract, store the creation address in the receipt.
		if msg.To() == nil {
			receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, msg.Nonce())
		}

		// FIXME: We are definitely able to retrieve the logs that happened through the exeuction, probably
		// we should keep them on the state provider.
		// Set the receipt logs and create a bloom for filtering
		// receipt.Logs = state.GetLogs(dmUnsetTrxHash, header.Hash())
		receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
		receipt.BlockHash = header.Hash()
		receipt.BlockNumber = header.Number
		receipt.TransactionIndex = 0

		dmContext.EndTransaction(receipt)
	}

	// If in deep mind context, we should most probably attach the error here inside the deep mind context
	// somehow so it's attached to the top-leve transaction trace and not return this here. The handler above
	// us will need to understand that a nil error and nil result means send me everything. For now, it will
	// simply fail, might even be the "good condition" to do.
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.Gas())
	}
	return result, nil
}

func fetchBlockHeader(ctx context.Context, atBlock bstream.BlockRef, provider StateProvider) (header *types.Header, err error) {
	isLatest := isLatestBlockRef(atBlock)
	if isLatest {
		header, err = provider.FetchLatestBlockHeader(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetch block header: %w", err)
		}
	} else {
		if atBlock.ID() != "" {
			header, err = provider.FetchBlockHeaderByHash(ctx, eth.MustNewHash(atBlock.ID()))
			if err != nil {
				return nil, fmt.Errorf("fetch block header by hash %q: %w", atBlock.ID(), err)
			}
		} else {
			header, err = provider.FetchBlockHeaderByNumber(ctx, atBlock.Num())
			if err != nil {
				return nil, fmt.Errorf("fetch block header by number %d: %w", atBlock.Num(), err)
			}
		}

		if header != nil {
			if atBlock.ID() != "" && hex.EncodeToString(header.Hash().Bytes()) != atBlock.ID() {
				return nil, fmt.Errorf("block hash mismatch between requested block %s and received block #%d (%s) from state provider", atBlock, header.Number.Uint64(), header.Hash())
			}

			// We do not perform the check for the genesis block
			if atBlock.Num() != 0 && atBlock.Num() != header.Number.Uint64() {
				return nil, fmt.Errorf("block num mismatch between requested block %s and received block #%d (%s) from state provider", atBlock, header.Number.Uint64(), header.Hash())
			}
		}
	}

	if header == nil {
		return nil, fmt.Errorf("block %s does not exist", atBlock)
	}

	return header, nil
}

func getEVM(ctx context.Context, msg core.Message, state vm.StateDB, header *types.Header, vmConfig *vm.Config, chainConfig *ChainConfig, dmContext *deepmind.Context) (*vm.EVM, func() error, error) {
	vmError := func() error { return nil }
	if vmConfig == nil {
		vmConfig = &vm.Config{}
	}

	// Does it have an importance in our context what consensus engine do we use? We could
	// also use Clique engine here which is easily creatable, but right now we assumed the the
	// PoW faker is good enough.
	chainContext := &fakeChainContext{engine: ethash.NewFaker(), header: header}

	txContext := core.NewEVMTxContext(msg)
	context := core.NewEVMBlockContext(header, chainContext, nil)
	return vm.NewEVM(context, txContext, state, chainConfig.ChainConfig, *vmConfig, dmContext), vmError, nil
}

type fakeChainContext struct {
	engine consensus.Engine
	header *types.Header
}

func (c *fakeChainContext) Engine() consensus.Engine {
	return c.engine
}

func (c *fakeChainContext) GetHeader(hash common.Hash, _ uint64) *types.Header {
	if hash == c.header.Hash() {
		return c.header
	}

	return nil
}

type DebugMessage types.Message

func (m *DebugMessage) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	msg := (*types.Message)(m)

	encoder.AddString("from", msg.From().String())
	encoder.AddString("to", msg.To().String())
	encoder.AddUint64("nonce", msg.Nonce())
	encoder.AddString("value", msg.Value().String())
	encoder.AddUint64("gas", msg.Gas())
	encoder.AddString("gas_price", msg.GasPrice().String())
	encoder.AddString("gas_fee_cap", msg.GasFeeCap().String())
	encoder.AddString("gas_tip_cap", msg.GasTipCap().String())
	encoder.AddString("data", eth.Hex(msg.Data()).String())
	encoder.AddString("from", msg.From().String())

	return nil
}
