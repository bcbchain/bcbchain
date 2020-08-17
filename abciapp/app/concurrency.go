package app

import (
	"errors"
	"fmt"
	"github.com/bcbchain/bcbchain/abciapp/service/check"
	"github.com/bcbchain/bclib/socket"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"runtime"
	"strings"
	"sync"
	"time"
)

var goroutineNumber = runtime.NumCPU() * 2

type TxParser struct {
	maxConcurrency int
	TxReceiveChan  chan [][]byte //目前是不带缓冲区的
	TxExecutor     *TxExecutor
	app            *BCChainApplication
	wg             *sync.WaitGroup
}

func NewTxParser(app *BCChainApplication) *TxParser {
	txParser := &TxParser{
		maxConcurrency: goroutineNumber,
		TxReceiveChan:  make(chan [][]byte, 1000),
		TxExecutor: &TxExecutor{
			app:        app,
			Result:     make([]types.Result, 0),
			ResultChan: make(chan []types.Result, 1000),
			wg:         &sync.WaitGroup{},
		},
		app: app,
		wg:  &sync.WaitGroup{},
	}
	go txParser.PutRawTxs()
	go txParser.Run()
	go txParser.TxExecutor.PutTxs()
	go txParser.TxExecutor.Run()
	go txParser.TxExecutor.PutResponse()

	return txParser
}

func (T *TxParser) PutRawTxs() {
	var txs [][]byte
	for {
		select {
		// 监听交易通道
		case tx := <-T.app.txPool.TxChan:
			T.app.logger.Info("CheckTxConcurrency--------tx", tx)
			txs = append(txs, tx)
			if len(txs) == T.maxConcurrency {
				T.TxReceiveChan <- txs
				txs = *new([][]byte)
			}
		default:
			if len(txs) != 0 {
				T.app.logger.Info("CheckTxConcurrency---tx等待超时-----txs", txs)
				T.TxReceiveChan <- txs
				txs = *new([][]byte)
			}
		}
	}
}

func (T *TxParser) Run() {
	for {
		select {
		case txs := <-T.TxReceiveChan:
			T.wg.Add(len(txs))
			for i, tx := range txs {
				result := &types.Result{}
				splitTx := strings.Split(string(tx), ".")
				T.app.logger.Info("CheckTxConcurrency--------splitTx", splitTx)
				if len(splitTx) == 5 {
					if splitTx[1] == "v1" && T.app.appv1 != nil {
						var connV2 *check.AppCheck
						if T.app.ChainVersion() == 2 {
							connV2 = T.app.connCheck
						}
						go T.app.appv1.CheckTxV1Concurrency(tx, T.wg, connV2, T.app.resultPool, i)

					} else if splitTx[1] == "v2" && T.app.ChainVersion() == 2 {
						go T.app.connCheck.CheckTxV2Concurrency(tx, T.wg, T.app.resultPool, i)

					} else if splitTx[1] == "v3" && T.app.ChainVersion() == 2 {
						go T.app.connCheck.CheckTxV3Concurrency(tx, T.wg, T.app.resultPool, i)

					} else {
						result.Errorlog = errors.New("invalid transaction 1")
						result.TxID = i
						T.app.resultPool.ResultChan <- *result
					}
				} else {
					result.Errorlog = errors.New("invalid transaction 1")
					result.TxID = i
					T.app.resultPool.ResultChan <- *result
					fmt.Println("tx:", string(tx))
					fmt.Println("tx len:", len(splitTx))
				}
			}
			T.wg.Wait()
		}
	}
}

type TxExecutor struct {
	app         *BCChainApplication
	Result      []types.Result
	ResultChan  chan []types.Result //目前是不带缓冲区的
	wg          *sync.WaitGroup
	ticker      *time.Ticker
	lenResponse int
}

func (T *TxExecutor) PutTxs() {
	for {
		select {
		case result := <-T.app.resultPool.ResultChan:
			T.app.logger.Info("CheckTxConcurrency--------result", result)
			T.Result = append(T.Result, result)
			if len(T.Result) == goroutineNumber {
				T.ResultChan <- T.Result
				T.Result = *new([]types.Result)
			}
		default:
			if len(T.Result) != 0 {
				T.app.logger.Info("CheckTxConcurrency----result等待超时----result", T.Result)
				T.ResultChan <- T.Result
				T.Result = *new([]types.Result)
			}
		}
	}
}

func (T *TxExecutor) Run() {
	for {
		select {
		case results := <-T.ResultChan:
			T.app.logger.Info("CheckTxConcurrency--------results", results)
			T.wg.Add(len(results))
			T.lenResponse = len(results)
			for _, result := range results {
				if result.TxVersion == "tx1" {
					var connV2 *check.AppCheck
					if T.app.ChainVersion() == 2 {
						connV2 = T.app.connCheck
					}
					go T.app.appv1.RunCheckTxV1Concurrency(result, T.app.responsePool, connV2, T.wg)
				} else if result.TxVersion == "tx2" {
					go T.app.connCheck.RunCheckTxV2Concurrency(result, T.app.responsePool, T.wg)

				} else if result.TxVersion == "tx3" {
					go T.app.connCheck.RunCheckTxV3Concurrency(result, T.app.responsePool, T.wg)
				}
			}
			T.wg.Wait()
		}
	}
}

func (T *TxExecutor) PutResponse() {
	ResponseCheckTxMap := make(map[int]types.ResponseCheckTx, 0)
	var i = 0
	for {
		select {
		case responseChanOrder := <-T.app.responsePool.ResponseOrder:
			T.app.logger.Info("CheckTxConcurrency--------responseChanOrder", responseChanOrder)
			ResponseCheckTxMap[responseChanOrder.Index] = responseChanOrder.Response
			for {
				T.app.responseChan = socket.GetResponse()
				if _, ok := ResponseCheckTxMap[i]; ok {
					T.app.logger.Info("发送的response是", ResponseCheckTxMap[i])
					T.app.responseChan <- types.ToResponseCheckTx(ResponseCheckTxMap[i])
					T.app.responseChan <- types.ToResponseFlush()
					delete(ResponseCheckTxMap, i)
					i++
					if i == T.lenResponse {

						i = 0
					}
				} else {
					break
				}
			}
		}
	}
}
