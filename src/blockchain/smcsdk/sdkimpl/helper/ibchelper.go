package helper

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/crypto/ed25519"
	"blockchain/smcsdk/sdk/ibc"
	"blockchain/smcsdk/sdk/rlp"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
	"blockchain/smcsdk/sdkimpl/helper/common"
	"blockchain/smcsdk/sdkimpl/object"
	"common/jsoniter"
	"encoding/hex"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tmlibs/merkle"
	"golang.org/x/crypto/sha3"
	"strings"
	"time"
)

// ReceiptHelper receipt helper information
type IBCHelper struct {
	smc sdk.ISmartContract //指向智能合约API对象指针

	receipts []types.KVPair
}

var _ sdk.IIBCHelper = (*IBCHelper)(nil)
var _ sdkimpl.IAcquireSMC = (*IBCHelper)(nil)

// SMC get smart contract object
func (ih *IBCHelper) SMC() sdk.ISmartContract { return ih.smc }

// SetSMC set smart contract object
func (ih *IBCHelper) SetSMC(smc sdk.ISmartContract) { ih.smc = smc }

// Hash create ibc hash from txHash、fromChainID and toChainID
func (ih *IBCHelper) IbcHash(toChainID string) types.HexBytes {
	hasherSHA3256 := sha3.New256()
	hasherSHA3256.Write(ih.smc.Tx().TxHash())
	hasherSHA3256.Write([]byte(ih.smc.Block().ChainID()))
	hasherSHA3256.Write([]byte(toChainID))

	return hasherSHA3256.Sum(nil)
}

// Run run function , then save receipts in member
func (ih *IBCHelper) Run(f func()) sdk.IIBCHelper {
	// step 1. save old all receipts
	oldReceipts := ih.smc.Message().(*object.Message).OutputReceipts()

	// step 2. run function
	f()

	// step 3. save new all receipts
	exReceipts := ih.smc.Message().(*object.Message).OutputReceipts()

	// step 4. sub new all receipts off old all receipts
	if len(exReceipts) > len(oldReceipts) {
		ih.receipts = exReceipts[len(oldReceipts):]
	}

	return ih
}

// Register invoke ibc contract's method of Register,
// register new side chain transaction
func (ih *IBCHelper) Register(toChainID string) {
	sdk.Require(len(ih.smc.Message().Origins()) == 1,
		types.ErrInvalidParameter, "invoke error")

	oldMsg := ih.smc.Message()
	methodID := "174bfaf8" // Register(string)

	// reset message for sdk
	ih.resetMessageForSDK(methodID, toChainID)

	_, receipts, err := sdkimpl.IBCInvokeFunc(ih.smc)
	sdk.Require(err.ErrorCode == types.CodeOK,
		err.ErrorCode, err.Error())

	ih.smc.(*sdkimpl.SmartContract).SetMessage(oldMsg)
	ih.smc.Message().(*object.Message).AppendOutput(receipts)
}

// Notify invoke ibc contract's method of Notify,
// it notify chainIDs one by one
func (ih *IBCHelper) Notify(chainIDs []string) {
	sdk.Require(len(ih.smc.Message().Origins()) == 1,
		types.ErrInvalidParameter, "invoke error")

	oldMsg := ih.smc.Message()
	methodID := "d1b772d0" // Notify([]string)

	// reset message for sdk
	ih.resetMessageForSDK(methodID, chainIDs)

	_, receipts, err := sdkimpl.IBCInvokeFunc(ih.smc)
	sdk.Require(err.ErrorCode == types.CodeOK,
		err.ErrorCode, err.Error())

	ih.smc.(*sdkimpl.SmartContract).SetMessage(oldMsg)
	ih.smc.Message().(*object.Message).AppendOutput(receipts)
}

// BroadcastToNet invoke ibc contract's method of BroadcastToNet,
// it notify all node in bcbChain one by one
func (ih *IBCHelper) Broadcast() {
	sdk.RequireMainChain()
	sdk.Require(len(ih.smc.Message().Origins()) == 1,
		types.ErrInvalidParameter, "invoke error")

	oldMsg := ih.smc.Message()
	methodID := "49ef1d15" // Broadcast()

	// reset message for sdk
	ih.resetMessageForSDK(methodID)

	_, receipts, err := sdkimpl.IBCInvokeFunc(ih.smc)
	sdk.Require(err.ErrorCode == types.CodeOK,
		err.ErrorCode, err.Error())

	ih.smc.(*sdkimpl.SmartContract).SetMessage(oldMsg)
	ih.smc.Message().(*object.Message).AppendOutput(receipts)
}

// CalcBlockHash cal
func (ih *IBCHelper) CalcBlockHash(h *ibc.Header) types.Hash {
	sdk.Require(h != nil,
		types.ErrInvalidParameter, "cannot calculate nil block header")

	// 将string装换为time.Time类型，否则计算的blockHash不正确
	t, err := time.Parse(ih.layoutByTime(h.Time), h.Time)
	sdk.RequireNotError(err, types.ErrInvalidParameter)
	aminoHasher := common.AminoHasher
	mapForHash := map[string]merkle.Hasher{
		"ChainID":        aminoHasher(h.ChainID),
		"Height":         aminoHasher(h.Height),
		"Time":           aminoHasher(t),
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
	if h.RandomOfBlock != nil {
		mapForHash["RandomOfBlock"] = aminoHasher(h.RandomOfBlock)
	}

	if h.LastMining != nil && *h.LastMining != 0 {
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
		type relayer struct {
			Address   types.Address `json:"address"`
			StartTime time.Time     `json:"start_time"`
		}
		t, err = time.Parse(ih.layoutByTime(h.Relayer.StartTime), h.Relayer.StartTime)
		sdk.RequireNotError(err, types.ErrInvalidParameter)
		r := relayer{
			Address:   h.Relayer.Address,
			StartTime: t,
		}
		mapForHash["Relayer"] = aminoHasher(r)
	}

	return merkle.SimpleHashFromMap(mapForHash)
}

// CalcQueueHash
func (ih *IBCHelper) CalcQueueHash(packet ibc.Packet, lastQueueHash types.Hash) types.Hash {

	packetBytes, err := jsoniter.Marshal(packet)
	if err != nil {
		panic(err)
	}

	hasherSHA256 := sha3.New256()
	hasherSHA256.Write(lastQueueHash)
	hasherSHA256.Write(packetBytes)

	return hasherSHA256.Sum(nil)
}

func (ih *IBCHelper) VerifyPrecommit(pubKey types.PubKey, precommit ibc.Precommit, chainID string, height int64) bool {
	type canonicalJSONPartSetHeader struct {
		Hash  types.HexBytes `json:"hash,omitempty"`
		Total int            `json:"total,omitempty"`
	}

	type canonicalJSONBlockID struct {
		Hash        types.HexBytes             `json:"hash,omitempty"`
		PartsHeader canonicalJSONPartSetHeader `json:"parts,omitempty"`
	}

	type canonicalJSONCommit struct {
		ChainID   string               `json:"@chain_id"`
		Type      string               `json:"@type"`
		BlockID   canonicalJSONBlockID `json:"block_id"`
		Height    int64                `json:"height"`
		Round     int                  `json:"round"`
		Timestamp string               `json:"timestamp"`
		VoteType  byte                 `json:"type"`
	}

	// TODO block: 2019-09-29T07:50:44.253022408Z/2019-09-29T11:29:36.490307Z(local) sig: 2019-09-29T07:50:44.253Z
	layout := ih.layoutByTime(precommit.Timestamp)
	sigTime, err := time.Parse(layout, precommit.Timestamp)
	sdk.RequireNotError(err, types.ErrInvalidParameter)

	// 此处不将hash重置为nil的话，MarshalJSON出来的结果如下：
	//{"@chain_id":"local","@type":"vote","block_id":{"parts":{}},"height":4512,"round":0,"timestamp":"2019-10-
	//23T07:53:37.793Z","type":2}
	// 重置为nil后，才是正确的结果，如下：
	//{"@chain_id":"local","@type":"vote","block_id":{},"height":4512,"round":0,"timestamp":"2019-10-23T07:53:37.793Z",
	// "type":2}
	blockHash := precommit.BlockID.Hash
	if len(blockHash) == 0 {
		blockHash = nil
	}
	partsHeaderHash := precommit.BlockID.PartsHeader.Hash
	if len(partsHeaderHash) == 0 {
		partsHeaderHash = nil
	}

	bz, err := common.CDC.MarshalJSON(canonicalJSONCommit{
		ChainID: chainID,
		Type:    "vote",
		BlockID: canonicalJSONBlockID{
			Hash: blockHash,
			PartsHeader: canonicalJSONPartSetHeader{
				Hash:  partsHeaderHash,
				Total: precommit.BlockID.PartsHeader.Total,
			},
		},
		Height:    height,
		Round:     precommit.Round,
		Timestamp: sigTime.Format(amino.RFC3339Millis),
		VoteType:  precommit.VoteType,
	})
	if err != nil {
		panic(err)
	}

	sdkimpl.Logger.Debug("verifySigh", string(bz), hex.EncodeToString(pubKey), hex.EncodeToString(precommit.Signature[:]))
	return ed25519.VerifySign(pubKey, bz, precommit.Signature[:])
}

// resetMessageForSDK create a new message and the reset it
func (ih *IBCHelper) resetMessageForSDK(mID string, params ...interface{}) {

	contract := ih.ibcContract()
	originMessage := ih.smc.Message()

	originList := ih.smc.Message().Origins()
	originList = append(originList, ih.smc.Message().Contract().Address())

	items := ih.pack(params...)

	newMsg := object.NewMessage(ih.smc, contract, mID, items, originMessage.Sender().Address(),
		originMessage.Payer().Address(), originList, ih.receipts)
	ih.smc.(*sdkimpl.SmartContract).SetMessage(newMsg)
	ih.receipts = nil
}

// pack pack params with rlp one by one
func (ih *IBCHelper) pack(params ...interface{}) []types.HexBytes {
	paramsRlp := make([]types.HexBytes, len(params))
	for i, param := range params {

		paramRlp, err := rlp.EncodeToBytes(param)
		if err != nil {
			panic(err)
		}
		paramsRlp[i] = paramRlp
	}

	return paramsRlp
}

// ibcContract return current effect ibc contract object
func (ih *IBCHelper) ibcContract() sdk.IContract {
	orgID := ih.smc.Helper().BlockChainHelper().CalcOrgID("genesis")

	key := std.KeyOfContractsWithName(orgID, "ibc")
	versionList := ih.smc.(*sdkimpl.SmartContract).LlState().McGet(key, &std.ContractVersionList{})
	sdk.Require(versionList != nil,
		types.ErrInvalidParameter, "deploy ibc contract first")

	vs := versionList.(*std.ContractVersionList)
	var contract sdk.IContract
	for i := len(vs.ContractAddrList) - 1; i >= 0; i-- {
		// return effective contract
		if ih.smc.Block().Height() >= vs.EffectHeights[i] {
			contract = ih.smc.Helper().ContractHelper().ContractOfAddress(vs.ContractAddrList[i])
			sdk.Require(contract.LoseHeight() == 0 || contract.LoseHeight() > ih.smc.Block().Height(),
				types.ErrInvalidParameter, "never effective ibc contract")
			break
		}
	}

	return contract
}

func (ih *IBCHelper) layoutByTime(timeStamp string) string {
	layout := "2006-01-02T15:04:05."
	splitTime := strings.Split(timeStamp, ".")
	sdk.Require(len(splitTime) == 2,
		types.ErrInvalidParameter, "invalid time")
	for index := 0; index < len(splitTime[1])-1; index += 1 {
		layout += "0"
	}
	layout += "Z"

	return layout
}
