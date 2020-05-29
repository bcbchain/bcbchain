package mining

import (
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	. "github.com/bcbchain/bclib/bignumber_v1.0"
	"encoding/json"
)

// Mine the node propose block will be rewarded, the reward amount calculate with block height
func (m *Mining) Mine() (rewardAmount uint64, err smc.Error) {
	err.ErrorCode = bcerrors.ErrCodeOK

	//奖励接收地址
	proposerRewardAddress := m.Block.RewardAddress
	currentHeight, lerror := m.GetCurrentBlockHeight()
	if lerror != nil {
		err.ErrorCode = bcerrors.ErrCodeLowLevelError
		err.ErrorDesc = lerror.Error()
		return
	}

	//初始化:起始挖矿高度,起始奖励金额uint64
	sHeight := m.miningStartHeight_()
	if sHeight == 0 {
		m.setMiningStartHeight_(currentHeight)
	}

	startRewardAmount := m.miningRewardAmount_()
	if startRewardAmount == 0 {
		m.setMiningRewardAmount_(uint64(150000000))
	}

	//计算奖励金额
	rewardAmount = m.calcRewardAmount(currentHeight, sHeight)

	//奖励转账
	token, e := m.State.GetGenesisToken()
	if e != nil {
		err.ErrorCode = bcerrors.ErrCodeLowLevelError
		err.ErrorDesc = e.Error()
		return
	}

	bal, err := m.GetBalance(token.Name, m.ContractAcct.Addr)
	if err.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	if bal.Uint64() >= rewardAmount {
		err = m.EventHandler.TransferByAddr(token.Address, m.ContractAcct.Addr, proposerRewardAddress, N(int64(rewardAmount)))
		if err.ErrorCode != bcerrors.ErrCodeOK {
			return
		}
		m.ReceiptOfMine(m.Block.ProposerAddress, currentHeight, proposerRewardAddress, rewardAmount)
	}

	return
}

func (m *Mining) calcRewardAmount(cHeight, sHeight int64) (rewardAmount uint64) {
	blockNum := cHeight - sHeight
	rewardAmount = m.miningRewardAmount_()

	if blockNum > 0 && blockNum%66000000 == 0 {
		rewardAmount = rewardAmount / 2
		if rewardAmount == 0 {
			rewardAmount = 1
		}
		m.setMiningRewardAmount_(rewardAmount)
	}
	return
}

func (m *Mining) ReceiptOfMine(proposer smc.Address, height int64, rewardAddr smc.Address, rewardValue uint64) {
	type mine struct {
		Proposer    smc.Address `json:"proposer"`    // 提案人地址
		Height      int64       `json:"height"`      // 挖矿区块高度
		RewardAddr  smc.Address `json:"rewardAddr"`  // 接收奖励地址
		RewardValue uint64      `json:"rewardValue"` // 奖励金额(单位：cong)
	}

	receipt := mine{
		Proposer:    proposer,
		Height:      height,
		RewardAddr:  rewardAddr,
		RewardValue: rewardValue}

	resBytes, err := json.Marshal(receipt)
	if err != nil {
		panic(err)
	}

	m.EventHandler.EmitReceipt("Mine", resBytes)
}
