package tokenbasic_team

import (
	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/smc"
	. "common/bignumber_v1.0"
	"time"
)

func (t *TBTeam) Withdraw() (err smc.Error) {
	err.ErrorCode = bcerrors.ErrCodeOK

	isCorrectAddr := false
	for _, addr := range addressList {
		if t.Sender.Addr == addr {
			isCorrectAddr = true
			break
		}
	}
	if !require(isCorrectAddr,
		bcerrors.ErrCodeInterContractsNoAuthorization, "Only stipulate address can call withdraw", &err) {
		return
	}

	global := t.global_()
	if len(global.UnlockInfo) == 0 {
		global.init()
		t.setGlobal_(global)
	}

	for i, value := range global.UnlockInfo {
		if !value.Settled {

			// check time
			unlockTime, _ := time.ParseInLocation(timeLayout, value.UnlockTime, time.UTC)
			if !require(t.Block.Time >= unlockTime.Unix(),
				bcerrors.ErrCodeInterContractsInvalidParameter, "It's not time to call withdraw", &err) {
				return
			}

			// check contract account balance
			token, _ := t.State.GetGenesisToken()
			tmp, _ := t.GetBalance(token.Name, t.ContractAcct.Addr)
			balance := NewNumberBigInt(&tmp)
			if !require(balance.Cmp(value.Amount) >= 0,
				bcerrors.ErrCodeInterContractsInsufficientBalance,
				"The balance of contract account is insufficient", &err) {
				return
			}

			eachAmount := value.Amount.Div_(int64(len(addressList))) //每个账户平均多少
			for _, toAddr := range addressList {
				err = t.EventHandler.TransferByAddr(
					token.Address,
					t.ContractAcct.Addr,
					toAddr,
					eachAmount,
				)
				if err.ErrorCode != bcerrors.ErrCodeOK {
					return
				}
			}

			//update global
			global.UnlockInfo[i].Settled = true
			t.setGlobal_(global)
			return
		}
	}

	err.ErrorCode = bcerrors.ErrCodeInterContractsOutOfRange
	return
}
