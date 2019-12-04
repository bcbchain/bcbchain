package smc

import (
	"blockchain/abciapp_v1.0/bcerrors"
	"github.com/tendermint/tmlibs/common"
	"golang.org/x/crypto/sha3"
	"math/big"
)

// Address uses for Account, Contract,
type Address = string

// Receipt for tx
type ReceiptBytes []byte

// Hash uses for public key and others, SHA3-256
type Hash = common.HexBytes

type Chromo = string

type Error = bcerrors.BCError

// Size of Hash is 32 bytes
const HASH_LEN = 32

// pubkey uses for public key and others, PubKeyEd25519
type PubKey = common.HexBytes

// Size of Hash is 32 bytes
const PUBKEY_LEN = 32

// Define the maximum of gas price as one BCB (1,000,000,000 cong),
// GasBasePrice either.
const Max_Gas_Price = 1000000000

// Size of
const Max_Name_Len = 40

func (r ReceiptBytes) String() string {
	return string(r)
}

type Receipt struct {
	Name            string  `json:"name"`            //收据名称：标准名称（trnsfer，...) 非标准名称（...）
	ContractAddress Address `json:"contractAddress"` //事件发起方的合约地址
	ReceiptBytes    []byte  `json:"receiptBytes"`
	ReceiptHash     Hash    `json:"receiptHash"`
}

// Receipt for transfer
type ReceiptOfTransfer struct {
	Token Address `json:"token"` // Token Address
	From  Address `json:"from"`  // External account address of Sender
	To    Address `json:"to"`    // External account address of Receiver
	Value big.Int `json:"value"` // Transfer value
}

// Name of receipt: AddSupply
type ReceiptOfAddSupply struct {
	Token       Address `json:"token"`       // 代币地址
	Value       big.Int `json:"value"`       // 增发的供应量（单位：cong）
	TotalSupply big.Int `json:"totalSupply"` // 新的总供应量（单位：cong）
}

// Name of receipt: Burn
type ReceiptOfBurn struct {
	Token       Address `json:"token"`       // 代币地址
	Value       big.Int `json:"value"`       // 燃烧的供应量（单位：cong）
	TotalSupply big.Int `json:"totalSupply"` // 新的总供应量（单位：cong）
}

// Name of receipt: SetGasPrice
type ReceiptOfSetGasPrice struct {
	Token    Address `json:"token"`    // 代币地址
	GasPrice uint64  `json:"gasPrice"` // 燃料价格（单位：cong）
}

// Name of receipt: SetOwner
type ReceiptOfSetOwner struct {
	ContractAddr Address `json:"contractAddr"` // 智能合约地址
	NewOwner     Address `json:"newOwner"`     // 合约新的拥有者的外部账户地址
}

// Name of receipt: Fee
type ReceiptOfFee struct {
	Token Address `json:"token"` // 代币地址
	From  Address `json:"from"`  // 支付手续费的账户地址
	Value uint64  `json:"value"` // 手续费（单位：cong）
}

type ReceiptOfNewToken struct {
	TokenAddress     Address `json:"tokenAddress"`     //代币的智能合约账户地址
	ContractAddress  Address `json:"contractAddress"`  //合约地址
	AccountAddress   Address `json:"accountAddress"`   //合约账户地址
	Owner            Address `json:"owner"`            //合约所有者的外部账户地址
	Version          string  `json:"version"`          //合约的版本
	Name             string  `json:"name,omitempty"`   //代币的名称
	Symbol           string  `json:"symbol"`           //代币的符号
	TotalSupply      big.Int `json:"totalSupply"`      //代币的总供应量
	AddSupplyEnabled bool    `json:"addSupplyEnabled"` //代币是否支持增发
	BurnEnabled      bool    `json:"burnEnabled"`      //代币是否支持燃烧
	GasPrice         uint64  `json:"gasprice"`         //代币燃料价格
}

func CalcReceiptHash(name string, addr Address, receiptByte []byte) Hash {
	hasherSHA3256 := sha3.New256()
	hasherSHA3256.Write([]byte(name))
	hasherSHA3256.Write([]byte(addr))
	hasherSHA3256.Write(receiptByte)

	return hasherSHA3256.Sum(nil)
}
