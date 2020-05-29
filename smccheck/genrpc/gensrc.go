package genrpc

import (
	"github.com/bcbchain/bcbchain/smccheck/parsecode"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const backQuote = "`"
const rpcTemplate = `package main

import (
    "blockchain/tx2"
	bcLibType "github.com/bcbchain/bclib/types"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"

  {{range $v,$vv := .Imports}}
{{$v.Name}} {{$v.Path}}{{end}}
)

type conf struct {
	BaseURL  string ` + backQuote + `yaml:"baseURL"` + backQuote + `
	GasLimit int64 ` + backQuote + `yaml:"gasLimit"` + backQuote + `
}

{{range $i,$u := .PlainUserStruct}}
type {{$u}}
{{end}}

func (c *conf) getConf() *conf {
	yamlFile, err := ioutil.ReadFile("conf.yaml")
	if err != nil {
		log.Fatalf("conf.yaml read error: #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("conf.yaml Unmarshal error: %v", err)
	}

	return c
}

type txParams struct {
	Note     string  ` + backQuote + `json:"note" form:"note"` + backQuote + `
	GasLimit int64 ` + backQuote + `json:"gasLimit" form:"gasLimit"` + backQuote + `
	Token    string ` + backQuote + `json:"token,omitempty" form:"token,omitempty"` + backQuote + `
	Value    string ` + backQuote + `json:"tokenValue,omitempty" form:"tokenValue,omitempty"` + backQuote + `
}

func main() {
	var c conf
	// c.getConf()

	fmt.Println(c)

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

{{range $i, $f := .Functions}}
	r.Any("/{{$f.Name|lowerFirst}}", {{$f.Name|lowerFirst}})
{{end}}
	e := r.Run(":{{.Port}}")
	if e != nil {
		panic(e)
	}
}

func getNonce(privKey string, password string) uint64 {
	return 1
}
{{range $i,$f := .Functions}}
func {{$f.Name|lowerFirst}}(c *gin.Context) {
	type msgParams struct { {{range $i0,$sPara := $f.SingleParams}}{{$sPara|expandNames|upperFirst}} {{$sPara|expandType}} ` + backQuote + `json:"{{$sPara|expandNames|lowerFirst}}" form:"{{$sPara|expandNames|lowerFirst}}"` + backQuote + `
{{end}}	}

	type params struct {
		EncPrivateKey string    ` + backQuote + `json:"encPrivateKey" form:"encPrivateKey"` + backQuote + `
		Password      string    ` + backQuote + `json:"password" form:"password"` + backQuote + `
		TxParams      txParams  ` + backQuote + `json:"txParams"` + backQuote + `
		MsgParams     msgParams ` + backQuote + `json:"msgParams"` + backQuote + `
	}

	type JSONReq struct {
		Version string ` + backQuote + `json:"jsonrpc"` + backQuote + `
		ID      string ` + backQuote + `json:"id"` + backQuote + `
		Method  string ` + backQuote + `json:"method"` + backQuote + `
		Params  params ` + backQuote + `json:"params"` + backQuote + `
	}

	tx := func(p params) string {
		bp := tx2.WrapInvokeParams({{range $i0,$sPara := $f.SingleParams}}p.MsgParams.{{$sPara|expandNames|upperFirst}},{{end}})
		nonce := getNonce(p.EncPrivateKey, p.Password)
		msg := bcLibType.Message{Contract: "{{if $.Owner}}{{calcConAddr $.ContractName $.Version $.Owner}}{{else}}{{getConAddr $.ContractName $.Version}}{{end}}", MethodID: {{$f.Method | createProto | calcMethodID | printf "0x%x"}}, Items: bp}
		pl := tx2.WrapPayload(nonce, p.TxParams.GasLimit, p.TxParams.Note, msg)
		return tx2.WrapTx(pl, p.EncPrivateKey)
	}
	var j JSONReq
	b2 := c.ShouldBindJSON(&j)
	if b2 == nil {
		if j.Version == "2.0" {
			str := tx(j.Params)
			c.String(http.StatusOK, str)
            return
		}
	} else {
		var p params
		b1 := c.ShouldBind(&p)
		if b1 == nil {
			if p.EncPrivateKey != "" {
				str := tx(p)
				c.String(http.StatusOK, str)
                return
			}
		}
	}
    c.String(http.StatusBadRequest, "BadRequest")
}
{{end}}
`

// FatFunction - flat params
type FatFunction struct {
	parsecode.Function
	SingleParams []parsecode.Field
}

// RPCExport - the functions for rpc & autogen types
type RPCExport struct {
	PackageName  string
	ReceiverName string
	ContractName string
	Version      string
	Owner        string
	Imports      map[parsecode.Import]struct{}
	Functions    []FatFunction
	MFunctions   []FatFunction
	IFunctions   []FatFunction
	Port         int

	PlainUserStruct []string
}

// Res2rpc - transform the parsed result to RPC Export struct
func Res2rpc(res *parsecode.Result, flag int64) RPCExport {
	exp := RPCExport{}
	exp.PackageName = res.PackageName
	exp.ReceiverName = strings.ToLower(string([]rune(res.ContractStructure)[0]))
	exp.ContractName = res.ContractName
	exp.Version = res.Version
	exp.Imports, exp.Functions, exp.PlainUserStruct = opFunctions(res, res.Functions)
	_, exp.MFunctions, _ = opFunctions(res, res.MFunctions)

	if flag == 2 {
		exp.Imports, exp.IFunctions, _ = opFunctions(res, res.IFunctions)
	} else {
		_, exp.IFunctions, _ = opFunctions(res, res.IFunctions)
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

// GenRPC - generate the rpc server go source
func GenRPC(res *parsecode.Result, port int, owner, outDir string) error {
	if err := os.MkdirAll(outDir, os.FileMode(0750)); err != nil {
		return err
	}
	filename := filepath.Join(outDir, "main.go")

	funcMap := template.FuncMap{
		"upperFirst":   parsecode.UpperFirst,
		"lowerFirst":   parsecode.LowerFirst,
		"expandNames":  parsecode.ExpandNames,
		"expandType":   parsecode.ExpandType,
		"createProto":  parsecode.CreatePrototype,
		"calcMethodID": parsecode.CalcMethodID,
		"calcConAddr":  parsecode.CalcContractAddress,
		"getConAddr":   parsecode.GetContractAddress,
	}
	tmpl, err := template.New("rpc").Funcs(funcMap).Parse(rpcTemplate)
	if err != nil {
		return err
	}

	obj := Res2rpc(res, 0)
	obj.Port = port
	obj.Owner = owner

	var buf bytes.Buffer

	if err = tmpl.Execute(&buf, obj); err != nil {
		return err
	}

	if err := parsecode.FmtAndWrite(filename, buf.String()); err != nil {
		return err
	}
	return nil
}
