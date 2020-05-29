package adapter

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/bcbchain/bcbchain/abciapp/common"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	rpcclient "github.com/bcbchain/bclib/rpc/lib/client"
	"github.com/bcbchain/bclib/socket"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	tx1 "github.com/bcbchain/bclib/tx/v1"
	tx2 "github.com/bcbchain/bclib/tx/v2"
	tx3 "github.com/bcbchain/bclib/tx/v3"
	"github.com/bcbchain/bclib/types"
	"github.com/bcbchain/sdk/sdk/bn"
	"github.com/bcbchain/sdk/sdk/jsoniter"
	"github.com/bcbchain/sdk/sdk/rlp"
	"github.com/bcbchain/sdk/sdk/std"
	types2 "github.com/bcbchain/sdk/sdk/types"
	"github.com/bcbchain/tendermint/rpc/core/types"
)

//Routes defines RPC functions route map
//var Routes = map[string]*rpcserver.RPCFunc{
//	"get":   rpcserver.NewRPCFunc(SdbGet, "transID,txID,key"),
//	"set":   rpcserver.NewRPCFunc(SdbSet, "transID,txID,data"),
//	"build": rpcserver.NewRPCFunc(SdbBuild, "transID,txID,contractMeta"),
//	"block": rpcserver.NewRPCFunc(GetBlock, "height"),
//}

const (
	transferMethodIDV1 = "af0228bc"
	transferMethodIDV2 = "44d8ca60"
)

var Routes = map[string]socket.CallBackFunc{
	"get":   SdbGet,
	"set":   SdbSet,
	"build": SdbBuild,
	"block": GetBlock,
}

var logger log.Logger

func SetLogger(l log.Logger) {
	logger = l
}

//SdbGet calls sdb get function
func SdbGet(req map[string]interface{}) (result interface{}, err error) {

	transID := int64(req["transID"].(float64))
	txID := int64(req["txID"].(float64))
	key := req["key"].(string)

	prefix := "$sdk"
	if strings.HasPrefix(key, prefix) {
		return processSDKGet(key, prefix)
	}

	rBytes, err := Get(transID, txID, key)
	return string(rBytes), err
}

func processSDKGet(key, prefix string) (interface{}, error) {
	subKey := strings.TrimPrefix(key, prefix)
	subList := strings.Split(subKey, "$")
	result := new(std.GetResult)
	if len(subList) < 2 {
		result.Code = types2.ErrInvalidParameter
		result.Msg = "invalid key:" + key
		res, _ := jsoniter.Marshal(result)
		return string(res), nil
	}
	switch subList[1] {
	case "getTx": // "$sdk$getTx$txHash"
		if len(subList) != 3 {
			result.Code = types2.ErrInvalidParameter
			result.Msg = "invalid key:" + key
			res, _ := jsoniter.Marshal(result)
			return string(res), nil
		}

		txResult, err := Tx(subList[2])
		if err != nil {
			result.Code = types2.ErrInvalidParameter
			result.Msg = err.Error()
			res, _ := jsoniter.Marshal(result)
			return string(res), nil
		}

		if len(txResult) == 0 {
			result.Code = types2.ErrInvalidParameter
			result.Msg = fmt.Sprintf("key=%s cannot get data.", key)
			res, _ := jsoniter.Marshal(result)
			return string(res), nil
		}
		result.Code = types.CodeOK
		result.Data = []byte(txResult)
		res, _ := jsoniter.Marshal(result)
		return string(res), nil

	default:
		result.Code = types2.ErrInvalidParameter
		result.Msg = "invalid key:" + key
		res, _ := jsoniter.Marshal(result)
		return string(res), nil
	}
}

//SdbSet calls sdb set function
func SdbSet(req map[string]interface{}) (result interface{}, err error) {

	transID := int64(req["transID"].(float64))
	txID := int64(req["txID"].(float64))
	mData := req["data"].(map[string]interface{})
	data := make(map[string][]byte)
	for k, v := range mData {
		data[k] = []byte(v.(string))
	}

	return Set(transID, txID, data)
}

//SdbBuild calls sdb build function
func SdbBuild(req map[string]interface{}) (result interface{}, err error) {

	transID := int64(req["transID"].(float64))
	txID := int64(req["txID"].(float64))
	resBytes := []byte(req["contractMeta"].(string))
	var contractMeta std.ContractMeta
	err = jsoniter.Unmarshal(resBytes, &contractMeta)
	if err != nil {
		return
	}

	buildResult, err := Build(transID, txID, contractMeta)
	if err != nil {
		return
	} else {
		resBytes, _ = jsoniter.Marshal(buildResult)
		return string(resBytes), nil
	}
}

func GetBlock(req map[string]interface{}) (result interface{}, err error) {

	heightFloat, ok := req["height"].(float64)
	if !ok || heightFloat < 0 {
		err = errors.New("invalid height")
		return
	}
	logger.Debug("Adapter RPC", "get block", heightFloat)

	var height = int64(heightFloat)
	if heightFloat == 0 {
		appState := statedbhelper.GetWorldAppState(0, 0)
		height = appState.BlockHeight
	}

	if common.TmCoreURL == "" {
		err = errors.New("can not get tendermint url")
		return nil, err
	}
	logger.Debug("Adapter RPC", "query RPC URL", common.TmCoreURL)

	res := new(core_types.ResultBlock)
	rpc := rpcclient.NewJSONRPCClientEx(common.TmCoreURL, "", true)
	_, err = rpc.Call("block", map[string]interface{}{"height": height}, res)
	if err != nil {
		common.TmCoreURL = strings.Replace(common.TmCoreURL, "http", "https", 1)
		res = new(core_types.ResultBlock)
		rpc = rpcclient.NewJSONRPCClientEx(common.TmCoreURL, "", true)
		_, err = rpc.Call("block", map[string]interface{}{"height": height}, res)
		if err != nil {
			return nil, err
		}
	}

	b := std.Block{
		ChainID:         res.BlockMeta.Header.ChainID,
		BlockHash:       types2.Hash(res.BlockMeta.BlockID.Hash),
		Height:          res.BlockMeta.Header.Height,
		Time:            res.BlockMeta.Header.Time.Unix(),
		NumTxs:          int32(res.BlockMeta.Header.NumTxs),
		DataHash:        types2.Hash(res.BlockMeta.Header.DataHash),
		ProposerAddress: res.BlockMeta.Header.ProposerAddress,
		RewardAddress:   res.BlockMeta.Header.RewardAddress,
		RandomNumber:    types2.HexBytes(res.BlockMeta.Header.RandomOfBlock),
		LastBlockHash:   types2.Hash(res.BlockMeta.BlockID.Hash),
		LastCommitHash:  types2.Hash(res.BlockMeta.Header.LastCommitHash),
		LastAppHash:     types2.Hash(res.BlockMeta.Header.LastAppHash),
		LastFee:         int64(res.BlockMeta.Header.LastFee),
		Version:         *res.BlockMeta.Header.Version,
	}

	resultByte, err := jsoniter.Marshal(b)
	if err != nil {
		return nil, err
	}
	result = string(resultByte)
	return
}

func Tx(txHash string) (result string, err error) {
	if len(txHash) > 2 && txHash[:2] == "0x" {
		txHash = txHash[2:]
	}

	resultTx := new(core_types.ResultTx)
	if err = callAndParse("tx", map[string]interface{}{"hash": txHash}, resultTx); err != nil {
		return
	}

	resultBlock := new(core_types.ResultBlock)
	if err = callAndParse("block", map[string]interface{}{"height": resultTx.Height}, resultBlock); err != nil {
		return
	}

	blkResults := new(core_types.ResultBlockResults)
	if err = callAndParse("block_results", map[string]interface{}{"height": resultTx.Height}, blkResults); err != nil {
		return
	}

	var txStr string
	for k, v := range blkResults.Results.DeliverTx {
		hash := hex.EncodeToString(v.TxHash)
		if hash[:2] == "0x" {
			txHash = txHash[2:]
		}
		if strings.ToLower(txHash) == strings.ToLower(hash) {
			txStr = string(resultBlock.Block.Txs[k])
			break
		}
	}

	_, _, fromAddr, note, messages, err0 := parseTx(resultBlock.Block.ChainID, txStr,
		resultBlock.Block.Height, resultBlock.Block.ChainVersion)
	if err0 != nil {
		return "", err0
	}

	r := std.TxResult{
		TxHash:      "0x" + txHash,
		Code:        resultTx.DeliverResult.Code,
		Log:         resultTx.DeliverResult.Log,
		BlockHeight: resultTx.DeliverResult.Height,
		From:        fromAddr,
		Note:        note,
		Message:     messages,
	}
	resultByte, err := jsoniter.Marshal(r)
	if err != nil {
		return "", err
	}
	result = string(resultByte)

	return
}

func parseTx(chainID, txStr string, height int64, chainVersion *int64) (nonce, gasLimit uint64, fromAddr, note string, messages []std.Message, err error) {

	messages = make([]std.Message, 0)

	splitTx := strings.Split(txStr, ".")
	if splitTx[1] == "v1" {
		var txv1 tx1.Transaction
		fromAddr, _, err = txv1.TxParse(chainID, txStr)
		if err != nil {
			return
		}
		nonce = txv1.Nonce
		note = txv1.Note
		gasLimit = txv1.GasLimit

		var msg std.Message
		msg, err = messageV1(txv1, height, chainVersion)
		if err != nil {
			return
		}
		messages = append(messages, msg)
	} else if splitTx[1] == "v2" {
		var txv2 types.Transaction
		var pubKey crypto.PubKeyEd25519
		txv2, pubKey, err = tx2.TxParse(txStr)
		if err != nil {
			return
		}
		fromAddr = pubKey.Address(chainID)
		nonce = txv2.Nonce
		note = txv2.Note
		gasLimit = uint64(txv2.GasLimit)

		var msg std.Message
		for i := 0; i < len(txv2.Messages); i++ {
			msg, err = message(txv2.Messages[i], height, chainVersion)
			if err != nil {
				return
			}
			messages = append(messages, msg)
		}
	} else if splitTx[1] == "v3" {
		var txv3 types.Transaction // v2 and v3 Transaction same
		var pubKey crypto.PubKeyEd25519

		txv3, pubKey, err = tx3.TxParse(txStr)
		if err != nil {
			return
		}
		fromAddr = pubKey.Address(chainID)
		nonce = txv3.Nonce
		note = txv3.Note
		gasLimit = uint64(txv3.GasLimit)

		var msg std.Message
		for i := 0; i < len(txv3.Messages); i++ {
			msg, err = message(txv3.Messages[i], height, chainVersion)
			if err != nil {
				return
			}
			messages = append(messages, msg)
		}
	} else {
		err = errors.New("unsupported tx=" + txStr)
		return
	}

	return
}

func messageV1(tx tx1.Transaction, height int64, chainVersion *int64) (msg std.Message, err error) {

	var methodInfo tx1.MethodInfo
	if err = rlp.DecodeBytes(tx.Data, &methodInfo); err != nil {
		return
	}
	methodID := fmt.Sprintf("%x", methodInfo.MethodID)

	if _, msg.Method, err = contractNameAndMethod(tx.To, methodID, height, chainVersion); err != nil {
		return
	}

	if methodID == transferMethodIDV1 {
		var itemsBytes = make([][]byte, 0)
		if err = rlp.DecodeBytes(methodInfo.ParamData, &itemsBytes); err != nil {
			return
		}
		msg.To = string(itemsBytes[0])
		msg.Value = new(big.Int).SetBytes(itemsBytes[1][:]).String()
	}

	return
}

func message(message types.Message, height int64, chainVersion *int64) (msg std.Message, err error) {

	methodID := fmt.Sprintf("%x", message.MethodID)

	msg.SmcAddress = message.Contract
	if _, msg.Method, err = contractNameAndMethod(message.Contract, methodID, height, chainVersion); err != nil {
		return
	}

	if methodID == transferMethodIDV2 {
		if len(message.Items) != 2 {
			return msg, errors.New("items count error")
		}

		var to types.Address
		if err = rlp.DecodeBytes(message.Items[0], &to); err != nil {
			return
		}

		var value bn.Number
		if err = rlp.DecodeBytes(message.Items[1], &value); err != nil {
			return
		}
		msg.To = to
		msg.Value = value.String()
	}

	return
}

func contractNameAndMethod(contractAddress types.Address, methodID string, height int64, chainVersion *int64) (contractName string, method string, err error) {

	contract := new(std.Contract)

	resultQuery := new(core_types.ResultABCIQuery)
	if err = callAndParse("abci_query", map[string]interface{}{"path": std.KeyOfContract(contractAddress)}, resultQuery); err != nil {
		return
	}

	if err = jsoniter.Unmarshal(resultQuery.Response.Value, contract); err != nil {
		return
	}

	if chainVersion != nil && contract.LoseHeight != 0 && contract.LoseHeight < height {
		conVer := new(std.ContractVersionList)
		resultQuery1 := new(core_types.ResultABCIQuery)

		if err = callAndParse("abci_query",
			map[string]interface{}{"path": std.KeyOfContractsWithName(contract.OrgID, contract.Name)},
			resultQuery1); err == nil {

			if err = jsoniter.Unmarshal(resultQuery1.Response.Value, conVer); err != nil {
				return
			}

			for index, eh := range conVer.EffectHeights {
				if eh <= height {
					tmp := new(std.Contract)
					resultQuery2 := new(core_types.ResultABCIQuery)
					if err = callAndParse("abci_query",
						map[string]interface{}{"path": std.KeyOfContract(conVer.ContractAddrList[index])},
						resultQuery2); err == nil {

						if err = jsoniter.Unmarshal(resultQuery2.Response.Value, tmp); err != nil {
							return
						}

						if tmp.LoseHeight == 0 || (tmp.LoseHeight != 0 && tmp.LoseHeight > height) {
							contract = tmp
							break
						}
					} else {
						return
					}
				}
			}
		} else {
			return
		}
	}

	for _, methodItem := range contract.Methods {
		if methodItem.MethodID == methodID {
			method = methodItem.ProtoType
			break
		}
	}
	contractName = contract.Name
	return
}

func callAndParse(methodName string, params map[string]interface{}, result interface{}) (err error) {
	rpc := rpcclient.NewJSONRPCClientEx(common.TmCoreURL, "", true)
	_, err = rpc.Call(methodName, params, result)
	if err != nil {
		if strings.HasPrefix(common.TmCoreURL, "https") {
			common.TmCoreURL = strings.Replace(common.TmCoreURL, "https", "http", 1)
		} else {
			common.TmCoreURL = strings.Replace(common.TmCoreURL, "http", "https", 1)
		}
		rpc = rpcclient.NewJSONRPCClientEx(common.TmCoreURL, "", true)
		_, err = rpc.Call(methodName, params, result)
		if err != nil {
			return err
		}
	}
	return nil
}
