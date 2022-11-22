package config

import (
	"context"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/eth-go/rpc"
	ethrpc "github.com/streamingfast/eth-go/rpc"
	"math/big"
)

func NetworkNameToChainConfig(networkName string) (*ChainConfig, error) {
	switch networkName {
	case "mainnet":
		return &ChainConfig{
			ChainConfig: params.MainnetChainConfig,
			NetworkID:   1,
		}, nil

	case "goerli":
		return &ChainConfig{
			ChainConfig: params.GoerliChainConfig,
			NetworkID:   5,
		}, nil

	case "battlefield":
		return &ChainConfig{
			ChainConfig: BattlefieldChainConfig,
			NetworkID:   1515,
		}, nil

	default:
		return nil, nil
	}
}

type ChainConfig struct {
	*params.ChainConfig

	// NetworkID is the version peers must be using to allow exchange of information.
	NetworkID uint64
}

var BattlefieldChainConfig = &params.ChainConfig{
	ChainID:             big.NewInt(1515),
	HomesteadBlock:      big.NewInt(0),
	DAOForkBlock:        nil,
	DAOForkSupport:      true,
	EIP150Block:         big.NewInt(0),
	EIP150Hash:          common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
	EIP155Block:         big.NewInt(0),
	EIP158Block:         big.NewInt(0),
	ByzantiumBlock:      big.NewInt(0),
	ConstantinopleBlock: big.NewInt(0),
	PetersburgBlock:     big.NewInt(0),
	IstanbulBlock:       big.NewInt(0),
	MuirGlacierBlock:    big.NewInt(0),
	BerlinBlock:         nil,
	LondonBlock:         nil,
	ArrowGlacierBlock:   nil,
	Ethash:              nil,
}

type CallExecutor interface {
	BlockByNumber(ctx context.Context, atBlock bstream.BlockRef) (*rpc.Block, error)
	BlockByHash(ctx context.Context, atBlock bstream.BlockRef) (*rpc.Block, error)
	ExecuteCall(ctx context.Context, callParams rpc.CallParams, atBlock bstream.BlockRef) (returnData []byte, gasUsed uint64, err error)
}

func ToBstreamBlockRef(ref *ethrpc.BlockRef) bstream.BlockRef {
	if ref.IsLatest() {
		return LatestBlockRef
	}

	if ref.IsEarliest() {
		return EarliestBlockRef
	}

	if ref.IsPending() {
		panic(fmt.Errorf("block ref is pending but it's not accepted for conversion to bstream.BlockRef"))
	}

	if hash, ok := ref.BlockHash(); ok {
		return bstream.NewBlockRef(hash.String(), 0)
	}

	blockNumber, ok := ref.BlockNumber()
	if !ok {
		panic(fmt.Errorf("block ref is not a block number but was expected to be so"))
	}

	return bstream.NewBlockRef("", blockNumber)
}

var EarliestBlockRef = bstream.NewBlockRef("", 0)

// LatestBlockRef is the block reference value when you want to execute relatively to the latest (HEAD) block of the chain.
var LatestBlockRef = bstream.NewBlockRef("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", math.MaxUint64)
