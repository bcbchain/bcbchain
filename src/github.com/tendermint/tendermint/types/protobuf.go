package types

import (
	"github.com/tendermint/abci/types"
)

// TM2PB is used for converting Tendermint types to protobuf types.
// UNSTABLE
var TM2PB = tm2pb{}

type tm2pb struct{}

func (tm2pb) Header(header *Header) types.Header {
	h := types.Header{
		ChainID:         header.ChainID,
		Height:          header.Height,
		Time:            header.Time.Unix(),
		NumTxs:          int32(header.NumTxs), // XXX: overflow
		LastBlockID:     TM2PB.BlockID(header.LastBlockID),
		LastCommitHash:  header.LastCommitHash,
		DataHash:        header.DataHash,
		LastAppHash:     header.LastAppHash,
		LastFee:         header.LastFee,
		LastAllocation:  header.LastAllocation,
		ProposerAddress: header.ProposerAddress,
		RewardAddress:   header.RewardAddress,
		LastMining:      header.LastMining,
	}
	if len(header.RandomOfBlock) != 0 {
		h.RandomeOfBlock = header.RandomOfBlock
	}

	if header.ChainVersion != nil && *header.ChainVersion != 0 {
		h.ChainVersion = *header.ChainVersion
		h.Version = *header.Version
	}

	if header.LastQueueChains != nil {
		h.LastQueueChains = &types.QueueChain{QueueBlocks: make([]types.QueueBlock, len(header.LastQueueChains.QueueBlocks))}
		for i, queueBlock := range header.LastQueueChains.QueueBlocks {
			h.LastQueueChains.QueueBlocks[i] = types.QueueBlock{
				QueueID:         queueBlock.QueueID,
				QueueHash:       queueBlock.QueueHash,
				LastQueueHash:   queueBlock.LastQueueHash,
				LastQueueHeight: queueBlock.LastQueueHeight}
		}
	}

	if header.Relayer != nil {
		h.Relayer = &types.Relayer{
			Address:   header.Relayer.Address,
			StartTime: header.Relayer.StartTime.Unix(),
		}
	}

	return h
}

func (tm2pb) BlockID(blockID BlockID) types.BlockID {
	return types.BlockID{
		Hash:  blockID.Hash,
		Parts: TM2PB.PartSetHeader(blockID.PartsHeader),
	}
}

func (tm2pb) PartSetHeader(partSetHeader PartSetHeader) types.PartSetHeader {
	return types.PartSetHeader{
		Total: int32(partSetHeader.Total), // XXX: overflow
		Hash:  partSetHeader.Hash,
	}
}

func (tm2pb) Validator(val *Validator) types.Validator {
	if val.VotingPower < 0 {
		val.VotingPower = 0
	}
	return types.Validator{
		PubKey:     val.PubKey.Bytes(),
		Power:      uint64(val.VotingPower),
		RewardAddr: val.RewardAddr,
		Name:       val.Name,
	}
}

func (tm2pb) Validators(vals *ValidatorSet) []types.Validator {
	validators := make([]types.Validator, len(vals.Validators))
	for i, val := range vals.Validators {
		validators[i] = TM2PB.Validator(val)
	}
	return validators
}

func (tm2pb) ConsensusParams(params *ConsensusParams) *types.ConsensusParams {
	return &types.ConsensusParams{
		BlockSize: &types.BlockSize{

			MaxBytes: int32(params.BlockSize.MaxBytes),
			MaxTxs:   int32(params.BlockSize.MaxTxs),
			MaxGas:   params.BlockSize.MaxGas,
		},
		TxSize: &types.TxSize{
			MaxBytes: int32(params.TxSize.MaxBytes),
			MaxGas:   params.TxSize.MaxGas,
		},
		BlockGossip: &types.BlockGossip{
			BlockPartSizeBytes: int32(params.BlockGossip.BlockPartSizeBytes),
		},
	}
}
