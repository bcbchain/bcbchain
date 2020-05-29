package exec

import (
	"github.com/bcbchain/sdk/sdk/bn"
	"fmt"
	crypto2 "github.com/bcbchain/bclib/tendermint/go-crypto"

	"github.com/bcbchain/bcbchain/hyperledger/burrow/binary"
)

type CallType uint32

const (
	CallTypeCall     = CallType(0x00)
	CallTypeCode     = CallType(0x01)
	CallTypeDelegate = CallType(0x02)
	CallTypeStatic   = CallType(0x03)
	CallTypeSNative  = CallType(0x04)
)

var nameFromCallType = map[CallType]string{
	CallTypeCall:     "Call",
	CallTypeCode:     "CallCode",
	CallTypeDelegate: "DelegateCall",
	CallTypeStatic:   "StaticCall",
	CallTypeSNative:  "SNativeCall",
}

var callTypeFromName = make(map[string]CallType)

func init() {
	for t, n := range nameFromCallType {
		callTypeFromName[n] = t
	}
}

func CallTypeFromString(name string) CallType {
	return callTypeFromName[name]
}

func (ct CallType) String() string {
	name, ok := nameFromCallType[ct]
	if ok {
		return name
	}
	return "UnknownCallType"
}

func (ct CallType) MarshalText() ([]byte, error) {
	return []byte(ct.String()), nil
}

func (ct *CallType) UnmarshalText(data []byte) error {
	*ct = CallTypeFromString(string(data))
	return nil
}

type CallData struct {
	Caller crypto2.Address `json:"Caller"`
	Callee crypto2.Address `json:"Callee"`
	Data   binary.HexBytes `json:"Data"`
	Value  bn.Number       `json:"Value,omitempty"`
	Gas    uint64          `json:"Gas,omitempty"`
}

func (m *CallData) String() string {
	return fmt.Sprintf("{Caller: %s, Callee: %s, Data: %s, Value: %s, Gas: %d}",
		m.Caller,
		m.Callee,
		m.Data.String(),
		m.Value.String(),
		m.Gas)
}

func (m *CallData) GetValue() bn.Number {
	if m != nil {
		return m.Value
	}
	return bn.N(0)
}

func (m *CallData) GetGas() uint64 {
	if m != nil {
		return m.Gas
	}
	return 0
}

type CallEvent struct {
	CallType   CallType        `json:"CallType,omitempty"`
	CallData   *CallData       `json:"CallData,omitempty"`
	Origin     crypto2.Address `json:"Origin"`
	StackDepth uint64          `json:"StackDepth,omitempty"`
	Return     binary.HexBytes `json:"Return"`
}

func (m *CallEvent) String() string {
	return fmt.Sprintf("{CallType: %s, CallData: %s, Origin: %s, StackDepth: %d, Return: %s}",
		m.CallType.String(),
		m.CallData.String(),
		m.Origin,
		m.StackDepth,
		m.Return.String())
}

func (m *CallEvent) GetCallType() CallType {
	if m != nil {
		return m.CallType
	}
	return 0
}

func (m *CallEvent) GetCallData() *CallData {
	if m != nil {
		return m.CallData
	}
	return nil
}

func (m *CallEvent) GetStackDepth() uint64 {
	if m != nil {
		return m.StackDepth
	}
	return 0
}
