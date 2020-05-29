package deliver

import (
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bcbchain/smcrunctl/adapter"
	"github.com/bcbchain/sdk/sdk/std"
	"github.com/bcbchain/bclib/types"
	"bytes"
	"github.com/bcbchain/bclib/jsoniter"
	"container/list"
	"fmt"

	abci "github.com/bcbchain/bclib/tendermint/abci/types"
)

//BCBeginBlock beginblock implementation of app
func (app *AppDeliver) BCBeginBlock(req abci.RequestBeginBlock) (abci.ResponseBeginBlock, map[string][]byte) {

	transID, _ := statedbhelper.NewCommittableTransactionID()
	app.transID = transID
	app.txID = statedbhelper.NewTx(transID)
	app.logger.Info("Recv ABCI interface: BeginBlock", "height", req.Header.Height, "transID", app.transID)

	//Checking app state
	app.getAndVerifyAppState(req.Header)
	//Init app members
	app.appState.BlockHeight = req.Header.Height
	app.appState.BeginBlock = req

	app.hashList = list.New().Init()
	app.sponser = req.Header.ProposerAddress
	app.rewarder = req.Header.RewardAddress
	app.blockHash = req.Hash
	app.blockHeader = req.Header

	// Reset fee & rewards for the block
	app.fee = 0
	app.rewards = map[string]int64{}
	app.rewardStrategy = statedbhelper.GetRewardStrategy(app.transID, app.txID, app.blockHeader.Height)

	//statedbhelper.BeginBlock(transID)
	//app.logger.Debug("SetAppState", "new appState", app.appState)
	// Set the last app state due to SDK depends on it to check/get block data,
	// update it when commit
	//statedbhelper.SetWorldAppState(transID, app.txID, app.appState)

	// call smcrunsvc to initChain or updateChain smart contract
	r, txBuffer := app.initOrUpdateSMC()
	if r.Code != types.CodeOK {
		return abci.ResponseBeginBlock{Code: r.Code, Log: r.Log}, nil
	}

	return abci.ResponseBeginBlock{Code: types.CodeOK}, txBuffer
}

func (app *AppDeliver) getAndVerifyAppState(blockHeader abci.Header) {
	app.appState = statedbhelper.GetWorldAppState(app.transID, app.txID)

	app.logger.Debug("WorldAppState",
		"height", app.appState.BlockHeight,
		"LastBlockHash", app.appState.AppHash)
	// Checking on new block height
	if blockHeader.Height != app.appState.BlockHeight+1 {
		app.logger.Fatal("Block height does not match",
			"abci_height", app.appState.BlockHeight,
			"block_height", blockHeader.Height)

		panic("Block height does not match")
	}
	// Checking on app hash
	if !bytes.EqualFold(blockHeader.LastAppHash, app.appState.AppHash) {
		app.logger.Fatal("App hash does not match",
			"abci_app_hash", app.appState.AppHash,
			"block_last_app_hash", blockHeader.LastAppHash)

		panic(fmt.Sprintf("App hash does not match, req.Header.LastAppHash %x:%d, app.appState.AppHash:%x:%d",
			blockHeader.LastAppHash, blockHeader.Height, app.appState.AppHash, app.appState.BlockHeight))
	}
}

func (app *AppDeliver) initOrUpdateSMC() (result *types.Response, txBuffer map[string][]byte) {
	app.logger.Info("initOrUpdateSMC")
	result = new(types.Response)
	result.Code = types.CodeOK
	contractsWithHeight := statedbhelper.GetContractsWithHeight(app.transID, app.txID, app.appState.BlockHeight)
	if len(contractsWithHeight) == 0 {
		app.logger.Debug("No contracts need to be initialized")
		return
	}
	for _, v := range contractsWithHeight {
		app.txID = statedbhelper.NewTx(app.transID)
		contract := statedbhelper.GetContract(v.ContractAddr)
		if contract == nil {
			result.Code = types.ErrLogicError
			result.Log = "can not get smart contract to initChain or updateChain when begin block"
			return
		}
		app.logger.Debug("This contract need to init：", contract)
		mgr := adapter.GetInstance()

		result = mgr.InitOrUpdateSMC(app.transID, app.txID, app.blockHeader, v.ContractAddr, contract.Owner, v.IsUpgrade)
		if result.Code != types.CodeOK {
			app.logger.Info(fmt.Sprintf("[transID=%d][txID=%d]init/update chain failed", app.transID, app.txID), "error", result.Log)
			app.forbidContract(v.ContractAddr)
			result.Code = types.CodeOK
		}

		var stateTx []byte
		stateTx, txBuffer = statedbhelper.CommitTx(app.transID, app.txID)
		if stateTx != nil {
			app.calcDeliverHash(nil, nil, stateTx)
		}
	}
	return
}

// forbidContract - forbid contract if initChain/updateChain failed
func (app *AppDeliver) forbidContract(contractAddr types.Address) {
	contract := statedbhelper.GetContract(contractAddr)
	contract.LoseHeight = app.blockHeader.Height
	statedbhelper.SetContract(app.transID, app.txID, contract)

	v, err := statedbhelper.GetFromDB(std.KeyOfMineContracts())
	if err != nil {
		panic(err)
	}

	if len(v) == 0 {
		return
	}

	var mines []std.MineContract
	err = jsoniter.Unmarshal(v, &mines)
	if err != nil {
		panic(err)
	}

	for index, mine := range mines {
		if mine.Address == contractAddr {
			mines = append(mines[:index], mines[index+1:]...)
			break
		}
	}
	statedbhelper.SetMineContract(app.transID, app.txID, mines)
}
