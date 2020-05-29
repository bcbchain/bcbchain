package acmstate

import (
	"github.com/bcbchain/bcbchain/hyperledger/burrow/crypto"
	goCrypto "github.com/bcbchain/bclib/tendermint/go-crypto"

	"github.com/bcbchain/bcbchain/hyperledger/burrow/acm"
	"github.com/bcbchain/bcbchain/hyperledger/burrow/binary"
)

type AccountGetter interface {
	// Get an account by its address return nil if it does not exist (which should not be an error)
	GetAccount(address crypto.BVMAddress) (*acm.Account, error)
	GetToken() goCrypto.Address
}

type AccountIterable interface {
	// Iterates through accounts calling passed function once per account, if the consumer
	// returns true the iteration breaks and returns true to indicate it iteration
	// was escaped
	IterateAccounts(consumer func(*acm.Account) error) (err error)
}

type AccountUpdater interface {
	// Updates the fields of updatedAccount by address, creating the account
	// if it does not exist
	UpdateAccount(updatedAccount *acm.Account) error
	// Remove the account at address
	RemoveAccount(address crypto.BVMAddress) error
	SetToken(token goCrypto.Address)
}

type StorageGetter interface {
	// Retrieve a 32-byte value stored at key for the account at address, return Zero256 if key does not exist but
	// error if address does not
	GetStorage(address crypto.BVMAddress, key binary.Word256) (value []byte, err error)
}

type StorageSetter interface {
	// Store a 32-byte value at key for the account at address, setting to Zero256 removes the key
	SetStorage(address crypto.BVMAddress, key binary.Word256, value []byte) error
}

type StorageIterable interface {
	// Iterates through the storage of account ad address calling the passed function once per account,
	// if the iterator function returns true the iteration breaks and returns true to indicate it iteration
	// was escaped
	IterateStorage(address crypto.BVMAddress, consumer func(key binary.Word256, value []byte) error) (err error)
}

type AccountStats struct {
	AccountsWithCode    uint64
	AccountsWithoutCode uint64
}

type AccountStatsGetter interface {
	GetAccountStats() AccountStats
}

// Compositions

// Read-only account and storage state
type Reader interface {
	AccountGetter
	StorageGetter
}

type Iterable interface {
	AccountIterable
	StorageIterable
}

// Read and list account and storage state
type IterableReader interface {
	Iterable
	Reader
}

type IterableStatsReader interface {
	Iterable
	Reader
	AccountStatsGetter
}

type Writer interface {
	AccountUpdater
	StorageSetter
}

// Read and write account and storage state
type ReaderWriter interface {
	Reader
	Writer
}

type IterableReaderWriter interface {
	Iterable
	Reader
	Writer
}
