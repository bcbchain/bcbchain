package transferAgency

import (
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	"github.com/bcbchain/bclib/algorithm"
	. "github.com/bcbchain/bclib/bignumber_v1.0"
)

func (tac *TransferAgency) SetManager(addressList []smc.Address) (err smc.Error) {
	err.ErrorCode = bcerrors.ErrCodeOK

	if !require(
		tac.Sender.Addr == tac.Owner.Addr,
		bcerrors.ErrCodeInterContractsNoAuthorization, "Only contract owner can do SetManager()", &err) {
		return
	}

	if !require(
		len(addressList) > 0 && len(addressList) <= 10,
		bcerrors.ErrCodeInterContractsInvalidParameter, "The length of addressList must be bigger than zero and smaller than eleven", &err) {
		return
	}

	for k := range addressList {
		//检查地址是否合法
		chErr := algorithm.CheckAddress(tac.State.StateDB.GetChainID(), addressList[k])
		if chErr != nil {
			err.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			err.ErrorDesc = chErr.Error()
			return
		}

		if !require(
			addressList[k] != tac.Owner.Addr,
			bcerrors.ErrCodeInterContractsInvalidParameter, "address error", &err) {
			return
		}

		if !require(
			addressList[k] != *tac.ContractAddr,
			bcerrors.ErrCodeInterContractsInvalidParameter, "address error", &err) {
			return
		}

		if !require(
			addressList[k] != tac.ContractAcct.Addr,
			bcerrors.ErrCodeInterContractsInvalidParameter, "address error", &err) {
			return
		}
	}

	managerInfo := tac.manager_()
	managerInfo.AddressList = addressList
	tac.setManager_(managerInfo)

	tac.ReceiptOfSetManager(addressList)

	return
}

func (tac *TransferAgency) SetTokenFee(strTokenFee string) (err smc.Error) {
	err.ErrorCode = bcerrors.ErrCodeOK

	managerInfo := tac.manager_()

	addressFlag := false
	for _, address := range managerInfo.AddressList {
		if tac.Sender.Addr == address {
			addressFlag = true
			break
		}
	}

	if !require(
		addressFlag == true,
		bcerrors.ErrCodeInterContractsNoAuthorization, "Only manager can do SetTokenFee()", &err) {
		return
	}

	tokenFeeList, err := tac.checkTokenFee(strTokenFee)
	if err.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	for k := range tokenFeeList {
		tac.setTokenFeeInfo_(&tokenFeeList[k])
	}

	tac.ReceiptOfSetTokenFeeInfo(tokenFeeList)

	return
}

func (tac *TransferAgency) Transfer(tokenName string, to smc.Address, amount Number) (err smc.Error) {
	err.ErrorCode = bcerrors.ErrCodeOK

	//代币名称不能为空
	if !require(
		tokenName != "",
		bcerrors.ErrCodeInterContractsInvalidParameter, "tokenName should not be empty", &err) {
		return
	}

	//判断tokenName是否存在
	tokenFeeInfo := tac.tokenFeeInfo_(tokenName)
	if !require(
		tokenFeeInfo.TokenName != "",
		bcerrors.ErrCodeInterContractsInvalidParameter, "The transaction is not supported", &err) {
		return
	}

	if !require(
		tokenFeeInfo.Fee.Cmp_(0) > 0,
		bcerrors.ErrCodeInterContractsInvalidParameter, "The transaction is not supported", &err) {
		return
	}

	//检查地址是否合法
	chErr := algorithm.CheckAddress(tac.State.StateDB.GetChainID(), to)
	if chErr != nil {
		err.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		err.ErrorDesc = chErr.Error()
		return
	}

	if !require(
		amount.Cmp_(0) > 0,
		bcerrors.ErrCodeInterContractsInvalidParameter, "Amount should bigger than zero", &err) {
		return
	}

	//transfer to contract account
	if err = tac.EventHandler.TransferByAddr(
		tokenFeeInfo.TokenAddr,
		tac.Sender.Addr,
		tac.ContractAcct.Addr,
		amount.Add(tokenFeeInfo.Fee),
	); err.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	if err = tac.EventHandler.TransferByAddr(
		tokenFeeInfo.TokenAddr,
		tac.ContractAcct.Addr,
		to,
		amount); err.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	return
}

func (tac *TransferAgency) WithdrawFunds(tokenName string, withdrawAmount Number) (err smc.Error) {
	err.ErrorCode = bcerrors.ErrCodeOK

	if !require(
		tac.Sender.Addr == tac.Owner.Addr,
		bcerrors.ErrCodeInterContractsNoAuthorization, "Only contract owner can do WithdrawFunds()", &err) {
		return
	}

	if !require(
		withdrawAmount.Cmp_(0) > 0,
		bcerrors.ErrCodeInterContractsInvalidParameter, "amount should be bigger than zero", &err) {
		return
	}

	balance, err := tac.GetBalance(tokenName, tac.ContractAcct.Addr)
	if err.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	if !require(
		withdrawAmount.Cmp(NB(&balance)) <= 0,
		bcerrors.ErrCodeInterContractsInvalidParameter, "WithdrawAmount cannot be larger than balance", &err) {
		return
	}

	tokenAddress, _ := tac.State.GetTokenAddrByName(tokenName)
	if err = tac.EventHandler.TransferByAddr(
		tokenAddress,
		tac.ContractAcct.Addr,
		tac.Owner.Addr,
		withdrawAmount); err.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	return
}
