package types

import (
	"blockchain/smcsdk/sdk/bn"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"common/sig"

	cfg "github.com/tendermint/tendermint/config"

	"github.com/tendermint/go-crypto"
	cmn "github.com/tendermint/tmlibs/common"
)

//------------------------------------------------------------
// core types for a genesis definition

// GenesisValidator is an initial validator.
type GenesisValidator struct {
	RewardAddr string        `json:"reward_addr"`
	PubKey     crypto.PubKey `json:"pub_key,omitempty"` // No Key In Genesis File,so omit empty
	Power      int64         `json:"power"`
	Name       string        `json:"name"`
}

// GenesisDoc defines the initial conditions for a tendermint blockchain, in particular its validator set.
type GenesisDoc struct {
	GenesisTime     time.Time          `json:"genesis_time"`
	ChainID         string             `json:"chain_id"`
	ChainVersion    string             `json:"chain_version"`
	ConsensusParams *ConsensusParams   `json:"consensus_params,omitempty"`
	Validators      []GenesisValidator `json:"validators"`
	AppHash         cmn.HexBytes       `json:"app_hash"`
	AppStateJSON    json.RawMessage    `json:"app_state,omitempty"`
}

type SideChainOrg struct {
	OrgName string `json:"orgName"`
	Owner   string `json:"owner"`
}

type AccountInfo struct {
	Address string    `json:"address"`
	Balance bn.Number `json:"balance"`
}

// AppState returns raw application state.
// TODO: replace with AppState field during next breaking release (0.18)
func (genDoc *GenesisDoc) AppState() json.RawMessage {
	return genDoc.AppStateJSON
}

// SaveAs is a utility method for saving GenesisDoc as a JSON file.
func (genDoc *GenesisDoc) SaveAs(file string) error {
	genDocBytes, err := cdc.MarshalJSONIndent(genDoc, "", "  ")
	if err != nil {
		return err
	}
	return cmn.WriteFile(file, genDocBytes, 0644)
}

// ValidatorHash returns the hash of the validator set contained in the GenesisDoc
func (genDoc *GenesisDoc) ValidatorHash() []byte {
	vals := make([]*Validator, len(genDoc.Validators))
	for i, v := range genDoc.Validators {
		if v.Power < 0 {
			v.Power = 0
		}
		vals[i] = NewValidator(v.PubKey, uint64(v.Power), v.RewardAddr, v.Name)
	}
	vset := NewValidatorSet(vals)
	return vset.Hash()
}

// ValidateAndComplete checks that all necessary fields are present
// and fills in defaults for optional fields left empty
func (genDoc *GenesisDoc) ValidateAndComplete() error {

	if genDoc.ChainID == "" {
		return cmn.NewError("Genesis doc must include non-empty chain_id")
	}

	if genDoc.ConsensusParams == nil {
		genDoc.ConsensusParams = DefaultConsensusParams()
	} else {
		if err := genDoc.ConsensusParams.Validate(); err != nil {
			return err
		}
	}

	if len(genDoc.Validators) == 0 {
		return cmn.NewError("The genesis file must have at least one validator")
	}

	for _, v := range genDoc.Validators {
		if v.Power == 0 {
			return cmn.NewError("The genesis file cannot contain validators with no voting power: %v", v)
		}
	}

	if genDoc.GenesisTime.IsZero() {
		genDoc.GenesisTime = time.Now()
	}

	return nil
}

//------------------------------------------------------------
// Make genesis state from file

// GenesisDocFromJSON unmarshalls JSON data into a GenesisDoc.
func GenesisDocFromJSON(jsonBlob []byte) (*GenesisDoc, error) {
	genDoc := GenesisDoc{}
	err := cdc.UnmarshalJSON(jsonBlob, &genDoc)
	if err != nil {
		return nil, err
	}

	if err := genDoc.ValidateAndComplete(); err != nil {
		return nil, err
	}

	return &genDoc, err
}

// GenesisDocFromFile reads JSON data from a file and unmarshals it into a GenesisDoc.
func GenesisDocFromFile(config *cfg.Config) (*GenesisDoc, error) {
	genesisFile := config.GenesisFile()

	//verify signature
	signatureFile := genesisFile[0:len(genesisFile)-5] + ".json.sig"
	_, err := sig.VerifyTextFile(genesisFile, signatureFile)
	if err != nil {
		return nil, cmn.ErrorWrap(err, cmn.Fmt("Genesis file verify failed, %v", err.Error()))
	}

	jsonBlob, err := ioutil.ReadFile(genesisFile)
	if err != nil {
		return nil, cmn.ErrorWrap(err, "Couldn't read GenesisDoc file")
	}
	genDoc, err := GenesisDocFromJSON(jsonBlob)
	if err != nil {
		return nil, cmn.ErrorWrap(err, cmn.Fmt("Error reading GenesisDoc at %v", config.GenesisFile()))
	}
	validators := ValidatorsFromFile(*genDoc, config.ValidatorsFile())
	genDoc.Validators = *validators

	return genDoc, nil
}

type contractCode struct {
	Name       string          `json:"name"`
	Version    string          `json:"version"`
	Code       string          `json:"code"`
	Owner      string          `json:"owner,omitempty"`
	CodeByte   cmn.HexBytes    `json:"codeByte,omitempty"`
	CodeHash   string          `json:"codeHash"`
	CodeDevSig json.RawMessage `json:"codeDevSig"`
	CodeOrgSig json.RawMessage `json:"codeOrgSig"`
}

// FillUpWithContractCode -
func FillUpWithContractCode(conf *cfg.Config, appStateJSON json.RawMessage) (json.RawMessage, error) {
	doc := make(map[string]json.RawMessage)
	err := cdc.UnmarshalJSON(appStateJSON, &doc)
	if err != nil {
		return nil, err
	}

	contracts := make([]contractCode, 0)
	err = cdc.UnmarshalJSON(doc["contracts"], &contracts)
	if err != nil {
		return nil, err
	}

	for idx, contract := range contracts {
		codePath := filepath.Join(conf.RootDir, "config", contract.Code)
		blob, err0 := ioutil.ReadFile(codePath)
		if err0 != nil {
			e, oke := err0.(*os.PathError)
			if oke && os.IsNotExist(e) {
				err00 := CopyFile(contract.Code, codePath)
				if err00 != nil {
					return nil, err00
				}
				blob, err0 = ioutil.ReadFile(contract.Code)
				if err0 != nil {
					return nil, err0
				}
			} else {
				return nil, err0
			}
		}
		contracts[idx].CodeByte = blob
	}
	contractBlob, err1 := cdc.MarshalJSON(contracts)
	if err1 != nil {
		return nil, err1
	}
	doc["contracts"] = contractBlob
	docBlob, err2 := cdc.MarshalJSON(doc)
	if err2 != nil {
		return nil, err2
	}
	return docBlob, nil
}

func ValidatorsFromFile(genDoc GenesisDoc, validatorsFile string) *[]GenesisValidator {
	jsonBlob, err := ioutil.ReadFile(validatorsFile)
	if err != nil {
		panic("Couldn't read Validators file")
	}
	validators := make([]GenesisValidator, 0)
	err = cdc.UnmarshalJSON(jsonBlob, &validators)
	if err != nil {
		panic(cmn.Fmt("Error reading Validators at %v", validatorsFile))
	}
	genValidators := genDoc.Validators
	flag := false
	for _, v := range genValidators {
		if !inSlice(v, validators) {
			flag = true
		}
	}
	if flag || len(genValidators) != len(validators) {
		panic("genesis.json & validators.json doesn't match!")
	}
	return &validators
}

func inSlice(a GenesisValidator, list []GenesisValidator) bool {
	for _, b := range list {
		if a.RewardAddr == b.RewardAddr && a.Name == b.Name && a.Power == b.Power {
			return true
		}
	}
	return false
}

// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfd, err := os.Stat(filepath.Dir(dst))
	if err != nil {
		if os.IsNotExist(err) {
			_ = os.MkdirAll(filepath.Dir(dst), 0755) // nolint unhandled
		} else {
			return fmt.Errorf("CopyFile: dir error %s (%q)", dfd.Name(), dfd.Mode().String())
		}
	}

	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	//if err = os.Link(src, dst); err == nil { // link 省不了多少空間卻帶來不少麻煩，哈哈哈
	//	return
	//}
	if err = copyFileContents(src, dst); err == nil {
		return
	}
	fmt.Println("Can't copy genesis template to destinaion, I have nothing to do, the only request is EXIT")
	os.Exit(1)
	return
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer func() { _ = in.Close() }() // nolint unhandled
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
