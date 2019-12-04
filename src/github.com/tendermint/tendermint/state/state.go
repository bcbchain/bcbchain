package state

import (
	"blockchain/smcsdk/sdk/ibc"
	"bytes"
	"fmt"
	"github.com/tendermint/go-crypto"
	cfg "github.com/tendermint/tendermint/config"
	"strconv"
	"time"

	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/tendermint/types"
)

// database keys
var (
	stateKey = []byte("stateKey")
)

//-----------------------------------------------------------------------------

// State is a short description of the latest committed block of the Tendermint consensus.
// It keeps all information necessary to validate new blocks,
// including the last validator set and the consensus params.
// All fields are exposed so the struct can be easily serialized,
// but none of them should be mutated directly.
// Instead, use state.Copy() or state.NextState(...).
// NOTE: not goroutine-safe.
type State struct {
	// Immutable
	ChainID string

	// LastBlockHeight=0 at genesis (ie. block(H=0) does not exist)
	LastBlockHeight  int64
	LastBlockTotalTx int64
	LastBlockID      types.BlockID
	LastBlockTime    time.Time
	LastFee          uint64
	LastAllocation   []abci.Allocation
	// LastValidators is used to validate block.LastCommit.
	// Validators are persisted to the database separately every time they change,
	// so we can query for historical validator sets.
	// Note that if s.LastBlockHeight causes a valset change,
	// we set s.LastHeightValidatorsChanged = s.LastBlockHeight + 1
	Validators                  *types.ValidatorSet
	LastValidators              *types.ValidatorSet
	LastHeightValidatorsChanged int64

	// Consensus parameters used for validating blocks.
	// Changes returned by EndBlock and updated after Commit.
	ConsensusParams                  types.ConsensusParams
	LastHeightConsensusParamsChanged int64

	// Merkle root of the results from executing prev block
	LastResultsHash []byte

	// The latest AppHash we've received from calling abci.Commit()
	LastAppHash []byte

	LastTxsHashList [][]byte

	// added 24 May 2019
	LastMining *int64

	// add 26 Mar. 2019
	ChainVersion int64

	// added 17 Sep. 2019
	LastQueueChains *ibc.QueueChain
}

// Copy makes a copy of the State for mutating.
func (s State) Copy() State {
	return State{
		ChainID: s.ChainID,

		LastBlockHeight:  s.LastBlockHeight,
		LastBlockTotalTx: s.LastBlockTotalTx,
		LastBlockID:      s.LastBlockID,
		LastBlockTime:    s.LastBlockTime,
		LastFee:          s.LastFee,
		LastAllocation:   s.LastAllocation,

		Validators:                  s.Validators.Copy(),
		LastValidators:              s.LastValidators.Copy(),
		LastHeightValidatorsChanged: s.LastHeightValidatorsChanged,

		ConsensusParams:                  s.ConsensusParams,
		LastHeightConsensusParamsChanged: s.LastHeightConsensusParamsChanged,

		LastAppHash:     s.LastAppHash,
		LastTxsHashList: s.LastTxsHashList,

		LastResultsHash: s.LastResultsHash,
		ChainVersion:    s.ChainVersion,

		LastMining:      s.LastMining,
		LastQueueChains: s.LastQueueChains,
	}
}

// Equals returns true if the States are identical.
func (s State) Equals(s2 State) bool {
	sbz, s2bz := s.Bytes(), s2.Bytes()
	return bytes.Equal(sbz, s2bz)
}

// Bytes serializes the State using go-amino.
func (s State) Bytes() []byte {
	return cdc.MustMarshalBinaryBare(s)
}

// IsEmpty returns true if the State is equal to the empty State.
func (s State) IsEmpty() bool {
	return s.Validators == nil // XXX can't compare to Empty
}

// GetValidators returns the last and current validator sets.
func (s State) GetValidators() (last *types.ValidatorSet, current *types.ValidatorSet) {
	return s.LastValidators, s.Validators
}

//------------------------------------------------------------------------
// Create a block from the latest state

//todo 截取apphash
// MakeBlock builds a block with the given txs and commit from the current state.
func (s State) MakeBlock(height int64, txs []types.Tx, commit *types.Commit, proposer crypto.Address, rewardAddr string, allocation []abci.Allocation,
	relayer *types.Relayer) (*types.Block, *types.PartSet) {
	// build base block
	//block := types.MakeBlock(height, txs, commit)

	block := types.GIMakeBlock(height, txs, commit, s.LastTxsHashList, proposer, s.LastFee, rewardAddr, allocation, s.ChainVersion, s.LastMining)

	// fill header with state data
	block.ChainID = s.ChainID
	block.TotalTxs = s.LastBlockTotalTx + block.NumTxs
	block.LastBlockID = s.LastBlockID
	block.ValidatorsHash = s.Validators.Hash()
	block.LastAppHash = s.LastAppHash

	block.ConsensusHash = s.ConsensusParams.Hash()
	block.LastResultsHash = s.LastResultsHash
	block.LastMining = s.LastMining
	block.LastQueueChains = s.LastQueueChains

	// fill header relay
	block.Relayer = relayer

	return block, block.MakePartSet(s.ConsensusParams.BlockGossip.BlockPartSizeBytes)
}

//------------------------------------------------------------------------
// Genesis

// MakeGenesisStateFromFile reads and unmarshals state from the given
// file.
//
// Used during replay and in tests.
func MakeGenesisStateFromFile(config *cfg.Config) (State, error) {
	genDoc, err := MakeGenesisDocFromFile(config)
	if err != nil {
		return State{}, err
	}
	return MakeGenesisState(genDoc)
}

// MakeGenesisDocFromFile reads and unmarshals genesis doc from the given file.
func MakeGenesisDocFromFile(config *cfg.Config) (*types.GenesisDoc, error) {
	return types.GenesisDocFromFile(config)
}

// MakeGenesisState creates state from types.GenesisDoc.
func MakeGenesisState(genDoc *types.GenesisDoc) (State, error) {
	err := genDoc.ValidateAndComplete()
	if err != nil {
		return State{}, fmt.Errorf("Error in genesis file: %v", err)
	}

	// Make validators slice
	validators := make([]*types.Validator, len(genDoc.Validators))
	for i, val := range genDoc.Validators {
		pubKey := val.PubKey
		address := pubKey.Address(crypto.GetChainId())

		nodeName := val.Name

		power := val.Power
		if power < 0 {
			power = 0
		}
		// Make validator
		validators[i] = &types.Validator{
			Address:     address,
			PubKey:      pubKey,
			VotingPower: uint64(power),
			RewardAddr:  val.RewardAddr,
			Name:        nodeName,
		}
	}

	chainVersion := int64(0)
	if len(genDoc.ChainVersion) != 0 {
		chainVersion, err = strconv.ParseInt(genDoc.ChainVersion, 10, 64)
		if err != nil {
			return State{}, fmt.Errorf("Invalid chain_version=%s ", genDoc.ChainVersion)
		}
	}

	return State{

		ChainID:      genDoc.ChainID,
		ChainVersion: chainVersion,

		LastBlockHeight: 0,
		LastBlockID:     types.BlockID{},
		LastBlockTime:   genDoc.GenesisTime,
		LastFee:         0,
		LastAllocation:  []abci.Allocation{},

		Validators:                  types.NewValidatorSet(validators),
		LastValidators:              types.NewValidatorSet(nil),
		LastHeightValidatorsChanged: 1,

		ConsensusParams:                  *genDoc.ConsensusParams,
		LastHeightConsensusParamsChanged: 1,

		LastAppHash: genDoc.AppHash,
	}, nil
}
