package deliver

import (
	"github.com/bcbchain/bcbchain/abciapp/softforks"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"bytes"
	"container/list"
	"encoding/hex"
	"fmt"
	abci "github.com/bcbchain/bclib/tendermint/abci/types"
)

func (conn *DeliverConnection) BCBeginBlock(req abci.RequestBeginBlock) abci.ResponseBeginBlock {

	conn.logger.Info("Recv ABCI interface: BeginBlock")
	conn.logger.Info("  block", "height", req.Header.Height, "last_app_hash", hex.EncodeToString(req.Header.LastAppHash))
	conn.logger.Info("  block", "height", req.Header.Height, "proposer", req.Header.ProposerAddress)
	conn.logger.Info("  block", "height", req.Header.Height, "reward", req.Header.RewardAddress)

	conn.sponser = req.Header.ProposerAddress
	conn.rewarder = req.Header.RewardAddress
	conn.blockHash = req.Hash
	conn.blockHeader = req.Header

	var err error
	conn.appState, err = conn.stateDB.GetWorldAppState()
	if err != nil {
		conn.logger.Fatal("failed to read app state & hash from stateDB", "error", err)
		panic(err)
	}

	//检查区块高度
	if req.Header.Height != conn.appState.BlockHeight+1 {
		conn.logger.Fatal("failed to match block height",
			"abci_height", conn.appState.BlockHeight,
			"block_height", req.Header.Height)

		panic("failed to match block height")
	}

	//判断StateDB中保存的appHash是否正确,跳过第一个区块
	if !bytes.EqualFold(req.Header.LastAppHash, conn.appState.AppHash) {
		conn.logger.Fatal("failed to match app hash",
			"abci_app_hash", conn.appState.AppHash,
			"block_last_app_hash", req.Header.LastAppHash)

		panic(fmt.Sprintf("failed to match app hash, req.Header.LastAppHash %x:%d, conn.appState.AppHash:%x:%d",
			req.Header.LastAppHash, req.Header.Height, conn.appState.AppHash, conn.appState.BlockHeight))
	}

	conn.appState.BlockHeight = req.Header.Height //保存最新的高度
	conn.appState.BeginBlock = req

	conn.hashList = list.New().Init()

	//reset fee & rewards for the block
	conn.fee = 0
	// Fixs bug #2092. For backward compatibility, once when the block reach the specified height,
	// using the correct function to record rewards data in block
	if softforks.V1_0_2_3233(conn.appState.BlockHeight) {
		conn.logger.Debug("BeginBlock: V1_0_2_3233 softfork is unavailable")
	} else {
		conn.rewards = map[string]uint64{}
		conn.logger.Debug("BeginBlock:  V1_0_2_3233 softfork is available")
	}
	//stateDB开始做缓存
	_, transaction := statedbhelper.NewCommittableTransactionID()
	conn.stateDB.BeginBlock(transaction)

	return abci.ResponseBeginBlock{Code: bcerrors.ErrCodeOK}
}

func (conn *DeliverConnection) BCBeginBlockToV2(req abci.RequestBeginBlock) {

	conn.sponser = req.Header.ProposerAddress
	conn.rewarder = req.Header.RewardAddress
	conn.blockHash = req.Hash
	conn.blockHeader = req.Header

	var err error
	conn.appState, err = conn.stateDB.GetWorldAppState()
	if err != nil {
		conn.logger.Fatal("failed to read app state & hash from stateDB", "error", err)
		panic(err)
	}

	conn.appState.BlockHeight = req.Header.Height //保存最新的高度
	conn.appState.BeginBlock = req

	conn.hashList = list.New().Init()

	//reset fee & rewards for the block
	conn.fee = 0
	// Fixs bug #2092. For backward compatibility, once when the block reach the specified height,
	// using the correct function to record rewards data in block
	if softforks.V1_0_2_3233(conn.appState.BlockHeight) {
		conn.logger.Debug("BeginBlock: V1_0_2_3233 softfork is unavailable")
	} else {
		conn.rewards = map[string]uint64{}
		conn.logger.Debug("BeginBlock:  V1_0_2_3233 softfork is available")
	}
	//stateDB开始做缓存
	_, transaction := statedbhelper.NewCommittableTransactionID()
	conn.stateDB.BeginBlock(transaction)
}
