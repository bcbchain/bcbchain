package genstub

import (
	"blockchain/smccheck/gen"
	"blockchain/smccheck/parsecode"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const stubTemplate = `package {{$.PackageName}}stub

import (
	"blockchain/smcsdk/sdkimpl"
	bcType "blockchain/types"
	"strings"
	"runtime"
	"fmt"

	"blockchain/smcsdk/sdk"
	"contract/stubcommon/common"
	"contract/stubcommon/softforks"
	stubType "contract/stubcommon/types"
	tmcommon "github.com/tendermint/tmlibs/common"
	"blockchain/smcsdk/sdk/types"
	. "contract/{{$.OrgID}}/code/{{$.DirectionName}}/v{{$.Version}}/{{$.DirectionName}}"

	"github.com/tendermint/tmlibs/log"
	{{- if (hasParam .MFunctions)}}
	"blockchain/smcsdk/sdk/rlp"
	{{- end}}
	{{- if (hasResult .MFunctions)}}
	"blockchain/smcsdk/sdk/jsoniter"
	{{- end}}

  	{{- range $v,$vv := .Imports}}
	{{- if (filterImport $v.Path)}}
  	{{$v.Name}} {{$v.Path}}
	{{- end}}
	{{- end}}
)

{{$stubName := (printf "%sStub" $.ContractStruct)}}

//{{$stubName}} an object
type {{$stubName}} struct {
	logger log.Logger
}

var _ stubType.IContractStub = (*{{$stubName}})(nil)
var recoverPanicMap map[string]struct{}

func init() {
	recoverPanicMap = make(map[string]struct{})
	recoverPanicMap["assignment to entry in nil map"] = struct{}{}
}

//New generate a stub
func New(logger log.Logger) stubType.IContractStub {
	return &{{$stubName}}{logger: logger}
}

//FuncRecover recover panic by Assert
func FuncRecover(smc sdk.ISmartContract, response *bcType.Response) {
	if err := recover(); err != nil {
		if _, ok := err.(types.Error); ok {
			e := err.(types.Error)

			if softforks.V2_0_1_13780(smc.Block().Height()) {
				response.Code = e.ErrorCode
				response.Log = e.ErrorDesc
			} else {
				*response = common.CreateResponse(smc, response.Tags, "", response.Fee, response.GasLimit, smc.Tx().GasLimit(), e)
			}
		} else if e, ok := err.(error); ok {
			if strings.HasPrefix(e.Error(), "runtime error") {
				logCaller()
				if softforks.V2_0_1_13780(smc.Block().Height()) {
					response.Code = types.ErrStubDefined
					response.Log = e.Error()
				} else {
					*response = common.CreateResponse(smc, response.Tags, "", response.Fee, response.GasLimit, smc.Tx().GasLimit(), types.Error{
						ErrorCode: types.ErrStubDefined,
						ErrorDesc: e.Error(),
					})
				}
			} else if _, ok = recoverPanicMap[e.Error()]; ok {
				logCaller()
				if softforks.V2_0_1_13780(smc.Block().Height()) {
					response.Code = types.ErrStubDefined
					response.Log = e.Error()
				} else {
					*response = common.CreateResponse(smc, response.Tags, "", response.Fee, response.GasLimit, smc.Tx().GasLimit(), types.Error{
						ErrorCode: types.ErrStubDefined,
						ErrorDesc: e.Error(),
					})
				}
			} else {
				panic(err)
			}
		} else {
			panic(err)
		}
	}
}

func logCaller() {
	caller := ""
	skip := 0
	_, callerFile, callerLine, ok := runtime.Caller(skip)
	if !ok {
		return
	}
	caller += fmt.Sprintf("%s %d\n", callerFile, callerLine)
	var testFile string
	var testLine int
	for {
		skip++
		if _, file, line, ok := runtime.Caller(skip); ok {
			testFile, testLine = file, line
			caller += fmt.Sprintf("%s %d\n", testFile, testLine)
		} else {
			break
		}
	}
	if testFile != "" && (testFile != callerFile || testLine != callerLine) {
		caller += fmt.Sprintf("%s %d\n", testFile, testLine)
	}
	caller += fmt.Sprintf("%s %d\n", testFile, testLine)

	sdkimpl.Logger.Error(caller)
}

// InitChain initial smart contract
func (pbs *{{$stubName}}) InitChain(smc sdk.ISmartContract) (response bcType.Response) {
	defer FuncRecover(smc, &response)

	{{- if $.IsExistInitChain}}
	contractObj := new({{$.ContractStruct}})
	contractObj.SetSdk(smc)
	contractObj.InitChain()
	{{- end}}	

	response.Code = types.CodeOK
	return response
}

// UpdateChain update smart contract
func (pbs *{{$stubName}}) UpdateChain(smc sdk.ISmartContract) (response bcType.Response) {
	defer FuncRecover(smc, &response)
	
	{{- if $.IsExistUpdateChain}}
	contractObj := new({{$.ContractStruct}})
	contractObj.SetSdk(smc)
	contractObj.UpdateChain()
	{{- end}}	

	response.Code = types.CodeOK
	return response
}

// Mine call mine of smart contract
func (pbs *{{$stubName}}) Mine(smc sdk.ISmartContract) (response bcType.Response) {
	defer FuncRecover(smc, &response)
	
	{{- if $.IsExistMine}}
	contractObj := new({{$.ContractStruct}})
	contractObj.SetSdk(smc)
	rewardAmount := contractObj.Mine()
	response.Data = fmt.Sprintf("%d", rewardAmount)
	{{- end}}	

	response.Code = types.CodeOK
	return response
}

//Invoke invoke function
func (pbs *{{$stubName}}) Invoke(smc sdk.ISmartContract) (response bcType.Response) {
	return pbs.InvokeInternal(smc, common.METHOD)
}

//InvokeInternal invoke function
func (pbs *{{$stubName}}) InvokeInternal(smc sdk.ISmartContract, invokeType int) (response bcType.Response) {
	defer FuncRecover(smc, &response)

	// 生成手续费收据
	fee, gasUsed, feeReceipt, err := common.FeeAndReceipt(smc, invokeType)
	response.Fee = fee
	response.GasUsed = gasUsed
 	response.Tags = append(response.Tags, tmcommon.KVPair{Key:feeReceipt.Key, Value:feeReceipt.Value})
	if err.ErrorCode != types.CodeOK {
		response = common.CreateResponse(smc, response.Tags, "", fee, gasUsed, smc.Tx().GasLimit(), err)
		return
	}

	var data string
	err = types.Error{ErrorCode:types.CodeOK}

	pbs.logger.Debug("invoke", "methodID", smc.Message().MethodID())
	switch smc.Message().MethodID() {
	{{- range $i,$f := $.MFunctions}}
	case "{{$f.Method | createProto | calcMethodID | printf "%x"}}":	// prototype: {{createProto $f.Method}}
		{{if eq (len $f.Results) 1}}data = {{end}}_{{lowerFirst $f.Name}}(smc)
	{{- end}}
	default:
		err.ErrorCode = types.ErrInvalidMethod
	}
	response = common.CreateResponse(smc, response.Tags, data, fee, gasUsed, smc.Tx().GasLimit(), err)
	return
}

{{range $i0,$f := $.MFunctions}}
func _{{lowerFirst $f.Name}}(smc sdk.ISmartContract) {{if (len $f.Results)}}string{{end}} {
	items := smc.Message().Items()
	sdk.Require(len(items) == {{paramsLen $f.Method}}, types.ErrStubDefined, "Invalid message data")

	{{- if len $f.SingleParams}}
	var err error
	{{- end}}
	{{range $i1,$param := $f.SingleParams}}
	var v{{$i1}} {{$param | expandType}}
	err = rlp.DecodeBytes(items[{{$i1}}], &v{{$i1}})
	sdk.RequireNotError(err, types.ErrInvalidParameter)
	{{end}}

	contractObj := new({{$.ContractStruct}})
	contractObj.SetSdk(smc)
	{{$l := dec (len $f.Results)}}{{if (len $f.Results)}}{{range $i0,$sPara := $f.Results}}rst{{$i0}}{{if lt $i0 $l}},{{end}}{{end}} := {{end}}contractObj.{{$f.Name}}{{$l2 := dec (len $f.SingleParams)}}({{range $i2,$sPara := $f.SingleParams}}v{{$i2}}{{if lt $i2 $l2}},{{end}}{{end}})
	{{- if (len $f.Results)}}
	resultList := make([]interface{}, 0)
	{{range $i0,$sPara := $f.Results}}resultList = append(resultList, rst{{$i0}})
	{{end}}
	resBytes, _ := jsoniter.Marshal(resultList)
	return string(resBytes)
	{{- end}}
}
{{end}}
`

// FatFunction - flat params
type FatFunction struct {
	parsecode.Function
	SingleParams []parsecode.Field
}

// RPCExport - the functions for rpc & autogen types
type StubExport struct {
	DirectionName  string
	PackageName    string
	ReceiverName   string
	ContractName   string
	ContractStruct string
	Version        string
	Versions       []string
	OrgID          string
	Owner          string
	Imports        map[parsecode.Import]struct{}
	Functions      []FatFunction
	MFunctions     []FatFunction
	IFunctions     []FatFunction
	TFunctions     []FatFunction
	Port           int

	IsExistUpdateChain bool
	IsExistInitChain   bool
	IsExistMine        bool
	PlainUserStruct    []string
}

// Res2rpc - transform the parsed result to RPC Export struct
// funcType: 0:method, 1:interface, 2:ibc
func Res2stub(res *parsecode.Result, funcType int) StubExport {
	exp := StubExport{}
	exp.DirectionName = res.DirectionName
	exp.PackageName = res.PackageName
	exp.ReceiverName = strings.ToLower(string([]rune(res.ContractStructure)[0]))
	exp.ContractName = res.ContractName
	exp.ContractStruct = res.ContractStructure
	exp.OrgID = res.OrgID
	exp.Version = res.Version
	exp.Versions = res.Versions
	exp.IsExistUpdateChain = res.IsExistUpdateChain
	exp.IsExistInitChain = res.IsExistInitChain
	exp.IsExistMine = res.IsExistMine

	_, exp.Functions, exp.PlainUserStruct = opFunctions(res, res.Functions)
	switch funcType {
	case 0:
		exp.Imports, exp.MFunctions, _ = opFunctions(res, res.MFunctions)
	case 1:
		exp.Imports, exp.IFunctions, _ = opFunctions(res, res.IFunctions)
	case 2:
		exp.Imports, exp.TFunctions, _ = opFunctions(res, res.TFunctions)
	}

	return exp
}

func opFunctions(res *parsecode.Result, funcs []parsecode.Function) (map[parsecode.Import]struct{}, []FatFunction, []string) {
	imports := make(map[parsecode.Import]struct{})
	fatFunctions := make([]FatFunction, 0, len(funcs))
	pus := make([]string, 0)
	for _, f := range funcs {
		fat := FatFunction{
			Function: f,
		}
		singleParams := make([]parsecode.Field, 0)
		for _, para := range f.Params {
			for imp := range para.RelatedImport {
				imports[imp] = struct{}{}
			}
			singleParams = append(singleParams, parsecode.FieldsExpand(para)...)
			t := parsecode.ExpandTypeNoStar(para)
			t = strings.TrimSpace(t)
			if u, ok := res.UserStruct[t]; ok {
				pus = append(pus, parsecode.ExpandStruct(u))
			}
		}
		fat.SingleParams = singleParams
		fatFunctions = append(fatFunctions, fat)
	}

	return imports, fatFunctions, pus
}

// GenMethodStub - generate the method stub go source
func GenMethodStub(res *parsecode.Result, outDir string) {
	newOutDir := filepath.Join(outDir, "v"+res.Version, res.DirectionName)
	if err := os.MkdirAll(newOutDir, os.FileMode(0750)); err != nil {
		panic(err)
	}
	filename := filepath.Join(newOutDir, res.PackageName+"stub_method.go")

	funcMap := template.FuncMap{
		"upperFirst":   parsecode.UpperFirst,
		"lowerFirst":   parsecode.LowerFirst,
		"expandNames":  parsecode.ExpandNames,
		"expandType":   parsecode.ExpandType,
		"createProto":  parsecode.CreatePrototype,
		"paramsLen":    parsecode.ParamsLen,
		"calcMethodID": parsecode.CalcMethodID,
		"filterImport": parsecode.FilterImports,
		"hasStruct":    hasStruct,
		"dec": func(i int) int {
			return i - 1
		},
		"hasMethod": func(functions []FatFunction) bool {
			return len(functions) != 0
		},
		"hasParam": func(functions []FatFunction) bool {
			for _, f := range functions {
				if len(f.Params) != 0 {
					return true
				}
			}

			return false
		},
		"hasResult": func(functions []FatFunction) bool {
			for _, function := range functions {
				if len(function.Results) > 0 {
					return true
				}
			}

			return false
		},
	}
	tmpl, err := template.New("methodStub").Funcs(funcMap).Parse(stubTemplate)
	if err != nil {
		panic(err)
	}

	obj := Res2stub(res, 0)

	var buf bytes.Buffer

	if err = tmpl.Execute(&buf, obj); err != nil {
		panic(err)
	}

	if err := parsecode.FmtAndWrite(filename, buf.String()); err != nil {
		panic(err)
	}
}

func hasStruct(functions []FatFunction) bool {
	for _, function := range functions {
		for _, param := range function.Params {
			if gen.IsLiteralType(param) {
				continue
			}

			if gen.IsLiteralTypeEx(param) {
				continue
			}

			if gen.IsBnNumber(param) {
				continue
			}

			if gen.IsMap(param) {
				continue
			}

			return true
		}

		for _, result := range function.Results {
			if gen.IsLiteralType(result) {
				continue
			}

			if gen.IsLiteralTypeEx(result) {
				continue
			}

			if gen.IsBnNumber(result) {
				continue
			}

			if gen.IsMap(result) {
				continue
			}

			return true
		}
	}

	return false
}
