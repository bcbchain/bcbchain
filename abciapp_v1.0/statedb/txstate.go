package statedb

import (
	"github.com/bcbchain/bcbchain/statedb"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"sort"
	"strconv"

	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/types"
	"github.com/pkg/errors"
)

type TxState struct {
	StateDB         *StateDB //read
	ContractAddress smc.Address
	SenderAddress   smc.Address
	Tx              *statedb.Tx
}

func keyOfBlackList(key string) string {
	return "/blacklist/" + key
}

func (txState *TxState) CheckBlackAddress(address smc.Address) bool {
	key := keyOfBlackList(address)

	resBytes, _ := txState.Get(key)

	if resBytes == nil {
		return false
	}

	var b string
	_ = json.Unmarshal(resBytes, &b)

	return b == "true"
}

func (txState *TxState) GetBalance(exAddress smc.Address, tokenAddress smc.Address) (big.Int, error) {
	//根据合约地址，在内部构造出key
	key := keyOfAccountToken(exAddress, tokenAddress)
	tokenData, err := txState.Get(key)
	if err != nil {
		return *big.NewInt(0), err
	}
	if tokenData == nil {
		return *big.NewInt(0), nil
	}

	var token types.TokenBalance
	err = json.Unmarshal(tokenData, &token)
	if err != nil {
		return *big.NewInt(0), err
	}
	return token.Balance, nil
}

func (txState *TxState) SetBalance(exAddress smc.Address, tokenAddress smc.Address, value big.Int) error {
	//根据合约地址，账户地址，构造出key，然后保存
	token := types.TokenBalance{tokenAddress, value}
	tokenData, err := json.Marshal(&token)
	if err != nil {
		return err
	}
	key := keyOfAccount(exAddress)
	childKey := keyOfAccountToken(exAddress, tokenAddress)
	data, err := txState.Get(childKey)
	if err != nil {
		return err
	}
	if data == nil {
		err = txState.addChildKey(key, childKey)
		if err != nil {
			return err
		}
	}
	return txState.Set(childKey, tokenData)
}

func (txState *TxState) GetToken(tokenAddr smc.Address) (*types.IssueToken, error) {
	key := keyOfToken(tokenAddr)
	tokenData, err := txState.Get(key)
	if err != nil {
		return nil, err
	}
	if tokenData == nil {
		return nil, nil
	}

	var token types.IssueToken
	err = json.Unmarshal(tokenData, &token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (txState *TxState) GetGenesisToken() (*types.IssueToken, error) {
	key := keyOfGenesisToken()
	tokenData, err := txState.Get(key)
	if err != nil {
		return nil, err
	}
	if tokenData == nil {
		return nil, nil
	}

	var token types.IssueToken
	err = json.Unmarshal(tokenData, &token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

//只设置Token信息，不设置账户的代币
func (txState *TxState) SetToken(issueToken *types.IssueToken) error {
	tokenData, err := json.Marshal(issueToken)
	if err != nil {
		return err
	}
	key := keyOfToken(issueToken.Address)
	err = txState.Set(key, tokenData)
	if err != nil {
		return err
	}

	err = txState.addChildKey(keyOfTokenAll(), issueToken.Address)
	if err != nil {
		return err
	}
	addressData, err := json.Marshal(issueToken.Address)
	if err != nil {
		return err
	}
	err = txState.Set(keyOfTokenName(issueToken.Name), addressData)
	if err != nil {
		return err
	}
	return txState.Set(keyOfTokenSymbol(issueToken.Symbol), addressData)
}

func (txState *TxState) SetTokenContract(contract *types.Contract) error {
	contractData, err := json.Marshal(contract)
	if err != nil {
		return err
	}
	key := keyOfContract(contract.Address)
	err = txState.Set(key, contractData) //增加智能合约
	if err != nil {
		return err
	}

	//把智能合约地址增加到列表中
	err = txState.addChildKey(keyOfContractAll(), contract.Address)
	if err != nil {
		return err
	}
	//保存智能合约Owner的相关信息
	key = keyOfAccount(contract.Owner)
	childKey := keyOfAccountContracts(contract.Owner)
	err = txState.addChildKey(childKey, contract.Address)
	if err != nil {
		return err
	}
	return txState.addChildKey(key, childKey)
}

func (txState *TxState) SetStrategys(strategys []types.RewardStrategy) error {
	key := keyOfRewardStrategys()
	strategysData, err := json.Marshal(strategys)
	if err != nil {
		return err
	}
	return txState.Set(key, strategysData)
}

func (txState *TxState) GetStrategys() ([]types.RewardStrategy, error) {
	key := keyOfRewardStrategys()
	value, err := txState.Get(key)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}

	var strategys []types.RewardStrategy
	err = json.Unmarshal(value, &strategys)
	if err != nil {
		return nil, err
	}
	return strategys, nil
}

func (txState *TxState) DeleteContractAddr(exAddress smc.Address, contractAddr smc.Address) error {
	//保存智能合约Owner的相关信息
	childKey := keyOfAccountContracts(exAddress)
	return txState.deleteChildKey(childKey, contractAddr)
}

func (txState *TxState) GetTokenAddrByName(name string) (smc.Address, error) {
	value, err := txState.Get(keyOfTokenName(name))
	if err != nil {
		return "", err
	}
	if value == nil {
		return "", nil
	}

	var address smc.Address
	err = json.Unmarshal(value, &address)
	if err != nil {
		return "", err
	}
	return address, nil
}

func (txState *TxState) GetTokenAddrBySymbol(name string) (smc.Address, error) {
	value, err := txState.Get(keyOfTokenSymbol(name))
	if err != nil {
		return "", err
	}
	if value == nil {
		return "", nil
	}

	var address smc.Address
	err = json.Unmarshal(value, &address)
	if err != nil {
		return "", err
	}
	return address, nil
}

func (txState *TxState) SetBaseGasPrice(gasPrice uint64) error {
	gasPriceData, err := json.Marshal(&gasPrice)
	if err != nil {
		return err
	}

	key := keyOfTokenBaseGasPrice()
	err = txState.Set(key, gasPriceData) //增加智能合约
	if err != nil {
		return err
	}

	return nil
}

func (txState *TxState) GetBaseGasPrice() uint64 {
	value, err := txState.Get(keyOfTokenBaseGasPrice())
	if err != nil {
		panic(err)
	}
	if value == nil {
		panic(errors.New("Base gas price is null"))
	}

	var price uint64
	err = json.Unmarshal(value, &price)
	if err != nil {
		panic(err)
	}
	return price
}

func (txState *TxState) GetChainID() string {
	value, err := txState.Get(keyOfGenesisChainId())
	if err != nil {
		panic(err)
	}

	if value == nil {
		panic(errors.New("ChainID is null"))
	}

	return string(value)
}

func (txState *TxState) GetGas(contractAddr smc.Address, methodId uint32) (uint64, error) {
	contract, err := txState.StateDB.GetContract(contractAddr)
	if err != nil {
		return 0, err
	}

	if contract == nil {
		return 0, errors.New("contractAddr invalid!")
	}

	for _, m := range contract.Methods {
		intMethodId, err := strconv.ParseUint(m.MethodId, 16, 32)
		if err != nil {
			return 0, err
		}
		if intMethodId == uint64(methodId) {
			return uint64(m.Gas), nil
		}
	}
	return 0, errors.New("methodId invalid!")
}

func (txState *TxState) SetUDCNonce(nonce uint64) error {
	key := keyOfUDCNonce()
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, nonce)
	txState.StateDB.Set(key, buf)
	return nil
}

func (txState *TxState) GetUDCNonce() (uint64, error) {
	key := keyOfUDCNonce()
	value, err := txState.Get(key)
	if err != nil {
		return 0, err
	}

	var nonce uint64
	err = json.Unmarshal(value, &nonce)
	if err != nil {
		return 0, err
	}

	return nonce, nil
}

func (txState *TxState) SetUDCOrder(udcOrder *types.UDCOrder) error {
	//保存到：/udc/key
	udcOrderBytes, err := json.Marshal(udcOrder)
	if err != nil {
		return err
	}

	key := keyOfUDCOrder(udcOrder.UDCHash)
	txState.StateDB.Set(key, udcOrderBytes)

	//保存到：account/ex/key1/udchashlist
	key = keyOfAccountUDCHashList(udcOrder.Owner)
	return txState.addChildKey(key, hex.EncodeToString(udcOrder.UDCHash))
}

func (txState *TxState) GetUDCOrder(udcHash []byte) (*types.UDCOrder, error) {
	key := keyOfUDCOrder(udcHash)
	value, err := txState.Get(key)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, errors.New("UDCOrder is empty")
	}

	var udcOrder types.UDCOrder
	err = json.Unmarshal(value, &udcOrder)
	if err != nil {
		return nil, err
	}

	return &udcOrder, nil
}

func (txState *TxState) SetValidator(validator *types.Validator) error {
	key := keyOfValidator(validator.NodeAddr)
	validatorData, err := json.Marshal(validator)
	if err != nil {
		return err
	}

	err = txState.Set(key, validatorData)
	if err != nil {
		return err
	}
	return txState.addChildKey(keyOfValidators(), validator.NodeAddr)
}

func (txState *TxState) GetAllValidatorPubKeys() ([]string, error) {
	return txState.getChildKeys(keyOfValidators())
}

func (txState *TxState) getChildKeys(key string) ([]string, error) {
	value, err := txState.Get(key)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, err
	}

	var strChildKeys []string
	err = json.Unmarshal(value, &strChildKeys)
	if err != nil {
		return nil, err
	}
	return strChildKeys, nil
}

func (txState *TxState) AddChildKey(key string, childKey string) error {
	return txState.addChildKey(key, childKey)
}

func (txState *TxState) addChildKey(key string, childKey string) error {
	childKeys, err := txState.getChildKeys(key)
	if err != nil {
		return err
	}

	index := sort.SearchStrings(childKeys, childKey)
	if index == len(childKeys) { // insert tail
		childKeys = append(childKeys, childKey)
	} else if childKeys[index] == childKey { //equal
		return nil
	} else {
		childKeys = append(childKeys[:index], append([]string{childKey}, childKeys[index:]...)...)
	}

	childKeysData, err := json.Marshal(childKeys)
	if err != nil {
		return err
	}
	_ = txState.Set(key, []byte(childKeysData))
	return nil
}

func (txState *TxState) deleteChildKey(key string, childKey string) error {
	childKeys, err := txState.getChildKeys(key)
	if err != nil {
		return err
	}

	index := sort.SearchStrings(childKeys, childKey)
	childKeys = append(childKeys[:index], childKeys[index+1:]...)

	childKeysData, err := json.Marshal(childKeys)
	if err != nil {
		return err
	}
	_ = txState.Set(key, []byte(childKeysData))
	return nil
}

//按照Tx缓存、block缓存、数据库顺序找
func (txState *TxState) Get(key string) ([]byte, error) {
	if txState.Tx == nil {
		return txState.StateDB.Get(key)
	}

	v := txState.Tx.Get(key)
	if len(v) > 0 {
		return v, nil
	} else {
		return txState.StateDB.Get(key)
	}
}

//如果没有初始化Tx缓存直接报错，如果想直接写入block缓存,可以调用StateDB的设置接口
func (txState *TxState) Set(key string, value []byte) error {
	if txState.Tx == nil {
		return errors.New("TxBuffer is nil")
	}

	txState.Tx.Set(key, value)
	return nil
}

//提交Tx缓存到block缓存，清除Tx缓存,把Tx缓存中的内容转换为字节数组返回
func (txState *TxState) CommitTx() ([]byte, map[string][]byte) {
	return txState.Tx.Commit()
}

func (txState *TxState) RollbackTx() {
	txState.Tx.Rollback()
}

func (txState *TxState) GetContractsListByName(name string) ([]smc.Address, error) {
	var contracts []smc.Address
	childKeys, err := txState.StateDB.GetContractAddrList()
	if err != nil {
		return nil, err
	}

	for _, k := range childKeys {
		contract, err := txState.StateDB.GetContract(k)
		if err != nil {
			return nil, err
		}
		if contract.Name == name {
			contracts = append(contracts, contract.Address)
		}
	}
	return contracts, nil
}
