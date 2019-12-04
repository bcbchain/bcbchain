package tx

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"

	"blockchain/abciapp_v1.0/tx/tx"
	atm "blockchain/algorithm"
	"blockchain/smcsdk/sdk/rlp"
)

//定义合约方法排序
var chainId string

var methodIdToItems = map[uint32]interface{}{}

//ConvertPrototype2ID 根据函数原型计算MethodID
func ConvertPrototype2ID(prototype string) uint32 {
	var id uint32
	bytesBuffer := bytes.NewBuffer(atm.CalcMethodId(prototype))
	binary.Read(bytesBuffer, binary.BigEndian, &id)
	return id
}

type resTx struct {
	Code           string   `json:"code,omitempty"`
	FromAddr       string   `json:"fromAddr,omitempty"`
	Nonce          string   `json:"nonce,omitempty"`
	GasLimit       string   `json:"gasLimit,omitempty"`
	Note           string   `json:"note,omitempty"`
	ToContractAddr string   `json:"toContractAddr,omitempty"`
	MethodId       string   `json:"methodId,omitempty"`
	Items          []string `json:"items,omitempty"`
}

//解包 验签 返回交易所有数据
func UnpackAndParseTx(strTx string) string {

	var transaction tx.Transaction
	fromAddr, _, err := transaction.TxParse(chainId, strTx)
	if err != nil {
		errInfo := string("{\"code\":-2, \"message\":\"Transaction.TxParse failed(") + err.Error() + ")\",\"data\":\"\"}"
		return errInfo
	}

	var methodInfo tx.MethodInfo
	err = rlp.DecodeBytes(transaction.Data, &methodInfo)
	if err != nil {
		errInfo := string("{\"code\":-2, \"message\":\"rlp.DecodeBytes failed(") + err.Error() + ")\",\"data\":\"\"}"
		return errInfo
	}

	var methodID string
	var buf = make([]byte, 4)
	binary.BigEndian.PutUint32(buf, methodInfo.MethodID)
	methodID = hex.EncodeToString(buf)

	resultItems, err := callFunc(methodInfo.MethodID, methodInfo.ParamData)

	if err != nil {
		return string("{\"code\":-2, \"message\":\"") + err.Error() + "\",\"data\":\"\"}"
	}

	var items []string
	items, _ = resultItems[0].Interface().([]string)

	// items以','拼接为string
	// itemArry := strings.Join(items, `","`)
	tx := resTx{
		"0",
		fromAddr,
		strconv.FormatUint(transaction.Nonce, 10),
		strconv.FormatUint(transaction.GasLimit, 10),
		transaction.Note,
		transaction.To,
		methodID,
		items,
	}

	res, _ := json.Marshal(&tx)
	return string(res)
}

func ByteSliceToInt64(b []byte) int64 {
	buf := bytes.NewBuffer(b)
	var v int64
	binary.Read(buf, binary.BigEndian, &v)
	return v
}

func callFunc(id uint32, params ...interface{}) (result []reflect.Value, err error) {

	items, ok := methodIdToItems[id]
	if !ok {
		err = errors.New("The specified method is unsupported")
		return
	}

	f := reflect.ValueOf(items)
	if f.IsNil() {
		err = errors.New("The specified method is unsupported")
		return
	}
	if len(params) != f.Type().NumIn() {
		err = errors.New("Invalid number of params passed.")
		return
	}

	in := make([]reflect.Value, len(params))
	for k, param := range params {
		in[k] = reflect.ValueOf(param)
	}

	result = f.Call(in)
	return
}
