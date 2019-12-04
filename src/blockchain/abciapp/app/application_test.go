package app

import (
	"blockchain/abciapp/common"
	"blockchain/abciapp/service/deliver"
	"blockchain/algorithm"
	"blockchain/smcbuilder"
	"blockchain/smcrunctl/adapter"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/jsoniter"
	"blockchain/smcsdk/sdk/rlp"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdkimpl/helper"
	"blockchain/statedb"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	btypes "blockchain/types"

	abci "github.com/tendermint/abci/types"
	tcommon "github.com/tendermint/tmlibs/common"

	"github.com/tendermint/abci/types"
	"github.com/tendermint/tmlibs/log"
)

//TestBCChainApplication_InitChain test init chain
func TestBCChainApplication_InitChain(t *testing.T) {
	initChain()
	//registerOrg()

}

func registerOrg() {
	adapterIns := adapter.GetInstance()
	methodIDBytes := algorithm.CalcMethodId("RegisterOrganization(string)string") // 0000
	aaaa := hex.EncodeToString(methodIDBytes)
	fmt.Println(aaaa)
	methodID := uint32(algorithm.BytesToInt32(methodIDBytes))
	//data, err := rlp.EncodeToBytes(string(infoByte))
	//if err != nil {
	//	panic(err.Error())
	//}

	pubKey := []byte{}
	for i := 0; i < 32; i++ {
		pubKey = append(pubKey, '0')
	}
	h := helper.BlockChainHelper{}
	orgID := h.CalcOrgID("genesis")

	rrr := statedb.Get(1, 2, "/organization/"+orgID)
	fmt.Println("查询数据库：")
	fmt.Println(string(rrr))

	conAddrs := statedb.Get(1, 2, "/contract/"+orgID+"/organization")
	cas := new(std.ContractVersionList)
	jsoniter.Unmarshal(conAddrs, cas)
	items := make([]tcommon.HexBytes, 0)
	data, _ := rlp.EncodeToBytes("hahaha")
	items = append(items, data)
	msg := btypes.Message{
		Contract: cas.ContractAddrList[0],
		MethodID: methodID,
		Items:    items,
	}

	bbb := statedb.Get(1, 2, "/contract/code/"+cas.ContractAddrList[0])
	fmt.Println(bbb)

	tx := btypes.Transaction{ // TODO nonce gasLimit Note 如何获取？
		Nonce:    1,
		GasLimit: 1000000,
		Note:     "genesis",
		Messages: make([]btypes.Message, 0),
	}
	tx.Messages = append(tx.Messages, msg)
	r := adapterIns.InvokeTx(1, 2, "bcb8yNeqAixZ7DDQx1fHSvQdA3kKDQ48gci7", tx, pubKey)
	//r := adapterIns.InvokeTx(1, 2, cas.ContractAddrList[0], tx, pubKey)

	fmt.Println("注册组织结果：")
	fmt.Println(r.Code)
	fmt.Println(r.Data)
	fmt.Println(r.GasUsed)
	fmt.Println(r.GasLimit)
}

// TestBCChainApplication_CheckTx test check tx
func TestBCChainApplication_CheckTx(t *testing.T) {
	//app := initChain()
	//var cases = []struct {
	//}{}
	//for i, c := range cases {
	//	req := createCheckTxRequest(int64(i + 1))
	//	response := app.CheckTx(req.Tx)
	//	utest.AssertEquals(response.Code, types2.CodeOK)
	//	//todo assert more ...
	//
	//}
}

//TestBCChainApplication_DeliverTx test begin block, deliver tx, end block, commit
func TestBCChainApplication_DeliverTx(t *testing.T) {
	//app := initChain()
	//var cases = []struct {
	//}{}
	//for _, c := range cases {
	//	breq := createBeginBlockRequest()
	//	dreq := createDeliverTxRequest()
	//	ereq := createEndBlockRequest()
	//
	//	bres := app.BeginBlock(*breq)
	//	utest.AssertEquals(bres.Code, types2.CodeOK)
	//	//todo assert more ...
	//	dres := app.DeliverTx(dreq.Tx)
	//	utest.AssertEquals(dres.Code, types2.CodeOK)
	//	//todo assert more ...
	//
	//	//todo create more delivertx
	//	//...
	//
	//	//End block
	//	eres := app.EndBlock(*ereq)
	//	//todo assert more ...
	//	cres := app.Commit()
	//
	//}

}

func initChain() *BCChainApplication {
	config := common.Config{
		Address:          "",
		Query_DB_Address: "",
		Abci:             "",
		Log_level:        "",
		Log_screen:       false,
		Log_file:         false,
		Log_async:        false,
		Log_size:         0,
		DB_name:          "test-db",
		DB_ip:            "127.0.0.1",
		DB_port:          "34343",
		Chain_id:         "bcb",
	}
	// TODO adapter init here?
	adapterIns := adapter.GetInstance()
	adapter.SetSdbCallback(get, set, build)
	loggerf := log.NewTMLogger("/", "abciTest")
	adapterIns.Init(loggerf, 35678)

	app := NewBCChainApplication(config, loggerf)
	req := createInitChainRequest()
	response := app.InitChain(*req)
	fmt.Println("创世结果：")
	fmt.Println(response)
	fmt.Println("GenAppState:" + string(response.GenAppState))

	adapterIns.Commit(1)
	//statedb.CommitTx(1, 2)
	//statedb.Commit(1)

	rrr := statedb.Get(1, 2, "/genesis/chainid")
	fmt.Println("查询数据库：")
	fmt.Println(string(rrr))

	rewardStrategy := statedb.Get(1, 2, "/rewardstrategys")
	fmt.Println("奖励策略：" + string(rewardStrategy))

	//utest.AssertEquals(response.Code, types2.CodeOK)
	//todo assert more ...

	return app
}

//createInitChainRequest create fake data
func createInitChainRequest() *types.RequestInitChain {
	//orgID := "orgJgaGConUyK81zibntUBjQ33PKctpk1K1G"
	//addrList := []string{}
	//org := makeOrg(orgID, addrList)
	//res, _ := jsoniter.Marshal(org)
	//statedb.Set(1, 1, "/organization/"+orgID, res)
	codeBytes, _ := ioutil.ReadFile("/Users/test/today/genesis-code/organization.tar.gz")
	//codeStr := hex.EncodeToString(codeBytes)
	contract := deliver.Contract{
		Name:     "organization",
		Version:  "2.0",
		CodeByte: codeBytes,
		CodeHash: "43EA",
		CodeDevSig: deliver.Signature{
			PubKey:    "F1EDF8F50848B8FA121A24E2A3A83CC5C8CBF85D6CE23A3A8413F46A717BEDA1",
			Signature: "signature",
		},
		CodeOrgSig: deliver.Signature{
			PubKey:    "F1EDF8F50848B8FA121A24E2A3A83CC5C8CBF85D6CE23A3A8413F46A717BEDA1",
			Signature: "signature",
		},
	}

	codeBytes, _ = ioutil.ReadFile("/Users/test/today/genesis-code/token-basic.tar.gz")
	//codeStr = hex.EncodeToString(codeBytes)
	tbContract := deliver.Contract{
		Name:     "token-basic",
		Version:  "2.0",
		CodeByte: codeBytes,
		CodeHash: "43EA",
		CodeDevSig: deliver.Signature{
			PubKey:    "F1EDF8F50848B8FA121A24E2A3A83CC5C8CBF85D6CE23A3A8413F46A717BEDA1",
			Signature: "signature",
		},
		CodeOrgSig: deliver.Signature{
			PubKey:    "F1EDF8F50848B8FA121A24E2A3A83CC5C8CBF85D6CE23A3A8413F46A717BEDA1",
			Signature: "signature",
		},
	}

	appState := deliver.InitAppState{
		Organization: "genesis",
		Token: std.Token{
			Address:          "",
			Owner:            "bcb8yNeqAixZ7DDQx1fHSvQdA3kKDQ48gci7", //bcb8yNeqAixZ7DDQx1fHSvQdA3kKDQ48gci7
			Name:             "BCB",
			Symbol:           "BCB",
			TotalSupply:      bn.N(5000000000000000000),
			AddSupplyEnabled: true,
			BurnEnabled:      true,
			GasPrice:         2500,
		},
		RewardStrategy: []deliver.Rewarder{
			{
				Name:          "validators",
				RewardPercent: "20.00",
				Address:       "bcbF4YDFW4wmzswuZ3pv1ktzi9ua3ePKatqu",
			},
			{
				Name:          "r_d_team",
				RewardPercent: "30.00",
				Address:       "bcbF4YDFW4wmzswuZ3pv1ktzi9ua3ePKatqu",
			},
			{
				Name:          "bonus_bcb",
				RewardPercent: "20.00",
				Address:       "bcbF4YDFW4wmzswuZ3pv1ktzi9ua3ePKatqu",
			},
			{
				Name:          "bonus_token",
				RewardPercent: "30.00",
				Address:       "bcbF4YDFW4wmzswuZ3pv1ktzi9ua3ePKatqu",
			},
		},
		Contracts: []deliver.Contract{contract, tbContract},
	}
	appStateBytes, _ := json.Marshal(appState)

	//r, _ := hex.DecodeString("F1EDF8F50848B8FA121A24E2A3A83CC5C8CBF85D6CE23A3A8413F46A717BEDA1")
	r := []byte{}
	for i := 0; i < 32; i++ {
		r = append(r, '0')
	}
	req := types.RequestInitChain{
		Validators: []abci.Validator{
			{
				PubKey:     r,
				Power:      10,
				RewardAddr: "bcb5xjCpYJrKkyM3kgo5Jk8Pge7XDTwuvSUm",
				Name:       "earth",
			},
			{
				PubKey:     r,
				Power:      10,
				RewardAddr: "bcb5rzgE1tSJbJuegEj4vbAkotmwRkxwiSyV",
				Name:       "venus",
			},
			{
				PubKey:     r,
				Power:      10,
				RewardAddr: "bcb3JivtUWiDG48dbWDvFBUHDL8veB7JvQvS",
				Name:       "jupiter",
			},
			{
				PubKey:     r,
				Power:      10,
				RewardAddr: "bcbKG7Y7hWLhjxiBNZ8UBgja4ocLcSHKW4b4",
				Name:       "mercury",
			},
			{
				PubKey:     r,
				Power:      10,
				RewardAddr: "bcbAb65XuCz8bWHB1ittqMQSRyLvDmb74KRB",
				Name:       "mars",
			},
			{
				PubKey:     r,
				Power:      10,
				RewardAddr: "bcbCfciTQNLQGwRWadxXg6ms8dBguP5sooX8",
				Name:       "pluto",
			},
		},
		ChainId:       "bcb",
		AppStateBytes: appStateBytes,
	}

	return &req
}

func makeOrg(orgID string, contractAddrs []string) std.Organization {

	return std.Organization{
		OrgID:            orgID,
		Name:             "test-org",
		OrgOwner:         "test-rog-owner",
		ContractAddrList: contractAddrs,
		OrgCodeHash:      algorithm.CalcCodeHash("helloA"),
		Signers:          nil,
	}
}

//createBeginBlockRequest create fake data
func createBeginBlockRequest(blockHeight int64) *types.RequestBeginBlock {
	req := types.RequestBeginBlock{
		Hash:                nil,
		Header:              types.Header{},
		AbsentValidators:    nil,
		ByzantineValidators: nil,
	}
	return &req
}

//createCheckTxRequest create fake data
func createCheckTxRequest(blockHeight int64, param ...interface{}) *types.RequestCheckTx {
	req := types.RequestCheckTx{
		Tx: nil,
	}
	return &req
}

//createDeliverTxRequest create fake data
func createDeliverTxRequest(blockHeight int64) *types.RequestDeliverTx {
	req := types.RequestDeliverTx{
		Tx: nil,
	}
	return &req
}

//createBeginBlockRequest create fake data
func createEndBlockRequest(blockHeight int64) *types.RequestEndBlock {
	req := types.RequestEndBlock{
		Height: blockHeight,
	}
	return &req
}

//createCommitRequest create fake data
func createCommitRequest() *types.RequestCommit {
	return &types.RequestCommit{}
}

////GetCallback callback of get()
//type GetCallback func(int64, int64, string) (*[]byte, error)
//
////SetCallback callback of set()
//type SetCallback func(int64, int64, map[string][]byte) (*bool, error)
//
////BuildCallback callback of build()
//type BuildCallback func(int64, int64, std.ContractMeta) (*std.BuildResult, error)
func get(transID, txID int64, key string) (*[]byte, error) {
	resDB := statedb.Get(transID, txID, key)

	result := new(std.GetResult)

	if resDB == nil || len(resDB) == 0 {
		result.Code = 5001
		result.Msg = "can not get data."
		res, _ := jsoniter.Marshal(result)
		return &res, nil
	}
	result.Code = 200
	result.Data = resDB
	res, _ := jsoniter.Marshal(result)
	return &res, nil
	//resDB := statedb.Get(transID, txID, key)
	//return &resDB, nil
}

func set(transID, txID int64, data map[string][]byte) (*bool, error) {
	statedb.BatchSet(transID, txID, data)
	b := true
	return &b, nil
}

func build(transID, txID int64, contractMeta std.ContractMeta) (result *std.BuildResult, err error) {
	b := smcbuilder.GetInstance()
	result1 := b.BuildContract(transID, txID, contractMeta)
	result = &result1

	fmt.Println(result)
	return
}
