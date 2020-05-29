package acmstate

import (
	"github.com/bcbchain/sdk/sdk/types"
	"bytes"
	"encoding/hex"
	"encoding/json"

	"github.com/bcbchain/bcbchain/hyperledger/burrow/acm"
)

type DumpState struct {
	bytes.Buffer
}

func (dw *DumpState) UpdateAccount(updatedAccount *acm.Account) error {
	dw.WriteString("UpdateAccount\n")
	bs, err := json.Marshal(updatedAccount)
	if err != nil {
		return err
	}
	dw.Write(bs)
	dw.WriteByte('\n')
	return nil
}

func (dw *DumpState) RemoveAccount(address types.Address) error {
	dw.WriteString("RemoveAccount\n")
	dw.WriteString(address)
	dw.WriteByte('\n')
	return nil
}

func (dw *DumpState) SetStorage(address types.Address, key, value []byte) error {
	dw.WriteString("SetStorage\n")
	dw.WriteString(address)
	dw.WriteByte('/')
	dw.WriteString(hex.EncodeToString(key[:]))
	dw.WriteByte('/')
	dw.WriteString(hex.EncodeToString(value[:]))
	dw.WriteByte('\n')
	return nil
}
