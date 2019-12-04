package receipt

import (
	"blockchain/common/statedbhelper"
	"blockchain/smcsdk/sdk/crypto/sha3"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdkimpl"
	"common/jsoniter"
	"fmt"
	"strconv"
	"strings"

	"github.com/tendermint/tmlibs/common"

	"github.com/hyperledger/burrow/execution/exec"
	"github.com/tendermint/tmlibs/log"
)

// Emit emit receipt object
func Emit(logger log.Logger, receipt interface{}, contractAddress string, idx int) *common.KVPair {
	if receipt == nil {
		return nil
	}

	bz, err := jsoniter.Marshal(receipt)
	if err != nil {
		sdkimpl.Logger.Fatalf("[sdk]Cannot marshal receipt data=%v", receipt)
		sdkimpl.Logger.Flush()
		panic(err)
	}

	name := receiptName(receipt)
	logger.Debug("receipt", "name", name)

	rcpt := std.Receipt{
		Name:         name,
		ContractAddr: contractAddress,
		Bytes:        bz,
		Hash:         nil,
	}
	rcpt.Hash = sha3.Sum256([]byte(rcpt.Name), []byte(rcpt.ContractAddr), bz)
	resBytes, _ := jsoniter.Marshal(rcpt) // nolint unhandled

	//将收据添加到message
	tag := common.KVPair{
		Key:   []byte(fmt.Sprintf("/0/%d/%s", idx, rcpt.Name)),
		Value: resBytes,
	}

	return &tag
}

// receiptName - 收据名，转帐收据命名定死成标准转帐收据，不调用这个方法，其他收据就只有evm事件了，加个evm::，好认
func receiptName(receipt interface{}) string {
	switch r := receipt.(type) {
	case *exec.LogEvent:
		return "evm::log_event"
	case *exec.CallEvent:
		return "evm::" + strings.ToLower(r.CallType.String())
	case *exec.TransferData:
		return "std::transfer"
	case std.Fee:
		return "std::fee"
	default:
		return "evm::unknownType"
	}
}

func Tags2Receipt(logger log.Logger, tags *[]interface{}, fee int64, tokenAddr, contractAddr, sender string) *[]common.KVPair {
	length := len(*tags)
	result := make([]common.KVPair, length+1, length+1)
	for idx, t := range *tags {
		tag := Emit(logger, t, contractAddr, idx+1)
		result[idx+1] = *tag
	}
	tag0 := Emit(logger,
		std.Fee{
			Token: tokenAddr,
			From:  sender,
			Value: fee,
		}, contractAddr, 0)
	result[0] = *tag0

	return &result
}

func GetGasPrice(transID, txID int64) int64 {
	genToken := statedbhelper.GetGenesisToken().Address
	genTokenCurrent := statedbhelper.GetTokenByAddress(transID, txID, genToken)

	gasPriceRatio := statedbhelper.GetGasPriceRatio(transID, txID)
	if gasPriceRatio == "" {
		gasPriceRatio = "1.000"
	}

	gasPriceRatio = strings.Replace(gasPriceRatio, ".", "", -1)
	uGasPriceRatio, err := strconv.ParseUint(gasPriceRatio, 10, 64)
	if err != nil {
		panic(err)
	}
	return genTokenCurrent.GasPrice * int64(uGasPriceRatio) / 1000
}
