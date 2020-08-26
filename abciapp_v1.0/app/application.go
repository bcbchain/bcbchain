package app

import (
	check2 "github.com/bcbchain/bcbchain/abciapp/service/check"
	deliver2 "github.com/bcbchain/bcbchain/abciapp/service/deliver"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/service/check"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/service/deliver"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/service/query"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smcrunctl"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/statedb"
	"github.com/bcbchain/bcbchain/version"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
)

type BCChainApplication struct {
	types.BaseApplication

	connQuery   *query.QueryConnection
	connCheck   *check.CheckConnection
	connDeliver *deliver.DeliverConnection
	logger      log.Loggerf
}

func NewBCChainApplication(logger log.Loggerf) *BCChainApplication {
	logger.Info("Init bcchain begin", "version", version.Version)

	app := BCChainApplication{
		connQuery:   &query.QueryConnection{},
		connCheck:   &check.CheckConnection{},
		connDeliver: &deliver.DeliverConnection{},
		logger:      logger,
	}

	app.connQuery.SetLogger(logger)
	app.connCheck.SetLogger(logger)
	app.connDeliver.SetLogger(logger)

	app.connQuery.NewStateDB()
	app.connCheck.NewStateDB()
	app.connDeliver.NewStateDB()

	// 启动数据库回调服务
	smcrunctl.StartServer(app.connDeliver.StateDB(), logger, 32332)

	//中途宕机后再次注册合约
	contractAddrArry, err := statedb.NewStateDB().GetContractAddrList()
	if err != nil {
		logger.Fatal("stateDB open failed", "error", err)
		panic(err)
	}
	if contractAddrArry != nil && len(contractAddrArry) > 0 {
		app.connCheck.InitContractDocker()
		app.connDeliver.InitContractDocker()
	}

	logger.Info("Init bcchain end")

	return &app
}
func (app *BCChainApplication) GetConnDeliver() *deliver.DeliverConnection {

	return app.connDeliver
}

func (app *BCChainApplication) Echo(req types.RequestEcho) types.ResponseEcho {

	return app.connQuery.Echo(req)
}

func (app *BCChainApplication) Info(req types.RequestInfo) types.ResponseInfo {

	return app.connQuery.Info(req)
}

func (app *BCChainApplication) SetOption(req types.RequestSetOption) types.ResponseSetOption {

	return app.connQuery.SetOption(req)
}

func (app *BCChainApplication) Query(reqQuery types.RequestQuery) types.ResponseQuery {

	return app.connQuery.Query(reqQuery)
}

func (app *BCChainApplication) CheckTx(tx []byte, connV2 *check2.AppCheck) types.ResponseCheckTx {

	return app.connCheck.CheckTx(tx, connV2)
}

//func (app *BCChainApplication) CheckTxV1Concurrency(tx []byte, wg *sync.WaitGroup, connV2 *check2.AppCheck, resultChan *types2.ResultPool, index int) {
//	defer wg.Done()
//	result := app.connCheck.CheckTxConcurrency(tx, connV2)
//	result.TxID = index
//	resultChan.ResultChan <- *result
//}
//func (app *BCChainApplication) RunCheckTxV1Concurrency(result types.Result, responsePool *types2.ResponsePool, connV2 *check2.AppCheck, wg *sync.WaitGroup) {
//
//	defer wg.Done()
//	if result.Errorlog != nil {
//		responseChanOrder := types2.ResponseChanOrder{
//			Response: types.ResponseCheckTx{
//				Code: types3.ErrCheckTx,
//				Log:  fmt.Sprint(result.Errorlog),
//			},
//			Index: result.TxID,
//		}
//		responsePool.ResponseOrder <- responseChanOrder
//		return
//	}
//	response := app.connCheck.RunCheckTxV1Concurrency(result, connV2)
//	responseChanOrder := types2.ResponseChanOrder{
//		Response: response,
//		Index:    result.TxID,
//	}
//	//responsePool.Response <- response
//	responsePool.ResponseOrder <- responseChanOrder
//	return
//}

func (app *BCChainApplication) DeliverTx(tx []byte, appV2 *deliver2.AppDeliver) types.ResponseDeliverTx {

	responseDeliverTx := app.connDeliver.DeliverTx(tx, appV2)
	// To register contract after new token or deploy contract
	if app.connDeliver.RespData != "" {
		app.connCheck.RegisterIntoContractDocker(app.connDeliver.RespData, app.connDeliver.RespCode)
		app.connDeliver.RegisterIntoContractDocker(app.connDeliver.RespData, app.connDeliver.RespCode, app.connDeliver.NameVersion)
	}

	// reset devliver response
	app.connDeliver.RespCode = 0
	app.connDeliver.RespData = ""

	return responseDeliverTx
}

func (app *BCChainApplication) Flush(req types.RequestFlush) types.ResponseFlush {

	res := app.connDeliver.Flush(req)
	return res
}

func (app *BCChainApplication) Commit() types.ResponseCommit {

	res := app.connDeliver.Commit()
	return res
}

//初次初始化链后立马注册合约
func (app *BCChainApplication) InitChain(req types.RequestInitChain) types.ResponseInitChain {

	responseInitChain := app.connDeliver.InitChain(req)
	if responseInitChain.Code == bcerrors.ErrCodeOK {
		app.connCheck.InitContractDocker()
		app.connDeliver.InitContractDocker()
	}

	return responseInitChain
}

func (app *BCChainApplication) BeginBlock(req types.RequestBeginBlock) types.ResponseBeginBlock {

	res := app.connDeliver.BeginBlock(req)
	return res
}

func (app *BCChainApplication) EndBlock(req types.RequestEndBlock) types.ResponseEndBlock {

	res := app.connDeliver.EndBlock(req)
	return res
}

// ------------- add for support new arch begin ----------------

// BeginBlockToV2 - invoked by v1 upgrade to v2 chain.
func (app *BCChainApplication) BeginBlockToV2(req types.RequestBeginBlock) {

	app.connDeliver.BeginBlockToV2(req)
}
