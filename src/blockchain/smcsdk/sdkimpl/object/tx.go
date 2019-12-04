package object

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdkimpl"
)

// Tx tx detail information
type Tx struct {
	smc sdk.ISmartContract //指向智能合约API对象指针

	note     string       //交易的备注
	gasLimit int64        //交易传入的最大燃料数
	gasLeft  int64        //剩余的燃料数
	txHash   []byte       //交易hash
	signer   sdk.IAccount //交易发送者的账户信息
}

var _ sdk.ITx = (*Tx)(nil)
var _ sdkimpl.IAcquireSMC = (*Tx)(nil)

// SMC get smart contract object
func (t *Tx) SMC() sdk.ISmartContract { return t.smc }

// SetSMC set smart contract object
func (t *Tx) SetSMC(smc sdk.ISmartContract) { t.smc = smc }

// Note get tx's note
func (t *Tx) Note() string { return t.note }

// GasLimit get tx's gasLimit
func (t *Tx) GasLimit() int64 { return t.gasLimit }

// GasLeft get tx's gasLeft
func (t *Tx) GasLeft() int64 { return t.gasLeft }

// SetGasLeft get tx's gasLeft
func (t *Tx) SetGasLeft(gasLeft int64) { t.gasLeft = gasLeft }

// TxHash get tx's hash
func (t *Tx) TxHash() []byte { return t.txHash }

// Sender get tx's sender
func (t *Tx) Signer() sdk.IAccount { return t.signer }
