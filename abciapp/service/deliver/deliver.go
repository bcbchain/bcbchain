package deliver

import (
	"container/list"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	abci "github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	"github.com/bcbchain/bclib/types"
	"strconv"
)

//AppDeliver object of delivertx
type AppDeliver struct {
	logger         log.Logger
	transID        int64
	txID           int64
	blockHash      []byte
	blockHeader    abci.Header //block header time
	appState       *abci.AppState
	hashList       *list.List //存储deliver时产生的hash
	chainID        string
	sponser        types.Address            //出块者地址
	rewarder       types.Address            //出块者的奖励地址
	rewardStrategy []statedbhelper.Rewarder //当前奖励策略
	udValidator    bool
	validators     []abci.Validator
	fee            int64                    //总费用
	rewards        map[types.Address]int64  //奖励策略
	scGenesis      []*abci.SideChainGenesis // 侧链创世信息

	rp *ReceiptParser //
}

//SetLogger set logger
func (app *AppDeliver) SetLogger(logger log.Logger) {
	app.logger = logger
}

//SetChainID set chainID
func (app *AppDeliver) SetChainID(chainID string) {
	app.chainID = chainID
}

//InitChain init chain
func (app *AppDeliver) InitChain(req abci.RequestInitChain) abci.ResponseInitChain {
	return app.initChain(req)
}

//BeginBlock beginblock interface of app
func (app *AppDeliver) BeginBlock(req abci.RequestBeginBlock) (abci.ResponseBeginBlock, map[string][]byte) {
	return app.BCBeginBlock(req)
}

//DeliverTx DeliverTx interface of app
func (app *AppDeliver) DeliverTx(tx []byte) (abci.ResponseDeliverTx, map[string][]byte) {
	return app.deliverBCTx(tx)
}

func (app *AppDeliver) CleanData() error {
	return app.cleanData()
}

func (app *AppDeliver) Rollback() error {
	return app.rollback()
}

//Flush flush interface of app
func (app *AppDeliver) Flush(req abci.RequestFlush) abci.ResponseFlush {
	app.logger.Debug("Recv ABCI interface: Flush")
	return abci.ResponseFlush{}
}

func (app *AppDeliver) EndBlock(req abci.RequestEndBlock) (abci.ResponseEndBlock, map[string][]byte) {
	app.logger.Info("Recv ABCI interface: EndBlock", "height", req.Height)

	response := abci.ResponseEndBlock{}
	// call mine method if contract has declare it
	resp, txBuffer := app.mine()
	if resp.Code == types.CodeOK && len(resp.Data) != 0 {
		response.RewardAmount, _ = strconv.ParseInt(resp.Data, 10, 64)
	}

	if app.udValidator {
		response.ValidatorUpdates = app.validators
		app.udValidator = false
		app.validators = nil
	}
	if len(app.scGenesis) > 0 {
		response.SCGenesis = app.scGenesis
		app.scGenesis = nil
	}
	response.ChainVersion = app.appState.ChainVersion
	return response, txBuffer
}

// Commit commit interface of app
func (app *AppDeliver) Commit() abci.ResponseCommit {
	return app.commit()
}

// ------------- add for support v1 transaction begin ----------------

// RunDeliverTx - invoked by v1 deliverTx, if it's standard transfer method.
func (app *AppDeliver) RunDeliverTx(tx []byte, transaction types.Transaction, pubKey crypto.PubKeyEd25519) (abci.ResponseDeliverTx, map[string][]byte) {
	return app.runDeliverTx(tx, transaction, pubKey)
}

// AddDeliverHash - invoked by v1 deliverTx, add v1 deliverHash to v2 tx hash list;
//				  - it was used to calculate appHash.
func (app *AppDeliver) AddDeliverHash(deliverHash []byte) {
	app.hashList.PushBack(deliverHash)
}

// AddFee - invoked by v1 deliverTx, add v1 deliverTx fee to v2,
// 		  - for calculate total fee of block
func (app *AppDeliver) AddFee(fee int64) {
	app.fee = app.fee + fee
}

// AddRewardValues - invoked by v1 deliverTx, add rewardValues to v2,
// 				   - it was used to calculate total fee of reward account
func (app *AppDeliver) AddRewardValues(rewardValues map[string]uint64) {
	for k, v := range rewardValues {
		app.rewards[k] = app.rewards[k] + int64(v)
	}
}

// TransID - invoked by v1 deliverTx, return current transID
//		   - for commit txBuffer to v2
func (app *AppDeliver) TransID() int64 {
	return app.transID
}

// ------------- add for support v1 transaction end ----------------

// todo 开启收据解析协程。
func (app *AppDeliver) RunReceiptParser() {
	app.rp = &ReceiptParser{ // todo make chan field.
		receiptChan:  nil,
		endBlockChan: nil,
	}
	go app.rp.ReceiptsRoutine()
}
