package query

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/bcbchain/bcbchain/burrow"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bcbchain/hyperledger/burrow/abi"
	crypto2 "github.com/bcbchain/bcbchain/hyperledger/burrow/crypto"
	"github.com/bcbchain/bclib/algorithm"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	types3 "github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	log2 "github.com/bcbchain/bclib/tendermint/tmlibs/log"
	tx2 "github.com/bcbchain/bclib/tx/v2"
	types2 "github.com/bcbchain/bclib/types"
	"github.com/bcbchain/bclib/wal"
	"github.com/bcbchain/sdk/sdk/std"
)

func BvmViewKey(key string, log log2.Logger) (resQuery types.ResponseQuery) {
	keys := strings.Split(key, "/")
	if len(keys) != 5 {
		return
	}
	contractAddr := keys[3]
	methods := keys[4]

	var methodName string
	params := make([]string, 0)
	if strings.Contains(methods, "(") && strings.HasSuffix(methods, ")") {
		methodSlice := strings.Split(methods, "(")
		methodName = methodSlice[0]
		params = strings.Split(methodSlice[1][:len(methodSlice[1])-1], ",")
	} else {
		methodName = methods
	}

	contract := new(std.BvmContract)
	res, _ := statedbhelper.GetFromDB("/bvm/contract/" + contractAddr)
	if err := json.Unmarshal(res, &contract); err != nil {
		return
	}

	Abi, err := abi.JSON(strings.NewReader(contract.BvmAbi))
	if err != nil {
		return
	}

	if !Abi.Methods[methodName].Const {
		return
	}

	Array := make([]interface{}, 0)
	for i := 0; i < len(params); i++ {
		Array = append(Array, params[i])
	}

	BinParams, err := PackParams(Abi, methodName, Array...)
	if err != nil {
		return
	}

	resQuery, err = commitTX(contractAddr, BinParams, log)
	if err != nil {
		return
	}
	resQuery.Key = []byte(key)

	return
}

// PrepareParam - prepare param for BVM exec
func PrepareMessages(ContractAddr, TokenAddr crypto.Address, TransMethodID uint32, TransParams, BVMParams, BVMAbi []byte, IsCreateCall bool) []types2.Message {
	Messages := make([]types2.Message, 0)
	Message1 := new(types2.Message)

	Message1.Contract = ContractAddr
	Message1.MethodID = 0xFFFFFFFF
	Message1.Items = tx2.WrapInvokeParams(BVMParams)
	Messages = append(Messages, *Message1)

	return Messages
}

func commitTX(contractAddr string, BinParams []byte, log log2.Logger) (resQuery types.ResponseQuery, err error) {

	Messages := PrepareMessages(contractAddr, "", 0, nil, BinParams, nil, false)

	privateKey := crypto.GenPrivKeyEd25519FromSecret([]byte("0"))
	acct := &wal.Account{
		PrivateKey: privateKey,
	}

	type account struct {
		Nonce uint64 `json:"nonce"`
	}

	a := new(account)
	var nonce uint64
	senderAddr := acct.Address(statedbhelper.GetChainID())
	nonceValue, _ := statedbhelper.GetFromDB(std.KeyOfAccountNonce(senderAddr))

	if len(nonceValue) == 0 {
		nonce = 1
	} else {
		err = json.Unmarshal(nonceValue, a)
		if err != nil {
			return
		}

		nonce = a.Nonce + 1
	}

	var header types3.Header
	var transaction types2.Transaction
	bu := burrow.GetInstance(log)
	transaction.Nonce = nonce
	transaction.Messages = Messages
	transaction.Note = ""
	transaction.GasLimit = 100000
	res := bu.InvokeTxEx(header, nil, 0, 0, senderAddr, transaction, acct.PubKey().Bytes())

	return types.ResponseQuery{
		Code:  types2.CodeBVMQueryOK,
		Value: []byte(res.Data),
		Log:   res.Log,
	}, err

}

func PackParams(abi2 abi.ABI, method string, param ...interface{}) ([]byte, error) {

	var length int
	newParam := make([]interface{}, 0)
	// 合约部署参数构造
	if method == "" {
		length = len(abi2.Constructor.Inputs)
		for i := 0; i < length; i++ {
			paramType := abi2.Constructor.Inputs[i].Type.String()
			newParam = append(newParam, DetermineType(paramType, param[i]))
		}
	} else {
		// 合约调用参数构造
		length = len(abi2.Methods[method].Inputs)
		for i := 0; i < length; i++ {
			if len(param) == 0 {
				return nil, nil
			}
			paramType := abi2.Methods[method].Inputs[i].Type.String()
			newParam = append(newParam, DetermineType(paramType, param[i]))
		}
	}

	paramBin, err := abi2.Pack(method, newParam...)
	if err != nil {
		return nil, err
	}

	return paramBin, nil
}

func DetermineType(paramType string, param interface{}) interface{} {

	if strings.HasPrefix(paramType, "bytes") {
		paramType = "bytes"
	}

	if strings.HasPrefix(paramType, "int") {
		if !strings.HasSuffix(paramType, "]") {
			paramType = "int"
		} else if strings.Contains(paramType, "][") {
			paramType = "int[][]"
		} else {
			paramType = "int[]"
		}
	}

	if strings.HasPrefix(paramType, "uint") {
		if !strings.HasSuffix(paramType, "]") {
			paramType = "uint"
		} else if strings.Contains(paramType, "][") {
			paramType = "uint[][]"
		} else {
			paramType = "uint[]"
		}
	}

	switch paramType {

	case "bool":
		all, err := strconv.ParseBool(param.(string))
		if err != nil {
			return err
		} else {
			return all
		}

	case "int", "int8", "int32", "int64", "int256":
		i, _ := strconv.Atoi(param.(string))
		return int64(i)

	case "uint", "uint8", "uint32", "uint64", "uint256":
		i, _ := strconv.Atoi(param.(string))
		return uint64(i)

	case "int[]", "int8[]", "int32[]", "int64[]", "int256[]":
		Islice := make([]int64, 0)
		slices := param.(string)
		str := strings.Split(slices[1:len(slices)-1], ",")
		for _, v := range str {
			i, _ := strconv.Atoi(v)
			Islice = append(Islice, int64(i))
		}
		return Islice

	case "uint[]", "uint8[]", "uint32[]", "uint64[]", "uint256[]":
		Uslice := make([]uint64, 0)
		slices := param.(string)
		str := strings.Split(slices[1:len(slices)-1], ",")
		for _, v := range str {
			i, _ := strconv.Atoi(v)
			Uslice = append(Uslice, uint64(i))
		}
		return Uslice

	case "int[][]":
		return GetBetweenStrInt(param.(string))

	case "uint[][]":
		return GetBetweenStrUint(param.(string))

	case "string":
		return param.(string)

	case "bytes":
		return []byte(param.(string))

	case "byte[]":
		return []byte(param.(string))

	case "byte[][]":
		return GetBetweenStrByte(param.(string))

	case "address":
		addr := param.(string)
		err := algorithm.CheckAddress(crypto.GetChainId(), addr)
		if err != nil {
			fmt.Println("Invalid address")
			return nil
		}
		address := crypto2.ToBVM(addr).Bytes()
		return abi.BytesToAddress(address)

	case "address[]":
		AddrSlice := make([]abi.Address, 0)
		slices := param.(string)
		str := strings.Split(slices[1:len(slices)-1], ",")
		for _, v := range str {
			err := algorithm.CheckAddress(crypto.GetChainId(), v)
			if err != nil {
				fmt.Println("Invalid address")
				return nil
			}
			address := crypto2.ToBVM(v).Bytes()
			AddrSlice = append(AddrSlice, abi.BytesToAddress(address))
		}
		return AddrSlice

	case "address[][]":
		return GetBetweenStrAddr(param.(string))

	default:
		fmt.Println("The input parameter is invalid, please check！")
		return nil
	}
}

func GetBetweenStrUint(str string) (doubleSlice [][]uint64) {

	doubleSliceStr := make([][]string, 0)
	reg := regexp.MustCompile(`\[(.*?)\]`)
	if reg != nil {
		doubleSliceStr = reg.FindAllStringSubmatch(str, -1)
	}

	for i := 0; i < len(doubleSliceStr); i++ {
		newSlice := make([]uint64, 0)
		newStr := strings.Split(doubleSliceStr[i][1], ",")
		for _, v := range newStr {
			newUint, _ := strconv.Atoi(v)
			newSlice = append(newSlice, uint64(newUint))
		}
		doubleSlice = append(doubleSlice, newSlice)
	}

	return
}

func GetBetweenStrInt(str string) (doubleSlice [][]int64) {

	doubleSliceStr := make([][]string, 0)
	reg := regexp.MustCompile(`\[(.*?)\]`)
	if reg != nil {
		doubleSliceStr = reg.FindAllStringSubmatch(str, -1)
	}

	for i := 0; i < len(doubleSliceStr); i++ {
		newSlice := make([]int64, 0)
		newStr := strings.Split(doubleSliceStr[i][1], ",")
		for _, v := range newStr {
			newUint, _ := strconv.Atoi(v)
			newSlice = append(newSlice, int64(newUint))
		}
		doubleSlice = append(doubleSlice, newSlice)
	}

	return
}

func GetBetweenStrByte(str string) (doubleSlice [][]byte) {

	doubleSliceStr := make([][]string, 0)
	reg := regexp.MustCompile(`\[(.*?)\]`)
	if reg != nil {
		doubleSliceStr = reg.FindAllStringSubmatch(str, -1)
	}

	for i := 0; i < len(doubleSliceStr); i++ {
		newStr := strings.Split(doubleSliceStr[i][1], ",")
		for _, v := range newStr {
			doubleSlice = append(doubleSlice, []byte(v))
		}
	}

	return
}

func GetBetweenStrAddr(str string) (doubleSlice [][]abi.Address) {

	doubleSliceStr := make([][]string, 0)
	reg := regexp.MustCompile(`\[(.*?)\]`)
	if reg != nil {
		doubleSliceStr = reg.FindAllStringSubmatch(str, -1)
	}

	for i := 0; i < len(doubleSliceStr); i++ {
		newSlice := make([]abi.Address, 0)
		newStr := strings.Split(doubleSliceStr[i][1], ",")
		for _, v := range newStr {
			err := algorithm.CheckAddress(crypto.GetChainId(), v)
			if err != nil {
				fmt.Println("Invalid address")
				return nil
			}
			address := crypto2.ToBVM(v).Bytes()
			newSlice = append(newSlice, abi.BytesToAddress(address))
		}
		doubleSlice = append(doubleSlice, newSlice)
	}

	return
}
