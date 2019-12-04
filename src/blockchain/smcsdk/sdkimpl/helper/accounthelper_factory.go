package helper

import (
	"blockchain/smcsdk/sdk"
)

// NewAccountHelper factory method for AccountHelper
func NewAccountHelper(smc sdk.ISmartContract) sdk.IAccountHelper {
	o := AccountHelper{}
	o.SetSMC(smc)
	return &o
}
