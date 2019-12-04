package tokenbyb

import (
	"blockchain/abciapp_v1.0/smc"
	"blockchain/abciapp_v1.0/statedb"
	"code/contract/smcapi"
	"code/contract/stubapi"
	"common/bcdb"
	"common/bignumber_v1.0"
	"fmt"
	"math/big"
	"testing"
)

//TestSetBlackHoleaddrs

func TestSetBlackHoleaddrs(t *testing.T) {

	// open db
	db, err := bcdb.OpenDB(".appstate", "127.0.0.1", "8888")
	if err != nil {
		panic(err)
	}
	db.Print()
	stateDB := statedb.NewStateDB(db)
	stateDB.BeginBlock()

	var txState *statedb.TxState
	//Generate txState to operate stateDB
	ownerAddr := "devtestQGzn5MhhksTcyDav39yZPW3ZyjexC7AaH"
	contractAddr := "devtestQGzn5MhhksTcyDav39yZPW3ZyjexC7AaH"
	txState = stateDB.NewTxState(contractAddr, ownerAddr)
	txState.BeginTx()

	// Generate accounts and execute
	sender := &stubapi.Account{
		ownerAddr,
		txState,
	}
	// Using tokenbasic owner as it doesn't be changed
	owner := &stubapi.Account{
		ownerAddr,
		txState,
	}
	api := &smcapi.SmcApi{sender, owner, nil, &contractAddr, txState, nil, nil}
	byb := TokenByb{api}

	var tests = []struct {
		blackHoles []smc.Address
	}{
		{[]smc.Address{"localA6QgXDbgMwWi6HR49pQ1mWNxA2EzKwoM4"}},
		{[]smc.Address{"detestA6QgXDbgMwWi6HR49pQ1mWNxA2EzLZW66"}},
		{[]smc.Address{"localA6QgXDbgMwWi6HR49pQ1mWNxA2EzKwoM4"}},
	}

	for _, con := range tests {
		byb.setBlackHole(con.blackHoles)
		BybBlackHolestest, err := byb.getBlackHole()
		if err != nil {
			return
		}
		a := CompareBlackHoles(con.blackHoles, BybBlackHolestest)
		if a != true {
			panic("error")
		} else {
			t.Log("success")
			fmt.Println("SetBlackHole Address:", con.blackHoles)
			fmt.Println("GetBlackHole Address:", BybBlackHolestest)
		}
	}
	db.Close()
}

//SetBybStockHolders

func TestSetBybStockHolders(t *testing.T) {
	// open db
	db, err := bcdb.OpenDB(".appstate", "127.0.0.1", "8888")
	if err != nil {
		panic(err)
	}
	db.Print()
	stateDB := statedb.NewStateDB(db)
	stateDB.BeginBlock()

	var txState *statedb.TxState
	//Generate txState to operate stateDB
	ownerAddr := "devtestQGzn5MhhksTcyDav39yZPW3ZyjexC7AaH"
	contractAddr := "devtestQGzn5MhhksTcyDav39yZPW3ZyjexC7AaH"
	txState = stateDB.NewTxState(contractAddr, ownerAddr)
	txState.BeginTx()

	// Generate accounts and execute
	sender := &stubapi.Account{
		ownerAddr,
		txState,
	}
	// Using tokenbasic owner as it doesn't be changed
	owner := &stubapi.Account{
		ownerAddr,
		txState,
	}
	api := &smcapi.SmcApi{sender, owner, nil, &contractAddr, txState, nil, nil}
	byb := TokenByb{api}

	var tests = []struct {
		stockHolders []smc.Address
	}{
		{[]smc.Address{"localA6QgXDbgMwWi6HR49pQ1mWNxA2EzKwoM4"}},
		{[]smc.Address{"detestA6QgXDbgMwWi6HR49pQ1mWNxA2EzLZW66"}},
		{[]smc.Address{"localA6QgXDbgMwWi6HR49pQ1mWNxA2EzKwoM4"}}}

	for _, con := range tests {
		byb.setBybStockHolders(con.stockHolders)
		BybStockHolderstest, err := byb.getBybStockHolders()
		if err != nil {
			return
		}
		a := CompareBybStockHolers(con.stockHolders, BybStockHolderstest)
		if a != true {
			panic("error")
		} else {
			t.Log("success")
			fmt.Println("SetBlackHole Address:", con.stockHolders)
			fmt.Println("GetBlackHole Address:", BybStockHolderstest)
		}
	}
	db.Close()
}

//TestSetBybBalance

func TestSetBybBalance(t *testing.T) {
	// open db
	db, err := bcdb.OpenDB(".appstate", "127.0.0.1", "8888")
	if err != nil {
		panic(err)
	}
	db.Print()
	stateDB := statedb.NewStateDB(db)
	stateDB.BeginBlock()

	var txState *statedb.TxState
	//Generate txState to operate stateDB
	ownerAddr := "devtestQGzn5MhhksTcyDav39yZPW3ZyjexC7AaH"
	contractAddr := "devtestQGzn5MhhksTcyDav39yZPW3ZyjexC7AaH"
	txState = stateDB.NewTxState(contractAddr, ownerAddr)
	txState.BeginTx()

	// Generate accounts and execute
	sender := &stubapi.Account{
		ownerAddr,
		txState,
	}
	// Using tokenbasic owner as it doesn't be changed
	owner := &stubapi.Account{
		ownerAddr,
		txState,
	}
	api := &smcapi.SmcApi{sender, owner, nil, &contractAddr, txState, nil, nil}
	byb := TokenByb{api}

	chromo1 := "01020304"
	chromo2 := "0102030411"

	var tests = []struct {
		bybBalanceaddr smc.Address
		bybBalance     []bybBalance
	}{
		{"localA6QgXDbgMwWi6HR49pQ1mWNxA2EzKwoM4", []bybBalance{{chromo1, *big.NewInt(10)}, {chromo1, *big.NewInt(1000)}, {chromo2, *big.NewInt(1000)}}},
		{"DevtestA6QgXDbgMwWi6HR49pQ1mWNxA2EzLZW66", []bybBalance{{chromo1, *big.NewInt(100)}, {chromo2, *big.NewInt(1000)}}},
		{"localA6QgXDbgMwWi6HR49pQ1mWNxA2Ezuabb643", []bybBalance{{chromo1, *big.NewInt(1000)}, {chromo2, *big.NewInt(1000)}}},
		{"localA6QgXDbgMwWi6HR49pQ1mWNxA2EzuaLBWNB", []bybBalance{{chromo2, *big.NewInt(1)}, {chromo2, *big.NewInt(100)}}}, //金额大于owner的金额
	}

	for _, con := range tests {
		byb.setBybBalance(con.bybBalanceaddr, con.bybBalance)
		BybBalancetest, err := byb.getBybBalance(con.bybBalanceaddr)
		if err != nil {
			return
		}
		a := CompareBybBalance(con.bybBalance, BybBalancetest)

		if a != true {
			panic("error")
		} else {
			t.Log("success")
		}
	}
	db.Close()
}

func CompareBybBalance(setBybBalance, getBybBalance []bybBalance) bool {
	for i := 0; i < len(getBybBalance); i++ {
		if getBybBalance[i].Chromo != setBybBalance[i].Chromo || bignumber.Compare(getBybBalance[i].Value, setBybBalance[i].Value) != 0 {
			return false
		}
	}
	return true
}

func CompareBybStockHolers(setBybStockHolders, getBybStockHolders []smc.Address) bool {
	if setBybStockHolders[0] == getBybStockHolders[0] {
		return true
	} else {
		return false
	}
}

func CompareBlackHoles(setBlackHoles, getBlackHoles []smc.Address) bool {

	if setBlackHoles[0] == getBlackHoles[0] {
		return true
	} else {
		return false
	}
}
