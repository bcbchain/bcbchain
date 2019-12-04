package deliver

import (
	"blockchain/common/statedbhelper"
	"blockchain/smcrunctl/adapter"
	"encoding/binary"
	"encoding/hex"
	"sort"

	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/tmlibs/common"
	"golang.org/x/crypto/sha3"
)

//nolint unhandled
func (app *AppDeliver) commit() abci.ResponseCommit {
	app.logger.Info("Recv ABCI interface: Commit",
		"height", app.appState.BlockHeight)

	// For empty block, its apphash is exactly same to the last one.
	if app.hashList.Len() == 0 {
		return app.commitEmptyBlock()
	}

	//txHashList
	hasherSHA3256 := sha3.New256()
	hashListBytes := make([]crypto.Hash, 0)
	for txHash := app.hashList.Front(); txHash != nil; txHash = txHash.Next() {
		deliverHash := txHash.Value.([]byte)
		hasherSHA3256.Write(deliverHash)
		hashListBytes = append(hashListBytes, crypto.Hash(deliverHash))
		app.logger.Info("Commit: txHash", "txHash", hex.EncodeToString(deliverHash))
	}
	hashListSha := hasherSHA3256.Sum(nil)
	appHash := sha3.New256()
	appHash.Write(app.appState.AppHash)
	appHash.Write(hashListSha)
	app.logger.Debug("666666 calc appHash", "lastAppHash", app.appState.AppHash, "hashListSha", hashListSha)
	app.appState.AppHash = appHash.Sum(nil)
	app.appState.TxsHashList = hashListBytes
	app.logger.Info("commitTx", "appHash", app.appState.AppHash)

	//rewards
	keys := make([]string, 0)
	for k := range app.rewards {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	kvps := make([]common.KVPair, len(keys))
	for i, k := range keys {
		uintByte := make([]byte, 8)
		binary.BigEndian.PutUint64(uintByte, uint64(app.rewards[k]))
		kvps[i] = common.KVPair{Key: []byte(k), Value: uintByte}
	}
	app.appState.Rewards = kvps
	app.appState.Fee = uint64(app.fee)

	// SetWorldAppState and commit block
	app.commitBlock()
	//SDK commit
	adapter.GetInstance().Commit(app.transID)
	return abci.ResponseCommit{AppState: abci.AppStateToByte(app.appState)}
}

func (app *AppDeliver) commitEmptyBlock() abci.ResponseCommit {
	app.commitBlock()
	// app hash of empty block is exactly same to the last one.
	appState := &abci.AppState{
		BlockHeight: app.appState.BlockHeight,
		AppHash:     app.appState.AppHash,
	}
	return abci.ResponseCommit{AppState: abci.AppStateToByte(appState)}
}

func (app *AppDeliver) commitBlock() {
	statedbhelper.SetWorldAppState(app.transID, app.txID, app.appState)

	statedbhelper.CommitTx(app.transID, app.txID)
	statedbhelper.CommitBlock(app.transID)
	app.sponser = "" //清空提案者
	app.rewarder = ""
	app.logger.Info("  commit",
		"height", app.appState.BlockHeight,
		"app_hash", hex.EncodeToString(app.appState.AppHash))
}
