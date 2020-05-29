package tokenbasic

import (
	"math/big"

	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	"github.com/bcbchain/bclib/bignumber_v1.0"
)

// TokenBasic is a reference of Contract structure
type TokenBasic struct {
	*contract.Contract
}

// Transfer is used to transfer token from sender to another specified account
// In the TokenBasic contract, it's only used to transfer the basic token
func (contract *TokenBasic) Transfer(to smc.Address, value big.Int) (smcError smc.Error) {
	// The value cannot be negative
	if bignumber.Compare(value, bignumber.Zero()) < 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid value: it cannot be a negative"
		return
	}

	// Checking "to" account, cannot transfer token to oneself
	sender := contract.Sender()
	if sender.Address() == to {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsUnsupportTransToSelf
		return
	}
	// Getting Sender's balance and calculate the new balance
	senderOldBalance := sender.Balance(contract.Ctx.TxState.ContractAddress)
	senderNewBalance := bignumber.Sub(senderOldBalance, value)
	if bignumber.Compare(senderNewBalance, bignumber.Zero()) < 0 { //insufficient balance
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInsufficientBalance
		return
	} else if bignumber.Compare(senderNewBalance, senderOldBalance) > 0 { //incorrect new balance
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidBalance
		return
	}
	// Set new balance to "sender" account
	if smcError = sender.SetBalance(contract.Ctx.TxState.ContractAddress, senderNewBalance); smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	//Getting to's balance and calculate the new balance
	toAccount := contract.GetAccount(to)

	toAccountOldBalance := toAccount.Balance(contract.Ctx.TxState.ContractAddress)
	toAccountNewBalance := bignumber.Add(toAccountOldBalance, value)
	if bignumber.Compare(toAccountNewBalance, toAccountOldBalance) < 0 { //incorrect new balance
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidBalance
		return
	}
	// Set new balance to "to" account
	return toAccount.SetBalance(contract.Ctx.TxState.ContractAddress, toAccountNewBalance)
}

//SetGasPrice is used to set gas price for token-basic contract
func (contract *TokenBasic) SetGasPrice(value uint64) (smcError smc.Error) {

	sender := contract.Sender()
	owner := contract.Owner()

	// Only owner can perform
	if sender.Address() != owner.Address() {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}

	// Token's gas price could not be smaller than GasBasePrice
	// and up to Max_Gas_Price (1E9 cong)
	if value < contract.GasBasePrice() || value > smc.Max_Gas_Price {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidGasPrice
		return
	}

	token := contract.Token()
	return token.SetGasPrice(value)
}

// SetGasBasePrice is used to set gas base price.
// The gas base price is a minimum limit to all of token's.
// All new tokens' gas price could not be set to a value what smaller than gas base price.
func (contract *TokenBasic) SetGasBasePrice(value uint64) (smcError smc.Error) {

	sender := contract.Sender()
	owner := contract.Owner()

	// Only the owner of basic contract can perform this function
	if sender.Address() != owner.Address() {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}

	// GasBasePrice must be limited within 1 ~ smc.Max_Gas_Price
	if value < 1 || value > smc.Max_Gas_Price {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidGasPrice
		return
	}

	return contract.SetGasBasePriceForToken(value)
}
