//validator manager stub

package stubs

import (
	"encoding/binary"
	"strconv"
	"strings"
	"unsafe"

	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/contract"
	"blockchain/abciapp_v1.0/contract/stubapi"
	"blockchain/abciapp_v1.0/contract/system"
	"blockchain/abciapp_v1.0/prototype"
	"blockchain/abciapp_v1.0/smc"
	"blockchain/algorithm"
	"blockchain/smcsdk/sdk/rlp"
	"github.com/tendermint/tmlibs/log"
)

const (
	SMC_METHODID_NEWVALIDATOR = iota
	SMC_METHODID_SETPOWER
	SMC_METHODID_SETREWARDADDR
	SMC_METHODID_FORBIDINTNCONTRACT
	SMC_METHODID_DEPLOYINTNCONTRACT
	SMC_METHODID_SETREWARDSTRATEGY
	SMC_METHODID_TOTAL_COUNT
)

var _ ContractStub = (*SystemStub)(nil)

type SystemStub struct {
	//contractAddr smc.Address
	logger     log.Logger
	SmcMethods []Method
}

func NewSystem(ctx *stubapi.InvokeContext) *contract.Contract {
	return &contract.Contract{Ctx: ctx}
}

//TokenBasicStub creates TokenBasic stub and initialize it with Methods
func NewSystemStub(logger log.Logger) *SystemStub {

	stubapi.SetLogger(logger)

	var stub SystemStub
	stub.logger = logger
	stub.SmcMethods = make([]Method, SMC_METHODID_TOTAL_COUNT)
	stub.SmcMethods[SMC_METHODID_NEWVALIDATOR].Prototype = prototype.SysNewValidator
	stub.SmcMethods[SMC_METHODID_SETPOWER].Prototype = prototype.SysSetPower
	stub.SmcMethods[SMC_METHODID_SETREWARDADDR].Prototype = prototype.SysSetRewardAddr
	stub.SmcMethods[SMC_METHODID_FORBIDINTNCONTRACT].Prototype = prototype.SysForbidInternalContract
	stub.SmcMethods[SMC_METHODID_DEPLOYINTNCONTRACT].Prototype = prototype.SysDeployInternalContract
	stub.SmcMethods[SMC_METHODID_SETREWARDSTRATEGY].Prototype = prototype.SysSetRewardStrategy

	for i, method := range stub.SmcMethods {
		stub.SmcMethods[i].MethodID = stubapi.ConvertPrototype2ID(method.Prototype)
		logger.Info("  method",
			"id", strconv.FormatUint(uint64(stub.SmcMethods[i].MethodID), 16),
			"prototype", stub.SmcMethods[i].Prototype)
	}

	return &stub
}

func (smcs *SystemStub) Methods(addr smc.Address) []Method {
	return smcs.SmcMethods
}

func (smcs *SystemStub) Name(addr smc.Address) string {
	return prototype.System
}

// Dispatcher decodes tx data that was sent by caller, and dispatch it to smart contract to execute.
// The response would be empty if there is error happens (err != nil)
func (smcs *SystemStub) Dispatcher(items *stubapi.InvokeParams, transID int64) (response stubapi.Response, bcerr bcerrors.BCError) {
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
		smcs.logger.Error("SystemStub, get Gas failed",
			"contract address", items.Ctx.TxState.ContractAddress,
			"MethodID", strconv.FormatUint(uint64(methodInfo.MethodID), 16),
			"error", err)

		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()

		return
	}
	smcs.logger.Debug("SystemStub", "MethodID", strconv.FormatUint(uint64(methodInfo.MethodID), 16), "Gas", gas)
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
		smcs.logger.Error("CheckAndPayForGas() failed", "error", err)
		return
	}
	var itemsBytes = make([]([]byte), 0)
	if err = rlp.DecodeBytes(methodInfo.ParamData, &itemsBytes); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}

	// To decode method parameter with RLP API and call specified Method of smart contract depends on MethodID
	switch methodInfo.MethodID {
	case smcs.SmcMethods[SMC_METHODID_NEWVALIDATOR].MethodID:

		smcs.logger.Debug("Dispatcher(), Calling SetValidator() function")
		response.RequestMethod = smcs.SmcMethods[SMC_METHODID_NEWVALIDATOR].Prototype

		if len(itemsBytes) != 4 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}

		if len(itemsBytes[0]) == 0 || //name
			len(itemsBytes[1]) != smc.PUBKEY_LEN || //pubKey
			len(itemsBytes[3]) > int(unsafe.Sizeof(uint64(0))) { //power uint64
			smcs.logger.Error("Dispatcher(), invalid parameter",
				"name", itemsBytes[0],
				"pubkey", itemsBytes[1],
				"rewardaddr", itemsBytes[2],
				"power", itemsBytes[3])
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		smcContract := system.System{NewSystem(items.Ctx)}
		power := decode2Uint64(itemsBytes[3])
		if bcerr = smcContract.NewValidator(string(itemsBytes[0][:]),
			itemsBytes[1][:],
			smc.Address(itemsBytes[2][:]),
			power); bcerr.ErrorCode != bcerrors.ErrCodeOK {

			return

		} else {
			//update validator to bcchain/tmcore
			response.Data = items.Ctx.GetValidatorUpdate(itemsBytes[1][:])
			smcs.logger.Debug("Dispatcher(), NewValidator()", "response validateupdate", response.Data)
		}
		response.Code = stubapi.RESPONSE_CODE_UPDATE_VALIDATORS

		return

	case smcs.SmcMethods[SMC_METHODID_SETPOWER].MethodID:

		smcs.logger.Debug("Dispatcher(), Calling SetPower() function")
		response.RequestMethod = smcs.SmcMethods[SMC_METHODID_SETPOWER].Prototype

		if len(itemsBytes[0]) != smc.PUBKEY_LEN || //pubKey
			len(itemsBytes[1]) > int(unsafe.Sizeof(uint64(0))) { //power uint64
			smcs.logger.Error("Dispatcher(), invalid parameter",
				"pubkey", itemsBytes[0],
				"power", itemsBytes[1])
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		smcContract := system.System{NewSystem(items.Ctx)}
		power := decode2Uint64(itemsBytes[1])
		if bcerr = smcContract.SetPower(itemsBytes[0][:], power); bcerr.ErrorCode != bcerrors.ErrCodeOK {
			return
		} else {
			//update validator to bcchain/tmcore
			response.Data = items.Ctx.GetValidatorUpdate(itemsBytes[0][:])

			smcs.logger.Debug("Dispatcher(), SetPower()", "response validateupdate", response.Data)
		}
		response.Code = stubapi.RESPONSE_CODE_UPDATE_VALIDATORS

		return

	case smcs.SmcMethods[SMC_METHODID_SETREWARDADDR].MethodID:

		smcs.logger.Debug("Dispatcher(), Calling SetRewardAddr() function")
		response.RequestMethod = smcs.SmcMethods[SMC_METHODID_SETREWARDADDR].Prototype

		if len(itemsBytes[0]) != smc.PUBKEY_LEN { //rewardAddr
			smcs.logger.Error("Dispatcher(), invalid parameter",
				"pubkey", itemsBytes[0],
				"rewardAddr", itemsBytes[1])
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		smcContract := system.System{NewSystem(items.Ctx)}
		if bcerr = smcContract.SetRewardAddr(itemsBytes[0][:], smc.Address(itemsBytes[1][:])); bcerr.ErrorCode != bcerrors.ErrCodeOK {

			return

		} else {
			//update validator to bcchain/tmcore
			response.Data = items.Ctx.GetValidatorUpdate(itemsBytes[0][:])

			smcs.logger.Debug("Dispatcher(), SetValidatorReward()", "response validateupdate", response.Data)
		}
		response.Code = stubapi.RESPONSE_CODE_UPDATE_VALIDATORS
		return
	case smcs.SmcMethods[SMC_METHODID_SETREWARDSTRATEGY].MethodID:

		smcs.logger.Debug("Dispatcher(), Calling SetRewardStrategy() function")
		response.RequestMethod = smcs.SmcMethods[SMC_METHODID_SETREWARDSTRATEGY].Prototype

		smcContract := system.System{NewSystem(items.Ctx)}
		if bcerr = smcContract.SetRewardStrategy(string(itemsBytes[0][:]), decode2Uint64(itemsBytes[1])); bcerr.ErrorCode != bcerrors.ErrCodeOK {
			return
		}

		return
	case smcs.SmcMethods[SMC_METHODID_DEPLOYINTNCONTRACT].MethodID:
		smcs.logger.Debug("Dispatcher(), Calling DeployInternalContract() function")
		response.RequestMethod = smcs.SmcMethods[SMC_METHODID_DEPLOYINTNCONTRACT].Prototype
		if len(itemsBytes) != 6 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}

		if len(itemsBytes[0]) == 0 || //name
			len(itemsBytes[2]) == 0 || //prototype
			len(itemsBytes[3]) == 0 ||
			len(itemsBytes[5]) > int(unsafe.Sizeof(uint64(0))) { // effectheight uint64
			smcs.logger.Error("Dispatcher(), invalid parameter",
				"name", itemsBytes[0],
				"prototype", itemsBytes[2],
				"gas", itemsBytes[3],
				"effectHeight", itemsBytes[5])
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		name := string(itemsBytes[0][:])
		version := string(itemsBytes[1][:])
		// Notes: prototype is split by ";" because "," is already using within prototype
		prototypes := strings.Split(string(itemsBytes[2][:]), ";")
		var gasList = make([]uint64, 0)
		for i := 0; i < len(itemsBytes[3]); i = i + 8 {
			gasList = append(gasList, binary.BigEndian.Uint64(itemsBytes[3][i:i+8]))
		}
		codeHash := smc.Hash(itemsBytes[4][:])
		effectHeight := decode2Uint64(itemsBytes[5])
		smcs.logger.Info("New contract",
			"name", name, "version", version,
			"prototype", prototypes,
			"gas", gasList,
			"effectHeight", effectHeight)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcContract := system.System{NewSystem(items.Ctx)}
		var newcontract smc.Address
		newcontract, bcerr = smcContract.DeployInternalContract(name,
			version,
			prototypes,
			gasList,
			codeHash,
			effectHeight)
		if bcerr.ErrorCode != bcerrors.ErrCodeOK {
			return
		}

		switch name {
		case prototype.TokenBYB:
			response.Code = stubapi.RESPONSE_CODE_NEWBYBCONTRACT
		case prototype.TAC:
			response.Code = stubapi.RESPONSE_CODE_NEWTRANSFERAGENCY
		case prototype.TB_Cancellation:
			response.Code = stubapi.RESPONSE_CODE_NEWTBCANCELLATIONCONTRACT
		case prototype.UPGRADE1TO2:
			response.Code = stubapi.RESPONSE_CODE_UPGRADE1TO2
		case prototype.TB_Team:
			response.Code = stubapi.RESPONSE_CODE_NEWTOKENBASICTEAM
		case prototype.TB_Foundation:
			response.Code = stubapi.RESPONSE_CODE_NEWTOKENBASICFOUNDATION
		}

		response.Log = name + "," + version
		response.Data = newcontract
		return
	case smcs.SmcMethods[SMC_METHODID_FORBIDINTNCONTRACT].MethodID:
		smcs.logger.Error("Dispatcher(), Unsupported Method", "MethodID", methodInfo.MethodID)
		response.RequestMethod = smcs.SmcMethods[SMC_METHODID_FORBIDINTNCONTRACT].Prototype
		if len(itemsBytes) != 2 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		if len(itemsBytes[0]) == 0 || //address
			len(itemsBytes[1]) > int(unsafe.Sizeof(uint64(0))) { // lose height
			smcs.logger.Error("Dispatcher(), invalid parameter",
				"address", itemsBytes[0],
				"effectHeight", itemsBytes[1])
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		address := string(itemsBytes[0])
		chainID := items.Ctx.TxState.StateDB.GetChainID()
		if err = algorithm.CheckAddress(chainID, address); err != nil {
			smcs.logger.Error("Checking affiliate address", "address", address, "error", err)
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		}

		smcContract := system.System{NewSystem(items.Ctx)}

		return response, smcContract.ForbidInternalContract(address, decode2Uint64(itemsBytes[1]))

	default:
		smcs.logger.Error("Dispatcher(), Invalid MethodID", "MethodID", methodInfo.MethodID)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidMethod
		return
	}
	return
}

//CodeHash gets smart contract code hash
func (tbs *SystemStub) CodeHash() []byte {
	//TBD
	return nil
}
