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
	"sync"
	"time"
)

// 交易池接口
type TxPool interface {
	PutDeliverTxs(deliverTxs []string)

	GetExecTxs() []*statedb.Tx
	GetParseTx(index int) *ParseTx
	//GetParseTx(index int) *statedb.Tx
	GetDeliverTxNum() int
	SetTransaction(transactionID int64)
	SetdeliverAppV1(*deliverV1.DeliverConnection)
}

// 交易池对象
type txPool struct {
	deliverTxsChan chan []string //接收需要deliver的全部txs
	deliverTxsNum  int           //所有交易的数量

	execTxChan chan *ParseTx //接收构造后的tx，从parseTxs中按顺序
	parseTxs   []*ParseTx    //存储所有构造好后的tx

	getExecTxChan chan []*statedb.Tx //接收transaction中的所有tx
	executeTxs    []*statedb.Tx      //存储所有构造好后的tx

	bitmap map[uint8]int //表示有多少批交易，每一批有多少笔交易；0=>有多少批交易

	createdExecTxChan chan struct{} // 生成可执行交易通知
	leftNum           int           // 剩余未执行交易数量,已经解析了的交易

	logger log.Logger

	transaction  *statedb.Transaction         //每个区块的transaction，但可以改成置存储transactionID
	deliverAppV1 *deliverV1.DeliverConnection //v1版本的Deliver
	deliverAppV2 *deliverV2.AppDeliver        //v2版本的Deliver

	cpuNum int //存储本机cpu数量，调节并发线程数量

}

var _ TxPool = (*txPool)(nil)

func NewTxPool(maxParseRoutineNum int, l log.Logger, deliverAppV2 *deliverV2.AppDeliver) TxPool {
	//maxParseRoutineNum = 128
	tp := &txPool{
		deliverTxsChan: make(chan []string),

		execTxChan:        make(chan *ParseTx, maxParseRoutineNum),
		createdExecTxChan: make(chan struct{}, maxParseRoutineNum),

		logger: l,
		//deliverAppV1: deliverAppV1,//只有生成v1版本时才会赋值
		deliverAppV2: deliverAppV2,

		getExecTxChan: make(chan []*statedb.Tx),
	}
	tp.cpuNum = 64

	go tp.parseDeliverTxsRoutine(maxParseRoutineNum)
	go tp.createExecTxRoutine()

	return tp
}

// PutDeliverTxs 区块原始交易列表
func (tp *txPool) PutDeliverTxs(deliverTxs []string) {
	tp.deliverTxsChan <- deliverTxs
	tp.logger.Debug("PutDeliverTxs的时间", time.Now())
}

// GetExecTxs 获取可执行交易列表，为准备妥当时阻塞
func (tp *txPool) GetExecTxs() []*statedb.Tx {
	tp.logger.Debug("获取GetExecTxs交易列表的时间", time.Now())
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

// SetTransaction 设置交易池中的Transaction对象
func (tp *txPool) SetTransaction(transactionID int64) {
	trans := statedbhelper.GetTransBytransID(transactionID)
	tp.transaction = trans.Transaction
}

// SetdeliverAppV1 设置交易池中的SetdeliverAppV1对象
func (tp *txPool) SetdeliverAppV1(deliverAppV1 *deliverV1.DeliverConnection) {
	tp.deliverAppV1 = deliverAppV1
}

// parseDeliverTxsRoutine 交易解析协程
func (tp *txPool) parseDeliverTxsRoutine(maxParseRoutineNum int) {
	tp.logger.Error("parseDeliverTxsRoutine", "maxParseRoutineNum", maxParseRoutineNum)
	for {
		select {
		case deliverTxs := <-tp.deliverTxsChan:
			tp.logger.Debug("parseDeliverTxsRoutine", "len of txs", len(deliverTxs), "txs", deliverTxs)
			//todo 暂时没有用到
			//tp.deliverTxs = deliverTxs

			tp.deliverTxsNum = len(deliverTxs)
			//tp.leftNum = tp.deliverTxsNum
			if tp.deliverTxsNum%tp.cpuNum != 0 {
				tp.bitmap = make(map[uint8]int, (tp.deliverTxsNum/tp.cpuNum)+2)
				tp.bitmap[0] = (tp.deliverTxsNum / tp.cpuNum) + 1 //0=>有多少批交易
				for i := 1; i < (tp.deliverTxsNum/tp.cpuNum)+1; i++ {
					tp.bitmap[uint8(i)] = tp.cpuNum
				}
				tp.bitmap[uint8(tp.deliverTxsNum/tp.cpuNum)+1] = tp.deliverTxsNum % tp.cpuNum

			} else {
				tp.bitmap = make(map[uint8]int, (tp.deliverTxsNum/tp.cpuNum)+1)
				tp.bitmap[0] = tp.deliverTxsNum / tp.cpuNum //0=>有多少批交易
				for i := 1; i <= tp.deliverTxsNum/tp.cpuNum; i++ {
					tp.bitmap[uint8(i)] = tp.cpuNum
				}
			}

			tp.parseTxs = make([]*ParseTx, tp.deliverTxsNum)
			tp.executeTxs = make([]*statedb.Tx, tp.deliverTxsNum)
			//go tp.carryParseTxRoutine()

			//使用
			var mutex = new(sync.Mutex)
			routineNum := 0
			wga := newWaitGroupAnyDone()
			for index, deliverTxStr := range deliverTxs { //index从1开始
				//todo 进行读写保护
				mutex.Lock()
				routineNum++
				mutex.Unlock()
				go tp.parseDeliverTxRoutine(deliverTxStr, index, mutex, &routineNum, wga) //进行交易解析

				if routineNum >= maxParseRoutineNum { //设置了最大并发数量
					wga.Wait()
				}
			}
		}
	}
}

// parseDeliverTxRoutine 交易解析协程
func (tp *txPool) parseDeliverTxRoutine(deliverTxStr string, index int, mutex *sync.Mutex, routineNum *int, wga *waitGroupAnyDone) {
	//todo 修改变量名
	sender, pubKey, rawTxV1, rawTxV2 := ParseDeliverTx(deliverTxStr) //统一调用接口

	ptx := &ParseTx{
		txOrder:   index,
		txStr:     deliverTxStr,
		txHash:    common.HexBytes(algorithm.CalcCodeHash(deliverTxStr)),
		sender:    sender,
		pubKey:    pubKey,
		rawTxV1:   rawTxV1,
		rawTxV2:   rawTxV2,
		carryDone: false,
	}

	//tp.parseTxs[index] = &ptx
	mutex.Lock()
	tp.parseTxs[index] = ptx
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
		//execTx := tp.transaction.NewTx(tp.deliverAppV1.RunExecTx, response, *pTx.rawTxV1, pTx.sender, tp.transaction.ID())
		execTx := statedbhelper.NewTxConcurrency(tp.transaction.ID(), tp.deliverAppV1.RunExecTx, response, *pTx.rawTxV1, pTx.sender, tp.transaction.ID())
		tp.executeTxs[pTx.txOrder] = execTx
		tp.bitmap[uint8(pTx.txOrder/tp.cpuNum)+1]-- //表示该交易已经解析完成
	} else if pTx.rawTxV2 != nil {
		var response interface{}
		err := statedbhelper.SetAccountNonceEx(pTx.sender, pTx.rawTxV2.Nonce)
		if err != nil {
			response = types2.ResponseDeliverTx{
				Code: types.ErrDeliverTx,
				Log:  "SetAccountNonce failed",
			}
		}

		//execTx := tp.transaction.NewTx(tp.deliverAppV2.RunExecTx, response, pTx.txHash, *pTx.rawTxV2, pTx.sender, pTx.pubKey)
		execTx := statedbhelper.NewTxConcurrency(tp.transaction.ID(), tp.deliverAppV2.RunExecTx, response, pTx.txHash, *pTx.rawTxV2, pTx.sender, pTx.pubKey)
		tp.executeTxs[pTx.txOrder] = execTx
		tp.bitmap[uint8(pTx.txOrder/tp.cpuNum)+1]--
		tp.logger.Debug("测试结果", "该交易的txorder", pTx.txOrder, "该交易的txID", execTx.ID())

	} else {
		panic("invalid rawTx version")
	}

	//进行判断是否一批次交易中已全部到齐
	//然后发送给txExecutor执行
	if tp.bitmap[uint8(pTx.txOrder/tp.cpuNum)+1] == 0 {
		if pTx.txOrder/tp.cpuNum+1 == tp.bitmap[0] && tp.deliverTxsNum%tp.cpuNum != 0 { //最后一批
			tp.getExecTxChan <- tp.executeTxs[(pTx.txOrder/tp.cpuNum)*tp.cpuNum : (pTx.txOrder/tp.cpuNum)*tp.cpuNum+(tp.deliverTxsNum%tp.cpuNum)]
		} else {
			tp.getExecTxChan <- tp.executeTxs[(pTx.txOrder/tp.cpuNum)*tp.cpuNum : (pTx.txOrder/tp.cpuNum)*tp.cpuNum+tp.cpuNum]
		}

	}
}

// 解析构造交易后的存储的数据结构
type ParseTx struct {
	txOrder   int //用以表示该交易在本区块中的交易顺序,从0开始
	txStr     string
	txHash    common.HexBytes
	sender    types.Address
	pubKey    crypto.PubKeyEd25519
	rawTxV1   *bctx.Transaction  //v1版本的transaction
	rawTxV2   *types.Transaction //v2版本的transaction
	carryDone bool               //标示本交易是否已经进入执行环节，
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

	<-wga.bAnyDoneChan //等待唤醒
}
