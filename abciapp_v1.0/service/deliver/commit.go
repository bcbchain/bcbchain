package deliver

import (
	"encoding/binary"
	"encoding/hex"
	abci "github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
	"golang.org/x/crypto/sha3"
)

func (conn *DeliverConnection) commitTx() abci.ResponseCommit {
	conn.logger.Info("Recv ABCI interface: Commit", "height", conn.appState.BlockHeight)

	// 空区块 返回固定apphash
	if conn.hashList.Len() == 0 {
		err := conn.stateDB.SetWorldAppState(conn.appState) //空区块的apphash与上一块保持一致
		conn.stateDB.CommitBlock()
		conn.sponser = "" //清空提案者
		conn.rewarder = ""
		if err != nil {
			conn.logger.Error("can't set app state&hash to db", "error", err)
			panic(err)
		}
		appState := &abci.AppState{
			BlockHeight: conn.appState.BlockHeight,
			AppHash:     conn.appState.AppHash,
		}

		return abci.ResponseCommit{AppState: abci.AppStateToByte(appState)}
	}

	//txHashList
	hasherSHA3256 := sha3.New256()
	hashListBytes := make([]crypto.Hash, 0)
	for txHash := conn.hashList.Front(); txHash != nil; txHash = txHash.Next() {

		deliverHash := txHash.Value.([]byte)
		hasherSHA3256.Write(deliverHash)
		hashListBytes = append(hashListBytes, crypto.Hash(deliverHash))
		conn.logger.Debug("Commit: txHash", "txHash", hex.EncodeToString(deliverHash))
	}
	hashListSha := hasherSHA3256.Sum(nil)

	appHash := sha3.New256()
	appHash.Write(conn.appState.AppHash)
	appHash.Write(hashListSha)
	conn.appState.AppHash = appHash.Sum(nil)
	conn.logger.Info("commitTx", "appHash", conn.appState.AppHash)

	kvps := make([]common.KVPair, 0)
	for k, v := range conn.rewards {
		uintByte := make([]byte, 8)
		binary.BigEndian.PutUint64(uintByte, v)
		kvps = append(kvps, common.KVPair{[]byte(k), uintByte})
	}

	conn.appState.TxsHashList = hashListBytes
	conn.appState.Rewards = kvps
	conn.appState.Fee = conn.fee

	// set chainVersion
	state, err := conn.stateDB.GetWorldAppState()
	if err != nil {
		conn.logger.Error("can't get app state db", "error", err)
		panic(err)
	}
	conn.appState.BeginBlock.Header.ChainVersion = state.BeginBlock.Header.ChainVersion

	err = conn.stateDB.SetWorldAppState(conn.appState)
	if err != nil {
		conn.logger.Error("can't set app state&hash to db", "error", err)
		panic(err)
	}

	// Commit
	conn.stateDB.CommitBlock()
	conn.sponser = "" //清空提案者
	conn.rewarder = ""

	conn.logger.Info("  commit", "height", conn.appState.BlockHeight, "app_hash", hex.EncodeToString(conn.appState.AppHash))

	return abci.ResponseCommit{AppState: abci.AppStateToByte(conn.appState)}
}
