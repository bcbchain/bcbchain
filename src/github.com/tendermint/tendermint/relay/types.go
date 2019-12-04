package relay

import (
	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/tendermint/types"
	cmn "github.com/tendermint/tmlibs/common"
	"strconv"
)

// KVPair define key value pair
type KVPair struct {
	Key   []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value []byte `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

type Packet struct {
	FromChainID  string       `json:"fromChainID"`  // 发起链的链ID（A）
	ToChainID    string       `json:"toChainID"`    // 目标链的链ID（B）
	QueueID      string       `json:"queueID"`      // 跨链通讯队列（A->B）
	Seq          uint64       `json:"seq"`          // (A->B)这个队列上跨链通讯包序号,从0开始累加+1
	OrgID        string       `json:"orgID"`        // 组织ID
	ContractName string       `json:"contractName"` // 合约名称
	IbcHash      cmn.HexBytes `json:"ibcHash"`      // 跨链事务哈希,通过此哈希从区块链上确认最终执行结果
	Type         string       `json:"type"`         // 跨链通讯类型："tcctx", "notify"
	State        State        `json:"state"`        // 状态
	Receipts     []KVPair     `json:"receipts"`     // 当前状态下需要传递到另一侧的应用层数据
}

type State struct {
	Status string `json:"status"` // 状态："NoAckWanted", "NoAck"
	Tag    string `json:"tag"`    // 表示业务层的状态标识：
	Log    string `json:"log"`    // 异常日志
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
	ValidatorAddress string           `json:"validator_address"` // 验证者节点地址
	ValidatorIndex   int              `json:"validator_index"`   // 验证者节点索引号
	Signature        SignatureEd25519 `json:"signature"`         // 签名数据
}

type BlockID struct {
	Hash        cmn.HexBytes  `json:"hash"`
	PartsHeader PartSetHeader `json:"parts"`
}

type PartSetHeader struct {
	Total int          `json:"total"`
	Hash  cmn.HexBytes `json:"hash"`
}

// Receipt receipt information
type Receipt struct {
	Name         string       `json:"name"`            // 收据名称：标准名称（trnsfer，...) 非标准名称（...）
	ContractAddr string       `json:"contractAddress"` // 合约地址
	Bytes        []byte       `json:"receiptBytes"`
	Hash         cmn.HexBytes `json:"receiptHash"`
}

// Info abci msg
type ResultABCIInfo struct {
	Response abci.ResponseInfo `json:"response"`
}

type ABCIResponses struct {
	DeliverTx []*abci.ResponseDeliverTx
	EndBlock  *abci.ResponseEndBlock
}

type ResultBlock struct {
	BlockMeta *types.BlockMeta `json:"block_meta"`
	Block     *types.Block     `json:"block"`
	BlockSize int              `json:"block_size"`
}

type ResultABCIQuery struct {
	Response abci.ResponseQuery `json:"response"`
}

type ResultBlockResults struct {
	Height  int64          `json:"height"`
	Results *ABCIResponses `json:"results"`
}

type ResultBroadcastTxCommit struct {
	CheckTx   abci.ResponseCheckTx   `json:"check_tx,omitempt"`
	DeliverTx abci.ResponseDeliverTx `json:"deliver_tx,omitempt"`
	Hash      cmn.HexBytes           `json:"hash,omitempt"`
	Height    int64                  `json:"height,omitempt"`
}

type MessageIndex struct {
	Height  int64        `json:"height"`
	IbcHash cmn.HexBytes `json:"ibcHash"`
}

type ContractVersionList struct {
	Name             string   `json:"name"`             // 合约名称
	ContractAddrList []string `json:"contractAddrList"` // 合约地址列表
	EffectHeights    []int64  `json:"effectHeights"`    // 合约生效高度列表
}

// Method method information
type Method struct {
	MethodID  string `json:"methodId"`  //方法ID
	Gas       int64  `json:"gas"`       //方法需要消耗的燃料
	ProtoType string `json:"prototype"` //方法原型
}

// Contract contract detail information
type Contract struct {
	Address      string       `json:"address"`        //合约地址
	Account      string       `json:"account"`        //合约的账户地址
	Owner        string       `json:"owner"`          //合约拥有者的账户地址
	Name         string       `json:"name"`           //合约名称
	Version      string       `json:"version"`        //合约版本
	CodeHash     cmn.HexBytes `json:"codeHash"`       //合约代码的哈希
	EffectHeight int64        `json:"effectHeight"`   //合约生效的区块高度
	LoseHeight   int64        `json:"loseHeight"`     //合约失效的区块高度
	KeyPrefix    string       `json:"keyPrefix"`      //合约在状态数据库中KEY值的前缀
	Methods      []Method     `json:"methods"`        //合约对外提供接口的方法列表
	Interfaces   []Method     `json:"interfaces"`     //合约提供的跨合约调用的方法列表
	Mine         []Method     `json:"mine"`           //合约提供的挖矿方法
	IBCs         []Method     `json:"ibcs,omitempty"` //合约提供的执行跨链业务的方法列表
	Token        string       `json:"token"`          //合约代币地址
	OrgID        string       `json:"orgID"`          //组织ID
	ChainVersion int64        `json:"chainVersion"`   //链版本
}

type Header struct {
	ChainID         string       `json:"chain_id"`
	Height          int64        `json:"height"`
	Time            string       `json:"time"`
	NumTxs          int64        `json:"num_txs"`
	LastBlockID     BlockID      `json:"last_block_id"`
	TotalTxs        int64        `json:"total_txs"`
	LastCommitHash  cmn.HexBytes `json:"last_commit_hash"`
	DataHash        cmn.HexBytes `json:"data_hash"`
	ValidatorsHash  cmn.HexBytes `json:"validators_hash"`
	ConsensusHash   cmn.HexBytes `json:"consensus_hash"`
	LastAppHash     cmn.HexBytes `json:"last_app_hash"`
	LastResultsHash cmn.HexBytes `json:"last_results_hash"`
	EvidenceHash    cmn.HexBytes `json:"evidence_hash"`
	LastFee         uint64       `json:"last_fee,omitempty"`
	LastAllocation  Allocation   `json:"last_allocation,omitempty"`
	ProposerAddress string       `json:"proposer_address,omitempty"`
	RewardAddress   string       `json:"reward_address,omitempty"`
	RandomOfBlock   cmn.HexBytes `json:"random_of_block,omitempty"`
	LastMining      *int64       `json:"last_mining,omitempty"`
	Version         *string      `json:"version,omitempty"`
	ChainVersion    *int64       `json:"chain_version,omitempty"`
	LastQueueChains *QueueChain  `json:"last_queue_chains,omitempty"`
	Relayer         *Relayer     `json:"relayer,omitempty"`
}

type Relayer struct {
	Address   string `json:"address"`
	StartTime string `json:"start_time"`
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
	QueueID         string       `json:"queue_id"`
	QueueHash       cmn.HexBytes `json:"queue_hash"`
	LastQueueHash   cmn.HexBytes `json:"last_queue_hash"`
	LastQueueHeight int64        `json:"last_queue_height"`
}

type ChainInfo struct {
	SideChainName string   `json:"sideChainName"` //侧链名称
	ChainID       string   `json:"chainID"`       //侧链ID
	NodeNames     []string `json:"NodeNames"`     //节点名称列表
	OrgName       string   `json:"orgName"`       //侧链所属组织名称
	Owner         string   `json:"owner"`         //侧链的所有者地址
	Height        int64    `json:"height"`        //侧链创世时在主链上的高度
	Status        string   `json:"status"`        //侧链状态
	GasPriceRatio string   `json:"gasPriceRatio"` //燃料价格调整比例
}

func keyOfChainInfo(chainID string) string {
	return "/sidechain/" + chainID + "/chaininfo"
}

func keyOfSequence(queueID string) string {
	return "/ibc/seq/" + queueID
}

func keyOfSideChainIDs() string {
	return "/sidechain/chainid/all"
}

func keyOfMessageIndex(queueID string, seq uint64) string {
	return "/ibc/seq/" + queueID + "/" + strconv.Itoa(int(seq))
}

func keyOfOpenURLs(chainId string) string {
	return "/sidechain/" + chainId + "/openurls"
}

func keyOfAccountNonce(address string) string {
	return "/account/ex/" + address + "/account"
}

func keyOfChainID() string {
	return "/genesis/chainid"
}

func keyOfGenesisOrgID() string {
	return "/genesis/orgid"
}

func keyOfContract(address string) string {
	return "/contract/" + address
}

func keyOfVersionList(contractName, orgID string) string {
	return "/contract/" + orgID + "/" + contractName
}
