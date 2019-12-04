package tokenbasic

import (
	"testing"

	"blockchain/smcsdk/sdk/types"

	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/utest"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func (mysuit *MySuite) TestToken_basic_SetGasBasePrice(c *C) {
	utest.Init(orgID)
	contractOwner := utest.DeployContract(c, contractName, orgID, contractMethods, contractMethods)
	test := NewTestObject(contractOwner)
	test.run().setSender(contractOwner).InitChain()

	account := utest.NewAccount("LOC", bn.N(1000000000))

	utest.AssertError(test.run().setSender(account).SetGasBasePrice(10000), types.ErrNoAuthorization)
	utest.AssertOK(test.run().setSender(contractOwner).SetGasBasePrice(10000))
	utest.AssertError(test.run().setSender(contractOwner).SetGasBasePrice(0), types.ErrInvalidParameter)
	utest.AssertError(test.run().setSender(contractOwner).SetGasBasePrice(maxGasPrice+1), types.ErrInvalidParameter)
}

func (mysuit *MySuite) TestToken_basic_Transfer(c *C) {
	utest.Init(orgID)
	contractOwner := utest.DeployContract(c, contractName, orgID, contractMethods, contractMethods)
	test := NewTestObject(contractOwner)
	test.run().setSender(contractOwner).InitChain()

	account1 := utest.NewAccount("", bn.N(0))
	account2 := utest.NewAccount("", bn.N(0))

	utest.AssertOK(test.run().setSender(contractOwner).Transfer(account1.Address(), bn.N(10000)))
	utest.AssertError(test.run().setSender(account1).Transfer(account2.Address(), bn.N(10001)), types.ErrInsufficientBalance)
	utest.AssertOK(test.run().setSender(account1).Transfer(account2.Address(), bn.N(10000)))
	utest.Assert(account2.Balance().CmpI(10000) == 0)
	utest.Assert(account1.Balance().CmpI(0) == 0)
}

func (mysuit *MySuite) TestToken_basic_SetGasPrice(c *C) {
	utest.Init(orgID)
	contractOwner := utest.DeployContract(c, contractName, orgID, contractMethods, contractMethods)
	test := NewTestObject(contractOwner)
	test.run().setSender(contractOwner).InitChain()

	utest.AssertError(test.run().setSender(contractOwner).SetGasBasePrice(10000), types.CodeOK)

	accounts := utest.NewAccounts("", bn.N(1E13), 1)
	if accounts == nil {
		panic("初始化newOwner失败")
	}
	utest.AssertError(test.run().setSender(accounts[0]).SetGasPrice(10000), types.ErrNoAuthorization)
	utest.AssertOK(test.run().setSender(contractOwner).SetGasBasePrice(10001))
	utest.AssertError(test.run().setSender(contractOwner).SetGasPrice(9999), types.ErrInvalidParameter)
	utest.AssertError(test.run().setSender(contractOwner).SetGasPrice(maxGasPrice+1), types.ErrInvalidParameter)
}
