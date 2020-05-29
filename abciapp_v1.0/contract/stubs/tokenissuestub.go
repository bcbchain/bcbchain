//tokenbasicstub

package stubs

import (
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	cntr "github.com/bcbchain/bcbchain/abciapp_v1.0/contract/tokenissue"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/prototype"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	"github.com/bcbchain/sdk/sdk/rlp"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	"math/big"
	"strconv"
	"unsafe"
)

var _ ContractStub = (*TokenIssueStub)(nil)

// The minimum size of name and symbol for common users tokens
// Only super manager can issue token with 1 or 2 characters name or symbol
const MIN_SIZE = 3

type TokenIssueStub struct {
	logger    log.Logger
	TiMethods []Method
}

type NewTokenParam struct {
	Name             string  //Token Name
	Symbol           string  //Token Symbol
	TotalSupply      big.Int //Token Supply（units in Cong）
	AddSupplyEnabled bool    //Supports adding supply or not, true: yes, false: no
	BurnEnabled      bool    //supports burning supply or not, true: yes, false: no
	GasPrice         int64   //gas price of transaction(units in Cong), and using gic.
}

func NewTokenIssue(ctx *stubapi.InvokeContext) *contract.Contract {
	return &contract.Contract{Ctx: ctx}
}

//TokenBasicStub  creates TokenIssue stub and initialize it with Methods
func NewTokenIssueStub(logger log.Logger) *TokenIssueStub {
	//生成MethodID
	var stub TokenIssueStub
	stub.logger = logger
	stub.TiMethods = make([]Method, 1)
	stub.TiMethods[0].Prototype = prototype.TiNewToken
	stub.TiMethods[0].MethodID = stubapi.ConvertPrototype2ID(stub.TiMethods[0].Prototype)
	logger.Info("  method",
		"id", strconv.FormatUint(uint64(stub.TiMethods[0].MethodID), 16),
		"prototype", stub.TiMethods[0].Prototype)
	stubapi.SetLogger(logger)

	return &stub
}

func (tbs *TokenIssueStub) Methods(addr smc.Address) []Method {
	return tbs.TiMethods
}

func (tbs *TokenIssueStub) Name(addr smc.Address) string {
	return prototype.TokenIssue
}

// Dispatcher decodes tx data that was sent by caller, and dispatch it to smart contract to execute.
// The response would be empty if there is error happens (err != nil)
func (tbs *TokenIssueStub) Dispatcher(items *stubapi.InvokeParams) (response stubapi.Response, bcerr bcerrors.BCError) {
	// Decode parameter with RLP API to get MethodInfo
	var methodInfo MethodInfo
	err := rlp.DecodeBytes(items.Params, &methodInfo)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	tbs.logger.Debug("Input Parameter", "MethodID", methodInfo.MethodID)
	// Check and pay for Gas
	gas, err := items.Ctx.TxState.GetGas(items.Ctx.TxState.ContractAddress, methodInfo.MethodID)
	tbs.logger.Debug("Dispatcher()",
		"MethodID", strconv.FormatUint(uint64(methodInfo.MethodID), 16),
		"Gas", gas)

	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}

	if response.GasUsed, response.GasPrice, response.RewardValues, bcerr = items.Ctx.CheckAndPayForGas(
		items.Ctx.Sender,
		items.Ctx.Proposer,
		items.Ctx.Rewarder,
		gas,
		items.Ctx.GasLimit); bcerr.ErrorCode != bcerrors.ErrCodeOK {
		tbs.logger.Error("Dispatcher(), CheckAndPayForGas() failed", "error", err)
		return
	}

	response.Tags = nil
	// To decode method parameter with RLP API and call specified Method of smart contract depends on MethodID
	switch methodInfo.MethodID {

	case tbs.TiMethods[0].MethodID:

		tbs.logger.Debug("Dispatcher(), Calling NewToken() function")
		response.Data = "" // Contract address
		response.RequestMethod = tbs.TiMethods[0].Prototype

		var itemsBytes = make([]([]byte), 0)
		if err = rlp.DecodeBytes(methodInfo.ParamData, &itemsBytes); err != nil {
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		} else if len(itemsBytes) < 6 { //number of parameter of NewToken()
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		tbs.logger.Debug("Input Parameter", "itemsBytes", itemsBytes)

		// Creates TokenIssue struct
		tokenIssueContract := cntr.TokenIssue{NewTokenIssue(items.Ctx)}
		for i, item := range itemsBytes {
			if len(item) == 0 && i != 2 { //For totalSupply, it could be 0
				tbs.logger.Error("Dispatcher(), parameter is nil", "index", i)
				bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
				return
			}
			if i == 5 && len(item) > int(unsafe.Sizeof(uint64(0))) { //gasprice is a parameter with uint64 type
				tbs.logger.Error("Dispatcher(), gaslimit is incorrect", "gaslimit", item)
				bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidGasLimit
				return
			}
		}

		name := string(itemsBytes[0][:])
		symbol := string(itemsBytes[1][:])
		totalSupply := new(big.Int).SetBytes(itemsBytes[2][:])
		bAddSupply, _ := strconv.ParseBool(string(itemsBytes[3][:]))
		bBurn, _ := strconv.ParseBool(string(itemsBytes[4][:]))
		gasprice := decode2Uint64(itemsBytes[5])
		tbs.logger.Debug("NewToken Parameters", "Token Name", name,
			"Symbol", symbol,
			"TotalSupply", *totalSupply,
			"AddSupplyEnabled", bAddSupply,
			"BurnEnabled", bBurn,
			"GasePrice", gasprice)
		if len(name) < MIN_SIZE || len(symbol) < MIN_SIZE {
			if items.Ctx.Sender.Addr != items.Ctx.Owner.Addr {
				tbs.logger.Error("Common user do not has permission to issue this kind of tokens")
				bcerr.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
				return
			}
		}
		var addr smc.Address
		if addr, bcerr = tokenIssueContract.NewToken(
			name,
			symbol,
			*totalSupply,
			bAddSupply,
			bBurn,
			gasprice); addr == "" || bcerr.ErrorCode != bcerrors.ErrCodeOK {
			return
		} else {
			response.Data = addr                           // Contract address
			response.Code = stubapi.RESPONSE_CODE_NEWTOKEN //
			tbs.logger.Debug("New Token", "Response", response)
			return
		}

	default:
		tbs.logger.Error("Dispatcher(), Invalid MethodID", "MethodID", methodInfo.MethodID)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidMethod
		return
	}
}

//CodeHash gets smart contract code hash
func (tbs *TokenIssueStub) CodeHash() []byte {
	//TBD
	return nil
}
