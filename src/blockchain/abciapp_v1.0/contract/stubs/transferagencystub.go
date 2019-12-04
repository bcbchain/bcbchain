package stubs

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/contract/transferAgency"
	. "common/bignumber_v1.0"

	"blockchain/abciapp_v1.0/smc"
	"blockchain/smcsdk/sdk/rlp"
	//"blockchain/abciapp_v1.0/contract/incentive"
	"blockchain/abciapp_v1.0/contract/smcapi"
	"blockchain/abciapp_v1.0/contract/stubapi"
	"blockchain/abciapp_v1.0/prototype"
	"github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"
)

const (
	TAC_METHODID_SETMANAGER = iota
	TAC_METHODID_SETTOKENFEE
	TAC_METHODID_TRANSFER
	TAC_METHODID_WITHDRAWFUNDS

	//number
	TAC_METHODID_TOTAL_COUNT
)

type TACStub struct {
	logger     log.Logger
	TACMethods []Method
}

func NewTAC(ctx *stubapi.InvokeContext) *smcapi.SmcApi {
	newsmcapi := smcapi.SmcApi{Sender: ctx.Sender,
		Owner:        ctx.Owner,
		ContractAcct: CreateContractAcct(ctx, prototype.TAC, transferAgency.BuyToken),
		ContractAddr: &ctx.Owner.TxState.ContractAddress,
		State:        ctx.TxState,
		Note:         ctx.Note,
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

// NewTACStub creates TokenTAC stub and initialize it with Methods
func NewTACStub(logger log.Logger) *TACStub {
	// create methodID
	var stub TACStub
	stub.logger = logger
	stub.TACMethods = make([]Method, TAC_METHODID_TOTAL_COUNT)
	stub.TACMethods[TAC_METHODID_SETMANAGER].Prototype = prototype.TACSetManager
	stub.TACMethods[TAC_METHODID_SETTOKENFEE].Prototype = prototype.TACSetTokenFee
	stub.TACMethods[TAC_METHODID_TRANSFER].Prototype = prototype.TACTransfer
	stub.TACMethods[TAC_METHODID_WITHDRAWFUNDS].Prototype = prototype.TACWithdrawFunds
	for i, method := range stub.TACMethods {
		stub.TACMethods[i].MethodID = stubapi.ConvertPrototype2ID(method.Prototype)
		logger.Info("  method()",
			"id", strconv.FormatUint(uint64(stub.TACMethods[i].MethodID), 16),
			"prototype", stub.TACMethods[i].Prototype)
	}
	stubapi.SetLogger(logger)

	return &stub
}

func (tac *TACStub) Methods(addr smc.Address) []Method {
	return tac.TACMethods
}

func (tac *TACStub) Name(addr smc.Address) string {
	return prototype.TAC
}

// Dispatcher decodes tx data that was sent by caller, and dispatch it to smart contract to execute.
// The response would be empty if there is error happens (err != nil)
func (tac *TACStub) Dispatcher(items *stubapi.InvokeParams, transID int64) (response stubapi.Response, bcerr bcerrors.BCError) {
	// Decode parameter with RLP API to get MethodInfo
	var methodInfo MethodInfo
	if err := rlp.DecodeBytes(items.Params, &methodInfo); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}

	gas, err := items.Ctx.TxState.GetGas(items.Ctx.TxState.ContractAddress, methodInfo.MethodID)
	tac.logger.Debug("Dispatcher()",
		"MethodID", strconv.FormatUint(uint64(methodInfo.MethodID), 16),
		"Gas", gas)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	response.Data = "" // Don't have response data from contract method

	tacObj := transferAgency.TransferAgency{SmcApi: NewTAC(items.Ctx)}
	//contractAccount :=
	var feePayer *stubapi.Account
	var feePayerAddr smc.Address
	if methodInfo.MethodID == tac.TACMethods[TAC_METHODID_TRANSFER].MethodID {
		feePayer = tacObj.ContractAcct
		feePayerAddr = tacObj.ContractAcct.Addr
	} else {
		feePayer = items.Ctx.Sender
		feePayerAddr = tacObj.Sender.Addr
	}

	if response.GasUsed, response.GasPrice, response.RewardValues, bcerr = items.Ctx.CheckAndPayForGas(
		feePayer,
		items.Ctx.Proposer,
		items.Ctx.Rewarder,
		gas,
		items.Ctx.GasLimit,
		transID); bcerr.ErrorCode != bcerrors.ErrCodeOK {
		tac.logger.Error("CheckAndPayForGas() failed", "error", err)
		return
	}
	//Receipts of Fee
	tokenBasic, _ := tacObj.State.GetGenesisToken()
	receiptsOfTransactionFee(tacObj.EventHandler, tokenBasic.Address, feePayerAddr, response.GasUsed*response.GasPrice, response.RewardValues)

	// Parse function paramter
	var itemsBytes = make([]([]byte), 0)
	if err = rlp.DecodeBytes(methodInfo.ParamData, &itemsBytes); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		return
	}

	// To decode method parameter with RLP API and call specified Method of smart contract depends on MethodID
	switch methodInfo.MethodID {
	case tac.TACMethods[TAC_METHODID_SETMANAGER].MethodID:
		tac.logger.Debug("Dispatcher, Calling SetManager() Function")
		response.RequestMethod = tac.TACMethods[TAC_METHODID_SETMANAGER].Prototype

		if len(itemsBytes) != 1 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		tac.logger.Debug("Input Parameter", "itemsBytes", itemsBytes)

		var addList []smc.Address
		strToList := string(itemsBytes[0][:])
		if strings.Contains(strToList, ",") {
			lists := strings.Split(strToList, ",")

			for _, address := range lists {
				addList = append(addList, address)
			}
		} else {
			addList = append(addList, strToList)
		}
		bcerr = tacObj.SetManager(addList)
		if bcerr.ErrorCode == bcerrors.ErrCodeOK {
			addReceiptToTACResponse(&tacObj, &response)
		}
		return
	case tac.TACMethods[TAC_METHODID_SETTOKENFEE].MethodID:
		tac.logger.Debug("Dispatcher, Calling SetTokenFee() Function")
		response.RequestMethod = tac.TACMethods[TAC_METHODID_SETTOKENFEE].Prototype

		if len(itemsBytes) != 1 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		tac.logger.Debug("Input Parameter", "itemsBytes", itemsBytes)

		bcerr = tacObj.SetTokenFee(string(itemsBytes[0]))
		if bcerr.ErrorCode == bcerrors.ErrCodeOK {
			addReceiptToTACResponse(&tacObj, &response)
		}
		return
	case tac.TACMethods[TAC_METHODID_TRANSFER].MethodID:
		tac.logger.Debug("Dispatcher, Calling Transfer() Function")
		response.RequestMethod = tac.TACMethods[TAC_METHODID_TRANSFER].Prototype

		if len(itemsBytes) != 3 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		tac.logger.Debug("Input Parameter", "itemsBytes", itemsBytes)
		//check targetAddress
		to := string(itemsBytes[1][:])
		tokenName := string(itemsBytes[0][:])
		amount := N(0).SetBytes(itemsBytes[2][:])
		bcerr = tacObj.Transfer(tokenName, to, amount)
		if bcerr.ErrorCode == bcerrors.ErrCodeOK {
			addReceiptToTACResponse(&tacObj, &response)
			return
		}
		return
	case tac.TACMethods[TAC_METHODID_WITHDRAWFUNDS].MethodID:
		tac.logger.Debug("Dispatcher, Calling WithdrawFunds() Function")
		response.RequestMethod = tac.TACMethods[TAC_METHODID_WITHDRAWFUNDS].Prototype

		if len(itemsBytes) != 2 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		tac.logger.Debug("Input Parameter", "itemsBytes", itemsBytes)

		tokenName := string(itemsBytes[0][:])
		_withdrawAmount := N(0).SetBytes(itemsBytes[1][:])

		bcerr = tacObj.WithdrawFunds(tokenName, _withdrawAmount)
		if bcerr.ErrorCode == bcerrors.ErrCodeOK {
			addReceiptToTACResponse(&tacObj, &response)
		}
		return

	default:
		tac.logger.Error("Dispatcher(), Invalid MethodID", "MethodID", methodInfo.MethodID)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidMethod
		return
	}
	return
}

// CodeHash gets smart contract code hash
func (tac *TACStub) CodeHash() []byte {

	return nil
}

func addReceiptToTACResponse(tacObj *transferAgency.TransferAgency, response *stubapi.Response) {
	for index, receipt := range tacObj.EventHandler.GetReceipts() {
		recByte, _ := json.Marshal(receipt)
		kvPair := common.KVPair{Key: []byte(fmt.Sprintf("%d", index)), Value: recByte}
		response.Tags = append(response.Tags, kvPair)
	}
}
