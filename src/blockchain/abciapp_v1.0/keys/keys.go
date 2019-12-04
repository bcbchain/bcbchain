package keys

import (
	"blockchain/algorithm"
	"blockchain/common/statedbhelper"
	"common/fs"
	"errors"
	"fmt"
	"io/ioutil"
	conv "strconv"

	"github.com/tendermint/go-amino"
	"github.com/tendermint/go-crypto"
	cmn "github.com/tendermint/tmlibs/common"
)

type Address = string

var cdc = amino.NewCodec()

func init() {
	crypto.RegisterAmino(cdc)
}

type Account struct {
	Name         string         `json:"name"`
	PrivKey      crypto.PrivKey `json:"privKey"`
	PubKey       crypto.PubKey  `json:"pubKey"`
	Address      Address        `json:"address"`
	Nonce        uint64         `json:"nonce"`
	KeystorePath string         `json:"keystore"`
}

func NewAccount(name string, keystoreDir string) (*Account, error) {
	var keystorePath string
	if keystoreDir != "" {
		keystorePath = keystoreDir + "/" + name + ".wal"
		exists, err := fs.PathExists(keystorePath)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("The account of " + name + " is already exist!")
		}
	}

	privKey := crypto.GenPrivKeyEd25519()
	pubKey := privKey.PubKey()
	address := pubKey.Address(statedbhelper.GetChainID())

	acct := Account{
		Name:         name,
		PrivKey:      privKey,
		PubKey:       pubKey,
		Address:      address,
		Nonce:        0,
		KeystorePath: keystorePath,
	}
	return &acct, nil
}

func NewAccountEx(prefix string, index int, keystoreDir string) (*Account, error) {
	name := prefix + conv.Itoa(index)
	return NewAccount(name, keystoreDir)
}

func NewAccountExTwo(name string, keystoreDir string) (*Account, error) {
	return NewAccount(name, keystoreDir)
}

func LoadAccount(keystorePath string, password, fingerprint []byte) (*Account, error) {
	acct := Account{}
	err := acct.Load(keystorePath, password, fingerprint)
	if err != nil {
		return nil, err
	}
	return &acct, nil
}

func (acct *Account) Save(password, fingerprint []byte) error {
	if acct.KeystorePath == "" {
		cmn.PanicSanity("Cannot save account because KeystorePath not set")
	}
	jsonBytes, err := cdc.MarshalJSON(acct)
	if err != nil {
		return err
	}
	walBytes := algorithm.EncryptWithPassword(jsonBytes, password, fingerprint)
	err = cmn.WriteFileAtomic(acct.KeystorePath, walBytes, 0600)
	if err != nil {
		return err
	}
	return nil
}

func (acct *Account) Load(keystorePath string, password, fingerprint []byte) error {
	if keystorePath == "" {
		cmn.PanicSanity("Cannot loads account because keystorePath not set")
	}
	walBytes, err := ioutil.ReadFile(keystorePath)
	if err != nil {
		return errors.New("account does not exist")
	}
	jsonBytes, err := algorithm.DecryptWithPassword(walBytes, password, fingerprint)
	if err != nil {
		return fmt.Errorf("the password is wrong err info : %s", err)
	}
	err = cdc.UnmarshalJSON(jsonBytes, acct)
	if err != nil {
		return err
	}
	acct.KeystorePath = keystorePath
	return nil
}

func (acct *Account) ToJson() []byte {
	jsonBytes, err := cdc.MarshalJSON(acct)
	if err != nil {
		panic("cannot cdc.Marshal account:" + acct.Name + ",address=" + acct.Address)
	}
	return jsonBytes
}

func (acct *Account) FromJson(jsonBytes []byte) {
	err := cdc.UnmarshalJSON(jsonBytes, acct)
	if err != nil {
		panic("cannot cdc.UnMarshalJson from bytes")
	}
}
