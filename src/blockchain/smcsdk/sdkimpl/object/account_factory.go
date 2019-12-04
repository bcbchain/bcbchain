package object

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/types"
)

// NewAccount factory method for create Account with address
func NewAccount(smc sdk.ISmartContract, addr types.Address) sdk.IAccount {
	sdk.RequireAddress(addr)

	account := &Account{
		address: addr,
	}
	account.SetSMC(smc)

	return account
}

// NewAccountWithPubKey factory method for create Account with pubKey
func NewAccountWithPubKey(smc sdk.ISmartContract, pubKey types.PubKey) sdk.IAccount {
	sdk.Require(pubKey != nil && len(pubKey) == 32,
		types.ErrInvalidParameter, "Invalid PubKey")

	account := &Account{
		address: smc.Helper().BlockChainHelper().CalcAccountFromPubKey(pubKey),
		pubKey:  pubKey,
	}
	account.SetSMC(smc)

	return account
}
