package check

import (
	"github.com/bcbchain/bcbchain/abciapp/service/check"
	"strings"
	"time"

	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/contractdocker"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubs"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/prototype"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/statedb"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
)

type CheckConnection struct {
	logger  log.Loggerf
	docker  *contractdocker.ContractDocker
	stateDB *statedb.StateDB
}

func (conn *CheckConnection) SetLogger(logger log.Loggerf) {
	conn.logger = logger
}

func (conn *CheckConnection) NewStateDB() {
	conn.stateDB = statedb.NewStateDB()
}

func (conn *CheckConnection) InitContractDocker() {

	//读取状态数据库的创世信息
	//如果没有，忽略
	//如果有，校验创世信息的完整性
	//校验通过，读取智能合约信息，注册stub

	for {
		conn.logger.Info("Init check docker for smart contract")
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
				if contract == nil || contract.ChainVersion >= 2 {
					continue
				}
				conn.logger.Info("register smart contract stub", "name", contract.Name, "addr", contractAddr)

				switch contract.Name {
				case prototype.System:
					bcerr := docker.RegisterStub(contractAddr, stubs.NewSystemStub(conn.logger))
					if bcerr.ErrorCode != bcerrors.ErrCodeOK {
						conn.logger.Error("register smart contract stub error ", "error", bcerr.Error())
					}
				case prototype.TokenBasic:
					bcerr := docker.RegisterStub(contractAddr, stubs.NewTokenBasicStub(conn.logger))
					if bcerr.ErrorCode != bcerrors.ErrCodeOK {
						conn.logger.Error("register smart contract stub error ", "error", bcerr.Error())
					}
				case prototype.TokenIssue:
					bcerr := docker.RegisterStub(contractAddr, stubs.NewTokenIssueStub(conn.logger))
					if bcerr.ErrorCode != bcerrors.ErrCodeOK {
						conn.logger.Error("register smart contract stub error ", "error", bcerr.Error())
					}
				case prototype.TokenTemplet:
					bcerr := docker.RegisterStub(contractAddr, stubs.NewTokenTempletStub(conn.logger))
					if bcerr.ErrorCode != bcerrors.ErrCodeOK {
						conn.logger.Error("register smart contract stub error ", "error", bcerr.Error())
					}
					//todo 增加注册
				case prototype.TokenTrade:
					continue
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
	conn.logger.Info("Check docker is ready")
}

func (conn *CheckConnection) RegisterIntoContractDocker(contractAddress smc.Address, code uint32) {
	conn.logger.Info("Register contract into check docker", "address", string(contractAddress))
	var bcerr bcerrors.BCError

	if code == stubapi.RESPONSE_CODE_NEWTOKEN {
		bcerr = conn.docker.RegisterStub(contractAddress, stubs.NewTokenTempletStub(conn.logger))
	} else if code == stubapi.RESPONSE_CODE_NEWBYBCONTRACT {
		bcerr = conn.docker.RegisterStub(contractAddress, stubs.NewTokenBYBStub(conn.logger))
	} else if code == stubapi.RESPONSE_CODE_NEWTRANSFERAGENCY {
		bcerr = conn.docker.RegisterStub(contractAddress, stubs.NewTACStub(conn.logger))
	} else if code == stubapi.RESPONSE_CODE_UPGRADE1TO2 {
		bcerr = conn.docker.RegisterStub(contractAddress, stubs.NewUpgrade1to2Stub(conn.logger))
	} else if code == stubapi.RESPONSE_CODE_NEWTBCANCELLATIONCONTRACT {
		bcerr = conn.docker.RegisterStub(contractAddress, stubs.NewTBCStub(conn.logger))
	} else if code == stubapi.RESPONSE_CODE_NEWTOKENBASICTEAM {
		bcerr = conn.docker.RegisterStub(contractAddress, stubs.NewTBTStub(conn.logger))
	} else if code == stubapi.RESPONSE_CODE_NEWTOKENBASICFOUNDATION {
		bcerr = conn.docker.RegisterStub(contractAddress, stubs.NewTBFStub(conn.logger))
	}

	if bcerr.ErrorCode != bcerrors.ErrCodeOK {
		conn.logger.Error("RegisterStub ", "error", bcerr.Error())
	}
}

func (conn *CheckConnection) CheckTx(tx []byte, connV2 *check.AppCheck) types.ResponseCheckTx {
	conn.logger.Info("Recv ABCI interface: CheckTx", "tx", string(tx))

	if conn.docker == nil {
		conn.logger.Error("can't find checkTx docker ")
		bcerr := bcerrors.BCError{bcerrors.ErrCodeDockerNotFindDocker, ""}
		return types.ResponseCheckTx{
			Code: bcerr.ErrorCode,
			Log:  bcerr.Error(),
		}
	}

	return conn.CheckBCTx(tx, connV2)
}
