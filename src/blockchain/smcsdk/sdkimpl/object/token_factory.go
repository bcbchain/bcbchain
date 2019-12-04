package object

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
)

// NewToken factory method for create token with all token's property
func NewToken(
	smc sdk.ISmartContract,
	address types.Address, //代币地址
	owner types.Address, //代币拥有者的账户地址
	name string, //代币的名称
	symbol string, //代币的符号
	totalSupply bn.Number, //代币的总供应量
	addSupplyEnabled bool, //代币是否支持增发
	burnEnabled bool, //代币是否支持燃烧
	gasPrice int64) sdk.IToken { //代币燃料价格
	o := Token{
		smc: smc,
		tk: std.Token{
			Address:          address,
			Owner:            owner,
			Name:             name,
			Symbol:           symbol,
			TotalSupply:      totalSupply,
			AddSupplyEnabled: addSupplyEnabled,
			BurnEnabled:      burnEnabled,
			GasPrice:         gasPrice,
		},
	}
	return &o
}

// NewTokenFromSTD factory method for create token from std
func NewTokenFromSTD(smc sdk.ISmartContract, stdToken *std.Token) sdk.IToken {
	token := &Token{tk: *stdToken}
	token.SetSMC(smc)

	return token
}
