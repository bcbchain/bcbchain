package stubs

import (
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/smcapi"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	tb_f "github.com/bcbchain/bcbchain/abciapp_v1.0/contract/tokenbasic_foundation"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/prototype"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	"github.com/bcbchain/sdk/sdk/rlp"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
)

type TBFStub struct {
	logger     log.Logger
	TBFMethods []Method
}

const (
	TBF_METHODID_WITHDRAW = iota
	//number
	TBF_METHODID_TOTAL_COUNT
)

func NewTBF(ctx *stubapi.InvokeContext) *smcapi.SmcApi {
	newsmcapi := smcapi.SmcApi{Sender: ctx.Sender,
		Owner:        ctx.Owner,
		ContractAcct: CreateContractAcct(ctx, prototype.TB_Foundation, ""),
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
func NewTBFStub(logger log.Logger) *TBFStub {
	// create methodID
	var stub TBFStub
	stub.logger = logger
	stub.TBFMethods = make([]Method, TBF_METHODID_TOTAL_COUNT)
	stub.TBFMethods[TBF_METHODID_WITHDRAW].Prototype = prototype.TBFWithdraw

	for i, method := range stub.TBFMethods {
		stub.TBFMethods[i].MethodID = stubapi.ConvertPrototype2ID(method.Prototype)
		logger.Info("  method()",
			"id", strconv.FormatUint(uint64(stub.TBFMethods[i].MethodID), 16),
			"prototype", stub.TBFMethods[i].Prototype)
	}
	stubapi.SetLogger(logger)

	return &stub
}

func (tbc *TBFStub) Methods(addr smc.Address) []Method {
	return tbc.TBFMethods
}

func (tbc *TBFStub) Name(addr smc.Address) string {
	return prototype.TB_Foundation
}

// Dispatcher dDTodes tx data that was sent by caller, and dispatch it to smart contract to exDTute.
// The response would be empty if there is error happens (err != nil)
func (tbc *TBFStub) Dispatcher(items *stubapi.InvokeParams) (response stubapi.Response, bcerr bcerrors.BCError) {
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
	tbfObj := tb_f.TBFoundation{SmcApi: NewTBF(items.Ctx)}

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
	tokenBasic, _ := tbfObj.State.GetGenesisToken()
	receiptsOfTransactionFee(
		tbfObj.EventHandler,
		tokenBasic.Address,
		tbfObj.Sender.Addr,
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
	case tbc.TBFMethods[TBF_METHODID_WITHDRAW].MethodID:
		tbc.logger.Debug("Dispatcher, Calling RefundBets() Function")
		response.RequestMethod = tbc.TBFMethods[TBF_METHODID_WITHDRAW].Prototype

		if len(itemsBytes) != 0 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		tbc.logger.Debug("Input Parameter", "itemsBytes", itemsBytes)
		bcerr = tbfObj.Withdraw()
		if bcerr.ErrorCode == bcerrors.ErrCodeOK {
			addReceiptToTBFResponse(&tbfObj, &response)
		}
		return
	default:
		tbc.logger.Error("Dispatcher(), Invalid MethodID", "MethodID", methodInfo.MethodID)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidMethod
		return
	}
}

// CodeHash gets smart contract code hash
func (tbc *TBFStub) CodeHash() []byte {
	//TBD
	return nil
}

func addReceiptToTBFResponse(tbfObj *tb_f.TBFoundation, response *stubapi.Response) {
	for index, rDTeipt := range tbfObj.EventHandler.GetReceipts() {
		rDTByte, _ := json.Marshal(rDTeipt)
		kvPair := common.KVPair{Key: []byte(fmt.Sprintf("%d", index)), Value: rDTByte}
		response.Tags = append(response.Tags, kvPair)
	}
}
