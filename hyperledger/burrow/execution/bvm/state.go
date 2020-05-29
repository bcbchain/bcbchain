package bvm

import (
	"github.com/bcbchain/sdk/sdk/bn"
	"fmt"
	crypto2 "github.com/bcbchain/bclib/tendermint/go-crypto"

	"github.com/bcbchain/bcbchain/hyperledger/burrow/crypto"

	"github.com/bcbchain/bcbchain/hyperledger/burrow/acm"
	"github.com/bcbchain/bcbchain/hyperledger/burrow/acm/acmstate"
	"github.com/bcbchain/bcbchain/hyperledger/burrow/binary"
	"github.com/bcbchain/bcbchain/hyperledger/burrow/execution/errors"
)

type Interface interface {
	Reader
	Writer
	// Capture any errors when accessing or writing state - will return nil if no errors have occurred so far
	errors.Provider
	errors.Sink
	// Create a new cached state over this one inheriting any cache options
	NewCache(cacheOptions ...acmstate.CacheOption) Interface
	// Sync this state cache to into its originator
	Sync() errors.CodedError
}

type Reader interface {
	GetStorage(address crypto.BVMAddress, key binary.Word256) []byte
	GetBalance(address crypto.BVMAddress) bn.Number
	GetToken() crypto2.Address
	GetBVMCode(address crypto.BVMAddress) acm.ByteCode
	GetWASMCode(address crypto.BVMAddress) acm.ByteCode
	Exists(address crypto.BVMAddress) bool
	GetBlockHash(blockNumber uint64) (binary.Word256, error)
}

type Writer interface {
	CreateAccount(address crypto.BVMAddress)
	InitCode(address crypto.BVMAddress, code []byte)
	InitWASMCode(address crypto.BVMAddress, code []byte)
	RemoveAccount(address crypto.BVMAddress)
	SetStorage(address crypto.BVMAddress, key binary.Word256, value []byte)
	AddToBalance(address crypto.BVMAddress, amount bn.Number)
	SubtractFromBalance(address crypto.BVMAddress, amount bn.Number)
}

type State struct {
	// Where we sync
	backend acmstate.ReaderWriter
	// Block chain info
	blockHashGetter func(height uint64) []byte
	// Cache this State wraps
	cache *acmstate.Cache
	// Any error that may have occurred
	error errors.CodedError
	// In order for nested cache to inherit any options
	cacheOptions []acmstate.CacheOption
}

func NewState(st acmstate.ReaderWriter, blockHashGetter func(height uint64) []byte, cacheOptions ...acmstate.CacheOption) *State {
	return &State{
		backend:         st,
		blockHashGetter: blockHashGetter,
		cache:           acmstate.NewCache(st, cacheOptions...),
		cacheOptions:    cacheOptions,
	}
}

func (st *State) NewCache(cacheOptions ...acmstate.CacheOption) Interface {
	return NewState(st.cache, st.blockHashGetter, append(st.cacheOptions, cacheOptions...)...)
}

func (st *State) Sync() errors.CodedError {
	// Do not sync if we have erred
	if st.error != nil {
		return st.error
	}
	err := st.cache.Sync(st.backend)
	if err != nil {
		return errors.AsException(err)
	}
	return nil
}

func (st *State) Error() errors.CodedError {
	if st.error == nil {
		return nil
	}
	return st.error
}

// Errors pushed to state may end up in TxExecutions and therefore the merkle state so it is essential that errors are
// deterministic and independent of the code path taken to execution (e.g. replay takes a different path to that of
// normal consensus reactor so stack traces may differ - as they may across architectures)
func (st *State) PushError(err error) {
	if st.error == nil {
		// Make sure we are not wrapping a known nil value
		ex := errors.AsException(err)
		if ex != nil {
			st.error = ex
		}
	}
}

// Reader

func (st *State) GetStorage(address crypto.BVMAddress, key binary.Word256) []byte {
	value, err := st.cache.GetStorage(address, key)
	if err != nil {
		st.PushError(err)
		return []byte{}
	}
	return value
}

func (st *State) GetBalance(address crypto.BVMAddress) bn.Number {
	acc := st.account(address)
	if acc == nil {
		return bn.N(0)
	}
	return acc.Balance
}

//func (st *State) GetTokenAddr(address crypto.BVMAddress) crypto.BVMAddress {
//	acc := st.account(address)
//	if acc == nil {
//		return crypto.ZeroAddress
//	}
//	return acc.BVMToken
//}

func (st *State) GetBVMCode(address crypto.BVMAddress) acm.ByteCode {
	acc := st.account(address)
	if acc == nil {
		return nil
	}
	return acc.BVMCode
}

func (st *State) GetToken() crypto2.Address {
	return st.backend.GetToken()
}

func (st *State) GetWASMCode(address crypto.BVMAddress) acm.ByteCode {
	acc := st.account(address)
	if acc == nil {
		return nil
	}

	return acc.WASMCode
}

func (st *State) Exists(address crypto.BVMAddress) bool {
	acc, err := st.cache.GetAccount(address)
	if err != nil {
		st.PushError(err)
		return false
	}
	if acc == nil {
		return false
	}
	return true
}

// Writer

func (st *State) CreateAccount(address crypto.BVMAddress) {
	if st.Exists(address) {
		st.PushError(errors.ErrorCodef(errors.ErrorCodeDuplicateAddress,
			"tried to create an account at an address that already exists: %v", address))
		return
	}

	st.updateAccount(&acm.Account{Address: address, Balance: bn.N(0)})
}

func (st *State) InitCode(address crypto.BVMAddress, code []byte) {
	acc := st.mustAccount(address)
	if acc == nil {
		st.PushError(errors.ErrorCodef(errors.ErrorCodeInvalidAddress,
			"tried to initialise code for an account that does not exist: %v", address))
		return
	}
	if acc.BVMCode != nil || acc.WASMCode != nil {
		st.PushError(errors.ErrorCodef(errors.ErrorCodeIllegalWrite,
			"tried to initialise code for a contract that already exists: %v", address))
		return
	}
	acc.BVMCode = code
	acc.BVMToken = st.cache.GetToken()
	st.updateAccount(acc)
}

func (st *State) InitWASMCode(address crypto.BVMAddress, code []byte) {
	acc := st.mustAccount(address)
	if acc == nil {
		st.PushError(errors.ErrorCodef(errors.ErrorCodeInvalidAddress,
			"tried to initialise code for an account that does not exist: %v", address))
		return
	}
	if acc.BVMCode != nil || acc.WASMCode != nil {
		st.PushError(errors.ErrorCodef(errors.ErrorCodeIllegalWrite,
			"tried to initialise code for a contract that already exists: %v", address))
		return
	}
	acc.WASMCode = code
	acc.BVMToken = st.cache.GetToken()
	st.updateAccount(acc)
}

func (st *State) RemoveAccount(address crypto.BVMAddress) {
	if !st.Exists(address) {
		st.PushError(errors.ErrorCodef(errors.ErrorCodeDuplicateAddress,
			"tried to remove an account at an address that does not exist: %v", address))
		return
	}
	st.removeAccount(address)
}

func (st *State) SetStorage(address crypto.BVMAddress, key binary.Word256, value []byte) {
	err := st.cache.SetStorage(address, key, value)
	if err != nil {
		st.PushError(err)
	}
}

func (st *State) AddToBalance(address crypto.BVMAddress, amount bn.Number) {
	acc := st.mustAccount(address)
	if acc == nil {
		return
	}
	st.PushError(acc.AddToBalance(amount))
	st.updateAccount(acc)
}

func (st *State) SubtractFromBalance(address crypto.BVMAddress, amount bn.Number) {
	acc := st.mustAccount(address)
	if acc == nil {
		return
	}
	st.PushError(acc.SubtractFromBalance(amount))
	st.updateAccount(acc)
}

func (st *State) GetBlockHash(height uint64) (binary.Word256, error) {
	hash := st.blockHashGetter(height)
	if len(hash) == 0 {
		st.PushError(fmt.Errorf("got empty BlockHash from blockHashGetter"))
	}
	return binary.LeftPadWord256(hash), nil
}

// Helpers

func (st *State) account(address crypto.BVMAddress) *acm.Account {
	acc, err := st.cache.GetAccount(address)
	if err != nil {
		st.PushError(err)
	}
	return acc
}

func (st *State) mustAccount(address crypto.BVMAddress) *acm.Account {
	acc := st.account(address)
	if acc == nil {
		st.PushError(errors.ErrorCodef(errors.ErrorCodeIllegalWrite,
			"attempted to modify non-existent account: %v", address))
	}
	return acc
}

func (st *State) updateAccount(account *acm.Account) {
	err := st.cache.UpdateAccount(account)
	if err != nil {
		st.PushError(err)
	}
}

func (st *State) removeAccount(address crypto.BVMAddress) {
	err := st.cache.RemoveAccount(address)
	if err != nil {
		st.PushError(err)
	}
}
