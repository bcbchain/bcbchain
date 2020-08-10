package app

import (
	"errors"
	"fmt"
	"github.com/bcbchain/bcbchain/abciapp/service/check"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"runtime"
	"strings"
	"sync"
)

type TxPool struct {
	TxChan chan []byte
}

func NewTxPool() *TxPool {
	return &TxPool{TxChan: make(chan []byte, 10000)}
}

type ResultPool struct {
	ResultChan chan types.Result
}

func NewResultPool() *ResultPool {
	return &ResultPool{ResultChan: make(chan types.Result, 1000)}
}

//记录Response的顺序，有序返回给socketsever
type ResponseChanOrder struct {
	Response types.ResponseCheckTx
	Index    int
}

type ResponsePool struct {
	ResponseChan  chan types.ResponseCheckTx
	ResponseOrder chan ResponseChanOrder
}

func NewResponsePool() *ResponsePool {
	return &ResponsePool{ResponseChan: make(chan types.ResponseCheckTx, 1000)}
}

type TxExecutor struct {
	app        *BCChainApplication
	Result     []types.Result
	ResultChan chan []types.Result //目前是不带缓冲区的
	wg         sync.WaitGroup
}

func (T *TxExecutor) Run() {
	for {
		select {
		case results := <-T.ResultChan:
			T.wg.Add(len(results))
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
func (T *TxExecutor) PutTxs() {
	for {

		select {
		case result := <-T.app.resultPool.ResultChan:
			T.Result = append(T.Result, result)
			if len(T.Result) == runtime.NumCPU()*2 {
				T.ResultChan <- T.Result
				T.Result = *new([]types.Result)
			}
		}
		T.Run()
	}
}

type TxParser struct {
	maxConcurrency int
	TxReceiveChan  chan [][]byte //目前是不带缓冲区的
	TxExecutor     *TxExecutor
	app            *BCChainApplication
	wg             sync.WaitGroup
}

func (T *TxParser) Run() {
	for {
		select {
		case txs := <-T.TxReceiveChan:
			T.wg.Add(len(txs))
			for i, tx := range txs {
				//result的序号需要进行标识
				result := &types.Result{}
				splitTx := strings.Split(string(tx), ".")
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
					}
				} else {
					result.Errorlog = errors.New("invalid transaction 1")
					fmt.Println("tx:", string(tx))
					fmt.Println("tx len:", len(splitTx))
				}
			}
			T.wg.Wait()
		}
	}
}

func (T *TxParser) PutRawTxs() {
	var txs [][]byte
	for {
		select {
		// 监听交易通道
		case tx := <-T.app.txPool.TxChan:
			txs = append(txs, tx)
			if len(txs) == runtime.NumCPU()*2 {
				T.TxReceiveChan <- txs
				txs = *new([][]byte)
			}
		}
	}
}

func NewTxParser(app *BCChainApplication) *TxParser {
	txParser := &TxParser{
		maxConcurrency: runtime.NumCPU() * 2,
		TxReceiveChan:  make(chan [][]byte, 0),
		TxExecutor: &TxExecutor{
			app:        app,
			Result:     make([]types.Result, 0),
			ResultChan: make(chan []types.Result, 0),
		},
		app: app,
		wg:  sync.WaitGroup{},
	}

	txParser.PutRawTxs()
	txParser.Run()
	txParser.TxExecutor.PutTxs()
	txParser.TxExecutor.Run()

	return txParser
}
