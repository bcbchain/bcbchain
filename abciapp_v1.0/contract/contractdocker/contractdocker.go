//ContractDocker

package contractdocker

import (
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubs"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smcrunctl"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
)

type ContractDocker struct {
	Name     string                             //Contract docker name
	MapStubs map[smc.Address]stubs.ContractStub //The map of contracts and their stubs
	CLogger  log.Logger
}

// RegisterStub is used to register contracts' stubs in Docker.
func (docker *ContractDocker) RegisterStub(addr smc.Address, contractStub stubs.ContractStub) bcerrors.BCError {

	if docker.MapStubs == nil {
		docker.MapStubs = make(map[smc.Address]stubs.ContractStub)
	}
	// if contract is existing, return error
	if _, ok := docker.MapStubs[addr]; ok {
		return bcerrors.BCError{bcerrors.ErrCodeDockerDupRegist, ""}
	}

	docker.MapStubs[addr] = contractStub

	return bcerrors.BCError{bcerrors.ErrCodeOK, ""}
}

// Invoke -- entry function of invoking contract.
func (docker *ContractDocker) Invoke(items *stubapi.InvokeParams, transID int64) (response stubapi.Response, bcErr bcerrors.BCError) {
	// check the sender is in black list                                  //blacklist to del
	if items.Ctx.TxState.CheckBlackAddress(items.Ctx.Sender.Addr) { //blacklist to del
		bcErr.ErrorCode = bcerrors.ErrCodeInterContractsSenderInBlackList //blacklist to del
		return                                                            //blacklist to del
	} //blacklist to del

	stub, ok := docker.MapStubs[items.Ctx.TxState.ContractAddress]
	if ok {
		// check the height is effective
		err := stubs.IsRightHeight(items, nil)
		if err != nil {
			bcErr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcErr.ErrorDesc = err.Error()
			return
		}

		return stub.Dispatcher(items)
	} else {
		return smcrunctl.GetInstance().Invoke(items, transID)
	}
}
