package helper

import (
	"blockchain/smcsdk/sdk"
)

// NewContractHelper factory method for create ContractHelper
func NewContractHelper(smc sdk.ISmartContract) sdk.IContractHelper {
	o := ContractHelper{}
	o.SetSMC(smc)
	return &o
}
