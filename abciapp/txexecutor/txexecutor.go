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
	"runtime"
)

type TxExecutor interface {
	GetResponse() []types.ResponseDeliverTx
	SetTransaction(transactionID int64)
	SetdeliverAppV1(*deliverV1.DeliverConnection)
}

type txExecutor struct {
	tpool         txpool.TxPool
	maxRoutineNum int
	transaction   *statedb.Transaction
	responsesChan chan []types.ResponseDeliverTx

	logger log.Logger

	deliverAppV1 *deliverV1.DeliverConnection
	deliverAppV2 *deliverV2.AppDeliver
}

var _ TxExecutor = (*txExecutor)(nil)

func NewTxExecutor(tp txpool.TxPool, l log.Logger, deliverAppV2 *deliverV2.AppDeliver) TxExecutor {
	te := &txExecutor{
		tpool:         tp,
		maxRoutineNum: runtime.NumCPU(),
		logger:        l,
		deliverAppV2:  deliverAppV2,
		//deliverAppV1:  deliverAppV1,
	}

	resChan := make(chan types.ResponseDeliverTx)
	go te.execRoutine(resChan)

	return te
}

// GetResponse 获取交易执行结果
func (te *txExecutor) GetResponse() []types.ResponseDeliverTx {
	responses := <-te.responsesChan

	return responses
}

// execRoutine 交易执行协程
func (te *txExecutor) execRoutine(resChan chan<- types.ResponseDeliverTx) {
	for {
		execTxs := te.tpool.GetExecTxs(te.maxRoutineNum)
		te.transaction.GoBatchExec(execTxs)

		for index, execTx := range execTxs {
			parseTx := te.tpool.GetParseTx(index)
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
					execTx.Response().(types.ResponseDeliverTx),
				)
				resChan <- resDeliverTx
			} else {
				// TODO
			}
		}
	}
}

// collectResponseRoutine 收集结果
func (te *txExecutor) collectResponseRoutine(resChan <-chan types.ResponseDeliverTx) {
	responses := make([]types.ResponseDeliverTx, 0)
	for {
		select {
		case response := <-resChan:
			responses = append(responses, response)
			if len(responses) == te.tpool.GetDeliverTxNum() {
				te.responsesChan <- responses
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
