package transferAgency

import (
	"encoding/json"

	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
)

func require(expr bool, errCode uint32, errInfo string, smcError *smc.Error) bool {
	if expr == false {
		smcError.ErrorCode = errCode
		smcError.ErrorDesc = errInfo
	}
	return expr
}

func (tac *TransferAgency) checkTokenFee(strTokenFee string) (tokenFeeList []TokenFeeInfo, err smc.Error) {
	err.ErrorCode = bcerrors.ErrCodeOK

	jsonErr := json.Unmarshal([]byte(strTokenFee), &tokenFeeList)
	if jsonErr != nil {
		err.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		err.ErrorDesc = jsonErr.Error()
		return
	}
	//对空切片进行检查
	if !require(
		tokenFeeList != nil,
		bcerrors.ErrCodeInterContractsInvalidParameter, "strTokenFee should not be empty", &err) {
		return
	}

	genesisToken, _ := tac.State.GetGenesisToken()
	tokenNameMap := make(map[string]bool)
	for k, item := range tokenFeeList {
		//代币名称不为空
		if !require(
			item.TokenName != "",
			bcerrors.ErrCodeInterContractsInvalidParameter, "TokenName should not be empty", &err) {
			return
		}

		//代币名称不能为本币
		if !require(
			item.TokenName != genesisToken.Name,
			bcerrors.ErrCodeInterContractsInvalidParameter, "TokenName should not be genesisToken", &err) {
			return
		}

		//代币必须为链上支持的币
		tokenAddress, _ := tac.State.GetTokenAddrByName(item.TokenName)
		if !require(
			tokenAddress != "",
			bcerrors.ErrCodeInterContractsInvalidParameter, "TokenName error", &err) {
			return
		}

		//代币手续费大于等于0
		if !require(
			item.Fee.Cmp_(0) >= 0,
			bcerrors.ErrCodeInterContractsInvalidParameter, "TokenFee should not be smaller than zero ", &err) {
			return
		}
		//todo
		_, ok := tokenNameMap[item.TokenName]
		if !require(
			ok != true,
			bcerrors.ErrCodeInterContractsInvalidParameter, "TokenName should not be repeat ", &err) {
			return
		}
		tokenNameMap[item.TokenName] = true

		tokenFeeList[k].TokenAddr = tokenAddress
	}

	return
}
