package state

import (
	"blockchain/smcsdk/sdk/ibc"
	"blockchain/smcsdk/sdk/std"
	"common/jsoniter"
	"encoding/binary"
	"fmt"
	"github.com/ebuchman/fail-test"
	"github.com/pkg/errors"
	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/tendermint/mempool"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/relay"
	"github.com/tendermint/tendermint/sidechain"
	"github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tmlibs/db"
	"github.com/tendermint/tmlibs/log"
	"golang.org/x/crypto/sha3"
	"os"
	"strings"
)

var (
	SyncTo int64 = 0

	//return: config.GenesisFile(), config.ConfigFilePath(), config.DBDir(),config.ValidatorsFile(),config.PrivValidatorFile()
	ConfigPath func() (string, string, string, string, string)
	NodeStop   func()
)

//-----------------------------------------------------------------------------
// BlockExecutor handles block execution and state updates.
// It exposes ApplyBlock(), which validates & executes the block, updates state w/ ABCI responses,
// then commits and updates the mempool atomically, then saves state.

// BlockExecutor provides the context and accessories for properly executing a block.
type BlockExecutor struct {
	// save validators, consensus params, abci responses here
	db dbm.DB
	// save state
	dbx dbm.DB
	// execute the app against this
	proxyApp proxy.AppConnConsensus

	// events
	eventBus types.BlockEventPublisher

	// update these with block results after commit
	mempool types.Mempool
	evpool  types.EvidencePool

	logger log.Logger
}

// NewBlockExecutor returns a new BlockExecutor with a NopEventBus.
// Call SetEventBus to provide one.
func NewBlockExecutor(dbx dbm.DB, db dbm.DB, logger log.Logger, proxyApp proxy.AppConnConsensus,
	mempool types.Mempool, evpool types.EvidencePool) *BlockExecutor {
	return &BlockExecutor{
		db:       db,
		dbx:      dbx,
		proxyApp: proxyApp,
		eventBus: types.NopEventBus{},
		mempool:  mempool,
		evpool:   evpool,
		logger:   logger,
	}
}

// SetEventBus - sets the event bus for publishing block related events.
// If not called, it defaults to types.NopEventBus.
func (blockExec *BlockExecutor) SetEventBus(eventBus types.BlockEventPublisher) {
	blockExec.eventBus = eventBus
}

// ValidateBlock validates the given block against the given state.
// If the block is invalid, it returns an error.
// Validation does not mutate state, but does require historical information from the stateDB,
// ie. to verify evidence from a validator at an old height.
func (blockExec *BlockExecutor) ValidateBlock(s State, block *types.Block) error {
	return validateBlock(blockExec.db, s, block)
}

// ApplyBlock validates the block against the state, executes it against the app,
// fires the relevant events, commits the app, and saves the new state and responses.
// It's the only function that needs to be called
// from outside this package to process and commit an entire block.
// It takes a blockID to avoid recomputing the parts hash.
func (blockExec *BlockExecutor) ApplyBlock(s State, blockID types.BlockID, block *types.Block) (State, error) {
	if SyncTo > 0 && block.Height >= SyncTo {
		fmt.Println("block height ===", block.Height, ", exiting...")
		os.Exit(0)
	}
	if err := blockExec.ValidateBlock(s, block); err != nil {
		return s, ErrInvalidBlock(err)
	}

	//blockExec.logger.Info("block apply","height",block.Height,"block header apphash",block.Header.AppHash,"block proposer",block.Header.ProposerAddress)

	abciResponses, err := execBlockOnProxyApp(blockExec.logger, blockExec.proxyApp, block)
	if err != nil {
		return s, ErrProxyAppConn(err)
	}

	fail.Fail() // XXX

	// save the results before we commit
	saveABCIResponses(blockExec.db, block.Height, abciResponses)

	fail.Fail() // XXX

	// update the state with the block and responses
	s, err = updateState(s, blockID, block.Header, abciResponses, blockExec.db)
	if err != nil {
		return s, fmt.Errorf("Commit failed for application: %v", err)
	}

	if len(abciResponses.EndBlock.SCGenesis) > 0 {
		blockExec.logger.Debug("side chain genesis", "genesisInfo", abciResponses.EndBlock.SCGenesis)
		sideChainGenesis(abciResponses.EndBlock.SCGenesis, blockExec)
	}

	// lock mempool, commit state, update mempoool
	res, err := blockExec.Commit(block)
	if err != nil {
		return s, fmt.Errorf("Commit failed for application: %v", err)
	}

	fail.Fail() // XXX
	// AppHash and txHashLists
	appState := abci.ByteToAppState(res.AppState)
	s.LastAppHash = appState.AppHash
	s.LastTxsHashList = make([][]byte, 0)
	for _, txhash := range appState.TxsHashList {
		s.LastTxsHashList = append(s.LastTxsHashList, txhash)
	}

	s.LastFee = appState.Fee
	s.LastAllocation = []abci.Allocation{}
	for _, r := range appState.Rewards {
		k := string(r.Key)
		v := binary.BigEndian.Uint64(r.Value)
		s.LastAllocation = append(s.LastAllocation, abci.Allocation{Addr: k, Fee: v})
	}

	blockExec.logger.Info("block apply", "height", block.Height, "got hash", s.LastAppHash, "s.LastTxsHashList.length", len(s.LastTxsHashList))

	SaveState(blockExec.dbx, s)

	fail.Fail() // XXX

	// Update evpool now that state is saved
	// TODO: handle the crash/recover scenario
	// ie. (may need to call Update for last block)
	blockExec.evpool.Update(block)

	// events are fired after everything else
	// NOTE: if we crash between Commit and Save, events wont be fired during replay
	fireEvents(blockExec.logger, blockExec.eventBus, block, abciResponses)

	return s, nil
}

// Commit locks the mempool, runs the ABCI Commit message, and updates the mempool.
// It returns the result of calling abci.Commit (the AppHash), and an error.
// The Mempool must be locked during commit and update because state is typically reset on Commit and old txs must be replayed
// against committed state before new txs are run in the mempool, lest they be invalid.
func (blockExec *BlockExecutor) Commit(block *types.Block) (*abci.ResponseCommit, error) {
	blockExec.mempool.Lock()
	defer blockExec.mempool.Unlock()

	// while mempool is Locked, flush to ensure all async requests have completed
	// in the ABCI app before Commit.
	//err := blockExec.mempool.FlushAppConn()
	//if err != nil {
	//	blockExec.logger.Error("Client error during mempool.FlushAppConn", "err", err)
	//	return nil, err
	//}

	// Commit block, get hash back
	res, err := blockExec.proxyApp.CommitSync()
	if err != nil {
		blockExec.logger.Error("Client error during proxyAppConn.CommitSync", "err", err)
		return nil, err
	}

	// ResponseCommit has no error code - just data
	blockExec.logger.Info("Committed state",
		"height", block.Height,
		"txs", block.NumTxs,
		"responseCommit", fmt.Sprintf("%v", res))

	// Update mempool.
	if err := blockExec.mempool.Update(block.Height, block.Txs); err != nil {
		return nil, err
	}

	return res, nil
}

// makeQueueChain make queue chain for last block height.
func makeQueueChain(db dbm.DB, abciResponses *ABCIResponses, currentHeight int64) *ibc.QueueChain {

	// collect ibc packet
	mapForPackets := make(map[string][][]byte)
	for _, deliverTx := range abciResponses.DeliverTx {
		packets := getPackets(deliverTx)
		for queueID, packetList := range packets {
			if queueIDPackets, ok := mapForPackets[queueID]; !ok {
				queueIDPackets = make([][]byte, 0)
				queueIDPackets = append(queueIDPackets, packetList...)
				mapForPackets[queueID] = queueIDPackets
			} else {
				queueIDPackets = append(queueIDPackets, packetList...)
				mapForPackets[queueID] = queueIDPackets
			}
		}
	}

	if len(mapForPackets) == 0 {
		return nil
	}

	queueChain := ibc.QueueChain{QueueBlocks: make([]ibc.QueueBlock, 0)}
	for queueID, packets := range mapForPackets {

		// loadLastQueueInfo
		lastQueueHeight, lastQueueHash, err := loadLastQueueInfo(db, queueID)
		if err != nil {
			//panic(err.Error())
			lastQueueHeight = 0
		}

		// calcQueueHash -> makeQueueChain
		queueHash := calcQueueHash(lastQueueHash, packets)

		queueBlock := ibc.QueueBlock{
			QueueID:         queueID,
			QueueHash:       queueHash,
			LastQueueHash:   lastQueueHash,
			LastQueueHeight: lastQueueHeight}
		queueChain.QueueBlocks = append(queueChain.QueueBlocks, queueBlock)

		// saveLastQueueInfo
		saveLastQueueInfo(db, queueID, currentHeight, queueHash)
	}

	return &queueChain
}

//---------------------------------------------------------
// Helper functions for executing blocks and updating state

// Executes block's transactions on proxyAppConn.
// Returns a list of transaction results and updates to the validator set
func execBlockOnProxyApp(logger log.Logger, proxyAppConn proxy.AppConnConsensus, block *types.Block) (*ABCIResponses, error) {
	rc := relay.GetRelayController()
	if rc != nil {
		rc.SetNewHeader(block.Header)
	}

	var validTxs, invalidTxs = 0, 0

	txIndex := 0
	abciResponses := NewABCIResponses(block)

	// Execute transactions and get hash
	proxyCb := func(req *abci.Request, res *abci.Response) {
		switch r := res.Value.(type) {
		case *abci.Response_DeliverTx:
			// TODO: make use of res.Log
			// TODO: make use of this info
			// Blocks may include invalid txs.
			txRes := r.DeliverTx
			if txRes.Code == abci.CodeTypeOK {
				validTxs++
			} else {
				logger.Debug("Invalid tx", "code", txRes.Code, "log", txRes.Log)
				invalidTxs++
			}
			abciResponses.DeliverTx[txIndex] = txRes
			txIndex++
		}
	}
	proxyAppConn.SetResponseCallback(proxyCb)

	// determine which validators did not sign last block
	absentVals := make([]int32, 0)
	for valI, vote := range block.LastCommit.Precommits {
		if vote == nil {
			absentVals = append(absentVals, int32(valI))
		}
	}

	// TODO: determine which validators were byzantine
	byzantineVals := make([]abci.Evidence, len(block.Evidence.Evidence))
	for i, ev := range block.Evidence.Evidence {
		byzantineVals[i] = abci.Evidence{
			PubKey: ev.Address(), // XXX
			Height: ev.Height(),
		}
	}

	// save txs to donecache
	mempool.GetDoneTxsCache().PushTxs(logger, block.Height, block.Txs)

	// Begin block
	res, err := proxyAppConn.BeginBlockSync(abci.RequestBeginBlock{
		Hash:                block.Hash(),
		Header:              types.TM2PB.Header(block.Header),
		AbsentValidators:    absentVals,
		ByzantineValidators: byzantineVals,
	})
	if err != nil {
		logger.Error("Error in proxyAppConn.BeginBlock", "err", err)
		return nil, err
	}
	//增加bcchain判断beginBlock是否正确
	if res.Code != abci.CodeTypeOK {

		logger.Error("abci server  excute beginBlock failed", "log", res.Log)
		return nil, errors.New(res.Log)
	}

	// Run txs of block
	for _, tx := range block.Txs {
		proxyAppConn.DeliverTxAsync(tx)
		if err := proxyAppConn.Error(); err != nil {
			return nil, err
		}
	}

	// End block
	abciResponses.EndBlock, err = proxyAppConn.EndBlockSync(abci.RequestEndBlock{Height: block.Height})
	if err != nil {
		logger.Error("Error in proxyAppConn.EndBlock", "err", err)
		return nil, err
	}

	logger.Warn("Executed block", "height", block.Height, "validTxs", validTxs, "invalidTxs", invalidTxs)

	valUpdates := abciResponses.EndBlock.ValidatorUpdates
	if len(valUpdates) > 0 {
		logger.Info("Updates to validators", "updates", abci.ValidatorsString(valUpdates))
	}

	go filterReceipts(abciResponses) // TODO 开一个线程专门解析收据
	return abciResponses, nil
}

// If more or equal than 1/3 of total voting power changed in one block, then
// a light client could never prove the transition externally. See
// ./lite/doc.go for details on how a light client tracks validators.
func updateValidators(currentSet *types.ValidatorSet, updates []abci.Validator) error {
	for _, v := range updates {
		pubkey, err := crypto.PubKeyFromBytes(v.PubKey) // NOTE: expects go-amino encoded pubkey
		if err != nil {
			return err
		}

		address := pubkey.Address(crypto.GetChainId())
		power := v.Power
		// mind the overflow from int64
		if power < 0 {
			return fmt.Errorf("Power (%d) overflows int64", v.Power)
		}

		_, val := currentSet.GetByAddress(address)
		if val == nil {
			// add val
			added := currentSet.Add(types.NewValidator(pubkey, power, v.RewardAddr, v.Name))
			if !added {
				return fmt.Errorf("Failed to add new validator %X with voting power %d", address, power)
			}
		} else if v.Power == 0 {
			// remove val
			_, removed := currentSet.Remove(address)
			if !removed {
				return fmt.Errorf("Failed to remove validator %X", address)
			}
		} else {
			// update val
			val.VotingPower = power
			updated := currentSet.Update(types.NewValidator(pubkey, power, v.RewardAddr, v.Name))
			if !updated {
				return fmt.Errorf("Failed to update validator %X with voting power %d", address, power)
			}
		}
	}
	return nil
}

// updateState returns a new State updated according to the header and responses.
func updateState(s State, blockID types.BlockID, header *types.Header,
	abciResponses *ABCIResponses, db dbm.DB) (State, error) {

	// copy the valset so we can apply changes from EndBlock
	// and update s.LastValidators and s.Validators
	prevValSet := s.Validators.Copy()
	nextValSet := prevValSet.Copy()

	// update the validator set with the latest abciResponses
	lastHeightValsChanged := s.LastHeightValidatorsChanged
	if len(abciResponses.EndBlock.ValidatorUpdates) > 0 {
		err := updateValidators(nextValSet, abciResponses.EndBlock.ValidatorUpdates)
		if err != nil {
			return s, fmt.Errorf("Error changing validator set: %v", err)
		}
		// change results from this height but only applies to the next height
		lastHeightValsChanged = header.Height + 1
	}

	// Update validator accums and set state variables
	nextValSet.IncrementAccum(1)

	// update the params with the latest abciResponses
	nextParams := s.ConsensusParams
	lastHeightParamsChanged := s.LastHeightConsensusParamsChanged
	if abciResponses.EndBlock.ConsensusParamUpdates != nil {
		// NOTE: must not mutate s.ConsensusParams
		nextParams = s.ConsensusParams.Update(abciResponses.EndBlock.ConsensusParamUpdates)
		err := nextParams.Validate()
		if err != nil {
			return s, fmt.Errorf("Error updating consensus params: %v", err)
		}
		// change results from this height but only applies to the next height
		lastHeightParamsChanged = header.Height + 1
	}

	chainVersion := s.ChainVersion
	if abciResponses.EndBlock.ChainVersion > s.ChainVersion {
		chainVersion = abciResponses.EndBlock.ChainVersion
	}

	var lastMining *int64
	if abciResponses.EndBlock.RewardAmount != 0 {
		lastMining = &abciResponses.EndBlock.RewardAmount
	}

	// makeQueueChain
	LastQueueChains := makeQueueChain(db, abciResponses, header.Height)

	// NOTE: the AppHash has not been populated.
	// It will be filled on state.Save.
	return State{
		ChainID:                          s.ChainID,
		LastBlockHeight:                  header.Height,
		LastBlockTotalTx:                 s.LastBlockTotalTx + header.NumTxs,
		LastBlockID:                      blockID,
		LastBlockTime:                    header.Time,
		Validators:                       nextValSet,
		LastValidators:                   s.Validators.Copy(),
		LastHeightValidatorsChanged:      lastHeightValsChanged,
		ConsensusParams:                  nextParams,
		LastHeightConsensusParamsChanged: lastHeightParamsChanged,
		LastResultsHash:                  abciResponses.ResultsHash(),
		LastAppHash:                      nil,
		ChainVersion:                     chainVersion,
		LastMining:                       lastMining,
		LastQueueChains:                  LastQueueChains,
	}, nil
}

// Fire NewBlock, NewBlockHeader.
// Fire TxEvent for every tx.
// NOTE: if Tendermint crashes before commit, some or all of these events may be published again.
func fireEvents(logger log.Logger, eventBus types.BlockEventPublisher, block *types.Block, abciResponses *ABCIResponses) {
	// NOTE: do we still need this buffer ?
	txEventBuffer := types.NewTxEventBuffer(eventBus, int(block.NumTxs))
	for i, tx := range block.Data.Txs {
		if e := txEventBuffer.PublishEventTx(types.EventDataTx{TxResult: types.TxResult{
			Height: block.Height,
			Index:  uint32(i),
			Tx:     tx,
			Result: *(abciResponses.DeliverTx[i]),
		}}); e != nil {
			logger.Warn("txEventBuffer.PublishEventTx", "e", e.Error())
		}
	}

	if e := eventBus.PublishEventNewBlock(types.EventDataNewBlock{Block: block}); e != nil {
		logger.Warn("eventBus.PublishEventNewBlock", "e", e.Error())
	}
	if e := eventBus.PublishEventNewBlockHeader(types.EventDataNewBlockHeader{Header: block.Header}); e != nil {
		logger.Warn("eventBus.PublishEventNewBlockHeader", "e", e.Error())
	}
	if err := txEventBuffer.Flush(); err != nil {
		logger.Error("Failed to flush event buffer", "err", err)
	}
}

//----------------------------------------------------------------------------------------------------
// Execute block without state. TODO: eliminate

// ExecCommitBlock executes and commits a block on the proxyApp without validating or mutating the state.
// It returns the application root hash (result of abci.Commit).
func ExecCommitBlock(appConnConsensus proxy.AppConnConsensus, block *types.Block, logger log.Logger) (*abci.ResponseCommit, error) {
	_, err := execBlockOnProxyApp(logger, appConnConsensus, block)
	if err != nil {
		logger.Error("Error executing block on proxy app", "height", block.Height, "err", err)
		return nil, err
	}
	// Commit block, get hash back
	res, err := appConnConsensus.CommitSync()
	if err != nil {
		logger.Error("Client error during proxyAppConn.CommitSync", "err", res)
		return nil, err
	}
	// ResponseCommit has no error or log, just data
	return res, nil
}

func sideChainGenesis(genesisInfos []*abci.SideChainGenesis, blockExec *BlockExecutor) {
	genesisInfo, ok := sidechain.ContainsCurrentNode(genesisInfos)
	if !ok {
		return
	}
	if NodeStop != nil {
		NodeStop()
	}

	sc := sidechain.NewSideChain(genesisInfo)
	err := sc.PrepareSCGenesis()
	if err != nil {
		panic(err)
	}

	response, err := blockExec.proxyApp.CleanDataSync()
	if err != nil {
		panic(err)
	}
	if response.Code != 200 { // todo 测试 panic
		panic("bcchain clean data failed:" + response.Log)
	}

	os.Exit(1)
}

// -------------------- IBC add 18 Seq. 2019------------------------

// getPackets get ibc::packet receipt from deliver tags.
// deliverTx only have one ibc::packet receipt in default
func getPackets(deliverTx *abci.ResponseDeliverTx) map[string][][]byte {
	if deliverTx.Code != abci.CodeTypeOK {
		return nil
	}

	packets := make(map[string][][]byte, 0)
	var err error
	for _, tag := range deliverTx.Tags {
		splitKey := strings.Split(string(tag.Key), "/")
		if strings.Contains(string(tag.Key), "/ibc::packet/") {
			var receipt std.Receipt
			err = jsoniter.Unmarshal(tag.Value, &receipt)
			if err != nil {
				panic(err)
			}

			queueID := splitKey[len(splitKey)-1]
			if items, ok := packets[queueID]; !ok {
				items = make([][]byte, 0)
				items = append(items, receipt.Bytes)
				packets[queueID] = items
			} else {
				items = append(items, receipt.Bytes)
				packets[queueID] = items
			}
		}
	}

	return packets
}

// -------------------- Relay add 17 Oct. 2019------------------------

// calcQueueHash calculate queue hash about lastQueueHash and packets.
func calcQueueHash(lastQueueHash []byte, packets [][]byte) []byte {

	hasherSHA256 := sha3.New256()

	for _, packetBytes := range packets {
		hasherSHA256.Reset()
		hasherSHA256.Write(lastQueueHash)
		hasherSHA256.Write(packetBytes)
		lastQueueHash = hasherSHA256.Sum(nil)
	}

	return lastQueueHash
}

func filterReceipts(abciResponse *ABCIResponses) {
	for _, deliverTx := range abciResponse.DeliverTx {
		if deliverTx.Code == 200 {
			chainIDToURLs := getOpenURLFromReceipts(deliverTx)
			for chainID, urls := range chainIDToURLs {
				rc := relay.GetRelayController()
				rc.UpdateOpenURL(chainID, urls)
			}
		}
	}
}

// TODO 检查侧链状态更改收据
func getOpenURLFromReceipts(deliverTx *abci.ResponseDeliverTx) (chainIDToURLs map[string][]string) {
	chainIDToURLs = make(map[string][]string)
	for _, tag := range deliverTx.Tags {
		if strings.HasSuffix(string(tag.Key), "/netgovernance.setOpenURL") ||
			strings.HasSuffix(string(tag.Key), "/IBC.setOpenURL") {
			var receipt relay.Receipt
			err := jsoniter.Unmarshal(tag.Value, &receipt)
			if err != nil {
				continue
			}

			type setOpenURL struct {
				SideChainID string   `json:"sideChainID"`
				OpenURLs    []string `json:"openURLs"`
			}

			r := new(setOpenURL)
			err = jsoniter.Unmarshal(receipt.Bytes, r)
			if err != nil {
				return
			}
			chainIDToURLs[r.SideChainID] = r.OpenURLs

		} else if strings.HasSuffix(string(tag.Key), "/netgovernance.genesisSideChain") {
			var receipt relay.Receipt
			err := jsoniter.Unmarshal(tag.Value, &receipt)
			if err != nil {
				panic(err)
			}

			type genesisSideChain struct {
				SideChainID string   `json:"sideChainID"`
				OpenURLs    []string `json:"openURLs"`
			}
			r := new(genesisSideChain)
			err = jsoniter.Unmarshal(receipt.Bytes, r)
			if err != nil {
				return
			}
			chainIDToURLs[r.SideChainID] = r.OpenURLs
		}
	}
	return
}
