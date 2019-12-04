package object

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
)

// Contract contract detail information
type Contract struct {
	smc sdk.ISmartContract //指向智能合约API对象指针

	ct std.Contract
}

var _ sdk.IContract = (*Contract)(nil)
var _ sdkimpl.IAcquireSMC = (*Contract)(nil)

// SMC get smart contract object
func (c *Contract) SMC() sdk.ISmartContract { return c.smc }

// SetSMC set smart contract object
func (c *Contract) SetSMC(smc sdk.ISmartContract) { c.smc = smc }

// Address get contract's address
func (c *Contract) Address() types.Address { return c.ct.Address }

// Account get contract's account
func (c *Contract) Account() sdk.IAccount {
	return c.smc.Helper().AccountHelper().AccountOf(c.ct.Account)
}

// Owner get contract's owner
func (c *Contract) Owner() sdk.IAccount {
	return c.smc.Helper().AccountHelper().AccountOf(c.ct.Owner)
}

// Name get contract's name
func (c *Contract) Name() string { return c.ct.Name }

// Version get contract's version
func (c *Contract) Version() string { return c.ct.Version }

// CodeHash get contract's codeHash
func (c *Contract) CodeHash() types.Hash { return c.ct.CodeHash }

// EffectHeight get contract's effectHeight
func (c *Contract) EffectHeight() int64 { return c.ct.EffectHeight }

// LoseHeight get contract's loseEffect
func (c *Contract) LoseHeight() int64 { return c.ct.LoseHeight }

// KeyPrefix get contract's keyPrefix
func (c *Contract) KeyPrefix() string { return c.ct.KeyPrefix }

// Methods get contract's methods
func (c *Contract) Methods() []std.Method { return c.ct.Methods }

// Interfaces get contract's interfaces
func (c *Contract) Interfaces() []std.Method { return c.ct.Interfaces }

// IBCs get contract's across method
func (c *Contract) IBCs() []std.Method { return c.ct.IBCs }

// Mine get contract's mine method
func (c *Contract) Mine() []std.Method { return c.ct.Mine }

// Token get contract's token
func (c *Contract) Token() types.Address { return c.ct.Token }

// SetToken get set contract's token address
func (c *Contract) SetToken(tokenAddr types.Address) { c.ct.Token = tokenAddr }

// OrgID get contract's orgID
func (c *Contract) OrgID() string { return c.ct.OrgID }

// ChainVersion get contract's chainVersion
func (c *Contract) ChainVersion() int64 { return c.ct.ChainVersion }

// OrgID get contract's orgID
func (c *Contract) StdContract() *std.Contract { return &c.ct }

// SetOwner set contract's owner
func (c *Contract) SetOwner(newOwner types.Address) {
	sdk.RequireMainChain()

	// 判断sender是否有修改的权限
	sdk.RequireOwner()
	sdk.RequireAddress(newOwner)

	sdk.Require(newOwner != c.Address(),
		types.ErrInvalidParameter, "newOwner address cannot be contract address")

	sdk.Require(newOwner != c.ct.Account,
		types.ErrInvalidParameter, "newOwner address cannot be contract account address")

	// cannot set new owner is contract's owner
	sdk.Require(c.ct.Owner != newOwner,
		types.ErrInvalidParameter, "Cannot set owner to self")

	// exchange token's owner from old to new
	token := c.smc.Helper().TokenHelper().Token()
	if token != nil {
		token.(*Token).SetOwner(newOwner)
	}

	// add contract to new owner and delete contract from old owner
	c.addContractToNewOwner(newOwner)
	c.delContractFromOldOwner(c.ct.Owner)

	// dirty old contract
	key := std.KeyOfContract(c.Address())
	sdkimpl.McInst.Dirty(key)
	// set new contract
	c.ct.Owner = newOwner
	c.smc.(*sdkimpl.SmartContract).LlState().McSet(key, &c.ct)

	// fire event
	c.smc.Helper().ReceiptHelper().Emit(
		std.SetOwner{
			ContractAddr: c.Address(),
			NewOwner:     newOwner,
		},
	)
}

func (c *Contract) addContractToNewOwner(newOwner types.Address) {
	acct := c.smc.Helper().AccountHelper().AccountOf(newOwner).(*Account)
	addrList := acct.accountOfContracts()
	addrList = append(addrList, c.Address())

	key := std.KeyOfAccountContracts(newOwner)
	c.smc.(*sdkimpl.SmartContract).LlState().McSet(key, &addrList)
}

func (c *Contract) delContractFromOldOwner(oldOwner types.Address) {
	acct := c.smc.Helper().AccountHelper().AccountOf(oldOwner).(*Account)
	addrList := acct.accountOfContracts()

	for index, addr := range addrList {
		if addr == c.Address() {
			addrList = append(addrList[:index], addrList[index+1:]...)
			break
		}
	}
	key := std.KeyOfAccountContracts(oldOwner)
	c.smc.(*sdkimpl.SmartContract).LlState().McSet(key, &addrList)
}
