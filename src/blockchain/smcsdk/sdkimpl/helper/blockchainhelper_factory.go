package helper

import (
	"blockchain/smcsdk/sdk"
)

// NewBlockChainHelper factory method fro create IBlockChainHelper object
func NewBlockChainHelper(smc sdk.ISmartContract) sdk.IBlockChainHelper {
	o := BlockChainHelper{}
	o.SetSMC(smc)
	return &o
}
