package object

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
)

// Account account detail information
type Account struct {
	smc sdk.ISmartContract //指向智能合约API对象指针

	address types.Address //账户地址
	pubKey  types.PubKey  //账户公钥
}

var _ sdk.IAccount = (*Account)(nil)
var _ sdkimpl.IAcquireSMC = (*Account)(nil)

// SMC get smart contract object
func (a *Account) SMC() sdk.ISmartContract { return a.smc }

// SetSMC set smart contract object
func (a *Account) SetSMC(smc sdk.ISmartContract) { a.smc = smc }

// Address get address
func (a *Account) Address() types.Address { return a.address }

// PubKey get pubKey
func (a *Account) PubKey() types.PubKey { return a.pubKey }

// Balance get balance of current contract's token
func (a *Account) Balance() bn.Number {
	tokenAddr := a.smc.Message().Contract().Token()
	if tokenAddr == "" {
		return bn.N(0)
	}

	return a.balanceOfToken(tokenAddr)
}

// BalanceOfToken get balance of token with address
func (a *Account) BalanceOfToken(token types.Address) bn.Number {
	sdk.RequireAddress(token)

	return a.balanceOfToken(token)
}

// BalanceOfName get balance of token with name
func (a *Account) BalanceOfName(name string) bn.Number {
	token := a.smc.Helper().TokenHelper().TokenOfName(name)
	if token == nil {
		return bn.N(0)
	}

	return a.balanceOfToken(token.Address())
}

// BalanceOfSymbol get balance of token with symbol
func (a *Account) BalanceOfSymbol(symbol string) bn.Number {
	token := a.smc.Helper().TokenHelper().TokenOfSymbol(symbol)
	if token == nil {
		return bn.N(0)
	}

	return a.balanceOfToken(token.Address())
}

// Transfer transfer current contract's token
func (a *Account) Transfer(to types.Address, value bn.Number) {
	sdk.RequireAddress(to)

	tokenAddr := a.smc.Message().Contract().Token()
	sdk.Require(tokenAddr != "",
		types.ErrInvalidParameter, "Contract does not register any token")

	a.transferByToken(tokenAddr, to, value, "")
}

// TransferWithNote transfer current contract's token
func (a *Account) TransferWithNote(to types.Address, value bn.Number, note string) {
	sdk.RequireAddress(to)

	tokenAddr := a.smc.Message().Contract().Token()
	sdk.Require(tokenAddr != "",
		types.ErrInvalidParameter, "Contract does not register any token")

	a.transferByToken(tokenAddr, to, value, note)
}

// TransferEx transfer token with address，return error version
func (a *Account) TransferWithNoteEx(to types.Address, value bn.Number, note string) (err types.Error) {
	err.ErrorCode = types.CodeOK

	defer func(e *types.Error) {
		if err := recover(); err != nil {
			if _, ok := err.(types.Error); ok {
				*e = err.(types.Error)
			}
		}
	}(&err)

	sdk.RequireAddress(to)

	tokenAddr := a.smc.Message().Contract().Token()
	sdk.Require(tokenAddr != "",
		types.ErrInvalidParameter, "Contract does not register any token")

	a.transferByToken(tokenAddr, to, value, note)

	return
}

// TransferByToken transfer token with address
func (a *Account) TransferByToken(token types.Address, to types.Address, value bn.Number) {
	sdk.RequireAddress(to)
	sdk.RequireAddress(token)

	sdk.Require(a.smc.Helper().TokenHelper().TokenOfAddress(token) != nil,
		types.ErrInvalidParameter, "Token not found(address="+token+")")

	a.transferByToken(token, to, value, "")
}

// TransferByTokenWithNote transfer token with address
func (a *Account) TransferByTokenWithNote(token types.Address, to types.Address, value bn.Number, note string) {
	sdk.RequireAddress(to)
	sdk.RequireAddress(token)

	sdk.Require(a.smc.Helper().TokenHelper().TokenOfAddress(token) != nil,
		types.ErrInvalidParameter, "Token not found(address="+token+")")

	a.transferByToken(token, to, value, note)
}

// TransferByName transfer token with name
func (a *Account) TransferByName(name string, to types.Address, value bn.Number) {
	sdk.RequireAddress(to)

	token := a.smc.Helper().TokenHelper().TokenOfName(name)
	sdk.Require(token != nil,
		types.ErrInvalidParameter, "Token not found(name="+name+")")

	a.transferByToken(token.Address(), to, value, "")
}

// TransferByNameWithNote transfer token with name
func (a *Account) TransferByNameWithNote(name string, to types.Address, value bn.Number, note string) {
	sdk.RequireAddress(to)

	token := a.smc.Helper().TokenHelper().TokenOfName(name)
	sdk.Require(token != nil,
		types.ErrInvalidParameter, "Token not found(name="+name+")")

	a.transferByToken(token.Address(), to, value, note)
}

// TransferBySymbol transfer token with symbol
func (a *Account) TransferBySymbol(symbol string, to types.Address, value bn.Number) {
	sdk.RequireAddress(to)

	token := a.smc.Helper().TokenHelper().TokenOfSymbol(symbol)
	sdk.Require(token != nil,
		types.ErrInvalidParameter, "Token not found(symbol="+symbol+")")

	a.transferByToken(token.Address(), to, value, "")
}

// TransferBySymbolWithNote transfer token with symbol
func (a *Account) TransferBySymbolWithNote(symbol string, to types.Address, value bn.Number, note string) {
	sdk.RequireAddress(to)

	token := a.smc.Helper().TokenHelper().TokenOfSymbol(symbol)
	sdk.Require(token != nil,
		types.ErrInvalidParameter, "Token not found(symbol="+symbol+")")

	a.transferByToken(token.Address(), to, value, note)
}

// SetBalanceOfToken set balance with token address
func (a *Account) SetBalanceOfToken(tokenAddr types.Address, bal bn.Number) {
	acctInfo := std.AccountInfo{
		Address: tokenAddr,
		Balance: bal,
	}

	key := std.KeyOfAccountToken(a.address, tokenAddr)
	// don't cache account information, don't use McSet
	a.smc.(*sdkimpl.SmartContract).LlState().Set(key, &acctInfo)
}

// balanceOfToken get balance of token with address and without checkAddress
func (a *Account) balanceOfToken(token types.Address) bn.Number {
	key := std.KeyOfAccountToken(a.Address(), token)

	// don't cache account information, don't use McGetEx
	accInfo := a.smc.(*sdkimpl.SmartContract).LlState().GetEx(key, &std.AccountInfo{Balance: bn.N(0)}).(*std.AccountInfo)
	return accInfo.Balance
}

// transferByToken transfer token without checkAddress
func (a *Account) transferByToken(tokenAddr types.Address, to types.Address, value bn.Number, note string) {
	from := a.address

	// token isn't basic token and current contract's token
	if tokenAddr != a.smc.Helper().GenesisHelper().Token().Address() && tokenAddr != a.smc.Message().Contract().Token() {
		// invoke other contract transfer function
		var receipts []types.KVPair
		receipts, err := sdkimpl.TransferFunc(a.smc, tokenAddr, to, value, note)
		sdk.Require(err.ErrorCode == types.CodeOK,
			err.ErrorCode, err.ErrorDesc)

		a.smc.Message().(*Message).AppendOutput(receipts)
	} else {

		contract := a.smc.Helper().ContractHelper().ContractOfAddress(to)
		if contract != nil {
			to = contract.Account().Address()
		}

		sdk.Require(value.IsGreaterThanI(0),
			types.ErrInvalidParameter, "Value must greater than zero")

		sdk.Require(from != to,
			types.ErrInvalidParameter, "Cannot transfer to self")

		if from != a.smc.Message().Contract().Account().Address() {
			sdk.Require(tokenAddr == a.smc.Message().Contract().Token() &&
				from == a.smc.Message().Sender().Address(),
				types.ErrNoAuthorization, "")
		}

		ibcContract := a.smc.Helper().ContractHelper().ContractOfName("ibc")
		if ibcContract == nil || from != ibcContract.Account().Address() {
			sdk.Require(a.BalanceOfToken(tokenAddr).IsGE(value),
				types.ErrInsufficientBalance, "")
		}

		toAcct := a.smc.Helper().AccountHelper().AccountOf(to).(*Account)

		toAcct.SetBalanceOfToken(tokenAddr, toAcct.BalanceOfToken(tokenAddr).Add(value))
		a.SetBalanceOfToken(tokenAddr, a.BalanceOfToken(tokenAddr).Sub(value))

		toAcct.AddAccountTokenKey(std.KeyOfAccountToken(toAcct.address, tokenAddr))

		// fire event
		a.smc.Helper().ReceiptHelper().Emit(
			std.Transfer{
				Token: tokenAddr,
				From:  from,
				To:    toAcct.address,
				Value: value,
				Note:  note,
			},
		)
	}
}

// AccountOfContracts get contract address list owned by account
func (a *Account) accountOfContracts() []types.Address {
	key := std.KeyOfAccountContracts(a.address)
	return a.smc.(*sdkimpl.SmartContract).LlState().GetStrings(key)
}

func (a *Account) AddAccountTokenKey(keyOfAccountToken string) {
	key := std.KeyOfAccount(a.address)

	itemList := a.smc.(*sdkimpl.SmartContract).LlState().GetStrings(key)

	isExist := false
	for _, item := range itemList {
		if item == keyOfAccountToken {
			isExist = true
			break
		}
	}
	if isExist == false {
		itemList = append(itemList, keyOfAccountToken)
	}

	a.smc.(*sdkimpl.SmartContract).LlState().Set(key, itemList)
}
