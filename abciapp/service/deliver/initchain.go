package deliver

import (
	"encoding/hex"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bcbchain/smcrunctl/adapter"
	"github.com/bcbchain/bclib/algorithm"
	"github.com/bcbchain/bclib/dockerlib"
	"github.com/bcbchain/bclib/fs"
	"github.com/bcbchain/bclib/types"
	"github.com/bcbchain/sdk/sdk/rlp"
	"github.com/bcbchain/sdk/sdk/std"
	sdktypes "github.com/bcbchain/sdk/sdk/types"
	"os"
	"path/filepath"
	"time"

	abci "github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
	"github.com/json-iterator/go"

	abcicommon "github.com/bcbchain/bcbchain/abciapp/common"
)

const (
	initChainMethodIDProtoType = "CreateGenesis(string)"
)

// InitChainInfo init chain parameters info
type InitChainInfo struct {
	Validators []abci.Validator `json:"validators,omitempty"`
	ChainID    string           `json:"chain_id,omitempty"`
	AppState   InitAppState     `json:"app_state,omitempty"`
}

// InitAppState init chain app state
type InitAppState struct {
	Organization   string        `json:"organization,omitempty"`
	GasPriceRatio  string        `json:"gas_price_ratio"`
	ChainVersion   int64         `json:"chainVersion,omitempty"`
	Token          std.Token     `json:"token,omitempty"`
	RewardStrategy []Rewarder    `json:"rewardStrategy,omitempty"`
	Contracts      []Contract    `json:"contracts,omitempty"`
	OrgBind        OrgBind       `json:"orgBind"`
	MainChain      MainChainInfo `json:"mainChain"`
}

type OrgBind struct {
	OrgName string        `json:"orgName"`
	Owner   types.Address `json:"owner"`
}

type MainChainInfo struct {
	OpenUrls   []string                           `json:"openUrls"`
	Validators map[string]statedbhelper.Validator `json:"validators"`
}

// Rewarder reward info
type Rewarder struct {
	Name          string `json:"name,omitempty"`          // 被奖励者名称
	RewardPercent string `json:"rewardPercent,omitempty"` // 奖励比例
	Address       string `json:"address,omitempty"`       // 被奖励者地址
}

// Contract contract info
type Contract struct {
	Name       string            `json:"name,omitempty"`
	Version    string            `json:"version,omitempty"`
	CodeByte   sdktypes.HexBytes `json:"codeByte,omitempty"`
	CodeHash   string            `json:"codeHash,omitempty"`
	Owner      string            `json:"owner,omitempty"`
	CodeDevSig Signature         `json:"codeDevSig,omitempty"`
	CodeOrgSig Signature         `json:"codeOrgSig,omitempty"`
}

// Signature sig for contract code
type Signature struct {
	PubKey    string `json:"pubkey"`
	Signature string `json:"signature"`
}

//nolint
func (app *AppDeliver) initChain(req abci.RequestInitChain) (response abci.ResponseInitChain) {
	// 解析 req ，构造 json
	app.logger.Info("Recv ABCI interface: InitChain", "chain_id", req.ChainId, "chain_version", req.ChainVersion)
	initAppState := new(InitAppState)
	err := jsoniter.Unmarshal(req.AppStateBytes, initAppState)
	if err != nil {
		return
	}

	chainIDFilePath := filepath.Join(abcicommon.GlobalConfig.Path, "genesis")
	exist, err := fs.PathExists(chainIDFilePath)
	if err != nil {
		panic(err)
	}

	if exist {
		fi, err := os.OpenFile(chainIDFilePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
		if err != nil {
			panic(err)
		}
		defer fi.Close()

		if _, err = fi.WriteString(req.ChainId); err != nil {
			panic(err)
		}
	}

	initChainInfo := new(InitChainInfo)
	initChainInfo.AppState = *initAppState
	initChainInfo.Validators = req.Validators

	for i, v := range req.Validators {
		pb, _ := crypto.PubKeyFromBytes(v.PubKey)

		test := pb.(crypto.PubKeyEd25519)
		app.logger.Debug("validator", "pubkey", hex.EncodeToString(pb.Bytes()))
		initChainInfo.Validators[i].PubKey = test[:]
	}
	initChainInfo.ChainID = req.ChainId
	app.SetChainID(req.ChainId)
	statedbhelper.SetChainIDOnce(req.ChainId)

	// 删除所有名字以chainID为前缀的容器
	prefix := req.ChainId + "."
	d := dockerlib.GetDockerLib()
	d.SetPrefix(prefix)
	d.Reset(prefix)

	// 检查创世世界状态hash，判断是否已清除数据。
	transID, _ := statedbhelper.NewCommittableTransactionID()
	txID := statedbhelper.NewTx(transID)
	appState := statedbhelper.GetWorldAppState(transID, txID)

	if appState != nil && len(appState.AppHash) != 0 {
		response.Code = types.ErrLogicError
		response.Log = "Not clear genesis data."
		return
	}
	crypto.SetChainId(req.ChainId)
	initChainInfo.AppState = *initAppState

	adapterIns := adapter.GetInstance()

	methodIDBytes := algorithm.CalcMethodId(initChainMethodIDProtoType)
	methodID := uint32(algorithm.BytesToInt32(methodIDBytes))
	infoByte, err := jsoniter.Marshal(initChainInfo)
	if err != nil {
		return
	}
	data, err := rlp.EncodeToBytes(string(infoByte))
	if err != nil {
		panic(err.Error())
	}

	addr := std.GetGenesisContractAddr(statedbhelper.GetChainID())
	items := make([]common.HexBytes, 0, 1)
	items = append(items, data)
	msg := types.Message{
		Contract: addr,
		MethodID: methodID,
		Items:    items,
	}

	tx := types.Transaction{
		Nonce:    1,
		GasLimit: 1,
		Note:     "genesis",
		Messages: make([]types.Message, 0, 1),
	}
	tx.Messages = append(tx.Messages, msg)
	header := abci.Header{
		ChainID:         req.ChainId,
		Height:          0,
		Time:            time.Now().Unix(),
		NumTxs:          0,
		LastBlockID:     abci.BlockID{},
		LastCommitHash:  nil,
		DataHash:        nil,
		ValidatorsHash:  nil,
		LastAppHash:     nil,
		LastFee:         0,
		LastAllocation:  nil,
		ProposerAddress: "",
		RewardAddress:   "",
		RandomeOfBlock:  nil,
		Version:         "",
		ChainVersion:    2,
	}

	r := adapterIns.InvokeTx(header, transID, txID, addr, tx, nil, nil, nil)
	if r.Code != types.CodeOK {
		panic("Genesis failed, log:" + r.Log)
	}

	var responseList []abci.ResponseInitChain
	err = jsoniter.Unmarshal([]byte(r.Data), &responseList)
	if err != nil {
		panic("Genesis failed, log:" + r.Log)
	}

	adapterIns.Commit(transID)
	statedbhelper.CommitTx(transID, 1)
	statedbhelper.CommitBlock(transID)
	response.Code = r.Code
	response.Log = r.Log
	if len(responseList) != 1 {
		panic("initChain failed.")
	}
	response = responseList[0]

	return
}
