package txpool

import (
	"container/list"
	deliverV2 "github.com/bcbchain/bcbchain/abciapp/service/deliver"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	deliverV1 "github.com/bcbchain/bcbchain/abciapp_v1.0/service/deliver"
	bctx "github.com/bcbchain/bcbchain/abciapp_v1.0/tx/tx"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bcbchain/statedb"
	"github.com/bcbchain/bclib/algorithm"
	types2 "github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	"github.com/bcbchain/bclib/types"
	"sync"
)

// 交易池接口
type TxPool interface {
	PutDeliverTxs(deliverTxs []string)
	GetExecTxs(execTxNum int) []*statedb.Tx
	GetParseTx(index int) *ParseTx
	GetDeliverTxNum() int
	SetTransaction(transactionID int64)
	SetdeliverAppV1(*deliverV1.DeliverConnection)
}

// 交易池对象
type txPool struct {
	deliverTxsChan chan []string
	deliverTxs     []string
	deliverTxsNum  int

	parseTxs []*ParseTx

	execTxChan chan *ParseTx
	execTxs    *list.List

	createdExecTxChan chan struct{} // 生成可执行交易通知
	leftNum           int           // 剩余未执行交易数量

	logger log.Logger

	transaction  *statedb.Transaction
	deliverAppV1 *deliverV1.DeliverConnection
	deliverAppV2 *deliverV2.AppDeliver
}

var _ TxPool = (*txPool)(nil)

func NewTxPool(maxParseRoutineNum int, l log.Logger, deliverAppV2 *deliverV2.AppDeliver) TxPool {
	tp := &txPool{
		deliverTxsChan: make(chan []string),

		execTxChan:        make(chan *ParseTx),
		execTxs:           list.New(),
		createdExecTxChan: make(chan struct{}),

		logger: l,
		//deliverAppV1: deliverAppV1,
		deliverAppV2: deliverAppV2,
	}

	go tp.parseDeliverTxsRoutine(maxParseRoutineNum)
	go tp.createExecTxRoutine()

	return tp
}

// PutDeliverTxs 区块原始交易列表
func (tp *txPool) PutDeliverTxs(deliverTxs []string) {
	tp.deliverTxsChan <- deliverTxs
}

// GetExecTxs 获取可执行交易列表，为准备妥当时阻塞
func (tp *txPool) GetExecTxs(execTxNum int) []*statedb.Tx {
	// 重置数量
	if tp.deliverTxsNum <= execTxNum && tp.execTxs.Len() == tp.deliverTxsNum {
		execTxNum = tp.deliverTxsNum
	} else if tp.deliverTxsNum > execTxNum && tp.execTxs.Len() == tp.leftNum {
		execTxNum = tp.leftNum
	}

	// 获取指定数量的可执行交易或者等待交易生成
	for {
		if tp.execTxs.Len() >= execTxNum {
			execTxs := make([]*statedb.Tx, execTxNum)

			index := 0
			for index < execTxNum {
				next := tp.execTxs.Front()
				if next == nil {
					break
				}

				execTx := next.Value.(*statedb.Tx)
				execTxs = append(execTxs, execTx)
				tp.execTxs.Remove(next)
				index++
			}

			tp.leftNum -= execTxNum
			return execTxs
		} else {
			<-tp.createdExecTxChan
		}
	}
}

// GetParseTx 获取解析后的交易信息
func (tp *txPool) GetParseTx(index int) *ParseTx {
	return tp.parseTxs[index]
}

// GetDeliverTxNum 返回当前区块交易数量
func (tp *txPool) GetDeliverTxNum() int {
	return tp.deliverTxsNum
}

// SetTransaction 设置交易池中的Transaction对象
func (tp *txPool) SetTransaction(transactionID int64) {
	trans := statedbhelper.GetTransBytransID(transactionID)
	tp.transaction = trans.Transaction
}

func (tp *txPool) SetdeliverAppV1(deliverAppV1 *deliverV1.DeliverConnection) {
	tp.deliverAppV1 = deliverAppV1
}

// parseDeliverTxsRoutine 交易解析协程
func (tp *txPool) parseDeliverTxsRoutine(maxParseRoutineNum int) {
	for {
		select {
		case deliverTxs := <-tp.deliverTxsChan:
			tp.deliverTxs = deliverTxs
			tp.deliverTxsNum = len(deliverTxs)
			tp.leftNum = tp.deliverTxsNum
			tp.parseTxs = make([]*ParseTx, tp.deliverTxsNum)

			go tp.carryParseTxRoutine()

			routineNum := 0
			wga := newWaitGroupAnyDone()
			for index, deliverTxStr := range deliverTxs {
				routineNum++
				go tp.parseDeliverTxRoutine(deliverTxStr, index, &routineNum, wga)

				if routineNum >= maxParseRoutineNum {
					wga.Wait()
				}
			}
		default:
			// TODO
		}
	}
}

// parseDeliverTxRoutine 交易解析协程
func (tp *txPool) parseDeliverTxRoutine(deliverTxStr string, index int, routineNum *int, wga *waitGroupAnyDone) {
	sender, pubKey, rawTxV1, rawTxV2 := ParseDeliverTx(deliverTxStr)

	ptx := ParseTx{
		txStr:     deliverTxStr,
		txHash:    common.HexBytes(algorithm.CalcCodeHash(deliverTxStr)),
		sender:    sender,
		pubKey:    pubKey,
		rawTxV1:   rawTxV1,
		rawTxV2:   rawTxV2,
		carryDone: false,
	}
	tp.parseTxs[index] = &ptx

	*routineNum--
	wga.Done()
}

// carryParseTxRoutine 解析结果搬运协程，按顺序搬运
func (tp *txPool) carryParseTxRoutine() {
	bDone := false
	for bDone != true {
		for _, pt := range tp.parseTxs {
			if pt == nil {
				bDone = false
				break
			}

			if pt.CarryDone() == false {
				pt.SetCarryDone()
				tp.execTxChan <- pt
			}
			bDone = true
		}
	}
}

// createExecTxRoutine 生成可执行交易协程
func (tp *txPool) createExecTxRoutine() {
	for {
		select {
		case pTx := <-tp.execTxChan:
			if pTx.rawTxV1 != nil {
				var response interface{}
				err := statedbhelper.SetAccountNonceEx(pTx.sender, pTx.rawTxV1.Nonce)
				if err != nil {
					response = stubapi.Response{
						Code: types.ErrDeliverTx,
						Log:  "SetAccountNonce failed",
					}
				}
				execTx := tp.transaction.NewTx(tp.deliverAppV1.RunExecTx, response, *pTx.rawTxV1, pTx.sender, tp.transaction.ID())
				tp.execTxs.PushBack(execTx)
			} else if pTx.rawTxV2 != nil {
				var response interface{}
				err := statedbhelper.SetAccountNonceEx(pTx.sender, pTx.rawTxV2.Nonce)
				if err != nil {
					response = types2.ResponseDeliverTx{
						Code: types.ErrDeliverTx,
						Log:  "SetAccountNonce failed",
					}
				}

				execTx := tp.transaction.NewTx(tp.deliverAppV2.RunExecTx, response, pTx.txHash, *pTx.rawTxV2, pTx.sender, pTx.pubKey)
				tp.execTxs.PushBack(execTx)
			} else {
				panic("invalid rawTx version")
			}
		default:
			// TODO
		}
	}
}

type ParseTx struct {
	txStr     string
	txHash    common.HexBytes
	sender    types.Address
	pubKey    crypto.PubKeyEd25519
	rawTxV1   *bctx.Transaction
	rawTxV2   *types.Transaction
	carryDone bool
}

func (pt *ParseTx) TxStr() string {
	return pt.txStr
}

func (pt *ParseTx) RawTxV1() *bctx.Transaction {
	return pt.rawTxV1
}

func (pt *ParseTx) RawTxV2() *types.Transaction {
	return pt.rawTxV2
}

func (pt *ParseTx) SetCarryDone() {
	pt.carryDone = true
}

func (pt *ParseTx) CarryDone() bool {
	return pt.carryDone
}

type waitGroupAnyDone struct {
	bWait        bool
	bAnyDoneChan chan struct{}
	mutex        sync.Mutex
}

func newWaitGroupAnyDone() *waitGroupAnyDone {
	return &waitGroupAnyDone{
		bWait:        false,
		bAnyDoneChan: make(chan struct{}),
	}
}

func (wga *waitGroupAnyDone) Done() {
	wga.mutex.Lock()
	if wga.bWait == true {
		wga.bWait = false
		wga.bAnyDoneChan <- struct{}{}
	}
	wga.mutex.Unlock()
}

func (wga *waitGroupAnyDone) Wait() {
	wga.mutex.Lock()
	wga.bWait = true
	wga.mutex.Unlock()

	<-wga.bAnyDoneChan
}
