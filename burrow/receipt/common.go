package receipt

import (
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/sdk/sdk/jsoniter"
	"github.com/bcbchain/sdk/sdk/std"
	"github.com/bcbchain/bclib/types"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/bcbchain/bcbchain/hyperledger/burrow/abi"
	crypto2 "github.com/bcbchain/bcbchain/hyperledger/burrow/crypto"
	abi2 "github.com/bcbchain/bcbchain/hyperledger/burrow/execution/bvm/abi"
	"github.com/bcbchain/bcbchain/hyperledger/burrow/execution/exec"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

func ConversionEventReceipt(logger log.Logger, transId, txId int64, res *exec.LogEvent, EventReceipt *LogEventParams, contractAddress types.Address, abiStr string) (name string, err error) {

	newAbi := abi.ABI{}
	if abiStr != "" {
		newAbi, err = GetAbiObject(logger, abiStr)
		if err != nil {
			return
		}
	} else {
		result, _ := statedbhelper.Get(transId, txId, "/bvm/contract/"+contractAddress)
		if len(result) == 0 {
			return "", err
		}

		contract := new(std.BvmContract)
		err := json.Unmarshal(result, contract)
		if err != nil {
			panic("state db helper get bvm contract err: " + err.Error())
		}

		newAbi, err = GetAbiObject(logger, contract.BvmAbi)
		if err != nil {
			return "", err
		}
	}

	for i := 0; i < len(newAbi.Events); i++ {
		event, err := newAbi.EventByID(res.Topics[0].Bytes())
		if err != nil {
			return "", err
		}
		name = "::" + event.RawName
		v := make(map[string]interface{})
		err = newAbi.UnpackLogIntoMap(v, event.RawName, *res)

		EventReceipt.Data = make(map[string]interface{})
		k := 0

		for _, iv := range event.Inputs {
			tMap := GetTypeMap(iv.Type.String(), iv.Type)
			if iv.Indexed == true {
				result, err := DetermineType(iv.Type.String(), v[iv.Name], tMap)
				if err != nil {
					return "", err
				}

				EventReceipt.Data[iv.Name] = result
			} else {
				result, err := DetermineType(iv.Type.String(), v[strconv.Itoa(k)], tMap)
				if err != nil {
					return "", err
				}

				EventReceipt.Data[iv.Name] = result
				k++
			}
		}
	}

	return
}

func GetAbiObject(log log.Logger, abiStr string) (Abi abi.ABI, err error) {
	if abiStr == "" {
		log.Debug("bvm", "abiFile is empty, please check")
		return
	}

	abiStr = strings.Replace(abiStr, "\n", "", -1)
	abiStr = strings.Replace(abiStr, "\t", "", -1)
	abiStr = strings.Replace(abiStr, `\`, "", -1)

	Abi, err = abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		return
	}

	return
}

func GetAbiObjectForReceipt(log log.Logger, abiStr string) (Abi abi2.AbiSpec, err error) {
	if abiStr == "" {
		log.Debug("bvm", "abiFile is empty, please check")
		return
	}

	abiStr = strings.Replace(abiStr, "\n", "", -1)
	abiStr = strings.Replace(abiStr, "\t", "", -1)
	abiStr = strings.Replace(abiStr, `\`, "", -1)

	Abi2, err := abi2.ReadAbiSpec([]byte(abiStr))
	if err != nil {
		return
	}

	return *Abi2, nil
}

func DetermineType(paramType string, param interface{}, tMap map[string]string) (interface{}, error) {

	if param == nil {
		return nil, nil
	}

	if ifStruct(tMap) {

		return determineStructType(param, tMap)
	} else {

		return determineType(paramType, param)
	}

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

func GetBetweenStrAddr(addresses [][]abi.Address) (doubleSlice [][]crypto.Address) {

	doubleSlice = make([][]crypto.Address, 0)
	for i := 0; i < len(addresses); i++ {
		newSlice := make([]crypto.Address, 0)
		for _, v := range addresses[i] {
			var address crypto.Address
			if !ifBVMAddrEmpty(v) {
				bvmAddr := crypto2.BVMAddress{}
				copy(bvmAddr[:], v[:])
				address = crypto2.ToAddr(bvmAddr)
			}

			newSlice = append(newSlice, address)
		}
		doubleSlice = append(doubleSlice, newSlice)
	}

	return
}

func GetTypeMap(paramType string, t abi.Type) map[string]string {
	tMap := make(map[string]string, 0)

	if t.Kind == reflect.Struct {
		typeList := GetTypeList(paramType)
		for i, v := range t.TupleRawNames {
			tMap[v] = typeList[i]
		}
	}

	return tMap
}

func GetTypeList(paramType string) []string {
	pt := strings.Replace(paramType, "(", "", -1)
	pt = strings.Replace(pt, ")", "", -1)
	return strings.Split(pt, ",")
}

func ifStruct(tMap map[string]string) bool {

	return len(tMap) != 0
}

func determineStructType(param interface{}, tMap map[string]string) (map[string]interface{}, error) {
	outData, _ := jsoniter.Marshal(param)

	vMap := make(map[string]interface{})
	err := jsoniter.Unmarshal(outData, &vMap)
	if err != nil {
		return nil, err
	}

	for k, v := range vMap {
		vbyte, err := jsoniter.Marshal(v)
		if err != nil {
			return nil, err
		}

		paramType := getParamType(tMap[k])
		switch paramType {
		case "bytes":
			var bin []byte
			err = jsoniter.Unmarshal(vbyte, &bin)
			if err != nil {
				return nil, err
			}

			if bin != nil && !ifByteSliceAllZero(bin) {
				vMap[k] = strings.Replace(string(bin[:]), "\u0000", "", -1)
			} else {
				vMap[k] = nil
			}

		case "bytes[]":

			s, err := jsoniter.Marshal(v)
			if err != nil {
				return nil, err
			}

			bin := make([]interface{}, 0)
			err = jsoniter.Unmarshal(s, &bin)
			if err != nil {
				return nil, err
			}

			vSlice := make([]interface{}, 0)
			for _, v := range bin {

				if v != nil {
					decodeBytes, err := base64.StdEncoding.DecodeString(v.(string))
					if err != nil {
						return nil, err
					}

					if !utf8.ValidString(string(decodeBytes)) {
						return nil, fmt.Errorf("invalid utf-8 data")
					}

					vSlice = append(vSlice, string(decodeBytes))
				} else {
					vSlice = append(vSlice, v)
				}
			}

			vMap[k] = vSlice

		case "bytes[][]":
			s, _ := jsoniter.Marshal(v)
			bin := make([][]interface{}, 0)
			err := jsoniter.Unmarshal(s, &bin)
			if err != nil {
				return nil, err
			}

			for i, v := range bin {
				vSlice := make([]interface{}, 0)
				for _, v2 := range v {
					if v2 != nil {
						decodeBytes, err := base64.StdEncoding.DecodeString(v2.(string))
						if err != nil {
							return nil, err
						}

						if !utf8.ValidString(string(decodeBytes)) {
							return nil, fmt.Errorf("invalid utf-8 data")
						}

						vSlice = append(vSlice, string(decodeBytes))
					} else {
						vSlice = append(vSlice, v2)
					}

				}
				bin[i] = vSlice
			}

			vMap[k] = bin

		case "address":
			addr := new(abi.Address)
			err = jsoniter.Unmarshal(vbyte, addr)
			if err != nil {
				return nil, err
			}

			var address crypto.Address
			if !ifBVMAddrEmpty(*addr) {
				bvmAddr := crypto2.BVMAddress{}
				copy(bvmAddr[:], addr[:])
				address = crypto2.ToAddr(bvmAddr)
			}
			vMap[k] = address

		case "address[]":
			AddrSlice := make([]crypto.Address, 0)
			var slices []abi.Address
			err = jsoniter.Unmarshal(vbyte, &slices)
			if err != nil {
				return nil, err
			}

			for _, v := range slices {

				var address crypto.Address
				if !ifBVMAddrEmpty(v) {
					bvmAddr := crypto2.BVMAddress{}
					copy(bvmAddr[:], v[:])
					address = crypto2.ToAddr(bvmAddr)
				}

				AddrSlice = append(AddrSlice, address)
			}

			vMap[k] = AddrSlice

		case "address[][]":
			var slices [][]abi.Address
			err = jsoniter.Unmarshal(vbyte, &slices)
			if err != nil {
				return nil, err
			}
			vMap[k] = GetBetweenStrAddr(slices)
		}
	}

	return vMap, nil
}

func determineType(paramType string, param interface{}) (interface{}, error) {

	paramType = getParamType(paramType)
	switch paramType {

	case "bytes":
		s, err := jsoniter.Marshal(param)
		if err != nil {
			return nil, err
		}

		var bin []byte
		err = jsoniter.Unmarshal(s, &bin)
		if err != nil {
			return nil, err
		}

		if !ifByteSliceAllZero(bin) {
			return strings.Replace(string(bin[:]), "\u0000", "", -1), nil
		} else {
			return nil, nil
		}

	case "bytes[]":

		s, err := jsoniter.Marshal(param)
		if err != nil {
			return nil, err
		}

		bin := make([]interface{}, 0)
		err = jsoniter.Unmarshal(s, &bin)
		if err != nil {
			return nil, err
		}

		vSlice := make([]interface{}, 0)
		for _, v := range bin {
			if v != nil {
				decodeBytes, err := base64.StdEncoding.DecodeString(v.(string))
				if err != nil {
					return nil, err
				}

				if !utf8.ValidString(string(decodeBytes)) {
					return nil, fmt.Errorf("invalid utf-8 data")
				}

				vSlice = append(vSlice, string(decodeBytes))
			} else {
				vSlice = append(vSlice, v)
			}
		}

		return vSlice, nil

	case "bytes[][]":
		s, err := jsoniter.Marshal(param)
		if err != nil {
			return nil, err
		}
		bin := make([][]interface{}, 0)
		err = jsoniter.Unmarshal(s, &bin)
		if err != nil {
			return nil, err
		}

		for i, v := range bin {
			vSlice := make([]interface{}, 0)
			for _, v2 := range v {
				if v2 != nil {
					decodeBytes, err := base64.StdEncoding.DecodeString(v2.(string))
					if err != nil {
						return nil, err
					}

					if !utf8.ValidString(string(decodeBytes)) {
						return nil, fmt.Errorf("invalid utf-8 data")
					}

					vSlice = append(vSlice, string(decodeBytes))
				} else {
					vSlice = append(vSlice, v2)
				}

			}
			bin[i] = vSlice
		}

		return bin, nil

	case "address":
		addr := param.(abi.Address)
		var address crypto.Address
		if !ifBVMAddrEmpty(addr) {
			bvmAddr := crypto2.BVMAddress{}
			copy(bvmAddr[:], addr[:])
			address = crypto2.ToAddr(bvmAddr)
		}

		return address, nil

	case "address[]":
		s, err := jsoniter.Marshal(param)
		if err != nil {
			return nil, err
		}

		var slices []abi.Address
		err = jsoniter.Unmarshal(s, &slices)
		if err != nil {
			return nil, err
		}

		AddrSlice := make([]crypto.Address, 0)
		for _, v := range slices {
			var address crypto.Address
			if !ifBVMAddrEmpty(v) {
				bvmAddr := crypto2.BVMAddress{}
				copy(bvmAddr[:], v[:])
				address = crypto2.ToAddr(bvmAddr)
			}

			AddrSlice = append(AddrSlice, address)
		}

		return AddrSlice, nil

	case "address[][]":
		s, err := jsoniter.Marshal(param)
		if err != nil {
			return nil, err
		}

		var slices [][]abi.Address
		err = jsoniter.Unmarshal(s, &slices)
		if err != nil {
			return nil, err
		}

		return GetBetweenStrAddr(slices), nil

	default:

		return param, nil
	}
}

func getParamType(paramType string) (pType string) {

	urlExpr := `[0-9]`
	for _, v := range paramType {

		s := string(v)
		match, _ := regexp.MatchString(urlExpr, s)
		if !match {
			pType = pType + s
		}
	}

	return
}

func ifBVMAddrEmpty(bvmAddr abi.Address) bool {

	count := 0

	for _, v := range bvmAddr {
		if v == byte(0) {
			count++
		}
	}

	return count == 20
}

func ifByteSliceAllZero(by []byte) bool {

	count := 0

	for _, v := range by {
		if v == byte(0) {
			count++
		}
	}

	return count == len(by)
}
