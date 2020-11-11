package stubapi

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/prototype"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/statedb"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/types"
	"github.com/bcbchain/bclib/algorithm"
	"github.com/bcbchain/bclib/bignumber_v1.0"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
)

var logger log.Logger

func SetLogger(log log.Logger) {
	logger = log
}

func (account *Account) BalanceOf(tokenAddr smc.Address) (value big.Int, bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK

	value, err := account.TxState.GetBalance(account.Addr, tokenAddr)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	}
	return
}

func (account *Account) SetBalance(tokenAddr smc.Address, value big.Int) (bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK

	err := account.TxState.SetBalance(account.Addr, tokenAddr, value)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	}
	return bcerr
}

//modify by hcy
func (ctx *InvokeContext) SetGasBasePrice(value uint64) (bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	err := ctx.TxState.SetBaseGasPrice(value)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	}
	return
}

func (ctx *InvokeContext) GasBasePrice() uint64 {
	return ctx.TxState.GetBaseGasPrice()
}

func (ctx *InvokeContext) SetGasPrice(value uint64) (bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	token, err := ctx.TxState.GetToken(ctx.TxState.ContractAddress)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	} else if token == nil {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsNoToken
		return
	}

	newtoken := types.IssueToken{
		Address:          token.Address,
		Owner:            token.Owner,
		Version:          token.Version,
		Name:             token.Name,
		Symbol:           token.Symbol,
		TotalSupply:      token.TotalSupply,
		AddSupplyEnabled: token.AddSupplyEnabled,
		BurnEnabled:      token.BurnEnabled,
		GasPrice:         value,
	}

	err = ctx.TxState.SetToken(&newtoken)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	}
	return
}

func (ctx *InvokeContext) GetTokenIssueContract() (tiContract *types.Contract, bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	tiContract, err := ctx.TxState.StateDB.GetContract(ctx.TxState.ContractAddress)
	if err != nil {
		logger.Error("Failed to get token-issue contract")
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	} else if tiContract == nil {
		logger.Error("No token-issue contract")
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsNoGenesis
		return
	}
	return tiContract, bcerr
}

// NewToken issues token and create a smart contract for new token
// And recharge the token's supply to the owner's address
func (ctx *InvokeContext) NewToken(addr smc.Address,
	name string,
	symbol string,
	totalSupply big.Int,
	addSupplyEnabled bool,
	burnEnabled bool,
	gasprice uint64) (bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK

	tiContract, bcerr := ctx.GetTokenIssueContract()
	if bcerr.ErrorCode != bcerrors.ErrCodeOK {
		return
	}
	// Issue Token
	token := types.IssueToken{
		Name:             name,
		Address:          addr,
		Owner:            ctx.Sender.Addr,
		Version:          tiContract.Version,
		Symbol:           symbol,
		TotalSupply:      totalSupply,
		AddSupplyEnabled: addSupplyEnabled,
		BurnEnabled:      burnEnabled,
		GasPrice:         gasprice,
	}
	if logger == nil {
		fmt.Println("SetNewToken()", "token", token)
	} else {
		logger.Debug("SetNewToken()", "token", token)
	}
	if err := ctx.TxState.SetToken(&token); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	// get token-templet contract and set contents into new token contract
	tempContractAddr := algorithm.CalcContractAddress(ctx.TxState.GetChainID(),
		crypto.Address(tiContract.Owner),
		prototype.TokenTemplet,
		token.Version)

	tempContract, err := ctx.TxState.StateDB.GetContract(smc.Address(tempContractAddr))
	if err != nil {
		logger.Error("Failed to get contract", "contract address", tempContractAddr)
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	} else if tempContract == nil {
		logger.Error("Failed to get contract",
			"invalid contract address", tempContractAddr)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidAddr
		return
	}
	// Create token Contract
	logger.Trace("SetNewToken()", "Contract", tempContractAddr, "Methods", tempContractAddr)
	contract := types.Contract{
		Address:      addr,
		Owner:        ctx.Sender.Addr,
		Name:         "token-templet-" + name,
		Version:      tempContract.Version,
		CodeHash:     tempContract.CodeHash,
		Methods:      tempContract.Methods[:],
		EffectHeight: tempContract.EffectHeight,
		LoseHeight:   tempContract.LoseHeight,
	}
	if err = ctx.TxState.SetTokenContract(&contract); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	// Using new TX to recharge the balance of owner's account,
	newtx := statedb.TxState{
		StateDB:         ctx.TxState.StateDB,
		ContractAddress: contract.Address,
		SenderAddress:   ctx.TxState.SenderAddress,
		Tx:              ctx.TxState.Tx,
	}
	logger.Debug("SetBalance for Owner",
		"owner", ctx.Sender.Addr,
		"balance", totalSupply)

	err = newtx.SetBalance(ctx.Sender.Addr, newtx.ContractAddress, totalSupply)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	}
	return
}

func (ctx *InvokeContext) GetToken(addr smc.Address) (token *types.IssueToken, bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	token, err := ctx.TxState.GetToken(addr)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	} else if token == nil {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidAddr
	}
	return
}

func (ctx *InvokeContext) GetTokenAddressByName(name string) (addr smc.Address, bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	addr, err := ctx.TxState.GetTokenAddrByName(name)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	}
	return
}

func (ctx *InvokeContext) GetTokenAddressBySymbol(symbol string) (addr smc.Address, bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	addr, err := ctx.TxState.GetTokenAddrBySymbol(symbol)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	}
	return
}

func (ctx *InvokeContext) TokenSupply() (value big.Int, bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	contract := ctx.TxState.ContractAddress
	token, err := ctx.TxState.GetToken(contract)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	} else if token == nil {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsNoToken
		return
	}
	value = token.TotalSupply
	return
}

func (ctx *InvokeContext) SetTokenSupply(value big.Int, isAddSupply bool, isBurn bool) (bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	contract := ctx.TxState.ContractAddress
	token, err := ctx.TxState.GetToken(contract)
	if err != nil {
		logger.Error("SetTokenSupply()", "Get Token failed, error", err)
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	} else if token == nil {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsNoToken
		return
	}

	if isAddSupply == true && token.AddSupplyEnabled == false {
		logger.Error("SetTokenSupply(), unsupport adding supply")
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsUnsupportAddSupply
		return
	} else if isBurn == true && token.BurnEnabled == false {
		logger.Error("SetTokenSupply(), unsupport burning")
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsUnsupportBurn
		return
	}
	newToken := types.IssueToken{
		Address:          token.Address,
		Owner:            token.Owner,
		Version:          token.Version,
		Name:             token.Name,
		Symbol:           token.Symbol,
		TotalSupply:      value,
		AddSupplyEnabled: token.AddSupplyEnabled,
		BurnEnabled:      token.BurnEnabled,
		GasPrice:         token.GasPrice,
	}
	logger.Debug("SetTokenSupply()", "newToken", newToken)
	err = ctx.TxState.SetToken(&newToken)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	}
	return
}

// SetTokenNewOwner sets a new owner for the specified token
func (ctx *InvokeContext) SetTokenNewOwner(owner smc.Address) (bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	contract := ctx.TxState.ContractAddress
	token, err := ctx.TxState.GetToken(contract)
	if err != nil {
		logger.Error("SetTokenNewOwner()", "Get Token smc.Error", err)
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	} else if token == nil {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsNoToken
		return
	}

	// Set token with new Owner
	oldOwner := token.Owner
	newToken := types.IssueToken{
		Address:          token.Address,
		Owner:            owner,
		Version:          token.Version,
		Name:             token.Name,
		Symbol:           token.Symbol,
		TotalSupply:      token.TotalSupply,
		AddSupplyEnabled: token.AddSupplyEnabled,
		BurnEnabled:      token.BurnEnabled,
		GasPrice:         token.GasPrice,
	}
	logger.Debug("SetTokenNewOwner()", "newToken", newToken)
	if err = ctx.TxState.SetToken(&newToken); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		return
	}
	// Set token's contract with new owner
	oldContract, err := ctx.TxState.StateDB.GetContract(contract)
	if err != nil {
		logger.Error("SetTokenNewOwner()", "Get Contract smc.Error", err)
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	} else if oldContract == nil {
		bcerr.ErrorCode = bcerrors.ErrCodeDockerNotFindContract
		return
	}

	newContract := types.Contract{
		Address:      oldContract.Address,
		Owner:        owner,
		Name:         oldContract.Name,
		Version:      oldContract.Version,
		CodeHash:     oldContract.CodeHash,
		Methods:      oldContract.Methods,
		EffectHeight: oldContract.EffectHeight,
		LoseHeight:   oldContract.LoseHeight,
	}
	if err = ctx.TxState.SetTokenContract(&newContract); err != nil {
		logger.Error("SetTokenNewOwner()", "Set Token Contract smc.Error", err)
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}

	// Delete contract from old owner's list
	err = ctx.TxState.DeleteContractAddr(oldOwner, contract)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	}
	return
}

func calcFee(gas, gaslimit, gasprice uint64) (fee uint64, bcerr bcerrors.BCError) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	// checking gaslimit, if it's smaller than gas required
	// return err and fee deduction with gaslimit
	var realgas uint64
	if gaslimit < gas {
		realgas = gaslimit
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidGasLimit
		logger.Debug("CheckAndPayForGas(), GasLimit is less than the required gas",
			"gaslimit", gaslimit, "gas", gas)
	} else {
		realgas = gas
	}
	// Calculate value of Gas
	if gasprice == 0 {
		logger.Error("CheckAndPayForGas", "token.GasPrice", gasprice)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidGasPrice
		return
	}
	// Checking Fee
	fee, overflow := safeMul(realgas, gasprice)
	if overflow {
		logger.Error("Fee is out of range")
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidFee
		return
	}

	logger.Debug("CheckAndPayForGas()", "gas", realgas, "fee", fee)
	return
}

// CheckAndPayForGas Checks and pay for gas from sender, it's payed with TokenBasic contract address.
// The parameter "contract" is the tokenbasic contract address.
func (ctx *InvokeContext) CheckAndPayForGas(sender, proposer *Account, rewarder *Account,
	gas, gaslimit uint64) (gasused, gasprice uint64, rewardValues map[crypto.Address]uint64, bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	// Construct token basic account of sender
	logger.Debug("CheckAndPayForGas()",
		"sender", sender.Addr,
		"gas", gas, "gaslimit", gaslimit)

	// check Proposer's rewarderAddr
	if proposer != nil {
		bcerr = ctx.CheckValidatorRewardAddr(proposer.Addr, rewarder.Addr)
		if bcerr.ErrorCode != bcerrors.ErrCodeOK {
			return
		}
	}

	// Get Sender's balance of account of tokenbasic
	// Get Genesis Token
	tokenbasic, err := sender.TxState.GetGenesisToken()
	if err != nil {
		logger.Error("Failed to get sender's tokenbasic contract")
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	} else if tokenbasic == nil {
		logger.Error("No Genesis")
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsNoToken
		bcerr.ErrorDesc = err.Error()
		return
	}

	genTxState := ctx.TxState.StateDB.NewTxState(tokenbasic.Address, sender.Addr)
	// Constructure sender's tokenbasic account
	senderBasic := Account{
		Addr:    sender.Addr,
		TxState: genTxState,
	}
	// Get tokenbasic balance
	senderBalance, bcerr := senderBasic.BalanceOf(tokenbasic.Address)
	if bcerr.ErrorCode != bcerrors.ErrCodeOK {
		logger.Error("Failed to get account's tokenbasic balance")
		return
	}
	logger.Debug("CheckAndPayForGas()",
		"Sender's basic token", tokenbasic.Address,
		"Sender's basic Address", senderBasic.Addr,
		"balance", senderBalance)

	//check sender's contract
	contract, err := ctx.TxState.StateDB.GetContract(sender.TxState.ContractAddress)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	} else if contract == nil {
		logger.Error("CheckAndPayForGas()",
			"Sender's txstate contract", sender.TxState.ContractAddress)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidAddr
		return
	}
	// Get token
	token, err := ctx.TxState.GetToken(sender.TxState.ContractAddress)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	} else if token == nil {
		// For issue-token and system contracts, it doesn't have a token,
		// uses basic token's gasprice
		logger.Debug("For issue-token and system contract, uses basic token's gasprice")
		token = tokenbasic
		err = nil
	}

	fee, bcerr := calcFee(gas, gaslimit, token.GasPrice)
	if bcerr.ErrorCode != bcerrors.ErrCodeOK &&
		bcerr.ErrorCode != bcerrors.ErrCodeInterContractsInvalidGasLimit {
		// If user specified a smaller gaslimit, charges and sends error out
		return
	}
	value := big.NewInt(0)
	*value = bignumber.UintToBigInt(fee)

	newSenderBalance := big.NewInt(0)
	if bignumber.Compare(senderBalance, *value) < 0 {
		logger.Error("insufficient balance")
		// Set this err and return it out after pay if that's required
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInsufficientBalance
		newSenderBalance = newSenderBalance.Set(big.NewInt(0))
		value = value.Set(&senderBalance)
	} else {
		temp := bignumber.Sub(senderBalance, *value)
		logger.Debug("CheckAndPayForGas()", "value", value)
		newSenderBalance = newSenderBalance.Set(&temp)
	}
	logger.Debug("CheckAndPayForGas()", "Sender's new balance", newSenderBalance)

	// Set sender balance
	// Use a new bcError to avoid rewrite error code by calcFee()
	inerr := senderBasic.SetBalance(tokenbasic.Address, *newSenderBalance)
	if inerr.ErrorCode != bcerrors.ErrCodeOK {
		logger.Error("Failed to set sender's balance", "smc.Error", inerr)
		return 0, 0, nil, inerr
	}

	// pay for gas
	if proposer != nil {
		// uses new genTxState for rewarder
		proposer.TxState = genTxState

		var strategy types.RewardStrategy
		// Use a new bcError to avoid rewrite error code by calcFee()
		strategy, giRewdErr := ctx.GetStrategy()
		if giRewdErr.ErrorCode != bcerrors.ErrCodeOK {
			logger.Error("Failed to get strategy", "smc.Error", err)
			return 0, 0, nil, giRewdErr
		}

		// calculate reward's values
		rewardValues, giRewdErr = ctx.CalcdRewards(strategy, *value, *rewarder)
		if giRewdErr.ErrorCode != bcerrors.ErrCodeOK {
			logger.Error("Failed to calc rewards", "smc.Error", err)
			return 0, 0, nil, giRewdErr
		}

		// set balance
		giRewdErr = setRewardBalances(proposer.TxState, rewardValues)
		if giRewdErr.ErrorCode != bcerrors.ErrCodeOK {
			logger.Error("Failed to set reward's balance", "smc.Error", err)
			return 0, 0, nil, giRewdErr
		}

		// Calculate GasUsed: gasused = fee/gasprice
		gasused = fee / token.GasPrice
		gasprice = token.GasPrice
	}

	// Commit Tx to save Fee into block buffer
	genTxState.CommitTx()
	//if transID != 0 {
	//	statedbhelper.CommitTx2V1(transID, txBuffer)
	//}

	logger.Debug("CheckAndPayForGas()", "gasused", gasused, "gasprice", gasprice)

	return
}

func (ctx *InvokeContext) CreateUDC(to smc.Address,
	value big.Int,
	matureDate string) (hash smc.Hash, bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	// Get Sender's Nonce
	nonce, err := ctx.TxState.GetUDCNonce()
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	// Calculate UDC Hash depends on parameters
	hash = algorithm.CalcUdcHash(nonce+1,
		crypto.Address(ctx.TxState.ContractAddress),
		crypto.Address(ctx.Sender.Addr),
		value,
		matureDate)

	// UDCOrder structure
	udc := &types.UDCOrder{
		UDCState:     UDCState_Unmatured,
		UDCHash:      hash,
		Nonce:        nonce,
		ContractAddr: to,
		Owner:        to,
		Value:        value,
		MatureDate:   matureDate,
	}
	// Set UDCOrder
	err = ctx.TxState.SetUDCOrder(udc)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	}
	return
}

func (ctx *InvokeContext) ExpiredUDC(udc smc.Hash) (bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	oldudc, err := ctx.TxState.GetUDCOrder(udc)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	if oldudc == nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = "" //ToDo: define new error message and code for this
		return
	}

	newudc := &types.UDCOrder{
		UDCState:     UDCState_Expired,
		UDCHash:      oldudc.UDCHash,
		Nonce:        oldudc.Nonce,
		ContractAddr: oldudc.ContractAddr,
		Owner:        oldudc.Owner,
		Value:        oldudc.Value,
		MatureDate:   oldudc.MatureDate,
	}

	err = ctx.TxState.SetUDCOrder(newudc)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	}
	return
}

func (ctx *InvokeContext) MatureUDC(udc smc.Hash) (bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	oldudc, err := ctx.TxState.GetUDCOrder(udc)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	} else if oldudc == nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = "" //ToDo: define new error message and code for this
		return
	}

	newudc := &types.UDCOrder{
		UDCState:     UDCState_Matured,
		UDCHash:      oldudc.UDCHash,
		Nonce:        oldudc.Nonce,
		ContractAddr: oldudc.ContractAddr,
		Owner:        oldudc.Owner,
		Value:        oldudc.Value,
		MatureDate:   oldudc.MatureDate,
	}
	if err = ctx.TxState.SetUDCOrder(newudc); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	}
	return
}

// safeadd
func safeAdd(a, b uint64) (uint64, bool) {
	if a > math.MaxUint64-b {
		return math.MaxUint64, true
	}
	return a + b, false
}

func safeMul(a uint64, b uint64) (uint64, bool) {
	if a == 0 || b == 0 {
		return 0, false
	}
	if a == 1 {
		return b, false
	}
	if b == 1 {
		return a, false
	}
	if a == math.MaxUint64 || b == math.MaxUint64 {
		return 0, true
	}
	c := a * b
	return c, c/b != a
}

// Get all validators and check each power to avoid it's over 1/3 of all validators'
func (ctx *InvokeContext) checkValidatorsPower(pubKey smc.PubKey, power uint64) (bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	validators, err := ctx.TxState.StateDB.GetAllValidators()
	if err != nil {
		logger.Error("Get validators failed", "smc.Error", err)
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	} else if validators == nil {
		logger.Error("Get validators failed, no validators")
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsNoValidators
		return
	}
	crptPubKey := crypto.PubKeyEd25519FromBytes(pubKey)
	addr := crptPubKey.Address(statedbhelper.GetChainID())
	// calculate the total power of all validators
	var totalPower, maxPower uint64
	var has bool = false

	for _, validator := range validators {
		if !has && validator.NodeAddr == addr {
			//validator, reset its power
			has = true
			validator.Power = power
		}

		newtotalPower, overflow := safeAdd(totalPower, validator.Power)
		if overflow {
			logger.Error("Total power is overflow")
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidPower
			return
		}
		if validator.Power > maxPower {
			maxPower = validator.Power
		}
		totalPower = newtotalPower
	}

	// The validator is not existing, adding to validators' list
	if !has {
		totalPower = totalPower + power
		if power > maxPower {
			maxPower = power
		}
	}

	// If the maxPower is equal to or over 1/3 totalPower, return smc.Error to avoid this happens
	if maxPower >= totalPower*1/3 {
		logger.Error("Invalid operation, one validator's power is over 1/3",
			"totalPower", totalPower, "max power", maxPower)
		if maxPower == power {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidPower
			return
		} else {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidPower
			return
		}
	}
	return
}

func (ctx *InvokeContext) AddValidator(name string,
	pubKey smc.PubKey,
	rewardAddr smc.Address,
	power uint64) (bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	if len(name) == 0 ||
		len(pubKey) != smc.PUBKEY_LEN ||
		power == 0 {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		return
	}

	if bcerr = ctx.checkValidatorsPower(pubKey, power); bcerr.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	crptPubKey := crypto.PubKeyEd25519FromBytes(pubKey)

	validator := types.Validator{Name: name,
		NodePubKey: crptPubKey.Bytes(),
		NodeAddr:   crptPubKey.Address(statedbhelper.GetChainID()),
		RewardAddr: rewardAddr,
		Power:      power}
	logger.Debug("AddValidator()",
		"NodePubKey", hex.EncodeToString(crptPubKey.Bytes()),
		"NodeAddr", crptPubKey.Address(statedbhelper.GetChainID()),
		"RewardAddr", rewardAddr,
		"Power", power)
	if err := ctx.TxState.SetValidator(&validator); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	}

	return
}

func (ctx *InvokeContext) GetValidatorUpdate(pubKey smc.PubKey) smc.Address {

	crptPubKey := crypto.PubKeyEd25519FromBytes(pubKey)
	return crptPubKey.Address(statedbhelper.GetChainID())
}

func (ctx *InvokeContext) HasValidator(pubKey smc.PubKey) bool {

	crptPubKey := crypto.PubKeyEd25519FromBytes(pubKey)
	nodeAddr := crptPubKey.Address(statedbhelper.GetChainID())

	if validator, _ := ctx.TxState.StateDB.GetValidator(nodeAddr); validator != nil && validator.Power != 0 {
		return true
	}
	return false
}

func (ctx *InvokeContext) CheckValidatorName(name string) (bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	pubkeys, err := ctx.TxState.GetAllValidatorPubKeys()
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	} else if pubkeys == nil {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsNoValidators
		return
	}

	for _, pubkey := range pubkeys {
		hexPubKey, _ := hex.DecodeString(pubkey)
		crptPubKey := crypto.PubKeyEd25519FromBytes(hexPubKey)
		nodeAddr := crptPubKey.Address(statedbhelper.GetChainID())

		validator, _ := ctx.TxState.StateDB.GetValidator(nodeAddr)
		if validator != nil && validator.Name == name && validator.Power != 0 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsDupName
			return
		}
	}
	return
}

func (ctx *InvokeContext) SetValidatorPower(pubKey smc.PubKey, power uint64) (bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	if bcerr = ctx.checkValidatorsPower(pubKey, power); bcerr.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	crptPubKey := crypto.PubKeyEd25519FromBytes(pubKey)
	nodeAddr := crptPubKey.Address(statedbhelper.GetChainID())

	validator, err := ctx.TxState.StateDB.GetValidator(nodeAddr)
	if err != nil || validator == nil {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsNoValidators
		return
	}

	validator.Power = power
	if err := ctx.TxState.SetValidator(validator); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	}
	return
}

func (ctx *InvokeContext) SetValidatorRewardAddr(pubKey smc.PubKey, rewardAddr smc.Address) (bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	crptPubKey := crypto.PubKeyEd25519FromBytes(pubKey)
	nodeAddr := crptPubKey.Address(statedbhelper.GetChainID())

	validator, err := ctx.TxState.StateDB.GetValidator(nodeAddr)
	if err != nil || validator == nil {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsNoValidators
		return
	}

	chainID := ctx.TxState.StateDB.GetChainID()
	if err := algorithm.CheckAddress(chainID, rewardAddr); err != nil {
		logger.Error("Invalid validator reward address", "addr", rewardAddr, "error", err)
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	validator.RewardAddr = rewardAddr
	if err := ctx.TxState.SetValidator(validator); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	}
	return
}

func (ctx *InvokeContext) CheckValidatorRewardAddr(validatorAddr smc.Address, rewardAddr smc.Address) (bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	validator, err := ctx.TxState.StateDB.GetValidator(validatorAddr)
	if err != nil || validator == nil {
		logger.Error("CheckValidatorRewardAddr()",
			"validator addr", validatorAddr, "smc.Error", err)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsNoValidators
		return
	}

	if validator.RewardAddr != rewardAddr {
		logger.Error("CheckValidatorRewardAddr()",
			"execpted rewarder addr", validator.RewardAddr,
			"delieved by proposal", rewardAddr)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidRewarderAddr
		return
	}

	return
}

// 加载有效的奖励策略
func (ctx *InvokeContext) GetStrategy() (resultStrategy types.RewardStrategy, bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	strategys, err := ctx.TxState.GetStrategys()
	if err != nil {
		logger.Error("GetStrategys()", "smc.Error", err)
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	} else if len(strategys) == 0 {
		logger.Error("Strategy is empty", "len", len(strategys))
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsNoStrategys
		return
	}
	logger.Debug("GetStrategy()", "Strategys", strategys)

	// get blockchain height
	appState, err := ctx.TxState.StateDB.GetWorldAppState()
	if err != nil {
		logger.Error("GetWorldAppState()", "Error", err)
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	height := uint64(appState.BlockHeight + 1)

	for _, strategy := range strategys {
		if strategy.EffectHeight <= height {
			if resultStrategy.Strategy != nil && resultStrategy.EffectHeight >= strategy.EffectHeight {
				continue
			}
			resultStrategy = strategy
		}
	}

	return
}

// 按照奖励分配策略
func (ctx *InvokeContext) CalcdRewards(strategy types.RewardStrategy, value big.Int, rewarder Account) (rewardValues map[crypto.Address]uint64, bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	logger.Debug("CalcRewards()", "current strategy", strategy.Strategy)

	var sumValue uint64
	rewardValues = make(map[crypto.Address]uint64)
	for _, item := range strategy.Strategy {
		rewardRatio := strings.Replace(item.RewardPercent, ".", "", 1)
		//rewardRatio := item.Reward[:len(item.Reward)-1]
		rewardRatioI, err := strconv.Atoi(rewardRatio)
		if err != nil {
			logger.Error("strconv.Atoi()", "smc.Error", err)
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}

		rewardValue, overflow := safeMul(value.Uint64(), uint64(rewardRatioI))
		if overflow {
			logger.Error("safeMul() overflow", "value", value, "rewardRatio", rewardRatioI)
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}

		divisorStr := "1"
		index := 0
		// 在1后面加上小数位数加2个数的0，因为有小数点所以最后+1
		for index < (len(item.RewardPercent) - strings.Index(item.RewardPercent, ".") + 1) {
			divisorStr += "0"
			index++
		}
		divisor, _ := strconv.Atoi(divisorStr)

		rewardValue = uint64(math.Floor(float64(rewardValue) / float64(divisor)))
		if item.Name == "validators" {
			rewardValues[rewarder.Addr] = rewardValue
		} else {
			rewardValues[item.Address] = rewardValue
		}
		sumValue += rewardValue
	}

	logger.Debug("rewardValues", rewardValues)
	// 最后将差额补齐或报错
	if sumValue < value.Uint64() {
		rewardValues[rewarder.Addr] = rewardValues[rewarder.Addr] + (value.Uint64() - sumValue)
	} else if sumValue > value.Uint64() {
		logger.Error("The last sumValue bigger than value", "sumValue", sumValue, "value", value)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		return
	}

	return
}

// 将奖励结果记录到本币状态数据库
func setRewardBalances(txState *statedb.TxState, rewardValues map[crypto.Address]uint64) (bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	for key, value := range rewardValues {
		proBalance, err := txState.GetBalance(key, txState.ContractAddress)
		if err != nil {
			logger.Error("Failed to get rewarder's balances", "smc.Error", err)
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		}

		logger.Debug("SetBalance()",
			"Address", key,
			"proBalance", proBalance.Uint64(),
			"Contract", txState.ContractAddress)

		newBalance := bignumber.Add(proBalance, bignumber.UintToBigInt(value))
		err = txState.SetBalance(key, txState.ContractAddress, newBalance)
		if err != nil {
			logger.Error("Failed to set rewarder's balance", "smc.Error", err)
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		}

		logger.Debug("SetBalance()", "Rewarder New Balance", newBalance)
	}

	return
}

// 检查effectHeight的有效性
func (ctx *InvokeContext) CheckEffectHeight(effectHeight uint64) (bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	// effectHeight必须大于目前区块高度
	appState, err := ctx.TxState.StateDB.GetWorldAppState()
	if err != nil {
		logger.Error("GetWorldAppState()", "smc.Error", err)
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	if effectHeight <= uint64(appState.BlockHeight) {
		logger.Error("CheckEffectHeight()", "incoming effectHeight", effectHeight, "github.com/bcbchain/bcbchain height", appState.BlockHeight)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		return
	}

	// 传入的effectHeight必须大于已保存的所有策略的effectHeight
	rewardStrategys, err := ctx.TxState.GetStrategys()
	if err != nil {
		logger.Error("GetStrategys()", "smc.Error", err)
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}

	for _, item := range rewardStrategys {
		if effectHeight <= item.EffectHeight {
			logger.Error("Check effect height failed", "incoming effectHeight", effectHeight, "storage effectHeight", item.EffectHeight)
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
	}

	return
}

// 检查奖励策略
func (ctx *InvokeContext) CheckRewardStrategy(strategy string) (bcerr smc.Error) {
	logger.Debug("start CheckRewardStrategy...")
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	var rwdStrategy types.RewardStrategy

	if err := json.Unmarshal([]byte(strategy), &rwdStrategy); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}

	var percent float64
	haveNameOfValidators := false
	// check each
	chainID := ctx.TxState.StateDB.GetChainID()
	for _, st := range rwdStrategy.Strategy {

		//Check Name length
		if len(st.Name) > smc.Max_Name_Len {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsNameTooLong
			return
		} else if len(st.Name) == 0 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsEmptyName
			return
		}
		// Check Percent format
		if strings.Contains(st.RewardPercent, ".") {
			index := strings.IndexByte(st.RewardPercent, '.')
			sub := []byte(st.RewardPercent)[index+1:]
			if len(sub) > 2 { // Two digits would be supported behind the decimal point
				bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidPercent
				return
			}
		}
		nodePerc, _ := strconv.ParseFloat(st.RewardPercent, 64)
		percent = percent + nodePerc
		if percent > 100 || nodePerc < 0 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidPercent
			return
		}
		// Check Address, check name with "validators", for validators, we don't care about its address
		if st.Name == "validators" {
			haveNameOfValidators = true
		} else if err := algorithm.CheckAddress(chainID, st.Address); err != nil {
			logger.Error("Invalid reward address", "addr", st.Address, "error", err)
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		}
	}
	if percent != 100.00 {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidPercent
		return
	}

	if haveNameOfValidators == false {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsLoseNameOfValidators
	}

	return
}

// 更新奖励策略到策略列表
func (ctx *InvokeContext) UpdateRewardStrategy(strategy string, effectHeight uint64) (bcerr smc.Error) {
	// 读取appState，从中获取当前区块高度
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	logger.Debug("start UpdateRewardStrategy...")

	appState, err := ctx.TxState.StateDB.GetWorldAppState()
	if err != nil {
		logger.Error("GetWorldAppState()", "smc.Error", err)
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}

	// 读取已保存策略列表
	rewardStrategys, err := ctx.TxState.GetStrategys()
	if err != nil {
		logger.Error("GetStrategys()", "smc.Error", err)
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	logger.Debug("UpdateRewardStrategy", "rewardStrategys", rewardStrategys)

	// 默认保留一个effectHeight小于等于当前区块高度的策略，删除其余小于区块高度的策略
	var resultStrategys []types.RewardStrategy
	var preStrategy types.RewardStrategy
	for _, rewardStrategy := range rewardStrategys {
		logger.Debug("UpdateRewardStrategy", "effectHeight", rewardStrategy.EffectHeight, "blockHeight", appState.BlockHeight)
		if rewardStrategy.EffectHeight <= uint64(appState.BlockHeight) {
			if preStrategy.EffectHeight >= rewardStrategy.EffectHeight {
				continue
			}
			preStrategy = rewardStrategy
		} else {
			if preStrategy.EffectHeight != 0 {
				logger.Debug("UpdateRewardStrategy", "preStrategy", preStrategy)
				resultStrategys = append(resultStrategys, preStrategy)
				preStrategy.EffectHeight = 0
			}
			resultStrategys = append(resultStrategys, rewardStrategy)
		}
	}

	if len(resultStrategys) == 0 && preStrategy.EffectHeight != 0 {
		logger.Debug("UpdateRewardStrategy", "preStrategy", preStrategy)
		resultStrategys = append(resultStrategys, preStrategy)
		preStrategy.EffectHeight = 0
	}

	// 将新策略添加到策略列表并保存
	var newRewarder rewardStrategy
	err = json.Unmarshal([]byte(strategy), &newRewarder)

	rewardStrategy := types.RewardStrategy{Strategy: newRewarder.RewardStrategy, EffectHeight: effectHeight}
	resultStrategys = append(resultStrategys, rewardStrategy)
	logger.Debug("UpdateRewardStrategy", "resultStrategys", resultStrategys)
	err = ctx.TxState.SetStrategys(resultStrategys)
	if err != nil {
		logger.Error("SetStategys()", "smc.Error", err)
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}

	return
}

func (ctx *InvokeContext) CheckNameAndSybmol(name, symbol string) smc.Error {
	addr1, bcerr1 := ctx.GetTokenAddressByName(name)
	addr2, bcerr2 := ctx.GetTokenAddressBySymbol(symbol)
	if bcerr1.ErrorCode != bcerrors.ErrCodeOK {
		logger.Error("Failed to get token with token name", "name", name, "error", bcerr1.Error())
		return bcerr1
	}
	if bcerr2.ErrorCode != bcerrors.ErrCodeOK {
		logger.Error("Failed to get token with token symbol", "symbol", symbol, "error", bcerr1.Error())
		return bcerr2
	}

	// No any token using the name & symbol
	if addr1 == "" && addr2 == "" {
		return bcerr1
	}
	// One token is using both the name and symbol
	if addr1 == addr2 {
		// The owner of token can dup-create which one was created with TotalSuppy=0.
		t, bcerr := ctx.GetToken(addr1)
		if bcerr.ErrorCode == bcerrors.ErrCodeOK && t != nil {
			if bignumber.Compare(t.TotalSupply, bignumber.Zero()) == 0 &&
				t.Owner == ctx.Sender.Addr {
				return bcerr
			}
		}

		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsDupName
		return bcerr
	}
	// Duplicating with one existing contract
	if addr1 != "" {
		bcerr1.ErrorCode = bcerrors.ErrCodeInterContractsDupName
	} else if addr2 != "" {
		bcerr1.ErrorCode = bcerrors.ErrCodeInterContractsDupSymbol
	}
	return bcerr1
}

// NewUnitedToken issues united-token and create a smart contract for new token
// And recharge the token's supply to the owner's address
func (ctx *InvokeContext) NewUnitedToken(addr smc.Address,
	name string,
	symbol string,
	totalSupply big.Int,
	addSupplyEnabled bool,
	burnEnabled bool,
	gasprice uint64) (bcerr smc.Error) {

	bcerr.ErrorCode = bcerrors.ErrCodeOK

	tiContract, err := ctx.TxState.StateDB.GetContract(ctx.TxState.ContractAddress)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	// Issue Token
	token := types.IssueToken{
		Name:             name,
		Address:          addr,
		Owner:            ctx.Sender.Addr,
		Version:          tiContract.Version,
		Symbol:           symbol,
		TotalSupply:      totalSupply,
		AddSupplyEnabled: addSupplyEnabled,
		BurnEnabled:      burnEnabled,
		GasPrice:         gasprice,
	}

	if err = ctx.TxState.SetToken(&token); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}

	// Create token Contract
	contract := types.Contract{
		Address:      addr,
		Owner:        ctx.Sender.Addr,
		Name:         "ut-templet-" + name,
		Version:      tiContract.Version,
		CodeHash:     tiContract.CodeHash,
		Methods:      tiContract.Methods[:],
		EffectHeight: tiContract.EffectHeight,
		LoseHeight:   tiContract.LoseHeight,
	}
	if err = ctx.TxState.SetTokenContract(&contract); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	// Using new TX to recharge the balance of owner's account,
	newtx := statedb.TxState{
		StateDB:         ctx.TxState.StateDB,
		ContractAddress: token.Address,
		SenderAddress:   ctx.TxState.SenderAddress,
		Tx:              ctx.TxState.Tx,
	}

	err = newtx.SetBalance(ctx.Sender.Addr, newtx.ContractAddress, totalSupply)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	}
	return
}

//ConvertPrototype2ID calculates MethodID with prototype
func ConvertPrototype2ID(prototype string) uint32 {

	var id uint32
	bytesBuffer := bytes.NewBuffer(algorithm.CalcMethodId(prototype))
	binary.Read(bytesBuffer, binary.BigEndian, &id)
	return id
}
