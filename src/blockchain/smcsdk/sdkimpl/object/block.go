package object

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
)

// Block block detail information
type Block struct {
	smc sdk.ISmartContract //智能合约API对象指针

	bk std.Block
}

var _ sdk.IBlock = (*Block)(nil)
var _ sdkimpl.IAcquireSMC = (*Block)(nil)

// SMC get smart contract object
func (b *Block) SMC() sdk.ISmartContract { return b.smc }

// SetSMC set smart contract object
func (b *Block) SetSMC(smc sdk.ISmartContract) { b.smc = smc }

// ChainID get block's chainID
func (b *Block) ChainID() string { return b.bk.ChainID }

// BlockHash get block's blockHash
func (b *Block) BlockHash() types.Hash { return b.bk.BlockHash }

// Height get block's height
func (b *Block) Height() int64 { return b.bk.Height }

// Time get block's time
func (b *Block) Time() int64 { return b.bk.Time }

// Now get block's now
func (b *Block) Now() bn.Number { return bn.N(b.bk.Time) }

// NumTxs get block's numTxs
func (b *Block) NumTxs() int32 { return b.bk.NumTxs }

// DataHash get block's dataHash
func (b *Block) DataHash() types.Hash { return b.bk.DataHash }

// ProposerAddress get block's proposerAddress
func (b *Block) ProposerAddress() types.Address { return b.bk.ProposerAddress }

// RewardAddress get block's rewardAddress
func (b *Block) RewardAddress() types.Address { return b.bk.RewardAddress }

// RandomNumber get block's randomNumber
func (b *Block) RandomNumber() types.HexBytes { return b.bk.RandomNumber }

// Version gets block's version
func (b *Block) Version() string { return b.bk.Version }

// LastBlockHash get block's lastBlockHash
func (b *Block) LastBlockHash() types.Hash { return b.bk.LastBlockHash }

// LastCommitHash get block's lastCommitHash
func (b *Block) LastCommitHash() types.Hash { return b.bk.LastCommitHash }

// LastAppHash get block's lastAppHash
func (b *Block) LastAppHash() types.Hash { return b.bk.LastAppHash }

// LastFee get block's lastFee
func (b *Block) LastFee() int64 { return b.bk.LastFee }
