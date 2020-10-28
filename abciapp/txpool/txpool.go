package txpool

import (
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
	"runtime"
	"sync"
)

// 交易池接口
type TxPool interface {
	PutDeliverTxs(deliverTxs []string)

	GetTxsExecPending() []*statedb.Tx
	GetParseTx(index int) *ParseTx
	//GetParseTx(index int) *statedb.Tx
	GetDeliverTxNum() int
	GetConcurrencyNum() int
	SetTransaction(transactionID int64)
	SetdeliverAppV1(*deliverV1.DeliverConnection)
	TENET() //	清空处理该区块时的某些运行数据，为处理新区块作准备
}

// 交易池对象
type txPool struct {
	deliverTxsChan  chan []string //接收需要deliver的全部txs
	deliverTxsNum   int           //所有交易的数量
	batchOrder      int           //表示该批次的顺序
	deliverTxsOrder int           //用来表示该交易在整个区块中的顺序

	execTxChan chan *ParseTx //接收构造后的tx，从parseTxs中按顺序
	parseTxs   []*ParseTx    //存储所有构造好后的tx

	getExecTxChan       chan []*statedb.Tx      //接收transaction中的所有tx
	executeTxs          map[uint8][]*statedb.Tx //按批次存储所有构造好后的tx,有并发访问的问题
	executeTxsSemaphore *sync.RWMutex           //executeTxs有并发访问的问题，使用信号量来控制访问

	bitmap map[uint8]int //表示有多少批交易，每一批有多少笔交易；0=>有多少批交易

	createdExecTxChan chan struct{} // 生成可执行交易通知

	logger log.Logger

	transaction  *statedb.Transaction         //每个区块的transaction，但可以改成置存储transactionID
	deliverAppV1 *deliverV1.DeliverConnection //v1版本的Deliver
	deliverAppV2 *deliverV2.AppDeliver        //v2版本的Deliver

	concurrencyNum int //存储本机cpu数量，调节并发线程数量

}

var _ TxPool = (*txPool)(nil)

func NewTxPool(maxParseRoutineNum int, l log.Logger, deliverAppV2 *deliverV2.AppDeliver) TxPool {
	//maxParseRoutineNum = 128
	tp := &txPool{
		deliverTxsChan: make(chan []string),

		execTxChan:          make(chan *ParseTx, maxParseRoutineNum),
		createdExecTxChan:   make(chan struct{}, maxParseRoutineNum),
		bitmap:              make(map[uint8]int, 32),
		executeTxs:          make(map[uint8][]*statedb.Tx, 32),
		executeTxsSemaphore: new(sync.RWMutex),
		logger:              l,
		deliverAppV2:        deliverAppV2,

		getExecTxChan: make(chan []*statedb.Tx),
	}
	tp.concurrencyNum = runtime.NumCPU() * 8

	go tp.parseDeliverTxsRoutine(maxParseRoutineNum)
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
	tp.bitmap = make(map[uint8]int, 32)
	tp.executeTxs = make(map[uint8][]*statedb.Tx, 32)
	tp.parseTxs = make([]*ParseTx, 0)
}

// parseDeliverTxsRoutine 交易解析协程
func (tp *txPool) parseDeliverTxsRoutine(maxParseRoutineNum int) {
	for {
		select {
		case deliverTxs := <-tp.deliverTxsChan:
			//收到新的一批交易时，原来的某些存储数据空间不够，为了放置扩容时产生并发读写的问题，加锁限制下
			tp.executeTxsSemaphore.Lock()

			tp.bitmap[0]++                                                             //增加批次数量
			tp.deliverTxsNum += len(deliverTxs)                                        //增加该批次的交易数，并增加总交易数
			tp.bitmap[uint8(tp.bitmap[0])] = len(deliverTxs)                           //填写该批次的总交易数量
			tp.parseTxs = append(tp.parseTxs, make([]*ParseTx, len(deliverTxs))...)    //扩容存储解析交易的切片
			tp.executeTxs[uint8(tp.batchOrder)] = make([]*statedb.Tx, len(deliverTxs)) //扩容存储构造交易的切片

			tp.executeTxsSemaphore.Unlock()

			//使用
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
				go tp.parseDeliverTxRoutine(deliverTxStr, batchOrder, index, order, mutex, &routineNum, wga) //进行交易解析

				if routineNum >= maxParseRoutineNum { //设置了最大并发数量
					wga.Wait()
				}
			}
			tp.batchOrder++
		}
	}
}

// parseDeliverTxRoutine 交易解析协程
func (tp *txPool) parseDeliverTxRoutine(deliverTxStr string, batchOrder int, batchTxorder int, order int, mutex *sync.Mutex, routineNum *int, wga *WaitGroupAnyDone) {
	//todo 修改变量名
	sender, pubKey, rawTxV1, rawTxV2 := ParseDeliverTx(deliverTxStr) //统一调用接口

	ptx := &ParseTx{
		batchOrder:   batchOrder,
		batchTxOrder: batchTxorder,
		txsOrder:     order,
		txStr:        deliverTxStr,
		txHash:       common.HexBytes(algorithm.CalcCodeHash(deliverTxStr)),
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
		var response interface{}
		err := statedbhelper.SetAccountNonceEx(pTx.sender, pTx.rawTxV1.Nonce)
		if err != nil {
			response = stubapi.Response{
				Code: types.ErrDeliverTx,
				Log:  "SetAccountNonce failed",
			}
		}
		execTx := statedbhelper.NewTxConcurrency(tp.transaction.ID(), tp.deliverAppV1.RunExecTx, response, *pTx.rawTxV1, pTx.sender, tp.transaction.ID())
		tp.executeTxsSemaphore.RLock()
		tp.executeTxs[uint8(pTx.batchOrder)][pTx.batchTxOrder] = execTx
		tp.bitmap[uint8(pTx.batchOrder)+1]--
		tp.executeTxsSemaphore.RUnlock()
	} else if pTx.rawTxV2 != nil {
		var response interface{}
		err := statedbhelper.SetAccountNonceEx(pTx.sender, pTx.rawTxV2.Nonce)
		if err != nil {
			response = types2.ResponseDeliverTx{
				Code: types.ErrDeliverTx,
				Log:  "SetAccountNonce failed",
			}
		}
		execTx := statedbhelper.NewTxConcurrency(tp.transaction.ID(), tp.deliverAppV2.RunExecTx, response, pTx.txHash, *pTx.rawTxV2, pTx.sender, pTx.pubKey)
		tp.executeTxsSemaphore.RLock()
		tp.executeTxs[uint8(pTx.batchOrder)][pTx.batchTxOrder] = execTx
		tp.bitmap[uint8(pTx.batchOrder)+1]--
		tp.executeTxsSemaphore.RUnlock()
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
