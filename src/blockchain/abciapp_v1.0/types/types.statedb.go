package types

import (
	"blockchain/abciapp_v1.0/smc"
	"github.com/tendermint/go-crypto"
	"math/big"
)

type IssueToken struct {
	Address          smc.Address `json:"address"`          //代币的智能合约账户地址
	Owner            smc.Address `json:"owner"`            //合约所有者的外部账户地址
	Version          string      `json:"version"`          //合约的版本
	Name             string      `json:"name,omitempty"`   //代币的名称
	Symbol           string      `json:"symbol"`           //代币的符号
	TotalSupply      big.Int     `json:"totalSupply"`      //代币的总供应量
	AddSupplyEnabled bool        `json:"addSupplyEnabled"` //代币是否支持增发
	BurnEnabled      bool        `json:"burnEnabled"`      //代币是否支持燃烧
	GasPrice         uint64      `json:"gasprice"`         //代币燃料价格
}

//statedb key = "/account/ex/.../tokens
type TokenBalance struct {
	Address smc.Address `json:"address"` //代币的合约账户地址
	Balance big.Int     `json:"balance"` //代币的余额
}
type TokenBalances []TokenBalance

type Method struct {
	MethodId  string `json:"methodId,omitempty"`  //方法ID
	Gas       int64  `json:"gas,omitempty"`       //该方法需要消耗的gas
	Prototype string `json:"prototype,omitempty"` //方法原型
}

//statedb key = "/genesis/sc/..."
//statedb key = "/account/sc/.../contract"
type Contract struct {
	Address      smc.Address `json:"address,omitempty"`      //智能合约账户地址
	Owner        smc.Address `json:"owner,omitempty"`        //合约所有者的外部账户地址
	Name         string      `json:"name,omitempty"`         //合约的名称
	Version      string      `json:"version,omitempty"`      //合约的版本
	CodeHash     string      `json:"codeHash,omitempty"`     //合约代码的散列值
	Methods      []Method    `json:"methods,omitempty"`      //合约公开的方法
	EffectHeight uint64      `json:"effectHeight,omitempty"` //合约生效高度
	LoseHeight   uint64      `json:"loseHeight,omitempty"`   //合约失效高度
	ChainVersion int64       `json:"chainVersion,omitempty"` //链版本
	OrgID        string      `json:"orgID,omitempty"`        //链版本
}

type RewardStrategy struct {
	Strategy     []Rewarder `json:"rewardStrategy,omitempty"` //奖励策略
	EffectHeight uint64     `json:"effectHeight,omitempty"`   //生效高度
}

type Rewarder struct {
	Name          string `json:"name"`          // 被奖励者名称
	RewardPercent string `json:"rewardPercent"` // 奖励比例
	Address       string `json:"address"`       // 被奖励者地址
}

type AccountInfo struct {
	Nonce uint64 //账户的nonce,从1开始，每次执行deliverTx时，Sender的nonce加1
}

type TokenFee struct {
	MaxFee uint64 `json:"maxFee"`
	MinFee uint64 `json:"minFee"`
	Ratio  uint64 `json:"ratio"`
}

type AccountFee struct {
	Fee   uint64 `json:"fee"`
	Payer string `json:"payer"`
}

type UDCOrder struct {
	UDCState     string      `json:"udcstate,omitempty"`     //订单状态：unmature, invalid, mature
	UDCHash      crypto.Hash `json:"udchash,omitempty"`      //UDCHash,计算Hash时不包含状态
	Nonce        uint64      `json:"nonce,omitempty"`        //索引号
	ContractAddr smc.Address `json:"contractaddr,omitempty"` //代币的智能合约地址
	Owner        smc.Address `json:"owner,omitempty"`        //接收地址
	Value        big.Int     `json:"value,omitempty"`        //接收的代币数量
	MatureDate   string      `json:"maturedate,omitempty"`   //到期日期（“2016-01-01”）
}

type Validator struct {
	Name       string      `json:"name,omitempty"`       //节点组织名称
	NodePubKey smc.PubKey  `json:"nodepubkey,omitempty"` //节点公钥
	NodeAddr   smc.Address `json:"nodeaddr,omitempty"`   //节点公钥
	RewardAddr smc.Address `json:"rewardaddr,omitempty"` //节点接收奖励的地址
	Power      uint64      `json:"power,omitempty"`      //节点记账权重
}
