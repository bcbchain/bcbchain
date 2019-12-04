package tokenbasic

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
)

const (
	maxGasPrice = 1000000000
)

//TokenBasic
//@:contract:token-basic
//@:version:2.0
//@:organization:orgJgaGConUyK81zibntUBjQ33PKctpk1K1G
//@:author:5e8339cb1a5cce65602fd4f57e115905348f7e83bcbe38dd77694dbe1f8903c9
type TokenBasic struct {
	sdk sdk.ISmartContract
}

//InitChain: construct function
//@:constructor
func (t *TokenBasic) InitChain() {

}

// Transfer is used to transfer token from sender to another specified account
// In the TokenBasic contract, it's  only used to transfer the basic token
//@:public:method:gas[500]
//@:public:interface:gas[450]
func (t *TokenBasic) Transfer(to types.Address, value bn.Number) {
	// Do transfer
	t.sdk.Message().Sender().Transfer(to, value)
}

//SetGasPrice is used to set gas price for token-basic contract
//@:public:method:gas[2000]
func (t *TokenBasic) SetGasPrice(value int64) {
	t.sdk.Helper().TokenHelper().Token().SetGasPrice(value)
}

// SetBaseGasPrice is used to set base gas price.
// The base gas price is a minimum limit to all of token's.
// All new tokens' gas price could not be set to a value that smaller than base gas price.
//@:public:method:gas[2000]
func (t *TokenBasic) SetBaseGasPrice(value int64) {

	sdk.RequireOwner()
	sdk.Require(value > 0 && value <= maxGasPrice,
		types.ErrInvalidParameter, "Invalid base gas price")

	t.sdk.Helper().StateHelper().McSet(std.KeyOfTokenBaseGasPrice(), &value)

	type BaseGasPrice struct {
		Value int64 `json:"value"`
	}
	t.sdk.Helper().ReceiptHelper().Emit(
		BaseGasPrice{
			Value: value,
		})
}
