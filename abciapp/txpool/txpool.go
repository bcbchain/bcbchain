package txpool

import (
	deliverV2 "github.com/bcbchain/bcbchain/abciapp/service/deliver"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	deliverV1 "github.com/bcbchain/bcbchain/abciapp_v1.0/service/deliver"
	bctx "github.com/bcbchain/bcbchain/abciapp_v1.0/tx/tx"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bcbchain/statedb"
	"github.com/bcbchain/bclib/algorithm"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	"github.com/bcbchain/bclib/types"
	"runtime"
	"sync"
)

// 交易池接口
type TxPool interface {
	PutDeliverTxs(deliverTxs []string)
	GetTxsExecPending() []*statedb.Tx
	GetParseTx(index int) *ParseTx
	GetDeliverTxNum() int
	GetConcurrencyNum() int
	SetTransaction(transactionID int64)
	SetdeliverAppV1(*deliverV1.DeliverConnection)
	TENET() // Clearing some of the data for processing a new block
}

// 交易池对象
type txPool struct {
	deliverTxsChan  chan []string // channel ofReceive the txs from the deliver.
	deliverTxsNum   int           // Number of all txs
	batchOrder      int           // Indicates the order of the batch
	deliverTxsOrder int           // Indicates the order of the tx in the entire block

	execTxChan chan *ParseTx // Channels of the received constructed tx, sequentially from parseTxs
	parseTxs   []*ParseTx    //Store all constructed tx

	getExecTxChan     chan []*statedb.Tx      // Receive all tx from transaction
	executeTxs        map[uint8][]*statedb.Tx // Store all constructed tx by batch.
	executeTxsRWMutex *sync.RWMutex           // executeTxs has concurrent access issues, uses RWMutex to control access.

	// Indicates how many txs are in each batch;
	// 0 => how many batches of transactions there are
	bitmap map[uint8]int

	createdExecTxChan chan struct{} // Generate notifications of executable txs

	logger log.Logger // log

	transaction  *statedb.Transaction         // transaction for each block, but can be changed to store the transactionID.
	deliverAppV1 *deliverV1.DeliverConnection // AppDeliver in v1.
	deliverAppV2 *deliverV2.AppDeliver        // AppDeliver in v2.

	concurrencyNum int // Store the number of local cpu and adjust the number of concurrent threads.

	chainVersion int
}

var _ TxPool = (*txPool)(nil)

// NewTxPool Creating a new txpool
func NewTxPool(maxRoutineNum int, log log.Logger, deliverAppV2 *deliverV2.AppDeliver) TxPool {
	tp := &txPool{
		deliverTxsChan:    make(chan []string, maxRoutineNum*maxRoutineNum),
		execTxChan:        make(chan *ParseTx, maxRoutineNum),
		getExecTxChan:     make(chan []*statedb.Tx, maxRoutineNum),
		createdExecTxChan: make(chan struct{}, maxRoutineNum),
		bitmap:            make(map[uint8]int, maxRoutineNum),
		executeTxs:        make(map[uint8][]*statedb.Tx, maxRoutineNum),
		executeTxsRWMutex: new(sync.RWMutex),
		logger:            log,
		deliverAppV2:      deliverAppV2,
	}
	tp.concurrencyNum = runtime.NumCPU()

	go tp.parseDeliverTxsRoutine(maxRoutineNum)
	go tp.createExecTxRoutine()

	return tp
}

// PutDeliverTxs 区块原始交易列表
func (tp *txPool) PutDeliverTxs(deliverTxs []string) {
	tp.deliverTxsChan <- deliverTxs
}

// GetExecTxs 获取可执行交易列表，为准备妥当时阻塞
func (tp *txPool) GetTxsExecPending() []*statedb.Tx {
	return <-tp.getExecTxChan
}

// GetParseTx 获取解析后的交易信息
func (tp *txPool) GetParseTx(index int) *ParseTx {
	return tp.parseTxs[index]
}

// GetDeliverTxNum 返回当前区块交易数量
func (tp *txPool) GetDeliverTxNum() int {
	return tp.deliverTxsNum
}

// GetDeliverTxNum 返回当前最大并发线程数量
func (tp *txPool) GetConcurrencyNum() int {
	return tp.concurrencyNum
}

// SetTransaction 设置交易池中的Transaction对象
func (tp *txPool) SetTransaction(transactionID int64) {
	trans := statedbhelper.GetTransBytransID(transactionID)
	tp.transaction = trans.Transaction
}

// SetdeliverAppV1 设置交易池中的SetdeliverAppV1对象
func (tp *txPool) SetdeliverAppV1(deliverAppV1 *deliverV1.DeliverConnection) {
	tp.deliverAppV1 = deliverAppV1
}

// TENET 清空该轮的某些数据，为下一个区块作准备
func (tp *txPool) TENET() {
	tp.deliverTxsNum = 0
	tp.deliverTxsOrder = 0
	tp.batchOrder = 0
	tp.bitmap = make(map[uint8]int, 0)
	tp.executeTxs = make(map[uint8][]*statedb.Tx, 0)
	tp.parseTxs = make([]*ParseTx, 0)
}

// parseDeliverTxsRoutine parse of txs
func (tp *txPool) parseDeliverTxsRoutine(maxRoutineNum int) {
	for {
		select {
		case deliverTxs := <-tp.deliverTxsChan:
			/*
				When a new batch of transactions is received,
				some of the original storage data space is not enough,
				in order to prevent the problem of read and write errors,
				the lock limit is added.
			*/
			tp.executeTxsRWMutex.Lock()

			// Increase in the number of batches
			tp.bitmap[0]++

			//Increase the number of txs in the batch to the total number of txs
			tp.deliverTxsNum += len(deliverTxs)

			// Fill in the total number of txs for the batch
			tp.bitmap[uint8(tp.bitmap[0])] = len(deliverTxs)

			// Slicing of augmented storage parsing txs
			tp.parseTxs = append(tp.parseTxs, make([]*ParseTx, len(deliverTxs))...)

			// Slices of expanded storage construction transactions
			tp.executeTxs[uint8(tp.batchOrder)] = make([]*statedb.Tx, len(deliverTxs))

			tp.executeTxsRWMutex.Unlock()

			var mutex = new(sync.Mutex)
			routineNum := 0
			wga := NewWaitGroupAnyDone()
			for index, deliverTxStr := range deliverTxs {
				batchOrder := tp.batchOrder
				mutex.Lock()
				routineNum++
				order := tp.deliverTxsOrder
				tp.deliverTxsOrder++
				mutex.Unlock()
				go tp.parseDeliverTxRoutine(deliverTxStr, batchOrder, index, order, mutex, &routineNum, wga)

				if routineNum >= maxRoutineNum {
					wga.Wait()
				}
			}
			tp.batchOrder++
		}
	}
}

// parseDeliverTxRoutine parse of tx
func (tp *txPool) parseDeliverTxRoutine(deliverTxStr string, batchOrder int, batchTxorder int, order int,
	mutex *sync.Mutex, routineNum *int, wga *WaitGroupAnyDone) {

	// Unified Call Interface
	sender, pubKey, rawTxV1, rawTxV2 := ParseDeliverTx(deliverTxStr)

	ptx := &ParseTx{
		batchOrder:   batchOrder,                                            //批次编号
		batchTxOrder: batchTxorder,                                          //该交易在本批次的编号
		txsOrder:     order,                                                 //该交易在本区块的编号
		txStr:        deliverTxStr,                                          //交易原始字符串
		txHash:       common.HexBytes(algorithm.CalcCodeHash(deliverTxStr)), //交易hash
		sender:       sender,
		pubKey:       pubKey,
		rawTxV1:      rawTxV1,
		rawTxV2:      rawTxV2,
		carryDone:    false,
	}
	mutex.Lock()
	tp.parseTxs[order] = ptx
	*routineNum--
	mutex.Unlock()

	tp.createdExecTxChan <- struct{}{}
	wga.Done()
}

// createExecTxRoutine 生成可执行交易协程
func (tp *txPool) createExecTxRoutine() {
	for {
		select {
		case <-tp.createdExecTxChan: //有交易完成了，但是不一定按照顺序，需要严格按照顺序接收
			//开始按顺序判断是否完成了
			for _, pt := range tp.parseTxs { //保证顺序
				if pt == nil { //tp.parseTxs初始值为空
					break
				}
				if pt.CarryDone() == false {
					pt.SetCarryDone()
					tp._createExecTxRoutine(pt)
				}
			}
		}
	}
}

func (tp *txPool) _createExecTxRoutine(pTx *ParseTx) {
	if pTx.rawTxV1 != nil {
		var response = stubapi.Response{
			ErrCode: bcerrors.ErrCodeOK,
		}
		_, err := tp.deliverAppV1.GetStateDB().SetAccountNonce(pTx.sender, pTx.rawTxV1.Nonce) // 设置该账户的nonce值
		if err != nil {
			e := bcerrors.BCError{
				ErrorCode: bcerrors.ErrCodeDeliverTxInvalidNonce,
			}
			response.ErrCode = e.ErrorCode
			response.ErrLog = e.Error()
		}
		execTx := statedbhelper.NewTxConcurrency(tp.transaction.ID(), statedbhelper.RollbackTx, tp.deliverAppV1.RunExecTx, response,
			*pTx.rawTxV1, pTx.sender, tp.transaction.ID())
		if response.ErrCode != bcerrors.ErrCodeOK {
			execTx.SetPreResult(false)
		}
		tp.executeTxsRWMutex.RLock()
		tp.executeTxs[uint8(pTx.batchOrder)][pTx.batchTxOrder] = execTx
		tp.bitmap[uint8(pTx.batchOrder)+1]--
		tp.executeTxsRWMutex.RUnlock()
	} else if pTx.rawTxV2 != nil {
		var response = &types.Response{Code: types.CodeOK}
		execTx := statedbhelper.NewTxConcurrency(tp.transaction.ID(), tp.deliverAppV2.RollbackTx, tp.deliverAppV2.RunExecTx, response,
			pTx.txHash, *pTx.rawTxV2, pTx.sender, pTx.pubKey)

		//检查该交易的note是否超出最大容量
		if len(pTx.rawTxV2.Note) > types.MaxSizeNote {
			response.Code = types.ErrDeliverTx
			response.Log = "tx note is out of range"
			execTx.SetPreResult(false)
		}

		//设置交易发起者账户的nonce值
		_, err := statedbhelper.SetAccountNonceEx(pTx.sender, pTx.rawTxV2.Nonce, execTx.ID())
		if err != nil {
			response.Code = types.ErrDeliverTx
			response.Log = "SetAccountNonce failed"
			execTx.SetPreResult(false)
		}

		tp.executeTxsRWMutex.RLock()
		tp.executeTxs[uint8(pTx.batchOrder)][pTx.batchTxOrder] = execTx
		tp.bitmap[uint8(pTx.batchOrder)+1]--
		tp.executeTxsRWMutex.RUnlock()
	} else {
		panic("invalid rawTx version")
	}

	if tp.bitmap[uint8(pTx.batchOrder)+1] == 0 {
		tp.getExecTxChan <- tp.executeTxs[uint8(pTx.batchOrder)]
	}
}

// 解析构造交易后的存储的数据结构
type ParseTx struct {
	batchOrder   int //该交易所在批次的编号，从0开始
	batchTxOrder int //该交易在本批次中的顺序，从0开始
	txsOrder     int //用以表示该交易在本区块中的顺序,从0开始
	txStr        string
	txHash       common.HexBytes
	sender       types.Address
	pubKey       crypto.PubKeyEd25519
	rawTxV1      *bctx.Transaction  //v1版本的transaction
	rawTxV2      *types.Transaction //v2版本的transaction
	carryDone    bool               //标示本交易是否已经进入执行环节，
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

type WaitGroupAnyDone struct {
	bWait        bool
	bAnyDoneChan chan struct{}
	mutex        sync.Mutex
}

func NewWaitGroupAnyDone() *WaitGroupAnyDone {
	return &WaitGroupAnyDone{
		bWait:        false,
		bAnyDoneChan: make(chan struct{}),
	}
}

func (wga *WaitGroupAnyDone) Done() {
	wga.mutex.Lock()
	if wga.bWait == true {
		wga.bWait = false
		wga.bAnyDoneChan <- struct{}{}
	}
	wga.mutex.Unlock()
}

func (wga *WaitGroupAnyDone) Wait() {
	wga.mutex.Lock()
	wga.bWait = true
	wga.mutex.Unlock()

	<-wga.bAnyDoneChan //等待唤醒
}
