package app

import (
	"github.com/bcbchain/bcbchain/abciapp/service/deliver"
	types2 "github.com/bcbchain/bcbchain/abciapp/service/types"
	"github.com/bcbchain/bcbchain/burrow"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bcbchain/smcrunctl/invokermgr"
	"github.com/bcbchain/bcbchain/statedb"
	abcicli "github.com/bcbchain/bclib/tendermint/abci/client"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	types3 "github.com/bcbchain/bclib/types"
	"runtime"
)

var MaxCurrency = runtime.NumCPU() * 2

//type TxOrder struct {
//	RawTx []byte
//	Index int
//}

type TxPool struct {
	txID    int
	transID int64
	log     log.Logger

	responseNum    int
	maxCurrency    int //控制一批交易的数量
	appDeliver     deliver.AppDeliver
	BeginBlockInfo types.RequestBeginBlock //存储区块头等数据

	responseChan chan<- *types.Response //最终结果返回的通道

	RawTxChan     chan types2.TxOrder       //池子中用来存储交易的管道
	ResultChan    chan types2.Result2       //提供给数据库层的tx
	ResponseOrder chan types2.ResponseOrder //带有序号的response.需要按需返回并且进行需要进行类型转换
}

func NewTxPool(log log.Logger) *TxPool {
	return &TxPool{RawTxChan: make(chan types2.TxOrder, 1000), ResultChan: make(chan types2.Result2, 1000), ResponseOrder: make(chan types2.ResponseOrder, 1000), maxCurrency: MaxCurrency, log: log}
}

func (T *TxPool) ResetTxPool(log log.Logger) {
	T = &TxPool{RawTxChan: make(chan types2.TxOrder, 1000), ResultChan: make(chan types2.Result2, 1000), ResponseOrder: make(chan types2.ResponseOrder, 1000), maxCurrency: MaxCurrency, log: log}
}

// SetBeginBlockInfo 向交易池中写入beginBlock信息
func (T *TxPool) SetBeginBlockInfo(beginBlockInfo types.RequestBeginBlock) {
	T.BeginBlockInfo = beginBlockInfo
}

func (T *TxPool) SetTransID(TransID int64) {
	T.transID = TransID
}

func (T *TxPool) SetResponseChan(responseChan chan<- *types.Response) {
	T.responseChan = responseChan
}

// PutRawTx 向交易池中写入原生交易
func (T *TxPool) PutRawTx(tx []byte, reqRes *abcicli.ReqRes) {
	T.log.Info("TxPool 收到 tx", tx)
	T.log.Info("TxPool 收到 reqRes.Request", reqRes.Request)

	//将交易写入交易池的通道中
	T.RawTxChan <- types2.TxOrder{RawTx: tx, Index: T.txID + 1, ReqRes: reqRes}
}

func (T *TxPool) PutResults(result types2.Result2) {
	T.log.Info("txPool 收到 result", result)
	//将交易写入交易池的通道中
	T.ResultChan <- result
}

// GetTx 从交易池中读出数据库层交易,若有交易，布尔值为真
func (T *TxPool) GetResults() ([]types2.Result2, bool) {
	var result2s = make([]types2.Result2, 0)
	for {
		select {
		case Result2 := <-T.ResultChan:
			result2s = append(result2s, Result2)
			if len(result2s) == T.maxCurrency {
				T.responseNum = len(result2s)
				T.log.Info("txPool 发送出一批完整交易 result2s", result2s)
				return result2s, true
			}
		default:
			if len(result2s) != 0 {
				T.responseNum = len(result2s)
				T.log.Info("txPool 发送出部分交易 result2s", result2s)
				return result2s, true
			} else {
				return nil, false
			}
		}
	}
}

// GetTx 从交易池中读出数据库层交易,若有交易，布尔值为真
func (T *TxPool) GetRawTxs() ([]types2.TxOrder, bool) {
	var txs = make([]types2.TxOrder, 0)
	for {
		select {
		case tx := <-T.RawTxChan:
			T.log.Info("txPool 收到txOrder", tx)
			txs = append(txs, tx)
			if len(txs) == T.maxCurrency {
				T.log.Info("txPool 发送出一批完整交易 txs", txs)
				return txs, true
			}
		default:
			if len(txs) != 0 {
				T.log.Info("txPool 发送出部分交易 txs", txs)
				return txs, true
			} else {
				return nil, false
			}
		}
	}
}

//需要给外部已经排好序的Response
func (T *TxPool) GetResponses() ([]types3.Response, bool) {
	ResponseCheckTxMap := make(map[int]types3.Response, 0)
	var Responses []types3.Response
	var i = 0
	for {
		select {
		case responseChanOrder := <-T.ResponseOrder:
			ResponseCheckTxMap[responseChanOrder.Index] = *responseChanOrder.Response
			for {
				if _, ok := ResponseCheckTxMap[i]; ok {
					Responses = append(Responses, ResponseCheckTxMap[i])
					delete(ResponseCheckTxMap, i)
					i++
					if i == T.maxCurrency {
						i = 0
					}
				} else {
					break
				}
			}
		}
	}
}

//需要给外部已经排好序的ResponseOrder
func (T *TxPool) GetResponsesOrder() ([]types2.ResponseOrder, bool) {
	ResponseCheckTxMap := make(map[int]types2.ResponseOrder, 0)
	var Responses []types2.ResponseOrder
	var i = 0
	for {
		select {
		case responseChanOrder := <-T.ResponseOrder:
			T.log.Info("TxPool 收到 responseChanOrder ", responseChanOrder)
			ResponseCheckTxMap[responseChanOrder.Index] = responseChanOrder
			for {
				if _, ok := ResponseCheckTxMap[i]; ok {
					Responses = append(Responses, ResponseCheckTxMap[i])
					delete(ResponseCheckTxMap, i)
					i++
					if i == T.responseNum {
						i = 0
						T.log.Info("TxPool 发出 responseChanOrder ", Responses)
						return Responses, true
					}
				} else {
					break
				}
			}
		}
	}
}

//// Run 将原始交易解析为可以提供给数据库层执行的交易
//func (T *TxPool) GetResults() ([]types.Result, bool) {
//	var results = []types.Result{}
//	select {
//	case result := <-T.ResultChan:
//		results = append(results, result)
//		if len(results) == T.maxCurrency {
//			return results, true
//		}
//	default:
//		if len(results) != 0 {
//			return results, true
//		}
//	}
//	return nil, false
//}

//
//func (T *TxParser) PutRawTxs() {
//	var txs [][]byte
//	for {
//		select {
//		// 监听交易通道
//		case tx := <-T.app.txPool.ResultChan:
//			T.app.logger.Info("CheckTxConcurrency--------tx", tx)
//			txs = append(txs, tx)
//			if len(txs) == T.maxConcurrency {
//				T.TxReceiveChan <- txs
//				txs = *new([][]byte)
//			}
//		default:
//			if len(txs) != 0 {
//				T.app.logger.Info("CheckTxConcurrency---tx等待超时-----txs", txs)
//				T.TxReceiveChan <- txs
//				txs = *new([][]byte)
//			}
//		}
//	}
//}
//
//func (T *TxParser) Run() {
//	for {
//		select {
//		case txs := <-T.TxReceiveChan:
//			T.wg.Add(len(txs))
//			for i, tx := range txs {
//				result := &types.Result{}
//				splitTx := strings.Split(string(tx), ".")
//				T.app.logger.Info("CheckTxConcurrency--------splitTx", splitTx)
//				if len(splitTx) == 5 {
//					if splitTx[1] == "v1" && T.app.appv1 != nil {
//						var connV2 *check.AppCheck
//						if T.app.ChainVersion() == 2 {
//							connV2 = T.app.connCheck
//						}
//					go T.app.appv1.CheckTxV1Concurrency(tx, T.wg, connV2, T.app.resultPool, i)
//
//					} else if splitTx[1] == "v2" && T.app.ChainVersion() == 2 {
//						go T.app.connCheck.CheckTxV2Concurrency(tx, T.wg, T.app.resultPool, i)
//
//					} else if splitTx[1] == "v3" && T.app.ChainVersion() == 2 {
//						go T.app.connCheck.CheckTxV3Concurrency(tx, T.wg, T.app.resultPool, i)
//
//					} else {
//						result.ErrorLog = errors.New("invalid transaction 1")
//						result.txID = i
//						T.app.resultPool.ResultChan <- *result
//					}
//				} else {
//					result.ErrorLog = errors.New("invalid transaction 1")
//					result.txID = i
//					T.app.resultPool.ResultChan <- *result
//					fmt.Println("tx:", string(tx))
//					fmt.Println("tx len:", len(splitTx))
//				}
//			}
//			T.wg.Wait()
//		}
//	}
//}
////
//type TxExecutor struct {
//	app         *BCChainApplication
//	Result      []types.Result
//	ResultChan  chan []types.Result //目前是不带缓冲区的
//	wg          *sync.WaitGroup
//	lenResponse int
//}
//
//func (T *TxExecutor) PutTxs() {
//	for {
//		select {
//		case result := <-T.app.resultPool.ResultChan:
//			T.app.logger.Info("CheckTxConcurrency--------result", result)
//			T.Result = append(T.Result, result)
//			if len(T.Result) == goroutineNumber {
//				T.ResultChan <- T.Result
//				T.Result = *new([]types.Result)
//			}
//		default:
//			if len(T.Result) != 0 {
//				T.app.logger.Info("CheckTxConcurrency----result等待超时----result", T.Result)
//				T.ResultChan <- T.Result
//				T.Result = *new([]types.Result)
//			}
//		}
//	}
//}
//
//func (T *TxExecutor) Run() {
//	for {
//		select {
//		case results := <-T.ResultChan:
//			T.app.logger.Info("CheckTxConcurrency--------results", results)
//			T.wg.Add(len(results))
//			T.lenResponse = len(results)
//			for _, result := range results {
//				if result.TxVersion == "tx1" {
//					var connV2 *check.AppCheck
//					if T.app.ChainVersion() == 2 {
//						connV2 = T.app.connCheck
//					}
//					go T.app.appv1.RunCheckTxV1Concurrency(result, T.app.responsePool, connV2, T.wg)
//				} else if result.TxVersion == "tx2" {
//					go T.app.connCheck.RunCheckTxV2Concurrency(result, T.app.responsePool, T.wg)
//
//				} else if result.TxVersion == "tx3" {
//					go T.app.connCheck.RunCheckTxV3Concurrency(result, T.app.responsePool, T.wg)
//				}
//			}
//			T.wg.Wait()
//		}
//	}
//}
//
//func (T *TxExecutor) PutResponse() {
//	ResponseCheckTxMap := make(map[int]types.ResponseCheckTx, 0)
//	var i = 0
//	for {
//		select {
//		case responseChanOrder := <-T.app.responsePool.ResponseOrder:
//			T.app.logger.Info("CheckTxConcurrency--------responseChanOrder", responseChanOrder)
//			ResponseCheckTxMap[responseChanOrder.Index] = responseChanOrder.Response
//			for {
//				T.app.responseChan = socket.GetResponses()
//				if _, ok := ResponseCheckTxMap[i]; ok {
//					T.app.logger.Info("发送的response是", ResponseCheckTxMap[i])
//					T.app.responseChan <- types.ToResponseCheckTx(ResponseCheckTxMap[i])
//					T.app.responseChan <- types.ToResponseFlush()
//					delete(ResponseCheckTxMap, i)
//					i++
//					if i == T.lenResponse {
//
//						i = 0
//					}
//				} else {
//					break
//				}
//			}
//		}
//	}
//}

func InvokeTx(Tx *statedb.Tx, params ...interface{}) bool {
	blockHeader := params[0].(types.Header)
	transID := params[1].(int64)
	txID := params[2].(int64)
	sender := params[3].(types3.Address)
	transaction := params[4].(types3.Transaction)
	publicKey := params[5].(types3.PubKey)
	txHash := params[6].(types3.Hash)
	blockHash := params[7].(types3.Hash)
	responseChan := params[8].(chan *types2.ResponseOrder) //response成功后返回的通道
	txOrder := params[9].(int)
	logger := params[10].(log.Logger)
	tx := params[11].([]byte)
	return invokeTx(blockHeader, transID, txID, sender, transaction, publicKey, txHash, blockHash, responseChan, txOrder, logger, tx)
}

func invokeTx(
	blockHeader types.Header,
	transID, txID int64,
	sender types3.Address,
	transaction types3.Transaction,
	publicKey types3.PubKey,
	txHash types3.Hash,
	blockHash types3.Hash,
	responseChan chan *types2.ResponseOrder,
	txOrder int,
	logger log.Logger,
	tx []byte) bool {
	// Sender can do nothing if it's in black list
	if statedbhelper.CheckBlackList(transID, txID, sender) == true {
		err := types3.BcError{
			ErrorCode: types3.ErrDealFailed,
		}
		responseOrder := &types2.ResponseOrder{
			Response: &types3.Response{
				Code: err.ErrorCode,
				Log:  err.Error(),
			},
			Index:       txOrder,
			TxID:        txID,
			Transaction: transaction,
			Tx:          tx,
		}
		responseChan <- responseOrder
		return false
	}

	methodID := transaction.Messages[len(transaction.Messages)-1].MethodID
	if methodID == 0 || methodID == 0xFFFFFFFF {
		if !statedbhelper.CheckBVMEnable(transID, txID) {

			responseOrder := &types2.ResponseOrder{
				Response: &types3.Response{
					Code: types3.ErrLogicError,
					Log:  "BVM is disabled",
				},
				Index:       txOrder,
				TxID:        txID,
				Transaction: transaction,
				Tx:          tx,
			}
			responseChan <- responseOrder
			return true
		}
		response := burrow.GetInstance(logger).InvokeTx(blockHeader, blockHash, transID, txID, sender, transaction, publicKey)

		responseOrder := &types2.ResponseOrder{
			Index:       txOrder,
			Response:    response,
			TxID:        txID,
			Transaction: transaction,
			Tx:          tx,
		}
		responseChan <- responseOrder
		return true
	}

	response := invokermgr.GetInstance().InvokeTx(blockHeader, transID, txID, sender, transaction, publicKey, txHash, blockHash)
	responseOrder := &types2.ResponseOrder{
		Index:       txOrder,
		Response:    response,
		TxID:        txID,
		Transaction: transaction,
		Tx:          tx,
	}
	responseChan <- responseOrder
	return true
}
