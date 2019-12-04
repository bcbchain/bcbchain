//tokenbasicstub

package stubs

import (
	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/contract"
	"blockchain/abciapp_v1.0/contract/stubapi"
	base "blockchain/abciapp_v1.0/contract/tokenbasic"
	"blockchain/abciapp_v1.0/prototype"
	"blockchain/abciapp_v1.0/smc"
	"blockchain/algorithm"
	"blockchain/smcsdk/sdk/rlp"
	"github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"
	"math/big"
	"strconv"
	"unsafe"
)

//The definition of Methods index
const (
	TB_METHODID_TRANSFER = iota
	TB_METHODID_SETGASPRICE
	TB_METHODID_SETGASBASEPRICE

	// Amount of methods
	TB_METHODID_TOTALCOUNT
)

// MethodInfo defines the structure of methodID and its parameter
type MethodInfo struct {
	MethodID  uint32
	ParamData []byte
}

// Method defines the structure of MethodID and Prototype
type Method struct {
	MethodID  uint32
	Gas       int64
	Prototype string
}

// TransferParam defines the parameter structure of Transfer method
type TransferParam struct {
	To    common.HexBytes
	Value big.Int
}

var _ ContractStub = (*TokenBasicStub)(nil)

type TokenBasicStub struct {
	logger    log.Logger
	TbMethods []Method
}

func NewTokenBasic(ctx *stubapi.InvokeContext) *contract.Contract {
	return &contract.Contract{Ctx: ctx}
}

//TokenBasicStub creates TokenBasic stub and initialize it with Methods
func NewTokenBasicStub(logger log.Logger) *TokenBasicStub {

	var stub TokenBasicStub
	stub.logger = logger
	stub.TbMethods = make([]Method, TB_METHODID_TOTALCOUNT)
	stub.TbMethods[TB_METHODID_TRANSFER].Prototype = prototype.TbTransfer
	stub.TbMethods[TB_METHODID_SETGASPRICE].Prototype = prototype.TbSetGasPrice
	stub.TbMethods[TB_METHODID_SETGASBASEPRICE].Prototype = prototype.TbSetGasBasePrice
	for i, method := range stub.TbMethods {
		stub.TbMethods[i].MethodID = stubapi.ConvertPrototype2ID(method.Prototype)
		logger.Info("  method",
			"id", strconv.FormatUint(uint64(stub.TbMethods[i].MethodID), 16),
			"prototype", stub.TbMethods[i].Prototype)
	}

	stubapi.SetLogger(logger)

	return &stub
}

func (tbs *TokenBasicStub) Methods(addr smc.Address) []Method {
	return tbs.TbMethods
}

func (tbs *TokenBasicStub) Name(addr smc.Address) string {
	return prototype.TokenBasic
}

// Dispatcher decodes tx data that was sent by caller, and dispatch it to smart contract to execute.
// The response would be empty if there is error happens (err != nil)
func (tbs *TokenBasicStub) Dispatcher(items *stubapi.InvokeParams, transID int64) (response stubapi.Response, bcerr bcerrors.BCError) {
	// Decode parameter with RLP API to get MethodInfo
	var methodInfo MethodInfo
	err := rlp.DecodeBytes(items.Params, &methodInfo)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	gas, err := items.Ctx.TxState.GetGas(items.Ctx.TxState.ContractAddress, methodInfo.MethodID)
	if err != nil {
		tbs.logger.Error("Dispatcher()， GetGas failed",
			"MethodID", strconv.FormatUint(uint64(methodInfo.MethodID), 16),
			"error", err)
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}

	tbs.logger.Debug("Dispatcher()", "MethodID", strconv.FormatUint(uint64(methodInfo.MethodID), 16), "Gas", gas)
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
	case tbs.TbMethods[TB_METHODID_TRANSFER].MethodID:

		tbs.logger.Debug("Dispatcher(), Calling Transfer() function")
		response.RequestMethod = tbs.TbMethods[TB_METHODID_TRANSFER].Prototype

		var itemsBytes = make([]([]byte), 0)
		if err = rlp.DecodeBytes(methodInfo.ParamData, &itemsBytes); err != nil {
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		} else if len(itemsBytes) < 2 {
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
			tbs.logger.Error("Dispatcher(), invalid address", "to", to, "error", err)
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		}

		baseContract := base.TokenBasic{NewTokenBasic(items.Ctx)}
		return response, baseContract.Transfer(to, *new(big.Int).SetBytes(itemsBytes[1][:]))

	case tbs.TbMethods[TB_METHODID_SETGASPRICE].MethodID:
		//Set GasPrice
		tbs.logger.Debug("Dispatcher(), Calling SetGasPrice() function")
		response.RequestMethod = tbs.TbMethods[TB_METHODID_SETGASPRICE].Prototype

		var itemsBytes = make([]([]byte), 0)
		if err = rlp.DecodeBytes(methodInfo.ParamData, &itemsBytes); err != nil {
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		}
		if len(itemsBytes[0]) > int(unsafe.Sizeof(uint64(0))) { //gasprice is a parameter with uint64 type
			tbs.logger.Error("Dispatcher(), invalid parameter",
				"gasprice", itemsBytes[0])
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidGasPrice
			return
		}
		baseContract := base.TokenBasic{NewTokenBasic(items.Ctx)}
		return response, baseContract.SetGasPrice(decode2Uint64(itemsBytes[0]))

	case tbs.TbMethods[TB_METHODID_SETGASBASEPRICE].MethodID:
		//Set GasBasePrice
		tbs.logger.Debug("Dispatcher(), Calling SetGasBasePrice() function")
		response.RequestMethod = tbs.TbMethods[TB_METHODID_SETGASBASEPRICE].Prototype

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
		baseContract := base.TokenBasic{NewTokenBasic(items.Ctx)}
		return response, baseContract.SetGasBasePrice(decode2Uint64(itemsBytes[0]))

	default:
		tbs.logger.Error("Dispatcher(), Invalid MethodID", "MethodID", methodInfo.MethodID)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidMethod
		return
	}
}

//CodeHash gets smart contract code hash
func (tbs *TokenBasicStub) CodeHash() []byte {
	//TBD
	return nil
}
