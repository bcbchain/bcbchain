package deliver

import (
	"blockchain/abciapp_v1.0/bcerrors"
	sdbtype "blockchain/abciapp_v1.0/types"
	"blockchain/algorithm"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/go-crypto"
	"golang.org/x/crypto/sha3"
	"strconv"
	"strings"
)

//创世文件appstate结构
type initAppState struct {
	Token          *sdbtype.IssueToken `json:"token"`
	RewardStrategy []sdbtype.Rewarder  `json:"rewardStrategy"`
	Contracts      []*sdbtype.Contract `json:"contracts"`
}

func (conn *DeliverConnection) initChain(req abci.RequestInitChain) abci.ResponseInitChain {

	conn.logger.Info("Recv ABCI interface: InitChain", "chain_id", req.ChainId, "chain_version", req.ChainVersion)
	conn.logger.Info(fmt.Sprintf("AppState:\n%v", string(req.AppStateBytes)))

	for i, v := range req.Validators {
		conn.logger.Info("Validators", "index", i, "name", v.Name, "reward_addr", v.RewardAddr, "power", v.Power)
	}
	crypto.SetChainId(req.ChainId)

	stateJson := req.AppStateBytes
	var genesisState = initAppState{}

	err := json.Unmarshal(stateJson, &genesisState)
	if err != nil {
		conn.logger.Error("Error to unmarshal AppStateBytes:", err)
		return abci.ResponseInitChain{
			Code: bcerrors.ErrCodeLowLevelError,
			Log:  err.Error(),
		}
	}

	stateDB := conn.stateDB
	stateDB.BeginBlock()
	//In case error happens during init
	//It's OK to call RollBlock() follows Commit()
	defer stateDB.RollBlock()

	conn.stateDB.SetChainID(req.ChainId)

	rewardStrategy := make([]sdbtype.RewardStrategy, 0)

	var initStrategy sdbtype.RewardStrategy
	initStrategy.Strategy = genesisState.RewardStrategy
	initStrategy.EffectHeight = 1
	rewardStrategy = append(rewardStrategy, initStrategy)

	err = conn.stateDB.SetStrategys(rewardStrategy)
	if err != nil {
		conn.logger.Error("error to set rewardStrategy", err)

		return abci.ResponseInitChain{
			Code: bcerrors.ErrCodeLowLevelError,
			Log:  err.Error(),
		}
	}

	//var val types.Validator
	for _, validator := range req.Validators {

		val := sdbtype.Validator{
			Name:       validator.Name,
			NodePubKey: validator.PubKey,
			NodeAddr:   algorithm.CalcAddressFromCdcPubKey(req.ChainId, validator.PubKey),
			RewardAddr: validator.RewardAddr,
			Power:      validator.Power,
		}
		stateDB.SetValidator(&val)
	}

	//创世数据
	genesisToken := genesisState.Token
	err = stateDB.SetGenesisToken(genesisToken)
	if err != nil {
		conn.logger.Error("Error to SetGenesisToken:", err)

		return abci.ResponseInitChain{
			Code: bcerrors.ErrCodeLowLevelError,
			Log:  err.Error(),
		}
	}

	for _, contract := range genesisState.Contracts {
		if !strings.EqualFold(contract.Owner, genesisToken.Owner) {
			break

			return abci.ResponseInitChain{
				Code: bcerrors.ErrCodeLowLevelError,
				Log:  "Contract owner mismatch to token basic owner",
			}
		}
		//todo 计算智能合约methodId ,现json转到type为uint32时会报错,暂定义为string
		for _, method := range contract.Methods {
			//
			var methodId uint32
			methodIdByte := algorithm.CalcMethodId(method.Prototype)
			bytesBuffer := bytes.NewBuffer(methodIdByte)
			binary.Read(bytesBuffer, binary.BigEndian, &methodId)
			methodIdStr := strconv.FormatUint(uint64(methodId), 10)

			if strings.EqualFold(method.MethodId, methodIdStr) {

				return abci.ResponseInitChain{
					Code: bcerrors.ErrCodeLowLevelError,
					Log:  "Contract methodID is wrong,method:" + method.Prototype,
				}
			}
		}
		conn.logger.Info("init contract addr", "name", contract.Name, "addr", contract.Address)

		err = stateDB.SetGenesisContract(contract)
		if err != nil {
			conn.logger.Error("SetGenesisContracts failed,", "error", err)

			return abci.ResponseInitChain{
				Code: bcerrors.ErrCodeLowLevelError,
				Log:  err.Error(),
			}
		}
	}

	genData := stateDB.GetBlockBuffer()
	conn.logger.Info("GenesisData", "genData", string(genData))
	genHash := sha3.New256()
	genHash.Write(genData)

	genAppState := abci.AppState{
		BlockHeight: 0,
		AppHash:     genHash.Sum(nil),
	}
	conn.logger.Info("GenesisData", "Hash", hex.EncodeToString(genAppState.AppHash))

	err = stateDB.SetWorldAppState(&genAppState)
	stateDB.CommitBlock()

	return abci.ResponseInitChain{
		Code:        bcerrors.ErrCodeOK,
		GenAppState: abci.AppStateToByte(&genAppState),
	}
}

//处理json marshal工具
func (conn *DeliverConnection) jsonMarshal(o interface{}, name string) []byte {

	jsonByte, err := json.Marshal(o)
	if err != nil {
		conn.logger.Error("json marsha falied,err%s", err)
		return nil
	}
	return jsonByte
}
