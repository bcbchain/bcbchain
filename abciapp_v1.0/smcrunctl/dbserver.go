package smcrunctl

import (
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/statedb"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bclib/socket"
	"fmt"
	tmcommon "github.com/bcbchain/bclib/tendermint/tmlibs/common"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	"time"
)

//Routes routes map
var Routes = map[string]socket.CallBackFunc{
	"setToTxBuffer":    setToTxBuffer,
	"setToBlockBuffer": setToBlockBuffer,
	"get":              get,
}

func StartServer(stateDB *statedb.StateDB, logger log.Logger, v1port int) {
	go func() {
		svr, err := socket.NewServer(fmt.Sprintf("tcp://0.0.0.0:%d", v1port), Routes, 10, logger)
		if err != nil {
			panic(err)
		}

		// start server and wait forever
		err = svr.Start()
		if err != nil {
			tmcommon.Exit(err.Error())
		}
	}()
	time.Sleep(100 * time.Millisecond) // 等待100毫秒确保服务启动

	GetInstance().SetLogger(logger)

	bcErr := GetInstance().InitContractDocker(stateDB)
	if bcErr.ErrorCode != bcerrors.ErrCodeOK {
		panic(bcErr.ErrorDesc)
	}
}

func setToTxBuffer(req map[string]interface{}) (interface{}, error) {
	transID := int64(req["transID"].(float64))

	invokeItems := GetInstance().GetInvokeItems(transID)
	if invokeItems == nil {
		panic("error transID")
	}

	mData := req["data"].(map[string]interface{})
	for key, value := range mData {
		err := invokeItems.Ctx.TxState.Set(key, []byte(value.(string)))
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func setToBlockBuffer(req map[string]interface{}) (interface{}, error) {
	transID := int64(req["transID"].(float64))

	invokeItems := GetInstance().GetInvokeItems(transID)
	if invokeItems == nil {
		panic("error transID")
	}

	mData := req["data"].(map[string]interface{})
	data := make(map[string][]byte)
	for key, value := range mData {
		invokeItems.Ctx.TxState.StateDB.Set(key, []byte(value.(string)))
		data[key] = []byte(value.(string))
	}

	if transID != 0 {
		statedbhelper.CommitTx2V1(transID, data)
	}

	return true, nil
}

func get(req map[string]interface{}) (interface{}, error) {
	transID := int64(req["transID"].(float64))

	invokeItems := GetInstance().GetInvokeItems(transID)
	if invokeItems == nil {
		panic("error transID")
	}

	key := req["key"].(string)
	value, err := invokeItems.Ctx.TxState.Get(key)
	if err != nil {
		return "", err
	}

	return string(value), nil
}
