package object

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
	"fmt"
)

// Token token detail information
type Token struct {
	smc sdk.ISmartContract //指向智能合约API对象指针

	tk std.Token
}

var _ sdk.IToken = (*Token)(nil)
var _ sdkimpl.IAcquireSMC = (*Token)(nil)

// SMC get smart contract object
func (t *Token) SMC() sdk.ISmartContract { return t.smc }

// SetSMC set smart contract object
func (t *Token) SetSMC(smc sdk.ISmartContract) { t.smc = smc }

// Define the minimum of total supply as one Token (1,000,000,000 cong),
const minTotalSupply = 1000000000

// Define the maximum of gas price as one Token (1,000,000,000 cong),
// GasBasePrice either.
const maxGasPrice = 1000000000

// Address get token's address
func (t *Token) Address() types.Address { return t.tk.Address }

// Owner get token's owner
func (t *Token) Owner() sdk.IAccount {
	return t.smc.Helper().AccountHelper().AccountOf(t.tk.Owner)
}

// Name get token's name
func (t *Token) Name() string { return t.tk.Name }

// Symbol get token's symbol
func (t *Token) Symbol() string { return t.tk.Symbol }

// TotalSupply get token's totalSupply
func (t *Token) TotalSupply() bn.Number { return t.tk.TotalSupply }

// AddSupplyEnabled get token's addSupplyEnabled
func (t *Token) AddSupplyEnabled() bool { return t.tk.AddSupplyEnabled }

// BurnEnabled get token's burnEnabled
func (t *Token) BurnEnabled() bool { return t.tk.BurnEnabled }

// GasPrice get token's gasPrice
func (t *Token) GasPrice() int64 { return t.tk.GasPrice }

// StdToken get token's standard struct data
func (t *Token) StdToken() *std.Token { return &t.tk }

// SetOwner set owner of Token
func (t *Token) SetOwner(newOwner types.Address) {

	// update the old owner and new owner's balance
	oldAcct := t.smc.Helper().AccountHelper().AccountOf(t.tk.Owner)
	oldOwnerBalance := oldAcct.Balance()
	oldAcct.(*Account).SetBalanceOfToken(t.Address(), bn.N(0))

	newAcct := t.smc.Helper().AccountHelper().AccountOf(newOwner)
	newOwnerBalance := newAcct.Balance().Add(oldOwnerBalance)
	newAcct.(*Account).SetBalanceOfToken(t.Address(), newOwnerBalance)

	// dirty mc and set new token data
	t.tk.Owner = newOwner
	keyOfToken := std.KeyOfToken(t.Address())
	sdkimpl.McInst.Dirty(keyOfToken)
	t.smc.(*sdkimpl.SmartContract).LlState().McSet(keyOfToken, &t.tk)

	// fire event of setOwner
	t.smc.Helper().ReceiptHelper().Emit(
		std.SetOwner{
			ContractAddr: t.smc.Message().Contract().Address(),
			NewOwner:     newOwner,
		},
	)
	// fire event of transfer
	t.smc.Helper().ReceiptHelper().Emit(
		std.Transfer{
			Token: t.Address(),
			From:  oldAcct.Address(),
			To:    newOwner,
			Value: oldOwnerBalance,
		},
	)
}

// SetTotalSupply set totalSupply of token
func (t *Token) SetTotalSupply(totalSupply bn.Number) {
	sdk.RequireMainChain()
	sdk.RequireOwner()

	// get update number
	updateSupply := totalSupply.Sub(t.TotalSupply())

	// first check burnEnabled or addSupplyEnabled flag and then
	// update token's totalSupply and owner's balance
	if updateSupply.CmpI(0) > 0 {
		sdk.Require(t.AddSupplyEnabled() == true,
			types.ErrAddSupplyNotEnabled, "")
	} else {
		sdk.Require(t.BurnEnabled() == true,
			types.ErrBurnNotEnabled, "")
	}

	// return ok if not change totalSupply
	if t.TotalSupply().Cmp(totalSupply) == 0 {
		return
	}

	// totalSupply must great than or equal one token(1E9 cong)
	sdk.Require(totalSupply.CmpI(minTotalSupply) >= 0,
		types.ErrInvalidParameter,
		fmt.Sprintf("TotalSupply must great than or equal %d cong", minTotalSupply))

	// create owner's account and compare the owner's balance,
	// if owner's balance less than burn number, then return error
	ownerAcct := t.smc.Helper().AccountHelper().AccountOf(t.tk.Owner)
	updateBalance := ownerAcct.Balance().Add(updateSupply)
	sdk.Require(updateBalance.CmpI(0) >= 0,
		types.ErrInvalidParameter, "The owner's balance not enough to burn")

	// dirty mc and set new token data
	t.tk.TotalSupply = totalSupply
	keyOfToken := std.KeyOfToken(t.Address())
	sdkimpl.McInst.Dirty(keyOfToken)
	t.smc.(*sdkimpl.SmartContract).LlState().McSet(keyOfToken, &t.tk)

	// update owner's balance
	ownerAcct.(*Account).SetBalanceOfToken(t.smc.Message().Contract().Token(), updateBalance)

	// fire event of addSupply or burn
	if updateSupply.CmpI(0) > 0 {
		// fire event of addSupply
		t.smc.Helper().ReceiptHelper().Emit(
			std.AddSupply{
				Token:       t.Address(),
				Value:       updateSupply,
				TotalSupply: totalSupply,
			},
		)

		// fire event of transfer
		t.smc.Helper().ReceiptHelper().Emit(
			std.Transfer{
				Token: t.Address(),
				From:  "",
				To:    t.tk.Owner,
				Value: updateSupply,
			},
		)
	} else {
		// fire event of burn
		t.smc.Helper().ReceiptHelper().Emit(
			std.Burn{
				Token:       t.Address(),
				Value:       bn.N(0).Sub(updateSupply),
				TotalSupply: totalSupply,
			},
		)

		// fire event of transfer
		t.smc.Helper().ReceiptHelper().Emit(
			std.Transfer{
				Token: t.Address(),
				From:  t.tk.Owner,
				To:    "",
				Value: bn.N(0).Sub(updateSupply),
			},
		)
	}
}

// SetGasPrice set gasPrice of token
func (t *Token) SetGasPrice(gasPrice int64) {
	sdk.RequireMainChain()
	sdk.RequireOwner()

	// TODO 验证线上有没有调过该接口，原逻辑不报错
	sdk.Require(t.tk.GasPrice != gasPrice,
		types.ErrInvalidParameter, "New gasPrice cannot equal old gasPrice")

	sdk.Require(gasPrice <= maxGasPrice,
		types.ErrInvalidParameter,
		fmt.Sprintf("New gasPrice cannot great than maxGasPrice=%d", maxGasPrice))

	// new gasPrice must great than baseGasPrice
	baseGasPrice := t.smc.Helper().TokenHelper().BaseGasPrice()
	sdk.Require(gasPrice >= baseGasPrice,
		types.ErrInvalidParameter,
		fmt.Sprintf("New gasPrice cannot less than baseGasPrice=%d", baseGasPrice))

	t.tk.GasPrice = gasPrice
	// dirty mc and submit new token data
	keyOfToken := std.KeyOfToken(t.Address())
	sdkimpl.McInst.Dirty(keyOfToken)

	key := std.KeyOfToken(t.Address())
	t.smc.(*sdkimpl.SmartContract).LlState().McSet(key, &t.tk)

	// fire event of setGasPrice
	t.smc.Helper().ReceiptHelper().Emit(
		std.SetGasPrice{
			Token:    t.Address(),
			GasPrice: gasPrice,
		},
	)
}
