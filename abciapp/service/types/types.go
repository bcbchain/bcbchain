package types

import (
	abcicli "github.com/bcbchain/bclib/tendermint/abci/client"
	types2 "github.com/bcbchain/bclib/tx/types"
	"github.com/bcbchain/bclib/types"
)

type TxOrder struct {
	RawTx  []byte
	Index  int
	ReqRes *abcicli.ReqRes
}

type ResponseOrder struct {
	TxID        int64
	Transaction types.Transaction
	Tx          []byte
	Response    *types.Response
	Index       int
	ReqRes      *abcicli.ReqRes
}
type Result2 struct {
	TxID       int64             `json:"txId,omitempty"`    //每一个区块transaction中的TxID,提供给数据库层
	TxOrder    int               `json:"txOrder,omitempty"` //每一批交易中的交易序号
	TxVersion  string            `json:"txVersion,omitempty"`
	Tx         []byte            `json:"tx,omitempty"`
	TxV1Result types2.TxV1Result `json:"txV1Result,omitempty"`
	TxV2Result types2.TxV2Result `json:"txV2Result,omitempty"`
	TxV3Result types2.TxV3Result `json:"txV3Result,omitempty"`
	ErrorLog   error             `json:"errorLog,omitempty"`
	ReqRes     *abcicli.ReqRes   `json:"reqRes,omitempty"`
}

//
//import "github.com/bcbchain/bclib/tendermint/abci/types"
//
//type txPool struct {
//	maxCurrency    int
//	beginBlockInfo types.RequestBeginBlock
//	txChan         chan []byte
//}
//
//func NewTxPool() *txPool {
//	return &txPool{txChan: make(chan []byte, 1000)}
//}
//
//// SetBeginBlockInfo 向交易池中写入beginblock信息
//func (T *txPool) SetBeginBlockInfo(beginBlockInfo types.RequestBeginBlock) {
//	T.beginBlockInfo = beginBlockInfo
//}
//
//// PutRawTx 向交易池中写入交易
//func (T *txPool) PutTx(tx []byte) {
//	//将交易写入交易池的通道中
//	T.txChan <- tx
//}
//
//// GetTx 从交易池中读出交易
//func (T *txPool) GetTxs() [][]byte {
//	var txs = make([][]byte, T.maxCurrency)
//	select {
//	case tx := <-T.txChan:
//		txs = append(txs, tx)
//		if len(txs) == T.maxCurrency {
//			return txs
//		}
//	default:
//		if len(txs) != 0 {
//			return txs
//		}
//	}
//	return txs
//}

//type ResultPool struct {
//	ResultChan chan types.Result
//}
//
//func NewResultPool() *ResultPool {
//	return &ResultPool{ResultChan: make(chan types.Result, 1000)}
//}
//
////记录Response的顺序，有序返回给socketsever
//type ResponseChanOrder struct {
//	Response types.ResponseCheckTx
//	Index    int
//}
//
//type ResponsePool struct {
//	ResponseOrder chan ResponseChanOrder
//}
//
//func NewResponsePool() *ResponsePool {
//	return &ResponsePool{ResponseOrder: make(chan ResponseChanOrder, 1000)}
//}
