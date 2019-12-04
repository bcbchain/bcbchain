package query

import (
	"blockchain/abciapp/version"
	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/go-crypto"
)

func (conn *QueryConnection) BCInfo(req abci.RequestInfo) (resInfo abci.ResponseInfo) {
	appState, err := conn.stateDB.GetWorldAppState()
	if err != nil {
		conn.logger.Error("get stateDB GetWorldAppState failed", "error", err)
	}
	if appState == nil {
		conn.logger.Info("first time to init chain and get stateDB BCINFO")
		respAppState := abci.AppState{BlockHeight: 0}

		return abci.ResponseInfo{
			Version:         version.Version,
			LastBlockHeight: 0,
			LastAppState:    abci.AppStateToByte(&respAppState),
		}
	}

	//BCInfo是bcchain每次启动后，第一个被调用的函数，在此对chainID进行设置
	chainID := conn.stateDB.GetChainID()
	crypto.SetChainId(chainID)

	return abci.ResponseInfo{
		Version:         version.Version,
		LastBlockHeight: appState.BlockHeight,
		LastAppState:    abci.AppStateToByte(appState),
	}
}
