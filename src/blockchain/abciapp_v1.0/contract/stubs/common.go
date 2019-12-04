package stubs

import (
	"blockchain/abciapp_v1.0/types"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/contract/smcapi"
	"blockchain/abciapp_v1.0/contract/stubapi"
	"blockchain/abciapp_v1.0/smc"
	"blockchain/abciapp_v1.0/statedb"
	"blockchain/algorithm"
	"blockchain/smcsdk/sdk/rlp"
	"common/bignumber_v1.0"
	"github.com/pkg/errors"
	"github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"
)

func IsRightHeight(items *stubapi.InvokeParams, logger log.Logger) error {
	contract, err := items.Ctx.TxState.StateDB.GetContract(items.Ctx.TxState.ContractAddress)
	if err != nil {
		return err
	}

	if contract.EffectHeight <= uint64(items.Ctx.BlockHeader.Height) {
		if contract.LoseHeight == 0 || contract.LoseHeight > uint64(items.Ctx.BlockHeader.Height) {
			return nil
		} else {
			if logger != nil {
				logger.Error("The specified contract has expired",
					"EffectHeight", contract.EffectHeight,
					"LoseHeight", contract.LoseHeight,
					"currentHeight", items.Ctx.BlockHeader.Height)
			}
			return errors.New("The specified contract has expired")
		}
	}
	if logger != nil {
		logger.Error("The specified contract is not yet in effect",
			"EffectHeight", contract.EffectHeight,
			"LoseHeight", contract.LoseHeight,
			"currentHeight", items.Ctx.BlockHeader.Height)
	}
	return errors.New("The specified contract is not yet in effect")
}

// Checking if contract is effecting or not.
// Use this func to replace above one
// TODO: for a token, need to add function to get its latest contract if its contract was upgraded
func CheckConctractStatus(items *stubapi.InvokeParams, logger log.Logger) error {
	contract, err := items.Ctx.TxState.StateDB.GetContract(items.Ctx.TxState.ContractAddress)
	if err != nil {
		return err
	}

	if contract.EffectHeight <= uint64(items.Ctx.BlockHeader.Height) {
		if contract.LoseHeight == 0 || contract.LoseHeight > uint64(items.Ctx.BlockHeader.Height) {
			return nil
		} else {
			if logger != nil {
				logger.Error("The specified contract has expired",
					"EffectHeight", contract.EffectHeight,
					"LoseHeight", contract.LoseHeight,
					"currentHeight", items.Ctx.BlockHeader.Height)
			}
			return errors.New("The specified contract has expired")
		}
	}
	if logger != nil {
		logger.Error("The specified contract is not yet in effect",
			"EffectHeight", contract.EffectHeight,
			"LoseHeight", contract.LoseHeight,
			"currentHeight", items.Ctx.BlockHeader.Height)
	}
	return errors.New("The specified contract is not yet in effect")
}

func TransactionFee(items *stubapi.InvokeParams, transID int64) (gasUsed, gasPrice uint64, rewards map[smc.Address]uint64, bcerr smc.Error) {
	// Decode parameter with RLP API to get MethodInfo
	var methodInfo MethodInfo
	if err := rlp.DecodeBytes(items.Params, &methodInfo); err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}

	gas, err := items.Ctx.TxState.GetGas(items.Ctx.TxState.ContractAddress, methodInfo.MethodID)
	if err != nil {
		bcerr.ErrorCode = bcerrors.ErrCodeLowLevelError
		bcerr.ErrorDesc = err.Error()
		return
	}
	// Check and pay for Gas
	gasUsed, gasPrice, rewards, bcerr = items.Ctx.CheckAndPayForGas(
		items.Ctx.Sender,
		items.Ctx.Proposer,
		items.Ctx.Rewarder,
		gas,
		items.Ctx.GasLimit,
		transID)

	return
}

func receiptsOfTransactionFee(evtHandler *smcapi.EventHandler, tokenAddress, sender smc.Address, fee uint64, rewardValues map[smc.Address]uint64) {

	//Receipts of Fee
	evtHandler.PackReceiptOfFee(tokenAddress, sender, fee)
	// Sort rewards FEE by address
	strKey := make([]smc.Address, 0)
	for k, _ := range rewardValues {
		strKey = append(strKey, k)
	}
	sort.Strings(strKey)
	for _, k := range strKey {
		bigV := bignumber.UintToBigInt(rewardValues[k])
		evtHandler.PackReceiptOfTransfer(tokenAddress, sender, k, bignumber.NB(&bigV))
	}
}

func addReceiptsToResponse(eventHandler *smcapi.EventHandler, response *stubapi.Response) {
	for index, receipt := range eventHandler.GetReceipts() {
		recByte, _ := json.Marshal(receipt)
		kvPair := common.KVPair{Key: []byte(fmt.Sprintf("%d", index)), Value: recByte}
		response.Tags = append(response.Tags, kvPair)
	}
}

func CreateAccount(ctx *stubapi.InvokeContext, address smc.Address, tokenName string) *stubapi.Account {
	if len(tokenName) == 0 {
		genToken, _ := ctx.TxState.GetGenesisToken()
		tokenName = genToken.Name
	}
	tokenAddr, _ := ctx.TxState.GetTokenAddrByName(tokenName)

	txState := statedb.TxState{
		ctx.TxState.StateDB,
		tokenAddr,
		ctx.TxState.SenderAddress,
		ctx.TxState.TxBuffer,
	}

	return &stubapi.Account{address, &txState}
}

func CreateContractAcct(ctx *stubapi.InvokeContext, contractName, tokenName string) *stubapi.Account {
	addr := algorithm.CalcContractAddress(
		ctx.TxState.GetChainID(),
		"",
		contractName,
		"")

	if len(tokenName) == 0 {
		genToken, _ := ctx.TxState.GetGenesisToken()
		tokenName = genToken.Name
	}
	tokenAddr, _ := ctx.TxState.GetTokenAddrByName(tokenName)
	txState := statedb.TxState{
		ctx.TxState.StateDB,
		tokenAddr,
		ctx.TxState.SenderAddress,
		ctx.TxState.TxBuffer,
	}

	return &stubapi.Account{addr, &txState}
}

func decode2Uint64(b []byte) uint64 {

	tx8 := make([]byte, 8)
	copy(tx8[len(tx8)-len(b):], b)

	return binary.BigEndian.Uint64(tx8[:])
}

func decode2Int64(b []byte) int64 {

	return int64(decode2Uint64(b))
}

func callFunc(methodMap map[uint32]interface{}, id uint32, params ...interface{}) (result []reflect.Value, err error) {

	v, ok := methodMap[id]
	if !ok {
		err = errors.New("The specified methodid is unsupported")
		return
	}
	f := reflect.ValueOf(v)
	if f.IsNil() {
		err = errors.New("The specified method is unsupported")
		return
	}
	if len(params) != f.Type().NumIn() {
		err = errors.New("Invalid number of parameters passed.")
		return
	}

	in := make([]reflect.Value, len(params))
	for k, param := range params {
		in[k] = reflect.ValueOf(param)
	}

	result = f.Call(in)
	return
}

func checkTokenStatus(ic *stubapi.InvokeContext) (bcerr bcerrors.BCError) {
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
	if bignumber.Compare(token.TotalSupply, bignumber.Zero()) == 0 {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsTokenNotInit
		return
	}

	bcerr.ErrorCode = bcerrors.ErrCodeOK
	return
}

func IsMiningRightHeight(items *stubapi.InvokeParams, contract *types.Contract, logger log.Logger) error {
	if contract.EffectHeight <= uint64(items.Ctx.BlockHeader.Height) {
		if contract.LoseHeight == 0 || contract.LoseHeight > uint64(items.Ctx.BlockHeader.Height) {
			return nil
		} else {
			if logger != nil {
				logger.Error("The specified contract has expired",
					"EffectHeight", contract.EffectHeight,
					"LoseHeight", contract.LoseHeight,
					"currentHeight", items.Ctx.BlockHeader.Height)
			}
			return errors.New("The specified contract has expired")
		}
	}
	if logger != nil {
		logger.Error("The specified contract is not yet in effect",
			"EffectHeight", contract.EffectHeight,
			"LoseHeight", contract.LoseHeight,
			"currentHeight", items.Ctx.BlockHeader.Height)
	}
	return errors.New("The specified contract is not yet in effect")
}

func CalcContractAcct(ctx *stubapi.InvokeContext, contractName string) *stubapi.Account {

	addr := algorithm.CalcContractAddress(
		ctx.TxState.GetChainID(),
		"",
		contractName,
		"")
	//TODO: using genesis token for now
	genToken, _ := ctx.TxState.GetGenesisToken()
	if genToken == nil {
		return nil
	}

	return &stubapi.Account{addr,
		&statedb.TxState{ctx.TxState.StateDB,
			genToken.Address,
			ctx.TxState.SenderAddress,
			ctx.TxState.TxBuffer}}
}
