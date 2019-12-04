package burrowrpc

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
	core_types "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tmlibs/log"
)

var Routes = map[string]socket.CallBackFunc{
	"block": GetBlock,
}

var logger log.Logger

func SetLogger(l log.Logger) {
	logger = l
}

func GetBlock(req map[string]interface{}) (result interface{}, err error) {
	height := req["height"].(int64)
	if height < 0 {
		err = errors.New(fmt.Sprintf("invalid height=%d", height))
		return
	}

	if height == 0 {
		appState := statedbhelper.GetWorldAppState(0, 0)

		height = appState.BlockHeight
	}

	url := query.TmCoreURL

	if url == "" {
		err = errors.New("can not get tendermint url")
		return nil, err
	}

	res := new(core_types.ResultBlock)
	rpc := rpcclient.NewJSONRPCClientEx(url, "", true)
	_, err = rpc.Call("block", map[string]interface{}{"height": height}, res)
	if err != nil {
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
