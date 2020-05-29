package query

import (
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	bctx "github.com/bcbchain/bcbchain/abciapp_v1.0/tx/tx"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"strings"
)

func (conn *QueryConnection) query(req types.RequestQuery) (resQuery types.ResponseQuery) {
	var query bctx.Query

	if len(req.Data) != 0 {
		chainID := conn.stateDB.GetChainID()
		addrStr, err := query.QueryDataParse(chainID, string(req.Data))
		if err != nil {
			conn.logger.Error("QueryDataParse parse failed:", err)
			return types.ResponseQuery{
				Code: bcerrors.ErrCodeLowLevelError,
				Log:  err.Error(),
			}
		}

		if strings.HasPrefix(query.QueryKey, "/account") { //如果key包含了签名的地址才让查询
			if !strings.Contains(query.QueryKey, addrStr) {
				conn.logger.Error("Query only can query it self,but get other address")
				bcerr := bcerrors.BCError{
					ErrorCode: bcerrors.ErrCodeNoAuthorization,
					ErrorDesc: "",
				}
				return types.ResponseQuery{
					Code: bcerr.ErrorCode,
					Log:  bcerr.Error(),
				}
			}
		}

	} else if req.Path != "" {
		query.QueryKey = req.Path
	}

	conn.logger.Trace("key info:", "key:", query.QueryKey)
	var kBytes []byte
	kBytes, err := conn.stateDB.Get(query.QueryKey)
	if err != nil {
		conn.logger.Fatal("query DB failed ", "error", err)
		panic(err)
	}
	conn.logger.Trace("value info:", "value:", kBytes)

	return types.ResponseQuery{
		Code:  bcerrors.ErrCodeOK,
		Key:   []byte(query.QueryKey),
		Value: kBytes,
	}
}
