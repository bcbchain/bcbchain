package burrow

import (
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/sdk/sdk/std"
	"github.com/bcbchain/bclib/jsoniter"
	"encoding/hex"
	"encoding/json"
	"github.com/bcbchain/bcbchain/hyperledger/burrow/abi"
	"sort"
	"strings"

	"github.com/bcbchain/bcbchain/hyperledger/burrow/crypto"
	goCrypto "github.com/bcbchain/bclib/tendermint/go-crypto"

	"github.com/bcbchain/bcbchain/hyperledger/burrow/acm"
	"github.com/bcbchain/bcbchain/hyperledger/burrow/acm/acmstate"
	"github.com/bcbchain/bcbchain/hyperledger/burrow/binary"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
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
	statedbhelper.Set(s.transID, s.txID, statedbhelper.KeyOfAccount(crypto.ToAddr(updated.Address)), val)
	statedbhelper.SetBalance(s.transID, s.txID, crypto.ToAddr(updated.Address), s.tokenAddr, updated.Balance)
	if updated.BVMCode != nil {
		statedbhelper.Set(s.transID, s.txID, codeKey(updated.Address), updated.BVMCode)
		if s.tokenAddr != "" {
			statedbhelper.Set(s.transID, s.txID, tokenKey(updated.Address), []byte(s.tokenAddr))
		} else {
			statedbhelper.Set(s.transID, s.txID, tokenKey(updated.Address), []byte(updated.BVMToken))
		}
	}

	err := updated.AddAccountTokenKey(s.transID, s.txID, std.KeyOfAccountToken(crypto.ToAddr(updated.Address), s.GetToken()))
	if err != nil {
		return err
	}

	return nil
}

func (s *State) RemoveAccount(removed crypto.BVMAddress) error {
	addr := crypto.ToAddr(removed)

	keyOfAccountToken, err := statedbhelper.Get(s.transID, s.txID, statedbhelper.KeyOfAccount(addr))
	if err != nil {
		return err
	}

	statedbhelper.Set(s.transID, s.txID, statedbhelper.KeyOfAccount(addr), nil)
	statedbhelper.Set(s.transID, s.txID, statedbhelper.KeyOfAccountToken(addr, s.tokenAddr), nil)
	statedbhelper.Set(s.transID, s.txID, string(keyOfAccountToken), nil)
	statedbhelper.Set(s.transID, s.txID, contractInfoKey(addr), nil)
	statedbhelper.Set(s.transID, s.txID, codeKey(removed), nil)
	statedbhelper.Set(s.transID, s.txID, tokenKey(removed), nil)

	return nil
}

func (s *State) SetStorage(address crypto.BVMAddress, key binary.Word256, value []byte) error {
	statedbhelper.Set(s.transID, s.txID, storageKey(address, key), value)
	return nil
}

func (s *State) GetAccount(address crypto.BVMAddress) (*acm.Account, error) {
	addr := crypto.ToAddr(address)
	s.logger.Debug("enter GetAccount:", "address", address, "stateToken", s.tokenAddr,
		"transID", s.transID, "txID", s.txID)
	value, _ := statedbhelper.Get(s.transID, s.txID, statedbhelper.KeyOfAccount(addr))
	if len(value) == 0 {
		return nil, nil
	}
	act := acm.Account{Address: address}
	balance := statedbhelper.BalanceOf(s.transID, s.txID, addr, s.tokenAddr)
	bvmToken, _ := statedbhelper.Get(s.transID, s.txID, tokenKey(address))
	bvmCode, _ := statedbhelper.Get(s.transID, s.txID, codeKey(address))
	act.Balance = balance
	act.BVMToken = string(bvmToken)
	act.BVMCode = bvmCode

	s.logger.Debug("GetAccount return:", "account", act.String())
	return &act, nil
}

func (s *State) GetStorage(address crypto.BVMAddress, key binary.Word256) (value []byte, err error) {
	byt, _ := statedbhelper.Get(s.transID, s.txID, storageKey(address, key))
	return byt, nil
}

func (s *State) SetContractInfo(transID, txID, ChainVersion int64, token, bvmAddr, senderAddr goCrypto.Address, abiStr string) (err error) {

	if abiStr == "" {
		s.logger.Debug("bvm", "abiFile is empty, please check")
		return
	}

	abiStr = strings.Replace(abiStr, "\n", "", -1)
	abiStr = strings.Replace(abiStr, "\t", "", -1)
	abiStr = strings.Replace(abiStr, `\`, "", -1)

	newAbi, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		return
	}

	Methods := make([]std.BvmMethod, 0)
	Events := make([]string, 0)
	keys := make([]string, 0)
	for k := range newAbi.Methods {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := newAbi.Methods[k]
		var Method std.BvmMethod
		Method.MethodID = hex.EncodeToString(v.ID())
		Method.ProtoType = v.ShortString()
		Methods = append(Methods, Method)
	}

	eventKeys := make([]string, 0)
	for k := range newAbi.Events {
		eventKeys = append(eventKeys, k)
	}
	sort.Strings(eventKeys)

	for _, k := range eventKeys {
		Events = append(Events, newAbi.Events[k].ShortString())
	}

	BVMContract := new(std.BvmContract)
	BVMContract.Methods = Methods
	BVMContract.Events = Events
	BVMContract.Address = bvmAddr
	BVMContract.Token = token
	BVMContract.Deployer = senderAddr
	BVMContract.BvmAbi = abiStr
	BVMContract.ChainVersion = ChainVersion

	err = setBVMContract(transID, txID, BVMContract)
	if err != nil {
		s.logger.Debug("bvm", "set bvmContract Info to db failed")
		return
	}

	return
}

func GetContractInfo(address goCrypto.Address) (contract *std.BvmContract) {

	res, _ := statedbhelper.Get(0, 0, contractInfoKey(address))
	if len(res) == 0 {
		return
	}

	contract = new(std.BvmContract)
	if err := json.Unmarshal(res, &contract); err != nil {
		return
	}

	return
}

func setBVMContract(transID, txID int64, contract *std.BvmContract) error {
	key := contractInfoKey(contract.Address)
	value, err := jsoniter.Marshal(contract)
	if err != nil {
		panic(err)
	}
	statedbhelper.Set(transID, txID, key, value)

	return nil
}

func tokenKey(addr crypto.BVMAddress) string {
	return "/bvm/" + crypto.ToAddr(addr) + "/bvmToken"
}

func codeKey(addr crypto.BVMAddress) string {
	return "/bvm/" + crypto.ToAddr(addr) + "/bvmCode"
}

func storageKey(addr crypto.BVMAddress, key binary.Word256) string {
	return "/bvm/" + crypto.ToAddr(addr) + "/storage/" + key.String()
}

func contractInfoKey(contractAddr goCrypto.Address) string {
	return "/bvm/contract/" + contractAddr
}

func NewState(transID, txID int64, logger log.Logger) *State {
	return &State{
		transID: transID,
		txID:    txID,
		logger:  logger,
	}
}
