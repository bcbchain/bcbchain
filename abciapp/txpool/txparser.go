package txpool

import (
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/prototype"
	bctx "github.com/bcbchain/bcbchain/abciapp_v1.0/tx/tx"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bclib/bn"
	"github.com/bcbchain/bclib/rlp"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
	tx2 "github.com/bcbchain/bclib/tx/v2"
	tx3 "github.com/bcbchain/bclib/tx/v3"
	types2 "github.com/bcbchain/bclib/types"
	"math/big"
	"strings"
)

const (
	V2TransferMethodID = 0x44d8ca60
)

var (
	ChainVerison = 0
)

// DeliverParseTx Uniformly parse txs and do basic checksum interfaces, call when executing blocks
func ParseDeliverTx(txStr string) (
	sender string,
	pubKey crypto.PubKeyEd25519,
	rawTxV1 *bctx.Transaction,
	rawTxV2 *types2.Transaction) {

	chainID := statedbhelper.GetChainID()
	splitTx := strings.Split(txStr, ".")
	if len(splitTx) == 5 {
		if splitTx[1] == "v1" {
			return parseTxV1(chainID, txStr)
		} else if splitTx[1] == "v2" {
			return parseTxV2(chainID, txStr)
		} else if splitTx[1] == "v3" {
			return parseTxV3(chainID, txStr)
		}
	} else {
		panic("invalid transaction")
	}

	return
}

// parseTxV1 解析v1版本交易并做基础校验
func parseTxV1(chainID, txStr string) (
	sender string,
	pubKey crypto.PubKeyEd25519,
	rawTxV1 *bctx.Transaction,
	rawTxV2 *types2.Transaction) {

	var err error
	rawTxV1 = new(bctx.Transaction)

	sender, pubKey, err = rawTxV1.TxParse(chainID, txStr)
	if err != nil {
		panic(err)
	}

	// Check note, it must stay within 256 characters limit
	if len(rawTxV1.Note) > bctx.MAX_SIZE_NOTE {
		panic("note is too long")
	}

	// 判断是否v1版本的转账交易，非转账交易不用转换
	var methodInfo bctx.MethodInfo
	if err = rlp.DecodeBytes(rawTxV1.Data, &methodInfo); err != nil {
		panic(err)
	}
	if ChainVerison == 2 && methodInfo.MethodID == stubapi.ConvertPrototype2ID(prototype.TbTransfer) {
		rawTxV2 = exchangeV1toV2(rawTxV1, methodInfo)
		rawTxV1 = nil
	}

	return
}

// exchangeV1toV2 将v1版本的代币转账交易转换为v2版本的代币转账交易
func exchangeV1toV2(rawTxV1 *bctx.Transaction, methodInfo bctx.MethodInfo) (rawTxV2 *types2.Transaction) {

	// 从数据库加载合约信息用于判断是否1.0标准代币合约
	contract := statedbhelper.GetContract(rawTxV1.To)
	if contract == nil || contract.ChainVersion != 0 {
		panic("contract not exist or version is wrong")
	}

	// 判断是否标准代币合约
	if contract.OrgID == statedbhelper.GetGenesisOrgID(0, 0) &&
		contract.ChainVersion == 0 &&
		(contract.Name == "token-basic" || strings.HasPrefix(contract.Name, "token-templet-")) {
		// 构造v2版本消息对像
		msg := types2.Message{
			Contract: contract.Token,
		}
		msg.MethodID, msg.Items = exchangeParams(methodInfo)

		rawTxV2 = &types2.Transaction{
			Nonce:    rawTxV1.Nonce,
			GasLimit: int64(rawTxV1.GasLimit),
			Note:     rawTxV1.Note,
			Messages: []types2.Message{msg},
		}
	}

	return
}

// exchangeParams 将v1版本格式的方法参数打包方式转换为v2版本格式的方法参数打包方式
func exchangeParams(methodInfo bctx.MethodInfo) (methodID uint32, items []common.HexBytes) {
	var itemsBytes = make([][]byte, 0)
	if err := rlp.DecodeBytes(methodInfo.ParamData, &itemsBytes); err != nil {
		panic(err)
	}

	if len(itemsBytes) != 2 {
		panic("invalid parameter's count")
	}

	to := string(itemsBytes[0][:])
	value := bn.NB(new(big.Int).SetBytes(itemsBytes[1][:]))

	items = tx2.WrapInvokeParams(to, value)
	methodID = V2TransferMethodID

	return
}

// parseTxV2 解析v2版本交易并做基础校验
func parseTxV2(chainID, txStr string) (
	sender string,
	pubKey crypto.PubKeyEd25519,
	rawTxV1 *bctx.Transaction,
	rawTxV2 *types2.Transaction) {

	var txV2 types2.Transaction
	var err error

	tx2.Init(chainID)
	txV2, pubKey, err = tx2.TxParse(txStr)
	if err != nil {
		panic(err) //eof
	}

	if len(txV2.Note) > types2.MaxSizeNote {
		panic("note is too long")
	}

	rawTxV2 = &txV2
	sender = pubKey.Address(chainID)
	return
}

// parseTxV3 解析v3版本交易并做基础校验
func parseTxV3(chainID, txStr string) (
	sender string,
	pubKey crypto.PubKeyEd25519,
	rawTxV1 *bctx.Transaction,
	rawTxV2 *types2.Transaction) {

	var txV3 types2.Transaction
	var err error

	tx3.Init(chainID)
	txV3, pubKey, err = tx3.TxParse(txStr)
	if err != nil {
		panic(err)
	}

	if len(txV3.Note) > types2.MaxSizeNote {
		panic("note is too long")
	}

	rawTxV2 = &txV3
	sender = pubKey.Address(chainID)
	return
}
