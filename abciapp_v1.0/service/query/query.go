package query

import (
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/statedb"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
)

type QueryConnection struct {
	logger  log.Loggerf
	stateDB *statedb.StateDB
}

func (conn *QueryConnection) SetLogger(logger log.Loggerf) {
	conn.logger = logger
}

func (conn *QueryConnection) NewStateDB() {
	conn.stateDB = statedb.NewStateDB()
}

func (conn *QueryConnection) Echo(req types.RequestEcho) types.ResponseEcho {
	conn.logger.Debug("Recv ABCI interface: Echo")
	return types.ResponseEcho{Message: req.Message}
}

func (conn *QueryConnection) Info(req types.RequestInfo) (resInfo types.ResponseInfo) {
	conn.logger.Debug("Recv ABCI interface: Info")
	return conn.BCInfo(req)
}

func (conn *QueryConnection) SetOption(req types.RequestSetOption) types.ResponseSetOption {
	conn.logger.Debug("Recv ABCI interface: SetOption")
	return types.ResponseSetOption{Code: bcerrors.ErrCodeOK}
}

func (conn *QueryConnection) Query(req types.RequestQuery) (resQuery types.ResponseQuery) {
	conn.logger.Debug("Recv ABCI interface: Query", "path", req.Path, "msg", string(req.Data))
	return conn.query(req)
}
