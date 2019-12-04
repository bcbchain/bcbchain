package helper

import (
	"blockchain/smcsdk/sdk"
)

// NewHelper factory method for create IHelper object
func NewHelper(smc sdk.ISmartContract) sdk.IHelper {
	o := Helper{
		accountHelper:    NewAccountHelper(smc),
		blockChainHelper: NewBlockChainHelper(smc),
		contractHelper:   NewContractHelper(smc),
		receiptHelper:    NewReceiptHelper(smc),
		genesisHelper:    NewGenesisHelper(smc),
		stateHelper:      NewStateHelper(smc),
		tokenHelper:      NewTokenHelper(smc),
		buildHelper:      NewBuildHelper(smc),
		ibcHelper:        NewIBCHelper(smc),
		ibcStubHelper:    NewIBCStubHelper(smc),
	}
	o.SetSMC(smc)

	return &o
}
