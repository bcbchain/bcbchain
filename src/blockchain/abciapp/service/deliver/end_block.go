package deliver

import (
	"blockchain/common/statedbhelper"
	"blockchain/smcrunctl/adapter"
	"blockchain/types"
	"fmt"
)

//call mine method
func (app *AppDeliver) mine() (result *types.Response, txBuffer map[string][]byte) {
	app.logger.Debug("mine")
	result = new(types.Response)
	result.Code = types.CodeOK

	mineContracts := statedbhelper.GetMineContract(app.transID, app.txID)
	if len(mineContracts) == 0 {
		app.logger.Debug("mine contracts is not exist in stateDB")
		return
	}

	for _, v := range mineContracts {
		contract := statedbhelper.GetContract(v.Address)
		if contract == nil {
			result.Code = types.ErrLogicError
			result.Log = "can not get smart contract to call mine method when begin block"
			return
		}

		if contract.ChainVersion == 2 {
			mgr := adapter.GetInstance()

			if v.MineHeight <= app.appState.BlockHeight {
				app.txID++

				result = mgr.Mine(app.transID, app.txID, app.blockHeader, v.Address, contract.Owner)
				if result.Code != types.CodeOK {
					app.logger.Info(fmt.Sprintf("[transID=%d][txID=%d]call mine method failed", app.transID, app.txID), "error", result.Log)
					result.Code = types.CodeOK
				}
				app.logger.Info("MiningTrans", "RewardAddress", app.blockHeader.RewardAddress, "bal", result.Data)

				var stateTx []byte
				stateTx, txBuffer = statedbhelper.CommitTx(app.transID, app.txID)
				if stateTx != nil {
					app.calcDeliverHash(nil, nil, stateTx)
				}
			}
		}
	}

	return
}
