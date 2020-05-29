package stubs

import (
	tb_c "github.com/bcbchain/bcbchain/abciapp_v1.0/contract/tokenbasic_cancellation"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/smcapi"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/prototype"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	"github.com/bcbchain/sdk/sdk/rlp"
	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
)

type TBCStub struct {
	logger     log.Logger
	TBCMethods []Method
}

const (
	TBC_METHODID_CANCELLATION = iota
	//number
	TBC_METHODID_TOTAL_COUNT
)

func NewTBC(ctx *stubapi.InvokeContext) *smcapi.SmcApi {
	newsmcapi := smcapi.SmcApi{Sender: ctx.Sender,
		Owner:        ctx.Owner,
		ContractAcct: CreateContractAcct(ctx, prototype.TB_Cancellation, ""),
		ContractAddr: &ctx.Owner.TxState.ContractAddress,
		State:        ctx.TxState,
		Block: &smcapi.Block{ctx.BlockHash,
			ctx.BlockHeader.ChainID,
			ctx.BlockHeader.Height,
			ctx.BlockHeader.Time,
			ctx.BlockHeader.NumTxs,
			ctx.BlockHeader.DataHash,
			ctx.BlockHeader.LastBlockID.Hash,
			ctx.BlockHeader.LastCommitHash,
			ctx.BlockHeader.LastAppHash,
			ctx.BlockHeader.LastFee,
			ctx.BlockHeader.ProposerAddress,
			ctx.BlockHeader.RewardAddress,
			ctx.BlockHeader.RandomeOfBlock}}

	smcapi.InitEventHandler(&newsmcapi)

	return &newsmcapi
}

// NewDTStub creates TokenDT stub and initialize it with Methods
func NewTBCStub(logger log.Logger) *TBCStub {
	// create methodID
	var stub TBCStub
	stub.logger = logger
	stub.TBCMethods = make([]Method, TBC_METHODID_TOTAL_COUNT)
	stub.TBCMethods[TBC_METHODID_CANCELLATION].Prototype = prototype.TBCCancel

	for i, method := range stub.TBCMethods {
		stub.TBCMethods[i].MethodID = stubapi.ConvertPrototype2ID(method.Prototype)
		logger.Info("  method()",
			"id", strconv.FormatUint(uint64(stub.TBCMethods[i].MethodID), 16),
			"prototype", stub.TBCMethods[i].Prototype)
	}
	stubapi.SetLogger(logger)

	return &stub
}

func (tbc *TBCStub) Methods(addr smc.Address) []Method {
	return tbc.TBCMethods
}

func (tbc *TBCStub) Name(addr smc.Address) string {
	return prototype.TB_Cancellation
}

// Dispatcher dDTodes tx data that was sent by caller, and dispatch it to smart contract to exDTute.
// The response would be empty if there is error happens (err != nil)
func (tbc *TBCStub) Dispatcher(items *stubapi.InvokeParams) (response stubapi.Response, bcerr bcerrors.BCError) {
	// DDTode parameter with RLP API to get MethodInfo
	var methodInfo MethodInfo
	if err := rlp.DecodeBytes(items.Params, &methodInfo); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}

	gas, err := items.Ctx.TxState.GetGas(items.Ctx.TxState.ContractAddress, methodInfo.MethodID)
	tbc.logger.Debug("Dispatcher()",
		"MethodID", strconv.FormatUint(uint64(methodInfo.MethodID), 16),
		"Gas", gas)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	response.Data = "" // Don't have response data from contract method

	// construct tbc tbcjDTt
	tbcObj := tb_c.TBCancellation{SmcApi: NewTBC(items.Ctx)}

	// ChDTk and pay for Gas

	if response.GasUsed, response.GasPrice, response.RewardValues, bcerr = items.Ctx.CheckAndPayForGas(
		items.Ctx.Sender,
		items.Ctx.Proposer,
		items.Ctx.Rewarder,
		gas,
		items.Ctx.GasLimit); bcerr.ErrorCode != bcerrors.ErrCodeOK {
		tbc.logger.Error("CheckAndPayForGas() failed", "error", err)
		return
	}
	//Receipts of Fee
	tokenBasic, _ := tbcObj.State.GetGenesisToken()
	receiptsOfTransactionFee(
		tbcObj.EventHandler,
		tokenBasic.Address,
		tbcObj.Sender.Addr,
		response.GasUsed*response.GasPrice,
		response.RewardValues,
	)

	// Parse function paramter
	var itemsBytes = make([]([]byte), 0)
	if err = rlp.DecodeBytes(methodInfo.ParamData, &itemsBytes); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		return
	}

	// To dDTode method parameter with RLP API and call spDTified Method of smart contract depends on MethodID
	switch methodInfo.MethodID {
	case tbc.TBCMethods[TBC_METHODID_CANCELLATION].MethodID:
		tbc.logger.Debug("Dispatcher, Calling RefundBets() Function")
		response.RequestMethod = tbc.TBCMethods[TBC_METHODID_CANCELLATION].Prototype

		if len(itemsBytes) != 0 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		tbc.logger.Debug("Input Parameter", "itemsBytes", itemsBytes)
		bcerr = tbcObj.Cancel()
		if bcerr.ErrorCode == bcerrors.ErrCodeOK {
			addReceiptToTBCResponse(&tbcObj, &response)
		}
		return
	default:
		tbc.logger.Error("Dispatcher(), Invalid MethodID", "MethodID", methodInfo.MethodID)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidMethod
		return
	}
	return
}

// CodeHash gets smart contract code hash
func (tbc *TBCStub) CodeHash() []byte {
	//TBD
	return nil
}

func addReceiptToTBCResponse(tbcObj *tb_c.TBCancellation, response *stubapi.Response) {
	for index, rDTeipt := range tbcObj.EventHandler.GetReceipts() {
		rDTByte, _ := json.Marshal(rDTeipt)
		kvPair := common.KVPair{Key: []byte(fmt.Sprintf("%d", index)), Value: rDTByte}
		response.Tags = append(response.Tags, kvPair)
	}
}
