package statedbhelper

import (
	"bytes"
	"fmt"
	"github.com/bcbchain/bcbchain/statedb"
	"github.com/bcbchain/bclib/types"
	"github.com/bcbchain/sdk/sdk/bn"
	"github.com/bcbchain/sdk/sdk/std"
	"strconv"
	"sync"

	abci "github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/sdk/sdk/jsoniter"
	types2 "github.com/bcbchain/sdk/sdk/types"
)

var (
	stateDB        *statedb.StateDB
	transactionMap sync.Map // transactionID => *Trans

	setOnce sync.Once
	chainID string

	currentCommittableTransaction *statedb.Transaction
)

type Trans struct {
	Transaction *statedb.Transaction
	TxMap       map[int64]*statedb.Tx // txID => *statedb2.Tx
}

func Init(sdbName string, maxSnapshotCount int) {
	stateDB = statedb.New(sdbName, maxSnapshotCount)
}

func InitX(sdb *statedb.StateDB, transMap map[int64]*Trans) {
	stateDB = sdb
	transactionMap = sync.Map{}
	for id, tran := range transMap {
		transactionMap.Store(id, tran)
	}
}

//NewCommittableTransactionID create a committable transaction and return ID
func NewCommittableTransactionID() (int64, *statedb.Transaction) {
	if currentCommittableTransaction != nil {
		return currentCommittableTransaction.ID(), currentCommittableTransaction
	}

	transaction := stateDB.NewCommittableTransaction()
	currentCommittableTransaction = transaction
	transactionMap.Store(transaction.ID(), &Trans{
		Transaction: transaction,
		TxMap:       make(map[int64]*statedb.Tx),
	})

	return transaction.ID(), transaction
}

//NewRollbackTransactionID create a rollback transaction and return ID
func NewRollbackTransactionID() (int64, *statedb.Transaction) {
	transaction := stateDB.NewRollbackTransaction()
	transactionMap.Store(transaction.ID(), &Trans{
		Transaction: transaction,
		TxMap:       make(map[int64]*statedb.Tx),
	})
	return transaction.ID(), transaction
}

func RollbackStateDB(rollbackTransactions int) {
	stateDB.Rollback(rollbackTransactions)
}

func GetFromDB(key string) ([]byte, error) {
	return stateDB.Get(key), nil
}

func NewTx(transID int64) int64 {
	temp, ok := transactionMap.Load(transID)
	if !ok {
		panic("invalid transID")
	}
	trans := temp.(*Trans)

	tx := trans.Transaction.NewTx(nil, nil)
	trans.TxMap[tx.ID()] = tx
	return tx.ID()
}

func Get(transID, txID int64, key string) ([]byte, error) {
	return get(transID, txID, key), nil
}

func Set(transID, txID int64, key string, value []byte) {
	set(transID, txID, key, value)
}

func GetWorldAppState(transID, txID int64) *abci.AppState {
	key := keyOfWorldAppState()
	value := get(transID, txID, key)

	if len(value) == 0 {
		return &abci.AppState{}
	}

	var appState abci.AppState
	err := jsoniter.Unmarshal(value, &appState)
	if err != nil {
		panic(err)
	}
	return &appState
}

//SetWorldAppState set data of app state
func SetWorldAppState(transID, txID int64, appState *abci.AppState) {
	key := keyOfWorldAppState()
	appStateData, err := jsoniter.Marshal(appState)
	if err != nil {
		panic(err)
	}
	set(transID, txID, key, appStateData)
}

func SetChainIDOnce(cID string) {
	setOnce.Do(func() {
		chainID = cID
	})
}

func GetChainID() string {
	if chainID != "" {
		return chainID
	}

	value := stateDB.Get(keyOfGenesisChainID())
	if value == nil || len(value) == 0 {
		return ""
	}

	err := jsoniter.Unmarshal(value, &chainID)
	if err != nil {
		// if blockChain from v1 upgrade to v2, then "chainID" value would be []byte("xxxx"),
		// otherwise it's be marshal result
		chainID = string(value)
	}

	return chainID
}

func GetOrgID(transID, txID int64, contractAddr types.Address) string {
	key := "/contract/" + contractAddr
	res := get(transID, txID, key)
	if len(res) == 0 {
		return ""
	}

	contract := new(std.Contract)
	err := jsoniter.Unmarshal(res, contract)
	if err != nil {
		panic("state db helper get org Id err: " + err.Error())
	}

	return contract.OrgID
}

func GetOrgSigners(transID, txID int64, orgID string) []types2.PubKey {
	key := keyOfOrganization(orgID)
	res := get(transID, txID, key)
	if len(res) == 0 {
		return nil
	}

	org := new(std.Organization)
	err := jsoniter.Unmarshal(res, org)
	if err != nil {
		panic("state db helper get org err: " + err.Error())
	}

	return org.Signers
}

//GetOrgCodeHash get org code hash
func GetOrgCodeHash(transID, txID int64, orgID string) []byte {

	key := "/organization/" + orgID
	res := get(transID, txID, key)
	if len(res) == 0 {
		return []byte{}
	}

	org := new(std.Organization)
	err := jsoniter.Unmarshal(res, org)
	if err != nil {
		panic("state db helper get org err: " + err.Error())
	}

	return org.OrgCodeHash
}

//GetContractCodeHash get contract code hash
func GetContractCodeHash(transID, txID int64, contractAddr types.Address) []byte {

	key := "/contract/" + contractAddr
	res := get(transID, txID, key)

	con := new(std.Contract)
	err := jsoniter.Unmarshal(res, con)
	if err != nil {
		panic("state db helper get contracts err: " + err.Error())
	}

	return con.CodeHash
}

//GetContracts get contracts of org
func GetContracts(transID, txID int64, orgID string) []types.Address {

	key := "/organization/" + orgID
	res := get(transID, txID, key)
	if len(res) == 0 {
		return nil
	}

	org := new(std.Organization)
	err := jsoniter.Unmarshal(res, org)
	if err != nil {
		panic("state db helper get contracts err: " + err.Error())
	}

	return org.ContractAddrList
}

//GetContractMeta get contract code
func GetContractMeta(transID, txID int64, contractAddr types.Address) std.ContractMeta {

	key := keyOfContractMeta(contractAddr)
	res := get(transID, txID, key)
	if len(res) == 0 {
		return std.ContractMeta{}
	}

	contractMeta := new(std.ContractMeta)
	err := jsoniter.Unmarshal(res, contractMeta)
	if err != nil {
		panic(err)
	}

	return *contractMeta
}

//CheckOrgInfo check org address
func CheckOrgInfo(transID, txID int64, orgID, orgAddr string) bool {

	key := "/organization/" + orgID
	res := get(transID, txID, key)

	orgDev := new(std.Organization)
	err := jsoniter.Unmarshal(res, orgDev)
	if err != nil {
		panic("state db helper check org info err: " + err.Error())
	}

	//if strings.Compare(orgAddr, orgDev.OrgOwner) == 0 {
	//	return false
	//}

	return true
}

//SetAccountNonce DeliverTx需要调用此接口检查并设置nonce。设置的nonce不会因为RollbackTx而取消。
func SetAccountNonce(transID, txID int64, exAddress types.Address, nonce uint64) (nonceBuffer map[string][]byte, err error) {
	//根据合约地址，账户地址，构造出key，然后保存
	err = CheckAccountNonce(transID, txID, exAddress, nonce)
	if err != nil {
		return
	}

	type AccountInfo struct {
		Nonce uint64
	}
	account := AccountInfo{nonce}
	accountData, err := jsoniter.Marshal(&account)
	if err != nil {
		panic(err)
	}

	nonceBuffer = make(map[string][]byte)
	key := KeyOfAccount(exAddress)
	childKey := KeyOfAccountNonce(exAddress)
	data := get(transID, txID, childKey)

	temp, ok := transactionMap.Load(transID)
	if !ok {
		panic("invalid transID")
	}
	trans := temp.(*Trans)

	if data == nil || len(data) == 0 {
		accAllKeyBytes := get(transID, txID, key)
		accAllKeys := new([]string)
		if accAllKeyBytes != nil {
			err := jsoniter.Unmarshal(accAllKeyBytes, accAllKeys)
			if err != nil {
				panic(err)
			}
		}
		*accAllKeys = append(*accAllKeys, childKey)

		resBytes, err := jsoniter.Marshal(accAllKeys)
		if err != nil {
			panic(err)
		}
		nonceBuffer[key] = resBytes
		trans.Transaction.Set(key, resBytes)
	}
	nonceBuffer[childKey] = accountData
	trans.Transaction.Set(childKey, accountData)

	return
}

func SetAccountNonceEx(exAddress types.Address, nonce uint64) (err error) {
	_, err = SetAccountNonce(currentCommittableTransaction.ID(), 0, exAddress, nonce)
	return
}

//CheckAccountNonce check account's nonce
func CheckAccountNonce(transID, txID int64, exAddr types.Address, nonce uint64) error {
	type AccountInfo struct {
		Nonce uint64
	}

	key := KeyOfAccountNonce(exAddr)
	value := get(transID, txID, key)

	var lastNonce uint64
	if value == nil || len(value) == 0 {
		lastNonce = 0
	} else {
		accountInfo := new(AccountInfo)
		err := jsoniter.Unmarshal(value, accountInfo)
		if err != nil {
			panic(err)
		}
		lastNonce = accountInfo.Nonce
	}

	if nonce != (lastNonce + 1) {
		return fmt.Errorf("address:%s nonce invalid! expected: %d, got: %d", exAddr, lastNonce+1, nonce)
	}
	return nil
}

//GetContract get specified contract data with contract address
func GetContract(contractAddr types.Address) *std.Contract {
	key := keyOfContract(contractAddr)
	value := stateDB.Get(key)
	if len(value) == 0 {
		return nil
	}

	contract := std.Contract{}
	err := jsoniter.Unmarshal(value, &contract)
	if err != nil {
		panic(err)
	}
	return &contract
}

//BeginBlock a fake beginblock() for checkTx
func BeginBlock(transID int64) {
	state := GetWorldAppState(transID, 0)

	state.BlockHeight = state.BlockHeight + 1
	SetWorldAppState(transID, NewTx(transID), state)
}

//RollbackBlock rollback block changes
func RollbackBlock(transID int64) {
	trans := getTrans(transID)
	trans.Transaction.Rollback()
	transactionMap.Delete(trans.Transaction.ID())

	if currentCommittableTransaction != nil && currentCommittableTransaction.ID() == transID {
		currentCommittableTransaction = nil
	}
}

//RollbackTx rollback tx changes
func RollbackTx(transID, txID int64) {
	trans := getTrans(transID)
	tx := trans.TxMap[txID]
	tx.Rollback()
}

//CommitBlock commit block changes
func CommitBlock(transID int64) {
	trans := getTrans(transID)
	trans.Transaction.Commit()
	currentCommittableTransaction = nil

	transactionMap.Delete(trans.Transaction.ID())
}

//CommitTx commit tx changes
func CommitTx(transID, txID int64) ([]byte, map[string][]byte) {
	trans := getTrans(transID)
	tx := trans.TxMap[txID]
	return tx.Commit()
}

//CommitTx2V2 commit tx changes
func CommitTx2V1(transID int64, txBuffer map[string][]byte) {
	trans := getTrans(transID)
	trans.Transaction.BatchSet(txBuffer)
}

//BalanceOf gets account's balance of given token
func BalanceOf(transID, txID int64, addr types.Address, token types.Address) bn.Number {
	key := KeyOfAccountToken(addr, token)
	value := get(transID, txID, key)
	if value == nil || len(value) == 0 {
		return bn.N(0)
	}
	acc := new(std.AccountInfo)
	err := jsoniter.Unmarshal(value, acc)
	if err != nil {
		panic(err)
	}
	return acc.Balance
}

//SetBalance set account's balance of given token to given value
func SetBalance(transID, txID int64, addr types.Address, token types.Address, value bn.Number) {
	var acc std.AccountInfo
	acc.Address = token
	acc.Balance = value

	resBytes, err := jsoniter.Marshal(acc)
	if err != nil {
		panic("cannot set account balance：" + err.Error())
	}

	key := KeyOfAccountToken(addr, token)
	set(transID, txID, key, resBytes)
}

func AddAccountToken(transID, txID int64, addr, token types.Address) {
	key := KeyOfAccount(addr)
	bs := get(transID, txID, key)
	newToken := KeyOfAccountToken(addr, token)
	tokenList := make([]types.Address, 0)
	if len(bs) == 0 {
		tokenList = append(tokenList, newToken)
	} else {
		err := jsoniter.Unmarshal(bs, &tokenList)
		if err != nil {
			panic(err)
		}

		for _, item := range tokenList {
			if item == newToken {
				return
			}
		}
		tokenList = append(tokenList, newToken)
	}

	resBytes, err := jsoniter.Marshal(tokenList)
	if err != nil {
		panic(err)
	}
	set(transID, txID, key, resBytes)
}

//Rewarder declare reward information
type Rewarder struct {
	Name          string `json:"name"`          // 被奖励者名称
	RewardPercent string `json:"rewardPercent"` // 奖励比例
	Address       string `json:"address"`       // 被奖励者地址
}

func (r *Rewarder) String() string {
	var byt bytes.Buffer
	byt.WriteString("[Name:")
	byt.WriteString(r.Name)
	byt.WriteString(",RewardPercent:")
	byt.WriteString(r.RewardPercent)
	byt.WriteString(",Address:")
	byt.WriteString(r.Address)
	byt.WriteString("]")
	return byt.String()
}

//RewardStrategy struct of reward strategy
type RewardStrategy struct {
	Strategy     []Rewarder `json:"rewardStrategy,omitempty"` //奖励策略
	EffectHeight int64      `json:"effectHeight,omitempty"`   //生效高度
}

//RewardStrategy gets reward strategy of chain
func GetRewardStrategy(transID, txID int64, blockHeight int64) []Rewarder {

	value := get(transID, txID, keyOfRewardStrategy())
	if len(value) == 0 {
		return []Rewarder{}
	}
	result := make([]RewardStrategy, 0)
	err := jsoniter.Unmarshal(value, &result)
	if err != nil {
		panic(err)
	}

	for i := len(result) - 1; i >= 0; i-- {
		if result[i].EffectHeight <= blockHeight {
			return (result)[i].Strategy
		}
	}

	return []Rewarder{}
}

//AdapterGetCallBack callback of get function
func AdapterGetCallBack(transID, txID int64, key string) ([]byte, error) {
	resDB := get(transID, txID, key)

	result := new(std.GetResult)

	if resDB == nil || len(resDB) == 0 {
		result.Code = types2.ErrInvalidParameter
		result.Msg = fmt.Sprintf("key=%s cannot get data.", key)
		res, _ := jsoniter.Marshal(result)
		return res, nil
	}
	result.Code = types.CodeOK
	result.Data = resDB
	res, _ := jsoniter.Marshal(result)
	return res, nil
}

//AdapterSetCallBack callback of set function
func AdapterSetCallBack(transID, txID int64, data map[string][]byte) (*bool, error) {
	batchSet(transID, txID, data)
	b := true
	return &b, nil
}

//CheckContractAddr check contract is valid or not
func CheckContractAddr(transID, txID int64, addr string) bool {
	key := keyOfContract(addr)
	contractBytes := get(transID, txID, key)
	contract := new(std.Contract)
	err := jsoniter.Unmarshal(contractBytes, contract)
	if err != nil {
		panic(err)
	}

	appState := GetWorldAppState(transID, txID)

	if appState.BlockHeight == 0 || contract.LoseHeight == 0 {
		return true
	}

	if contract.EffectHeight > appState.BlockHeight || contract.LoseHeight < appState.BlockHeight {
		return false
	}

	return true
}

var genesisToken = new(std.Token)

//GetGenesisToken get genesis token of block chain
func GetGenesisToken() *std.Token {
	if genesisToken != nil && genesisToken.Address != "" {
		return genesisToken
	}

	key := "/genesis/token"
	tokenBytes := stateDB.Get(key)
	if err := jsoniter.Unmarshal(tokenBytes, genesisToken); err != nil {
		panic("Get Genesis Token Failed: " + err.Error())
	}

	return genesisToken
}

//Validator data struct of validator
type Validator struct {
	PubKey     types.PubKey `json:"nodepubkey,omitempty"` //节点公钥
	Power      int64        `json:"power,omitempty"`      //节点记账权重
	RewardAddr string       `json:"rewardaddr,omitempty"` //节点接收奖励的地址
	Name       string       `json:"name,omitempty"`       //节点名称
	NodeAddr   string       `json:"nodeaddr,omitempty"`   //节点地址
}

//GetAllValidators get all validators information
func GetAllValidators(transID, txID int64) []Validator {
	value := get(transID, txID, keyOfValidators())
	var nodeAddrs []string
	err := jsoniter.Unmarshal(value, &nodeAddrs)
	if err != nil {
		panic(err)
	}
	var validators = make([]Validator, len(nodeAddrs))
	for index, nodeAddr := range nodeAddrs {
		var validator Validator
		val := get(transID, txID, keyOfValidator(nodeAddr))
		err := jsoniter.Unmarshal(val, &validator)
		if err != nil {
			panic(err)
		}

		validators[index] = validator
	}
	return validators
}

//CheckBlackList check if an address is in black list
func CheckBlackList(transID, txID int64, addr types.Address) bool {
	value := get(transID, txID, keyOfBlackList(addr))
	if value == nil {
		return false
	}
	var status string
	err := jsoniter.Unmarshal(value, &status)
	if err != nil {
		panic(err)
	}

	return status == "true"
}

var genesisContractAddr types.Address

//SetGenesisContractAddr set genesis contract address to statedb
func SetGenesisContractAddr(transID, txID int64, addr types.Address) {
	v, _ := jsoniter.Marshal(addr)
	set(transID, txID, keyOfGenesisContract(), v)
	genesisContractAddr = addr
}

//GetGenesisContractAddr Get genesis contract address
func GetGenesisContractAddr(transID, txID int64) types.Address {
	if genesisContractAddr != "" {
		return genesisContractAddr
	}
	v := get(transID, txID, keyOfGenesisContract())
	if v == nil || len(v) == 0 {
		return ""
	}
	err := jsoniter.Unmarshal(v, &genesisContractAddr)
	if err != nil {
		panic(err)
	}

	return genesisContractAddr
}

func GetContractsByName(transID, txID int64, name, orgID string) []types.Address {
	key := keyOfContractOrgID(orgID, name)
	v := get(transID, txID, key)
	if v == nil || len(v) == 0 {
		return nil
	}

	cv := new(std.ContractVersionList)
	err := jsoniter.Unmarshal(v, cv)
	if err != nil {
		panic(err)
	}

	return cv.ContractAddrList
}

func GetEffectContractByName(transID, txID, height int64, name, orgID string) *std.Contract {
	key := keyOfContractOrgID(orgID, name)
	v := get(transID, txID, key)
	if v == nil || len(v) == 0 {
		return nil
	}

	cv := new(std.ContractVersionList)
	err := jsoniter.Unmarshal(v, cv)
	if err != nil {
		panic(err)
	}

	for i := len(cv.EffectHeights) - 1; i >= 0; i-- {
		if cv.EffectHeights[i] <= height {
			return GetContract(cv.ContractAddrList[i])
		}
	}

	return nil
}

func GetContractsWithHeight(transID, txID, height int64) (contractAddrs []std.ContractWithEffectHeight) {
	h := strconv.FormatInt(height, 10)
	key := keyOfContractWithHeight(h)
	v := get(transID, txID, key)
	if v == nil || len(v) == 0 {
		return
	}

	err := jsoniter.Unmarshal(v, &contractAddrs)
	if err != nil {
		panic(err)
	}
	return
}

func SetContract(transID, txID int64, contract *std.Contract) {
	key := keyOfContract(contract.Address)
	value, err := jsoniter.Marshal(contract)
	if err != nil {
		panic(err)
	}
	set(transID, txID, key, value)
}

func SetContractMeta(transID, txID int64, contract *std.ContractMeta) {
	key := keyOfContractMeta(contract.ContractAddr)
	value, err := jsoniter.Marshal(contract)
	if err != nil {
		panic(err)
	}
	set(transID, txID, key, value)
}

func SetMineContract(transID, txID int64, mines []std.MineContract) {
	key := keyOfMineContracts()
	value, err := jsoniter.Marshal(mines)
	if err != nil {
		panic(err)
	}
	set(transID, txID, key, value)
}

func SetOrganization(transID, txID int64, org *std.Organization) {
	key := keyOfOrganization(org.OrgID)
	value, err := jsoniter.Marshal(org)
	if err != nil {
		panic(err)
	}
	set(transID, txID, key, value)
}

func SetContractVersionList(transID, txID int64, orgID string, vl *std.ContractVersionList) {
	key := keyOfContractOrgID(orgID, vl.Name)
	value, err := jsoniter.Marshal(vl)
	if err != nil {
		panic(err)
	}
	set(transID, txID, key, value)
}

func GetMineContract(transID, txID int64) (mineContract []std.MineContract) {
	key := std.KeyOfMineContracts()
	v := get(transID, txID, key)
	if v == nil || len(v) == 0 {
		return
	}

	err := jsoniter.Unmarshal(v, &mineContract)
	if err != nil {
		panic(err)
	}
	return
}

func GetTokenByAddress(transID, txID int64, addr types.Address) *std.Token {
	key := KeyOfToken(addr)
	v := get(transID, txID, key)
	if v == nil || len(v) == 0 {
		return nil
	}

	token := new(std.Token)
	err := jsoniter.Unmarshal(v, &token)
	if err != nil {
		panic(err)
	}
	return token
}

func GetGasPriceRatio(transID, txID int64) string {
	key := keyOfGasPriceRatio()
	v := get(transID, txID, key)
	if len(v) == 0 {
		return ""
	}
	ratio := ""
	err := jsoniter.Unmarshal(v, &ratio)
	if err != nil {
		panic(err)
	}
	return ratio
}

func GetGenesisOrgID(transID, txID int64) string {
	key := keyOfGenesisOrgID()
	v := get(transID, txID, key)
	if len(v) == 0 {
		return ""
	}
	orgID := ""
	err := jsoniter.Unmarshal(v, &orgID)
	if err != nil {
		panic(err)
	}
	return orgID
}

func CheckBVMEnable(transID, txID int64) bool {
	key := "/bvm/status"
	v := get(transID, txID, key)
	if len(v) == 0 {
		return true
	}

	enable := false
	err := jsoniter.Unmarshal(v, &enable)
	if err != nil {
		panic(err)
	}
	return enable
}

func GetChainGenesisVersion() int {
	key := "/genesis/chainversion"
	value := stateDB.Get(key)
	if len(value) == 0 {
		return 0
	}

	var genesisChainVersion int64
	err := jsoniter.Unmarshal(value, &genesisChainVersion)
	if err != nil {
		panic(err)
	}

	if genesisChainVersion == 0 {
		return 0
	} else if genesisChainVersion == 2 {
		return 2
	}
	panic("invalid genesisChainVersion")
}

func getTrans(transID int64) *Trans {
	temp, ok := transactionMap.Load(transID)
	if !ok {
		panic("invalid transID")
	}
	trans := temp.(*Trans)
	return trans
}

func get(transID, txID int64, key string) []byte {
	temp, ok := transactionMap.Load(transID)
	if !ok {
		if transID == 0 {
			return stateDB.Get(key)
		}

		panic("invalid transID")
	}
	trans := temp.(*Trans)

	tx, ok := trans.TxMap[txID]
	if ok {
		value := tx.Get(key)
		if len(value) != 0 {
			return value
		}
	}

	value := trans.Transaction.Get(key)
	return value
}

func set(transID, txID int64, key string, value []byte) {
	temp, ok := transactionMap.Load(transID)
	if !ok {
		panic("invalid transID")
	}
	trans := temp.(*Trans)

	var tx *statedb.Tx
	tx, ok = trans.TxMap[txID]
	if !ok {
		panic(fmt.Sprintf("invalid txID: %d", txID))
	}
	tx.Set(key, value)
}

func batchSet(transID, txID int64, data map[string][]byte) {
	temp, ok := transactionMap.Load(transID)
	if !ok {
		panic("invalid transID")
	}
	trans := temp.(*Trans)

	var tx *statedb.Tx
	tx, ok = trans.TxMap[txID]
	if !ok {
		panic(fmt.Sprintf("invalid txID: %d", txID))
	}
	tx.BatchSet(data)
}

func GoBatchExec(transID int64, txs []*statedb.Tx) {
	temp, ok := transactionMap.Load(transID)
	if !ok {
		panic("invalid transID")
	}
	trans := temp.(*Trans)
	trans.Transaction.GoBatchExec(txs)
}
