package tokentemplet

import (
	"fmt"
	"math/big"

	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	"github.com/bcbchain/bclib/bignumber_v1.0"
)

// TokenTemplet is a reference of Contract structure
type TokenTemplet struct {
	*contract.Contract
}

const MIN_Token_Supply = 1000000000

// The maximum accounts per batch transfer
const MAX_ACCOUNTS_FOR_BATCH = 1000

// Transfer is a function to transfer from Sender to another specified account
// In TokenTemplet contract, it's only used to transfer a specified token of contract
func (contract *TokenTemplet) Transfer(to smc.Address, value big.Int) (smcError smc.Error) {
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

//BatchTransfer is a function to transfer from Sender to another specified accounts in batch
func (contract *TokenTemplet) BatchTransfer(toList []smc.Address, value big.Int) (smcError smc.Error) {

	// The value cannot be negative
	if bignumber.Compare(value, bignumber.Zero()) < 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid value: it cannot be a negative"
		return
	}

	// Checking accounts limit in batch
	if len(toList) > MAX_ACCOUNTS_FOR_BATCH {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsOutOfRange
		err := fmt.Sprintf("Exceeds the limit:, up to %d accounts be supported for batch transfer", MAX_ACCOUNTS_FOR_BATCH)
		smcError.ErrorDesc = err
		return
	}

	// Calculate the sum
	sum, error := bignumber.Multi(value, len(toList))
	if error != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = error.Error()
		return
	}
	if bignumber.Compare(sum, value) < 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsOutOfRange
		return
	}

	sender := contract.Sender()
	senderOldBalance := sender.Balance(contract.Ctx.TxState.ContractAddress)
	// Checking sender's balance
	senderNewBalance := bignumber.Sub(senderOldBalance, sum)
	if bignumber.Compare(senderNewBalance, bignumber.Zero()) < 0 { //insufficient balance
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInsufficientBalance
		return
	} else if bignumber.Compare(senderNewBalance, senderOldBalance) > 0 { //incorrect new balance
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidBalance
		return
	}

	// Set sender's new balance
	if smcError = sender.SetBalance(contract.Ctx.TxState.ContractAddress, senderNewBalance); smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}
	// Set to's new balance one by one. Return error once when an error be met,
	// and all of transfer will be rollback.
	for _, to := range toList {
		// Checking "to" account, cannot transfer token to oneself
		if sender.Address() == to {
			smcError.ErrorCode = bcerrors.ErrCodeInterContractsUnsupportTransToSelf
			return
		}

		toAccount := contract.GetAccount(to)
		toAccountOldBalance := toAccount.Balance(contract.Ctx.TxState.ContractAddress)

		// Calculate to's new balance
		toAccountNewBalance := bignumber.Add(toAccountOldBalance, value)
		if bignumber.Compare(toAccountNewBalance, toAccountOldBalance) < 0 { //incorrect balance
			smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidBalance
			return
		}
		if smcError = toAccount.SetBalance(contract.Ctx.TxState.ContractAddress, toAccountNewBalance); smcError.ErrorCode != bcerrors.ErrCodeOK {
			return
		}
	}
	smcError.ErrorCode = bcerrors.ErrCodeOK
	return
}

// AddSupply is used to add token's supply after the token be issued.
// Only token's owner can perform. And the increased token will be transferred to owner.
func (contract *TokenTemplet) AddSupply(value big.Int) (smcError smc.Error) {

	// Value cannot be negative or ZERO
	if bignumber.Compare(value, bignumber.Zero()) <= 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid value: it cannot be a negative"
		return
	}
	sender := contract.Sender()
	owner := contract.Owner()

	// Only token's owner can perform
	if sender.Address() != owner.Address() {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}

	// Get token's total supply and calculate the new total suppl.
	token := contract.Token()
	totalSupply, smcError := token.GetTotalSupply()
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}
	newSupply := bignumber.Add(totalSupply, value)
	if bignumber.Compare(newSupply, totalSupply) < 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidSupply
		return
	}
	if smcError = token.SetSupply(newSupply); smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	// Get owner's balance and calculate new balance
	ownerOldBalance := owner.Balance(contract.Ctx.TxState.ContractAddress)
	ownerNewBalance := bignumber.Add(ownerOldBalance, value)
	if bignumber.Compare(ownerNewBalance, ownerOldBalance) < 0 { //incorrect balance
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidBalance
		return
	}
	return owner.SetBalance(contract.Ctx.TxState.ContractAddress, ownerNewBalance)
}

// Burn is used to burn token's supply after the token be issued.
// Only token's owner can perform. And the owner's balance will be decreased as well.
func (contract *TokenTemplet) Burn(value big.Int) (smcError smc.Error) {

	// Value cannot be negative
	if bignumber.Compare(value, bignumber.Zero()) < 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid value: it cannot be a negative"
		return
	}

	sender := contract.Sender()
	owner := contract.Owner()
	// Only token's owner can perform
	if sender.Address() != owner.Address() {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}

	// Get owner's balance and calculate new balance
	ownerOldBalance := owner.Balance(contract.Ctx.TxState.ContractAddress)
	ownerNewBalance := bignumber.Sub(ownerOldBalance, value)
	if bignumber.Compare(ownerNewBalance, bignumber.Zero()) < 0 { //insufficient balance
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInsufficientBalance
		return
	} else if bignumber.Compare(ownerNewBalance, ownerOldBalance) > 0 { //incorrect balance
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidBalance
		return
	}
	if smcError = sender.SetBalance(contract.Ctx.TxState.ContractAddress, ownerNewBalance); smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	// Get token's total supply and calculate the new total suppl
	token := contract.Token()
	totalSupply, _ := token.GetTotalSupply()
	newSupply := bignumber.Sub(totalSupply, value)
	if bignumber.Compare(newSupply, totalSupply) > 0 { //incorrect balance
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidSupply
		return
	} else if bignumber.Compare(newSupply, *big.NewInt(MIN_Token_Supply)) < 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidSupply
		smcError.ErrorDesc = "Invalid supply: token supply is less than limit"
		return
	}

	return token.Burn(newSupply)
}

// SetOwner is used to set a new account as the token's owner.
// Only token's owner can perform. And the owner's balance will be transferred to new owner as well.
func (contract *TokenTemplet) SetOwner(newOwner smc.Address) (smcError smc.Error) {

	sender := contract.Sender()
	owner := contract.Owner()
	// Only token's owner can perform
	if sender.Address() != owner.Address() {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}

	// Transfer owner's balance to new owner
	balance := owner.Balance(contract.Ctx.TxState.ContractAddress)
	if smcError = contract.Transfer(newOwner, balance); smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}
	// Set newOwner for token
	token := contract.Token()
	return token.SetNewOwner(newOwner)
}

// SetGasPrice is used to set gas price for specified token
func (contract *TokenTemplet) SetGasPrice(value uint64) (smcError smc.Error) {

	sender := contract.Sender()
	owner := contract.Owner()
	// Only token's owner can perform
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
	//Set token's gas price
	token := contract.Token()
	return token.SetGasPrice(value)
}

// UdcCreate is used to create an UDC (Unmatured Deposit Certificate) for specified token
// Anyone could call this function to create an UDC
// Parameters:
// 		to: the account address of UDC receiver
//		value: the amount of UDC (in uint Cong)
// 		MatureDate: Mature Date of this UDC (Format as "2016-01-01")
// Return Data:
// 		udcHash: The UDC Hash of this created UDC
//		err: Error or nil
func (contract *TokenTemplet) UdcCreate(to smc.Address, value big.Int, matureDate string) (udcHash smc.Hash, smcError smc.Error) {
	// Get Sender and check its balance
	//sender := contract.Sender()
	//balance := sender.Balance()
	//if bignumber.Compare(balance, value) < 0{
	//	return nil, errors.New("Insufficient balance")
	//}

	// Check MatureDate, it should be late than the current date
	// If the MatureDate is incorrect, return error

	// Get token object
	// token := contract.Token()
	// Create token UDC
	// return token.CreateUDC(to, value, matureDate)
	return
}

// UdcTransfer is used to transfer an UDC (Unmatured Deposit Certificate)  to specified receiver
// And a new Rest UDC would be created and transferred to the sender if it has a rest
// Only the owner of UDC is allowed to execute this function
// Parameters:
//		udc: the UDC Hash that belongs to the sender
// 		to: the account address of UDC receiver
//		value: the amount of UDC (in uint Cong)
// Return Data:
// 		receiverUdcHash: The UDC Hash that sent to receiver
//		restUdcHash: The UDC Hash that sent to sender if it's rest
//		err: Error or nil
func (contract *TokenTemplet) UdcTransfer(udc smc.Hash, to smc.Address,
	value big.Int) (receiverUdcHash smc.Hash, restUdcHash smc.Hash, smcError smc.Error) {
	// Get Sender and check udc's owner
	//sender := contract.Sender()
	//udcInfo := sender.UdcInfo()
	//if  sender.Addr.String() != udcInfo.Owner.String(){
	//	return nil, nil, errors.New("Permission denied")
	//}

	// Check UDC amount
	// if bigNumber.Compare(udcInfo.value, value) < 0{
	//		return nil, nil, errors.New("Insufficient balance")
	//}

	// Calculate the rest
	// restValue := bigNumber.Sub(udcInfo.value, value)

	// Get token object
	// token := contract.Token()

	// Set this UDC status as "Expired", and check the return value
	// if err = token.ExpiredUDC(udc); err != nil{
	//		return
	//}

	// Create receiver UDC
	// receiverUdcHash, err = token.CreateUDC(to, value, udcInfo.MatureDate)
	// if err != nil{
	//		return
	//}

	// Create rest UDC
	// if bigNumber.Compare(restValue, bigNumber.Zero()) > 0 {
	//  restUdcHash, err = token.CreateUDC(sender.Addr, restValue, udcInfo.MatureDate)
	//}
	return
}

// UdcMature is used to mature an UDC (Unmatured Deposit Certificate)
// After that, this UDC will be matured, and transfer token to owner
// Only the owner of UDC is allowed to execute this function
// Parameters:
// 		udc: the UDC Hash that belongs to the sender
// Return Data:
//		err: Error or nil
func (contract *TokenTemplet) UdcMature(udc smc.Hash) (smcError smc.Error) {
	// Get Sender and check udc's owner
	//sender := contract.Sender()
	//udcInfo := sender.UdcInfo()
	//if  sender.Addr.String() != udcInfo.Owner.String(){
	//	return nil, nil, errors.New("Permission denied")
	//}

	// Check MatureDate, it should be before the current date
	// If not, return error

	// Get token object
	// token := contract.Token()

	// Set this UDC status as "matured", and check the return value
	// if err = token.MatureUDC(udc); err != nil{
	//		return
	//}

	// Get and Set sender's balance
	// balance, err := sender.Balance()
	// if err != nil {
	//  return err
	//}
	// newbalance := bigNumber.Add(balance, udcInfo.Value)
	// return sender.SetBalance(newbalance)
	return
}
