package receipt

import (
	"fmt"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bcbchain/hyperledger/burrow/execution/exec"
	"github.com/bcbchain/bclib/jsoniter"
	"github.com/bcbchain/sdk/sdk/crypto/sha3"
	"github.com/bcbchain/sdk/sdk/std"
	"github.com/bcbchain/sdk/sdkimpl"
	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	"strconv"
	"strings"
)

type LogEventParams struct {
	Data map[string]interface{} `json:"data"`
}

// Emit emit receipt object
func Emit(logger log.Logger, receipt interface{}, contractAddress, abiStr string, idx1, idx2 int, transId, txId int64) (*common.KVPair, error) {
	if receipt == nil {
		return nil, nil
	}

	name, receiptObject := receiptName(receipt)

	EventReceipt := new(LogEventParams)
	if name == "bvm::log_event" {
		EventName, err := ConversionEventReceipt(logger, transId, txId, receiptObject, EventReceipt, contractAddress, abiStr)
		if err != nil {
			logger.Debug("bvm", "ConversionEventReceiptErr", err)
			return nil, err
		}

		name += EventName
	}

	Bz := make([]byte, 0)
	if strings.HasPrefix(name, "bvm::log_event") {
		bz, err := jsoniter.Marshal(EventReceipt.Data)
		if err != nil {
			sdkimpl.Logger.Fatalf("[sdk]Cannot marshal receipt data=%v", receipt)
			sdkimpl.Logger.Flush()
			panic(err)
		}
		Bz = append(Bz, bz...)
	} else {
		bz, err := jsoniter.Marshal(receipt)
		if err != nil {
			sdkimpl.Logger.Fatalf("[sdk]Cannot marshal receipt data=%v", receipt)
			sdkimpl.Logger.Flush()
			panic(err)
		}
		Bz = append(Bz, bz...)
	}

	rcpt := std.Receipt{
		Name:         name,
		ContractAddr: contractAddress,
		Bytes:        Bz,
		Hash:         nil,
	}
	rcpt.Hash = sha3.Sum256([]byte(rcpt.Name), []byte(rcpt.ContractAddr), Bz)
	resBytes, _ := jsoniter.Marshal(rcpt) // nolint unhandled

	//将收据添加到message
	tag := common.KVPair{
		Key:   []byte(fmt.Sprintf("/%d/%d/%s", idx1, idx2, rcpt.Name)),
		Value: resBytes,
	}

	return &tag, nil
}

// receiptName - 收据名，转帐收据命名定死成标准转帐收据，不调用这个方法，其他收据就只有bvm事件了，加个bvm::，好认
func receiptName(receipt interface{}) (name string, event *exec.LogEvent) {
	switch r := receipt.(type) {
	case *exec.LogEvent:
		op, _ := receipt.(*exec.LogEvent)
		return "bvm::log_event", op
	case *exec.CallEvent:
		return "bvm::" + strings.ToLower(r.CallType.String()), nil
	case *exec.TransferData:
		return "std::transfer", nil
	case std.Fee:
		return "std::fee", nil
	default:
		return "bvm::unknownType", nil
	}
}

func Tags2Receipt(logger log.Logger, tags *[]interface{}, transId, txId, fee int64, tokenAddr, contractAddr, sender, abiStr string, isCascadeCall bool) (*[]common.KVPair, error) {
	length := len(*tags)
	result := make([]common.KVPair, length+1, length+1)

	tag0, err := Emit(logger,
		std.Fee{
			Token: tokenAddr,
			From:  sender,
			Value: fee,
		}, contractAddr, abiStr, 0, 0, transId, txId)
	if err != nil {
		return nil, err
	}

	result[0] = *tag0

	idx2 := 0
	if isCascadeCall {
		idx1 := 0
		for idx, t := range *tags {
			if idx == 0 {
				idx2 = 1
			} else {
				idx1 = 1
				idx2 = idx - 1
			}

			tag, err := Emit(logger, t, contractAddr, abiStr, idx1, idx2, transId, txId)
			if err != nil {
				return nil, err
			}

			result[idx+1] = *tag
		}
	} else {
		for idx, t := range *tags {
			idx2 = idx + 1
			tag, err := Emit(logger, t, contractAddr, abiStr, 0, idx2, transId, txId)
			if err != nil {
				return nil, err
			}

			result[idx+1] = *tag
		}
	}

	return &result, nil
}

func GetGasPrice(transID, txID int64, isCreate bool) int64 {
	genToken := statedbhelper.GetGenesisToken().Address
	genTokenCurrent := statedbhelper.GetTokenByAddress(transID, txID, genToken)

	if isCreate {
		return genTokenCurrent.GasPrice
	}

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
