package stubs

import (
	"blockchain/abciapp_v1.0/contract/stubapi"
	"blockchain/abciapp_v1.0/prototype"
	"blockchain/abciapp_v1.0/smc"
	"blockchain/abciapp_v1.0/statedb"
	"blockchain/abciapp_v1.0/tx/tx"
	"blockchain/smcsdk/sdk/rlp"
	"bytes"
	"common/bcdb"
	"common/utils"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"github.com/tendermint/tmlibs/log"
	"strconv"
	"testing"
)

func TestSystemStub_Dispatcher(t *testing.T) {
	it, err := newSystemContext(prototype.System)
	if err != nil {
		return
	}
	defer db.Close()

	var items stubapi.InvokeParams

	items.Params = packItems_DeployInternalContract()
	items.Ctx = it
	syslog := log.NewTMLogger("./log", "contract")
	sysStub := NewSystemStub(syslog)
	res, bcerr := sysStub.Dispatcher(&items)
	fmt.Println(res)
	fmt.Println(bcerr)
}
func uint64ToByteSlice(i uint64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, i)
	return buf
}

func packItems_DeployInternalContract() []byte {

	var items = make([]string, 0)
	items = append(items, "name")
	items = append(items, "1.0")
	items = append(items, "transfer(smc.Addr,big.Int)smc.Error;setgasprice(uint64)smc.Error")

	var gasList = make([][]byte, 2)
	gas1 := uint64ToByteSlice(3000)
	gas2 := uint64ToByteSlice(5462)
	gasList = append(gasList, gas1)
	gasList = append(gasList, gas2)
	gas := bytes.Join(gasList, []byte(""))
	items = append(items, string(gas))
	items = append(items, "AB4572847763237243387422744898092222555") //CodeHash
	items = append(items, string(uint64ToByteSlice(2000)))           //EffectHeight
	methodId := strconv.FormatUint(uint64(stubapi.ConvertPrototype2ID(prototype.SysDeployInternalContract)), 16)
	param, _ := packItems("0x"+methodId, items)

	fmt.Println("MethodID", methodId)
	return param
}
func packItems(methodId string, items []string) ([]byte, error) {

	var mi tx.MethodInfo

	//parse methodId
	_, err := utils.ParseHexUint32(methodId, "methodId")
	if err != nil {
		return nil, err
	}
	dataBytes, _ := hex.DecodeString(string([]byte(methodId[2:])))
	mi.MethodID = binary.BigEndian.Uint32(dataBytes)

	// Parse parameters
	var itemsBytes = make([]([]byte), 0)
	for _, item := range items {
		itemBytes := []byte(item)
		itemsBytes = append(itemsBytes, itemBytes)
	}
	mi.ParamData, err = rlp.EncodeToBytes(itemsBytes)
	if err != nil {
		return nil, err
	}

	data, err := rlp.EncodeToBytes(mi)
	if err != nil {
		return nil, err
	}

	return data, nil
}

var db *bcdb.GILevelDB

func newSystemContext(contractName string) (*stubapi.InvokeContext, error) {

	// open db
	var err error
	db, err = bcdb.OpenDB("D:\\Work\\Code-SVN\\GIBlockChain\\trunk\\code\\v1.0\\bcchain\\bin\\BCState", "127.0.0.1", "8889")
	if err != nil {
		return nil, err
	}

	stateDB := statedb.NewStateDB(db)
	stateDB.BeginBlock()

	appState, _ := stateDB.GetWorldAppState()
	// 获取系统合约地址，填入txState
	var syscontractAddr, ownerAddr smc.Address
	genContracts, _ := stateDB.GetContractAddrList()
	for _, addr := range genContracts {
		contract, err := stateDB.GetContract(addr)
		if err != nil || contract == nil {
			panic(errors.New("Failed to get contract"))
		}
		if contract.Name == contractName && appState.BlockHeight >= int64(contract.EffectHeight) &&
			(appState.BlockHeight < int64(contract.LoseHeight) || contract.LoseHeight == 0) {
			syscontractAddr = contract.Address
			ownerAddr = contract.Owner
			break
		}
	}
	var txState *statedb.TxState
	//Generate txState to operate stateDB
	// 系统合约的地址和拥有者
	txState = stateDB.NewTxState(syscontractAddr, ownerAddr)
	txState.BeginTx()

	// Generate accounts and execute
	sender := &stubapi.Account{
		ownerAddr,
		txState,
	}
	// Get token basic and its contract
	tb, err := stateDB.GetGenesisToken()
	if err != nil || tb == nil {
		return nil, err
	}
	// Using tokenbasic owner as it doesn't be changed
	owner := &stubapi.Account{
		ownerAddr,
		txState,
	}
	invokeContext := stubapi.InvokeContext{
		sender,
		owner,
		txState,
		nil,
		appState.BeginBlock.Header,
		nil,
		nil,
		300000,
	}

	return &invokeContext, nil
}
