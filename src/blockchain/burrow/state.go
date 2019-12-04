package burrow

import (
	"blockchain/common/statedbhelper"
	"blockchain/statedb"
	"common/jsoniter"

	"github.com/hyperledger/burrow/crypto"
	goCrypto "github.com/tendermint/go-crypto"

	"github.com/tendermint/tmlibs/log"

	"github.com/hyperledger/burrow/acm"
	"github.com/hyperledger/burrow/acm/acmstate"
	"github.com/hyperledger/burrow/binary"
)

var _ acmstate.ReaderWriter = &State{}

// 去调用 blockchain/statedb 里的 Get/Set
type State struct {
	transID   int64
	txID      int64
	tokenAddr goCrypto.Address
	logger    log.Logger
}

func (s *State) SetToken(token goCrypto.Address) {
	s.tokenAddr = token
}

func (s *State) GetToken() goCrypto.Address {
	return s.tokenAddr
}

func (s *State) UpdateAccount(updated *acm.Account) error {
	val, _ := jsoniter.Marshal([]string{statedbhelper.KeyOfAccountNonce(crypto.ToAddr(updated.Address))})
	statedb.Set(s.transID, s.txID, statedbhelper.KeyOfAccount(crypto.ToAddr(updated.Address)), val)
	statedbhelper.SetBVMBalance(s.transID, s.txID, crypto.ToAddr(updated.Address), s.tokenAddr, updated.Balance)
	if updated.EVMCode != nil {
		statedb.Set(s.transID, s.txID, codeKey(updated.Address), updated.EVMCode)
		if s.tokenAddr != "" {
			statedb.Set(s.transID, s.txID, tokenKey(updated.Address), []byte(s.tokenAddr))
		} else {
			statedb.Set(s.transID, s.txID, tokenKey(updated.Address), []byte(updated.EVMToken))
		}
	}
	return nil
}

func (s *State) RemoveAccount(removed crypto.EVMAddress) error {
	addr := crypto.ToAddr(removed)
	statedb.Set(s.transID, s.txID, statedbhelper.KeyOfAccount(addr), nil)
	statedb.Set(s.transID, s.txID, statedbhelper.KeyOfAccountNonce(addr), nil)
	statedb.Set(s.transID, s.txID, codeKey(removed), nil)
	statedb.Set(s.transID, s.txID, tokenKey(removed), nil)
	return nil
}

func (s *State) SetStorage(address crypto.EVMAddress, key binary.Word256, value []byte) error {
	statedb.Set(s.transID, s.txID, storageKey(address, key), value)
	return nil
}

func (s *State) GetAccount(address crypto.EVMAddress) (*acm.Account, error) {
	addr := crypto.ToAddr(address)
	s.logger.Debug("enter GetAccount:", "address", address, "stateToken", s.tokenAddr,
		"transID", s.transID, "txID", s.txID)
	if statedb.Get(s.transID, s.txID, statedbhelper.KeyOfAccount(addr)) == nil {
		return nil, nil
	}
	act := acm.Account{Address: address}
	balance := statedbhelper.BVMBalanceOf(s.transID, s.txID, addr, s.tokenAddr)
	evmToken := statedb.Get(s.transID, s.txID, tokenKey(address))
	evmCode := statedb.Get(s.transID, s.txID, codeKey(address))
	act.Balance = balance
	act.EVMToken = string(evmToken)
	act.EVMCode = evmCode

	s.logger.Debug("GetAccount return:", "account", act.String())
	return &act, nil
}

func (s *State) GetStorage(address crypto.EVMAddress, key binary.Word256) (value []byte, err error) {
	byt := statedb.Get(s.transID, s.txID, storageKey(address, key))
	return byt, nil
}

func tokenKey(addr crypto.EVMAddress) string {
	return "evm/" + crypto.ToAddr(addr) + "/evmToken"
}

func codeKey(addr crypto.EVMAddress) string {
	return "evm/" + crypto.ToAddr(addr) + "/evmCode"
}

func storageKey(addr crypto.EVMAddress, key binary.Word256) string {
	return "evm/" + crypto.ToAddr(addr) + "/storage/" + key.String()
}

func NewState(transID, txID int64, logger log.Logger) *State {
	return &State{
		transID: transID,
		txID:    txID,
		logger:  logger,
	}
}
