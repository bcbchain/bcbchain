package evm

import (
	"bytes"
	"fmt"

	"github.com/hyperledger/burrow/crypto"
	goCrypto "github.com/tendermint/go-crypto"

	"github.com/hyperledger/burrow/acm"
	"github.com/hyperledger/burrow/acm/acmstate"
	. "github.com/hyperledger/burrow/binary"
)

type FakeAppState struct {
	Accounts  map[crypto.EVMAddress]*acm.Account
	Storage   map[string][]byte
	tokenAddr goCrypto.Address
}

func (fas *FakeAppState) GetToken() goCrypto.Address {
	return fas.tokenAddr
}

func (fas *FakeAppState) SetToken(token goCrypto.Address) {
	fas.tokenAddr = token
}

var _ acmstate.ReaderWriter = &FakeAppState{}

func (fas *FakeAppState) GetAccount(addr crypto.EVMAddress) (*acm.Account, error) {
	account := fas.Accounts[addr]
	return account, nil
}

func (fas *FakeAppState) UpdateAccount(account *acm.Account) error {
	fas.Accounts[account.Address] = account
	return nil
}

func (fas *FakeAppState) RemoveAccount(address crypto.EVMAddress) error {
	_, ok := fas.Accounts[address]
	if !ok {
		panic(fmt.Sprintf("Invalid account addr: %s", address))
	} else {
		// Remove account
		delete(fas.Accounts, address)
	}
	return nil
}

func (fas *FakeAppState) GetStorage(addr crypto.EVMAddress, key Word256) ([]byte, error) {
	_, ok := fas.Accounts[addr]
	if !ok {
		return []byte{}, nil
		// panic(fmt.Sprintf("Invalid account addr: %s", addr))
	}

	value, ok := fas.Storage[addr.String()+key.String()]
	if ok {
		return value, nil
	} else {
		return []byte{}, nil
	}
}

func (fas *FakeAppState) SetStorage(addr crypto.EVMAddress, key Word256, value []byte) error {
	_, ok := fas.Accounts[addr]
	if !ok {

		fmt.Println("\n\n", fas.accountsDump())
		panic(fmt.Sprintf("Invalid account addr: %s", addr))
	}

	fas.Storage[addr.String()+key.String()] = value
	return nil
}

func (fas *FakeAppState) accountsDump() string {
	buf := new(bytes.Buffer)
	fmt.Fprint(buf, "Dumping Accounts...", "\n")
	for _, acc := range fas.Accounts {
		fmt.Fprint(buf, acc.Address, "\n")
	}
	return buf.String()
}
