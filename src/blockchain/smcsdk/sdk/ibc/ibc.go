package ibc

import (
	"blockchain/smcsdk/sdk/types"
)

const (
	// 协议状态
	NoAckWanted = "NoAckWanted"
	NoAck       = "NoAck"

	// 交易状态
	RecastPending  = "RecastPending"
	ConfirmPending = "ConfirmPending"
	CancelPending  = "CancelPending"
	Confirmed      = "Confirmed"
	Canceled       = "Canceled"

	// 通知状态
	NotifyPending = "NotifyPending"
	NotifySuccess = "Success"
	NotifyFailure = "Failure"

	// 类型
	TccTxType  = "tcctx"
	NotifyType = "notify"
)

type Packet struct {
	FromChainID  string         `json:"fromChainID"`  // 发起链的链ID（A）
	ToChainID    string         `json:"toChainID"`    // 目标链的链ID（B）
	QueueID      string         `json:"queueID"`      // 跨链通讯队列（A->B）
	Seq          uint64         `json:"seq"`          // (A->B)这个队列上跨链通讯包序号,从0开始累加+1
	OrgID        string         `json:"orgID"`        // 组织ID
	ContractName string         `json:"contractName"` // 合约名称
	IbcHash      types.Hash     `json:"ibcHash"`      // 跨链事务哈希,通过此哈希从区块链上确认最终执行结果
	Type         string         `json:"type"`         // 跨链通讯类型："tcctx", "notify"
	State        State          `json:"state"`        // 状态
	Receipts     []types.KVPair `json:"receipts"`     // 当前状态下需要传递到另一侧的应用层数据
}

// Tag value：
// "RecastPending", "TryHubPending",
// "CancelPending", "ConfirmPending",
// "NotifyPending",
// "Canceled", "Confirmed"
// State ibc packet's state
type State struct {
	Status string `json:"status"` // 状态："NoAckWanted", "NoAck"
	Tag    string `json:"tag"`    // 表示业务层的状态标识：
	Log    string `json:"log"`    // 异常日志
}

// Final ibc transaction's final state
type Final struct {
	IBCHash types.Hash `json:"ibcHash"` // 跨链事务hash
	State   State      `json:"state"`   // 跨链事务状态
}

// PktsProof ibc proof and packet
type PktsProof struct {
	Packets    []Packet    `json:"packets"`    // 跨链数据包列表
	Header     Header      `json:"header"`     // 跨链数据包所在区块的区块头
	Precommits []Precommit `json:"precommits"` // 每个验证者节点针对这个区块的投票及签名列表
}

type SignatureEd25519 [64]byte

// Precommit
type Precommit struct {
	Round            int              `json:"round"` // 投票轮次
	Timestamp        string           `json:"timestamp"`
	VoteType         byte             `json:"type"` // 投票类型
	BlockID          BlockID          `json:"block_id"`
	ValidatorAddress types.Address    `json:"validator_address"` // 验证者节点地址
	ValidatorIndex   int              `json:"validator_index"`   // 验证者节点索引号
	Signature        SignatureEd25519 `json:"signature"`         // 签名数据
}

type Header struct {
	ChainID         string          `json:"chain_id"`
	Height          int64           `json:"height"`
	Time            string          `json:"time"`
	NumTxs          int64           `json:"num_txs"`
	LastBlockID     BlockID         `json:"last_block_id"`
	TotalTxs        int64           `json:"total_txs"`
	LastCommitHash  types.Hash      `json:"last_commit_hash"`
	DataHash        types.Hash      `json:"data_hash"`
	ValidatorsHash  types.Hash      `json:"validators_hash"`
	ConsensusHash   types.Hash      `json:"consensus_hash"`
	LastAppHash     types.Hash      `json:"last_app_hash"`
	LastResultsHash types.Hash      `json:"last_results_hash"`
	EvidenceHash    types.Hash      `json:"evidence_hash"`
	LastFee         uint64          `json:"last_fee,omitempty"`
	LastAllocation  Allocation      `json:"last_allocation,omitempty"`
	ProposerAddress string          `json:"proposer_address,omitempty"`
	RewardAddress   string          `json:"reward_address,omitempty"`
	RandomOfBlock   *types.HexBytes `json:"random_of_block,omitempty"`
	LastMining      *int64          `json:"last_mining,omitempty"`
	Version         *string         `json:"version,omitempty"`
	ChainVersion    *int64          `json:"chain_version,omitempty"`
	LastQueueChains *QueueChain     `json:"last_queue_chains,omitempty"`
	Relayer         *Relayer        `json:"relayer,omitempty"`
}

type Relayer struct {
	Address   types.Address `json:"address"`
	StartTime string        `json:"start_time"`
}

type BlockID struct {
	Hash        types.Hash    `json:"hash,omitempty"`
	PartsHeader PartSetHeader `json:"parts,omitempty"`
}

type PartSetHeader struct {
	Total int        `json:"total,omitempty"`
	Hash  types.Hash `json:"hash,omitempty"`
}

type AllocItem struct {
	Addr string `json:"addr"`
	Fee  uint64 `json:"fee"`
}

type Allocation []AllocItem

type QueueChain struct {
	QueueBlocks []QueueBlock `json:"ibc_queue_blocks"`
}

type QueueBlock struct {
	QueueID         string     `json:"queue_id"`
	QueueHash       types.Hash `json:"queue_hash"`
	LastQueueHash   types.Hash `json:"last_queue_hash"`
	LastQueueHeight int64      `json:"last_queue_height"`
}

func KeyOfIBCPacket(ibcHash types.Hash) string {
	return "/ibc" + ibcHash.String()
}
