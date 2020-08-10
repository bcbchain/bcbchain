package check

import (
	"encoding/binary"
	"errors"
	tx1 "github.com/bcbchain/bclib/tx/v1"
	"math/big"
	"strings"

	"github.com/bcbchain/bcbchain/abciapp/service/check"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/prototype"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	bctx "github.com/bcbchain/bcbchain/abciapp_v1.0/tx/tx"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
	tx2 "github.com/bcbchain/bclib/tx/v2"
	types2 "github.com/bcbchain/bclib/types"
	"github.com/bcbchain/sdk/sdk/bn"
	"github.com/bcbchain/sdk/sdk/rlp"
)

const (
	transferMethodID = 0x44d8ca60
)

func (conn *CheckConnection) CheckBCTx(tx []byte, connV2 *check.AppCheck) types.ResponseCheckTx {

	var transaction bctx.Transaction
	chainID := conn.stateDB.GetChainID()
	fromAddr, pubKey, err := transaction.TxParse(chainID, string(tx))
	if err != nil {
		conn.logger.Error("tx parse failed:", "error", err)
		bcError := bcerrors.BCError{
			ErrorCode: bcerrors.ErrCodeCheckTxTransData,
			ErrorDesc: "",
		}
		return types.ResponseCheckTx{
			Code: bcError.ErrorCode,
			Log:  bcError.Error(),
		}
	}

	if connV2 != nil {
		return conn.runCheckBCTxEx(tx, fromAddr, pubKey, transaction, connV2)
	}

	return conn.runCheckBCTx(fromAddr, transaction)
}

func (conn *CheckConnection) CheckBCTxV1Concurrency(tx []byte, connV2 *check.AppCheck) *types.Result {

	result := &types.Result{
		TxVersion: "tx1",
		Tx:        tx,
	}
	var transaction bctx.Transaction
	chainID := conn.stateDB.GetChainID()
	fromAddr, pubKey, err := transaction.TxParse(chainID, string(tx))
	if err != nil {
		result.Errorlog = err
		return result
	}
	if connV2 != nil {
		//return conn.runCheckBCTxEx(tx, fromAddr, pubKey, transaction, connV2)
		result.TxV1Result.FromAddr = fromAddr
		result.TxV1Result.Pubkey = pubKey
		result.TxV1Result.Transaction = tx1.Transaction(transaction)
		//connv2 在最开始调用处有指针存储
	}

	//return conn.runCheckBCTx(fromAddr, transaction)
	result.TxV1Result.FromAddr = fromAddr
	result.TxV1Result.Transaction = tx1.Transaction(transaction)

	return result
}

func (conn *CheckConnection) runCheckBCTx(fromAddr smc.Address, transaction bctx.Transaction) types.ResponseCheckTx {

	// Check note, it must stay within 40 characters limit
	if len(transaction.Note) > bctx.MAX_SIZE_NOTE {
		bcError := bcerrors.BCError{
			ErrorCode: bcerrors.ErrCodeCheckTxNoteExceedLimit,
			ErrorDesc: "",
		}
		return types.ResponseCheckTx{
			Code: bcError.ErrorCode,
			Log:  bcError.Error(),
		}
	}
	// Check Nonce
	err := conn.stateDB.CheckAccountNonce(smc.Address(fromAddr), transaction.Nonce)
	if err != nil {
		conn.logger.Error("check nonce error:", err)
		bcError := bcerrors.BCError{
			ErrorCode: bcerrors.ErrCodeCheckTxInvalidNonce,
			ErrorDesc: "",
		}
		return types.ResponseCheckTx{
			Code: bcError.ErrorCode,
			Log:  bcError.Error(),
		}
	}

	// Generate a fake BeginBlock as a workaround to
	// cache balance changes while checking sender's balance for Fee.
	// Rollback once when checkTx is done
	_, trans := statedbhelper.NewRollbackTransactionID()
	conn.stateDB.BeginBlock(trans)
	defer conn.stateDB.RollBlock()

	txState := conn.stateDB.NewTxState(smc.Address(transaction.To), smc.Address(fromAddr))

	//根据智能合约地址找到智能合约外部账户地址
	contract, err := conn.stateDB.GetContract(smc.Address(transaction.To))
	if err != nil {
		conn.logger.Error("failed json.Unmarshal(contractBytes,contract)", "error", err)

		bcError := bcerrors.BCError{
			ErrorCode: bcerrors.ErrCodeLowLevelError,
			ErrorDesc: err.Error(),
		}
		return types.ResponseCheckTx{
			Code: bcError.ErrorCode,
			Log:  bcError.Error(),
		}
	}
	if contract == nil {
		conn.logger.Error("can't find this contract from DB", "contract address", transaction.To)
		bcError := bcerrors.BCError{
			ErrorCode: bcerrors.ErrCodeCheckTxNoContract,
			ErrorDesc: "",
		}
		return types.ResponseCheckTx{
			Code: bcError.ErrorCode,
			Log:  bcError.Error(),
		}
	}

	sender := &stubapi.Account{
		smc.Address(fromAddr),
		txState,
	}

	owner := &stubapi.Account{
		contract.Owner,
		txState,
	}

	app, err := conn.stateDB.GetWorldAppState()
	app.BeginBlock.Header.Height++
	app.BeginBlock.Header.Time++
	invokeContext := &stubapi.InvokeContext{
		sender,
		owner,
		txState,
		nil,
		app.BeginBlock.Header,
		nil,
		nil,
		transaction.GasLimit,
		transaction.Note,
	}

	conn.logger.Debug("start invoke.....")

	item := &stubapi.InvokeParams{
		invokeContext,
		transaction.Data,
	}

	_, bcerr := conn.docker.Invoke(item, 0)
	if bcerr.ErrorCode != bcerrors.ErrCodeOK {
		conn.logger.Error("docker invoke failed to check TX",
			"error code", bcerr.ErrorCode,
			"error", bcerr.Error())

		return types.ResponseCheckTx{
			Code: bcerr.ErrorCode,
			Log:  bcerr.Error(),
		}
	}

	conn.logger.Debug("end invoke checkTx contract docker .....")

	return types.ResponseCheckTx{
		Code: bcerrors.ErrCodeOK,
		Log:  "Check tx succeed"}
}

func (conn *CheckConnection) RunCheckBCTxConcurrency(result types.Result) types.ResponseCheckTx {

	// Check note, it must stay within 40 characters limit
	if len(result.TxV1Result.Transaction.Note) > bctx.MAX_SIZE_NOTE {
		bcError := bcerrors.BCError{
			ErrorCode: bcerrors.ErrCodeCheckTxNoteExceedLimit,
			ErrorDesc: "",
		}
		return types.ResponseCheckTx{
			Code: bcError.ErrorCode,
			Log:  bcError.Error(),
		}
	}
	// Check Nonce
	err := conn.stateDB.CheckAccountNonce(smc.Address(result.TxV1Result.FromAddr), result.TxV1Result.Transaction.Nonce)
	if err != nil {
		conn.logger.Error("check nonce error:", err)
		bcError := bcerrors.BCError{
			ErrorCode: bcerrors.ErrCodeCheckTxInvalidNonce,
			ErrorDesc: "",
		}
		return types.ResponseCheckTx{
			Code: bcError.ErrorCode,
			Log:  bcError.Error(),
		}
	}

	// Generate a fake BeginBlock as a workaround to
	// cache balance changes while checking sender's balance for Fee.
	// Rollback once when checkTx is done
	_, trans := statedbhelper.NewRollbackTransactionID()
	conn.stateDB.BeginBlock(trans)
	defer conn.stateDB.RollBlock()

	txState := conn.stateDB.NewTxState(smc.Address(result.TxV1Result.Transaction.To), smc.Address(result.TxV1Result.FromAddr))

	//根据智能合约地址找到智能合约外部账户地址
	contract, err := conn.stateDB.GetContract(smc.Address(result.TxV1Result.Transaction.To))
	if err != nil {
		conn.logger.Error("failed json.Unmarshal(contractBytes,contract)", "error", err)

		bcError := bcerrors.BCError{
			ErrorCode: bcerrors.ErrCodeLowLevelError,
			ErrorDesc: err.Error(),
		}
		return types.ResponseCheckTx{
			Code: bcError.ErrorCode,
			Log:  bcError.Error(),
		}
	}
	if contract == nil {
		conn.logger.Error("can't find this contract from DB", "contract address", result.TxV1Result.Transaction.To)
		bcError := bcerrors.BCError{
			ErrorCode: bcerrors.ErrCodeCheckTxNoContract,
			ErrorDesc: "",
		}
		return types.ResponseCheckTx{
			Code: bcError.ErrorCode,
			Log:  bcError.Error(),
		}
	}

	sender := &stubapi.Account{
		smc.Address(result.TxV1Result.FromAddr),
		txState,
	}

	owner := &stubapi.Account{
		contract.Owner,
		txState,
	}

	app, err := conn.stateDB.GetWorldAppState()
	app.BeginBlock.Header.Height++
	app.BeginBlock.Header.Time++
	invokeContext := &stubapi.InvokeContext{
		sender,
		owner,
		txState,
		nil,
		app.BeginBlock.Header,
		nil,
		nil,
		result.TxV1Result.Transaction.GasLimit,
		result.TxV1Result.Transaction.Note,
	}

	conn.logger.Debug("start invoke.....")

	item := &stubapi.InvokeParams{
		invokeContext,
		result.TxV1Result.Transaction.Data,
	}

	_, bcerr := conn.docker.Invoke(item, 0)
	if bcerr.ErrorCode != bcerrors.ErrCodeOK {
		conn.logger.Error("docker invoke failed to check TX",
			"error code", bcerr.ErrorCode,
			"error", bcerr.Error())

		return types.ResponseCheckTx{
			Code: bcerr.ErrorCode,
			Log:  bcerr.Error(),
		}
	}

	conn.logger.Debug("end invoke checkTx contract docker .....")

	return types.ResponseCheckTx{
		Code: bcerrors.ErrCodeOK,
		Log:  "Check tx succeed"}
}

func (conn *CheckConnection) runCheckBCTxEx(tx []byte, fromAddr smc.Address, pubKey crypto.PubKeyEd25519, tx1 bctx.Transaction, connV2 *check.AppCheck) types.ResponseCheckTx {

	contract, err := conn.stateDB.GetContract(tx1.To)
	if err != nil {
		return types.ResponseCheckTx{
			Code: bcerrors.ErrCodeLowLevelError,
			Log:  err.Error(),
		}
	}

	if contract == nil {
		return types.ResponseCheckTx{
			Code: bcerrors.ErrCodeLowLevelError,
			Log:  "invalid smcAddress"}
	}

	// check chainVersion
	if contract.ChainVersion != 0 {
		return types.ResponseCheckTx{
			Code: bcerrors.ErrCodeLowLevelError,
			Log:  "v1 transaction cannot use other version contract",
		}
	}

	// check orgID and name
	if !(contract.OrgID == statedbhelper.GetGenesisOrgID(0, 0) &&
		contract.ChainVersion == 0 &&
		(contract.Name == "token-basic" || strings.HasPrefix(contract.Name, "token-templet-"))) {
		return conn.runCheckBCTx(fromAddr, tx1)
	}

	appState := statedbhelper.GetWorldAppState(0, 0)

	effectContract := statedbhelper.GetEffectContractByName(0, 0, appState.BeginBlock.Header.Height+1, contract.Name, contract.OrgID)
	if effectContract == nil {
		return types.ResponseCheckTx{
			Code: bcerrors.ErrCodeLowLevelError,
			Log:  "cannot get effectContract",
		}
	}
	msg := types2.Message{
		Contract: effectContract.Token,
	}
	msg.MethodID, msg.Items, err = conn.exMessageParams(tx1.Data)
	if err != nil {
		return conn.runCheckBCTx(fromAddr, tx1)
	}

	tx2 := types2.Transaction{
		Nonce:    tx1.Nonce,
		GasLimit: int64(tx1.GasLimit),
		Note:     tx1.Note,
		Messages: []types2.Message{msg},
	}

	return connV2.RunCheckTx(tx, tx2, pubKey)
}

func (conn *CheckConnection) RunCheckBCTxExConcurrency(result types.Result, connV2 *check.AppCheck) types.ResponseCheckTx {

	contract, err := conn.stateDB.GetContract(result.TxV1Result.Transaction.To)
	if err != nil {
		return types.ResponseCheckTx{
			Code: bcerrors.ErrCodeLowLevelError,
			Log:  err.Error(),
		}
	}

	if contract == nil {
		return types.ResponseCheckTx{
			Code: bcerrors.ErrCodeLowLevelError,
			Log:  "invalid smcAddress"}
	}

	// check chainVersion
	if contract.ChainVersion != 0 {
		return types.ResponseCheckTx{
			Code: bcerrors.ErrCodeLowLevelError,
			Log:  "v1 transaction cannot use other version contract",
		}
	}

	// check orgID and name
	if !(contract.OrgID == statedbhelper.GetGenesisOrgID(0, 0) &&
		contract.ChainVersion == 0 &&
		(contract.Name == "token-basic" || strings.HasPrefix(contract.Name, "token-templet-"))) {
		return conn.runCheckBCTx(result.TxV1Result.FromAddr, bctx.Transaction(result.TxV1Result.Transaction))
	}

	appState := statedbhelper.GetWorldAppState(0, 0)

	effectContract := statedbhelper.GetEffectContractByName(0, 0, appState.BeginBlock.Header.Height+1, contract.Name, contract.OrgID)
	if effectContract == nil {
		return types.ResponseCheckTx{
			Code: bcerrors.ErrCodeLowLevelError,
			Log:  "cannot get effectContract",
		}
	}
	msg := types2.Message{
		Contract: effectContract.Token,
	}
	msg.MethodID, msg.Items, err = conn.exMessageParams(result.TxV1Result.Transaction.Data)
	if err != nil {
		return conn.runCheckBCTx(result.TxV1Result.FromAddr, bctx.Transaction(result.TxV1Result.Transaction))
	}

	tx2 := types2.Transaction{
		Nonce:    result.TxV1Result.Transaction.Nonce,
		GasLimit: int64(result.TxV1Result.Transaction.GasLimit),
		Note:     result.TxV1Result.Transaction.Note,
		Messages: []types2.Message{msg},
	}

	return connV2.RunCheckTx(result.Tx, tx2, result.TxV1Result.Pubkey)
}

func (conn *CheckConnection) parseTx(tx []byte) (fromAddr string, pubKey crypto.PubKeyEd25519, transaction bctx.Transaction, resp types.ResponseCheckTx) {
	resp.Code = bcerrors.ErrCodeOK

	chainID := conn.stateDB.GetChainID()
	fromAddr, pubKey, err := transaction.TxParse(chainID, string(tx))
	if err != nil {
		conn.logger.Error("tx parse failed:", "error", err)
		bcError := bcerrors.BCError{
			ErrorCode: bcerrors.ErrCodeCheckTxTransData,
			ErrorDesc: "",
		}
		resp = types.ResponseCheckTx{
			Code: bcError.ErrorCode,
			Log:  bcError.Error(),
		}
		return
	}
	// Check note, it must stay within 40 characters limit
	if len(transaction.Note) > bctx.MAX_SIZE_NOTE {
		bcError := bcerrors.BCError{
			ErrorCode: bcerrors.ErrCodeCheckTxNoteExceedLimit,
			ErrorDesc: "",
		}
		resp = types.ResponseCheckTx{
			Code: bcError.ErrorCode,
			Log:  bcError.Error(),
		}
		return
	}
	// Check Nonce
	err = conn.stateDB.CheckAccountNonce(smc.Address(fromAddr), transaction.Nonce)
	if err != nil {
		conn.logger.Error("check nonce error:", err)
		bcError := bcerrors.BCError{
			ErrorCode: bcerrors.ErrCodeCheckTxInvalidNonce,
			ErrorDesc: "",
		}
		resp = types.ResponseCheckTx{
			Code: bcError.ErrorCode,
			Log:  bcError.Error(),
		}
		return
	}

	return
}

// parse transaction's params and create message with them
func (conn *CheckConnection) exMessageParams(data []byte) (methodID uint32, items []common.HexBytes, err error) {
	// DDTode parameter with RLP API to get MethodInfo
	var methodInfo bctx.MethodInfo
	if err = rlp.DecodeBytes(data, &methodInfo); err != nil {
		return
	}

	var itemsBytes = make([][]byte, 0)
	if err = rlp.DecodeBytes(methodInfo.ParamData, &itemsBytes); err != nil {
		return
	}

	switch methodInfo.MethodID {
	case stubapi.ConvertPrototype2ID(prototype.TbTransfer):
		if len(itemsBytes) != 2 {
			err = errors.New("invalid parameter's count")
			return
		}

		to := string(itemsBytes[0][:])
		value := bn.NB(new(big.Int).SetBytes(itemsBytes[1][:]))

		items = tx2.WrapInvokeParams(to, value)
		methodID = transferMethodID
	default:
		err = errors.New("invalid tx")
	}

	return
}

func decode2Uint64(b []byte) uint64 {

	tx8 := make([]byte, 8)
	copy(tx8[len(tx8)-len(b):], b)

	return binary.BigEndian.Uint64(tx8[:])
}
