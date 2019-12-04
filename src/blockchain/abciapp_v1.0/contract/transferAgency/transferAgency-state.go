package transferAgency

func (tac *TransferAgency) manager_() *ManagerInfo {
	if tac.manager__ != nil {
		return tac.manager__
	}

	tac.manager__, _ = tac.getManagerInfoDB()

	return tac.manager__
}

func (tac *TransferAgency) setManager_(v *ManagerInfo) {
	tac.SetManagerInfoDB(v)
}

func (tac *TransferAgency) tokenFeeInfo_(tokenName string) *TokenFeeInfo {
	tac.tokenFeeInfo__, _ = tac.getTokenFeeInfoDB(tokenName)

	return tac.tokenFeeInfo__
}

func (tac *TransferAgency) setTokenFeeInfo_(v *TokenFeeInfo) {
	tac.setTokenFeeInfoDB(v.TokenName, v)
}
