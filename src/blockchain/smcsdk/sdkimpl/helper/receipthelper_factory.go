package helper

import (
	"blockchain/smcsdk/sdk"
)

// NewReceiptHelper factory method to create IReceiptHelper
func NewReceiptHelper(smc sdk.ISmartContract) sdk.IReceiptHelper {
	o := ReceiptHelper{}
	o.SetSMC(smc)
	return &o
}
