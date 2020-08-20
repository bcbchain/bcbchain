package app

import (
	"errors"
	"fmt"
	"github.com/bcbchain/bcbchain/abciapp/common"
	"github.com/bcbchain/bcbchain/abciapp/service/check"
	"github.com/bcbchain/bcbchain/abciapp/service/deliver"
	"github.com/bcbchain/bcbchain/abciapp/service/query"
	types3 "github.com/bcbchain/bcbchain/abciapp/service/types"
	"github.com/bcbchain/bcbchain/abciapp/softforks"
	appv1 "github.com/bcbchain/bcbchain/abciapp_v1.0/app"
	"github.com/bcbchain/bcbchain/common/builderhelper"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bcbchain/smcrunctl/adapter"
	"github.com/bcbchain/bcbchain/statedb"
	"github.com/bcbchain/bcbchain/version"
	"github.com/bcbchain/bclib/algorithm"
	"github.com/bcbchain/bclib/jsoniter"
	abcicli "github.com/bcbchain/bclib/tendermint/abci/client"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	cmn "github.com/bcbchain/bclib/tendermint/tmlibs/common"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	types2 "github.com/bcbchain/bclib/types"
	"github.com/bcbchain/sdk/sdk/std"
	"strings"
	"sync"
)

//BCChainApplication object of application
type BCChainApplication struct {
	types.BaseApplication

	connQuery   *query.QueryConnection
	connCheck   *check.AppCheck
	connDeliver *deliver.AppDeliver
	logger      log.Loggerf

	// v1 app
	appv1 *appv1.BCChainApplication

	// current chain version
	chainVersion *int64
	// update current chain version
	updateChainVersion int64

	txPool *TxPool
	//resultPool   *types3.ResultPool
	//responsePool *types3.ResponsePool

	responseChan chan<- *types.Response
}

//NewBCChainApplication create an application object
func NewBCChainApplication(config common.Config, logger log.Loggerf) *BCChainApplication {
	logger.Info("Init bcchain begin", "version", version.Version)
	statedbhelper.Init(config.DBName, 100)

	app := BCChainApplication{
		connQuery:   &query.QueryConnection{},
		connCheck:   &check.AppCheck{},
		connDeliver: &deliver.AppDeliver{},
		logger:      logger,
	}

	app.txPool = NewTxPool(app.logger)
	//app.resultPool = types3.NewResultPool()
	//app.responsePool = types3.NewResponsePool()

	softforks.Init() //存疑　bcbtest

	app.connQuery.SetLogger(logger)
	app.connCheck.SetLogger(logger)
	app.connDeliver.SetLogger(logger)

	chainID := statedbhelper.GetChainID()
	app.connCheck.SetChainID(chainID)
	app.connDeliver.SetChainID(chainID)
	crypto.SetChainId(chainID)

	//app.connDeliver.RunReceiptParser() // todo

	adapterIns := adapter.GetInstance()
	adapterIns.Init(logger, 32333)
	adapter.SetSdbCallback(statedbhelper.AdapterGetCallBack, statedbhelper.AdapterSetCallBack, builderhelper.AdapterBuildCallBack)

	if checkGenesisChainVersion() == 0 {

		app.appv1 = appv1.NewBCChainApplication(logger)
	}
	logger.Info("Init bcchain end")

	go app.Parser()
	go app.Controller()
	go app.PutResponse()
	return &app
}

//Echo echo interface
func (app *BCChainApplication) Echo(req types.RequestEcho) types.ResponseEcho {

	res := app.connQuery.Echo(req)
	return res
}

//Info info interface
func (app *BCChainApplication) Info(req types.RequestInfo) types.ResponseInfo {

	res := app.connQuery.Info(req)
	return res
}

//SetOption set option interface
func (app *BCChainApplication) SetOption(req types.RequestSetOption) types.ResponseSetOption {

	res := app.connQuery.SetOption(req)
	return res
}

//Query query interface
func (app *BCChainApplication) Query(reqQuery types.RequestQuery) types.ResponseQuery {

	res := app.connQuery.Query(reqQuery)
	return res
}

//Query queryEx interface
func (app *BCChainApplication) QueryEx(reqQuery types.RequestQueryEx) types.ResponseQueryEx {

	res := app.connQuery.QueryEx(reqQuery)
	return res
}

//CheckTx checkTx interface
func (app *BCChainApplication) CheckTx(tx []byte) types.ResponseCheckTx {
	var res types.ResponseCheckTx

	splitTx := strings.Split(string(tx), ".")
	if len(splitTx) == 5 {
		if splitTx[1] == "v1" && app.appv1 != nil {
			var connV2 *check.AppCheck
			if app.ChainVersion() == 2 {
				connV2 = app.connCheck
			}
			res = app.appv1.CheckTx(tx, connV2)
			//go

		} else if (splitTx[1] == "v2" || splitTx[1] == "v3") && app.ChainVersion() == 2 {
			res = app.connCheck.CheckTx(tx)
			//go
		} else {
			res.Code = types2.ErrLogicError
			res.Log = "invalid transaction 1"
		}
	} else {
		res.Code = types2.ErrLogicError
		res.Log = "invalid transaction 2"
		fmt.Println("tx:", string(tx))
		fmt.Println("tx len:", len(splitTx))
	}

	res.TxHash = cmn.HexBytes(algorithm.CalcCodeHash(string(tx)))
	app.logger.Info("checkTx 处理完结果为", res)
	return res
}

//DeliverTx deliverTx interface
func (app *BCChainApplication) DeliverTx(tx []byte) types.ResponseDeliverTx {
	app.logger.Info("start DeliverTx")
	var res types.ResponseDeliverTx

	app.logger.Info("deliver 收到 tx", tx)
	splitTx := strings.Split(string(tx), ".")
	app.logger.Info("DeliverTx", "splitTx", splitTx)
	if len(splitTx) == 5 {
		if splitTx[1] == "v1" && app.appv1 != nil {
			// if chain version never upgrade, give appv2 nil.
			var connV2 *deliver.AppDeliver
			if app.ChainVersion() == 2 {
				connV2 = app.connDeliver
			}
			res = app.appv1.DeliverTx(tx, connV2)

		} else if (splitTx[1] == "v2" || splitTx[1] == "v3") && app.ChainVersion() == 2 {
			res, _ = app.connDeliver.DeliverTx(tx)

		} else {
			res.Code = types2.ErrLogicError
			res.Log = "invalid transaction"
		}
	} else {
		res.Code = types2.ErrLogicError
		res.Log = "invalid transaction"
	}

	res.TxHash = algorithm.CalcCodeHash(string(tx))
	app.logger.Info("deliver tx 结果 res", res)
	return res
}

func (app *BCChainApplication) DeliverTxConcurrency(tx []byte, v interface{}) {
	app.logger.Info("start DeliverTx")

	app.logger.Info("deliver 收到 tx", tx)

	reqRes := v.(abcicli.ReqRes)
	app.logger.Info("deliver 收到请求 reqRes", reqRes.Request)
	app.txPool.PutRawTx(tx, &reqRes)

}

//Flush flush interface
func (app *BCChainApplication) Flush(req types.RequestFlush) types.ResponseFlush {

	res := app.connDeliver.Flush(req)
	return res
}

//Commit commit interface
func (app *BCChainApplication) Commit() types.ResponseCommit {

	var res types.ResponseCommit

	if app.ChainVersion() == 0 {
		res = app.appv1.Commit()
	} else if app.ChainVersion() == 2 {
		res = app.connDeliver.Commit()
	} else {
		panic("invalid chain version in state")
	}

	if app.updateChainVersion != 0 {
		*app.chainVersion = app.updateChainVersion
		app.updateChainVersion = 0
	}

	return res
}

//InitChain 初次初始化链后立马注册合约
func (app *BCChainApplication) InitChain(req types.RequestInitChain) types.ResponseInitChain {

	var res types.ResponseInitChain
	if req.ChainVersion == 0 {
		if app.appv1 == nil {
			app.appv1 = appv1.NewBCChainApplication(app.logger)
		}

		res = app.appv1.InitChain(req)
	} else if req.ChainVersion == 2 {
		res = app.connDeliver.InitChain(req)
		app.appv1 = nil
	} else {
		res.Code = types2.ErrLogicError
		res.Log = "invalid genesis doc"
	}

	return res
}

//BeginBlock beginblock interface
func (app *BCChainApplication) BeginBlock(req types.RequestBeginBlock) types.ResponseBeginBlock {

	var res types.ResponseBeginBlock

	if app.ChainVersion() == 0 {
		res = app.appv1.BeginBlock(req)
	} else if app.ChainVersion() == 2 {
		// if chain was upgrade from v1, then invoke appv1 BeginBlockToV2 before v2 BeginBlock
		if checkGenesisChainVersion() == 0 {
			app.appv1.BeginBlockToV2(req)
		}

		res, _ = app.connDeliver.BeginBlock(req)

	} else {
		panic("invalid chain version in state")
	}

	return res
}

//EndBlock endblock interface
func (app *BCChainApplication) EndBlock(req types.RequestEndBlock) types.ResponseEndBlock {

	app.txPool.ResetTxPool(app.logger) //重置交易池
	var res types.ResponseEndBlock

	if app.ChainVersion() == 0 {
		res = app.appv1.EndBlock(req)
	} else if app.ChainVersion() == 2 {
		// if chain was upgrade from v1, then invoke appv1 BeginBlockToV2 before v2 BeginBlock
		res, _ = app.connDeliver.EndBlock(req)

	} else {
		panic("invalid chain version in state")
	}

	if app.chainVersion != nil && *app.chainVersion != res.ChainVersion {
		app.updateChainVersion = res.ChainVersion
	}
	return res
}

// CleanData clean all bcchain data when side chain genesis
func (app *BCChainApplication) CleanData() types.ResponseCleanData {
	response := types.ResponseCleanData{
		Code: 200,
		Log:  "",
	}

	if err := app.connDeliver.CleanData(); err != nil {
		response.Code = types2.ErrLogicError
		response.Log = err.Error()
	}

	return response
}

func (app *BCChainApplication) GetGenesis() types.ResponseGetGenesis {
	response := types.ResponseGetGenesis{
		Code: 200,
		Log:  "",
	}

	if data, err := app.connQuery.GetGenesis(); err != nil {
		response.Code = types2.ErrLogicError
		response.Log = err.Error()
	} else {
		response.Data = data
	}

	return response
}

func (app *BCChainApplication) Rollback() types.ResponseRollback {
	response := types.ResponseRollback{
		Code: 200,
		Log:  "",
	}

	if err := app.connDeliver.Rollback(); err != nil {
		response.Code = types2.ErrLogicError
		response.Log = err.Error()
	}

	return response
}

func checkGenesisChainVersion() int {
	value, err := statedbhelper.GetFromDB(std.KeyOfGenesisChainVersion())
	if err != nil {
		panic(err)
	}

	if len(value) == 0 {
		return 0
	}

	var genesisChainVersion int64
	err = jsoniter.Unmarshal(value, &genesisChainVersion)
	if err != nil {
		panic(err)
	}

	if genesisChainVersion == 0 {
		return 0
	} else if genesisChainVersion == 2 {
		return 2
	}

	panic("invalid genesisChainVersion")
}

func (app *BCChainApplication) ChainVersion() int64 {
	if app.chainVersion == nil {
		state := statedbhelper.GetWorldAppState(0, 0)
		app.chainVersion = new(int64)
		*app.chainVersion = state.ChainVersion
	}

	return *app.chainVersion
}

//parser 得到交易后，根据不同的版本交易，执行构造函数
func (app *BCChainApplication) Parser() {
	app.logger.Info("start Parser")
	var wg *sync.WaitGroup
	for {
		if rawTxsOrders, ok := app.txPool.GetRawTxs(); ok {
			app.logger.Info("Parser 收到交易", rawTxsOrders)
			wg.Add(len(rawTxsOrders))
			for i, txOrder := range rawTxsOrders {
				result := &types3.Result2{}
				splitTx := strings.Split(string(txOrder.RawTx), ".")
				if len(splitTx) == 5 {
					if splitTx[1] == "v1" && app.appv1 != nil {
						// if chain version never upgrade, give appv2 nil.
						var connV2 *deliver.AppDeliver
						if app.ChainVersion() == 2 {
							connV2 = app.connDeliver
						}
						res := app.appv1.DeliverTx(txOrder.RawTx, connV2)
						txOrder.ReqRes.Response = types.ToResponseDeliverTx(res)
						wg.Done()
						//go app.txPool.PutResults(app.appv1.DeliverTxV1Concurrency(txOrder, connV2, wg))
					} else if (splitTx[1] == "v2" || splitTx[1] == "v3") && app.ChainVersion() == 2 {
						go app.txPool.PutResults(app.connDeliver.DeliverTxCurrency(txOrder, wg))
					} else {
						result.ErrorLog = errors.New("invalid transaction")
						result.TxID = int64(i)
						result.ReqRes = txOrder.ReqRes
						go app.txPool.PutResults(*result)
						wg.Done()
					}
				} else {
					result.ErrorLog = errors.New("invalid transaction")
					result.TxID = int64(i)
					result.ReqRes = txOrder.ReqRes
					go app.txPool.PutResults(*result)
					wg.Done()
				}
			}
			wg.Wait()
		}
	}
}

func (app *BCChainApplication) Controller() {
	app.logger.Info("start Controller")
	for {
		if results, ok := app.txPool.GetResults(); ok {
			app.logger.Info("Controller 收到一批results", results)
			//result中存储的是do_tx所需要的参数
			//do_tx需要根据result的不同选择不同的invoke
			txs := make([]*statedb.Tx, 0)
			for _, result := range results {
				app.logger.Info("Controller 解析到的 result", result)
				switch result.TxVersion {
				case "v1": //还是走之前的版本
				case "v2":
					tx := statedbhelper.NewTxCurrency(app.connDeliver.TransID(),
						result.TxID,
						InvokeTx,
						app.txPool.BeginBlockInfo.Header,
						app.connDeliver.TransID(),
						result.TxID,
						result.TxV2Result.Pubkey.Address(statedbhelper.GetChainID()),
						result.TxV2Result.Transaction,
						result.TxV2Result.Pubkey.Bytes(),
						cmn.HexBytes(algorithm.CalcCodeHash(string(result.Tx))),
						app.txPool.BeginBlockInfo.Hash,
						app.txPool.ResponseOrder,
						result.TxOrder,
						app.logger,
						result.Tx)
					app.logger.Info("Controller 解析到的 v2 tx", tx)
					txs = append(txs, tx)
				case "v3":
					tx := statedbhelper.NewTxCurrency(app.connDeliver.TransID(),
						result.TxID,
						InvokeTx,
						app.txPool.BeginBlockInfo.Header,
						app.connDeliver.TransID(),
						result.TxID,
						result.TxV3Result.Pubkey.Address(statedbhelper.GetChainID()),
						result.TxV3Result.Transaction,
						result.TxV3Result.Pubkey.Bytes(),
						cmn.HexBytes(algorithm.CalcCodeHash(string(result.Tx))),
						app.txPool.BeginBlockInfo.Hash,
						app.txPool.ResponseOrder,
						result.TxOrder,
						app.logger,
						result.Tx)
					app.logger.Info("Controller 解析到的 v3 tx", tx)
					txs = append(txs, tx)
				}
			}
			app.logger.Info("Controller 解析到的 txs", txs)
			statedbhelper.GoBatchExec(app.connDeliver.TransID(), txs)
		}
	}
}

//large vivid receive ill plastic protect maid alone allow buyer elegant liar
func (app *BCChainApplication) PutResponse() {
	app.logger.Info("start PutResponse")
	for {
		if responsesOrders, ok := app.txPool.GetResponsesOrder(); ok {
			app.logger.Info("PutResponse 收到 responsesOrders", responsesOrders)
			for _, responseOrder := range responsesOrders {
				var resDeliverTx types.ResponseDeliverTx
				adp := adapter.GetInstance()
				if responseOrder.Response.Code != types2.CodeOK {
					app.logger.Error("docker invoke error.....", "error", responseOrder.Response.Log)
					app.logger.Debug("docker invoke error.....", "response", responseOrder.Response.String())
					statedbhelper.RollbackTx(app.connDeliver.TransID(), responseOrder.TxID)
					adp.RollbackTx(app.connDeliver.TransID(), responseOrder.TxID)
					resDeliverTx, _, totalFee := app.connDeliver.ReportInvokeFailure(responseOrder.Tx, responseOrder.Transaction, responseOrder.Response)
					resDeliverTx.Fee = uint64(totalFee)
					app.logger.Info("交易处理失败", resDeliverTx)
					responseOrder.ReqRes.Response = types.ToResponseDeliverTx(resDeliverTx)
					//return resDeliverTx
				}
				app.logger.Debug("docker invoke response.....", "code", responseOrder.Response.Code)

				// pack validators if update validator info
				if deliver.HasUpdateValidatorReceipt(responseOrder.Response.Tags) {
					app.connDeliver.PackValidators()
				}

				// pack side chain genesis info
				if t, ok := deliver.HasSideChainGenesisReceipt(responseOrder.Response.Tags); ok {
					app.connDeliver.PackSideChainGenesis(t)
				}

				//emit new summary fee  and transferFee receipts
				tags, totalFee := app.connDeliver.EmitFeeReceipts(responseOrder.Transaction, responseOrder.Response, true)

				resDeliverTx.Code = responseOrder.Response.Code
				resDeliverTx.Log = responseOrder.Response.Log
				resDeliverTx.Tags = tags
				resDeliverTx.GasLimit = uint64(responseOrder.Transaction.GasLimit)
				resDeliverTx.GasUsed = uint64(responseOrder.Response.GasUsed)
				resDeliverTx.Fee = uint64(totalFee)
				resDeliverTx.Data = responseOrder.Response.Data
				resDeliverTxStr := resDeliverTx.String()
				app.logger.Debug("deliverBCTx()", "resDeliverTx length", len(resDeliverTxStr), "resDeliverTx", resDeliverTxStr) // log value of async instance must be immutable to avoid data race

				stateTx, _ := statedbhelper.CommitTx(app.connDeliver.TransID(), responseOrder.TxID)
				app.connDeliver.CalcDeliverHash(responseOrder.Tx, &resDeliverTx, stateTx)
				app.logger.Debug("deliverBCTx() ", "stateTx length", len(stateTx), "stateTx ", string(stateTx))

				//calculate Fee
				//app.connCheck.fee = app.fee + response.Fee
				app.connDeliver.SetFee(responseOrder.Response.Fee)
				//app.logger.Debug("deliverBCTx()", "app.fee", app.fee, "app.rewards", map2String(app.rewards))
				app.responseChan <- types.ToResponseDeliverTx(resDeliverTx)
				app.logger.Debug("end deliver invoke.....")
				app.logger.Info("交易处理成功", resDeliverTx)
				responseOrder.ReqRes.Response = types.ToResponseDeliverTx(resDeliverTx)
				//return resDeliverTx, combineBuffer(nonceBuffer, txBuffer)
			}
		}
	}
}
