package sdkhelper

import (
	"blockchain/smcsdk/common/gls"
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
	"blockchain/smcsdk/sdkimpl/helper"
	"blockchain/smcsdk/sdkimpl/llfunction"
	"blockchain/smcsdk/sdkimpl/llstate"
	"blockchain/smcsdk/sdkimpl/object"

	"github.com/tendermint/tmlibs/log"
)

// Init initialize sdk, contains callback functions and logger.It invoked one time in sdk life time
func Init(
	transferFunc llfunction.TransferCallBack,
	buildFunc llfunction.BuildCallBack,
	setFunc llfunction.SetCallback,
	getFunc llfunction.GetCallback,
	getBlockFunc llfunction.GetBlockCallBack,
	ibcInvoke llfunction.IBCInvoke,
	logger *log.Loggerf) {

	sdkimpl.Init(transferFunc, buildFunc, getBlockFunc, ibcInvoke, logger)
	llstate.Init(setFunc, getFunc)
}

// New create a new ISmartContract object with many parameters
func New(
	transID int64,
	txID int64,
	sender types.Address,
	payer types.Address,
	gasLimit int64,
	gasLeft int64,
	note string,
	txHash []byte,
	smcAddr types.Address,
	methodID string,
	items []types.HexBytes,
	receipts []types.KVPair) sdk.ISmartContract {

	smc := new(sdkimpl.SmartContract)

	gls.Mgr.SetValues(gls.Values{gls.SDKKey: smc}, func() {
		llState := llstate.NewLowLevelSDB(smc, transID, txID)
		smc.SetLlState(llState)

		helperObj := helper.NewHelper(smc)
		smc.SetHelper(helperObj)

		block := helper.GetCurrentBlock(smc)
		smc.SetBlock(block)

		contract := object.NewContractFromAddress(smc, smcAddr)
		origin := make([]types.Address, 1)
		origin[0] = sender

		cloneItems := make([]types.HexBytes, len(items))
		for i, item := range items {
			cloneItems[i] = item
		}
		message := object.NewMessage(smc, contract, methodID, cloneItems, sender, payer, origin, receipts)
		smc.SetMessage(message)

		tx := object.NewTx(smc, note, gasLimit, gasLeft, txHash, sender)
		smc.SetTx(tx)
	})

	return smc
}

// OriginNewMessage create new message with origin ISmartContract
func OriginNewMessage(
	origin sdk.ISmartContract,
	contract sdk.IContract,
	methodID string,
	receipts []types.KVPair) sdk.ISmartContract {

	originList := origin.Message().Origins()
	originList = append(originList, origin.Message().Contract().Address())
	message := object.NewMessage(origin,
		contract,
		methodID,
		nil,
		origin.Message().Sender().Address(),
		origin.Message().Payer().Address(),
		originList,
		receipts)

	origin.(*sdkimpl.SmartContract).SetMessage(message)

	return origin
}

// McCommit commit update data
func McCommit(transID int64) {
	sdkimpl.McInst.CommitTrans(transID)
}

// McDirtyTrans dirty data of map by transID
func McDirtyTrans(transID int64) {
	sdkimpl.McInst.DirtyTrans(transID)
}

// McDirtyTransTx dirty data of map by transID and txID
func McDirtyTransTx(transID, txID int64) {
	sdkimpl.McInst.DirtyTransTx(transID, txID)
}

// McDirtyToken dirty token of map by token address
func McDirtyToken(tokenAddr types.Address) {
	if tokenAddr == "*" {
		sdkimpl.McInst.Clear()
		return
	}

	fullKey := std.KeyOfToken(tokenAddr)
	sdkimpl.McInst.Dirty(fullKey)
}

// McDirtyContract dirty contract of map by contract address
func McDirtyContract(contractAddr types.Address) {
	if contractAddr == "*" {
		sdkimpl.McInst.Clear()
		return
	}

	fullKey := std.KeyOfContract(contractAddr)
	sdkimpl.McInst.Dirty(fullKey)
}
