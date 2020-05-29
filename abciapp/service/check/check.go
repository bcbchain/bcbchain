package check

import (
	types2 "github.com/bcbchain/bclib/types"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
)

//AppCheck object of check tx
//nolint
type AppCheck struct {
	logger  log.Logger
	chainID string
}

//SetLogger set logger
func (app *AppCheck) SetLogger(logger log.Logger) {
	app.logger = logger
}

//SetChainID set chainID
func (app *AppCheck) SetChainID(chainID string) {
	app.chainID = chainID
}

//CheckTx check tx
func (app *AppCheck) CheckTx(tx []byte) types.ResponseCheckTx {
	app.logger.Info("Recv ABCI interface: CheckTx", "tx", string(tx))

	return app.CheckBCTx(tx)
}

// ------------- add for support v1 transaction begin ----------------

//RunCheckTx - invoked by v1 checkTx, if it's standard transfer method.
func (app *AppCheck) RunCheckTx(tx []byte, transaction types2.Transaction, pubKey crypto.PubKeyEd25519) types.ResponseCheckTx {
	app.logger.Debug("Recv ABCI interface: CheckTx", "transaction", transaction)

	return app.runCheckBCTx(tx, transaction, pubKey)
}

// ------------- add for support v1 transaction end ----------------
