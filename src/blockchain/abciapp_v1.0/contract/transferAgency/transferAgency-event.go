package transferAgency

import (
	"encoding/json"

	"blockchain/abciapp_v1.0/smc"
)

func (tac *TransferAgency) ReceiptOfSetManager(addressList []smc.Address) {
	type ReceiptOfSetManager struct {
		ManagerList []smc.Address `json:"managerList"` // 管理者地址
	}

	managerMsg := ReceiptOfSetManager{
		ManagerList: addressList,
	}

	resBytes, _ := json.Marshal(managerMsg)

	tac.EventHandler.EmitReceipt("onSetManager", resBytes)
}

func (tac *TransferAgency) ReceiptOfSetTokenFeeInfo(tokenFeeInfo []TokenFeeInfo) {
	type ReceiptOfSetTokenFeeInfo struct {
		TokenFeeInfo []TokenFeeInfo `json:"tokenFeeInfo"` // 手续费信息
	}

	tokenFeeInfoMsg := ReceiptOfSetTokenFeeInfo{
		TokenFeeInfo: tokenFeeInfo,
	}

	resBytes, _ := json.Marshal(tokenFeeInfoMsg)

	tac.EventHandler.EmitReceipt("onSetTokenFeeInfo", resBytes)
}
