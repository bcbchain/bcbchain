//tokenbasicstub

package stubs

import (
	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/contract"
	"blockchain/abciapp_v1.0/contract/stubapi"
	tokencontract "blockchain/abciapp_v1.0/contract/tokentemplet"
	"blockchain/abciapp_v1.0/prototype"
	"blockchain/abciapp_v1.0/smc"
	"blockchain/algorithm"
	"blockchain/smcsdk/sdk/rlp"
	"common/bignumber_v1.0"
	"github.com/tendermint/tmlibs/log"
	"math/big"
	"strconv"
	"strings"
	"unsafe"
)

const (
	TOKEN_METHODID_TRANSFER = iota
	TOKEN_METHODID_BATCHTRANS
	TOKEN_METHODID_ADDSUPPLY
	TOKEN_METHODID_SETOWNER
	TOKEN_METHODID_BURN
	TOKEN_METHODID_SETGASPRICE

	TOKEN_METHODID_TOTAL_COUNT
)

var _ ContractStub = (*TokenTempletStub)(nil)

type TokenTempletStub struct {
	logger    log.Logger
	TtMethods []Method
}

type BatchTransferParam struct {
	ToList []smc.Address
	Value  big.Int
}

func NewTokenTemplet(ctx *stubapi.InvokeContext) *contract.Contract {
	return &contract.Contract{Ctx: ctx}
}

// NewTokenTempletStub creates TokenTemplet stub and initialize it with Methods
func NewTokenTempletStub(logger log.Logger) *TokenTempletStub {
	//生成MethodID
	var stub TokenTempletStub
	stub.logger = logger
	stub.TtMethods = make([]Method, TOKEN_METHODID_TOTAL_COUNT)
	stub.TtMethods[TOKEN_METHODID_TRANSFER].Prototype = prototype.TbTransfer
	stub.TtMethods[TOKEN_METHODID_BATCHTRANS].Prototype = prototype.TtBatchTransfer
	stub.TtMethods[TOKEN_METHODID_ADDSUPPLY].Prototype = prototype.TtAddSupply
	stub.TtMethods[TOKEN_METHODID_BURN].Prototype = prototype.TtBurn
	stub.TtMethods[TOKEN_METHODID_SETOWNER].Prototype = prototype.TtSetOwner
	stub.TtMethods[TOKEN_METHODID_SETGASPRICE].Prototype = prototype.TtSetGasPrice

	for i, method := range stub.TtMethods {
		stub.TtMethods[i].MethodID = stubapi.ConvertPrototype2ID(method.Prototype)
		logger.Info("  method()",
			"id", strconv.FormatUint(uint64(stub.TtMethods[i].MethodID), 16),
			"prototype", stub.TtMethods[i].Prototype)
	}
	stubapi.SetLogger(logger)

	return &stub
}

func (tbs *TokenTempletStub) Methods(addr smc.Address) []Method {
	return tbs.TtMethods
}

func (tbs *TokenTempletStub) Name(addr smc.Address) string {
	return prototype.TokenTemplet
}

// Dispatcher decodes tx data that was sent by caller, and dispatch it to smart contract to execute.
// The response would be empty if there is error happens (err != nil)
func (tbs *TokenTempletStub) Dispatcher(items *stubapi.InvokeParams, transID int64) (response stubapi.Response, bcerr bcerrors.BCError) {
	// Decode parameter with RLP API to get MethodInfo
	var methodInfo MethodInfo
	if err := rlp.DecodeBytes(items.Params, &methodInfo); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}

	if bcerr = tbs.checkTokenStatus(items.Ctx, methodInfo.MethodID); bcerr.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	gas, err := items.Ctx.TxState.GetGas(items.Ctx.TxState.ContractAddress, methodInfo.MethodID)
	tbs.logger.Debug("Dispatcher()",
		"MethodID", strconv.FormatUint(uint64(methodInfo.MethodID), 16),
		"Gas", gas)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	response.Data = "" // Don't have response data from contract method
	response.Tags = nil

	// Check and pay for Gas
	if response.GasUsed, response.GasPrice, response.RewardValues, bcerr = items.Ctx.CheckAndPayForGas(
		items.Ctx.Sender,
		items.Ctx.Proposer,
		items.Ctx.Rewarder,
		gas,
		items.Ctx.GasLimit,
		transID); bcerr.ErrorCode != bcerrors.ErrCodeOK {
		tbs.logger.Error("CheckAndPayForGas() failed", "error", err)
		return
	}

	// To decode method parameter with RLP API and call specified Method of smart contract depends on MethodID
	switch methodInfo.MethodID {
	// Transfer
	case tbs.TtMethods[TOKEN_METHODID_TRANSFER].MethodID:

		tbs.logger.Debug("Dispatcher(), Calling Transfer() function")
		response.RequestMethod = tbs.TtMethods[TOKEN_METHODID_TRANSFER].Prototype

		var itemsBytes = make([]([]byte), 0)
		if err = rlp.DecodeBytes(methodInfo.ParamData, &itemsBytes); err != nil {
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		} else if len(itemsBytes) != 2 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		if len(itemsBytes[1]) == 0 {
			tbs.logger.Error("Dispatcher(), invalid parameter",
				"toaddress", itemsBytes[0],
				"value", itemsBytes[1])
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}

		// Check "to" address
		to := string(itemsBytes[0][:])
		chainID := items.Ctx.TxState.StateDB.GetChainID()
		if err = algorithm.CheckAddress(chainID, to); err != nil {
			tbs.logger.Error("Dispatcher(), invalid address of to", "error", err)
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		}

		tempContract := tokencontract.TokenTemplet{NewTokenTemplet(items.Ctx)}
		return response, tempContract.Transfer(to, *new(big.Int).SetBytes(itemsBytes[1][:]))

	case tbs.TtMethods[TOKEN_METHODID_BATCHTRANS].MethodID:
		// Batch transfer
		tbs.logger.Debug("Dispatcher(), Calling BatchTransfer() function")
		response.RequestMethod = tbs.TtMethods[TOKEN_METHODID_BATCHTRANS].Prototype

		var itemsBytes = make([]([]byte), 0)
		if err = rlp.DecodeBytes(methodInfo.ParamData, &itemsBytes); err != nil {
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		} else if len(itemsBytes) != 2 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}

		chainID := items.Ctx.TxState.StateDB.GetChainID()

		var batchTransferParam BatchTransferParam
		batchTransferParam.ToList = make([]smc.Address, 0)

		strToList := string(itemsBytes[0][:])
		if strings.Contains(strToList, ",") {
			lists := strings.Split(strToList, ",")

			for _, list := range lists {
				if err = algorithm.CheckAddress(chainID, list); err != nil {
					tbs.logger.Error("Dispatcher(), invalid address of to", "error", err)
					bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
					bcerr.ErrorDesc = err.Error()
					return
				}
				batchTransferParam.ToList = append(batchTransferParam.ToList, list)
			}

		} else {

			if err = algorithm.CheckAddress(chainID, strToList); err != nil {
				tbs.logger.Error("Dispatcher(), invalid address of to", "error", err)
				bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
				bcerr.ErrorDesc = err.Error()
				return
			}
			batchTransferParam.ToList = append(batchTransferParam.ToList, strToList)

		}
		tbs.logger.Debug("Decode Parameter", "to List", batchTransferParam.ToList)

		// The last one is value (big.Int)
		batchTransferParam.Value = *new(big.Int).SetBytes(itemsBytes[1][:])
		tbs.logger.Debug("Decode Parameter", "value", batchTransferParam.Value)

		tempContract := tokencontract.TokenTemplet{NewTokenTemplet(items.Ctx)}
		return response, tempContract.BatchTransfer(batchTransferParam.ToList, batchTransferParam.Value)

	case tbs.TtMethods[TOKEN_METHODID_ADDSUPPLY].MethodID:
		// add supply
		tbs.logger.Debug("Dispatcher(), Calling AddSupply() function")
		response.RequestMethod = tbs.TtMethods[TOKEN_METHODID_ADDSUPPLY].Prototype

		var itemsBytes = make([]([]byte), 0)
		if err = rlp.DecodeBytes(methodInfo.ParamData, &itemsBytes); err != nil {
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		}
		if len(itemsBytes[0]) == 0 {
			tbs.logger.Error("Dispatcher(), invalid parameter",
				"AddSupply", itemsBytes[0])
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		tempContract := tokencontract.TokenTemplet{NewTokenTemplet(items.Ctx)}
		return response, tempContract.AddSupply(*new(big.Int).SetBytes(itemsBytes[0][:]))

	case tbs.TtMethods[TOKEN_METHODID_BURN].MethodID:
		// burn
		tbs.logger.Debug("Dispatcher(), Calling Burn() function")
		response.RequestMethod = tbs.TtMethods[TOKEN_METHODID_BURN].Prototype

		var itemsBytes = make([]([]byte), 0)
		if err = rlp.DecodeBytes(methodInfo.ParamData, &itemsBytes); err != nil {
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		}
		if len(itemsBytes[0]) == 0 {
			tbs.logger.Error("Dispatcher(), invalid parameter",
				"burn", itemsBytes[0])
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		tempContract := tokencontract.TokenTemplet{NewTokenTemplet(items.Ctx)}
		return response, tempContract.Burn(*new(big.Int).SetBytes(itemsBytes[0][:]))

	case tbs.TtMethods[TOKEN_METHODID_SETOWNER].MethodID:
		//set owner
		tbs.logger.Debug("Dispatcher(), Calling SetOwner() function")
		response.RequestMethod = tbs.TtMethods[TOKEN_METHODID_SETOWNER].Prototype

		var itemsBytes = make([]([]byte), 0)
		if err = rlp.DecodeBytes(methodInfo.ParamData, &itemsBytes); err != nil {
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		}

		newOwner := string(itemsBytes[0][:])
		chainID := items.Ctx.TxState.StateDB.GetChainID()
		if err = algorithm.CheckAddress(chainID, newOwner); err != nil {
			tbs.logger.Error("Dispatcher(), invalid address of to", "error", err)
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		}

		tempContract := tokencontract.TokenTemplet{NewTokenTemplet(items.Ctx)}
		return response, tempContract.SetOwner(newOwner)

	case tbs.TtMethods[TOKEN_METHODID_SETGASPRICE].MethodID:
		// set gas price
		tbs.logger.Debug("Dispatcher(), Calling SetGasPrice() function")
		response.RequestMethod = tbs.TtMethods[TOKEN_METHODID_SETGASPRICE].Prototype

		var itemsBytes = make([]([]byte), 0)
		if err = rlp.DecodeBytes(methodInfo.ParamData, &itemsBytes); err != nil {
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		}
		if len(itemsBytes[0]) > int(unsafe.Sizeof(uint64(0))) { //gasprice is a parameter with uint64 type
			tbs.logger.Error("Dispatcher(), invalid parameter",
				"gasprice", itemsBytes[0])
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		tempContract := tokencontract.TokenTemplet{NewTokenTemplet(items.Ctx)}
		return response, tempContract.SetGasPrice(decode2Uint64(itemsBytes[0]))

	default:
		tbs.logger.Error("Dispatcher(), Invalid MethodID", "MethodID", methodInfo.MethodID)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidMethod
		return
	}
}

//CodeHash gets smart contract code hash
func (tbs *TokenTempletStub) CodeHash() []byte {
	//TBD
	return nil
}

func (tbs *TokenTempletStub) checkTokenStatus(ic *stubapi.InvokeContext, methodId uint32) (bcerr bcerrors.BCError) {
	token, err := ic.TxState.GetToken(ic.TxState.ContractAddress)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	if token == nil {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidAddr
		return
	}
	//only SetOwner is allowed if a token is not completed issuing
	if bignumber.Compare(token.TotalSupply, bignumber.Zero()) == 0 &&
		methodId != tbs.TtMethods[TOKEN_METHODID_SETOWNER].MethodID {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsTokenNotInit
		return
	}

	bcerr.ErrorCode = bcerrors.ErrCodeOK
	return
}
