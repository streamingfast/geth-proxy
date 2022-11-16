package evmexecutor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/tests"
)

func MakePreState(db ethdb.Database) *state.StateDB {
	sdb := state.NewDatabase(db) // Use `NewDatabaseWithCache` if we want caching, etc..

	// Here, second argument of state is `snapshot.Tree` which as far as we understand, enables
	// a faster code path for data retrieval. You should check about what it would take to
	// use it.
	statedb, _ := state.New(common.Hash{}, sdb, nil)
	root, _ := statedb.Commit(false)
	statedb, _ = state.New(root, sdb, nil)
	return statedb
}

func getVMConfig(forkString string) (baseConfig *params.ChainConfig, eips []int, err error) {
	var (
		splitForks            = strings.Split(forkString, "+")
		ok                    bool
		baseName, eipsStrings = splitForks[0], splitForks[1:]
	)
	if baseConfig, ok = tests.Forks[baseName]; !ok {
		return nil, nil, tests.UnsupportedForkError{Name: baseName}
	}

	for _, eip := range eipsStrings {
		eipNum, err := strconv.Atoi(eip)
		if err != nil {
			return nil, nil, fmt.Errorf("syntax error, invalid eip number %v", eipNum)
		}

		eips = append(eips, eipNum)
	}

	return baseConfig, eips, nil
}
