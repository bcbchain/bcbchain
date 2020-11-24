package txexecutor

import (
	"fmt"
	deliverV2 "github.com/bcbchain/bcbchain/abciapp/service/deliver"
	"github.com/bcbchain/bcbchain/abciapp/txpool"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	deliverV1 "github.com/bcbchain/bcbchain/abciapp_v1.0/service/deliver"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bcbchain/statedb"
	"github.com/bcbchain/bclib/algorithm"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	types2 "github.com/bcbchain/bclib/types"
	"runtime"
	"strings"
)

const (
	genesisOrgID = "orgJgaGConUyK81zibntUBjQ33PKctpk1K1G"
)

type TxExecutor interface {
	GetResponse() *types.ResponseDeliverTx
	SetTransaction(transactionID int64)
	SetDeliverAppV1(*deliverV1.DeliverConnection)

	TENET()
}

type txExecutor struct {
	txPool        txpool.TxPool
	transaction   *statedb.Transaction
	responsesChan chan types.ResponseDeliverTx

	handleTxsNum int //记录已经处理过的是交易的数量，处理完就可以发送给tendermint
	logger       log.Logger

	deliverAppV1 *deliverV1.DeliverConnection
	deliverAppV2 *deliverV2.AppDeliver
}

var _ TxExecutor = (*txExecutor)(nil)

func NewTxExecutor(tp txpool.TxPool, l log.Logger, deliverAppV2 *deliverV2.AppDeliver) TxExecutor {
	te := &txExecutor{
		txPool:        tp,
		logger:        l,
		deliverAppV2:  deliverAppV2,
		responsesChan: make(chan types.ResponseDeliverTx, runtime.NumCPU()*2),
	}

	go te.execRoutine()

	return te
}

// GetResponse 获取交易执行结果
func (te *txExecutor) GetResponse() *types.ResponseDeliverTx {
	select {
	case responses := <-te.responsesChan:
		return &responses
	default:
		return nil
	}
}

// execRoutine 交易执行协程
func (te *txExecutor) execRoutine() {
	for {
		execTxs := te.txPool.GetTxsExecPending() //获取一批解析好后的交易，数量不确定

		if te.haveV1Transaction(execTxs) == true {
			// 存在v1版本交易时，当前分片按照串行方式执行
			te.GoBatchExec(execTxs, false)
		} else if te.haveSideChainTx(execTxs) == true {
			// 存在v2版本跨链交易时，当前分片按照串行方式执行
			te.GoBatchExec(execTxs, false)
		} else {
			te.GoBatchExec(execTxs, true)
		}

	}
}

// GoBatchExec 如果需要并发执行，concurrency传入true，否则传入false
func (te *txExecutor) GoBatchExec(execTxs []*statedb.Tx, concurrency bool) {
	if concurrency {
		te.transaction.GoBatchExec(execTxs) //进入数据库中执行tx所带的执行函数

		for _, execTx := range execTxs {
			parseTx := te.txPool.GetParseTx(te.handleTxsNum)
			te.handleTxsNum++
			te.handleResponse(execTx, parseTx)
		}
	} else {
		for _, execTx := range execTxs {
			tempExecTxs := make([]*statedb.Tx, 0)
			tempExecTxs = append(tempExecTxs, execTx)
			te.transaction.GoBatchExec(tempExecTxs)

			parseTx := te.txPool.GetParseTx(te.handleTxsNum)
			te.handleTxsNum++
			te.handleResponse(execTx, parseTx)
		}
	}

}

func (te *txExecutor) handleResponse(execTx *statedb.Tx, parseTx *txpool.ParseTx) {
	if parseTx.RawTxV1() != nil {
		var connV2 *deliverV2.AppDeliver
		if txpool.ChainVerison == 2 {
			connV2 = te.deliverAppV2
		}
		resDeliverTx := te.deliverAppV1.HandleResponse(
			execTx,
			parseTx.TxStr(),
			parseTx.RawTxV1(),
			execTx.Response().(stubapi.Response),
			connV2,
		)
		resDeliverTx.TxHash = algorithm.CalcCodeHash(parseTx.TxStr())
		te.responsesChan <- resDeliverTx
	} else if parseTx.RawTxV2() != nil {
		resDeliverTx := te.deliverAppV2.HandleResponse(
			execTx,
			parseTx.TxStr(),
			parseTx.RawTxV2(),
			execTx.Response().(*types2.Response),
		)
		resDeliverTx.TxHash = algorithm.CalcCodeHash(parseTx.TxStr())
		te.responsesChan <- resDeliverTx
	}
}

func (te *txExecutor) SetTransaction(transactionID int64) {
	trans := statedbhelper.GetTransBytransID(transactionID)
	te.transaction = trans.Transaction
}

func (te *txExecutor) SetDeliverAppV1(deliverAppV1 *deliverV1.DeliverConnection) {
	te.deliverAppV1 = deliverAppV1
}

func (te *txExecutor) TENET() {
	state := statedbhelper.GetWorldAppState(0, 0)
	txpool.ChainVerison = int(state.ChainVersion)
	te.handleTxsNum = 0
	te.txPool.TENET()
}

func (te *txExecutor) haveV1Transaction(execTxs []*statedb.Tx) bool {
	tempHandleTxsNum := te.handleTxsNum
	for range execTxs {
		parseTx := te.txPool.GetParseTx(tempHandleTxsNum)
		tempHandleTxsNum++
		if parseTx.RawTxV1() != nil {
			return true
		}
	}

	return false
}

func (te *txExecutor) haveSideChainTx(execTxs []*statedb.Tx) bool {
	tempHandleTxsNum := te.handleTxsNum
	contractAddr := statedbhelper.GetContractsByName(0, 0, "ibc", genesisOrgID)
	for range execTxs {
		parseTx := te.txPool.GetParseTx(tempHandleTxsNum)
		tempHandleTxsNum++
		messagesContract := strings.Split(fmt.Sprint(parseTx.RawTxV2().Messages[0].Contract), ".")
		if parseTx.RawTxV2() != nil && messagesContract[0] == genesisOrgID && messagesContract[len(messagesContract)-1] == contractAddr[len(contractAddr)-1] {
			return true
		}
	}

	return false
}
