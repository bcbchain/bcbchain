package tx

import (
	"blockchain/abciapp_v1.0/keys"
	"math/big"
)

type MethodInfo struct {
	MethodID  uint32
	ParamData []byte
}

type BigNumber = big.Int

type Method struct {
	MethodID  uint32
	Prototype string
}

// 定义交易数据结构
type Transaction struct {
	Nonce    uint64       // 交易发起者发起交易的计数值，从1开始，必须单调增长，增长步长为1。
	GasLimit uint64       // 交易发起者愿意为执行此次交易支付的GAS数量的最大值。
	Note     string       // UTF-8编码的备注信息，要求小于256个字符。
	To       keys.Address // 合约地址
	Data     []byte       // 调用智能合约所需要的参数，RLP编码格式。
}

const MAX_SIZE_NOTE = 256

type Query struct {
	QueryKey string
}

// 生成一笔交易框架数据结构；
// 如果不是调用智能合约，data参数传入nil
// 如果是调用智能合约，data参数需要事先调用TokenBasicInvoker...的接口，生成经过RLP编码的合约调用参数
func NewTransaction(nonce uint64, gaslimit uint64, note string, to keys.Address, data []byte) Transaction {
	tx := Transaction{
		Nonce:    nonce,
		GasLimit: gaslimit,
		Note:     note,
		To:       to,
		Data:     data,
	}
	return tx
}
