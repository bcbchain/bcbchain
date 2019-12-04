package std

import (
	"blockchain/smcsdk/sdk/types"
)

// Block block detail information
type Block struct {
	ChainID         string         `json:"chainID"`         //链ID
	BlockHash       types.Hash     `json:"blockHash"`       //区块哈希
	Height          int64          `json:"height"`          //区块高度
	Time            int64          `json:"time"`            //区块时间（单位为秒，始于1970-01-01 00:00:00）
	NumTxs          int32          `json:"numTxs"`          //区块中包含的交易笔数
	DataHash        types.Hash     `json:"dataHash"`        //区块中Data字段的哈希
	ProposerAddress types.Address  `json:"proposerAddress"` //区块提案者地址
	RewardAddress   types.Address  `json:"rewardAddress"`   //接收区块奖励的地址
	RandomNumber    types.HexBytes `json:"randomNumber"`    //区块随机数（取Linux系统的真随机数）
	Version         string         `json:"version"`         //当前区块提案人的软件版本
	LastBlockHash   types.Hash     `json:"lastBlockHash"`   //上一区块的区块哈希
	LastCommitHash  types.Hash     `json:"lastCommitHash"`  //上一区块的确认信息哈希
	LastAppHash     types.Hash     `json:"lastAppHash"`     //上一区块的应用层哈希
	LastFee         int64          `json:"lastFee"`         //上一区块的手续费总和（单位为Cong）
}

// KeyOfAppState the access key for appState in state database
// data for this key refer AppState
func KeyOfAppState() string { return "/world/appstate" }

// KeyOfGenesisChainVersion the access key for genesis chain version in state database
// data for this key refer string
func KeyOfGenesisChainVersion() string { return "/genesis/chainversion" }

// KeyOfChainID the access key for chain_id in state database
// data for this key refer string
func KeyOfChainID() string { return "/genesis/chainid" }

// KeyOfOrgID the access key for org_id in state database
// data for this key refer string
func KeyOfOrgID() string { return "/genesis/orgid" }

// KeyOfGasPriceRatio the access key for gasPriceRatio in state database
// data for this key refer uint64
func KeyOfGasPriceRatio() string { return "/genesis/gaspriceratio" }
