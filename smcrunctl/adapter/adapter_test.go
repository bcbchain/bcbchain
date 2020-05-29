package adapter

import (
	"github.com/bcbchain/sdk/sdk/rlp"
	"github.com/bcbchain/bclib/types"
	"fmt"
	"testing"

	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
)

func TestAdapter_InvokeTx(t *testing.T) {
	adapter := GetInstance()

	// todo 构造Sender
	sender := ""
	pubkey := []byte("")
	// todo 构造transid txid
	transid := int64(1)
	txid := int64(1)
	//todo 构造Tx
	message := make([]types.Message, 0)
	message = append(message, *NewMessage())
	tx := NewTx(uint64(1), int64(100000), "", message)
	response := adapter.InvokeTx(transid, txid, sender, *tx, pubkey)
	fmt.Println(response)
}

func NewTx(nonce uint64, gaslimit int64, note string, message []types.Message) *types.Transaction {

	tx := types.Transaction{
		Nonce:    nonce,
		GasLimit: gaslimit,
		Note:     note,
		Messages: message,
	}
	return &tx
}

func NewMessage() *types.Message {
	message := types.Message{}

	message.Contract = "localWkNWzXyqMmumfxfXva2QV1qKa3aroVyu" //contract address, myplayerbook v2.0
	message.MethodID = 0xe463fdb2

	message.Items = make([]common.HexBytes, 0)
	name := "jacky"

	b, _ := rlp.EncodeToBytes(name)
	message.Items = append(message.Items, b)

	return &message
}
