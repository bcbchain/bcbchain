// contract packages APIs of smart contracts

package contract

import (
	"math/big"

	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/contract/stubapi"
	"blockchain/abciapp_v1.0/smc"
	"blockchain/algorithm"
	"github.com/tendermint/go-crypto"
)

type Contract struct {
	Ctx *stubapi.InvokeContext
}

type Account struct {
	uaccount *stubapi.Account
}

func (base *Contract) Sender() *Account {
	return &Account{base.Ctx.Sender}
}

func (base *Contract) Owner() *Account {
	return &Account{base.Ctx.Owner}
}

//GetAccount creats Account structure with account address
func (base *Contract) GetAccount(to smc.Address) *Account {
	return &Account{&stubapi.Account{to, base.Ctx.TxState}}
}

//SetBalance sets account's balance
func (account *Account) SetBalance(tokenAddr smc.Address, value big.Int) smc.Error {
	return account.uaccount.SetBalance(tokenAddr, value)
}

//Balance gets account's balance
func (account *Account) Balance(tokenAddr smc.Address) big.Int {
	balance, _ := account.uaccount.BalanceOf(tokenAddr)
	return balance
}

//Address gets account's address
func (account *Account) Address() smc.Address {
	return account.uaccount.Addr
}

//CalcAddress calculates address of smart contract with contract name and other flags
func (base *Contract) CalcAddress(name string) smc.Address {

	tiContract, bcerr := base.Ctx.GetTokenIssueContract()
	if bcerr.ErrorCode != bcerrors.ErrCodeOK {
		return ""
	}

	return smc.Address(
		algorithm.CalcContractAddress(
			base.Ctx.TxState.GetChainID(),
			crypto.Address(base.Owner().Address()),
			name,
			tiContract.Version))
}

//CreateToken issues new token
func (base *Contract) CreateToken(addr smc.Address,
	name string,
	symbol string,
	totalsupply big.Int,
	addSupplyEnabled bool,
	burnEnabled bool,
	gasprice uint64) smc.Error {
	return base.Ctx.NewToken(addr, name, symbol, totalsupply, addSupplyEnabled, burnEnabled, gasprice)
}

func (base *Contract) SetGasBasePriceForToken(value uint64) smc.Error {
	return base.Ctx.SetGasBasePrice(value)
}

func (base *Contract) GasBasePrice() uint64 {
	return base.Ctx.GasBasePrice()
}

// check effectHeight with blockchain height
func (base *Contract) CheckEffectHeight(effectHeight uint64) smc.Error {
	return base.Ctx.CheckEffectHeight(effectHeight)
}

// delete lose effect strategy and add new strategy
func (base *Contract) UpdateRewardStrategy(strategy string, effectHeight uint64) smc.Error {
	return base.Ctx.UpdateRewardStrategy(strategy, effectHeight)
}

// delete lose effect strategy and add new strategy
func (base *Contract) CheckRewardStrategy(strategy string) smc.Error {
	return base.Ctx.CheckRewardStrategy(strategy)
}

type ValidatorMgr struct {
	*stubapi.InvokeContext
}

func (base *Contract) ValidatorMgr() *ValidatorMgr {
	return &ValidatorMgr{
		&stubapi.InvokeContext{
			Sender:   base.Ctx.Sender,
			Owner:    base.Ctx.Owner,
			TxState:  base.Ctx.TxState,
			GasLimit: base.Ctx.GasLimit,
			Proposer: base.Ctx.Proposer}}
}

func (vmc *ValidatorMgr) NewValidator(name string, pubKey smc.PubKey, rewardAddr smc.Address, power uint64) smc.Error {
	return vmc.AddValidator(name, pubKey, rewardAddr, power)
}

func (vmc *ValidatorMgr) Has(pubKey smc.PubKey) bool {
	return vmc.HasValidator(pubKey)
}

func (vmc *ValidatorMgr) SetPower(pubKey smc.PubKey, power uint64) smc.Error {
	return vmc.SetValidatorPower(pubKey, power)
}

func (vmc *ValidatorMgr) SetRewardAddr(pubKey smc.PubKey, rewardAddr smc.Address) smc.Error {
	return vmc.SetValidatorRewardAddr(pubKey, rewardAddr)
}

func (vmc *ValidatorMgr) CheckNameDuplicate(name string) smc.Error {
	return vmc.CheckValidatorName(name)
}

func (vmc *ValidatorMgr) CheckRewardAddress(rewardAddr string) (bcerr smc.Error) {
	bcerr.ErrorCode = bcerrors.ErrCodeOK
	if err := algorithm.CheckAddress(vmc.TxState.StateDB.GetChainID(), rewardAddr); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
	}
	return
}

type Token struct {
	*stubapi.Token
}

func (base *Contract) Token() *Token {
	return &Token{&stubapi.Token{base.Ctx.TxState.ContractAddress, base.Ctx}}
}

func (token *Token) GetTotalSupply() (big.Int, smc.Error) {
	return token.Ctx.TokenSupply()
}

func (token *Token) SetGasPrice(value uint64) smc.Error {
	return token.Ctx.SetGasPrice(value)
}

func (token *Token) SetSupply(value big.Int) smc.Error {
	return token.Ctx.SetTokenSupply(value, true, false)
}

func (token *Token) Burn(value big.Int) smc.Error {
	return token.Ctx.SetTokenSupply(value, false, true)
}

func (token *Token) SetNewOwner(owner smc.Address) smc.Error {
	return token.Ctx.SetTokenNewOwner(owner)
}

//CheckNameDuplicate checks token name, and return error if it's duplicated with an existing token
func (token *Token) CheckNameAndSybmol(name, symbol string) smc.Error {
	return token.Ctx.CheckNameAndSybmol(name, symbol)
}

//CheckSymbolDuplicate  checks token symbol, and return error if it's duplicated with an existing token
func (token *Token) CreateUDC(to smc.Address, value big.Int, matureDate string) (smc.Hash, smc.Error) {
	return token.Ctx.CreateUDC(to, value, matureDate)
}

//ExpiredUDC set an UDC as Expired
func (token *Token) ExpiredUDC(udc smc.Hash) smc.Error {
	return token.Ctx.ExpiredUDC(udc)
}

//MatureUDC sets an UDC as Matured
func (token *Token) MatureUDC(udc smc.Hash) smc.Error {
	return token.Ctx.MatureUDC(udc)
}
