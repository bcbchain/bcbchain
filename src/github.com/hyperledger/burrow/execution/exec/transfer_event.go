package exec

import (
	"blockchain/smcsdk/sdk/bn"
	"fmt"
	crypto2 "github.com/tendermint/go-crypto"
)

type TransferType uint32

const (
	TransferTypeCall = TransferType(0x00)
)

var nameFromTransferType = map[TransferType]string{
	TransferTypeCall: "Transfer",
}

var TransferTypeFromName = make(map[string]TransferType)

func init() {
	for t, n := range nameFromTransferType {
		TransferTypeFromName[n] = t
	}
}

func TransferTypeFromString(name string) TransferType {
	return TransferTypeFromName[name]
}

func (tt TransferType) String() string {
	name, ok := nameFromTransferType[tt]
	if ok {
		return name
	}
	return "UnknownTransferType"
}

func (tt TransferType) MarshalText() ([]byte, error) {
	return []byte(tt.String()), nil
}

func (tt *TransferType) UnmarshalText(data []byte) error {
	*tt = TransferTypeFromString(string(data))
	return nil
}

type TransferData struct {
	Token crypto2.Address `json:"token"`          // Token types.Address
	From  crypto2.Address `json:"from"`           // Account address of Sender
	To    crypto2.Address `json:"to"`             // Account address of Receiver
	Value bn.Number       `json:"value"`          // Transfer value
	Note  string          `json:"note,omitempty"` // Transfer note
}

func (m *TransferData) String() string {
	return fmt.Sprintf("{Caller: %s, Callee: %s, Data: %s, Value: %s, Gas: %d}",
		m.Token,
		m.From,
		m.To,
		m.Value.String(),
		m.Note)
}

func (m *TransferData) GetValue() bn.Number {
	if m != nil {
		return m.Value
	}
	return bn.N(0)
}

func (m *TransferData) GetNote() string {
	if m != nil {
		return m.Note
	}
	return ""
}

type TransferEvent struct {
	TransferData *TransferData `json:"TransferData,omitempty"`
}

func (m *TransferEvent) String() string {
	return fmt.Sprintf("{TransferData: %s,}", m.TransferData.String())
}

func (m *TransferEvent) GetTransferData() *TransferData {
	if m != nil {
		return m.TransferData
	}
	return nil
}
