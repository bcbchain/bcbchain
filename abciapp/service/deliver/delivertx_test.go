package deliver

import (
	"container/list"
	"fmt"
	"github.com/bcbchain/bclib/algorithm"
	tx2 "github.com/bcbchain/bclib/tx/v2"
	"github.com/bcbchain/bclib/types"
	"github.com/bcbchain/sdk/sdk/bn"
	"github.com/bcbchain/sdk/sdk/jsoniter"
	"github.com/bcbchain/sdk/sdk/std"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/bcbchain/bclib/tendermint/tmlibs/common"

	types2 "github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
)

func TestAppDeliver_DeliverTx(t *testing.T) {
	app := createAppDeliver()
	//req := types2.RequestBeginBlock{}
	//app.BeginBlock(req)

	tx := txWrapper()
	app.deliverBCTx([]byte(tx))
}

func TestAppDeliver_emitFeeReceipts(t *testing.T) {
	app := createAppDeliver()
	gasLimit := int64(50000)

	response := types.Response{
		Code:     200,
		Log:      "DeliverTx success",
		Data:     "",
		Info:     "",
		GasLimit: gasLimit,
		GasUsed:  20000,
		Fee:      20000 * 2500,
		Tags:     fakeTags(),
		TxHash:   nil,
		Height:   0,
	}
	fmt.Println(response)
	tags, _ := app.emitFeeReceipts(types.Transaction{}, nil, true)
	//verify
	if len(tags) != num+q {
		t.Error("number of total fee receipt is wrong", "got: ", len(tags), "exp: ", num+q)
	}
	for _, tag := range tags {
		rcpt := types.Receipt{}
		jsoniter.Unmarshal(tag.Value, &rcpt)
		fee := std.Fee{}
		jsoniter.Unmarshal(rcpt.ReceiptBytes, &fee)

		fmt.Println(rcpt)
		fmt.Println(fee)
	}

	fmt.Println("test emitFeeReceipts(false)")
	//failure
	tags, _ = app.emitFeeReceipts(types.Transaction{}, nil, false)
	//verify
	if len(tags) != q {
		t.Error("number of total fee receipt is wrong", "got: ", len(tags), "exp: ", q)
	}
	for _, tag := range tags {
		rcpt := types.Receipt{}
		jsoniter.Unmarshal(tag.Value, &rcpt)
		fee := std.Fee{}
		jsoniter.Unmarshal(rcpt.ReceiptBytes, &fee)

		fmt.Println(rcpt)
		fmt.Println(fee)
	}
}

const (
	num = 14
	q   = 3
)

func fakeTags() []common.KVPair {
	tags := make([]common.KVPair, 0)
	for i := 0; i < num; i++ {
		fee := std.Fee{
			Token: "0123456789",
			From:  "addddddddddddddddddddddddd" + strconv.Itoa(i%q),
			Value: 10000,
		}
		bf, _ := jsoniter.Marshal(fee)
		receipt := types.Receipt{
			Name:            "std.fee",
			ContractAddress: "",
			ReceiptBytes:    bf,
			ReceiptHash:     nil,
		}

		b, _ := jsoniter.Marshal(receipt)

		kv := common.KVPair{
			Key:   []byte(fmt.Sprintf("/%d/%s", len(tags), "std.fee")),
			Value: b,
		}
		tags = append(tags, kv)
	}
	return tags
}

func createAppDeliver() AppDeliver {
	app := AppDeliver{
		logger:      nil,
		txID:        0,
		blockHash:   nil,
		blockHeader: types2.Header{},
		appState:    nil,
		hashList:    list.New().Init(),
		chainID:     "bcb",
		sponser:     "",
		rewarder:    "",
		udValidator: false,
		validators:  nil,
		fee:         0,
		rewards:     nil,
	}
	app.logger = createLogger()
	return app
}

func createLogger() log.Logger {
	home := os.Getenv("HOME")
	fmt.Println(home)
	logger := log.NewTMLogger(filepath.Join(home, "log"), "bcchain")
	logger.AllowLevel("debug")
	logger.SetOutputAsync(false)
	logger.SetOutputToFile(false)
	logger.SetOutputToScreen(true)
	//	logger.SetOutputFileSize(common.GlobalConfig.Log_size)
	return logger
}

func txWrapper() string {
	tx2.Init("bcb")

	methodID1 := algorithm.BytesToUint32(algorithm.CalcMethodId("Transfer(types.Address,bn.Number)"))
	toContract1 := "bcbMWedWqzzW8jkt5tntTomQQEN7fSwWFhw6"

	toAccount := "bcbCpeczqoSoxLxx1x3UyuKsaS4J8yamzWzz"
	value := bn.N(1000000000)
	itemInfo1 := tx2.WrapInvokeParams(toAccount, value)
	message1 := types.Message{
		Contract: toContract1,
		MethodID: methodID1,
		Items:    itemInfo1,
	}
	nonce := uint64(1)
	gasLimit := int64(500)
	note := "Example for cascade invoke smart contract."
	txPayloadBytesRlp := tx2.WrapPayload(nonce, gasLimit, note, message1)
	privKeyStr := "0x4a2c14697282e658b3ed7dd5324de1a102d216d6fa50d5937ffe89f35cbc12aa68eb9a09813bdf7c0869bf34a244cc545711509fe70f978d121afd3a4ae610e6"
	finalTx := tx2.WrapTx(txPayloadBytesRlp, privKeyStr)

	return finalTx
}
