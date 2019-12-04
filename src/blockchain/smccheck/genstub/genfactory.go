package genstub

import (
	"blockchain/smccheck/parsecode"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const factoryTemplate = `package {{$.PackageName}}

import (
	"blockchain/smcsdk/sdk"
	sdktypes "blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl/helper"
	"contract/stubcommon/types"

	{{- range $i,$v := $.Versions}}
	{{$v | replace}} "contract/{{$.OrgID}}/stub/{{$.DirectionName}}/{{$v}}/{{$.DirectionName}}"
	{{- end}}
)

//NewInterfaceStub new interface stub
func NewInterfaceStub(smc sdk.ISmartContract, contractName string) types.IContractIntfcStub {
	//Get contract with ContractName
	ch := helper.ContractHelper{}
	ch.SetSMC(smc)
	contract := ch.ContractOfName(contractName)
	sdk.Require(contract != nil, sdktypes.ErrExpireContract, "")
	switch contract.Version() {
	{{- range $i1,$v1 := $.Versions}}
	case "{{v $v1}}":
		return {{replace $v1}}.NewInterStub(smc)
	{{- end}}
	}
	return nil
}
`

// GenStubFactory - generate the interface stub factory go source
func GenStFactory(res *parsecode.Result, outDir string) {
	if err := os.MkdirAll(outDir, os.FileMode(0750)); err != nil {
		panic(err)
	}
	filename := filepath.Join(outDir, "interfacestubfactory.go")

	funcMap := template.FuncMap{
		"replace": func(version string) string {
			return strings.Replace(version, ".", "_", -1)
		},
		"v": func(version string) string {
			return version[1:]
		},
	}

	tmpl, err := template.New("interfaceStubFactory").Funcs(funcMap).Parse(factoryTemplate)
	if err != nil {
		panic(err)
	}

	obj := Res2stub(res, 1)

	var buf bytes.Buffer

	if err = tmpl.Execute(&buf, obj); err != nil {
		panic(err)
	}

	if err := parsecode.FmtAndWrite(filename, buf.String()); err != nil {
		panic(err)
	}
}
