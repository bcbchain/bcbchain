package deliver

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	types3 "github.com/bcbchain/bcbchain/abciapp/service/types"
	statedb2 "github.com/bcbchain/bcbchain/statedb"
	"github.com/bcbchain/bclib/tx/v1"
	"math/big"
	"strconv"
	"strings"

	"github.com/bcbchain/bcbchain/abciapp/service/deliver"
	"github.com/bcbchain/bcbchain/abciapp/softforks"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/prototype"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/statedb"
	bctx "github.com/bcbchain/bcbchain/abciapp_v1.0/tx/tx"
	bctypes "github.com/bcbchain/bcbchain/abciapp_v1.0/types"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bclib/bn"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
	tx2 "github.com/bcbchain/bclib/tx/v2"
	types2 "github.com/bcbchain/bclib/types"
	"github.com/bcbchain/sdk/sdk/rlp"
	"github.com/bcbchain/sdk/sdk/std"
)

const (
	transferMethodID = 0x44d8ca60
)

func (conn *DeliverConnection) deliverBCTx(tx []byte, connV2 *deliver.AppDeliver) types.ResponseDeliverTx {

	conn.logger.Info("Recv ABCI interface: DeliverTx", "tx", string(tx))

	fromAddr, pubKey, transaction, bFailed, bcError := conn.parseTx(tx)
	if bcError.ErrorCode != bcerrors.ErrCodeOK {
		return types.ResponseDeliverTx{
			Code: bcError.ErrorCode,
			Log:  bcError.Error(),
		}
	}

	if connV2 != nil {
		return conn.runDeliverBCTxEx(tx, bFailed, fromAddr, pubKey, transaction, connV2)
	}

	return conn.runDeliverBCTx(tx, bFailed, fromAddr, transaction, connV2)
}

func (conn *DeliverConnection) deliverBCTxCurrency(tx []byte, connV2 *deliver.AppDeliver) types3.Result2 {

	conn.logger.Info("Recv ABCI interface: DeliverTxCurrency", "tx", string(tx))

	var result types3.Result2
	fromAddr, pubKey, transaction, bFailed, bcError := conn.parseTx(tx)

	if bcError.ErrorCode != bcerrors.ErrCodeOK {
		result.ErrorLog = errors.New(bcError.Error())
	}
	result.Tx = tx
	result.TxV1Result.FromAddr = fromAddr
	result.TxV1Result.Transaction = tx1.Transaction(transaction)
	result.TxV1Result.BFailed = bFailed

	if connV2 != nil {
		//return conn.runDeliverBCTxEx(tx, bFailed, fromAddr, pubKey, transaction, connV2)
		result.TxV1Result.Pubkey = pubKey
	}

	return result
	//return conn.runDeliverBCTx(tx, bFailed, fromAddr, transaction, connV2)
}

func (conn *DeliverConnection) runDeliverBCTx(tx []byte, bFailed bool, fromAddr smc.Address, transaction bctx.Transaction, connV2 *deliver.AppDeliver) (resDeliverTx types.ResponseDeliverTx) {

	var transID int64
	if connV2 != nil {
		transID = connV2.TransID()
	}
	var bcError smc.Error

	if !bFailed {
		_, err := conn.stateDB.SetAccountNonce(smc.Address(fromAddr), transaction.Nonce)
		if err != nil {
			bFailed = true

			conn.logger.Error("set account nonce error:", err)
			bcError.ErrorCode = bcerrors.ErrCodeDeliverTxInvalidNonce
			bcError.ErrorDesc = ""
		}
	}

	var err error
	var txState *statedb.TxState
	var contract *bctypes.Contract
	if !bFailed {
		//Generate txState to operate stateDB
		txState = conn.stateDB.NewTxState(smc.Address(transaction.To), smc.Address(fromAddr))

		// Get contract detailed information depends on contract address
		contract, err = conn.stateDB.GetContract(smc.Address(transaction.To))
		if err != nil {
			bFailed = true

			conn.logger.Error("failed json.Unmarshal(contractBytes,contract)", "error", err)
			bcError.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcError.ErrorDesc = err.Error()

		}
		if !bFailed && contract == nil {
			bFailed = true

			conn.logger.Error("can't find this contract from DB", "contract address", transaction.To)
			bcError.ErrorCode = bcerrors.ErrCodeDeliverTxNoContract
			bcError.ErrorDesc = ""

		}
	}

	if bFailed {
		// write failure status into hash
		resDeliverTx = types.ResponseDeliverTx{
			Code: bcError.ErrorCode,
			Log:  bcError.Error(),
		}

		conn.calcDeliverTxHash(tx, &resDeliverTx, nil, connV2)

		return resDeliverTx
	}

	// Generate accounts and execute
	sender := &stubapi.Account{
		smc.Address(fromAddr),
		txState,
	}

	owner := &stubapi.Account{
		contract.Owner,
		txState,
	}
	proposer := &stubapi.Account{
		Addr: smc.Address(conn.sponser),
		//gTokenState,
	}
	rewarder := &stubapi.Account{
		Addr: smc.Address(conn.rewarder),
		//gTokenState,
	}
	invokeContext := &stubapi.InvokeContext{
		sender,
		owner,
		txState,
		conn.blockHash,
		conn.blockHeader,
		proposer,
		rewarder,
		transaction.GasLimit,
		transaction.Note,
	}

	item := &stubapi.InvokeParams{
		invokeContext,
		transaction.Data,
	}

	conn.logger.Debug("start invoke.....")

	//write response into hash
	response, bcerr := conn.docker.Invoke(item, transID)
	if bcerr.ErrorCode != bcerrors.ErrCodeOK {

		txState.RollbackTx()
		conn.logger.Error("docker invoke error.....", "error", bcerr.Error())
		resDeliverTx = types.ResponseDeliverTx{
			Code:     bcerr.ErrorCode,
			Log:      bcerr.Error(),
			GasLimit: transaction.GasLimit,
			GasUsed:  response.GasUsed,
			Fee:      response.GasPrice * response.GasUsed,
		}

		conn.calcDeliverTxHash(tx, &resDeliverTx, nil, connV2)

		return resDeliverTx
	}

	conn.NameVersion = response.Log
	resDeliverTx = types.ResponseDeliverTx{
		Code:     bcerrors.ErrCodeOK,
		Tags:     response.Tags,
		Log:      "Deliver tx succeed",
		GasLimit: transaction.GasLimit,
		GasUsed:  response.GasUsed,
		Fee:      response.GasPrice * response.GasUsed,
		Data:     response.Data,
	}

	if response.Code == stubapi.RESPONSE_CODE_UPDATE_VALIDATORS {
		conn.udValidator = true
		conn.validators = append(conn.validators, response.Data)
		resDeliverTx.Data = ""
	} else if response.Code == stubapi.RESPONSE_CODE_RUNUPGRADE1TO2 {
		conn.appState.ChainVersion, _ = strconv.ParseInt(response.Data, 10, 64)
		resDeliverTx.Data = ""
	} else {
		conn.RespCode = response.Code
		conn.RespData = response.Data
	}

	conn.logger.Debug("deliverBCTx()", "resDeliverTx length", len(resDeliverTx.String()), "resDeliverTx", resDeliverTx.String())

	stateTx, _ := txState.CommitTx()

	conn.logger.Debug("deliverBCTx() ", "stateTx length", len(stateTx), "stateTx ", string(stateTx))

	conn.calcDeliverTxHash(tx, &resDeliverTx, stateTx, connV2)

	//calculate Fee, will use safeAdd() method
	conn.fee = conn.fee + resDeliverTx.Fee

	// Fixs bug #2092. For backward compatibility, once when the block reach the specified height,
	// using the correct function to record rewards data in block
	if softforks.V1_0_2_3233(conn.appState.BlockHeight) {
		conn.rewards = map[string]uint64{}
		conn.logger.Debug("DeliverTx:  V1_0_2_3233 softfork is unavailable")
	} else {
		conn.logger.Debug("DeliverTx:  V1_0_2_3233 softfork is avalible")
	}
	// calculate amount of fee for each reward
	for k, v := range response.RewardValues {
		conn.rewards[k] = conn.rewards[k] + v
	}

	// if connV2 not nil, then add rewardValues,fee and deliverHash to connV2
	if connV2 != nil {
		connV2.AddFee(int64(resDeliverTx.Fee))
		connV2.AddRewardValues(response.RewardValues)
	}

	conn.logger.Debug("deliverBCTx()", "conn.fee", conn.fee, "conn.rewards", map2String(conn.rewards))
	conn.logger.Debug("end deliver invoke.....")

	return resDeliverTx
}

func (conn *DeliverConnection) runDeliverBCTxEx(tx []byte,
	bFailed bool,
	fromAddr smc.Address,
	pubKey crypto.PubKeyEd25519,
	tx1 bctx.Transaction,
	connV2 *deliver.AppDeliver) types.ResponseDeliverTx {

	contract, err := conn.stateDB.GetContract(tx1.To)
	if err != nil {
		return types.ResponseDeliverTx{
			Code: bcerrors.ErrCodeLowLevelError,
			Log:  err.Error(),
		}
	}

	if contract == nil {
		return types.ResponseDeliverTx{
			Code: bcerrors.ErrCodeLowLevelError,
			Log:  "invalid smcAddress"}
	}

	// check chainVersion
	if contract.ChainVersion != 0 {
		return types.ResponseDeliverTx{
			Code: bcerrors.ErrCodeLowLevelError,
			Log:  "v1 transaction cannot use other version contract",
		}
	}

	// check orgID and name
	if !(contract.OrgID == statedbhelper.GetGenesisOrgID(0, 0) &&
		contract.ChainVersion == 0 &&
		(contract.Name == "token-basic" || strings.HasPrefix(contract.Name, "token-templet-"))) {
		return conn.runDeliverBCTx(tx, bFailed, fromAddr, tx1, connV2)
	}

	effectContract := statedbhelper.GetEffectContractByName(0, 0, conn.appState.BeginBlock.Header.Height, contract.Name, contract.OrgID)
	if effectContract == nil {
		return types.ResponseDeliverTx{
			Code: bcerrors.ErrCodeLowLevelError,
			Log:  "cannot get effectContract",
		}
	}
	msg := types2.Message{
		Contract: effectContract.Token,
	}
	msg.MethodID, msg.Items, err = conn.exMessageParams(tx1.Data)
	if err != nil {
		return conn.runDeliverBCTx(tx, bFailed, fromAddr, tx1, connV2)
	}

	tx2 := types2.Transaction{
		Nonce:    tx1.Nonce,
		GasLimit: int64(tx1.GasLimit),
		Note:     tx1.Note,
		Messages: []types2.Message{msg},
	}

	res, _ := connV2.RunDeliverTx(tx, tx2, pubKey)

	return res
}

func (conn *DeliverConnection) parseTx(tx []byte) (fromAddr string, pubKey crypto.PubKeyEd25519, transaction bctx.Transaction, bFailed bool, bcError bcerrors.BCError) {

	bcError.ErrorCode = bcerrors.ErrCodeOK

	chainID := conn.stateDB.GetChainID()

	fromAddr, pubKey, err := transaction.TxParse(chainID, string(tx))
	if err != nil {
		bFailed = true

		conn.logger.Error("tx parse failed:", err)
		bcError.ErrorCode = bcerrors.ErrCodeDeliverTxTransData
		bcError.ErrorDesc = ""

	}

	// Check note, it must stay within 256 characters limit
	if !bFailed && len(transaction.Note) > bctx.MAX_SIZE_NOTE {
		bFailed = true

		bcError.ErrorCode = bcerrors.ErrCodeCheckTxNoteExceedLimit
		bcError.ErrorDesc = ""

	}

	return
}

func (conn *DeliverConnection) exMessageParams(data []byte) (methodID uint32, items []common.HexBytes, err error) {
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

func (conn *DeliverConnection) calcDeliverTxHash(tx []byte, response *types.ResponseDeliverTx, stateTx []byte, connV2 *deliver.AppDeliver) {
	md5TX := md5.New()
	if tx != nil {
		md5TX.Write(tx)
	}
	if response != nil {
		md5TX.Write([]byte(response.String()))
	}
	if stateTx != nil {
		md5TX.Write(stateTx)
	}

	deliverHash := md5TX.Sum(nil)
	// v1代码中未判断三个输入参数是否为空的情况并已将线上程序升级，
	// 所以当链未升级前执行if代码，链升级后执行else if代码
	conn.logger.Debug("calcDeliverTxHash() ", "resp==nil", response == nil, "stateTx", string(stateTx))
	conn.logger.Debug("calcDeliverTxHash() ", "resp", response.String(), "deliverHash", hex.EncodeToString(deliverHash))
	if connV2 == nil {
		conn.hashList.PushBack(deliverHash)
	} else if tx != nil || response != nil || stateTx != nil {
		conn.hashList.PushBack(deliverHash)
	}

	if connV2 != nil {
		connV2.AddDeliverHash(deliverHash)
	}
}

func mapFee2String(m map[smc.Address]std.Fee) string {
	b := new(bytes.Buffer)
	b.WriteString("{")
	for key, value := range m {
		_, _ = fmt.Fprintf(b, "%s:'%s',", key, value.String())
	}
	b.WriteString("}")
	return b.String()
}

func map2String(m map[smc.Address]uint64) string {
	b := new(bytes.Buffer)
	b.WriteString("{")
	for key, value := range m {
		_, _ = fmt.Fprintf(b, "%s:%d,", key, value)
	}
	b.WriteString("}")
	return b.String()
}

func (conn *DeliverConnection) RunExecTx(tx *statedb2.Tx, params ...interface{}) (doneSuccess bool, response interface{}) {

	transaction := params[0].(bctx.Transaction)
	fromAddr := params[1].(smc.Address)
	transID := params[2].(int64)

	var bcError smc.Error
	var bFailed bool

	var err error
	var txState *statedb.TxState
	var contract *bctypes.Contract
	if !bFailed {
		//Generate txState to operate stateDB
		txState = &statedb.TxState{
			StateDB:         conn.stateDB,
			ContractAddress: transaction.To,
			SenderAddress:   fromAddr,
			Tx:              tx,
		}
		//txState = conn.stateDB.NewTxState(transaction.To, fromAddr)

		// Get contract detailed information depends on contract address
		contract, err = conn.stateDB.GetContract(transaction.To)
		if err != nil {
			bFailed = true

			conn.logger.Error("failed json.Unmarshal(contractBytes,contract)", "error", err)
			bcError.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcError.ErrorDesc = err.Error()

		}
		if !bFailed && contract == nil {
			bFailed = true

			conn.logger.Error("can't find this contract from DB", "contract address", transaction.To)
			bcError.ErrorCode = bcerrors.ErrCodeDeliverTxNoContract
			bcError.ErrorDesc = ""

		}
	}

	if bFailed {
		// write failure status into hash
		res := response.(types.ResponseDeliverTx)
		res.Code = bcError.ErrorCode
		res.Log = bcError.Error()

		return true, res
	}

	// Generate accounts and execute
	sender := &stubapi.Account{
		Addr:    fromAddr,
		TxState: txState,
	}

	owner := &stubapi.Account{
		Addr:    contract.Owner,
		TxState: txState,
	}

	proposer := &stubapi.Account{
		Addr: conn.sponser,
		//gTokenState,
	}
	rewarder := &stubapi.Account{
		Addr: conn.rewarder,
		//gTokenState,
	}

	invokeContext := &stubapi.InvokeContext{
		Sender:      sender,
		Owner:       owner,
		TxState:     txState,
		BlockHash:   conn.blockHash,
		BlockHeader: conn.blockHeader,
		Proposer:    proposer,
		Rewarder:    rewarder,
		GasLimit:    transaction.GasLimit,
		Note:        transaction.Note,
	}

	item := &stubapi.InvokeParams{
		Ctx:    invokeContext,
		Params: transaction.Data,
	}

	conn.logger.Debug("start invoke.....")

	//write response into hash
	invokeRes, bcerr := conn.docker.Invoke(item, transID)
	if bcerr.ErrorCode != bcerrors.ErrCodeOK {
		conn.logger.Error("docker invoke error.....", "error", bcerr.Error())
		txState.RollbackTx()
		invokeRes.ErrCode = bcerr.ErrorCode
		invokeRes.ErrLog = bcerr.Error()
	}

	return true, invokeRes
}

func (conn *DeliverConnection) HandleResponse(
	tx *statedb2.Tx,
	txStr string,
	rawTxV1 *bctx.Transaction,
	response stubapi.Response,
	connV2 *deliver.AppDeliver) (resDeliverTx types.ResponseDeliverTx) {

	if response.ErrCode != 0 {
		resDeliverTx = types.ResponseDeliverTx{
			Code:     response.ErrCode,
			Log:      response.ErrLog,
			GasLimit: rawTxV1.GasLimit,
			GasUsed:  response.GasUsed,
			Fee:      response.GasPrice * response.GasUsed,
		}
		conn.calcDeliverTxHash([]byte(txStr), &resDeliverTx, nil, connV2)
		return
	}

	conn.NameVersion = response.Log
	resDeliverTx = types.ResponseDeliverTx{
		Code:     bcerrors.ErrCodeOK,
		Tags:     response.Tags,
		Log:      "Deliver tx succeed",
		GasLimit: rawTxV1.GasLimit,
		GasUsed:  response.GasUsed,
		Fee:      response.GasPrice * response.GasUsed,
		Data:     response.Data,
	}

	if response.Code == stubapi.RESPONSE_CODE_UPDATE_VALIDATORS {
		conn.udValidator = true
		conn.validators = append(conn.validators, response.Data)
		resDeliverTx.Data = ""
	} else if response.Code == stubapi.RESPONSE_CODE_RUNUPGRADE1TO2 {
		conn.appState.ChainVersion, _ = strconv.ParseInt(response.Data, 10, 64)
		resDeliverTx.Data = ""
	} else {
		conn.RespCode = response.Code
		conn.RespData = response.Data
	}

	conn.logger.Debug("deliverBCTx()", "resDeliverTx length", len(resDeliverTx.String()), "resDeliverTx", resDeliverTx.String())

	stateTx, _ := tx.GetBuffer()

	conn.logger.Debug("deliverBCTx() ", "stateTx length", len(stateTx), "stateTx ", string(stateTx))

	conn.calcDeliverTxHash([]byte(txStr), &resDeliverTx, stateTx, connV2)

	//calculate Fee, will use safeAdd() method
	conn.fee = conn.fee + resDeliverTx.Fee

	// Fixs bug #2092. For backward compatibility, once when the block reach the specified height,
	// using the correct function to record rewards data in block
	if softforks.V1_0_2_3233(conn.appState.BlockHeight) {
		conn.rewards = map[string]uint64{}
		conn.logger.Debug("DeliverTx:  V1_0_2_3233 softfork is unavailable")
	} else {
		conn.logger.Debug("DeliverTx:  V1_0_2_3233 softfork is avalible")
	}
	// calculate amount of fee for each reward
	for k, v := range response.RewardValues {
		conn.rewards[k] = conn.rewards[k] + v
	}

	// if connV2 not nil, then add rewardValues,fee and deliverHash to connV2
	if connV2 != nil {
		connV2.AddFee(int64(resDeliverTx.Fee))
		connV2.AddRewardValues(response.RewardValues)
	}

	conn.logger.Debug("deliverBCTx()", "conn.fee", conn.fee, "conn.rewards", map2String(conn.rewards))
	conn.logger.Debug("end deliver invoke.....")

	return
}
