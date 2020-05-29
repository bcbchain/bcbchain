package stubs

import (
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
)

type ContractStub interface {
	Name(addr smc.Address) string
	Methods(addr smc.Address) []Method
	Dispatcher(items *stubapi.InvokeParams) (response stubapi.Response, bcerr bcerrors.BCError)
	CodeHash() []byte
}
