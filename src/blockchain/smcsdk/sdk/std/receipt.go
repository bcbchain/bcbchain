package std

import (
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/types"
	"strconv"
)

// Receipt receipt information
type Receipt struct {
	Name         string     `json:"name"`            // 收据名称：标准名称（trnsfer，...) 非标准名称（...）
	ContractAddr string     `json:"contractAddress"` // 合约地址
	Bytes        []byte     `json:"receiptBytes"`
	Hash         types.Hash `json:"receiptHash"`
}

// Transfer transfer receipt information
type Transfer struct {
	Token types.Address `json:"token"`          // Token types.Address
	From  types.Address `json:"from"`           // Account address of Sender
	To    types.Address `json:"to"`             // Account address of Receiver
	Value bn.Number     `json:"value"`          // Transfer value
	Note  string        `json:"note,omitempty"` // Transfer note
}

// SetOwner setOwner receipt information
type SetOwner struct {
	ContractAddr types.Address `json:"contractAddr"` // 智能合约地址
	NewOwner     types.Address `json:"newOwner"`     // 合约新的拥有者的外部账户地址
}

// Fee fee receipt information
type Fee struct {
	Token types.Address `json:"token"` // 代币地址
	From  types.Address `json:"from"`  // 支付手续费的账户地址
	Value int64         `json:"value"` // 手续费（单位：cong）
}

func (f *Fee) String() string {
	return "[Token='" + f.Token + "',From='" + f.From + "',Value=" + strconv.FormatInt(f.Value, 10) + "]"
}

// SetGasPrice setGasPrice receipt information
type SetGasPrice struct {
	Token    types.Address `json:"token"`    // 代币地址
	GasPrice int64         `json:"gasPrice"` // 燃料价格（单位：cong）
}

// Burn burn receipt information
type Burn struct {
	Token       types.Address `json:"token"`       // 代币地址
	Value       bn.Number     `json:"value"`       // 燃烧的供应量（单位：cong）
	TotalSupply bn.Number     `json:"totalSupply"` // 新的总供应量（单位：cong）
}

// AddSupply addSupply receipt information
type AddSupply struct {
	Token       types.Address `json:"token"`       // 代币地址
	Value       bn.Number     `json:"value"`       // 增发的供应量（单位：cong）
	TotalSupply bn.Number     `json:"totalSupply"` // 新的总供应量（单位：cong）
}

// NewToken newToken receipt information
type NewToken struct {
	TokenAddress     types.Address `json:"tokenAddress"`     // 代币地址
	ContractAddress  types.Address `json:"contractAddress"`  // 代币的合约地址
	Owner            types.Address `json:"owner"`            // 代币拥有者的外部账户地址
	Name             string        `json:"name"`             // 代币名称
	Symbol           string        `json:"symbol"`           // 代币符号
	TotalSupply      bn.Number     `json:"totalSupply"`      // 代币总供应量（单位：cong）
	AddSupplyEnabled bool          `json:"addSupplyEnabled"` // 代币是否支持增发
	BurnEnabled      bool          `json:"burnEnabled"`      // 代币是否支持燃烧
	GasPrice         int64         `json:"gasPrice"`         // 代币燃料价格（单位：cong）
}

// AddAddress addAddress receipt information
type AddressList struct {
	Blacklist []types.Address `json:"blacklist"` // 黑名单地址
}
