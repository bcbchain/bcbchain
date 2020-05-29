// TokenBYBStub

package stubs

import (
	"strconv"
	"unsafe"

	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/smcapi"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/tokenbyb"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/prototype"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	"github.com/bcbchain/bclib/algorithm"
	"github.com/bcbchain/sdk/sdk/rlp"
	"github.com/bcbchain/bclib/bignumber_v1.0"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
)

const (
	BYB_METHODID_INIT = iota
	BYB_METHODID_SETOWNER
	BYB_METHODID_SETGASPRICE
	BYB_METHODID_ADDSUPPLY
	BYB_METHODID_BURN
	BYB_METHODID_NEWBLACKHOLE
	BYB_METHODID_NEWSTOCKHOLDER
	BYB_METHODID_DELSTOCKHOLDER
	BYB_METHODID_CHANGECHROMOOWNERSHIP
	BYB_METHODID_TRANSFER
	BYB_METHODID_TRANSFERBYCHROMO

	// number
	BYB_METHODID_TOTAL_COUNT
)

type TokenBYBStub struct {
	logger     log.Logger
	BYBMethods []Method
}

var _ ContractStub = (*TokenBYBStub)(nil)

func NewTokenBYB(ctx *stubapi.InvokeContext) *smcapi.SmcApi {

	newsmcapi := smcapi.SmcApi{
		Sender:       ctx.Sender,
		Owner:        ctx.Owner,
		ContractAcct: CreateContractAcct(ctx, prototype.TokenBYB, tokenbyb.BybName),
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

// NewTokenBYBStub creates TokenBYB stub and initialize it with Methods
func NewTokenBYBStub(logger log.Logger) *TokenBYBStub {
	// create methodID
	var stub TokenBYBStub
	stub.logger = logger
	stub.BYBMethods = make([]Method, BYB_METHODID_TOTAL_COUNT)
	stub.BYBMethods[BYB_METHODID_INIT].Prototype = prototype.BYBInit
	stub.BYBMethods[BYB_METHODID_SETOWNER].Prototype = prototype.BYBSetOwner
	stub.BYBMethods[BYB_METHODID_SETGASPRICE].Prototype = prototype.BYBSetGasPrice
	stub.BYBMethods[BYB_METHODID_ADDSUPPLY].Prototype = prototype.BYBAddSupply
	stub.BYBMethods[BYB_METHODID_BURN].Prototype = prototype.BYBBurn
	stub.BYBMethods[BYB_METHODID_NEWBLACKHOLE].Prototype = prototype.BYBNewBlackHole
	stub.BYBMethods[BYB_METHODID_NEWSTOCKHOLDER].Prototype = prototype.BYBNewStockHolder
	stub.BYBMethods[BYB_METHODID_DELSTOCKHOLDER].Prototype = prototype.BYBDelStockHolder
	stub.BYBMethods[BYB_METHODID_CHANGECHROMOOWNERSHIP].Prototype = prototype.BYBChangeChromoOwnerShip
	stub.BYBMethods[BYB_METHODID_TRANSFER].Prototype = prototype.BYBTransfer
	stub.BYBMethods[BYB_METHODID_TRANSFERBYCHROMO].Prototype = prototype.BYBTransferByChromo

	for i, method := range stub.BYBMethods {
		stub.BYBMethods[i].MethodID = stubapi.ConvertPrototype2ID(method.Prototype)
		logger.Info("  method()",
			"id", strconv.FormatUint(uint64(stub.BYBMethods[i].MethodID), 16),
			"prototype", stub.BYBMethods[i].Prototype)
	}
	stubapi.SetLogger(logger)

	return &stub
}

func (bybs *TokenBYBStub) Methods(addr smc.Address) []Method {
	return bybs.BYBMethods
}

func (bybs *TokenBYBStub) Name(addr smc.Address) string {
	return prototype.TokenBYB
}

// Dispatcher decodes tx data that was sent by caller, and dispatch it to smart contract to execute.
// The response would be empty if there is error happens (err != nil)
func (bybs *TokenBYBStub) Dispatcher(items *stubapi.InvokeParams) (response stubapi.Response, bcerr bcerrors.BCError) {
	// Decode parameter with RLP API to get MethodInfo
	var methodInfo MethodInfo
	if err := rlp.DecodeBytes(items.Params, &methodInfo); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}

	gas, err := items.Ctx.TxState.GetGas(items.Ctx.TxState.ContractAddress, methodInfo.MethodID)
	bybs.logger.Debug("Dispatcher()",
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
		bybs.logger.Error("CheckAndPayForGas() failed", "error", err)
		return
	}

	// construct TokenByb object
	bybContract := tokenbyb.TokenByb{NewTokenBYB(items.Ctx)}

	tokenbasic, _ := bybContract.State.GetGenesisToken()
	receiptsOfTransactionFee(bybContract.EventHandler, tokenbasic.Address, bybContract.Sender.Addr, response.GasUsed*response.GasPrice, response.RewardValues)

	// Parse function paramter
	var itemsBytes = make([]([]byte), 0)
	if err = rlp.DecodeBytes(methodInfo.ParamData, &itemsBytes); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		return
	}

	// To decode method parameter with RLP API and call specified Method of smart contract depends on MethodID
	switch methodInfo.MethodID {
	case bybs.BYBMethods[BYB_METHODID_TRANSFER].MethodID: // Transfer
		bybs.logger.Debug("Dispatcher, Calling Transfer() Function")
		response.RequestMethod = bybs.BYBMethods[BYB_METHODID_TRANSFER].Prototype

		if len(itemsBytes) != 2 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		bybs.logger.Debug("Input Parameter", "itemsBytes", itemsBytes)

		to := string(itemsBytes[0][:])
		chainID := items.Ctx.TxState.StateDB.GetChainID()
		if err = algorithm.CheckAddress(chainID, to); err != nil {
			bybs.logger.Error("Dispatcher(), invalid address", "to", to, "error", err)
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		}

		bcerr = bybContract.Transfer(to, *bignumber.SetBytes(itemsBytes[1]))

	case bybs.BYBMethods[BYB_METHODID_TRANSFERBYCHROMO].MethodID: // TransferByChromo
		bybs.logger.Debug("Dispatcher, Calling TransferByChromo() Function")
		response.RequestMethod = bybs.BYBMethods[BYB_METHODID_TRANSFERBYCHROMO].Prototype

		if len(itemsBytes) != 3 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		chromo := string(itemsBytes[0][:])
		if len(chromo) == 0 {
			bybs.logger.Error("Dispatcher(), invalid chromo")
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			bcerr.ErrorDesc = "Chromo cannot be empty"
			return
		}
		to := string(itemsBytes[1][:])
		chainID := items.Ctx.TxState.StateDB.GetChainID()
		if err = algorithm.CheckAddress(chainID, to); err != nil {
			bybs.logger.Error("Dispatcher(), invalid address", "to", to, "error", err)
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		}

		bcerr = bybContract.TransferByChromo(chromo, to, bignumber.N(0).SetBytes(itemsBytes[2]))

	case bybs.BYBMethods[BYB_METHODID_INIT].MethodID: // Init
		bybs.logger.Debug("Dispatcher, Calling Init() Function")
		response.RequestMethod = bybs.BYBMethods[BYB_METHODID_INIT].Prototype

		if len(itemsBytes) != 3 { //number of parameter of Init()
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		bybs.logger.Debug("Input Parameter", "itemsBytes", itemsBytes)

		totalSupply := bignumber.N(0).SetBytes(itemsBytes[0])
		bAddSupply, _ := strconv.ParseBool(string(itemsBytes[1][:]))
		bBurn, _ := strconv.ParseBool(string(itemsBytes[2][:]))

		bcerr = bybContract.Init(totalSupply, bAddSupply, bBurn)

	case bybs.BYBMethods[BYB_METHODID_SETOWNER].MethodID: // SetOwner
		bybs.logger.Debug("Dispatcher, Calling SetOwner() Function")
		response.RequestMethod = bybs.BYBMethods[BYB_METHODID_SETOWNER].Prototype

		if len(itemsBytes) != 1 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}

		address := string(itemsBytes[0][:])
		chainID := items.Ctx.TxState.StateDB.GetChainID()
		if err = algorithm.CheckAddress(chainID, address); err != nil {
			bybs.logger.Error("Dispatcher(), invalid address", "address", address, "error", err)
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidAddr
			bcerr.ErrorDesc = err.Error()
			return
		}

		bcerr = bybContract.SetOwner(address)

	case bybs.BYBMethods[BYB_METHODID_SETGASPRICE].MethodID: // SetGasPrice
		bybs.logger.Debug("Dispatcher, Calling SetGasPrice() Function")
		response.RequestMethod = bybs.BYBMethods[BYB_METHODID_SETGASPRICE].Prototype

		if len(itemsBytes) != 1 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		if len(itemsBytes[0]) > int(unsafe.Sizeof(int64(0))) {
			bybs.logger.Error("Dispatcher(), invalid team",
				"gasprice", itemsBytes[1])
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidTeam
			return
		}

		bcerr = bybContract.SetGasPrice(decode2Uint64(itemsBytes[0]))

	case bybs.BYBMethods[BYB_METHODID_ADDSUPPLY].MethodID: // AddSupply
		bybs.logger.Debug("Dispatcher, Calling AddSupply() Function")
		response.RequestMethod = bybs.BYBMethods[BYB_METHODID_ADDSUPPLY].Prototype

		if len(itemsBytes) != 1 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		bcerr = bybContract.AddSupply(bignumber.N(0).SetBytes(itemsBytes[0]))

	case bybs.BYBMethods[BYB_METHODID_BURN].MethodID: // Burn
		bybs.logger.Debug("Dispatcher, Calling Burn() Function")
		response.RequestMethod = bybs.BYBMethods[BYB_METHODID_BURN].Prototype

		if len(itemsBytes) != 1 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		bcerr = bybContract.Burn(bignumber.N(0).SetBytes(itemsBytes[0]))

	case bybs.BYBMethods[BYB_METHODID_NEWBLACKHOLE].MethodID: // NewBlackHole
		bybs.logger.Debug("Dispatcher, Calling NewBlockHole() Function")
		response.RequestMethod = bybs.BYBMethods[BYB_METHODID_NEWBLACKHOLE].Prototype

		if len(itemsBytes) != 1 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}

		address := string(itemsBytes[0][:])
		chainID := items.Ctx.TxState.StateDB.GetChainID()
		if err = algorithm.CheckAddress(chainID, address); err != nil {
			bybs.logger.Error("Dispatcher(), invalid address", "address", address, "error", err)
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidAddr
			bcerr.ErrorDesc = err.Error()
			return
		}
		bcerr = bybContract.NewBlackHole(address)

	case bybs.BYBMethods[BYB_METHODID_NEWSTOCKHOLDER].MethodID: // NewStockHolder
		bybs.logger.Debug("Dispatcher, Calling NewstockHolder() Function")
		response.RequestMethod = bybs.BYBMethods[BYB_METHODID_NEWSTOCKHOLDER].Prototype

		if len(itemsBytes) != 2 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		bybs.logger.Debug("Input Parameter", "itemsBytes", itemsBytes)

		stockAddr := string(itemsBytes[0][:])
		chainID := items.Ctx.TxState.StateDB.GetChainID()
		if err = algorithm.CheckAddress(chainID, stockAddr); err != nil {
			bybs.logger.Error("Dispatcher(), invalid address", "stockAddr", stockAddr, "error", err)
			bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcerr.ErrorDesc = err.Error()
			return
		}

		value := bignumber.N(0).SetBytes(itemsBytes[1])
		response.Code = stubapi.RESPONSE_CODE_BYBCHROMO
		response.Data, bcerr = bybContract.NewStockHolder(stockAddr, value)

	case bybs.BYBMethods[BYB_METHODID_DELSTOCKHOLDER].MethodID: // DelStockHolder
		bybs.logger.Debug("Dispatcher, Calling DekStockHolder() Function")
		response.RequestMethod = bybs.BYBMethods[BYB_METHODID_DELSTOCKHOLDER].Prototype

		if len(itemsBytes) != 1 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}

		address := string(itemsBytes[0][:])
		chainID := items.Ctx.TxState.StateDB.GetChainID()
		if err = algorithm.CheckAddress(chainID, address); err != nil {
			bybs.logger.Error("Dispatcher(), invalid address", "address", address, "error", err)
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidAddr
			bcerr.ErrorDesc = err.Error()
			return
		}
		bcerr = bybContract.DelStockHolder(address)

	case bybs.BYBMethods[BYB_METHODID_CHANGECHROMOOWNERSHIP].MethodID: // ChangeChromoOwnerShip
		bybs.logger.Debug("Dispatcher, Calling ChangeChromoOwnerShip() Function")
		response.RequestMethod = bybs.BYBMethods[BYB_METHODID_CHANGECHROMOOWNERSHIP].Prototype

		if len(itemsBytes) != 2 {
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			return
		}
		chromo := string(itemsBytes[0][:])
		if len(chromo) == 0 {
			bybs.logger.Error("Dispatcher(), invalid chromo")
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			bcerr.ErrorDesc = "Chromo cannot be empty"
			return
		}

		address := string(itemsBytes[1][:])
		chainID := items.Ctx.TxState.StateDB.GetChainID()
		if err = algorithm.CheckAddress(chainID, address); err != nil {
			bybs.logger.Error("Dispatcher(), invalid address", "address", address, "error", err)
			bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidAddr
			bcerr.ErrorDesc = err.Error()
			return
		}

		bcerr = bybContract.ChangeChromoOwnership(chromo, address)

	default:
		bybs.logger.Error("Dispatcher(), Invalid MethodID", "MethodID", methodInfo.MethodID)
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidMethod
		return
	}

	if bcerr.ErrorCode == bcerrors.ErrCodeOK {
		addReceiptsToResponse(bybContract.EventHandler, &response)
	}
	return
}

// CodeHash gets smart contract code hash
func (bybs *TokenBYBStub) CodeHash() []byte {
	//TBD
	return nil
}
