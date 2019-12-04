package stubs

import (
	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/contract/stubapi"
	"blockchain/abciapp_v1.0/smc"
)

type ContractStub interface {
	Name(addr smc.Address) string
	Methods(addr smc.Address) []Method
	Dispatcher(items *stubapi.InvokeParams, transID int64) (response stubapi.Response, bcerr bcerrors.BCError)
	CodeHash() []byte
}
