package smcbuilder

import (
	"blockchain/algorithm"
	"blockchain/common/statedbhelper"
	"blockchain/smccheck"
	"blockchain/smccheck/gen"
	"blockchain/smcsdk/sdk/crypto/sha3"
	"blockchain/smcsdk/sdk/jsoniter"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl/helper"
	"bytes"
	"common/dockerlib"
	"common/fs"
	"common/sig"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/tendermint/tmlibs/log"
)

const ThirdPartyContract = "smcrunsvc_v1.0_3dcontract"

// GolangImageTag 編譯我們用 golang 的 1.11.1 版本
const GolangImageTag = "golang:alpine"
const AlpineImage = "alpine:latest"
const goInstallShell = `#!/bin/sh

a=$(go install ./cmd/smcrunsvc 2>&1)

if [[ $? -eq 0 ]]; then
    echo "success:" > log
else
    echo "fail:" > log
fi

echo ${a} >> log
`
const sha256Shell = `#!/bin/sh
sha256sum smcrunsvc > smcrunsvc.sha2
`

// Builder 是一個 Service 外部參數只需要 WorkDir
type Builder struct {
	Logger  log.Logger
	WorkDir string
	lib     *dockerlib.DockerLib
}

// Signature sig for contract code
type Signature struct {
	PubKey    string `json:"pubkey"`
	Signature string `json:"signature"`
}

var (
	builder  *Builder
	initOnce sync.Once
)

// GetInstance 返回單例 Builder
func GetInstance() *Builder {
	return builder
}

// Init 初始化
func Init(l log.Logger, p string) *Builder {
	initOnce.Do(func() {
		builder = &Builder{
			Logger:  l,
			WorkDir: p,
		}
		builder.lib = dockerlib.GetDockerLib()
	})
	return builder
}

// GetContractDllPath 直接一步編譯，成功返回全路徑，不成功返回錯誤描述(可以認爲不是/開頭就是失敗了)
func (b *Builder) GetContractDllPath(transID int64, txID int64, orgID string) (string, error) {

	// 1.0第三方合约docker路径
	if orgID == ThirdPartyContract {
		targetBinPath := filepath.Join(b.WorkDir, orgID, "bin")
		return filepath.Join(targetBinPath, "smcrunsvc"), nil
	}

	b.Logger.Debug("GetContractDllPath entered:", "transID", transID, "txID", txID, "orgID", orgID)
	blh := helper.BlockChainHelper{}
	genesisOrgID := blh.CalcOrgID("genesis")
	orgCodeHash := statedbhelper.GetOrgCodeHash(transID, txID, orgID)

	if orgID == genesisOrgID && len(orgCodeHash) == 0 {
		b.Logger.Debug("GetContractDllPath genesis.")
		genesisPath, err := b.expandGenesisContract()
		if err != nil {
			return "", errors.New(err.Error())
		}

		genesisBinPath := filepath.Join(b.WorkDir, "bin", orgID, "genesis")
		targetBinPath := filepath.Join(genesisBinPath, "bin")
		err = os.MkdirAll(targetBinPath, 0750)
		if err != nil {
			panic(err)
		}

		err = b.runDocker(genesisPath, targetBinPath)
		if err != nil {
			return "", err
		}

		err = os.RemoveAll(genesisPath)
		if err != nil {
			b.Logger.Error("Can not remove genesis contract code", "dir", genesisPath)
		}
		if runtime.GOOS == "windows" {
			return filepath.Join(targetBinPath, "smcrunsvc.exe"), nil
		}
		return filepath.Join(targetBinPath, "smcrunsvc"), nil
	} /*else if len(orgCodeHash) == 0 {
		b.Logger.Error("BuildContract can't get orgCodeHash", "orgID", orgID)
		return "", errors.New("BuildContract can't get orgCodeHash")
	}*/

	var genesisOrgHashStr string
	if orgID != genesisOrgID {
		genesisOrgHashStr = string(statedbhelper.GetOrgCodeHash(transID, txID, genesisOrgID))
	}
	smcsvcFilePath := algorithm.CalcCodeHash(genesisOrgHashStr + string(orgCodeHash))
	smcsvcFilePathStr := hex.EncodeToString(smcsvcFilePath)
	resPath := filepath.Join(b.WorkDir, "bin", orgID, smcsvcFilePathStr, "smcrunsvc")
	ok := fs.CheckSha2(resPath)
	if ok {
		return resPath, nil
	}

	buildPath := filepath.Join(b.WorkDir, "build")
	err := os.MkdirAll(buildPath, 0750)
	if err != nil {
		panic(err)
	}

	tempDirName, err := ioutil.TempDir(buildPath, orgID)
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDirName)

	codePath := filepath.Join(tempDirName, "src", "contract", orgID, "code")
	targetBinDir := filepath.Join(b.WorkDir, "bin", orgID)

	err = os.MkdirAll(targetBinDir, 0750)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(codePath, 0750)
	if err != nil {
		panic(err)
	}
	worldAppState := statedbhelper.GetWorldAppState(transID, txID)

	if orgID != genesisOrgID {
		genesisCodePath := filepath.Join(tempDirName, "src", "contract", genesisOrgID, "code")
		err = os.MkdirAll(genesisCodePath, 0750)
		if err != nil {
			panic(err)
		}
		b.expandOldCode(transID, txID, worldAppState.BlockHeight+1, genesisOrgID, genesisCodePath)
	}

	_, contractInfoList := b.expandOldCode(transID, txID, worldAppState.BlockHeight+1, orgID, codePath)
	targetBinPath := filepath.Join(targetBinDir, smcsvcFilePathStr)
	err = os.MkdirAll(targetBinPath, 0750)
	if err != nil {
		panic(err)
	}

	_, genErr := smccheck.Gen(filepath.Join(tempDirName, "src", "contract"), "", "", contractInfoList)
	if genErr.ErrorCode != types.CodeOK {
		return "", errors.New(genErr.ErrorDesc)
	}

	err = b.runDocker(tempDirName, targetBinPath)
	if err != nil {
		return "", err
	}

	sha2ok := b.genSha2(targetBinPath, "smcrunsvc")
	if !sha2ok {
		return "", errors.New("can not create sha256 file")
	}

	if runtime.GOOS == "windows" {
		return filepath.Join(targetBinPath, "smcrunsvc.exe"), nil
	}
	return filepath.Join(targetBinPath, "smcrunsvc"), nil
}

// BuildContract 直接一步編譯，最新的合約是通過參數傳進來，因爲還沒上鏈，返回合約方法列表/exe路徑/出錯信息
// nolint gocyclo
func (b *Builder) BuildContract(transID int64, txID int64, contractMeta std.ContractMeta) std.BuildResult {
	b.Logger.Debug("BuildContract entered:", "transID", transID, "txID", txID)
	b.Logger.Trace("contractMeta", contractMeta)
	blh := helper.BlockChainHelper{}
	genesisOrgID := blh.CalcOrgID("genesis")

	err := b.checkSign(transID, txID, contractMeta.CodeDevSig, contractMeta.CodeOrgSig, contractMeta.CodeHash, contractMeta.OrgID, genesisOrgID)
	if err != nil {
		return std.BuildResult{
			Code:        types.ErrInvalidParameter,
			Error:       "check sign error," + err.Error(),
			Methods:     nil,
			Interfaces:  nil,
			OrgCodeHash: nil,
		}
	}

	if !b.checkCodeHash(contractMeta) {
		return std.BuildResult{
			Code:        types.ErrInvalidParameter,
			Error:       "check code hash failed.",
			Methods:     nil,
			Interfaces:  nil,
			OrgCodeHash: nil,
		}
	}

	buildPath := filepath.Join(b.WorkDir, "build")
	err = os.MkdirAll(buildPath, 0750)
	if err != nil {
		panic(err)
	}

	tempDirName, err := ioutil.TempDir(buildPath, contractMeta.OrgID)
	if err != nil {
		panic(err)
	}

	codePath := filepath.Join(tempDirName, "src", "contract", contractMeta.OrgID, "code")
	targetBinDir := filepath.Join(b.WorkDir, "bin", contractMeta.OrgID)

	err = os.MkdirAll(targetBinDir, 0750)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(codePath, 0750)
	if err != nil {
		panic(err)
	}
	worldAppState := statedbhelper.GetWorldAppState(transID, txID)

	if contractMeta.OrgID != genesisOrgID {
		genesisCodePath := filepath.Join(tempDirName, "src", "contract", genesisOrgID, "code")
		err = os.MkdirAll(genesisCodePath, 0750)
		if err != nil {
			panic(err)
		}
		b.expandOldCode(transID, txID, worldAppState.BlockHeight+1, genesisOrgID, genesisCodePath)
	}
	// 取舊的，展開舊的
	codeHashListStr, contractInfoList := b.expandOldCode(transID, txID, worldAppState.BlockHeight+1, contractMeta.OrgID, codePath)

	contractInfoList = append(contractInfoList, gen.ContractInfo{
		Name:         contractMeta.Name,
		Version:      contractMeta.Version,
		EffectHeight: contractMeta.EffectHeight,
		LoseHeight:   contractMeta.LoseHeight,
	})
	// 展開新的
	newCodePath := filepath.Join(codePath, contractMeta.Name, "v"+contractMeta.Version, contractMeta.Name)
	err = os.MkdirAll(newCodePath, 0750)
	if err != nil {
		panic(err)
	}
	err = fs.UnTarGz(newCodePath, bytes.NewReader(contractMeta.CodeData), b.Logger)
	if err != nil {
		b.Logger.Error("BuildContract can't extract code.tar.gz", "err", err)
		return std.BuildResult{Code: types.ErrInvalidParameter, Error: "BuildContract can't extract code tar.gz file."}
	}

	orgCodeHash := algorithm.CalcCodeHash(codeHashListStr + string(contractMeta.CodeHash))
	var genesisOrgHashStr string
	if contractMeta.OrgID != genesisOrgID {
		genesisOrgHashStr = string(statedbhelper.GetOrgCodeHash(transID, txID, genesisOrgID))
	}
	smcsvcFilePath := algorithm.CalcCodeHash(genesisOrgHashStr + string(orgCodeHash))
	smcsvcFilePathStr := hex.EncodeToString(smcsvcFilePath)
	targetBinPath := filepath.Join(targetBinDir, smcsvcFilePathStr)
	err = os.MkdirAll(targetBinPath, 0750)
	if err != nil {
		panic(err)
	}

	genResult, genErr := smccheck.Gen(filepath.Join(tempDirName, "src", "contract"), contractMeta.Name, contractMeta.Version, contractInfoList)
	if genErr.ErrorCode != types.CodeOK {
		return std.BuildResult{Code: genErr.ErrorCode, Error: genErr.Error()}
	}

	if !b.isBuilt(targetBinPath) {
		err = b.runDocker(tempDirName, targetBinPath)
		if err != nil {
			os.RemoveAll(targetBinPath)
			return std.BuildResult{Code: types.ErrInvalidParameter, Error: err.Error()}
		}

		genSha2ok := b.genSha2(targetBinPath, "smcrunsvc")
		if !genSha2ok {
			os.RemoveAll(targetBinPath)
			return std.BuildResult{Code: types.ErrInvalidParameter, Error: "Build contract failed. Can not gen sha2 file for smcrunsvc"}
		}
	}
	for _, v := range genResult {
		if v.ContractName == contractMeta.Name && v.Version == contractMeta.Version && v.OrgID == contractMeta.OrgID {
			result := std.BuildResult{
				Code:        types.CodeOK,
				Error:       "",
				Methods:     v.Methods,
				Interfaces:  v.Interfaces,
				IBCs:        v.IBCs,
				OrgCodeHash: orgCodeHash,
			}
			if len(v.Mine) == 1 {
				if v.OrgID == genesisOrgID {
					result.Mine = v.Mine
				} else {
					os.RemoveAll(targetBinPath)
					return std.BuildResult{Code: types.ErrInvalidParameter,
						Error: "Only genesis organization can use Mine func"}
				}
			} else if len(v.Mine) > 1 {
				os.RemoveAll(targetBinPath)
				return std.BuildResult{Code: types.ErrInvalidParameter,
					Error: "Must only one or zero Mine func"}
			}
			os.RemoveAll(tempDirName)
			return result
		}
	}
	os.RemoveAll(targetBinPath)

	return std.BuildResult{Code: types.ErrInvalidParameter,
		Error: "Build contract failed. Name or version or orgID not match codeData"}
}

func (b *Builder) isBuilt(targetBinPath string) bool {
	filePath := filepath.Join(targetBinPath, "smcrunsvc")
	_, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	checkOk := fs.CheckSha2(filePath)
	if !checkOk {
		return false
	}
	return true
}

func (b *Builder) expandGenesisContract() (string, error) {
	b.Logger.Debug("Expand genesis contract code.")
	genesisTarPath := ""
	configFile := viper.ConfigFileUsed()
	configFile = strings.Replace(configFile, "\\", "/", -1)
	genesisTarPath = strings.Replace(configFile, path.Base(configFile), "", 1)

	fi, err := ioutil.ReadDir(genesisTarPath)
	if err != nil {
		panic(err)
	}
	genesisTarList := make([]string, 0)
	for _, v := range fi {
		if !v.IsDir() && strings.HasPrefix(v.Name(), "genesis-smcrunsvc") {
			genesisTarList = append(genesisTarList, v.Name())
		}
	}
	if len(genesisTarList) != 1 {
		b.Logger.Error("Must only one genesis contract tar.gz in " + genesisTarPath)
		return "", errors.New("Must only one genesis contract tar.gz in " + genesisTarPath)
	}
	genesisTarPath = filepath.Join(genesisTarPath, genesisTarList[0])
	b.Logger.Debug("genesisTarPath", "genesisTarPath", genesisTarPath)
	data, err := ioutil.ReadFile(genesisTarPath)
	if err != nil {
		panic(err)
	}
	err = fs.UnTarGz(b.WorkDir, bytes.NewReader(data), b.Logger)
	if err != nil {
		panic(err)
	}
	return filepath.Join(b.WorkDir, "genesis"), nil
}

func (b *Builder) expandOldCode(transID, txID, currentBlockHeight int64, orgID, codePath string) (string, []gen.ContractInfo) {
	contractInfoList := []gen.ContractInfo{}
	codeHashListStr := ""
	contractAddrList := statedbhelper.GetContracts(transID, txID, orgID)
	for _, v := range contractAddrList {
		meta := statedbhelper.GetContractMeta(transID, txID, v)
		if (meta.LoseHeight <= currentBlockHeight) && meta.LoseHeight != 0 || len(meta.CodeData) == 0 {
			continue
		}
		codeHashListStr += string(statedbhelper.GetContractCodeHash(transID, txID, v))

		contractInfo := gen.ContractInfo{
			Name:         meta.Name,
			Version:      meta.Version,
			EffectHeight: meta.EffectHeight,
			LoseHeight:   meta.LoseHeight,
		}
		contractInfoList = append(contractInfoList, contractInfo)

		newCodePath := filepath.Join(codePath, meta.Name, "v"+meta.Version, meta.Name)
		err := os.MkdirAll(newCodePath, 0750)
		if err != nil {
			panic(err)
		}
		err = fs.UnTarGz(newCodePath, bytes.NewReader(meta.CodeData), b.Logger)
		if err != nil {
			panic(err)
		}
	}
	return codeHashListStr, contractInfoList
}

func (b *Builder) runDocker(buildPath, targetPath string) error {

	if runtime.GOOS == "windows" {
		params := dockerlib.DockerRunParams{
			Cmd: []string{"go", "install"},
			Env: []string{"GOPATH=" + buildPath + ";" + b.WorkDir + "\\sdk" + ";" + b.WorkDir + "/thirdparty",
				"CGO_ENABLED=0", "GOCACHE=" + buildPath, "GOBIN=" + targetPath},
			WorkDir:    buildPath,
			NeedRemove: true,
			NeedOut:    true,
			NeedWait:   true,
		}

		ok, err := b.lib.Run(GolangImageTag, "", &params)
		b.Logger.Debug("Run docker result", "dockerRunResult", ok)
		if !ok {
			panic(err)
		}
		ret := params.FirstOutput
		if ok && ret == "" {
			b.Logger.Debug("Run docker result output", "firstOutput", ret)
			return nil
		}
		// 如果包含 .go:88:88: 类似的字符串，说明是代码编译不通过，否则需要 panic
		regOk := b.checkRegex(ret, "^*.go:[0-9]+:[0-9]+:*")
		if !regOk {
			panic(ret)
		}

		b.Logger.Debug("Run docker result output", "output", ret)
		return errors.New(ret)
	}

	fi, err := os.Create(buildPath + "/src/go-install.sh")
	if err != nil {
		panic(err)
	}
	defer fi.Close()
	_, err = fi.Write([]byte(goInstallShell))
	if err != nil {
		panic(err)
	}

	params := dockerlib.DockerRunParams{
		Cmd:     []string{"/bin/sh", "./go-install.sh"},
		Env:     []string{"GOPATH=/build:/blockchain/sdk:/blockchain/thirdparty", "CGO_ENABLED=0"},
		WorkDir: "/build/src",
		Mounts: []dockerlib.Mounts{
			{
				Source:      targetPath,
				Destination: "/build/bin",
			},
			{
				Source:      b.WorkDir + "/thirdparty",
				Destination: "/blockchain/thirdparty",
				ReadOnly:    true,
			},
			{
				Source:      b.WorkDir + "/sdk",
				Destination: "/blockchain/sdk",
				ReadOnly:    true,
			},
			{
				Source:      buildPath,
				Destination: "/build",
			},
		},
		NeedRemove: true,
		NeedOut:    true,
		NeedWait:   true,
	}
	ok, err := b.lib.Run(GolangImageTag, "", &params)
	b.Logger.Debug("Run docker result", "dockerRunResult", ok)
	if !ok {
		panic(err)
	}

	fb, err := ioutil.ReadFile(filepath.Join(buildPath, "src", "log"))
	if err != nil {
		panic(err)
	}
	ret := string(fb)
	if ok && strings.HasPrefix(ret, "success") {
		b.Logger.Debug("Run docker result output", "output", ret)
		return nil
	}

	// 如果包含 .go:88:88: 类似的字符串，说明是代码编译不通过，否则需要 panic
	regOk := b.checkRegex(ret, "^*.go:[0-9]+:[0-9]+:*")
	if !regOk {
		panic(ret)
	}

	b.Logger.Debug("Run docker result output", "output", ret)
	return errors.New(ret)
}

func (b *Builder) checkCodeHash(contractMeta std.ContractMeta) bool {
	hashStr := hex.EncodeToString(sha3.Sum256(contractMeta.CodeData))
	hashStr1 := hex.EncodeToString(contractMeta.CodeHash)

	if hashStr != hashStr1 {
		b.Logger.Error("Code hash mismatch", "code", hashStr, "doc", hashStr1)
		return false
	}
	return true
}

// nolint gocyclo
func (b *Builder) checkSign(transID, txID int64, codeDevSig, codeOrgSig, codeHash []byte, orgID, genesisOrgID string) error {

	sigDevMap := map[string]string{}
	var codeDevSigStr string

	err := jsoniter.Unmarshal(codeDevSig, &codeDevSigStr)
	if err != nil {
		b.Logger.Error("Unmarshal Fail", "sig", "Dev")
		return err
	}
	err = jsoniter.Unmarshal([]byte(codeDevSigStr), &sigDevMap)
	if err != nil {
		b.Logger.Error("Unmarshal to map Fail", "sig", "Dev")
		return err
	}

	pubKey, err := hex.DecodeString(sigDevMap["pubkey"])
	if err != nil {
		return fmt.Errorf("UnmarshalJSON \"%v\" failed, %v", sigDevMap["pubkey"], err.Error())
	}
	codeSig, err := hex.DecodeString(sigDevMap["signature"])
	if err != nil {
		return fmt.Errorf("UnmarshalJSON \"%v\" failed, %v", sigDevMap["signature"], err.Error())
	}

	ok, err := sig.Verify(pubKey, codeHash, codeSig)
	if err != nil {
		return errors.New("check devSig failed, error:" + err.Error())
	}
	if !ok {
		return fmt.Errorf("CheckDevSign Failed")
	}

	sigOrgMap := map[string]string{}
	var codeOrgSigStr string
	err = jsoniter.Unmarshal(codeOrgSig, &codeOrgSigStr)
	if err != nil {
		b.Logger.Error("Unmarshal Fail", "sig", "Dev")
		return err
	}
	err = jsoniter.Unmarshal([]byte(codeOrgSigStr), &sigOrgMap)
	if err != nil {
		b.Logger.Error("CheckOrgSign Fail", "sig", "Dev")
		return err
	}
	orgSigPubKey := sigOrgMap["pubkey"]
	// 如果创世组织部署合约也要签名组织签名公钥是否正确，但是创世的时候不验证
	if genesisOrgID != orgID || (genesisOrgID == orgID && len(statedbhelper.GetOrgCodeHash(0, 0, orgID)) != 0) {
		orgSigned := false
		signers := statedbhelper.GetOrgSigners(transID, txID, orgID)
		if signers == nil || len(signers) == 0 {
			return fmt.Errorf("Can not get current org signers.")
		}

		for _, v := range signers {
			if strings.ToUpper(hex.EncodeToString(v)) == strings.ToUpper(orgSigPubKey) {
				orgSigned = true
			}
		}
		if !orgSigned {
			return errors.New("Org signers not sign.")
		}
	}

	OrgPubKey, err := hex.DecodeString(orgSigPubKey)
	if err != nil {
		return fmt.Errorf("UnmarshalJSON \"%v\" failed, %v", sigDevMap["pubkey"], err.Error())
	}

	orgSig, err := hex.DecodeString(sigOrgMap["signature"])
	if err != nil {
		return fmt.Errorf("UnmarshalJSON \"%v\" failed, %v", sigDevMap["signature"], err.Error())
	}

	orgS := new(Signature)
	err = jsoniter.Unmarshal([]byte(codeDevSigStr), orgS)
	if err != nil {
		return errors.New("check orgSig failed, unmarshal error:" + err.Error())
	}

	orgSigData, err := hex.DecodeString(orgS.Signature)
	if err != nil {
		return errors.New("check orgSig failed, hex decode string failed:" + err.Error())
	}

	ok, err = sig.Verify(OrgPubKey, orgSigData, orgSig)
	if err != nil {
		return errors.New("check orgSig failed, VerifySign error:" + err.Error())
	}
	if !ok {
		return errors.New("check orgSig failed")
	}

	return nil
}

func (b *Builder) genSha2(tarPath, fileName string) bool {
	sha2 := ""
	shaFile, err := os.Create(filepath.Join(tarPath, fileName+".sha2"))
	if err != nil {
		panic(err)
	}
	defer shaFile.Close()
	if runtime.GOOS == "windows" {
		fb, err := ioutil.ReadFile(filepath.Join(tarPath, fileName+".exe"))
		if err != nil {
			panic(err)
		}
		sha2 = hex.EncodeToString(sha3.Sum256(fb))
		_, err = shaFile.Write([]byte(sha2))
		if err != nil {
			panic(err)
		}
		return true
	} else {
		fi, err := os.Create(tarPath + "/sha256sum.sh")
		if err != nil {
			panic(err)
		}
		defer fi.Close()
		_, err = fi.Write([]byte(sha256Shell))
		if err != nil {
			panic(err)
		}

		params := dockerlib.DockerRunParams{
			Cmd:     []string{"/bin/sh", "./sha256sum.sh"},
			WorkDir: "/smcrunsvc",
			Mounts: []dockerlib.Mounts{
				{
					Source:      tarPath,
					Destination: "/smcrunsvc",
				},
			},
			NeedRemove: true,
			NeedOut:    true,
			NeedWait:   true,
		}

		ok, err := b.lib.Run(AlpineImage, "", &params)
		b.Logger.Debug("Run gen sha2 docker", "dockerRunResult", ok)
		if !ok {
			panic(err)
		}
		return true
	}
}

func (b *Builder) checkRegex(obj string, regex string) bool {
	r, e := regexp.Compile(regex)
	if e != nil {
		return false
	}
	return r.MatchString(obj)
}
