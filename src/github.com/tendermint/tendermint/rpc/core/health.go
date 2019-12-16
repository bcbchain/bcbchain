package core

import (
	"github.com/pkg/errors"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/version"
)

// Get node health. Returns empty result (200 OK) on success, no response - in
// case of an error.
//
// ```shell
// curl 'localhost:46657/health'
// ```
//
// ```go
// client := client.NewHTTP("tcp://0.0.0.0:46657", "/websocket")
// result, err := client.Health()
// ```
//
// > The above command returns JSON structured like this:
//
// ```json
// {
// 	"error": "",
// 	"result": {},
// 	"id": "",
// 	"jsonrpc": "2.0"
// }
// ```
func Health() (*ctypes.ResultHealth, error) {
	if completeStarted == false {
		return nil, errors.New("wait application complete started")
	}

	state := consensusState.GetState()

	return &ctypes.ResultHealth{
		ChainID:         genDoc.ChainID,
		Version:         version.Version,
		ChainVersion:    state.ChainVersion,
		LastBlockHeight: state.LastBlockHeight,
		ValidatorCount:  int64(len(state.Validators.Validators))}, nil
}
