package sidechain

import (
	"common/fs"
	"common/jsoniter"
	"encoding/hex"
	"errors"
	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/tendermint/types"
	pvm "github.com/tendermint/tendermint/types/priv_validator"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	//returns: config.GenesisFile(), config.ConfigFilePath(),
	//        config.DBDir(),config.ValidatorsFile(),config.PrivValidatorFile()
	ConfigPathFunc func() (string, string, string, string, string)
)

type SideChain struct {
	GenesisInfo *abci.SideChainGenesis
	TempPath    string
}

var cdc = amino.NewCodec()

func init() {
	crypto.RegisterAmino(cdc)
}

// NewSideChain new SideChain instance
func NewSideChain(genesisInfo *abci.SideChainGenesis) *SideChain {
	sc := &SideChain{
		GenesisInfo: genesisInfo,
		TempPath:    filepath.Join(configPath(), ".sidechaintemp"),
	}
	return sc
}

// Genesis side chain genesis, copy prepare file to desitination.
func (sc *SideChain) Genesis() error {
	// copy all side chain genesis file
	desContractPath := configPath()
	err := filepath.Walk(sc.TempPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && info.Name() != "needgenesis" {
			_, err := fs.CopyFile(path, filepath.Join(desContractPath, info.Name()))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// remove tmcore db
	dbDir := dbDir()
	if err := os.RemoveAll(dbDir); err != nil {
		return err
	}

	// remove addrbook.json
	addrBook := filepath.Join(configPath(), "addrbook.json")
	if err := os.RemoveAll(addrBook); err != nil {
		return err
	}
	return os.RemoveAll(sc.TempPath)
}

// NeedSCGenesis return true if has side chain need genesis
func (sc *SideChain) NeedSCGenesis() bool {
	exist, err := fs.PathExists(filepath.Join(sc.TempPath, "needgenesis"))
	if err != nil {
		panic(err)
	}

	return exist
}

// PrepareSCGenesis generate temp files for side chain genesis.
func (sc *SideChain) PrepareSCGenesis() error {
	var err error

	if err = os.MkdirAll(sc.TempPath, 0750); err != nil {
		panic(err)
	}

	if err = sc.genContratTarGZ(); err != nil {
		return err
	}

	if err = sc.genGenesisJson(); err != nil {
		return err
	}

	if err = sc.genValidatorJson(); err != nil {
		return err
	}

	if err = sc.genPrivValidatorJson(); err != nil {
		return err
	}

	if err = sc.genConfigToml(); err != nil {
		return err
	}

	if err = sc.delForksFiles(); err != nil {
		return err
	}

	if _, err = os.Create(filepath.Join(sc.TempPath, "needgenesis")); err != nil {
		return err
	}

	return nil
}

// CopyGenesisFiles copy config files to genesis dir
func (sc *SideChain) CopyGenesisFiles() error {
	tmPath := filepath.Dir(configPath())
	genesisPath := filepath.Join(tmPath, "genesis")

	chainID, err := sc.getChainID()
	if err != nil {
		return err
	}

	genesisPath = filepath.Join(genesisPath, chainID)
	if err := os.MkdirAll(genesisPath, 0750); err != nil {
		return err
	}

	err = fs.CopyDir(configPath(), genesisPath, "",
		"node_key.json|priv_validator.json|addrbook.json|config.toml")
	if err != nil {
		return err
	}

	err = filepath.Walk(genesisPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() &&
			info.Name() == "genesis.json" || info.Name() == "genesis.json.sig" || info.Name() == "validators.json" {
			newPath := filepath.Join(genesisPath, chainID+"-"+filepath.Base(path))
			return os.Rename(path, newPath)
		}
		return nil
	})

	return err
}

func (sc *SideChain) getChainID() (string, error) {
	b, err := ioutil.ReadFile(genesisFile())
	if err != nil {
		return "", err
	}

	type genesisInfo struct {
		ChainID string `json:"chain_id"`
	}

	g := new(genesisInfo)
	err = jsoniter.Unmarshal(b, g)
	if err != nil {
		return "", err
	}
	return g.ChainID, nil
}

func (sc *SideChain) delForksFiles() error {
	currentPath, err := os.Executable()
	if err != nil {
		return err
	}

	currentDir := path.Dir(currentPath)
	if err = os.RemoveAll(filepath.Join(currentDir, "tendermint-forks.json")); err != nil {
		return err
	}

	if err = os.RemoveAll(filepath.Join(currentDir, "tendermint-forks.json.sig")); err != nil {
		return err
	}
	return nil
}

func (sc *SideChain) genContratTarGZ() error {
	for _, v := range sc.GenesisInfo.ContractData {
		fileName := v.Name + "-" + v.Version + ".tar.gz"
		fi, err := os.Create(filepath.Join(sc.TempPath, fileName))
		if err != nil {
			return err
		}
		if _, err = fi.Write(v.CodeData); err != nil {
			return err
		}
		_ = fi.Close()
	}
	return nil
}

func (sc *SideChain) genGenesisJson() error {
	gensisFileName := filepath.Base(genesisFile())
	genesisPath := filepath.Join(sc.TempPath, gensisFileName)
	fi, err := os.Create(genesisPath)
	if err != nil {
		return err
	}

	defer func() {
		_ = fi.Close()
	}()

	_, err = fi.WriteString(sc.GenesisInfo.GenesisInfo)
	if err != nil {
		return err
	}

	genesisBlob, e := ioutil.ReadFile(genesisPath)
	if e != nil {
		return err
	}

	// 对 genesis.json 签名并生成 genesis.json.sig
	p := privValidatorFile()
	pv := pvm.LoadFilePV(p)
	sign := pv.PrivKey.Sign(genesisBlob)

	type Signature struct {
		PubKey    string `json:"pubkey"`
		Signature string `json:"signature"`
	}
	pk := pv.PubKey.(crypto.PubKeyEd25519)
	sn := sign.(crypto.SignatureEd25519)
	signature := Signature{
		PubKey:    hex.EncodeToString(pk[:]),
		Signature: hex.EncodeToString(sn[:]),
	}

	signByte, err := jsoniter.Marshal(signature)
	if err != nil {
		return err
	}
	fi, err = os.Create(genesisPath + ".sig")
	if err != nil {
		return err
	}
	defer func() {
		_ = fi.Close()
	}()

	_, err = fi.WriteString(string(signByte))
	if err != nil {
		return err
	}
	return nil
}

func (sc *SideChain) genValidatorJson() error {
	if len(sc.GenesisInfo.Validators) == 0 {
		return errors.New("invalid side chain validator")
	}
	v := sc.GenesisInfo.Validators[0]

	gv := types.GenesisValidator{
		PubKey:     crypto.PubKeyEd25519FromBytes(v.PubKey),
		RewardAddr: v.RewardAddr,
		Power:      int64(v.Power),
		Name:       v.Name,
	}

	result := make([]types.GenesisValidator, 0, 1)
	result = append(result, gv)

	outByte, err := cdc.MarshalJSONIndent(result, "", "  ")
	if err != nil {
		return err
	}

	p := filepath.Join(sc.TempPath, filepath.Base(validatorsFile()))
	err = ioutil.WriteFile(p, outByte, 0600)
	if err != nil {
		return err
	}

	return nil
}

func (sc *SideChain) genPrivValidatorJson() error {
	privValidatorFile := privValidatorFile()
	tempPrivValJson := filepath.Join(sc.TempPath, filepath.Base(privValidatorFile))

	_, err := fs.CopyFile(privValidatorFile, tempPrivValJson)
	if err != nil {
		return err
	}

	pv := pvm.LoadFilePV(tempPrivValJson)
	pv.LastHeight = 0
	pv.LastRound = 0
	pv.LastStep = 0
	pv.Address = pv.GetPubKey().Address(sc.GenesisInfo.SideChainID)
	pv.LastSignature = nil
	pv.LastSignBytes = nil
	pv.Save()
	return nil
}

func (sc *SideChain) genConfigToml() error {
	_, configFile, _, _, _ := ConfigPathFunc()
	tempConfig := filepath.Join(sc.TempPath, filepath.Base(configFile))

	configContent, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}

	tempFile, err := os.Create(tempConfig)
	if err != nil {
		return err
	}
	defer tempFile.Close()

	configSplit := strings.Split(string(configContent), "\n")
	for _, line := range configSplit {
		if strings.HasPrefix(line, "persistent_peers") {
			if strings.HasSuffix(line, "\r") {
				line = `persistent_peers = ""\r`
			} else {
				line = `persistent_peers = ""`
			}
		}
		line += "\n"
		_, err = tempFile.WriteString(line)
		if err != nil {
			return err
		}
	}

	return nil
}

// ContainsCurrentNode if genesisInfoList contains current node,
// 		return genesis info and true, or else return nil and false
func ContainsCurrentNode(genesisInfoList []*abci.SideChainGenesis) (genesisInfo *abci.SideChainGenesis, ok bool) {
	privValidatorFile := privValidatorFile()
	currentNodePubKey := pvm.LoadFilePV(privValidatorFile).GetPubKey()

	for _, info := range genesisInfoList {
		for _, v := range info.Validators {
			if currentNodePubKey.Equals(crypto.PubKeyEd25519FromBytes(v.PubKey)) {
				genesisInfo = info
				ok = true
				return
			}
		}
	}

	return
}

func genesisFile() string {
	genesisFile, _, _, _, _ := ConfigPathFunc()
	return genesisFile
}

func configPath() string {
	_, configFile, _, _, _ := ConfigPathFunc()
	return filepath.Dir(configFile)
}

func dbDir() string {
	_, _, dbDir, _, _ := ConfigPathFunc()
	return dbDir
}

func validatorsFile() string {
	_, _, _, validatorsFile, _ := ConfigPathFunc()
	return validatorsFile
}

func privValidatorFile() string {
	_, _, _, _, privValidatorFile := ConfigPathFunc()
	return privValidatorFile
}
