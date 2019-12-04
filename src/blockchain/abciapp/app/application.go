package app

import (
	"blockchain/abciapp/common"
	"blockchain/abciapp/service/check"
	"blockchain/abciapp/service/deliver"
	"blockchain/abciapp/service/query"
	"blockchain/abciapp/softforks"
	"blockchain/abciapp/version"
	"blockchain/algorithm"
	"blockchain/common/builderhelper"
	"blockchain/common/statedbhelper"
	"blockchain/smcrunctl/adapter"
	"blockchain/smcsdk/sdk/std"
	"blockchain/statedb"
	types2 "blockchain/types"
	"common/bcdb"
	"common/jsoniter"
	"errors"
	"fmt"
	"strings"

	appv1 "blockchain/abciapp_v1.0/app"
	"github.com/tendermint/go-crypto"

	"github.com/tendermint/abci/types"
	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"
)

//BCChainApplication object of application
type BCChainApplication struct {
	types.BaseApplication

	connQuery   *query.QueryConnection
	connCheck   *check.AppCheck
	connDeliver *deliver.AppDeliver
	db          *bcdb.GILevelDB
	logger      log.Loggerf

	// v1 app
	appv1 *appv1.BCChainApplication
}

//NewBCChainApplication create an application object
func NewBCChainApplication(config common.Config, logger log.Loggerf) *BCChainApplication {
	logger.Info("Init bcchain begin", "version", version.Version)
	db, ret := statedb.Init(config.DBName, config.DBIP, config.DBPort)
	if !ret {
		err := errors.New("init statedb error")
		logger.Fatal("Failed to startup the stateDB", "error", err)
		panic(err)
	}

	app := BCChainApplication{
		connQuery:   &query.QueryConnection{},
		connCheck:   &check.AppCheck{},
		connDeliver: &deliver.AppDeliver{},
		logger:      logger,
		db:          db,
	}

	softforks.Init()

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
		app.appv1 = appv1.NewBCChainApplication(db, logger)
	}
	logger.Info("Init bcchain end")

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

	state := statedbhelper.GetWorldAppState(0, 0)

	splitTx := strings.Split(string(tx), ".")
	if len(splitTx) == 5 {
		if splitTx[1] == "v1" && app.appv1 != nil {
			var connV2 *check.AppCheck
			if state.ChainVersion == 2 {
				connV2 = app.connCheck
			}
			res = app.appv1.CheckTx(tx, connV2)

		} else if splitTx[1] == "v2" && state.ChainVersion == 2 {
			res = app.connCheck.CheckTx(tx)
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
	return res
}

//DeliverTx deliverTx interface
func (app *BCChainApplication) DeliverTx(tx []byte) types.ResponseDeliverTx {

	var res types.ResponseDeliverTx

	state := statedbhelper.GetWorldAppState(0, 0)

	splitTx := strings.Split(string(tx), ".")
	if len(splitTx) == 5 {
		if splitTx[1] == "v1" && app.appv1 != nil {
			// if chain version never upgrade, give appv2 nil.
			var connV2 *deliver.AppDeliver
			if state.ChainVersion == 2 {
				connV2 = app.connDeliver
			}
			res = app.appv1.DeliverTx(tx, connV2)

		} else if splitTx[1] == "v2" && state.ChainVersion == 2 {
			var txBuffer map[string][]byte
			res, txBuffer = app.connDeliver.DeliverTx(tx)
			if app.appv1 != nil {
				app.appv1.CommitTx2V1(txBuffer)
			}
		} else {
			res.Code = types2.ErrLogicError
			res.Log = "invalid transaction"
		}
	} else {
		res.Code = types2.ErrLogicError
		res.Log = "invalid transaction"
	}

	res.TxHash = cmn.HexBytes(algorithm.CalcCodeHash(string(tx)))
	return res
}

//Flush flush interface
func (app *BCChainApplication) Flush(req types.RequestFlush) types.ResponseFlush {

	res := app.connDeliver.Flush(req)
	return res
}

//Commit commit interface
func (app *BCChainApplication) Commit() types.ResponseCommit {

	var res types.ResponseCommit
	state := statedbhelper.GetWorldAppState(0, 0)

	if state.ChainVersion == 0 {
		res = app.appv1.Commit()
	} else if state.ChainVersion == 2 {
		if checkGenesisChainVersion() == 0 {
			app.appv1.CommitToV2()
		}

		res = app.connDeliver.Commit()
	} else {
		panic("invalid chain version in state")
	}

	return res
}

//InitChain 初次初始化链后立马注册合约
func (app *BCChainApplication) InitChain(req types.RequestInitChain) types.ResponseInitChain {

	var res types.ResponseInitChain
	if req.ChainVersion == 0 {
		if app.appv1 == nil {
			app.appv1 = appv1.NewBCChainApplication(app.db, app.logger)
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
	state := statedbhelper.GetWorldAppState(0, 0)

	if state.ChainVersion == 0 {
		res = app.appv1.BeginBlock(req)
	} else if state.ChainVersion == 2 {
		// if chain was upgrade from v1, then invoke appv1 BeginBlockToV2 before v2 BeginBlock
		if checkGenesisChainVersion() == 0 {
			app.appv1.BeginBlockToV2(req)
		}

		var txBuffer map[string][]byte
		res, txBuffer = app.connDeliver.BeginBlock(req)
		if app.appv1 != nil {
			app.appv1.CommitTx2V1(txBuffer)
		}
	} else {
		panic("invalid chain version in state")
	}

	return res
}

//EndBlock endblock interface
func (app *BCChainApplication) EndBlock(req types.RequestEndBlock) types.ResponseEndBlock {

	var res types.ResponseEndBlock
	state := statedbhelper.GetWorldAppState(0, 1)

	if state.ChainVersion == 0 {
		res = app.appv1.EndBlock(req)
	} else if state.ChainVersion == 2 {
		// if chain was upgrade from v1, then invoke appv1 BeginBlockToV2 before v2 BeginBlock
		var txBuffer map[string][]byte
		res, txBuffer = app.connDeliver.EndBlock(req)
		if app.appv1 != nil {
			app.appv1.CommitTx2V1(txBuffer)
		}
	} else {
		panic("invalid chain version in state")
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

func checkGenesisChainVersion() int {
	value, err := statedbhelper.Get(std.KeyOfGenesisChainVersion())
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
