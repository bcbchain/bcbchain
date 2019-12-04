package stubs

import (
	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/contract/smcapi"
	"blockchain/abciapp_v1.0/contract/stubapi"
	tb_t "blockchain/abciapp_v1.0/contract/tokenbasic_team"
	"blockchain/abciapp_v1.0/prototype"
	"blockchain/abciapp_v1.0/smc"
	"blockchain/smcsdk/sdk/rlp"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"
)

type TBTStub struct {
	logger     log.Logger
	TBTMethods []Method
}

const (
	TBT_METHODID_WITHDRAW = iota
	//number
	TBT_METHODID_TOTAL_COUNT
)

func NewTBT(ctx *stubapi.InvokeContext) *smcapi.SmcApi {
	newsmcapi := smcapi.SmcApi{Sender: ctx.Sender,
		Owner:        ctx.Owner,
		ContractAcct: CreateContractAcct(ctx, prototype.TB_Team, ""),
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
func NewTBTStub(logger log.Logger) *TBTStub {
	// create methodID
	var stub TBTStub
	stub.logger = logger
	stub.TBTMethods = make([]Method, TBT_METHODID_TOTAL_COUNT)
	stub.TBTMethods[TBT_METHODID_WITHDRAW].Prototype = prototype.TBTWithdraw

	for i, method := range stub.TBTMethods {
		stub.TBTMethods[i].MethodID = stubapi.ConvertPrototype2ID(method.Prototype)
		logger.Info("  method()",
			"id", strconv.FormatUint(uint64(stub.TBTMethods[i].MethodID), 16),
			"prototype", stub.TBTMethods[i].Prototype)
	}
	stubapi.SetLogger(logger)

	return &stub
}

func (tbc *TBTStub) Methods(addr smc.Address) []Method {
	return tbc.TBTMethods
}

func (tbc *TBTStub) Name(addr smc.Address) string {
	return prototype.TB_Team
}

// Dispatcher dDTodes tx data that was sent by caller, and dispatch it to smart contract to exDTute.
// The response would be empty if there is error happens (err != nil)
func (tbc *TBTStub) Dispatcher(items *stubapi.InvokeParams, transID int64) (response stubapi.Response, bcerr bcerrors.BCError) {
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
	tbtObj := tb_t.TBTeam{SmcApi: NewTBT(items.Ctx)}

	// ChDTk and pay for Gas

	if response.GasUsed, response.GasPrice, response.RewardValues, bcerr = items.Ctx.CheckAndPayForGas(
		items.Ctx.Sender,
		items.Ctx.Proposer,
		items.Ctx.Rewarder,
		gas,
		items.Ctx.GasLimit,
		transID); bcerr.ErrorCode != bcerrors.ErrCodeOK {
		tbc.logger.Error("CheckAndPayForGas() failed", "error", err)
		return
	}
	//Receipts of Fee
	tokenBasic, _ := tbtObj.State.GetGenesisToken()
	receiptsOfTransactionFee(
		tbtObj.EventHandler,
		tokenBasic.Address,
		tbtObj.Sender.Addr,
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
	case tbc.TBTMethods[TBT_METHODID_WITHDRAW].MethodID:
		tbc.logger.Debug("Dispatcher, Calling RefundBets() Function")
		response.RequestMethod = tbc.TBTMethods[TBT_METHODID_WITHDRAW].Prototype

		if len(itemsBytes) != 0 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		tbc.logger.Debug("Input Parameter", "itemsBytes", itemsBytes)
		bcerr = tbtObj.Withdraw()
		if bcerr.ErrorCode == bcerrors.ErrCodeOK {
			addReceiptToTBTResponse(&tbtObj, &response)
		}
		return
	default:
		tbc.logger.Error("Dispatcher(), Invalid MethodID", "MethodID", methodInfo.MethodID)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidMethod
		return
	}
}

// CodeHash gets smart contract code hash
func (tbc *TBTStub) CodeHash() []byte {
	//TBD
	return nil
}

func addReceiptToTBTResponse(tbtObj *tb_t.TBTeam, response *stubapi.Response) {
	for index, rDTeipt := range tbtObj.EventHandler.GetReceipts() {
		rDTByte, _ := json.Marshal(rDTeipt)
		kvPair := common.KVPair{Key: []byte(fmt.Sprintf("%d", index)), Value: rDTByte}
		response.Tags = append(response.Tags, kvPair)
	}
}
