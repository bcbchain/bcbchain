package wal

import (
	"blockchain/algorithm"
	"bytes"
	"common/fs"
	"common/sig"
	"common/utils"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/tendermint/go-amino"
	"github.com/tendermint/go-crypto"
	cmn "github.com/tendermint/tmlibs/common"
	"golang.org/x/crypto/sha3"
)

var cdc = amino.NewCodec()

const (
	pattern     = "^[a-zA-Z0-9_@.-]{1,40}$"
	passwordErr = "Password must contains by [Uppercase and lowercase letters, numbers, ASCII 32 through 127] and length must be [8-20]"
)

func init() {
	crypto.RegisterAmino(cdc)
}

// ----- account struct -----
type Account struct {
	Name         string         `json:"name"`
	PrivateKey   crypto.PrivKey `json:"privateKey"`
	Hash         []byte         `json:"hash"`
	keyStoreFile string
}

func NewAccount(keyStoreDir, name, password string) (acct *Account, err error) {
	privateKey := crypto.GenPrivKeyEd25519()
	err = checkName(name)
	if err != nil {
		Error(fmt.Sprintf("Create account \"%v\" failed, %v", name, err.Error()))
		return
	}
	return ImportAccount(keyStoreDir, name, password, privateKey)
}

func ImportAccount(keyStoreDir, name, password string, privKey crypto.PrivKey) (acct *Account, err error) {
	err = checkName(name)
	if err != nil {
		return
	}
	privateKey := privKey.(crypto.PrivKeyEd25519)
	keyStoreFile := filepath.Join(keyStoreDir, name+".wal")

	acct = &Account{
		Name:         name,
		PrivateKey:   privateKey,
		keyStoreFile: keyStoreFile,
	}

	sha256 := sha3.New256()
	sha256.Write([]byte(name))
	sha256.Write(privateKey[:])
	acct.Hash = sha256.Sum(nil)

	if err = acct.save(password, true); err != nil {
		acct = nil
	}
	return
}

func LoadAccount(keyStoreDir, name, password string) (acct *Account, err error) {
	err = checkName(name)
	if err != nil {
		Error(fmt.Sprintf("Load account \"%v\" failed, %v", name, err.Error()))
		return
	}
	acct = &Account{}
	keyStoreFile := filepath.Join(keyStoreDir, name+".wal")

	_, err = os.Stat(keyStoreFile)
	if os.IsNotExist(err) {
		return nil, errors.New("KeyStorePath does not exist")
	}
	walBytes, err := ioutil.ReadFile(keyStoreFile)
	if err != nil {
		return nil, errors.New("account does not exist")
	}

	passwordBytes := make([]byte, 0)
	if password == "" {
		passwordBytes, err = utils.CheckPassword("Enter password (" + name + "): ")
		if err != nil {
			return nil, err
		}
	} else {
		passwordBytes = []byte(password)
	}
	flag := checkPassword(string(passwordBytes))
	if flag != true {
		Error(fmt.Sprintf(passwordErr))
		return
	}

	jsonBytes, err := algorithm.DecryptWithPassword(walBytes, passwordBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("the password is wrong, err info : %s", err)
	}
	err = cdc.UnmarshalJSON(jsonBytes, acct)
	if err != nil {
		return nil, err
	}

	privkey := acct.PrivateKey.(crypto.PrivKeyEd25519)
	sha256 := sha3.New256()
	sha256.Write([]byte(acct.Name))
	sha256.Write(privkey[:])
	hash := sha256.Sum(nil)
	if bytes.Equal(hash, acct.Hash) == false {
		return nil, fmt.Errorf("verify hash of wallet failed")
	}

	acct.keyStoreFile = keyStoreFile
	return acct, nil
}

func (acct *Account) Save(password string) (err error) {
	return acct.save(password, false)
}

func (acct *Account) save(password string, notAllowExist bool) (err error) {
	privkey := acct.PrivateKey.(crypto.PrivKeyEd25519)
	sha256 := sha3.New256()
	sha256.Write([]byte(acct.Name))
	sha256.Write(privkey[:])
	hash := sha256.Sum(nil)
	if bytes.Equal(hash, acct.Hash) == false {
		return fmt.Errorf("verify hash of wallet failed")
	}

	if acct.keyStoreFile == "" {
		return errors.New("no key store file specified in account object")
	}
	if ok, _ := fs.PathExists(acct.keyStoreFile); ok && notAllowExist {
		return errors.New("key store file is already exist")
	}

	keyStoreDir := filepath.Dir(acct.keyStoreFile)
	if ok, _ := fs.PathExists(keyStoreDir); !ok {
		if ok, err = fs.MakeDir(keyStoreDir); err != nil {
			return err
		}
	}

	passwordBytes := []byte(password)
	if password == "" {
		passwordBytes, err = utils.GetAndCheckPassword(
			"Enter  password ("+acct.Name+"): ",
			"Repeat password ("+acct.Name+"): ")
		if err != nil {
			return err
		}
	} else {
		passwordBytes = []byte(password)
	}
	flag := checkPassword(string(passwordBytes))
	if flag != true {
		return errors.New(passwordErr)
	}

	jsonBytes, err := cdc.MarshalJSON(acct)
	if err != nil {
		return err
	}
	walBytes := algorithm.EncryptWithPassword(jsonBytes, passwordBytes, nil)
	err = cmn.WriteFileAtomic(acct.keyStoreFile, walBytes, 0600)
	if err != nil {
		return err
	}
	return
}

// Check name format of wallet
func checkName(name string) error {
	valid, err := regexp.Match(pattern, []byte(name))
	if err != nil {
		return errors.New("Regular expression error=" + err.Error())
	}
	if !valid {
		return errors.New(`Name contains by [letters, numbers, "_", "@", "." and "-"] and length must be [1-40] `)
	}

	return nil
}

// Check password format of wallet
func checkPassword(s string) (flag bool) {
	ascOther := ` !"#$%&'()*+,-/:;<=>?[]\^{|}~@_.` + "`"
	count := 0
	number := false
	upper := false
	lower := false
	special := false
	other := true
	for _, c := range s {
		switch {
		case unicode.IsNumber(c):
			number = true
			count++
		case unicode.IsUpper(c):
			upper = true
			count++
		case unicode.IsLower(c):
			lower = true
			count++
		case strings.Contains(ascOther, string(c)):
			special = true
			count++
		default:
			other = false
		}
	}

	flag = number && upper && lower && special && other && 8 <= count && count <= 20

	return
}

func (acct *Account) PubKey() (pubKey crypto.PubKey) {
	return acct.PrivateKey.PubKey()
}

func (acct *Account) Address(chainId string) (address string) {
	return acct.PrivateKey.PubKey().Address(chainId)
}

func (acct *Account) Sign(data []byte) (sigInfo *sig.Ed25519Sig, err error) {
	return sig.Sign(acct.PrivateKey, data)
}

func (acct *Account) Sign2File(data []byte, sigFile string) (err error) {
	return sig.Sign2File(acct.PrivateKey, data, sigFile)
}

func (acct *Account) SignBinFile(binFile, sigFile string) (err error) {
	return sig.SignBinFile(acct.PrivateKey, binFile, sigFile)
}

func (acct *Account) SignTextFile(textFile, sigFile string) (err error) {
	return sig.SignTextFile(acct.PrivateKey, textFile, sigFile)
}

func Error(s string) {
	fmt.Printf("ERROR! -- %v\n", s)
	os.Exit(1)
}
