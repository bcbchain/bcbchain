package adapter

import (
	"blockchain/abciapp/service/query"
	"blockchain/common/statedbhelper"
	"blockchain/smcsdk/sdk/jsoniter"
	"blockchain/smcsdk/sdk/std"
	types2 "blockchain/smcsdk/sdk/types"
	rpcclient "common/rpc/lib/client"
	"common/socket"
	"errors"
	"fmt"
	"github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tmlibs/log"
)

//Routes defines RPC functions route map
//var Routes = map[string]*rpcserver.RPCFunc{
//	"get":   rpcserver.NewRPCFunc(SdbGet, "transID,txID,key"),
//	"set":   rpcserver.NewRPCFunc(SdbSet, "transID,txID,data"),
//	"build": rpcserver.NewRPCFunc(SdbBuild, "transID,txID,contractMeta"),
//	"block": rpcserver.NewRPCFunc(GetBlock, "height"),
//}
var Routes = map[string]socket.CallBackFunc{
	"get":   SdbGet,
	"set":   SdbSet,
	"build": SdbBuild,
	"block": GetBlock,
}

var logger log.Logger

func SetLogger(l log.Logger) {
	logger = l
}

//SdbGet calls sdb get function
func SdbGet(req map[string]interface{}) (result interface{}, err error) {

	transID := int64(req["transID"].(float64))
	txID := int64(req["txID"].(float64))
	key := req["key"].(string)

	rBytes, err := Get(transID, txID, key)
	return string(rBytes), err
}

//SdbSet calls sdb set function
func SdbSet(req map[string]interface{}) (result interface{}, err error) {

	transID := int64(req["transID"].(float64))
	txID := int64(req["txID"].(float64))
	mData := req["data"].(map[string]interface{})
	data := make(map[string][]byte)
	for k, v := range mData {
		data[k] = []byte(v.(string))
	}

	return Set(transID, txID, data)
}

//SdbBuild calls sdb build function
func SdbBuild(req map[string]interface{}) (result interface{}, err error) {

	transID := int64(req["transID"].(float64))
	txID := int64(req["txID"].(float64))
	resBytes := []byte(req["contractMeta"].(string))
	var contractMeta std.ContractMeta
	err = jsoniter.Unmarshal(resBytes, &contractMeta)
	if err != nil {
		return
	}

	buildResult, err := Build(transID, txID, contractMeta)
	if err != nil {
		return
	} else {
		resBytes, _ = jsoniter.Marshal(buildResult)
		return string(resBytes), nil
	}
}

func GetBlock(req map[string]interface{}) (result interface{}, err error) {

	height := int64(req["height"].(float64))
	if height < 0 {
		err = errors.New(fmt.Sprintf("invalid height=%d", height))
		return
	}
	logger.Debug("Adapter RPC", "get block", height)
	if height == 0 {
		appState := statedbhelper.GetWorldAppState(0, 0)

		height = appState.BlockHeight
	}

	url := query.TmCoreURL
	if url == "" {
		err = errors.New("can not get tendermint url")
		return nil, err
	}
	logger.Debug("Adapter RPC", "query RPC URL", url)

	res := new(core_types.ResultBlock)
	rpc := rpcclient.NewJSONRPCClientEx(url, "", true)
	_, err = rpc.Call("block", map[string]interface{}{"height": height}, res)
	if err != nil {
		logger.Error("Adapter RPC", "query block error", err.Error())
		return nil, err
	}

	b := std.Block{
		ChainID:         res.BlockMeta.Header.ChainID,
		BlockHash:       types2.Hash(res.BlockMeta.BlockID.Hash),
		Height:          res.BlockMeta.Header.Height,
		Time:            res.BlockMeta.Header.Time.Unix(),
		NumTxs:          int32(res.BlockMeta.Header.NumTxs),
		DataHash:        types2.Hash(res.BlockMeta.Header.DataHash),
		ProposerAddress: res.BlockMeta.Header.ProposerAddress,
		RewardAddress:   res.BlockMeta.Header.RewardAddress,
		RandomNumber:    types2.HexBytes(res.BlockMeta.Header.RandomOfBlock),
		LastBlockHash:   types2.Hash(res.BlockMeta.BlockID.Hash),
		LastCommitHash:  types2.Hash(res.BlockMeta.Header.LastCommitHash),
		LastAppHash:     types2.Hash(res.BlockMeta.Header.LastAppHash),
		LastFee:         int64(res.BlockMeta.Header.LastFee),
		Version:         *res.BlockMeta.Header.Version,
	}

	resultByte, err := jsoniter.Marshal(b)
	if err != nil {
		return nil, err
	}
	result = string(resultByte)
	return
}
