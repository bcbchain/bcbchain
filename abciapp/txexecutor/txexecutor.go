package txexecutor

import (
	deliverV2 "github.com/bcbchain/bcbchain/abciapp/service/deliver"
	"github.com/bcbchain/bcbchain/abciapp/txpool"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	deliverV1 "github.com/bcbchain/bcbchain/abciapp_v1.0/service/deliver"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bcbchain/statedb"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	"sync"
	"time"
)

type TxExecutor interface {
	GetResponse() []types.ResponseDeliverTx
	SetTransaction(transactionID int64)
	SetdeliverAppV1(*deliverV1.DeliverConnection)
}

type txExecutor struct {
	txPool        txpool.TxPool
	maxRoutineNum int
	transaction   *statedb.Transaction
	responsesChan chan []types.ResponseDeliverTx

	logger log.Logger

	deliverAppV1 *deliverV1.DeliverConnection
	deliverAppV2 *deliverV2.AppDeliver

	responseDeliverTxs []types.ResponseDeliverTx
}

var _ TxExecutor = (*txExecutor)(nil)

func NewTxExecutor(tp txpool.TxPool, l log.Logger, deliverAppV2 *deliverV2.AppDeliver) TxExecutor {
	te := &txExecutor{
		txPool:             tp,
		maxRoutineNum:      64,
		logger:             l,
		deliverAppV2:       deliverAppV2,
		responsesChan:      make(chan []types.ResponseDeliverTx),
		responseDeliverTxs: make([]types.ResponseDeliverTx, 0),
		//deliverAppV1:  deliverAppV1,
	}

	go te.execRoutine()

	return te
}

// GetResponse 获取交易执行结果
func (te *txExecutor) GetResponse() []types.ResponseDeliverTx {
	responses := <-te.responsesChan
	te.logger.Info("返回交易的时间为", time.Now())
	return responses
}

// execRoutine 交易执行协程
func (te *txExecutor) execRoutine() {
	for {
		//todo 修改变量名 GetTxsExecPending
		execTxs := te.txPool.GetExecTxs() //获取一批解析好后的交易，数量不确定
		for _, v := range execTxs {
			te.logger.Debug("测试结果", "准备发送到数据库中执行的交易ID为", v.ID())
		}

		time1 := time.Now()
		te.transaction.GoBatchExec(execTxs) //进入数据库中执行tx所带的执行函数

		te.logger.Error("测试结果", "交易计算所用的时间", time.Now().Sub(time1), "交易数量为", len(execTxs), "transactionID为", te.transaction.ID())
		//运算时间过多，需要进行优化
		time5 := time.Now()

		for index, execTx := range execTxs {
			parseTx := te.txPool.GetParseTx(index)
			if parseTx.RawTxV1() != nil {
				resDeliverTx := te.deliverAppV1.HandleResponse(
					execTx,
					parseTx.TxStr(),
					parseTx.RawTxV1(),
					execTx.Response().(stubapi.Response),
					te.deliverAppV2,
				)
				//resChan <- resDeliverTx
				te.responseDeliverTxs = append(te.responseDeliverTxs, resDeliverTx)
			} else if parseTx.RawTxV2() != nil {
				resDeliverTx := te.deliverAppV2.HandleResponse(
					parseTx.TxStr(),
					parseTx.RawTxV2(),
					execTx.Response().(*types.ResponseDeliverTx),
				)
				te.responseDeliverTxs = append(te.responseDeliverTxs, resDeliverTx)
				//resChan <- resDeliverTx
			} else {
				// TODO
			}
		}

		if len(te.responseDeliverTxs) == te.txPool.GetDeliverTxNum() {
			te.responsesChan <- te.responseDeliverTxs
			te.responseDeliverTxs = make([]types.ResponseDeliverTx, te.txPool.GetDeliverTxNum())
			te.responseDeliverTxs = make([]types.ResponseDeliverTx, 0)
		}
		te.logger.Error("测试结果", "HandleResponse总花费的时间", time.Now().Sub(time5), "总交易数量", len(execTxs))
	}
}

// execRoutineMap 交易执行协程放置到Map中
func (te *txExecutor) execRoutineMap(resChan chan<- types.ResponseDeliverTx, mutex *sync.Mutex) {
	for {
		execTxs := te.txPool.GetExecTxs() //获取一批解析好后的交易，数量不确定
		te.logger.Debug("execRoutine的时间", time.Now())
		time1 := time.Now()
		te.transaction.GoBatchExec(execTxs) //进入数据库中执行tx所带的执行函数

		te.logger.Debug("测试结果", "交易计算所用的时间", time.Now().Sub(time1), "交易数量为", len(execTxs), "transactionID为", te.transaction.ID())
		//运算时间过多，需要进行优化
		time5 := time.Now()
		go te.handleResponse(execTxs, resChan, mutex)

		te.logger.Debug("测试结果", "HandleResponse总花费的时间", time.Now().Sub(time5), "总交易数量", len(execTxs))
	}
}

func (te *txExecutor) handleResponse(execTxs []*statedb.Tx, resChan chan<- types.ResponseDeliverTx, mutex *sync.Mutex) {
	mutex.Lock()
	time6 := time.Now()
	for index, execTx := range execTxs {
		parseTx := te.txPool.GetParseTx(index)
		if parseTx.RawTxV1() != nil {
			resDeliverTx := te.deliverAppV1.HandleResponse(
				execTx,
				parseTx.TxStr(),
				parseTx.RawTxV1(),
				execTx.Response().(stubapi.Response),
				te.deliverAppV2,
			)
			resChan <- resDeliverTx
		} else if parseTx.RawTxV2() != nil {
			resDeliverTx := te.deliverAppV2.HandleResponse(
				parseTx.TxStr(),
				parseTx.RawTxV2(),
				execTx.Response().(*types.ResponseDeliverTx),
			)
			resChan <- resDeliverTx
		} else {
			// TODO
		}
	}
	te.logger.Debug("测试结果", "HandleResponse总花费的时间", time.Now().Sub(time6), "总交易数量", len(execTxs))
	mutex.Unlock()
}

// collectResponseRoutine 收集结果
func (te *txExecutor) collectResponseRoutine(resChan <-chan types.ResponseDeliverTx) {
	responses := make([]types.ResponseDeliverTx, 0)
	for {
		select {
		case response := <-resChan:
			te.logger.Debug("collectResponseRoutine的时间", time.Now())
			responses = append(responses, response)
			if len(responses) == te.txPool.GetDeliverTxNum() { //等待所有交易全部收集完毕
				te.logger.Debug("return responses的时间", time.Now())
				te.responsesChan <- responses
				responses = make([]types.ResponseDeliverTx, 0)
			}
		}
	}
}

func (te *txExecutor) SetTransaction(transactionID int64) {
	trans := statedbhelper.GetTransBytransID(transactionID)
	te.transaction = trans.Transaction
}

func (te *txExecutor) SetdeliverAppV1(deliverAppV1 *deliverV1.DeliverConnection) {
	te.deliverAppV1 = deliverAppV1
}
