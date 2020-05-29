package smccheck

import (
	"fmt"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bcbchain/smccheck/gen"
	"github.com/bcbchain/bcbchain/smccheck/gencmd"
	"github.com/bcbchain/bcbchain/smccheck/genstub"
	"github.com/bcbchain/bcbchain/smccheck/parsecode"
	"github.com/bcbchain/sdk/sdk/std"
	"github.com/bcbchain/sdk/sdk/types"
	"github.com/docker/docker/api/types/versions"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Gen - walk contract path and generate code
func Gen(contractDir, contractName, version string, contractInfoList []gen.ContractInfo) (results []std.GenResult, err types.Error) {

	err.ErrorCode = types.CodeOK

	subDirs, er := ioutil.ReadDir(contractDir)
	if er != nil {
		panic(er)
	} else if len(subDirs) > 2 ||
		(len(subDirs) == 2 &&
			!(subDirs[0].Name() == statedbhelper.GetGenesisOrgID(0, 0) || subDirs[1].Name() == statedbhelper.GetGenesisOrgID(0, 0))) {
		panic("invalid directory")
	}

	// check contractName and version
	if !checkNameAndVersion(subDirs, contractDir, contractName, version) {
		err.ErrorCode = types.ErrInvalidParameter
		err.ErrorDesc = "invalid contractName or version"
		return
	}

	results = make([]std.GenResult, 0)
	var another string
	totalResList := make([]*parsecode.Result, 0)
	for _, dir := range subDirs {

		firstContractPath := filepath.Join(contractDir, dir.Name())
		codePath := filepath.Join(firstContractPath, "code")
		contractDirs, er := ioutil.ReadDir(codePath)
		if er != nil {
			panic(er)
		}

		//packageNameMap := make(map[string]struct{})
		//contractNameMap := make(map[string]struct{})
		for _, contractPath := range contractDirs {
			var resList []*parsecode.Result
			resList, err = checkAndAutoGen(filepath.Join(codePath, contractPath.Name()))
			if err.ErrorCode != types.CodeOK {
				return
			}

			// get methods and interfaces
			results = append(results, getGenResult(resList)...)

			totalResList = append(totalResList, resList...)
		}

		// generate contract stub factory
		stubPath := filepath.Join(contractDir, dir.Name()+"/stub")
		if another == "" {
			another = stubPath
		} else {
			genstub.GenConStFactory(totalResList, another)
		}
		genstub.GenConStFactory(totalResList, stubPath)

		for _, res := range totalResList {
			for index := range res.ImportContracts {
				if res.OrgID == dir.Name() {
					inPath := filepath.Join(filepath.Join(filepath.Join(codePath, res.DirectionName), "v"+res.Version), res.DirectionName)
					er := gen.GenImport(inPath, res, totalResList, contractInfoList, index)
					if er != nil {
						err.ErrorCode = types.ErrInvalidParameter
						err.ErrorDesc = er.Error()
						return
					}
				}
			}
		}
	}

	// generate stub common
	genstub.GenStubCommon(filepath.Join(contractDir, "stubcommon"))

	// generate cmd
	stubName := subDirs[0].Name()
	if len(subDirs) == 2 {
		if subDirs[0].Name() == statedbhelper.GetGenesisOrgID(0, 0) {
			stubName = subDirs[1].Name()
		}
	}
	gencmd.GenCmd(filepath.Dir(contractDir), stubName)

	return
}

// checkAndAutoGen - generate auto gen code and stub code
func checkAndAutoGen(contractPath string) (resList []*parsecode.Result, err types.Error) {

	versionDirs, er := ioutil.ReadDir(contractPath)
	if er != nil {
		panic(er)
	}

	resList = make([]*parsecode.Result, 0, len(versionDirs))
	for _, versionDir := range versionDirs {
		versionPath := filepath.Join(contractPath, versionDir.Name())
		var secDir []os.FileInfo
		secDir, er = ioutil.ReadDir(versionPath)
		if er != nil {
			panic(er)
		} else if len(secDir) != 1 {
			panic("invalid path " + versionPath)
		}

		secPath := filepath.Join(versionPath, secDir[0].Name())
		var res *parsecode.Result
		res, err = parsecode.Check(secPath)
		if err.ErrorCode != types.CodeOK {
			return
		}
		res.DirectionName = secDir[0].Name()
		resList = append(resList, res)

		stubPath := filepath.Join(filepath.Dir(filepath.Dir(contractPath)), "stub")
		err = parsecode.CheckVersions(contractPath, res)
		if err.ErrorCode != types.CodeOK {
			return
		}

		// auto gen
		genAutoGenCode(secPath, res)
		genStubCode(stubPath, res)
	}

	return
}

// genAutoGenCode - generate contract assist code
func genAutoGenCode(secPath string, res *parsecode.Result) {

	gen.GenReceipt(secPath, res)

	gen.GenSDK(secPath, res)

	gen.GenStore(secPath, res)

	gen.GenTypes(secPath, res)
}

// genStubCode - generate stub code
func genStubCode(stubPath string, res *parsecode.Result) {
	stubConPath := filepath.Join(stubPath, res.DirectionName)

	genstub.GenMethodStub(res, stubConPath)

	genstub.GenInterfaceStub(res, stubConPath)

	genstub.GenIBCStub(res, stubConPath)

	genstub.GenStFactory(res, stubConPath)
}

// getGenResult - get gen result
func getGenResult(resList []*parsecode.Result) (genResult []std.GenResult) {
	genResult = make([]std.GenResult, 0, len(resList))
	for _, res := range resList {
		item := std.GenResult{}
		item.ContractName = res.ContractName
		item.Version = res.Version
		item.OrgID = res.OrgID
		item.Methods = make([]std.Method, 0, len(res.MFunctions))
		item.Interfaces = make([]std.Method, 0, len(res.IFunctions))
		item.IBCs = make([]std.Method, 0, len(res.IFunctions))
		item.Mine = make([]std.Method, 0, 1)

		for _, function := range res.MFunctions {
			proto := parsecode.CreatePrototype(function.Method)
			method := std.Method{
				Gas:       function.MGas,
				ProtoType: proto,
				MethodID:  fmt.Sprintf("%x", parsecode.CalcMethodID(proto))}

			item.Methods = append(item.Methods, method)
		}

		for _, function := range res.IFunctions {
			proto := parsecode.CreatePrototype(function.Method)
			method := std.Method{
				Gas:       function.IGas,
				ProtoType: proto,
				MethodID:  fmt.Sprintf("%x", parsecode.CalcMethodID(proto))}

			item.Interfaces = append(item.Interfaces, method)
		}

		for _, function := range res.TFunctions {
			proto := parsecode.CreatePrototype(function.Method)
			method := std.Method{
				Gas:       function.TGas,
				ProtoType: proto,
				MethodID:  fmt.Sprintf("%x", parsecode.CalcMethodID(proto))}

			item.IBCs = append(item.IBCs, method)
		}

		if res.IsExistMine == true {
			proto := parsecode.CreatePrototype(res.Mine.Method)
			item.Mine = append(item.Mine, std.Method{
				ProtoType: proto,
				MethodID:  fmt.Sprintf("%x", parsecode.CalcMethodID(proto)),
			})
		}

		genResult = append(genResult, item)
	}

	return
}

func checkNameAndVersion(subDirs []os.FileInfo, contractDir, contractName, version string) bool {
	if len(contractName) == 0 && len(version) == 0 {
		return true
	}

	for _, dir := range subDirs {
		firstContractPath := filepath.Join(contractDir, dir.Name())
		codePath := filepath.Join(firstContractPath, "code")
		contractDirs, er := ioutil.ReadDir(codePath)
		if er != nil {
			panic(er)
		}

		for _, contractPath := range contractDirs {
			if contractPath.Name() == contractName {
				lv, err := getLatestVersion(codePath, contractPath.Name())
				if err != nil {
					panic(err)
				}

				if version == lv {
					return true
				}
			}
		}
	}

	return false
}

func getLatestVersion(root, contractName string) (string, error) {

	versionDirs, er := ioutil.ReadDir(filepath.Join(root, contractName))
	if er != nil {
		panic(er)
	}

	var latestVer string
	for _, v := range versionDirs {
		path := filepath.Join(root, contractName, v.Name(), contractName)
		tempVer, err := getVersion(path)
		if err != nil {
			return "", err
		}

		if versions.GreaterThan(tempVer, latestVer) {
			latestVer = tempVer
		}
	}

	return latestVer, nil
}

func getVersion(path string) (string, error) {
	var v string
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			contents, err := ioutil.ReadFile(path)
			if err != nil {
				panic(err)
			}

			contentStr := string(contents)
			contentSplit := strings.Split(contentStr, "\n")
			if len(contentSplit) == 1 {
				contentSplit = strings.Split(contentStr, "\r\n")
			}

			for _, line := range contentSplit {
				if strings.HasPrefix(line, "//@:version:") {
					v = line[len("//@:version:"):]
					break
				}
			}
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return v, nil
}
