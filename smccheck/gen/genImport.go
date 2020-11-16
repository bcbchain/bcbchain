package gen

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/bcbchain/bcbchain/smccheck/parsecode"
	"github.com/bcbchain/sdk/sdk/std"
	"go/ast"
	"path/filepath"
	"strings"
	"text/template"
)

var importTemplate = `package {{.PackageName}}

import (
	bcbgls "github.com/bcbchain/sdk/common/gls"
	"github.com/bcbchain/sdk/sdk"
	{{range $v,$vv := .Imports}}
	{{$v.Name}} {{$v.Path}}{{end}}
	"github.com/bcbchain/sdk/sdk/types"
	"github.com/bcbchain/sdk/sdkimpl"
	"github.com/bcbchain/sdk/sdkimpl/object"
	"github.com/bcbchain/sdk/sdkimpl/sdkhelper"
	"github.com/bcbchain/sdk/sdk/jsoniter"
	"contract/{{$.OrgID}}/stub/{{$.DirectionName}}"
	types2 "contract/stubcommon/types"
	types3 "github.com/bcbchain/bclib/types"

	{{range $i, $contract := $.IContracts}}{{if eq $contract.Name $.ImportContract}}
	{{$contract.Name}}v{{vEx $contract.Version}} "contract/{{$.OrgID}}/code/{{$.DirectionName}}/v{{$contract.Version}}/{{$.DirectionName}}"
	{{- end}}{{- end}}
)

//InterfaceStub{{$.Index}} interface stub of {{$.ImportContract}}
type InterfaceStub{{$.Index}} struct {
    stub types2.IContractIntfcStub
	receipts []types.KVPair
}

const importContractName{{$.Index}} = "{{$.ImportContract}}"
func (s *{{.ContractStructure}}) {{$.ImportContract}}() *InterfaceStub{{$.Index}} {
    return &InterfaceStub{{$.Index}}{ 
		stub: {{$.ImportPackage}}.NewInterfaceStub(s.sdk, importContractName{{$.Index}}),
		receipts: make([]types.KVPair, 0),
	}
}

func (intfc *InterfaceStub{{$.Index}}) run(f func()) *InterfaceStub{{$.Index}} {
	// step 1. save old all receipts
	oldReceipts := intfc.stub.GetSdk().Message().(*object.Message).OutputReceipts()

	// step 2. run function
	f()

	// step3. save new all receipts
	newReceipts := intfc.stub.GetSdk().Message().(*object.Message).OutputReceipts()

	// step4. set new receipts into object
	if len(newReceipts) > len(oldReceipts) {
		intfc.receipts = newReceipts[len(oldReceipts):]
	}
	return intfc
}

func (intfc *InterfaceStub{{$.Index}}) contract() sdk.IContract {
	return intfc.stub.GetSdk().Helper().ContractHelper().ContractOfName(importContractName{{$.Index}})
}

{{range $j, $method := $.ImportInterfaces}}
// {{$method.Name}}
func (intfc *InterfaceStub{{$.Index}}) {{$method.Name}}({{range $i0, $param := $method.Params}}{{$param | expNames}} {{$param | expType}}{{if lt $i0 (dec (len $method.Params))}},{{end}}{{end}}) {{if (len $method.Results)}}string{{end}} {

    methodID := "{{$method | createProto | calcMethodID | printf "%x"}}" // prototype: {{createProto $method}}
    oldSmc := intfc.stub.GetSdk()
    defer intfc.stub.SetSdk(oldSmc)
	oldmsg := oldSmc.Message()
	defer oldSmc.(*sdkimpl.SmartContract).SetMessage(oldmsg)
    //合约调用时的输入收据，同时可作为跨合约调用的输入收据
    contract := oldSmc.Helper().ContractHelper().ContractOfName(importContractName{{$.Index}})
	sdk.Require(contract != nil, types.ErrExpireContract, "")
    newSmc := sdkhelper.OriginNewMessage(oldSmc, contract, methodID, intfc.receipts)
    intfc.stub.SetSdk(newSmc)

    // 编译时builder从数据库已获取合约版本和失效高度，直接使用
    height := newSmc.Block().Height()
    var rn interface{}
	{{createVar $.Contracts $.ImportContract $method}}
	
	var response types3.Response
	bcbgls.Mgr.SetValues(bcbgls.Values{bcbgls.SDKKey: newSmc}, func() {
		response = intfc.stub.Invoke(methodID, rn)
	})
    if response.Code != types.CodeOK {
		err := types.Error{ErrorCode: response.Code, ErrorDesc: response.Log}
        panic(err)
    }
	receipts := make([]types.KVPair, 0, len(response.Tags))
	for _, v := range response.Tags {
		receipts = append(receipts, types.KVPair{Key:v.Key, Value:v.Value})
	}
    oldmsg.(*object.Message).AppendOutput(receipts)
	intfc.receipts = nil

	if len(response.Data) == 0 {
		response.Data = "[]"
	}
	var results []interface{}
	err := jsoniter.Unmarshal([]byte(response.Data), &results)
	sdk.RequireNotError(err, types.ErrInvalidParameter)

    {{if (len $method.Results)}}return {{range $i, $result := $method.Results}}results[$i].({{expType $result}}){{end}}{{end}}
}

{{- end}}
`

type ContractInfo struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	EffectHeight int64  `json:"effectHeight"`
	LoseHeight   int64  `json:"loseHeight"`
}

type OtherContract struct {
	OrgID         string
	DirectionName string
	Name          string
	PackageName   string
	Version       string
	LoseHeight    int64
	EffectHeight  int64
	Functions     []parsecode.Function
	IFunctions    []parsecode.Function
	UserStruct    map[string]ast.GenDecl
}

type ImportContract struct {
	Contracts  []OtherContract
	IContracts []OtherContract

	// 当前合约参数
	OrgID             string
	PackageName       string
	ContractStructure string

	// 跨合约信息
	ImportContract     string
	DirectionName      string
	ImportPackage      string
	Index              int
	ImportInterfaces   []parsecode.Method
	ImportContractInfo std.ContractVersionList
	Imports            map[parsecode.Import]struct{}
}

func res2importContract(res *parsecode.Result, reses []*parsecode.Result, contractInfoList []ContractInfo, index int) (*ImportContract, error) {

	sLen := len(reses)
	importContract := ImportContract{
		Contracts:         make([]OtherContract, 0, sLen),
		IContracts:        make([]OtherContract, 0, sLen),
		ImportInterfaces:  res.ImportContracts[index].Interfaces,
		ContractStructure: res.ContractStructure,
		PackageName:       res.PackageName,
		Index:             index,
		ImportContract:    res.ImportContracts[index].Name,
		OrgID:             res.OrgID,
	}

	for _, item := range reses {
		if len(importContract.DirectionName) == 0 && item.ContractName == importContract.ImportContract {
			importContract.DirectionName = item.DirectionName
		}

		contract := OtherContract{
			OrgID:         item.OrgID,
			DirectionName: item.DirectionName,
			Name:          item.ContractName,
			PackageName:   item.PackageName,
			Version:       item.Version,
			Functions:     item.Functions,
			IFunctions:    item.IFunctions,
			UserStruct:    item.UserStruct,
		}
		contract.EffectHeight, contract.LoseHeight = contractInfoOfNameVersion(contract.Name, contract.Version, contractInfoList)

		importContract.Contracts = append(importContract.Contracts, contract)
	}

	mContracts := importVersions(importContract.Contracts, importContract.ImportContract, importContract.ImportInterfaces)
	if len(mContracts) == 0 {
		return nil, errors.New("genImport error, no adaptive contract's version")
	}

	for _, value := range mContracts {
		importContract.IContracts = append(importContract.IContracts, value)
	}

	importContract.ImportPackage = importContract.IContracts[0].PackageName
	importContract.ImportContractInfo.Name = importContract.ImportContract
	importContract.ImportContractInfo.EffectHeights = []int64{1000, -1}

	imports := make(map[parsecode.Import]struct{})
	for _, p := range importContract.ImportInterfaces {
		for _, m := range p.Params {
			for imp := range m.RelatedImport {
				imports[imp] = struct{}{}

			}
		}
	}
	importContract.Imports = make(map[parsecode.Import]struct{})
	importContract.Imports = imports
	return &importContract, nil
}

func contractInfoOfNameVersion(name, version string, contractInfoList []ContractInfo) (effectHeight, loseHeight int64) {

	for _, contractInfo := range contractInfoList {
		if contractInfo.Name == name && contractInfo.Version == version {
			return contractInfo.EffectHeight, contractInfo.LoseHeight
		}
	}

	return 0, 0
}

// GenImport - generate import code from source smart contract to destination smart contract
func GenImport(inPath string, res *parsecode.Result, reses []*parsecode.Result, contractInfoList []ContractInfo, index int) error {
	filename := filepath.Join(inPath, res.PackageName+"_autogen_import_"+res.ImportContracts[index].Name+".go")

	funcMap := template.FuncMap{
		"upperFirst":   parsecode.UpperFirst,
		"lowerFirst":   parsecode.LowerFirst,
		"expNames":     parsecode.ExpandNames,
		"expType":      parsecode.ExpandType,
		"expNoS":       parsecode.ExpandTypeNoStar,
		"expK":         parsecode.ExpandMapFieldKey,
		"expV":         parsecode.ExpandMapFieldVal,
		"expVNoS":      parsecode.ExpandMapFieldValNoStar,
		"createProto":  parsecode.CreatePrototype,
		"calcMethodID": parsecode.CalcMethodID,
		"createVar":    createVar,
		"isEx":         isEx,
		"vEx":          vEx,
		"dec": func(i int) int {
			return i - 1
		},
	}

	importContract, err := res2importContract(res, reses, contractInfoList, index)
	if err != nil {
		return err
	}

	tmpl, err := template.New("import").Funcs(funcMap).Parse(importTemplate)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, importContract); err != nil {
		panic(err)
	}

	if err = parsecode.FmtAndWrite(filename, buf.String()); err != nil {
		panic(err)
	}

	return nil
}

func vEx(version string) string {
	return strings.Replace(version, ".", "", -1)
}

// createVar - create string of stride smart contract code about parameter for method
func createVar(allContracts []OtherContract, contractName string, method parsecode.Method) string {
	contracts := getContracts(allContracts, contractName)

	formatStr := ""

	for _, contract := range contracts {
		if isOK(contract.IFunctions, method) {
			formatStr += exchangeVar(contract, method)
			break
		}
	}

	for index, contract := range contracts {
		item := getArg(contract.IFunctions, method)
		if index == 0 {
			if contract.LoseHeight == 0 {
				formatStr += fmt.Sprintf("\tif height >= %d {\n", contract.EffectHeight)
			} else {
				formatStr += fmt.Sprintf("\tif height < %d {\n", contract.LoseHeight)
			}
		} else if index < len(contracts)-1 {
			formatStr += fmt.Sprintf(" else if height < %d {\n", contract.LoseHeight)
		} else {
			formatStr += fmt.Sprintf(" else {\n")
		}
		formatStr += "\t\t"
		if isOK(contract.IFunctions, method) {
			formatStr += fmt.Sprintf("rn = %sv%s.%sParam", contract.Name, vEx(contract.Version), method.Name)
		}
		formatStr += item
		formatStr += "\n\t}"
	}

	return formatStr
}

var baseTypes = []string{"int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64",
	"bool", "[]byte", "types.Address", "bn.Number", "types.HexBytes", "types.Hash", "types.PubKey", "string", "byte"}

func exchangeVar(contract OtherContract, method parsecode.Method) string {

	exchangeStr := ""
	isF := true
	for _, function := range contract.Functions {
		if function.Name == method.Name {
			for index, filed := range function.Params {
				isBase := true
				varType := strings.TrimLeft(parsecode.ExpandType(filed), "*")
				varType = strings.TrimLeft(varType, "[]")
				varType = strings.TrimLeft(varType, "*")
				for _, typeStr := range baseTypes {
					if varType == typeStr {
						isBase = true
						break
					}
				}

				if isBase == false {
					var names []string
					for indexN, name := range method.Params[index].Names {
						if isF {
							exchangeStr += "var err error\n"
							exchangeStr += "\tvar resBytes []byte\n"
							isF = false
						}
						typeTemp := varType
						if isTypeIn(contract, varType) {
							typeTemp = fmt.Sprintf("%sv%s.%s", contract.Name, vEx(contract.Version), varType)
						}
						exchangeStr += fmt.Sprintf("\tvar p%d%d %s\n", index, indexN, typeTemp)
						exchangeStr += fmt.Sprintf("\tresBytes, err = jsoniter.Marshal(%s)\n", name)
						exchangeStr += "\tif err != nil {\n"
						exchangeStr += "\t\tpanic(err)\n"
						exchangeStr += "\t}\n"
						exchangeStr += fmt.Sprintf("\terr = jsoniter.Unmarshal(resBytes, &p%d%d)\n", index, indexN)
						exchangeStr += "\tif err != nil {\n"
						exchangeStr += "\t\tpanic(err)\n"
						exchangeStr += "\t}\n\n"
						names = append(names, fmt.Sprintf("p%d%d", index, indexN))
					}
					method.Params[index].Names = names
				}
			}
		}
	}

	return exchangeStr
}

func isTypeIn(contract OtherContract, typeStr string) bool {
	for key := range contract.UserStruct {
		if key == typeStr {
			return true
		}
	}

	return false
}

func getContracts(allContracts []OtherContract, contractName string) []OtherContract {
	contracts := make([]OtherContract, 0)

	for _, contract := range allContracts {
		if contract.Name == contractName {
			contracts = append(contracts, contract)
		}
	}

	return contracts
}

func isOK(functions []parsecode.Function, method parsecode.Method) bool {
	for _, function := range functions {
		if function.Name == method.Name {
			// step 1. check the parameter's count
			// step 2. check the parameter's name and type to same, different type will make different methodID
			mLenParams := 0
			for index, param := range method.Params {
				mLenParams += len(param.Names)

				if parsecode.ExpandType(param) != parsecode.ExpandType(function.Params[index]) {
					return false
				}
			}

			fLenParams := 0
			for _, param := range function.Params {
				fLenParams += len(param.Names)
			}

			if fLenParams == mLenParams {
				return true
			} else {
				return false
			}
		}
	}

	return false
}

func getArg(functions []parsecode.Function, method parsecode.Method) string {
	for _, function := range functions {
		if function.Name == method.Name {
			mLenParams := 0
			for _, param := range method.Params {
				mLenParams += len(param.Names)
			}

			paramStr := ""
			fLenParams := 0
			for _, param := range function.Params {
				fLenParams += len(param.Names)
			}

			if fLenParams != mLenParams {
				return `panic("Invalid parameter")`
			}

			for index1, param := range function.Params {
				for index2, name := range param.Names {
					paramStr += parsecode.UpperFirst(name) + ":" + method.Params[index1].Names[index2] + ","
				}
			}

			if len(paramStr) != 0 {
				paramStr = paramStr[:len(paramStr)-1]
			}
			paramStr = "{" + paramStr + "}"

			return paramStr
		}
	}

	return `panic("Invalid parameter")`
}

func isEx(allContracts []OtherContract, contractName string, methods []parsecode.Method) bool {
	for _, method := range methods {
		contracts := getContracts(allContracts, contractName)

		for _, contract := range contracts {
			if isOK(contract.IFunctions, method) {
				temp := exchangeVar(contract, method)
				if len(temp) != 0 {
					return true
				}
			}
		}
	}

	return false
}

//func isBn(methods []parsecode.Method) bool {
//	for _, m := range methods {
//		for _, p := range m.Params {
//			//p.
//		}
//	}
//
//	return false
//}

func importVersions(allContracts []OtherContract, contractName string, interfaces []parsecode.Method) map[string]OtherContract {
	contracts := getContracts(allContracts, contractName)

	verList := make(map[string]OtherContract, 0)

	for _, item := range interfaces {
		for _, contract := range contracts {
			if isOK(contract.IFunctions, item) {
				verList[contract.Version] = contract
			}
		}
	}

	return verList
}
