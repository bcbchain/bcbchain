package app

import (
	"fmt"
	"github.com/bcbchain/bcbchain/abciapp/common"
	"github.com/bcbchain/bcbchain/abciapp/service/check"
	"github.com/bcbchain/bcbchain/abciapp/service/deliver"
	"github.com/bcbchain/bcbchain/abciapp/service/query"
	"github.com/bcbchain/bcbchain/abciapp/softforks"
	"github.com/bcbchain/bcbchain/abciapp/txexecutor"
	"github.com/bcbchain/bcbchain/abciapp/txpool"
	appv1 "github.com/bcbchain/bcbchain/abciapp_v1.0/app"
	"github.com/bcbchain/bcbchain/common/builderhelper"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bcbchain/smcrunctl/adapter"
	"github.com/bcbchain/bcbchain/version"
	"github.com/bcbchain/bclib/algorithm"
	"github.com/bcbchain/bclib/jsoniter"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	types2 "github.com/bcbchain/bclib/types"
	"github.com/bcbchain/sdk/sdk/std"
	"runtime"
	"strings"
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

	txPool     txpool.TxPool
	txExecutor txexecutor.TxExecutor
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

	softforks.Init() //存疑　bcbtest

	app.connQuery.SetLogger(logger)
	app.connCheck.SetLogger(logger)
	app.connDeliver.SetLogger(logger)

	chainID := statedbhelper.GetChainID()
	app.connCheck.SetChainID(chainID)
	app.connDeliver.SetChainID(chainID)
	crypto.SetChainId(chainID)

	adapterIns := adapter.GetInstance()
	adapterIns.Init(logger, 32333)
	adapter.SetSdbCallback(statedbhelper.AdapterGetCallBack, statedbhelper.AdapterSetCallBack, builderhelper.AdapterBuildCallBack)

	if checkGenesisChainVersion() == 0 {
		app.appv1 = appv1.NewBCChainApplication(logger)
		app.txPool.SetdeliverAppV1(app.appv1.GetConnDeliver())
		app.txExecutor.SetdeliverAppV1(app.appv1.GetConnDeliver())
	}
	logger.Info("Init bcchain end")

	app.txPool = txpool.NewTxPool(runtime.NumCPU(), logger, app.connDeliver)
	app.txExecutor = txexecutor.NewTxExecutor(app.txPool, logger, app.connDeliver)
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

	res.TxHash = algorithm.CalcCodeHash(string(tx))
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

// DeliverTxs deliverTxs interface
func (app *BCChainApplication) DeliverTxs(deliverTxs []string) []types.ResponseDeliverTx {
	app.txPool.PutDeliverTxs(deliverTxs)
	return app.txExecutor.GetResponse()
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
	app.txPool.SetTransaction(app.connDeliver.TransID())
	app.txExecutor.SetTransaction(app.connDeliver.TransID())
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

	app.txPool.SetTransaction(app.connDeliver.TransID())
	app.txExecutor.SetTransaction(app.connDeliver.TransID())
	return res
}

//EndBlock endblock interface
func (app *BCChainApplication) EndBlock(req types.RequestEndBlock) types.ResponseEndBlock {

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
