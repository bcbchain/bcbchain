package genstub

import (
	"github.com/bcbchain/bcbchain/smccheck/parsecode"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const templateText = `package stub
import (
	"github.com/bcbchain/sdk/sdk"
	"contract/stubcommon/common"
	"contract/stubcommon/types"
	"fmt"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"

	{{- range $i,$directionName := $.DirectionNames}}
	{{getName $i $.PackageNames}}{{replace (version $i $.Versions)}} "contract/{{getOrgID $i $.OrgIDs}}/stub/{{$directionName}}/v{{version $i $.Versions}}/{{$directionName}}"
	{{- end}}
)

func NewStub(smc sdk.ISmartContract, logger log.Logger) types.IContractStub {

	switch common.CalcKey(smc.Message().Contract().Name(), smc.Message().Contract().Version()) {
	{{- range $j,$contractName := $.ContractNames}}
	case "{{$contractName}}{{replace (version $j $.Versions)}}":
		return {{getName $j $.PackageNames}}{{replace (version $j $.Versions)}}.New(logger)
	{{- end}}
	default:
		logger.Fatal(fmt.Sprintf("NewStub error, contract=%s,version=%s", smc.Message().Contract().Name(), smc.Message().Contract().Version()))
	}

	return nil
}

func NewIBCStub(smc sdk.ISmartContract, logger log.Logger) types.IContractIBCStub {

	switch common.CalcKey(smc.Message().Contract().Name(), smc.Message().Contract().Version()) {
	{{- range $j,$contractName := $.ContractNames}}
	case "{{$contractName}}{{replace (version $j $.Versions)}}":
		return {{getName $j $.PackageNames}}{{replace (version $j $.Versions)}}.NewIBC(logger)
	{{- end}}
	default:
		logger.Fatal(fmt.Sprintf("NewIBCStub error, contract=%s,version=%s", smc.Message().Contract().Name(), smc.Message().Contract().Version()))
	}

	return nil
}
`

type OrgContracts struct {
	OrgIDs         []string
	DirectionNames []string
	ContractNames  []string
	PackageNames   []string
	Versions       []string
}

func res2factory(reses []*parsecode.Result) OrgContracts {

	sLen := len(reses)
	factory := OrgContracts{
		OrgIDs:         make([]string, 0, sLen),
		DirectionNames: make([]string, 0, sLen),
		ContractNames:  make([]string, 0, sLen),
		PackageNames:   make([]string, 0, sLen),
		Versions:       make([]string, 0, sLen),
	}

	for _, res := range reses {
		factory.OrgIDs = append(factory.OrgIDs, res.OrgID)
		factory.DirectionNames = append(factory.DirectionNames, res.DirectionName)
		factory.ContractNames = append(factory.ContractNames, res.ContractName)
		factory.PackageNames = append(factory.PackageNames, res.PackageName)
		factory.Versions = append(factory.Versions, res.Version)
	}

	return factory
}

// GenConStFactory - generate the contract stub factory go source
func GenConStFactory(reses []*parsecode.Result, outDir string) {
	if err := os.MkdirAll(outDir, os.FileMode(0750)); err != nil {
		panic(err)
	}
	filename := filepath.Join(outDir, "contractstubfactory.go")

	funcMap := template.FuncMap{
		"version": func(index int, versions []string) string {
			return versions[index]
		},
		"replace": func(version string) string {
			return strings.Replace(version, ".", "", -1)
		},
		"getName": func(index int, packageNames []string) string {
			return packageNames[index]
		},
		"getOrgID": func(index int, orgIDs []string) string {
			return orgIDs[index]
		},
	}

	tmpl, err := template.New("contractStubFactory").Funcs(funcMap).Parse(templateText)
	if err != nil {
		panic(err)
	}

	factory := res2factory(reses)

	var buf bytes.Buffer

	if err = tmpl.Execute(&buf, factory); err != nil {
		panic(err)
	}

	if err := parsecode.FmtAndWrite(filename, buf.String()); err != nil {
		panic(err)
	}
}
