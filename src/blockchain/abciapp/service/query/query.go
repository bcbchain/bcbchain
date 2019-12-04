package query

import (
	bctypes "blockchain/types"

	"github.com/tendermint/abci/types"
	"github.com/tendermint/tmlibs/log"
)

type QueryConnection struct {
	logger log.Logger
}

func (conn *QueryConnection) SetLogger(logger log.Logger) {
	conn.logger = logger
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
	return types.ResponseSetOption{Code: bctypes.CodeOK}
}

func (conn *QueryConnection) Query(req types.RequestQuery) (resQuery types.ResponseQuery) {
	conn.logger.Debug("Recv ABCI interface: Query", "path", req.Path, "msg", string(req.Data))
	return conn.query(req)
}

func (conn *QueryConnection) QueryEx(req types.RequestQueryEx) (resQueryEx types.ResponseQueryEx) {
	conn.logger.Debug("Recv ABCI interface: QueryEx", "path", req.Path)
	return conn.queryEx(req)
}
