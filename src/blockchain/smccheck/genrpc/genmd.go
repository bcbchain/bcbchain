package genrpc

import (
	"blockchain/smccheck/parsecode"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const tripleBackQuote = "```"

const mdTemplate = `
# RPC Doc

{{range $idx, $func := .Functions}}
## {{$idx | inc}}. {{$func.Name}}

gas - {{$func.Comments | getGas}}

- **Request URI over HTTPS**

  ` + tripleBackQuote + `url
  http://localhost:{{$.Port}}/{{$.ContractName}}/{{$.Version}}/{{$func.Name}}?encPrivateKey=_&password=_&note=_&gasLimit=_&token=_&value=_&to=_&{{$l:=len $func.SingleParams}}{{$l:=dec $l}}{{range $i0,$sPara := $func.SingleParams}}_{{$sPara|expandNames}}=_{{if lt $i0 $l}}&{{end}}{{end}}
  ` + tripleBackQuote + `

- **Request JSONRPC over HTTPS**

  ` + tripleBackQuote + `json
  {
    "jsonrpc": "2.0",
    "id": "dontcare/anything",
    "method": "{{$func.Name}}",
    "params": {
      "encPrivateKey": "0x5B4CBCDB3029FCF72AC9CF74EE2C336B2F9051FCA57E33713D6C663B0EC4BD26325A9C9CB3B710FDDC85D829234660F9",
      "password": "8717$326819019",
      "txParams": {
        "note": "",
        "gasLimit": 600,
        "token": "bcbLocFJG5Q792eLQXhvNkG417kwiaaoPH5a",
        "value": "1500000000"
      },
      "msgParams": {  {{$l:=len $func.SingleParams}}{{$l:=dec $l}}{{range $i0,$sPara := $func.SingleParams}}
        "_{{$sPara|expandNames}}": "something"{{if lt $i0 $l}},{{end}}{{end}}
      }
    }
  }
  ` + tripleBackQuote + `

- **Request Parameters**

  | **语法** | **类型** | **注释** |
  | --- | :---: | --- |
  | encPrivateKey | String | 私鑰 |
  | password | String | 钱包创建以后，设置钱包账户的密码。 |
  | txParams | 結構 | tx 通用參數組 |
  | note | String | 注釋 |
  | gasLimit | Number | 方法的 Gas 消費限制 |
  | token | Address | 代弊(可選) |
  | value | bigNumber | String 表示的大數，代弊數(可選) |
  | msgParams | 結構 | 消息參數組 |  {{range $i0,$sPara := $func.SingleParams}}
  | _{{$sPara|expandNames}} | {{$sPara|expandType}} | comment.. |{{end}}

- **Response SUCCESS Example**

  ` + tripleBackQuote + `json
  {
    "jsonrpc": "2.0",
    "id": "1",
    "result": {
      "...": "..."
    }
  }
  ` + tripleBackQuote + `

- **Response SUCCESS Parameters**

  | **语法** | **类型** | **注释** |
  | --- | :---: | --- |
  | ... | ... | ... |
{{end}}
`

// GenMarkdown - generate rpc document in markdown format.
func GenMarkdown(res *parsecode.Result, port int, destDir string) error {
	if err := os.MkdirAll(destDir, os.FileMode(0750)); err != nil {
		return err
	}
	filename := filepath.Join(destDir, "rpc_.md")

	funcMap := template.FuncMap{
		"upperFirst":  parsecode.UpperFirst,
		"lowerFirst":  parsecode.LowerFirst,
		"expandNames": parsecode.ExpandNames,
		"expandType":  parsecode.ExpandType,
		"getGas":      parsecode.GetGas,
		"dec": func(i int) int {
			return i - 1
		},
		"inc": func(i int) int {
			return i + 1
		},
	}
	tmpl, err := template.New("md").Funcs(funcMap).Parse(mdTemplate)
	if err != nil {
		return err
	}

	obj := Res2rpc(res, 0)
	obj.Port = port

	var buf bytes.Buffer

	if err = tmpl.Execute(&buf, obj); err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if ee := f.Close(); ee != nil {
			fmt.Println(ee)
		}
	}()

	_, err = f.WriteString(buf.String())
	if err != nil {
		return err
	}
	// fmt.Println(n, "byte write to file")

	return nil
}
