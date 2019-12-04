package object

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/types"
)

// NewMessage factory method for create message with all receipt property
func NewMessage(smc sdk.ISmartContract, contract sdk.IContract, methodID string, items []types.HexBytes, _sender, _payer types.Address, origins []types.Address, receipts []types.KVPair) sdk.IMessage {
	senderAcct := NewAccount(smc, _sender)
	var payerAcct sdk.IAccount
	if _payer != "" {
		payerAcct = NewAccount(smc, _payer)
	}

	var gasPrice int64
	token := smc.Helper().TokenHelper().TokenOfContract(contract.Address())
	if token != nil {
		gasPrice = token.GasPrice()
	} else {
		gasPrice = smc.Helper().TokenHelper().BaseGasPrice()
	}

	o := &Message{
		contract:       contract,
		methodID:       methodID,
		items:          items,
		gasPrice:       gasPrice,
		sender:         senderAcct,
		payer:          payerAcct,
		origins:        origins,
		inputReceipts:  receipts,
		outputReceipts: make([]types.KVPair, 0),
	}
	o.SetSMC(smc)

	return o
}
