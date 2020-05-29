package genstub

import (
	"github.com/bcbchain/bcbchain/smccheck/parsecode"
	"bytes"
	"os"
	"path/filepath"
	"text/template"
)

const ibcStubTemplate = `package {{$.PackageName}}stub

import (
	bcType "github.com/bcbchain/bclib/types"

	"github.com/bcbchain/sdk/sdk"
	"contract/stubcommon/common"
	stubType "contract/stubcommon/types"
	tmcommon "github.com/tendermint/tmlibs/common"
	"github.com/bcbchain/sdk/sdk/types"
	{{if ne (len .TFunctions) 0}}. "contract/{{$.OrgID}}/code/{{$.DirectionName}}/v{{$.Version}}/{{$.DirectionName}}"{{end}}

	"github.com/tendermint/tmlibs/log"
	{{- if (hasParam .TFunctions)}}
	"github.com/bcbchain/sdk/sdk/rlp"
	{{- end}}
	{{- if (hasResult .TFunctions)}}
	"github.com/bcbchain/sdk/sdk/jsoniter"
	{{- end}}

  	{{- range $v,$vv := .Imports}}
	{{- if (filterImport $v.Path)}}
  	{{$v.Name}} {{$v.Path}}
	{{- end}}
	{{- end}}
)

{{$stubName := (printf "%sIBCStub" $.ContractStruct)}}

//{{$stubName}} an object
type {{$stubName}} struct {
	logger log.Logger
}

var _ stubType.IContractIBCStub = (*{{$stubName}})(nil)

func init() {
	recoverPanicMap = make(map[string]struct{})
	recoverPanicMap["assignment to entry in nil map"] = struct{}{}
}

//New generate a stub
func NewIBC(logger log.Logger) stubType.IContractIBCStub {
	return &{{$stubName}}{logger: logger}
}

//Invoke invoke function
func (pbs *{{$stubName}}) Invoke(smc sdk.ISmartContract) (response bcType.Response) {
	defer FuncRecover(smc, &response)

	// 生成手续费收据
	fee, gasUsed, feeReceipt, err := common.FeeAndReceipt(smc, common.IBC)
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
	{{- range $i,$f := $.TFunctions}}
	case "{{$f.Method | createProto | calcMethodID | printf "%x"}}":	// prototype: {{createProto $f.Method}}
		{{if eq (len $f.Results) 1}}data = {{end}}_{{lowerFirst $f.Name}}(smc)
	{{- end}}
	default:
		err.ErrorCode = types.ErrInvalidMethod
	}
	response = common.CreateResponse(smc, response.Tags, data, fee, gasUsed, smc.Tx().GasLimit(), err)
	return
}

{{range $i0,$f := $.TFunctions}}
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

// GenIBCStub - generate the ibc stub go source
func GenIBCStub(res *parsecode.Result, outDir string) {
	newOutDir := filepath.Join(outDir, "v"+res.Version, res.DirectionName)
	if err := os.MkdirAll(newOutDir, os.FileMode(0750)); err != nil {
		panic(err)
	}
	filename := filepath.Join(newOutDir, res.PackageName+"stub_ibc.go")

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
	tmpl, err := template.New("ibcStub").Funcs(funcMap).Parse(ibcStubTemplate)
	if err != nil {
		panic(err)
	}

	obj := Res2stub(res, 2)

	var buf bytes.Buffer

	if err = tmpl.Execute(&buf, obj); err != nil {
		panic(err)
	}

	if err := parsecode.FmtAndWrite(filename, buf.String()); err != nil {
		panic(err)
	}
}
