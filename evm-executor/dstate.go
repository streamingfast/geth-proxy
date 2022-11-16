package evmexecutor

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/deepmind"
	"github.com/jinzhu/copier"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var _ vm.StateDB = (*DStateDB)(nil)

type stateSnapshot struct {
	accounts map[common.Address]*Account
	storage  map[common.Address]map[common.Hash]common.Hash
	logs     map[common.Address][]*types.Log
}

type DStateDB struct {
	ctx      context.Context
	provider StateProvider
	atBlock  bstream.BlockRef

	dmContext *deepmind.Context

	initialAccounts map[common.Address]*Account
	initialStorage  map[common.Address]map[common.Hash]common.Hash

	dirtyAccounts map[common.Address]*Account
	dirtyStorage  map[common.Address]map[common.Hash]common.Hash
	dirtyLogs     map[common.Address][]*types.Log

	refund    uint64
	snapshots []*stateSnapshot

	logger *zap.Logger
}

func NewDStateDB(ctx context.Context, provider StateProvider, atBlock bstream.BlockRef, dmContext *deepmind.Context) *DStateDB {
	return &DStateDB{
		ctx:       ctx,
		provider:  provider,
		atBlock:   atBlock,
		dmContext: dmContext,

		initialAccounts: make(map[common.Address]*Account),
		initialStorage:  make(map[common.Address]map[common.Hash]common.Hash),

		dirtyAccounts: make(map[common.Address]*Account),
		dirtyStorage:  make(map[common.Address]map[common.Hash]common.Hash),
		dirtyLogs:     make(map[common.Address][]*types.Log),

		logger: logging.Logger(ctx, zlog),
	}
}

func (db *DStateDB) CreateAccount(address common.Address, dmContext *deepmind.Context) {
	db.logger.Debug("creating account", zap.Stringer("address", address))
	panic("not implemented") // TODO: Implement
}

func (db *DStateDB) getOrCreateAccount(addr common.Address) *Account {
	account := db.getAccount(addr)
	if account != nil {
		return account
	}

	account = &Account{
		Balance: new(big.Int),
	}

	db.dirtyAccounts[addr] = account
	db.initialAccounts[addr] = account

	return account
}

func (db *DStateDB) getAccount(addr common.Address) *Account {
	// Check if we have it in cache,
	acct := db.findLocalAccount(addr)
	if acct != nil {
		return acct
	}

	account, err := db.provider.FetchAccount(db.ctx, addr, db.atBlock)
	if err != nil && err != ErrNotFound {
		panic(fmt.Errorf("fetch account: %w", err))
	}

	if account != nil {
		db.dirtyAccounts[addr] = account

		if tracer.Enabled() {
			db.logger.Debug("fetched account", zap.Stringer("addr", addr), zap.Reflect("account", account), zap.Stringer("at_block", db.atBlock))
		}
	}

	return account
}

func (db *DStateDB) findLocalAccount(addr common.Address) *Account {
	if acct, found := db.dirtyAccounts[addr]; found {
		return acct
	}

	if acct, found := db.initialAccounts[addr]; found {
		return acct
	}

	return nil
}

// FIXME: Increase code sharing between SubBalance and AddBalance
func (db *DStateDB) SubBalance(addr common.Address, amount *big.Int, dmContext *deepmind.Context, reason deepmind.BalanceChangeReason) {
	if amount.Sign() == 0 {
		// No need to change anything since there is no changes to apply here
		return
	}

	db.logger.Debug("sub balance", zap.Stringer("address", addr), zap.Stringer("amount", amount), zap.String("reason", string(reason)))

	acct := db.getOrCreateAccount(addr)
	balance := acct.Balance

	acct.Balance = new(big.Int).Sub(balance, amount)
	db.dirtyAccounts[addr] = acct

	db.dmContext.RecordBalanceChange(addr, balance, acct.Balance, reason)
}

// FIXME: Increase code sharing between SubBalance and AddBalance
func (db *DStateDB) AddBalance(addr common.Address, amount *big.Int, isPrecompile bool, dmContext *deepmind.Context, reason deepmind.BalanceChangeReason) {
	if amount.Sign() == 0 {
		// No need to change anything since there is no changes to apply here
		return
	}

	db.logger.Debug("add balance", zap.Stringer("address", addr), zap.Stringer("amount", amount), zap.String("reason", string(reason)))

	acct := db.getOrCreateAccount(addr)
	balance := acct.Balance

	acct.Balance = new(big.Int).Add(balance, amount)
	db.dirtyAccounts[addr] = acct

	db.dmContext.RecordBalanceChange(addr, balance, acct.Balance, reason)
}

func (db *DStateDB) GetBalance(addr common.Address) *big.Int {
	db.logger.Debug("getting balance", zap.Stringer("address", addr))

	acct := db.getAccount(addr)
	if acct == nil {
		return common.Big0
	}

	return acct.Balance
}

func (db *DStateDB) GetNonce(addr common.Address) uint64 {
	db.logger.Debug("getting nonce", zap.Stringer("address", addr))

	acct := db.getAccount(addr)
	if acct == nil {
		return 0
	}

	return acct.Nonce
}

func (db *DStateDB) SetNonce(addr common.Address, nonce uint64, dmContext *deepmind.Context) {
	db.logger.Debug("setting nonce", zap.Stringer("address", addr))

	acct := db.getOrCreateAccount(addr)

	db.dmContext.RecordNonceChange(addr, acct.Nonce, nonce)

	acct.Nonce = nonce
	db.dirtyAccounts[addr] = acct
}

func (db *DStateDB) GetCodeHash(addr common.Address) common.Hash {
	db.logger.Debug("getting code hash", zap.Stringer("address", addr))

	acct := db.getAccount(addr)
	if acct == nil {
		return common.Hash{}
	}

	return common.BytesToHash(acct.CodeHash)
}

func (db *DStateDB) GetCode(addr common.Address) []byte {
	db.logger.Debug("getting code", zap.Stringer("address", addr))
	acct := db.getAccount(addr)
	if acct == nil {
		return nil
	}

	return acct.Code
}

func (db *DStateDB) SetCode(addr common.Address, code []byte, dmContext *deepmind.Context) {
	db.logger.Debug("setting code", zap.Stringer("address", addr))
	panic("not implemented SetCode") // TODO: Implement
}

func (db *DStateDB) GetCodeSize(addr common.Address) int {
	db.logger.Debug("getting code size", zap.Stringer("address", addr))

	acct := db.getAccount(addr)
	if acct == nil {
		return 0
	}

	return len(acct.Code)
}

func (db *DStateDB) AddRefund(gas uint64) {
	if gas == 0 {
		// No need to change anything since there is no changes to apply here
		return
	}

	db.logger.Debug("add refund", zap.Uint64("amount", gas))
	db.refund += gas
}

func (db *DStateDB) SubRefund(gas uint64) {
	if gas == 0 {
		// No need to change anything since there is no changes to apply here
		return
	}

	db.logger.Debug("sub refund", zap.Uint64("amount", gas))

	if gas > db.refund {
		panic("Refund counter below zero")
	}

	db.refund -= gas
}

func (db *DStateDB) GetRefund() uint64 {
	db.logger.Debug("getting refund")
	return db.refund
}

func (db *DStateDB) GetCommittedState(address common.Address, key common.Hash) common.Hash {
	db.logger.Debug("reading committed state", zap.Stringer("addr", address), zap.Stringer("key", key))

	acctStore := db.initialStorage[address]
	if acctStore != nil {
		if value, found := acctStore[key]; found {
			return value
		}
	}

	value, err := db.provider.FetchStorage(db.ctx, address, key, db.atBlock)
	if err != nil {
		db.logger.Debug("unable to get storage value", zap.String("err", err.Error()))
		if errors.Is(err, ErrNotFound) {
			return EmptyHash
		}

		panic(fmt.Errorf("cannot fetch storage: %w", err))
	}

	if acctStore == nil {
		acctStore = map[common.Hash]common.Hash{}
		db.initialStorage[address] = acctStore
	}

	acctStore[key] = value

	return value
}

func (db *DStateDB) GetState(address common.Address, key common.Hash) (out common.Hash) {
	db.logger.Debug("getting contract state", zap.Stringer("address", address), zap.Stringer("key", key))

	acctStore := db.dirtyStorage[address]
	if acctStore == nil {
		return db.GetCommittedState(address, key)
	}

	value, found := acctStore[key]
	if found {
		return value
	}

	return db.GetCommittedState(address, key)
}

func (db *DStateDB) SetState(address common.Address, key, value common.Hash, dmContext *deepmind.Context) {
	db.logger.Debug("setting contract state", zap.Stringer("address", address), zap.Stringer("key", key), zap.Stringer("value", value))

	acctStore := db.dirtyStorage[address]
	if acctStore == nil {
		acctStore = map[common.Hash]common.Hash{}
		db.dirtyStorage[address] = acctStore
	}

	db.dmContext.RecordStorageChange(address, key, acctStore[key], value)

	acctStore[key] = value
}

func (db *DStateDB) Suicide(address common.Address, dmContext *deepmind.Context) bool {
	db.logger.Debug("suicding contract", zap.Stringer("address", address))

	// TODO: verify if we're in READ-ONLY mode or not.
	panic("not implemented suicide") // TODO: Implement
}

func (db *DStateDB) HasSuicided(address common.Address) bool {
	db.logger.Debug("checking if contract has suicided", zap.Stringer("address", address))

	panic("not implemented has suicided") // TODO: Implement
}

// Exist reports whether the given account exists in state.
// Notably this should also return true for suicided accounts.
func (db *DStateDB) Exist(addr common.Address) bool {
	db.logger.Debug("checking if account exists", zap.Stringer("address", addr))
	return db.getAccount(addr) != nil
}

// Empty returns whether the given account is empty. Empty
// is defined according to EIP161 (balance = nonce = code = 0).
func (db *DStateDB) Empty(address common.Address) bool {
	db.logger.Debug("checking if account is empty", zap.Stringer("address", address))

	account := db.getAccount(address)
	return account == nil || account.IsEmpty()
}

func (db *DStateDB) RevertToSnapshot(id int) {
	db.logger.Debug("reverting state to snapshot", zap.Int("snapshot_id", id))
	snapshot := db.snapshots[id]
	if snapshot == nil {
		panic(fmt.Errorf("no snapshot with id %d found", id))
	}

	db.dirtyAccounts = db.snapshots[id].accounts
	if db.dirtyAccounts == nil {
		panic(fmt.Errorf("snapshot %d accounts are nil which is unexpected", id))
	}

	db.dirtyStorage = db.snapshots[id].storage
	if db.dirtyStorage == nil {
		panic(fmt.Errorf("snapshot %d storage is nil which is unexpected", id))
	}

	db.dirtyLogs = db.snapshots[id].logs
	if db.dirtyLogs == nil {
		panic(fmt.Errorf("snapshot %d logs are nil which is unexpected", id))
	}
}

func (db *DStateDB) Snapshot() int {
	nextID := len(db.snapshots)
	db.logger.Debug("taking state snapshot", zap.Int("next_id", nextID))

	newSnapshot := stateSnapshot{}
	err := copier.Copy(&newSnapshot.accounts, db.dirtyAccounts)
	if err != nil {
		panic(fmt.Errorf("copying dirty accounts: %w", err))
	}

	err = copier.Copy(&newSnapshot.storage, db.dirtyStorage)
	if err != nil {
		panic(fmt.Errorf("copying dirty storage: %w", err))
	}

	err = copier.Copy(&newSnapshot.logs, db.dirtyLogs)
	if err != nil {
		panic(fmt.Errorf("copying dirty logs: %w", err))
	}

	db.snapshots = append(db.snapshots, &newSnapshot)

	return nextID
}

func (db *DStateDB) AddLog(log *types.Log, dmContext *deepmind.Context) {
	db.dirtyLogs[log.Address] = append(db.dirtyLogs[log.Address], log)

	db.dmContext.RecordLog(log)
}

func (db *DStateDB) AddPreimage(hash common.Hash, preimage []byte) {
	db.logger.Debug("adding preimage")
	panic("not implemented preimage") // TODO: Implement
}

func (db *DStateDB) ForEachStorage(address common.Address, f func(common.Hash, common.Hash) bool) error {
	db.logger.Debug("foreach storage")
	panic("not implemented for each storage") // TODO: Implement
}

func (db *DStateDB) getAllLogs() []*types.Log {
	var allLogs []*types.Log
	for _, logs := range db.dirtyLogs {
		allLogs = append(allLogs, logs...)
	}

	return allLogs
}

// AddAddressToAccessList implements vm.StateDB
func (*DStateDB) AddAddressToAccessList(addr common.Address) {
	// Nothing to add, we do not care about this currently
}

// AddSlotToAccessList implements vm.StateDB
func (*DStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	// Nothing to add, we do not care about this currently
}

// AddressInAccessList implements vm.StateDB
func (*DStateDB) AddressInAccessList(addr common.Address) bool {
	// FIXME We should implement this so that correct gas information is tracked, right now we always returns false
	// false.
	return false
}

// PrepareAccessList implements vm.StateDB
func (*DStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	// Nothing to prepare, we do not care about this currently
}

// SlotInAccessList implements vm.StateDB
func (*DStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	// FIXME We should implement this so that correct gas information is tracked, right now we always returns false
	// false.
	return true, false
}
