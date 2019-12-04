package relay

import (
	rpcclient "common/rpc/lib/client"
	jsoniter "github.com/json-iterator/go"
)

// 超时时间包括建立连接和等待返回结果得时间
func getClient(url string) *rpcclient.JSONRPCClient {
	return rpcclient.NewJSONRPCClientExWithTimeout(url, "", true, 5)
}

func abciQueryAndParse(url, path string, data interface{}) (err error) {
	var result *ResultABCIQuery
	if result, err = abciQuery(url, path); err != nil {
		return
	}

	return jsoniter.Unmarshal(result.Response.Value, data)
}

func abciQuery(url, path string) (resultQuery *ResultABCIQuery, err error) {
	resultQuery = new(ResultABCIQuery)
	client := getClient(url)
	_, err = client.Call("abci_query", map[string]interface{}{"path": path}, resultQuery)
	return
}
