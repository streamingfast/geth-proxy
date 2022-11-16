package evmexecutor

import (
	"encoding/hex"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/streamingfast/bstream"
)

// EarliestBlockRef is the block reference value when you want to execute relatively to the earliest block of the chain.
var EarliestBlockRef = bstream.NewBlockRef("", 0)

// LatestBlockRef is the block reference value when you want to execute relatively to the latest (HEAD) block of the chain.
var LatestBlockRef = bstream.NewBlockRef("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", math.MaxUint64)

func isLatestBlockRef(ref bstream.BlockRef) bool {
	return ref == nil || ref == bstream.BlockRefEmpty || ref == LatestBlockRef
}

func newEthereumBlockRef(hash []byte, number uint64) bstream.BlockRef {
	return bstream.NewBlockRef(hex.EncodeToString(hash), number)
}

func newGethBlockRef(hash common.Hash, number uint64) bstream.BlockRef {
	return bstream.NewBlockRef(hex.EncodeToString(hash[:]), number)
}

type Account struct {
	Nonce    uint64
	Balance  *big.Int
	Code     []byte
	CodeHash []byte

	// Root     common.Hash // merkle root of the storage trie
}

func NewAccount(nonce uint64, balance *big.Int, code []byte) *Account {
	account := &Account{
		Code:    code,
		Balance: balance,
		Nonce:   nonce,
	}

	if len(code) > 0 {
		account.CodeHash = computeCodeHash(code)
	}

	return account
}

func (a Account) IsEmpty() bool {
	return a.Nonce == 0 && len(a.Code) == 0 && a.Balance.Sign() == 0
}

func computeCodeHash(code []byte) []byte {
	return crypto.Keccak256Hash(code).Bytes()
}
