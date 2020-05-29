package stubs

import (
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/smcapi"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	upgrade1to22 "github.com/bcbchain/bcbchain/abciapp_v1.0/contract/upgrade1to2"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/prototype"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	"github.com/bcbchain/sdk/sdk/rlp"
	"encoding/json"
	"fmt"
	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	"strconv"
)

const (
	UPGRADE1TO2_METHOD_UPGRADE = iota

	//number
	UPGRADE1TO2_METHOD_TOTAL_COUNT
)

type Upgrade1to2Stub struct {
	logger  log.Logger
	UMethod []Method
}

func NewUpgrade1to2(ctx *stubapi.InvokeContext) *smcapi.SmcApi {
	newsmcapi := smcapi.SmcApi{Sender: ctx.Sender,
		Owner:        ctx.Owner,
		ContractAcct: CalcContractAcct(ctx, prototype.UPGRADE1TO2),
		ContractAddr: &ctx.Owner.TxState.ContractAddress,
		State:        ctx.TxState,
		Note:         ctx.Note,
		Block: &smcapi.Block{
			BlockHash:       ctx.BlockHash,
			ChainID:         ctx.BlockHeader.ChainID,
			Height:          ctx.BlockHeader.Height,
			Time:            ctx.BlockHeader.Time,
			NumTxs:          ctx.BlockHeader.NumTxs,
			DataHash:        ctx.BlockHeader.DataHash,
			LastBlockHash:   ctx.BlockHeader.LastBlockID.Hash,
			LastCommitHash:  ctx.BlockHeader.LastCommitHash,
			LastAppHash:     ctx.BlockHeader.LastAppHash,
			LastFee:         ctx.BlockHeader.LastFee,
			ProposerAddress: ctx.BlockHeader.ProposerAddress,
			RewardAddress:   ctx.BlockHeader.RewardAddress,
			RandomeOfBlock:  ctx.BlockHeader.RandomeOfBlock}}

	smcapi.InitEventHandler(&newsmcapi)

	return &newsmcapi
}

func NewUpgrade1to2Stub(logger log.Logger) *Upgrade1to2Stub {
	stub := new(Upgrade1to2Stub)
	stub.logger = logger
	stub.UMethod = make([]Method, UPGRADE1TO2_METHOD_TOTAL_COUNT)
	stub.UMethod[UPGRADE1TO2_METHOD_UPGRADE].Prototype = prototype.UPGRADE1TO2Upgrade

	for i, method := range stub.UMethod {
		stub.UMethod[i].MethodID = stubapi.ConvertPrototype2ID(method.Prototype)
		logger.Info("  method()",
			"id", strconv.FormatUint(uint64(stub.UMethod[i].MethodID), 16),
			"prototype", stub.UMethod[i].Prototype)
	}
	stubapi.SetLogger(logger)
	return stub
}

func (u *Upgrade1to2Stub) Name(addr smc.Address) string {
	return prototype.UPGRADE1TO2
}

func (u *Upgrade1to2Stub) Methods(addr smc.Address) []Method {
	return u.UMethod
}

func (u *Upgrade1to2Stub) Dispatcher(items *stubapi.InvokeParams) (response stubapi.Response, bcerr bcerrors.BCError) {
	var methodInfo MethodInfo
	if err := rlp.DecodeBytes(items.Params, &methodInfo); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}

	gas, err := items.Ctx.TxState.GetGas(items.Ctx.TxState.ContractAddress, methodInfo.MethodID)
	u.logger.Debug("Dispatcher()",
		"MethodID", strconv.FormatUint(uint64(methodInfo.MethodID), 16),
		"Gas", gas)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	response.Data = "" // Don't have response data from contract method

	// Check and pay for Gas
	if response.GasUsed, response.GasPrice, response.RewardValues, bcerr = items.Ctx.CheckAndPayForGas(
		items.Ctx.Sender,
		items.Ctx.Proposer,
		items.Ctx.Rewarder,
		gas,
		items.Ctx.GasLimit); bcerr.ErrorCode != bcerrors.ErrCodeOK {
		u.logger.Error("CheckAndPayForGas() failed", "error", err)
		return
	}

	upgrade1to2 := upgrade1to22.Upgrade1to2{SmcApi: NewUpgrade1to2(items.Ctx)}

	//Receipts of Fee
	tokenbasic, _ := upgrade1to2.State.GetGenesisToken()
	receiptsOfTransactionFee(upgrade1to2.EventHandler, tokenbasic.Address, upgrade1to2.Sender.Addr, response.GasUsed*response.GasPrice, response.RewardValues)

	// Parse function parameter
	var itemsBytes = make([]([]byte), 0)
	if err = rlp.DecodeBytes(methodInfo.ParamData, &itemsBytes); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		return
	}

	switch methodInfo.MethodID {
	case u.UMethod[UPGRADE1TO2_METHOD_UPGRADE].MethodID: // todo
		u.logger.Info("Dispatcher, Calling Upgrade() Function")
		response.RequestMethod = u.UMethod[UPGRADE1TO2_METHOD_UPGRADE].Prototype

		if len(itemsBytes) != 1 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		u.logger.Debug("Input Parameter", "itemsBytes", string(itemsBytes[0]))
		response.Data, bcerr = upgrade1to2.Upgrade(string(itemsBytes[0]))
		if bcerr.ErrorCode == bcerrors.ErrCodeOK {
			addReceiptToUpgrade1to2Response(&upgrade1to2, &response)
			response.Code = stubapi.RESPONSE_CODE_RUNUPGRADE1TO2
		}

	default:
		u.logger.Error("Dispatcher(), Invalid MethodID", "MethodID", methodInfo.MethodID)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidMethod
		return
	}
	return
}

func (u *Upgrade1to2Stub) CodeHash() []byte {
	return nil
}

func addReceiptToUpgrade1to2Response(upObj *upgrade1to22.Upgrade1to2, response *stubapi.Response) {
	for index, rDTeipt := range upObj.EventHandler.GetReceipts() {
		rDTByte, _ := json.Marshal(rDTeipt)
		kvPair := common.KVPair{Key: []byte(fmt.Sprintf("%d", index)), Value: rDTByte}
		response.Tags = append(response.Tags, kvPair)
	}
}
