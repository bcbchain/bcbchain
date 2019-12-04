package relay

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	jsoniter "github.com/json-iterator/go"
	"github.com/tendermint/abci/types"
	"github.com/tendermint/tmlibs/log"
	"strconv"
	"strings"
)

type QueueRelay struct {
	LocalURL   string
	RemoteURLs []string
	QueueID    string

	genesisOrgID string

	signalChan chan bool

	remoteSeq       uint64
	currentRoundURL string
	remoteIBC       *IBCContractInfo

	currentNode *CurrentNodeInfo

	logger log.Logger
}

type CurrentNodeInfo struct {
	Address    string
	HexPrivKey string
	Nonce      uint64
}

type IBCContractInfo struct {
	Address  string
	MethodID uint32
}

// Start start relay goroutine for queueID
func (qr *QueueRelay) Start() {

	running := false
	lastResult := false

	for {
		if running {
			lastResult = qr.carry(lastResult)
			if lastResult == false {
				running = false
			}
		} else {
			running = <-qr.signalChan
			lastResult = false
		}

		// 如果 signalChan 中存在多个数据，全部读取出来，取最后的状态
		for {
			select {
			case running = <-qr.signalChan:
				continue
			default:
			}
			break
		}

	}
}

func (qr *QueueRelay) calcRoundURL() {
	r := make([]byte, 4)
	_, e := rand.Read(r)
	if e != nil {
		panic(e)
	}

	buf := bytes.NewBuffer(r)
	var random uint8
	_ = binary.Read(buf, binary.BigEndian, &random)

	qr.currentRoundURL = qr.RemoteURLs[int(random)%len(qr.RemoteURLs)]
}

func (qr *QueueRelay) carry(lastResult bool) bool {
	qr.calcRoundURL()

	if qr.remoteIBC == nil {
		if err := qr.getTargetIBCContract(); err != nil {
			qr.logger.Debug("RELAY", "query remote ibc failed", err)
			return false
		}
	}

	if qr.currentNode.Nonce == 0 {
		if err := qr.getNonce(); err != nil {
			qr.logger.Debug("RELAY", "get nonce failed", err)
			return false
		}
	}

	pktsProof := qr.collectIBCPktsProof(lastResult)
	if len(pktsProof) == 0 {
		// 此时说明还没有发生跨链交易
		qr.logger.Debug("RELAY", "no packets")
		return false
	}

	if err := qr.sendIBCPackets(pktsProof); err != nil {
		// 不能发送跨链交易到目标链
		qr.logger.Debug("RELAY", "send tx failed", err)
		return false
	}
	return true
}

func (qr *QueueRelay) sendIBCPackets(pktsProofs []*PktsProof) error {
	tx, err := qr.packTx(pktsProofs)
	if err != nil {
		return err
	}

	result := new(ResultBroadcastTxCommit)
	client := getClient(qr.currentRoundURL)
	if _, err = client.Call(
		"broadcast_tx_commit",
		map[string]interface{}{"tx": []byte(tx)},
		result); err != nil {

		return err
	}
	return qr.processTxResult(result)
}

func (qr *QueueRelay) processTxResult(result *ResultBroadcastTxCommit) error {

	if result.CheckTx.Code == 200 && result.DeliverTx.Code == 200 {
		qr.currentNode.Nonce++
		return nil
	}

	qr.remoteIBC = nil
	qr.currentNode.Nonce = 0

	if result.CheckTx.Code != 200 {
		return errors.New(result.CheckTx.Log)
	} else {
		return errors.New(result.DeliverTx.Log)
	}
}

func (qr *QueueRelay) packTx(pktsProofs []*PktsProof) (string, error) {
	params := make([]interface{}, 1)
	params[0] = pktsProofs

	otherOrgID := getOtherOrgID(pktsProofs, qr.genesisOrgID)
	ibcContract := qr.remoteIBC.Address

	to := ""
	if len(otherOrgID) != 0 {
		to = otherOrgID + "." + ibcContract
	} else {
		to = qr.genesisOrgID + "." + ibcContract
	}
	_, toChainID := splitQueueID(qr.QueueID)

	return generateTx(to, qr.remoteIBC.MethodID, params, qr.currentNode.Nonce+1, 0, "", qr.currentNode.HexPrivKey, toChainID), nil
}

func (qr *QueueRelay) collectIBCPktsProof(lastResult bool) (pktsProofs []*PktsProof) {
	if lastResult == false {
		if seq, err := qr.getRemoteSequence(); err != nil {
			return
		} else {
			qr.remoteSeq = seq
		}
	}

	for {
		height := qr.getHeight(qr.remoteSeq + 1)
		if height == 0 {
			return
		}

		pktsProof, ibcMsgCount := qr.getPacketsProof(height)
		if pktsProof == nil {
			return
		}
		qr.remoteSeq += ibcMsgCount

		pktsProofs = append(pktsProofs, pktsProof)
		if len(pktsProofs) == 10 {
			return
		}
	}
}

func (qr *QueueRelay) getPacketsProof(height int64) (*PktsProof, uint64) {
	blkResults, err := qr.getBlockResult(height)
	if err != nil {
		return nil, 0
	}

	pktsProof, err := qr.getProof(height + 1)
	if err != nil {
		qr.logger.Warn("RELAY", "get proof err", err)
		return nil, 0
	}

	pktsProof.Packets = qr.getIBCPackets(blkResults)

	return pktsProof, uint64(len(pktsProof.Packets))
}

func (qr *QueueRelay) getProof(headerHeight int64) (pktsProof *PktsProof, err error) {
	var (
		block1, block2 *ResultBlock
	)

	if block1, err = qr.getBlock(headerHeight); err != nil {
		return
	}

	if block2, err = qr.getBlock(headerHeight + 1); err != nil {
		return
	}

	headerBytes, err := jsoniter.Marshal(block1.BlockMeta.Header)
	if err != nil {
		return
	}

	var header Header
	err = jsoniter.Unmarshal(headerBytes, &header)
	if err != nil {
		return
	}

	pktsProof = new(PktsProof)
	pktsProof.Header = header

	preCommitBytes, err := jsoniter.Marshal(block2.Block.LastCommit.Precommits)
	var preCommits []Precommit
	err = jsoniter.Unmarshal(preCommitBytes, &preCommits)
	if err != nil {
		return
	}
	pktsProof.Precommits = preCommits

	return
}

func (qr *QueueRelay) getRemoteSequence() (sequence uint64, err error) {
	sequence, err = querySequence(qr.currentRoundURL, qr.QueueID)
	return
}

func (qr *QueueRelay) getHeight(sequence uint64) int64 {
	msgIndex, err := queryIBCMsgIndex(qr.LocalURL, qr.QueueID, sequence)
	if err != nil {
		qr.logger.Debug("RELAY", "query msgIndex err", err)
		return 0
	}
	return msgIndex.Height
}

func (qr *QueueRelay) getBlockResult(height int64) (abciRes *ABCIResponses, err error) {
	response, err := blockResultQuery(qr.LocalURL, height)
	if err != nil {
		return
	}
	abciRes = response.Results
	return
}

func (qr *QueueRelay) getBlock(height int64) (resultBlock *ResultBlock, err error) {
	resultBlock, err = blockQuery(qr.LocalURL, height)
	return
}

func (qr *QueueRelay) getIBCPackets(abciResponse *ABCIResponses) (packets []Packet) {
	for _, deliverTx := range abciResponse.DeliverTx {
		packets = append(packets, qr.getPacketsFromTx(deliverTx)...)
	}
	return
}

func (qr *QueueRelay) getPacketsFromTx(deliverTx *types.ResponseDeliverTx) []Packet {
	if deliverTx.Code != types.CodeTypeOK {
		return nil
	}

	packets := make([]Packet, 0)
	for _, tag := range deliverTx.Tags {
		if strings.HasSuffix(string(tag.Key), "/ibc::packet/"+qr.QueueID) {
			var receipt Receipt
			err := jsoniter.Unmarshal(tag.Value, &receipt)
			if err != nil {
				panic(err)
			}

			var packet Packet
			err = jsoniter.Unmarshal(receipt.Bytes, &packet)
			if err != nil {
				panic(err)
			}

			packets = append(packets, packet)
		}
	}

	return packets
}

func (qr *QueueRelay) getTargetIBCContract() error {
	contract, err := queryIBCContract(qr.currentRoundURL, qr.genesisOrgID)
	if err != nil {
		qr.logger.Debug("RELAY", "query ibc err", err)
		return err
	}

	var item Method
	for _, methodItem := range contract.Methods {
		if strings.HasPrefix(methodItem.ProtoType, "Input") {
			item = methodItem
			break
		}
	}

	methodID, _ := strconv.ParseUint(item.MethodID, 16, 64)

	qr.remoteIBC = &IBCContractInfo{
		Address:  contract.Address,
		MethodID: uint32(methodID),
	}
	return nil
}

func (qr *QueueRelay) getNonce() error {
	nonce, err := queryAccountNonce(qr.currentRoundURL, qr.currentNode.Address)
	if err != nil {
		return err
	}

	qr.currentNode.Nonce = nonce
	return nil
}
