package tokenbasic_cancellation

import (
	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/contract/smcapi"
	"blockchain/abciapp_v1.0/smc"
	"common/bignumber_v1.0"
	"encoding/json"
	"fmt"

	"math/big"
)

type TBCancellation struct {
	*smcapi.SmcApi
}

const (
	// Define the minimum of token supply for issuing new token.
	BCB_TOKEN_SUPPLY     = 5000000000000000000
	BCB_TOKEN_NEW_SUPPLY = 66000000000000000
	BCB_TOKEN_SUB_SUPPLY = 4934000000000000000
)

func (tbc *TBCancellation) Cancel() (smcError smc.Error) {
	smcError.ErrorCode = bcerrors.ErrCodeOK

	// Only the owner of basic contract can perform this function
	tokenBasicOwner, _ := tbc.GetOwnerOfGenesisToken()
	if (tbc.Sender.Addr != tbc.Owner.Addr) || (tbc.Owner.Addr != tokenBasicOwner) {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		smcError.ErrorDesc = "Only contract & token basic owner just can call Cancellation()"
		return
	}
	//get total supply
	tokenInfo, _ := tbc.SmcApi.State.GetGenesisToken()
	tokenInfo, _ = tbc.SmcApi.State.GetToken(tokenInfo.Address)
	oldSupply := tokenInfo.TotalSupply
	//check
	if bignumber.Compare(bignumber.UintToBigInt(BCB_TOKEN_SUPPLY), oldSupply) != 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidBalance
		smcError.ErrorDesc = " Error supply balance"
		return
	}

	//get Owner token balance
	ownerBalance, _ := tbc.Owner.BalanceOf(tokenInfo.Address)

	//check Owner balance
	subSupply := bignumber.UintToBigInt(BCB_TOKEN_SUB_SUPPLY)
	if bignumber.Compare(ownerBalance, subSupply) < 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidBalance
		smcError.ErrorDesc = " Owner balance amount should be bigger than " + fmt.Sprintf("%d", BCB_TOKEN_SUB_SUPPLY)
		return
	}

	newSupply := bignumber.UintToBigInt(BCB_TOKEN_NEW_SUPPLY)
	tokenInfo.TotalSupply = newSupply

	// set new token supply
	err := tbc.State.SetToken(tokenInfo)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	// set balance
	smcError = tbc.Owner.SetBalance(tokenInfo.Address, bignumber.Sub(ownerBalance, subSupply))
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	//fire event
	tbc.ReceiptOfCancellation(newSupply)

	// if func Cancel() is success, destroy this function
	smcError = tbc.ForbidSpecificContract(*tbc.ContractAddr, uint64(tbc.Block.Height+1))
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}
	return
}

func (tbc *TBCancellation) ReceiptOfCancellation(supply big.Int) {
	type ReceiptOfCancellation struct {
		NewBCBSupply big.Int `json:"newBCBSupply"` // bcb剩余供应量
	}

	setCancellation := ReceiptOfCancellation{
		NewBCBSupply: supply,
	}

	resBytes, _ := json.Marshal(setCancellation)
	tbc.EventHandler.EmitReceipt("onCancellation", resBytes)
}
