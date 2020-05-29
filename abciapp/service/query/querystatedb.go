package query

import (
	"fmt"
	"strings"

	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	tx2 "github.com/bcbchain/bclib/tx/v2"
	bctypes "github.com/bcbchain/bclib/types"
)

func (conn *QueryConnection) query(req types.RequestQuery) (resQuery types.ResponseQuery) {
	var query bctypes.Query

	if len(req.Data) != 0 {
		chainID := statedbhelper.GetChainID()
		addrStr, query2, err := tx2.QueryDataParse(chainID, string(req.Data))
		if err != nil {
			conn.logger.Error("QueryDataParse parse failed:", err)
			return types.ResponseQuery{
				Code: bctypes.ErrLogicError,
				Log:  err.Error(),
			}
		}
		query = query2
		if strings.HasPrefix(query.QueryKey, "/account") { //如果key包含了签名的地址才让查询
			if !strings.Contains(query.QueryKey, addrStr) {
				conn.logger.Error("Query only can query itself,but get other address")
				bcerr := bctypes.BcError{
					ErrorCode: bctypes.ErrNoAuthorization,
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

	if strings.HasPrefix(query.QueryKey, "/bvm/view/") {
		return BvmViewKey(query.QueryKey, conn.logger)
	}

	conn.logger.Debug("key info:", "key:", req.Path)
	var kBytes []byte
	kBytes, err := statedbhelper.GetFromDB(query.QueryKey)
	if err != nil {
		conn.logger.Fatal("query DB failed ", "error", err)
		panic(err)
	}
	conn.logger.Debug("value info:", "value byte length:", len(kBytes))

	return types.ResponseQuery{
		Code:  types.CodeTypeOK,
		Key:   []byte(req.Path),
		Value: kBytes,
	}
}

func (conn *QueryConnection) queryEx(req types.RequestQueryEx) (resQuery types.ResponseQueryEx) {
	var query bctypes.Query

	if req.Path != "" {
		query.QueryKey = req.Path
	}

	conn.logger.Debug("key info:", "key:", req.Path)
	//提取字符串

	keys, err := ResolvePath(query.QueryKey)
	if err != nil {
		return types.ResponseQueryEx{
			Code: bctypes.ErrPath,
			Log:  err.Error(),
		}
	}
	//var kBytes []byte
	kv := make([]types.KeyValue, len(keys))
	for i, v := range keys {
		kBytes, err := statedbhelper.GetFromDB(v)
		if err != nil {
			conn.logger.Fatal("query DB failed ", "error", err)
			panic(err)
		}
		conn.logger.Debug("value info:", "value byte length:", len(kBytes))
		kv[i].Key = []byte(v)
		kv[i].Value = kBytes
	}

	return types.ResponseQueryEx{
		Code:      types.CodeTypeOK,
		KeyValues: kv,
	}
}

func ResolvePath(path string) ([]string, error) {
	keys := make([]string, 0, 0)

	//判断url的有效性
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("path does not start with '/'")
	}
	if strings.HasSuffix(path, "/") {
		return nil, fmt.Errorf("path cannot end with '/'")
	}
	if !strings.Contains(path, "/[") {
		return nil, fmt.Errorf("path does not contain '/['")
	}
	if strings.Index(path, "/[") > strings.LastIndex(path, "]") {
		return nil, fmt.Errorf("']' is not on the right of '/['")
	}

	//解析url
	subPath := path[strings.Index(path, "/[")+2 : strings.LastIndex(path, "]")]
	elements := strings.Split(subPath, ",")
	for _, v := range elements {
		keys = append(keys, fmt.Sprint(path[:strings.Index(path, "/[")+1], v, path[strings.LastIndex(path, "]")+1:]))
	}

	return keys, nil
}
