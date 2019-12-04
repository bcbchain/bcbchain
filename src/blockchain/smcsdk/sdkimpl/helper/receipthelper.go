package helper

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/crypto/sha3"
	"blockchain/smcsdk/sdk/ibc"
	"blockchain/smcsdk/sdk/jsoniter"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
	"blockchain/smcsdk/sdkimpl/object"
	"fmt"
	"reflect"
	"strings"
)

// ReceiptHelper receipt helper information
type ReceiptHelper struct {
	smc sdk.ISmartContract //指向智能合约API对象指针
}

var _ sdk.IReceiptHelper = (*ReceiptHelper)(nil)
var _ sdkimpl.IAcquireSMC = (*ReceiptHelper)(nil)

// SMC get smart contract object
func (rh *ReceiptHelper) SMC() sdk.ISmartContract { return rh.smc }

// SetSMC set smart contract object
func (rh *ReceiptHelper) SetSMC(smc sdk.ISmartContract) { rh.smc = smc }

// Emit emit receipt object
func (rh *ReceiptHelper) Emit(receipt interface{}) {
	if receipt == nil {
		return
	}

	bz, err := jsoniter.Marshal(receipt)
	if err != nil {
		sdkimpl.Logger.Fatalf("[sdk]Cannot marshal receipt data=%v", receipt)
		sdkimpl.Logger.Flush()
		panic(err)
	}

	rcpt := std.Receipt{
		Name:         rh.receiptName(receipt),
		ContractAddr: rh.smc.Message().Contract().Address(),
		Bytes:        bz,
		Hash:         nil,
	}
	rcpt.Hash = sha3.Sum256([]byte(rcpt.Name), []byte(rcpt.ContractAddr), bz)
	resBytes, _ := jsoniter.Marshal(rcpt) // nolint unhandled

	//将收据添加到message
	receipts := types.KVPair{
		Key:   []byte(fmt.Sprintf("/%d/%s", len(rh.smc.Message().(*object.Message).OutputReceipts()), rcpt.Name)),
		Value: resBytes,
	}
	rh.smc.Message().(*object.Message).FillOutputReceipts(receipts)
}

func (rh *ReceiptHelper) receiptName(receipt interface{}) string {
	typeOfInterface := reflect.TypeOf(receipt).String()
	typeOfInterface = strings.TrimLeft(typeOfInterface, "*")

	if strings.HasPrefix(typeOfInterface, "std.") {
		prefixLen := len("std.")
		return "std::" + strings.ToLower(typeOfInterface[prefixLen:prefixLen+1]) + typeOfInterface[prefixLen+1:]
	} else if strings.HasPrefix(typeOfInterface, "ibc.") {
		prefixLen := len("ibc.")
		name := "ibc::" + strings.ToLower(typeOfInterface[prefixLen:prefixLen+1]) + typeOfInterface[prefixLen+1:]
		switch packet := receipt.(type) {
		case ibc.Packet:
			name += "/" + packet.QueueID
		case *ibc.Packet:
			name += "/" + packet.QueueID
		}
		return name
	}

	return typeOfInterface
}
