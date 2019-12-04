package helper

import "blockchain/smcsdk/sdk"

// NewIBCStubHelper factory method to create IBCStubHelper
func NewIBCStubHelper(smc sdk.ISmartContract) sdk.IIBCStubHelper {
	o := IBCStubHelper{}
	o.SetSMC(smc)
	return &o
}
