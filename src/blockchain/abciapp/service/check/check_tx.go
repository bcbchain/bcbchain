package check

import (
	"blockchain/algorithm"
	"blockchain/common/statedbhelper"
	"blockchain/smcrunctl/adapter"
	"blockchain/tx2"
	types2 "blockchain/types"
	"github.com/tendermint/tmlibs/common"

	"github.com/tendermint/abci/types"
	"github.com/tendermint/go-crypto"
)

// CheckBCTx check tx data
func (app *AppCheck) CheckBCTx(tx []byte) types.ResponseCheckTx {

	if app.chainID == "" {
		app.SetChainID(statedbhelper.GetChainID())
	}
	tx2.Init(app.chainID)
	transaction, pubKey, err := tx2.TxParse(string(tx))
	if err != nil {
		app.logger.Error("tx parse failed:", "error", err)
		return types.ResponseCheckTx{
			Code: types2.ErrCheckTx,
			Log:  err.Error()}
	}
	// Check note
	return app.runCheckBCTx(tx, transaction, pubKey)
}

func (app *AppCheck) runCheckBCTx(tx []byte, transaction types2.Transaction, pubKey crypto.PubKeyEd25519) types.ResponseCheckTx {
	// Check note
	if len(transaction.Note) > types2.MaxSizeNote {
		return types.ResponseCheckTx{
			Code: types2.ErrCheckTx,
			Log:  "Invalid transaction note"}
	}

	transID := statedbhelper.NewTransactionID()
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
	if result.Code != types2.CodeOK {
		app.logger.Error("CheckTx failed", "code", result.Code, "error", result.Log)
		return types.ResponseCheckTx{
			Code: result.Code,
			Log:  result.Log}
	}

	return types.ResponseCheckTx{
		Code: types2.CodeOK,
		Log:  "CheckTx success"}
}
