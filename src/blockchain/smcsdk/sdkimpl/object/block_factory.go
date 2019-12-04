package object

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
)

// NewBlock factory method for create block object
func NewBlock(smc sdk.ISmartContract,
	chainID, version string,
	blockHash, dataHash types.Hash,
	height, time int64,
	numTxs int32,
	proposerAddress, rewardAddress types.Address,
	randomNumber types.HexBytes,
	lastBlockHash, lastCommitHash, lastAppHash types.Hash,
	lastFee int64) sdk.IBlock {
	block := &Block{
		bk: std.Block{
			ChainID:         chainID,
			BlockHash:       blockHash,
			Height:          height,
			Time:            time,
			NumTxs:          numTxs,
			DataHash:        dataHash,
			ProposerAddress: proposerAddress,
			RewardAddress:   rewardAddress,
			RandomNumber:    randomNumber,
			Version:         version,
			LastBlockHash:   lastBlockHash,
			LastCommitHash:  lastCommitHash,
			LastAppHash:     lastAppHash,
			LastFee:         lastFee,
		},
	}
	block.SetSMC(smc)

	return block
}

// NewBlockFromSTD factory method for create block with standard block data
func NewBlockFromSTD(smc sdk.ISmartContract, stdBlock *std.Block) sdk.IBlock {
	block := &Block{bk: *stdBlock}
	block.SetSMC(smc)

	return block
}
