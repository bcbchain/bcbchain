package tokenbyb

import (
	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/keys"
	"blockchain/abciapp_v1.0/smc"
	"blockchain/abciapp_v1.0/statedb"
	"blockchain/algorithm"
	"code/contract/smcapi"
	"code/contract/stubapi"
	"code/prototype"
	"common/bcdb"
	"errors"
	"fmt"
	"github.com/tendermint/go-crypto"
	"math/big"
	"testing"
)

var dbtest *bcdb.GILevelDB
var chainId string

func TestInit(t *testing.T) {
	byb, err := newContext()
	if err != nil {
		panic(err)
	}
	defer dbtest.Close()
	bybToken := TokenByb{byb}

	addr := NewAccount("heheda").Address

	sender := *bybToken.Sender
	sender.Addr = addr
	bybFalse := smcapi.SmcApi{
		&sender,
		bybToken.Owner,
		bybToken.ContractAcct,
		bybToken.ContractAddr,
		bybToken.State,
		bybToken.Block,
		bybToken.EventHandler,
	}
	bybTokenFalse := TokenByb{
		&bybFalse,
	}

	var tests = []struct {
		bybToken         TokenByb
		totalSupply      big.Int
		addSupplyEnabled bool
		burnEnabled      bool
		want             bool
	}{ //非合约所有者调用init接口
		{bybTokenFalse, *big.NewInt(1E9), true, true, false},

		//下面全是合约所有者调用
		//验证参数-参数-supply
		{bybToken, *big.NewInt(0), true, true, false},
		{bybToken, *big.NewInt(2000000000000000000), true, true, true},
		{bybToken, *big.NewInt(100), true, true, false},
		{bybToken, *big.NewInt(1E9), true, true, true},

		//验证参数-addSupplyEnabled-burnEnabled
		{bybToken, *big.NewInt(1E9), false, false, true},
		{bybToken, *big.NewInt(1E9), true, false, true},
		{bybToken, *big.NewInt(1E9), false, true, true},
		{bybToken, *big.NewInt(1E9), true, true, true}, //不回滚了

		{bybToken, *big.NewInt(1E9), true, true, false}, //已经进行过初始化，再次初始化
	}

	for k, test := range tests {
		byb.State.StateDB.BeginBlock()
		byb.State.BeginTx()
		smcErr := test.bybToken.Init(test.totalSupply, test.addSupplyEnabled, test.burnEnabled)
		if smcErr.ErrorCode != bcerrors.ErrCodeOK {
			if test.want {
				t.Errorf("init_failed :want=%v-----init(%v,%v,%v) = %v ", test.want, test.totalSupply, test.addSupplyEnabled, test.burnEnabled, smcErr)
			} else {
				t.Logf("init_succed :want=%v", test.want)
			}

			byb.State.RollbackTx()
		} else { //倒数第二个设置一个正常的初始化，就不回滚了，供后续使用，最后一个是错误的，也得回滚
			if k == len(tests)-2 {
				byb.State.CommitTx()
				byb.State.StateDB.CommitBlock()
			}
			t.Logf("init_succed :want=%v", test.want)
			byb.State.RollbackTx()
		}

	}
}

func TestNewStockHolder(t *testing.T) {
	byb, err := newContext()
	if err != nil {
		panic(err)
	}
	defer dbtest.Close()
	addrOne := NewAccount("heheda").Address
	bybToken := TokenByb{byb}

	addrTwo := NewAccount("heheda").Address

	sender := *bybToken.Sender
	sender.Addr = addrTwo
	bybFalse := smcapi.SmcApi{
		&sender,
		bybToken.Owner,
		nil,
		bybToken.ContractAddr,
		bybToken.State,
		bybToken.Block,
		bybToken.EventHandler,
	}
	bybTokenFalse := TokenByb{
		&bybFalse,
	}

	var tests = []struct {
		bybToken    TokenByb
		stockHolder smc.Address
		value       big.Int
		want        bool
	}{
		//调用者不等于owner
		{bybTokenFalse, NewAccount("heheda").Address, *big.NewInt(10), false},
		//调用者都是owner
		{bybToken, addrOne, *big.NewInt(10), true},
		{bybToken, addrOne, *big.NewInt(10), true},
		{bybToken, addrOne, *big.NewInt(10), true},
		{bybToken, addrOne, *big.NewInt(1E10), false}, //金额大于owner的金额
	}

	for _, test := range tests {
		byb.State.StateDB.BeginBlock()
		byb.State.BeginTx()
		_, smcErr := test.bybToken.NewStockHolder(test.stockHolder, test.value)
		if smcErr.ErrorCode != bcerrors.ErrCodeOK {
			if test.want {
				t.Errorf("NewStockHolder_failed :want=%v-----NewStockHolder(%v,%v) = %v ", test.want, test.stockHolder, test.value, smcErr)
			} else {
				t.Logf("NewStockHolder_succed :want=%v", test.want)
			}
			byb.State.RollbackTx()
		} else {
			t.Logf("NewStockHolder_succed :want=%v", test.want)
			byb.State.CommitTx()
			byb.State.StateDB.CommitBlock()
		}
	}

}

/*
*测试转账
有BYB币的股东账户
{"localLWGFThWBBtQM6VQdDhu3xPdJ1QqkhGad1"},
{"localA1ydhRuyU1New79vvDYCJT9qCRKYQqZiL"},
{"localPjv43NsigvwWtw4qnnZmocBEuhpQfrsUZ"},
{"localPRAbnXe2dqAM5mBr3PLEGxmzvVi1SaXks"},
{"localCkuTi1B3uRnq59NKKX8j6cRoA42wv9XS3"},
{"local86ob92x7Y6ReBjynwTGahGd9NHLYemFWK"},
{"local7RufMNXKMPqKLZc63VA6PviJERZ4EJVc8"},
{"localGXoDZ5D6XfCFJdo9DbCtQZVfjtAUkcU8y"},
{"localLmXsQ3uciTdjK3ehJaruu4EHA1MBWJQgX"},
{"local3hec5RuhS7bbmghTaL7FmjTqngA4XbfWY"},
{"localBk7Spr6AD4rXuYCDiNWLepW8K3oZfxnWa"},
{"localHtxGVmqCAYDCg2h76Gff3ksBC6FTW6z8T"},
{"localGAPvAGsUWqWEkM1zJrAsuWUPsb1eC4oWW"},
{"local5aZpxJtVqRNeqDJkC1XhdHPrKcwdSb3Fm"},
{"localEp5inYWYjuGErKyBzXUYnPVfkzYU7Bh4K"},
{"localDamZRBcK9rRgQDs42JhMKPQsLRWtRK2Vp"},
{"localJg7t7TWrevk1yf2t38KmtJ3GysgpD5d42"},
{"localDvX3E87r2QszGJNYrq4yv8JctX4XMV12d"},
{"localFoDQGmMttuZEwm16vzYC2xQxxB21nh5Dn"},
{"localJWwaDUzt7qmvA1t4Ei5DCjpguGq2ZdMb6"},
{"localAjjtz6WYNb2EUe9THPCK5ybGbxPqtcTtH"},
{"local9XjmLnDDzf1JQEySXxKi76A5VaRtmJwzb"},
{"localNoKpgFJLP6saDpgbMQxcBBn8GnFAqGaGk"},
{"localGtaePqkUaBsZ9Aex1y76oUhCLuRFHfyp1"},
{"localDnBsw6QMjwhSMNmiiDXLR35fCwvLPQKYx"},
{"local2Zn63kGVFBksQW93gLirr7n7YJmprrh2j"},
{"local21krTuDARMV85VVrTmN8JvYewhw7MDerk"},
{"localFz5KMw84cfDrfSz4DEiKWr7MqKiaLA9x8"},
{"localBpVcTXJj2WY4hebf4ob1wZQNup9fiq3xs"},
{"local8cRU43MJw1D3McWEizKvvMR7CdnmRE7e8"},
{"local7fQH9TVPtuux6YGed8ADzeVYm4N9ePzQN"},
//黑洞地址
//委员会地址

*/
func TestTransfer(t *testing.T) {
	var sends = []struct {
		sendarr    smc.Address
		statusCode int
	}{
		//以下情况，请选一个测，测一个时请把其他的注释掉
		//模拟发送人地址不能是owner地址
		//{"local7fQH9TVPtuux6YGed8ADzeVYm4N9ePzQN",1,},
		//模拟发送人地址黑洞地址
		//{"",1,},
		//模拟发送地址是股东地址
		//	{"local7fQH9TVPtuux6YGed8ADzeVYm4N9ePzQN",1,},
		//模拟发送地址与转账地址相同
		//	{"local3jdbKpAmJiJ4tqB3rnyB3hfe4zTPFe5Rd",1,},
		//模拟发送地址是普通用户地址
		{"1111111", 0},
		//模拟发送账户金额跟转账金额一致
		//	{"222222",0,},
		//模拟发送账户金额小于转账金额
		//	{"333333",1,},
		//模拟发送账户金额为0
		//	{"666666",1,},
		//模拟发送账户金额染色体只有一个
		//	{"local9XjmLnDDzf1JQEySXxKi76A5VaRtmJwzb",0,},
		//模拟发送账户染色体有很多
		//	{"1111111",0,},
		//模拟染色体 金额平均
		//	{"888888",0,},
	}
	for _, x := range sends {
		//连接状态数据库
		ic, err := newContext1(x.sendarr)
		if err != nil || ic == nil {
			t.Fatal("Init invokeContext failed: " + err.Error())
		}
		api := &smcapi.SmcApi{ic.Sender,
			ic.Owner,
			nil,
			ic.ContractAddr,
			ic.State,
			ic.Block,
			ic.EventHandler,
		}
		byb := TokenByb{api}

		var trans = []struct {
			to         smc.Address
			value      big.Int
			statusCode int
		}{
			//模拟正常转账场景
			{"1111111", *big.NewInt(200), 0},
			//模拟接收地址是股东地址
			{"local3jdbKpAmJiJ4tqB3rnyB3hfe4zTPFe5Rd", *big.NewInt(200), 1},
			//模拟接收地址是黑洞地址
			//{"local3jdbKpAmJiJ4tqB3rnyB3hfe4zTPFe5Rd", *big.NewInt(200),  0,},
			//模拟接收地址是owner地址
			{"localMHvsTofey2d3jP1Ngf1G1GEY4b9EW8ame", *big.NewInt(200), 1},

			//模拟转账金额为0
			{"222222", *big.NewInt(0), 1},
			//模拟转账金额为1
			{"222222", *big.NewInt(1), 0},
			//模拟转账金额为2
			{"222222", *big.NewInt(2), 0},
			//模拟转账金额=账户金额
			{"222222", *big.NewInt(10000000), 0},
		}
		var bcerr bcerrors.BCError

		defer dbtest.Close()

		for _, con := range trans {
			ic.State.StateDB.BeginBlock()
			ic.State.BeginTx()
			bcerr = byb.Transfer(con.to, con.value)
			if con.statusCode == 0 && bcerr.ErrorCode != bcerrors.ErrCodeOK {
				t.Errorf("test failed: " + bcerr.Error())
				ic.State.RollbackTx()
			} else if con.statusCode == 1 && bcerr.ErrorCode != bcerrors.ErrCodeOK {
				t.Logf("test failed: " + bcerr.Error())
				ic.State.RollbackTx()
			} else if con.statusCode == 1 && bcerr.ErrorCode == bcerrors.ErrCodeOK {
				t.Errorf("test failed, the expected error didn't happen")
				ic.State.RollbackTx()
			} else {
				t.Logf("test OKay,transfer success ")
				ic.State.CommitTx()
				byb.State.StateDB.CommitBlock()
			}
			//ic.State.BeginTx()
		}
	}
}

func NewAccount(name string) *keys.Account {
	crypto.SetChainId(chainId)
	privKey := crypto.GenPrivKeyEd25519()
	pubKey := privKey.PubKey()
	address := pubKey.Address()

	acct := keys.Account{
		Name:         name,
		PrivKey:      privKey,
		PubKey:       pubKey,
		Address:      address,
		Nonce:        0,
		KeystorePath: "",
	}
	return &acct
}

func newContext() (*smcapi.SmcApi, error) {
	// open db
	var err error
	dbtest, err = bcdb.OpenDB("D:\\Work\\Code-SVN\\BCBlockChain\\trunk\\code\\v1.0\\bcchain\\bin\\.appState", "127.0.0.1", "8888")
	if err != nil {
		return nil, err
	}
	stateDB := statedb.NewStateDB(dbtest)
	//stateDB.BeginBlock()

	chainId = stateDB.GetChainID()

	// 获取系统合约地址，填入txState
	var contractAddr, ownerAddr smc.Address
	genContracts, _ := stateDB.GetContractAddrList()
	for _, addr := range genContracts {
		contract, err := stateDB.GetContract(addr)
		if err != nil || contract == nil {
			panic(errors.New("Failed to get contract"))
		}
		if contract.Name == prototype.TokenBYB {
			contractAddr = contract.Address
			ownerAddr = contract.Owner
		}
	}
	var txState *statedb.TxState
	//Generate txState to operate stateDB
	// 系统合约的地址和拥有者
	txState = stateDB.NewTxState(contractAddr, ownerAddr)
	//txState.BeginTx()

	// Generate accounts and execute
	sender := &stubapi.Account{
		ownerAddr,
		txState,
	}
	// Get token basic and its contract
	//tb, err := stateDB.GetGenesisToken()
	//if err != nil || tb == nil {
	//	return nil, err
	//}
	// Using tokenbasic owner as it doesn't be changed
	owner := &stubapi.Account{
		ownerAddr,
		txState,
	}

	invokeContext := smcapi.SmcApi{
		sender,
		owner,
		nil,
		&contractAddr,
		txState,
		nil,
		nil,
	}
	sct := stubapi.InvokeContext{Sender: invokeContext.Sender, Owner: invokeContext.Owner, TxState: invokeContext.State}

	invokeContext.ContractAcct = calcContractAcct(&sct, prototype.TokenBYB)
	return &invokeContext, nil
}

func calcContractAcct(ctx *stubapi.InvokeContext, contractName string) *stubapi.Account {

	addr := algorithm.CalcContractAddress(
		ctx.TxState.GetChainID(),
		"",
		contractName,
		"")
	//TODO: using genesis token for now
	genToken, _ := ctx.TxState.GetGenesisToken()
	if genToken == nil {
		return nil
	}

	return &stubapi.Account{addr,
		&statedb.TxState{ctx.TxState.StateDB,
			genToken.Address,
			ctx.TxState.SenderAddress,
			ctx.TxState.TxBuffer}}
}

//生成很多股东地址
func TestNewStockHolder1(t *testing.T) {
	byb, err := newContext()
	if err != nil {
		panic(err)
	}
	defer dbtest.Close()
	bybToken := TokenByb{byb}

	var tests = []struct {
		bybToken    TokenByb
		stockHolder smc.Address
		value       big.Int
		want        bool
	}{
		{bybToken, NewAccount("1").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("2").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("3").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("4").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("5").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("6").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("7").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("8").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("9").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("10").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("11").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("12").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("13").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("14").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("15").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("16").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("17").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("18").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("19").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("20").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("21").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("22").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("23").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("24").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("25").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("26").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("27").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("28").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("29").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("30").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("31").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("32").Address, *big.NewInt(10000000), true},
		{bybToken, NewAccount("33").Address, *big.NewInt(10000000), true},
	}

	for _, test := range tests {
		byb.State.StateDB.BeginBlock()
		byb.State.BeginTx()
		_, smcErr := test.bybToken.NewStockHolder(test.stockHolder, test.value)
		if smcErr.ErrorCode != bcerrors.ErrCodeOK {
			if test.want {
				t.Errorf("NewStockHolder_failed :want=%v-----NewStockHolder(%v,%v) = %v ", test.want, test.stockHolder, test.value, smcErr)
			} else {
				t.Logf("NewStockHolder_succed :want=%v", test.want)
			}
			byb.State.RollbackTx()
		} else {

			t.Logf("NewStockHolder_succed :want=%v", test.want, test.stockHolder)
			byb.State.CommitTx()
			byb.State.StateDB.CommitBlock()

		}
	}
}

func TestTokenByb_GetStackHolders(t *testing.T) {
	ctx, err := newContext()
	if err != nil {
		panic(err)
	}

	defer dbtest.Close()

	byb := TokenByb{&smcapi.SmcApi{ctx.Sender, ctx.Owner, ctx.ContractAcct, ctx.ContractAddr, ctx.State, ctx.Block, ctx.EventHandler}}
	holders, _ := byb.getBybStockHolders()
	fmt.Println(holders)

	for _, holder := range holders {
		fmt.Println(byb.getBybBalance(holder))
	}
}

func TestTokenByb_TrasnferByChromo(t *testing.T) {
	var testCases = []struct {
		sender smc.Address
		to     smc.Address
		value  big.Int
		bOK    bool
	}{
		// 股东转出不存在的byb
		{"localQ3w1qcU9dmM5m71LAogJ1h3KkWWfyQN6s", "localFs34yu31jwv5eiKGktHczFQP58FtS5L5F", *big.NewInt(100), false},
		//接收转出存在的byb 给普通用户
		//	{"local8mUovyiiM1pxGh3cCvvixXFetjd1cTowC","localFs34yu31jwv5eiKGktHczFQP58FtS5L5F", *big.NewInt(100000), true},
		//普通用户转给其他股东
		{"localFs34yu31jwv5eiKGktHczFQP58FtS5L5F", "localQ3w1qcU9dmM5m71LAogJ1h3KkWWfyQN6s", *big.NewInt(100), false},
		//普通用户转给其他普通用户
		{"localFs34yu31jwv5eiKGktHczFQP58FtS5L5F", "localQ33333U9dmM5m71LAogJ1h3KkWWfyQN6s", *big.NewInt(100000), true},
	}

	ctx, err := newContext()
	if err != nil {
		panic(err)
	}

	defer dbtest.Close()

	chromo := "6"

	for _, test := range testCases {
		ctx.State.StateDB.BeginBlock()
		ctx.State.BeginTx()

		ctx.Sender.Addr = test.sender

		byb := TokenByb{ctx}
		smcErr := byb.TransferByChromo(chromo, test.to, test.value)
		if smcErr.ErrorCode != bcerrors.ErrCodeOK {
			if test.bOK {
				t.Errorf("test failed :want=%v-----, got %v ", test.bOK, smcErr)
			} else {
				t.Logf("test passed :want=%v", test.bOK)
			}

			byb.State.RollbackTx()
		} else {
			if test.bOK {
				t.Errorf("test passed :want=%v-----, got %v ", test.bOK, smcErr)
				byb.State.CommitTx()
				byb.State.StateDB.CommitBlock()

			} else {
				t.Logf("test failed :want=%v", test.bOK)
				byb.State.RollbackTx()
			}
		}
	}
}

func TestTokenByb_ChangeChromoOwnership(t *testing.T) {
	var testCases = []struct {
		sender string
		//		chromo smc.Hash
		newOwner string
		bOK      bool
	}{
		// 接收地址为非股东
		{"localFs34yu31jwv5eiKGktHczFQP58FtS5L5F", "localETK7Zh9hNSPrEKdmCgnHDtFPatcs9WwVL", false},
		//接收地址为股东
		{"localFs34yu31jwv5eiKGktHczFQP58FtS5L5F", "local8mUovyiiM1pxGh3cCvvixXFetjd1cTowC", true},
		//已转移，重复转移操作
		{"localFs34yu31jwv5eiKGktHczFQP58FtS5L5F", "localLXiDHMoCq87UT2zwKDpKCoJBEhyv5xNu9", false},
	}

	ctx, err := newContext()
	if err != nil {
		panic(err)
	}

	defer dbtest.Close()

	chromo := "6"

	for _, test := range testCases {
		ctx.State.StateDB.BeginBlock()
		ctx.State.BeginTx()

		ctx.Sender.Addr = test.sender

		byb := TokenByb{ctx}
		smcErr := byb.ChangeChromoOwnership(chromo, test.newOwner)
		if smcErr.ErrorCode != bcerrors.ErrCodeOK {
			if test.bOK {
				t.Errorf("test failed :want=%v-----, got %v ", test.bOK, smcErr)
			} else {
				t.Logf("test passed :want=%v", test.bOK)
			}

			byb.State.RollbackTx()
		} else {
			if test.bOK {
				t.Errorf("test passed :want=%v-----, got %v ", test.bOK, smcErr)
				byb.State.CommitTx()
				byb.State.StateDB.CommitBlock()

			} else {
				t.Logf("test failed :want=%v", test.bOK)
				byb.State.RollbackTx()
			}
		}
	}
}

func TestTokenByb_DelStockHolder(t *testing.T) {
	var testCases = []struct {
		sender string
		bOK    bool
	}{
		// 非股东
		{"localETK7Zh9hNSPrEKdmCgnHDtFPatcs9WwVL", false},
		//股东, 没有byb
		{"localFs34yu31jwv5eiKGktHczFQP58FtS5L5F", true},
		//股东，拥有byb
		{"local8mUovyiiM1pxGh3cCvvixXFetjd1cTowC", false},
	}

	ctx, err := newContext()
	if err != nil {
		panic(err)
	}

	defer dbtest.Close()

	for _, test := range testCases {
		ctx.State.StateDB.BeginBlock()
		ctx.State.BeginTx()

		byb := TokenByb{ctx}
		smcErr := byb.DelStockHolder(test.sender)
		if smcErr.ErrorCode != bcerrors.ErrCodeOK {
			if test.bOK {
				t.Errorf("test failed :want=%v-----, got %v ", test.bOK, smcErr)
			} else {
				t.Logf("test passed :want=%v", test.bOK)
			}

			byb.State.RollbackTx()
		} else {
			if test.bOK {
				t.Errorf("test passed :want=%v-----, got %v ", test.bOK, smcErr)
				byb.State.CommitTx()
				byb.State.StateDB.CommitBlock()

			} else {
				t.Logf("test failed :want=%v", test.bOK)
				byb.State.RollbackTx()
			}
		}
	}
}

func newContext1(senderAddr smc.Address) (*smcapi.SmcApi, error) {
	// open db
	var err error
	dbtest, err = bcdb.OpenDB("D:\\BCState", "127.0.0.1", "8889")
	if err != nil {
		return nil, err
	}
	stateDB := statedb.NewStateDB(dbtest)
	chainId = stateDB.GetChainID()

	// 获取系统合约地址，填入txState
	var contractAddr, ownerAddr smc.Address
	genContracts, _ := stateDB.GetContractAddrList()
	for _, addr := range genContracts {
		contract, err := stateDB.GetContract(addr)
		if err != nil || contract == nil {
			panic(errors.New("Failed to get contract"))
		}
		if contract.Name == prototype.TokenBYB {
			contractAddr = contract.Address
			ownerAddr = contract.Owner
		}
	}
	var txState *statedb.TxState
	//Generate txState to operate stateDB
	// 系统合约的地址和拥有者
	txState = stateDB.NewTxState(contractAddr, ownerAddr)

	sender := &stubapi.Account{
		senderAddr,
		txState,
	}

	owner := &stubapi.Account{
		ownerAddr,
		txState,
	}

	invokeContext := smcapi.SmcApi{
		sender,
		owner,
		nil,
		&contractAddr,
		txState,
		nil,
		nil,
	}

	return &invokeContext, nil
}

func TestTokenByb_SetOwner(t *testing.T) {
	var testCases = []struct {
		sender   string
		newOwner string
		bOK      bool
	}{
		{"localETK7Zh9hNSPrEKdmCgnHDtFPatcs9WwVL", "localETK7Zh9hNSPrEKdmCgnHDtFPatcs9WwVL", false},
		{"localPRAbnXe2dqAM5mBr3PLEGxmzvVi1SaXks", "localETK7Zh9hNSPrEKdmCgnHDtFPatcs9WwVL", false},
		{"localETK7Zh9hNSPrEKdmCgnHDtFPatcs9WwVL", "localPRAbnXe2dqAM5mBr3PLEGxmzvVi1SaXks", true},
	}

	ctx, err := newContext()
	if err != nil {
		panic(err)
	}

	defer dbtest.Close()
	for _, test := range testCases {
		ctx.State.StateDB.BeginBlock()
		ctx.State.BeginTx()

		ctx.Sender.Addr = test.sender

		byb := TokenByb{ctx}
		smcErr := byb.SetOwner(test.newOwner)
		if smcErr.ErrorCode != bcerrors.ErrCodeOK {
			if test.bOK {
				t.Errorf("test failed :want=%v-----, got %v ", test.bOK, smcErr)
			} else {
				t.Logf("test passed :want=%v", test.bOK)
			}

			byb.State.RollbackTx()
		} else {
			if test.bOK {
				t.Errorf("test passed :want=%v-----, got %v ", test.bOK, smcErr)
				byb.State.CommitTx()
				byb.State.StateDB.CommitBlock()

			} else {
				t.Logf("test failed :want=%v", test.bOK)
				byb.State.RollbackTx()
			}
		}
	}
}

func TestTokenByb_AddSupply(t *testing.T) {
	var testCases = []struct {
		sender string
		value  big.Int
		bOK    bool
	}{
		{"localETK7Zh9hNSPrEKdmCgnHDtFPatcs9WwVL", *big.NewInt(-1), false},
		{"localPRAbnXe2dqAM5mBr3PLEGxmzvVi1SaXks", *big.NewInt(1), false},
		{"localETK7Zh9hNSPrEKdmCgnHDtFPatcs9WwVL", *big.NewInt(100000000000), true},
	}

	ctx, err := newContext()
	if err != nil {
		panic(err)
	}

	defer dbtest.Close()
	for _, test := range testCases {
		ctx.State.StateDB.BeginBlock()
		ctx.State.BeginTx()

		ctx.Sender.Addr = test.sender

		byb := TokenByb{ctx}
		smcErr := byb.AddSupply(test.value)
		if smcErr.ErrorCode != bcerrors.ErrCodeOK {
			if test.bOK {
				t.Errorf("test failed :want=%v-----, got %v ", test.bOK, smcErr)
			} else {
				t.Logf("test passed :want=%v", test.bOK)
			}

			byb.State.RollbackTx()
		} else {
			if test.bOK {
				t.Errorf("test passed :want=%v-----, got %v ", test.bOK, smcErr)
				byb.State.CommitTx()
				byb.State.StateDB.CommitBlock()

			} else {
				t.Logf("test failed :want=%v", test.bOK)
				byb.State.RollbackTx()
			}
		}
	}
}

func TestTokenByb_Burn(t *testing.T) {
	var testCases = []struct {
		sender string
		value  big.Int
		bOK    bool
	}{
		{"localETK7Zh9hNSPrEKdmCgnHDtFPatcs9WwVL", *big.NewInt(-1), false},
		{"localPRAbnXe2dqAM5mBr3PLEGxmzvVi1SaXks", *big.NewInt(1), true},
		{"localPRAbnXe2dqAM5mBr3PLEGxmzvVi1SaXks", *big.NewInt(10000), true},
		{"localPRAbnXe2dqAM5mBr3PLEGxmzvVi1SaXks", *big.NewInt(10000000000000000), false},
	}

	ctx, err := newContext()
	if err != nil {
		panic(err)
	}

	defer dbtest.Close()
	for _, test := range testCases {
		ctx.State.StateDB.BeginBlock()
		ctx.State.BeginTx()

		ctx.Sender.Addr = test.sender

		byb := TokenByb{ctx}
		smcErr := byb.Burn(test.value)
		if smcErr.ErrorCode != bcerrors.ErrCodeOK {
			if test.bOK {
				t.Errorf("test failed :want=%v-----, got %v ", test.bOK, smcErr)
			} else {
				t.Logf("test passed :want=%v , got %v ", test.bOK, smcErr)
			}

			byb.State.RollbackTx()
		} else {
			if test.bOK {
				t.Errorf("test passed :want=%v-----, got %v ", test.bOK, smcErr)
				byb.State.CommitTx()
				byb.State.StateDB.CommitBlock()

			} else {
				t.Logf("test failed :want=%v, got %v", test.bOK, smcErr)
				byb.State.RollbackTx()
			}
		}
	}
}

func TestTokenByb_SetGasPrice(t *testing.T) {
	var testCases = []struct {
		sender string
		value  uint64
		bOK    bool
	}{
		{"localETK7Zh9hNSPrEKdmCgnHDtFPatcs9WwVL", 0, false},
		{"localPRAbnXe2dqAM5mBr3PLEGxmzvVi1SaXks", 1, true},
		{"localPRAbnXe2dqAM5mBr3PLEGxmzvVi1SaXks", 3000, true},
		{"localPRAbnXe2dqAM5mBr3PLEGxmzvVi1SaXks", 10000000000, false},
	}

	ctx, err := newContext()
	if err != nil {
		panic(err)
	}

	defer dbtest.Close()
	for _, test := range testCases {
		ctx.State.StateDB.BeginBlock()
		ctx.State.BeginTx()

		ctx.Sender.Addr = test.sender

		byb := TokenByb{ctx}
		smcErr := byb.SetGasPrice(test.value)
		if smcErr.ErrorCode != bcerrors.ErrCodeOK {
			if test.bOK {
				t.Errorf("test failed :want=%v-----, got %v ", test.bOK, smcErr)
			} else {
				t.Logf("test passed :want=%v , got %v ", test.bOK, smcErr)
			}

			byb.State.RollbackTx()
		} else {
			if test.bOK {
				t.Errorf("test passed :want=%v-----, got %v ", test.bOK, smcErr)
				byb.State.CommitTx()
				byb.State.StateDB.CommitBlock()

			} else {
				t.Logf("test failed :want=%v, got %v", test.bOK, smcErr)
				byb.State.RollbackTx()
			}
		}
	}
}

func TestTokenByb_Chromo(t *testing.T) {

	ctx, err := newContext()
	if err != nil {
		panic(err)
	}

	defer dbtest.Close()
	for i := 0; i < 100; i++ {
		ctx.State.StateDB.BeginBlock()
		ctx.State.BeginTx()

		byb := TokenByb{ctx}
		chromo := byb.calcNewChromo()
		if chromo != "" {
			fmt.Println("chromo:", chromo)
			byb.State.CommitTx()
			byb.State.StateDB.CommitBlock()

		} else {
			t.Logf("test failed")
			byb.State.RollbackTx()
		}
	}
}
