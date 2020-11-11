package system

import (
	"encoding/hex"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/prototype"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/statedb"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bclib/bcdb"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/pkg/errors"
	"testing"
)

var db *bcdb.GILevelDB

func TestSystem_DeployInternalContract(t *testing.T) {
	var Contracts = []struct {
		name         string
		version      string
		prototype    []string
		gas          []uint64
		effectHeight uint64
		statusCode   int
	}{
		{prototype.TokenBYB, "1.0", []string{prototype.BYBAddSupply,
			prototype.BYBBurn, prototype.BYBChangeChromoOwnerShip, prototype.BYBDelStockHolder, prototype.BYBInit, prototype.BYBNewBlackHole,
			prototype.BYBNewStockHolder, prototype.BYBSetGasPrice, prototype.BYBSetOwner, prototype.BYBTransfer, prototype.BYBTransferByChromo},
			[]uint64{500, 600, 500, 500, 500, 500, 500, 500, 500, 500, 500}, 14, 0},
		//{"fomo-bcb", "1.0", []string{"transfer()", "new()"}, []uint64{500, 600}, 1, 1,},
		//{"fomo-bcb", "1.0", []string{"transfer()", "new()"}, []uint64{500}, 100, 1,},
		//{"fomo-bcb", "1.0", []string{"transfer()", "new()"}, []uint64{500, 600, 660}, 500, 1,},
		//{"fomo-bcb", "1.0", []string{"transfer()", "new()"}, []uint64{500, 600}, 5, 0,},
		//{"fomo-bcb", "1.0", []string{"transfer()", "new()"}, []uint64{500, 600}, 500, 1,},
		//{"fomo-bcb", "1.1", []string{"transfer()", "new()"}, []uint64{500, 600}, 500, 1,},
		//{"fomo-bcb", "1.2", []string{"Init()smc.Error", "Active()smc.Error","Buy(uint64,*Number)smc.Error"}, []uint64{500, 600,600}, 60, 0,},
		//{"fomo-bcb", "1.0",
		//	[]string{"Init()smc.Error", "Active()smc.Error", "SetCommunity(smc.Address)smc.Error", "SetP3D(smc.Address)smc.Error", "Buy(int64,*Number)smc.Error"},
		//	[]uint64{100, 100, 500, 600, 600}, 80, 0,},
		//{prototype.Cgs, "1.1",
		//	[]string{"Init()smc.Error",
		//		"Active()smc.Error",
		//		"Buy(int64,Number)smc.Error",
		//		"BuyXid(int64,int64,Number)smc.Error",
		//		"BuyXaddr(smc.Address,int64,Number)smc.Error",
		//		"BuyXname(string,int64,Number)smc.Error",
		//		"Withdraw()smc.Error",
		//		"ReLoadXid(int64,int64,Number)smc.Error",
		//		"ReLoadXaddr(smc.Address,int64,Number)smc.Error",
		//		"ReLoadXname(string,int64,Number)smc.Error",
		//		"RegisterNameXid(string,int64,Number)smc.Error",
		//		"RegisterNameXaddr(string,smc.Address,Number)smc.Error",
		//		"RegisterNameXname(string,string,Number)smc.Error",
		//		"SetOwner(smc.Address)smc.Error",
		//		"GetCurrentRoundInfo()(int64,Number,Number,Number,bool,Number,int64,int64,smc.Address,string,Number,Number,Number,Number,int64,Number,smc.Error)",
		//		"GetBuyPrice()(Number,smc.Error)",
		//		"GetPrice(Number)(Number,smc.Error)",
		//		"GetKeys(Number)(Number,smc.Error)",
		//		"GetPlayerInfoByAddress(smc.Address)(int64,string,Number,Number,Number,Number,Number,smc.Error)",
		//		"GetPlayerVaults(int64)(Number,Number,Number,smc.Error)",
		//		"GetTimeLeft()(Number,smc.Error)"},
		//	[]uint64{500, 500, 500, 500, 500, 500, 500, 500, 500, 500, 500, 500, 500, 500, 100, 100, 100, 100, 100, 100, 100},
		//	2238, 0,},
	}

	var bcerr bcerrors.BCError
	ic, err := newSystemContext()
	if err != nil {
		t.Fatal("Init invokeContext failed: " + err.Error())
	}
	// Commit and close db once the testing is completed
	defer db.Close()
	codeHash, _ := hex.DecodeString("abcdddd")
	sys := System{&contract.Contract{ic}}
	_, ts := statedbhelper.NewCommittableTransactionID()
	ic.TxState.StateDB.BeginBlock(ts)
	var cAddr smc.Address
	for _, con := range Contracts {
		cAddr, bcerr = sys.DeployInternalContract(con.name, con.version, con.prototype, con.gas, codeHash, con.effectHeight)

		if con.statusCode == 0 && bcerr.ErrorCode != bcerrors.ErrCodeOK {
			t.Errorf("test failed: " + bcerr.Error())
			ic.TxState.RollbackTx()
		} else if con.statusCode == 1 && bcerr.ErrorCode != bcerrors.ErrCodeOK {
			t.Logf("test OKay: " + bcerr.Error())
			ic.TxState.CommitTx()
		} else if con.statusCode == 1 && bcerr.ErrorCode == bcerrors.ErrCodeOK {
			t.Errorf("test failed, the expected error didn't happen")
			ic.TxState.RollbackTx()
		} else {
			t.Logf("test OKay, contract address: " + cAddr)
			ic.TxState.CommitTx()
		}
	}
	//ic.TxState.GetStateDB.RollBlock()
	ic.TxState.StateDB.CommitBlock()
}

func TestSystem_ForbidInternalContract(t *testing.T) {
	var Contracts = []struct {
		addr         smc.Address
		effectHeight uint64
		statusCode   int //1 means failed, 0 means success
	}{
		// 小于等于生效高度
		{"local9ge366rtqV9BHqNwn7fFgA8XbDQmJGZqE", 1, 1},
		//小于当前区块高度
		{"local9ge366rtqV9BHqNwn7fFgA8XbDQmJGZqE", 10, 1},
		//不存在的合约
		{"local9ge366rtqV9BHqNwn7fFgA8XbDQQJGZqA", 100, 1},
		//token-trade
		{"localGDnZKiVYfudRTxuHoctvNhHirrjWuZYed", 100, 0},
	}

	var bcerr bcerrors.BCError
	ic, err := newSystemContext()
	if err != nil {
		t.Fatal("Init invokeContext failed: " + err.Error())
	}
	// Commit and close db once the testing is completed
	defer db.Close()
	sys := System{&contract.Contract{ic}}
	_, ts := statedbhelper.NewCommittableTransactionID()
	ic.TxState.StateDB.BeginBlock(ts)
	var cAddr smc.Address
	for _, con := range Contracts {
		bcerr = sys.ForbidInternalContract(con.addr, con.effectHeight)

		if con.statusCode == 0 && bcerr.ErrorCode != bcerrors.ErrCodeOK {
			t.Errorf("test failed: " + bcerr.Error())
			ic.TxState.RollbackTx()
		} else if con.statusCode == 1 && bcerr.ErrorCode != bcerrors.ErrCodeOK {
			t.Logf("test OKay: " + bcerr.Error())
			ic.TxState.CommitTx()
		} else if con.statusCode == 1 && bcerr.ErrorCode == bcerrors.ErrCodeOK {
			t.Errorf("test failed, the expected error didn't happen")
			ic.TxState.RollbackTx()
		} else {
			t.Logf("test OKay, contract address: " + cAddr)
			ic.TxState.CommitTx()
		}
	}
	//ic.TxState.GetStateDB.RollBlock()
	ic.TxState.StateDB.CommitBlock()
}

func newSystemContext() (*stubapi.InvokeContext, error) {

	// open db
	var err error
	db, err = bcdb.OpenDB("D:\\Work\\Code-SVN\\GIBlockChain\\trunk\\code\\v1.0\\bcchain\\bin\\.appState", "127.0.0.1", "8888")
	if err != nil {
		return nil, err
	}

	stateDB := statedb.NewStateDB()
	//stateDB.BeginBlock()
	// 获取系统合约地址，填入txState
	var syscontractAddr, ownerAddr smc.Address
	genContracts, _ := stateDB.GetGenesisContractList()
	for _, addr := range genContracts {
		contract, err := stateDB.GetContract(addr)
		if err != nil || contract == nil {
			panic(errors.New("Failed to get contract"))
		}
		if contract.Name == prototype.System {
			syscontractAddr = contract.Address
			ownerAddr = contract.Owner
		}
	}
	var txState *statedb.TxState
	//Generate txState to operate stateDB
	// 系统合约的地址和拥有者
	txState = stateDB.NewTxState(syscontractAddr, ownerAddr)
	//txState.BeginTx()

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
		types.Header{},
		nil,
		nil,
		300000,
		"",
	}

	return &invokeContext, nil
}
