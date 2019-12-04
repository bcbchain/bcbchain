package types

import (
	"blockchain/smcsdk/sdk/ibc"
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/tendermint/abci/types"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/tendermint/softforks"
	"github.com/tendermint/tendermint/version"
	"strings"
	"sync"
	"time"

	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/merkle"
	"golang.org/x/crypto/ripemd160"
)

// Block defines the atomic unit of a Tendermint blockchain.
// TODO: add Version byte
type Block struct {
	mtx        sync.Mutex
	*Header    `json:"header"`
	*Data      `json:"data"`
	Evidence   EvidenceData `json:"evidence"`
	LastCommit *Commit      `json:"last_commit"`
}

// MakeBlock returns a new block with an empty header, except what can be computed from itself.
// It populates the same set of fields validated by ValidateBasic
func MakeBlock(height int64, txs []Tx, commit *Commit) *Block {
	block := &Block{
		Header: &Header{
			Height: height,
			Time:   time.Now(),
			NumTxs: int64(len(txs)),
		},
		LastCommit: commit,
		Data: &Data{
			Txs: txs,
		},
	}
	block.fillHeader()
	return block
}

//将上一次交易hashlist存入到下一个block的data中
func GIMakeBlock(height int64, txs []Tx, commit *Commit, txHashList [][]byte,
	proposer string, lastFee uint64, rewardAddr string, lastAllocation []types.Allocation,
	chainVersion int64, lastMining *int64) *Block {

	block := &Block{
		Header: &Header{
			Height:          height,
			Time:            time.Now(),
			NumTxs:          int64(len(txs)),
			LastFee:         lastFee,
			LastAllocation:  lastAllocation,
			LastMining:      lastMining,
			ProposerAddress: proposer,
			RewardAddress:   rewardAddr,
		},
		LastCommit: commit,
		Data: &Data{
			Txs:             txs,
			LastTxsHashList: txHashList,
		},
	}

	if softforks.V1_0_2_3233(height) {
		// old code
	} else {
		// new code
		r := make([]byte, 32)
		_, e := rand.Read(r)
		if e != nil {
			panic(e)
		}
		block.Header.RandomOfBlock = r
	}
	if chainVersion != 0 {
		block.ChainVersion = &chainVersion
		v := version.Version
		block.Version = &v
	}
	block.fillHeader()
	return block
}

// AddEvidence appends the given evidence to the block
func (b *Block) AddEvidence(evidence []Evidence) {
	b.Evidence.Evidence = append(b.Evidence.Evidence, evidence...)
}

// ValidateBasic performs basic validation that doesn't involve state data.
// It checks the internal consistency of the block.
func (b *Block) ValidateBasic() error {
	if b == nil {
		return errors.New("Nil blocks are invalid")
	}
	b.mtx.Lock()
	defer b.mtx.Unlock()

	newTxs := int64(len(b.Data.Txs))
	if b.NumTxs != newTxs {
		return fmt.Errorf("Wrong Block.Header.NumTxs. Expected %v, got %v", newTxs, b.NumTxs)
	}
	if !bytes.Equal(b.LastCommitHash, b.LastCommit.Hash()) {
		return fmt.Errorf("Wrong Block.Header.LastCommitHash.  Expected %v, got %v", b.LastCommitHash, b.LastCommit.Hash())
	}
	if b.Header.Height != 1 {
		if err := b.LastCommit.ValidateBasic(); err != nil {
			return err
		}
	}
	if !bytes.Equal(b.DataHash, b.Data.Hash()) {
		return fmt.Errorf("Wrong Block.Header.DataHash.  Expected %v, got %v", b.DataHash, b.Data.Hash())
	}
	if !bytes.Equal(b.EvidenceHash, b.Evidence.Hash()) {
		return errors.New(cmn.Fmt("Wrong Block.Header.EvidenceHash.  Expected %v, got %v", b.EvidenceHash, b.Evidence.Hash()))
	}
	return nil
}

// fillHeader fills in any remaining header fields that are a function of the block data
func (b *Block) fillHeader() {
	if b.LastCommitHash == nil {
		b.LastCommitHash = b.LastCommit.Hash()
	}
	if b.DataHash == nil {
		b.DataHash = b.Data.Hash()
	}
	if b.EvidenceHash == nil {
		b.EvidenceHash = b.Evidence.Hash()
	}
}

// Hash computes and returns the block hash.
// If the block is incomplete, block hash is nil for safety.
func (b *Block) Hash() cmn.HexBytes {
	if b == nil {
		return nil
	}
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b == nil || b.Header == nil || b.Data == nil || b.LastCommit == nil {
		return nil
	}
	b.fillHeader()
	return b.Header.Hash()
}

// MakePartSet returns a PartSet containing parts of a serialized block.
// This is the form in which the block is gossipped to peers.
func (b *Block) MakePartSet(partSize int) *PartSet {
	if b == nil {
		return nil
	}
	b.mtx.Lock()
	defer b.mtx.Unlock()

	// We prefix the byte length, so that unmarshaling
	// can easily happen via a reader.
	bz, err := cdc.MarshalBinary(b)
	if err != nil {
		panic(err)
	}
	return NewPartSetFromData(bz, partSize)
}

// HashesTo is a convenience function that checks if a block hashes to the given argument.
// A nil block never hashes to anything, and nothing hashes to a nil hash.
func (b *Block) HashesTo(hash []byte) bool {
	if len(hash) == 0 {
		return false
	}
	if b == nil {
		return false
	}
	return bytes.Equal(b.Hash(), hash)
}

// String returns a string representation of the block
func (b *Block) String() string {
	return b.StringIndented("")
}

// StringIndented returns a string representation of the block
func (b *Block) StringIndented(indent string) string {
	if b == nil {
		return "nil-Block"
	}
	return fmt.Sprintf(`Block{
%s  %v
%s  %v
%s  %v
%s  %v
%s}#%v`,
		indent, b.Header.StringIndented(indent+"  "),
		indent, b.Data.StringIndented(indent+"  "),
		indent, b.Evidence.StringIndented(indent+"  "),
		indent, b.LastCommit.StringIndented(indent+"  "),
		indent, b.Hash())
}

// StringShort returns a shortened string representation of the block
func (b *Block) StringShort() string {
	if b == nil {
		return "nil-Block"
	}
	return fmt.Sprintf("Block#%v", b.Hash())
}

type Allocation []types.Allocation

//-----------------------------------------------------------------------------

// Header defines the structure of a Tendermint block header
// TODO: limit header size
// NOTE: changes to the Header should be duplicated in the abci Header
type Header struct {
	// basic block info
	ChainID string    `json:"chain_id"`
	Height  int64     `json:"height"`
	Time    time.Time `json:"time"`
	NumTxs  int64     `json:"num_txs"`

	// prev block info
	LastBlockID BlockID `json:"last_block_id"`
	TotalTxs    int64   `json:"total_txs"`

	// hashes of block data
	LastCommitHash cmn.HexBytes `json:"last_commit_hash"` // commit from validators from the last block
	DataHash       cmn.HexBytes `json:"data_hash"`        // transactions

	// hashes from the app output from the prev block
	ValidatorsHash  cmn.HexBytes `json:"validators_hash"`   // validators for the current block
	ConsensusHash   cmn.HexBytes `json:"consensus_hash"`    // consensus params for current block
	LastAppHash     cmn.HexBytes `json:"last_app_hash"`     // state after txs from the previous block
	LastResultsHash cmn.HexBytes `json:"last_results_hash"` // root hash of all results from the txs from the previous block

	// consensus info
	EvidenceHash cmn.HexBytes `json:"evidence_hash"` // evidence included in the block

	// Proposer Address
	LastFee         uint64     `json:"last_fee"`
	LastAllocation  Allocation `json:"last_allocation"`
	ProposerAddress string     `json:"proposer_address"`
	RewardAddress   string     `json:"reward_address"`

	// added 06 August 2018
	RandomOfBlock cmn.HexBytes `json:"random_of_block,omitempty"`
	LastMining    *int64       `json:"last_mining,omitempty"` // added 24 May 2019
	// version of block - added 14 Dec. 2018
	Version *string `json:"version,omitempty"`
	// added 26 Mar. 2019
	ChainVersion *int64 `json:"chain_version,omitempty"`
	// added 17 Sep. 2019
	LastQueueChains *ibc.QueueChain `json:"last_queue_chains,omitempty"`

	// added 12 Oct. 2019
	Relayer *Relayer `json:"relayer,omitempty"`
}

type Relayer struct {
	Address   crypto.Address `json:"address"`
	StartTime time.Time      `json:"start_time"`
}

// Hash returns the hash of the header.
// Returns nil if ValidatorHash is missing,
// since a Header is not valid unless there is
// a ValidatorsHash (corresponding to the validator set).
func (h *Header) Hash() cmn.HexBytes {
	if h == nil || len(h.ValidatorsHash) == 0 {
		return nil
	}
	mapForHash := map[string]merkle.Hasher{
		"ChainID":        aminoHasher(h.ChainID),
		"Height":         aminoHasher(h.Height),
		"Time":           aminoHasher(h.Time),
		"NumTxs":         aminoHasher(h.NumTxs),
		"TotalTxs":       aminoHasher(h.TotalTxs),
		"LastBlockID":    aminoHasher(h.LastBlockID),
		"LastCommit":     aminoHasher(h.LastCommitHash),
		"Data":           aminoHasher(h.DataHash),
		"Validators":     aminoHasher(h.ValidatorsHash),
		"LastApp":        aminoHasher(h.LastAppHash),
		"Consensus":      aminoHasher(h.ConsensusHash),
		"Results":        aminoHasher(h.LastResultsHash),
		"Evidence":       aminoHasher(h.EvidenceHash),
		"LastFee":        aminoHasher(h.LastFee),
		"LastAllocation": aminoHasher(h.LastAllocation),
		"Proposer":       aminoHasher(h.ProposerAddress),
		"RewardAddr":     aminoHasher(h.RewardAddress),
	}

	if len(h.RandomOfBlock) != 0 {
		mapForHash["RandomOfBlock"] = aminoHasher(h.RandomOfBlock)
	}

	if h.LastMining != nil {
		mapForHash["last_mining"] = aminoHasher(h.LastMining)
	}

	if h.ChainVersion != nil && *h.ChainVersion != 0 {
		mapForHash["Version"] = aminoHasher(h.Version)
		mapForHash["ChainVersion"] = aminoHasher(h.ChainVersion)
	}

	if h.LastQueueChains != nil {
		mapForHash["LastQueueChains"] = aminoHasher(h.LastQueueChains)
	}

	if h.Relayer != nil {
		mapForHash["Relayer"] = aminoHasher(h.Relayer)
	}

	return merkle.SimpleHashFromMap(mapForHash)
}

func (as *Allocation) StringIndented(indent string) string {
	if as == nil {
		return "[]"
	}
	res := "["
	for i, v := range *as {
		res += fmt.Sprintf(`{Addr:%s, Fee:%d}`, v.Addr, v.Fee)
		if i != len(*as)-1 {
			res += "," + indent
		}
	}
	return res + "]"
}

// StringIndented returns a string representation of the header
func (h *Header) StringIndented(indent string) string {
	if h == nil {
		return "nil-Header"
	}

	return fmt.Sprintf(`Header{
%s  ChainID:        %v
%s  Height:         %v
%s  Time:           %v
%s  NumTxs:         %v
%s  TotalTxs:       %v
%s  LastBlockID:    %v
%s  LastCommit:     %v
%s  Data:           %v
%s  Validators:     %v
%s  LastApp:        %v
%s  Consensus:      %v
%s  Results:        %v
%s  Evidence:       %v
%s  LastFee:        %v
%s  LastAllocation: %v
%s  Proposer:       %v
%s  RewardAddr:     %v
%s  RandomOfBlock:  %v
%s  LastMining:     %v
%s  Version:		%v
%s  ChainVersion:	%v
%s  QueueChain:	    %v
%s  Relayer:	    %v
%s}#%v`,
		indent, h.ChainID,
		indent, h.Height,
		indent, h.Time,
		indent, h.NumTxs,
		indent, h.TotalTxs,
		indent, h.LastBlockID,
		indent, h.LastCommitHash,
		indent, h.DataHash,
		indent, h.ValidatorsHash,
		indent, h.LastAppHash,
		indent, h.ConsensusHash,
		indent, h.LastResultsHash,
		indent, h.EvidenceHash,
		indent, h.LastFee,
		indent, h.LastAllocation.StringIndented(" "),
		indent, h.ProposerAddress,
		indent, h.RewardAddress,
		indent, h.RandomOfBlock,
		indent, h.LastMining,
		indent, h.Version,
		indent, h.ChainVersion,
		indent, h.LastQueueChains,
		indent, h.Relayer,
		indent, h.Hash())
}

//-------------------------------------

// Commit contains the evidence that a block was committed by a set of validators.
// NOTE: Commit is empty for height 1, but never nil.
type Commit struct {
	// NOTE: The Precommits are in order of address to preserve the bonded ValidatorSet order.
	// Any peer with a block can gossip precommits by index with a peer without recalculating the
	// active ValidatorSet.
	BlockID    BlockID `json:"block_id"`
	Precommits []*Vote `json:"precommits"`

	// Volatile
	firstPrecommit *Vote
	hash           cmn.HexBytes
	bitArray       *cmn.BitArray
}

// FirstPrecommit returns the first non-nil precommit in the commit.
// If all precommits are nil, it returns an empty precommit with height 0.
func (commit *Commit) FirstPrecommit() *Vote {
	if len(commit.Precommits) == 0 {
		return nil
	}
	if commit.firstPrecommit != nil {
		return commit.firstPrecommit
	}
	for _, precommit := range commit.Precommits {
		if precommit != nil {
			commit.firstPrecommit = precommit
			return precommit
		}
	}
	return &Vote{
		Type: VoteTypePrecommit,
	}
}

// Height returns the height of the commit
func (commit *Commit) Height() int64 {
	if len(commit.Precommits) == 0 {
		return 0
	}
	return commit.FirstPrecommit().Height
}

// Round returns the round of the commit
func (commit *Commit) Round() int {
	if len(commit.Precommits) == 0 {
		return 0
	}
	return commit.FirstPrecommit().Round
}

// Type returns the vote type of the commit, which is always VoteTypePrecommit
func (commit *Commit) Type() byte {
	return VoteTypePrecommit
}

// Size returns the number of votes in the commit
func (commit *Commit) Size() int {
	if commit == nil {
		return 0
	}
	return len(commit.Precommits)
}

// BitArray returns a BitArray of which validators voted in this commit
func (commit *Commit) BitArray() *cmn.BitArray {
	if commit.bitArray == nil {
		commit.bitArray = cmn.NewBitArray(len(commit.Precommits))
		for i, precommit := range commit.Precommits {
			// TODO: need to check the BlockID otherwise we could be counting conflicts,
			// not just the one with +2/3 !
			commit.bitArray.SetIndex(i, precommit != nil)
		}
	}
	return commit.bitArray
}

// GetByIndex returns the vote corresponding to a given validator index
func (commit *Commit) GetByIndex(index int) *Vote {
	return commit.Precommits[index]
}

// IsCommit returns true if there is at least one vote
func (commit *Commit) IsCommit() bool {
	return len(commit.Precommits) != 0
}

// ValidateBasic performs basic validation that doesn't involve state data.
func (commit *Commit) ValidateBasic() error {
	if commit.BlockID.IsZero() {
		return errors.New("Commit cannot be for nil block")
	}
	if len(commit.Precommits) == 0 {
		return errors.New("No precommits in commit")
	}
	height, round := commit.Height(), commit.Round()

	// validate the precommits
	for _, precommit := range commit.Precommits {
		// It's OK for precommits to be missing.
		if precommit == nil {
			continue
		}
		// Ensure that all votes are precommits
		if precommit.Type != VoteTypePrecommit {
			return fmt.Errorf("Invalid commit vote. Expected precommit, got %v",
				precommit.Type)
		}
		// Ensure that all heights are the same
		if precommit.Height != height {
			return fmt.Errorf("Invalid commit precommit height. Expected %v, got %v",
				height, precommit.Height)
		}
		// Ensure that all rounds are the same
		if precommit.Round != round {
			return fmt.Errorf("Invalid commit precommit round. Expected %v, got %v",
				round, precommit.Round)
		}
	}
	return nil
}

// Hash returns the hash of the commit
func (commit *Commit) Hash() cmn.HexBytes {
	if commit.hash == nil {
		bs := make([]merkle.Hasher, len(commit.Precommits))
		for i, precommit := range commit.Precommits {
			bs[i] = aminoHasher(precommit)
		}
		commit.hash = merkle.SimpleHashFromHashers(bs)
	}
	return commit.hash
}

// StringIndented returns a string representation of the commit
func (commit *Commit) StringIndented(indent string) string {
	if commit == nil {
		return "nil-Commit"
	}
	precommitStrings := make([]string, len(commit.Precommits))
	for i, precommit := range commit.Precommits {
		precommitStrings[i] = precommit.String()
	}
	return fmt.Sprintf(`Commit{
%s  BlockID:    %v
%s  Precommits: %v
%s}#%v`,
		indent, commit.BlockID,
		indent, strings.Join(precommitStrings, "\n"+indent+"  "),
		indent, commit.hash)
}

//-----------------------------------------------------------------------------

// SignedHeader is a header along with the commits that prove it
type SignedHeader struct {
	Header *Header `json:"header"`
	Commit *Commit `json:"commit"`
}

//-----------------------------------------------------------------------------

// Data contains the set of transactions included in the block
type Data struct {
	// Txs that will be applied by state @ block.Height+1.
	// NOTE: not all txs here are valid.  We're just agreeing on the order first.
	// This means that block.AppHash does not include these txs.
	Txs Txs `json:"txs"`

	// Volatile
	hash cmn.HexBytes

	//txs hash list for all txs in last block
	LastTxsHashList HashList `json:"lastTxsHashList"`
}

// Hash returns the hash of the data
func (data *Data) Hash() cmn.HexBytes {
	if data == nil {
		return (Txs{}).Hash()
	}
	if data.hash == nil {
		data.hash = data.Txs.Hash() // NOTE: leaves of merkle tree are TxIDs
	}
	return data.hash
}

// StringIndented returns a string representation of the transactions
func (data *Data) StringIndented(indent string) string {
	if data == nil {
		return "nil-Data"
	}
	txStrings := make([]string, cmn.MinInt(len(data.Txs), 21))
	for i, tx := range data.Txs {
		if i == 20 {
			txStrings[i] = fmt.Sprintf("... (%v total)", len(data.Txs))
			break
		}
		txStrings[i] = fmt.Sprintf("Tx:%v", tx)
	}
	return fmt.Sprintf(`Data{
%s  %v
%s}#%v`,
		indent, strings.Join(txStrings, "\n"+indent+"  "),
		indent, data.hash)
}

//-----------------------------------------------------------------------------

// EvidenceData contains any evidence of malicious wrong-doing by validators
type EvidenceData struct {
	Evidence EvidenceList `json:"evidence"`

	// Volatile
	hash cmn.HexBytes
}

// Hash returns the hash of the data.
func (data *EvidenceData) Hash() cmn.HexBytes {
	if data.hash == nil {
		data.hash = data.Evidence.Hash()
	}
	return data.hash
}

// StringIndented returns a string representation of the evidence.
func (data *EvidenceData) StringIndented(indent string) string {
	if data == nil {
		return "nil-Evidence"
	}
	evStrings := make([]string, cmn.MinInt(len(data.Evidence), 21))
	for i, ev := range data.Evidence {
		if i == 20 {
			evStrings[i] = fmt.Sprintf("... (%v total)", len(data.Evidence))
			break
		}
		evStrings[i] = fmt.Sprintf("Evidence:%v", ev)
	}
	return fmt.Sprintf(`Data{
%s  %v
%s}#%v`,
		indent, strings.Join(evStrings, "\n"+indent+"  "),
		indent, data.hash)
	return ""
}

//--------------------------------------------------------------------------------

// BlockID defines the unique ID of a block as its Hash and its PartSetHeader
type BlockID struct {
	Hash        cmn.HexBytes  `json:"hash"`
	PartsHeader PartSetHeader `json:"parts"`
}

// IsZero returns true if this is the BlockID for a nil-block
func (blockID BlockID) IsZero() bool {
	return len(blockID.Hash) == 0 && blockID.PartsHeader.IsZero()
}

// Equals returns true if the BlockID matches the given BlockID
func (blockID BlockID) Equals(other BlockID) bool {
	return bytes.Equal(blockID.Hash, other.Hash) &&
		blockID.PartsHeader.Equals(other.PartsHeader)
}

// Key returns a machine-readable string representation of the BlockID
func (blockID BlockID) Key() string {
	bz, err := cdc.MarshalBinaryBare(blockID.PartsHeader)
	if err != nil {
		panic(err)
	}
	return string(blockID.Hash) + string(bz)
}

// String returns a human readable string representation of the BlockID
func (blockID BlockID) String() string {
	return fmt.Sprintf(`%v:%v`, blockID.Hash, blockID.PartsHeader)
}

//-------------------------------------------------------

type hasher struct {
	item interface{}
}

func (h hasher) Hash() []byte {
	hasher := ripemd160.New()
	if h.item != nil && !cmn.IsTypedNil(h.item) && !cmn.IsEmpty(h.item) {
		bz, err := cdc.MarshalBinaryBare(h.item)
		if err != nil {
			panic(err)
		}
		_, err = hasher.Write(bz)
		if err != nil {
			panic(err)
		}
	}
	return hasher.Sum(nil)

}

func aminoHash(item interface{}) []byte {
	h := hasher{item}
	return h.Hash()
}

func aminoHasher(item interface{}) merkle.Hasher {
	return hasher{item}
}
