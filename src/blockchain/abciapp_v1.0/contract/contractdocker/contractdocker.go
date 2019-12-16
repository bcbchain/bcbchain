//ContractDocker

package contractdocker

import (
	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/contract/stubapi"
	"blockchain/abciapp_v1.0/contract/stubs"
	"blockchain/abciapp_v1.0/smc"
	"blockchain/abciapp_v1.0/smcrunctl"
	"github.com/tendermint/tmlibs/log"
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

	stub, ok := docker.MapStubs[items.Ctx.TxState.ContractAddress]
	if ok {
		// check the height is effective
		err := stubs.IsRightHeight(items, nil)
		if err != nil {
			bcErr.ErrorCode = bcerrors.ErrCodeLowLevelError
			bcErr.ErrorDesc = err.Error()
			return
		}

		return stub.Dispatcher(items, transID)
	} else {
		return smcrunctl.GetInstance().Invoke(items, transID)
	}
}
