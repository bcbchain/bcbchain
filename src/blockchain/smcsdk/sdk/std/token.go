package std

import (
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/types"
	"strings"
)

// Token token detail information
type Token struct {
	Address          types.Address `json:"address"`          //代币地址
	Owner            types.Address `json:"owner"`            //代币拥有者的账户地址
	Name             string        `json:"name"`             //代币的名称
	Symbol           string        `json:"symbol"`           //代币的符号
	TotalSupply      bn.Number     `json:"totalSupply"`      //代币的总供应量
	AddSupplyEnabled bool          `json:"addSupplyEnabled"` //代币是否支持增发
	BurnEnabled      bool          `json:"burnEnabled"`      //代币是否支持燃烧
	GasPrice         int64         `json:"gasprice"`         //代币燃料价格
}

// KeyOfAllToken the access key for all tokens
// data for this key refer []types.Address
func KeyOfAllToken() string { return "/token/all/0" }

// KeyOfToken for create key for token with address
// data for this key refer Token
func KeyOfToken(tokenAddr types.Address) string { return "/token/" + tokenAddr }

// KeyOfTokenWithName for create key for token with name
// data for this key refer types.Address
func KeyOfTokenWithName(name string) string { return "/token/name/" + strings.ToLower(name) }

// KeyOfTokenWithSymbol for create key for token with symbol
// data for this key refer types.Address
func KeyOfTokenWithSymbol(symbol string) string { return "/token/symbol/" + strings.ToLower(symbol) }

// KeyOfTokenBaseGasPrice for create key for token base gasPrice with address
// data for this key refer uint64
func KeyOfTokenBaseGasPrice() string { return "/token/basegasprice" }

// KeyOfGenesisToken the access key for genesis token in state database
// data for this key refer Token
func KeyOfGenesisToken() string { return "/genesis/token" }

// KeyOfSupportSideChains the access key for support side chains of token address in state database
// data for this key refer []string
func KeyOfSupportSideChains(tokenAddr types.Address) string {
	return "/token/supportsidechains/" + tokenAddr
}
