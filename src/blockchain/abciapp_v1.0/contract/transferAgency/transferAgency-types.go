package transferAgency

import (
	"blockchain/abciapp_v1.0/contract/smcapi"
	"blockchain/abciapp_v1.0/smc"
	. "common/bignumber_v1.0"
)

const (
	Tag      = "TAC(BCB)"
	BuyToken = ""
)

//new
type TransferAgency struct {
	*smcapi.SmcApi
	manager__      *ManagerInfo
	tokenFeeInfo__ *TokenFeeInfo
}

type TokenFeeInfo struct {
	TokenName string      `json:"tokenName"` // 代币名称
	TokenAddr smc.Address `json:"tokenAddr"` // 代币地址
	Fee       Number      `json:"fee"`       // 代币手续费
}

func (tokenFee *TokenFeeInfo) init() {
	tokenFee.TokenName = ""
	tokenFee.Fee = N(0)
}

type ManagerInfo struct {
	AddressList []smc.Address `json:"addressList"` // 管理者地址
}

func (manager *ManagerInfo) init() {
	manager.AddressList = make([]smc.Address, 0)
}
