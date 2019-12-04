package deliver

import (
	"blockchain/abciapp/service/deliver"
	"blockchain/abciapp_v1.0/smcrunctl"
	"container/list"
	"strconv"
	"strings"
	"time"

	"common/bcdb"

	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/contract/contractdocker"
	"blockchain/abciapp_v1.0/contract/stubapi"
	"blockchain/abciapp_v1.0/contract/stubs"
	"blockchain/abciapp_v1.0/prototype"
	"blockchain/abciapp_v1.0/smc"
	"blockchain/abciapp_v1.0/statedb"
	"blockchain/abciapp_v1.0/types"
	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/tmlibs/log"
)

type DeliverConnection struct {
	logger      log.Loggerf
	blockHash   []byte
	blockHeader abci.Header // block header time
	db          *bcdb.GILevelDB
	docker      *contractdocker.ContractDocker
	appState    *abci.AppState
	stateDB     *statedb.StateDB
	hashList    *list.List // 存储deliver时产生的hash
	chainID     string
	sponser     smc.Address // 出块者地址
	rewarder    smc.Address // 奖励地址
	udValidator bool
	validators  []string
	fee         uint64                 // 总费用
	rewards     map[smc.Address]uint64 // 奖励策略
	minestub    *stubs.MNStub
	RespCode    uint32 // Response code of deliver tx, see stubapi.RESPONSE_CODE_*
	RespData    string // Response data of deliver tx,
	NameVersion string
	// it's contract address for RESPONSE_CODE_NEWTOKEN and RESPONSE_CODE_NEWCONTRACT
	// others, nil
}

func (conn *DeliverConnection) SetLogger(logger log.Loggerf) {
	conn.logger = logger
}

func (conn *DeliverConnection) SetDB(stateDB *bcdb.GILevelDB) {
	conn.db = stateDB
	conn.stateDB = statedb.NewStateDB(stateDB)
}

func (conn *DeliverConnection) StateDB() *statedb.StateDB {
	return conn.stateDB
}

func (conn *DeliverConnection) InitContractDocker() {

	for {
		conn.logger.Info("Init deliver docker for smart contract")
		contractAddrArry, err := conn.stateDB.GetContractAddrList()
		if err != nil {
			conn.logger.Fatal("Failed to init smart contract docker", "error", err)
			panic(err)
		}
		if contractAddrArry != nil && len(contractAddrArry) != 0 {
			var docker contractdocker.ContractDocker
			for _, contractAddr := range contractAddrArry {
				contract, err := conn.stateDB.GetContract(contractAddr)
				if err != nil {
					conn.logger.Error("get contract failed from stateDB ", "contractAddr", contractAddr)
					panic(err)
				}
				if contract.ChainVersion >= 2 {
					continue
				}
				conn.logger.Info("register smart contract stub", "name", contract.Name, "addr", contractAddr)

				switch contract.Name {
				case prototype.System:
					bcerr := docker.RegisterStub(contractAddr, stubs.NewSystemStub(conn.logger))
					if bcerr.ErrorCode != bcerrors.ErrCodeOK {
						conn.logger.Error("register smart contract stub error", "error", bcerr.Error())
					}
				case prototype.TokenBasic:
					bcerr := docker.RegisterStub(contractAddr, stubs.NewTokenBasicStub(conn.logger))
					if bcerr.ErrorCode != bcerrors.ErrCodeOK {
						conn.logger.Error("register smart contract stub error", "error", bcerr.Error())
					}
				case prototype.TokenIssue:
					bcerr := docker.RegisterStub(contractAddr, stubs.NewTokenIssueStub(conn.logger))
					if bcerr.ErrorCode != bcerrors.ErrCodeOK {
						conn.logger.Error("register smart contract stub error", "error", bcerr.Error())
					}
				case prototype.TokenTemplet:
					bcerr := docker.RegisterStub(contractAddr, stubs.NewTokenTempletStub(conn.logger))
					if bcerr.ErrorCode != bcerrors.ErrCodeOK {
						conn.logger.Error("register smart contract stub error", "error", bcerr.Error())
					}
					//todo 增加注册
				case prototype.TokenTrade:
					continue
					//代币注册
				case prototype.TokenBYB:
					bcerr := docker.RegisterStub(contractAddr, stubs.NewTokenBYBStub(conn.logger))
					if bcerr.ErrorCode != bcerrors.ErrCodeOK {
						conn.logger.Error("register smart contract stub error", "error", bcerr.Error())
					}
				case prototype.UPGRADE1TO2:
					bcerr := docker.RegisterStub(contractAddr, stubs.NewUpgrade1to2Stub(conn.logger))
					if bcerr.ErrorCode != bcerrors.ErrCodeOK {
						conn.logger.Error("register smart contract stub error ", "error", bcerr.Error())
					}
				case prototype.TB_Cancellation:
					bcerr := docker.RegisterStub(contractAddr, stubs.NewTBCStub(conn.logger))
					if bcerr.ErrorCode != bcerrors.ErrCodeOK {
						conn.logger.Error("register smart contract stub error ", "error", bcerr.Error())
					}
				case prototype.TB_Team:
					bcerr := docker.RegisterStub(contractAddr, stubs.NewTBTStub(conn.logger))
					if bcerr.ErrorCode != bcerrors.ErrCodeOK {
						conn.logger.Error("register smart contract stub error ", "error", bcerr.Error())
					}
				case prototype.TB_Foundation:
					bcerr := docker.RegisterStub(contractAddr, stubs.NewTBFStub(conn.logger))
					if bcerr.ErrorCode != bcerrors.ErrCodeOK {
						conn.logger.Error("register smart contract stub error ", "error", bcerr.Error())
					}
				case prototype.TAC:
					bcerr := docker.RegisterStub(contractAddr, stubs.NewTACStub(conn.logger))
					if bcerr.ErrorCode != bcerrors.ErrCodeOK {
						conn.logger.Error("register smart contract stub error ", "error", bcerr.Error())
					}
				default:
					if strings.Contains(contract.Name, "token-templet-") {
						bcerr := docker.RegisterStub(contractAddr, stubs.NewTokenTempletStub(conn.logger))
						if bcerr.ErrorCode != bcerrors.ErrCodeOK {
							conn.logger.Error("register smart contract stub error ", "error", bcerr.Error())
						}
					}
				}
			}
			conn.docker = &docker
			break
		}
		//2s检查一次状态库是否有写入
		time.Sleep(time.Second * 2)
	}
	conn.logger.Info("Deliver docker is ready")
}

func (conn *DeliverConnection) RegisterIntoContractDocker(respData string, respCode uint32, nameVersion string) {
	conn.logger.Info("Register new contract into deliver docker", "address", string(respData))

	var bcerr bcerrors.BCError
	if respCode == stubapi.RESPONSE_CODE_NEWTOKEN {
		bcerr = conn.docker.RegisterStub(respData, stubs.NewTokenTempletStub(conn.logger))
	} else if respCode == stubapi.RESPONSE_CODE_NEWBYBCONTRACT {
		bcerr = conn.docker.RegisterStub(respData, stubs.NewTokenBYBStub(conn.logger))
	} else if respCode == stubapi.RESPONSE_CODE_NEWTRANSFERAGENCY {
		bcerr = conn.docker.RegisterStub(respData, stubs.NewTACStub(conn.logger))
	} else if respCode == stubapi.RESPONSE_CODE_UPGRADE1TO2 {
		bcerr = conn.docker.RegisterStub(respData, stubs.NewUpgrade1to2Stub(conn.logger))
	} else if respCode == stubapi.RESPONSE_CODE_NEWTBCANCELLATIONCONTRACT {
		bcerr = conn.docker.RegisterStub(respData, stubs.NewTBCStub(conn.logger))
	} else if respCode == stubapi.RESPONSE_CODE_NEWTOKENBASICTEAM {
		bcerr = conn.docker.RegisterStub(respData, stubs.NewTBTStub(conn.logger))
	} else if respCode == stubapi.RESPONSE_CODE_NEWTOKENBASICFOUNDATION {
		bcerr = conn.docker.RegisterStub(respData, stubs.NewTBFStub(conn.logger))
	} else {
		bcerr = smcrunctl.GetInstance().RegisterIntoContractDocker(respData, nameVersion)
	}
	if bcerr.ErrorCode != bcerrors.ErrCodeOK {
		conn.logger.Error("RegisterStub ", "error", bcerr.Error())
	}
}

func (conn *DeliverConnection) InitChain(req abci.RequestInitChain) abci.ResponseInitChain {
	return conn.initChain(req)
}

func (conn *DeliverConnection) BeginBlock(req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return conn.BCBeginBlock(req)
}

func (conn *DeliverConnection) BeginBlockToV2(req abci.RequestBeginBlock) {
	conn.BCBeginBlockToV2(req)
}

func (conn *DeliverConnection) DeliverTx(tx []byte, connV2 *deliver.AppDeliver) abci.ResponseDeliverTx {
	return conn.deliverBCTx(tx, connV2)
}

func (conn *DeliverConnection) Flush(req abci.RequestFlush) abci.ResponseFlush {
	conn.logger.Debug("Recv ABCI interface: Flush")
	return abci.ResponseFlush{}
}

func (conn *DeliverConnection) EndBlock(req abci.RequestEndBlock) abci.ResponseEndBlock {
	conn.logger.Info("Recv ABCI interface: EndBlock", "height", req.Height)

	resp := abci.ResponseEndBlock{}

	// mining
	item, miningContract := conn.newInvokeParams()
	if conn.minestub == nil {
		conn.minestub = stubs.NewMNStub(conn.logger)
	}

	// check the height is effective
	if miningContract != nil && miningContract.ChainVersion == 0 {
		err := stubs.IsMiningRightHeight(item, miningContract, conn.logger)
		if err != nil {
			conn.logger.Debug("The contract is not effect", "err", err)
		} else {
			response, lerr := conn.minestub.Dispatcher(item, 0)
			if lerr.ErrorCode != bcerrors.ErrCodeOK {
				conn.logger.Error("mine fail", "errCode", lerr.ErrorCode, "err", lerr.Error())
			} else {
				u, _ := strconv.ParseInt(response.Data, 0, 64)
				resp.RewardAmount = u

				stateTx, _ := item.Ctx.TxState.CommitTx()
				if stateTx != nil {
					conn.calcDeliverTxHash(nil, nil, stateTx, nil)
				}
			}
		}
	}

	resp.ChainVersion = conn.appState.ChainVersion

	if conn.udValidator {
		var upValidators abci.Validators
		var stateDBValidators []*types.Validator
		for _, e := range conn.validators {
			validator, err := conn.stateDB.GetValidator(e)
			if err != nil {
				conn.logger.Error("can't get validators from stateDB", "err", err)
			}
			stateDBValidators = append(stateDBValidators, validator)
		}

		for _, e := range stateDBValidators {
			val := abci.Validator{
				PubKey:     e.NodePubKey,
				Power:      e.Power,
				RewardAddr: e.RewardAddr,
				Name:       e.Name,
			}
			upValidators = append(upValidators, val)
		}
		conn.udValidator = false
		conn.validators = nil

		resp.ValidatorUpdates = upValidators
		//return abci.ResponseEndBlock{ValidatorUpdates: upValidators, ChainVersion: conn.appState.ChainVersion}
	}
	//return abci.ResponseEndBlock{ChainVersion: conn.appState.ChainVersion}
	return resp
}

func (conn *DeliverConnection) Commit() abci.ResponseCommit {
	return conn.commitTx()
}

func (conn *DeliverConnection) newInvokeParams() (item *stubapi.InvokeParams, miningContract *types.Contract) {

	var txState *statedb.TxState

	//get mine contract addr
	contracts, err := conn.stateDB.GetContractAddrList()
	if err != nil {
		conn.logger.Fatal("Failed to init smart contract docker", "error", err)
		panic(err)
	}
	if contracts != nil && len(contracts) != 0 {
		var contract *types.Contract
		for _, v := range contracts {
			contract, err = conn.stateDB.GetContract(v)
			if err != nil {
				conn.logger.Error("get contract failed from stateDB ", "contractAddr", v)
				panic(err)
			}
			if contract.Name == prototype.MINING && contract.ChainVersion == 0 {
				miningContract = contract
			}
		}
	}
	if miningContract != nil {
		txState = conn.stateDB.NewTxState(miningContract.Address, "")
		txState.BeginTx()
	}

	// Generate accounts and execute
	sender := &stubapi.Account{
		smc.Address(conn.sponser),
		txState,
	}
	token, err := conn.stateDB.GetGenesisToken()
	if err != nil {
		conn.logger.Fatal("Failed to init smart contract docker", "error", err)
		panic(err)
	}
	owner := &stubapi.Account{
		token.Owner,
		txState,
	}
	proposer := &stubapi.Account{
		Addr: smc.Address(conn.sponser),
		//gTokenState,
	}
	rewarder := &stubapi.Account{
		Addr: smc.Address(conn.rewarder),
		//gTokenState,
	}
	invokeContext := &stubapi.InvokeContext{
		Sender:      sender,
		Owner:       owner,
		TxState:     txState,
		BlockHash:   conn.blockHash,
		BlockHeader: conn.blockHeader,
		Proposer:    proposer,
		Rewarder:    rewarder,
	}

	item = &stubapi.InvokeParams{
		Ctx: invokeContext,
	}

	return item, miningContract
}

// CommitTx2V1 - commit v2 deliverTx txBuffer to v1 blockBuffer.
func (conn *DeliverConnection) CommitTx2V1(txBuffer map[string][]byte) {
	conn.stateDB.CommitTx(txBuffer)
}
