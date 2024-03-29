package check

import (
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bcbchain/smcrunctl/adapter"
	"github.com/bcbchain/bclib/algorithm"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
	tx2 "github.com/bcbchain/bclib/tx/v2"
	tx3 "github.com/bcbchain/bclib/tx/v3"
	types2 "github.com/bcbchain/bclib/types"
)

// CheckBCTx check tx data
func (app *AppCheck) CheckBCTx(tx []byte) types.ResponseCheckTx {

	if app.chainID == "" {
		app.SetChainID(statedbhelper.GetChainID())
	}

	// for base58
	tx2.Init(app.chainID)
	transaction, pubKey, err := tx2.TxParse(string(tx))
	if err != nil {
		// for base64
		tx3.Init(app.chainID)
		transaction, pubKey, err = tx3.TxParse(string(tx))
		if err != nil {
			app.logger.Error("tx parse failed:", "error", err)
			return types.ResponseCheckTx{
				Code: types2.ErrCheckTx,
				Log:  err.Error()}
		}
	}

	return app.runCheckBCTx(tx, transaction, pubKey)
}

func (app *AppCheck) runCheckBCTx(tx []byte, transaction types2.Transaction, pubKey crypto.PubKeyEd25519) types.ResponseCheckTx {
	// Check note
	if len(transaction.Note) > types2.MaxSizeNote {
		return types.ResponseCheckTx{
			Code: types2.ErrCheckTx,
			Log:  "Invalid transaction note"}
	}

	transID, _ := statedbhelper.NewRollbackTransactionID()
	txID := int64(1)
	// Check Nonce
	err := statedbhelper.CheckAccountNonce(transID, txID, pubKey.Address(statedbhelper.GetChainID()), transaction.Nonce)
	if err != nil {
		app.logger.Debug("check nonce error:", "err", err)
		return types.ResponseCheckTx{
			Code: types2.ErrCheckTx,
			Log:  "Invalid nonce"}
	}

	statedbhelper.BeginBlock(transID)
	defer statedbhelper.RollbackBlock(transID)

	adp := adapter.GetInstance()
	defer adp.Rollback(transID)
	appStat := statedbhelper.GetWorldAppState(0, 0)

	blockHeader := types.Header{}
	if appStat.BlockHeight == 0 {
		blockHeader.ChainID = app.chainID
		blockHeader.Height = 0
	} else {
		blockHeader = appStat.BeginBlock.Header
		blockHeader.Height = blockHeader.Height + 1
	}
	app.logger.Debug("CheckTx", "block height", blockHeader.Height)

	txHash := common.HexBytes(algorithm.CalcCodeHash(string(tx)))
	result := adp.InvokeTx(blockHeader, transID, txID, pubKey.Address(statedbhelper.GetChainID()), transaction, pubKey.Bytes(), txHash, appStat.BeginBlock.Hash)
	if result.Code == types2.CodeBVMQueryOK {
		return types.ResponseCheckTx{
			Code: types2.CodeBVMQueryOK,
			Log:  result.Log,
			Data: result.Data}

	} else if result.Code != types2.CodeOK {
		app.logger.Error("CheckTx failed", "code", result.Code, "error", result.Log)
		return types.ResponseCheckTx{
			Code: result.Code,
			Log:  result.Log}
	}

	return types.ResponseCheckTx{
		Code: types2.CodeOK,
		Log:  "CheckTx success"}
}
