package evmexecutor

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

type ChainConfig struct {
	*params.ChainConfig

	// NetworkID is the version peers must be using to allow exchange of information.
	NetworkID uint64
}

// NetworkNameToChainConfig accepts a network name and returns the correct chain config for it.
// If the network name is unknown, we return `nil`.
func NetworkNameToChainConfig(networkName string) *ChainConfig {
	switch networkName {
	case "mainnet":
		return &ChainConfig{
			ChainConfig: params.MainnetChainConfig,
			NetworkID:   1,
		}

	case "goerli":
		return &ChainConfig{
			ChainConfig: params.GoerliChainConfig,
			NetworkID:   5,
		}

	case "battlefield":
		return &ChainConfig{
			ChainConfig: BattlefieldChainConfig,
			NetworkID:   1515,
		}

	default:
		return nil
	}
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
