package tokenbasic

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/types"
	ut "blockchain/smcsdk/utest"
)

var (
	contractName    = "token-basic" //contract name
	contractMethods = []string{"Transfer(types.Address,big.Int)types.Error", "SetGasPrice(uint64)types.Error", "SetGasBasePrice(uint64)types.Error"}
	orgID           = "orgAJrbk6Wdf7TCbunrXXS5kKvbWVszhC1T"
)

type TestObject struct {
	obj *TokenBasic
}

//FuncRecover recover panic by Assert
func FuncRecover(err *types.Error) {
	if rerr := recover(); rerr != nil {
		if _, ok := rerr.(types.Error); ok {
			err.ErrorCode = rerr.(types.Error).ErrorCode
			err.ErrorDesc = rerr.(types.Error).ErrorDesc
		} else {
			panic(rerr)
		}
	}
}

func NewTestObject(sender sdk.IAccount) *TestObject {
	return &TestObject{&TokenBasic{sdk: ut.UTP.ISmartContract}}
}
func (t *TestObject) transfer(balance bn.Number) *TestObject {
	t.obj.sdk.Message().Sender().Transfer(t.obj.sdk.Message().Contract().Account(), balance)
	return t
}
func (t *TestObject) setSender(sender sdk.IAccount) *TestObject {
	t.obj.sdk = ut.SetSender(sender.Address())
	return t
}
func (t *TestObject) run() *TestObject {
	t.obj.sdk = ut.ResetMsg()
	return t
}

func (t *TestObject) InitChain() (err types.Error) {
	err.ErrorCode = types.CodeOK
	defer FuncRecover(&err)
	ut.NextBlock(1)
	t.obj.InitChain()
	ut.Commit()
	return
}

func (t *TestObject) Transfer(to types.Address, value bn.Number) (err types.Error) {
	err.ErrorCode = types.CodeOK
	defer FuncRecover(&err)
	ut.NextBlock(1)
	t.obj.Transfer(to, value)
	ut.Commit()
	return
}

func (t *TestObject) SetGasPrice(value int64) (err types.Error) {
	err.ErrorCode = types.CodeOK
	defer FuncRecover(&err)
	ut.NextBlock(1)
	t.obj.SetGasPrice(value)
	ut.Commit()
	return
}

func (t *TestObject) SetGasBasePrice(value int64) (err types.Error) {
	err.ErrorCode = types.CodeOK
	defer FuncRecover(&err)
	ut.NextBlock(1)
	t.obj.SetBaseGasPrice(value)
	ut.Commit()
	return
}
