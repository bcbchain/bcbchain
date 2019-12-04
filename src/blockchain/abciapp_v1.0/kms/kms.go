package kms

import (
	"blockchain/abciapp_v1.0/keys"
	"blockchain/abciapp_v1.0/types"
	"common/fs"
	"common/rpc/lib/client"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tendermint/go-crypto"
	cmn "github.com/tendermint/tmlibs/common"
	"io/ioutil"
	"regexp"
	"strings"
)

const (
	pattern       = "^[a-zA-Z0-9_@.]+$"
	MinPassLength = 8
	MaxPassLength = 20
	LocalMode     = "local_mode"
	RemoteMode    = "remote_mode"
)

type Address = cmn.HexBytes

var (
	keystoreDir string //账户目录
	SigMode     string
	SigUrl      string
	CaPath      string
)

// 初始化密钥管理库sigMode="local_mode" or "remote_mode"
func InitKMS(keyStoreDir, sigMode, sigUrl, caPath string) {
	keystoreDir = keyStoreDir
	SigMode = sigMode
	SigUrl = sigUrl
	CaPath = caPath
}

// 生成私钥
// name			账户名称
// passphrase	用来保护账户的口令
func GenPrivKey(name string, passphrase []byte) error {

	if name == "" || len(passphrase) == 0 {
		return errors.New("user name and password cannot to te empty")
	}

	if name != "" && len(name) > 40 {
		return errors.New("user name length only can be 1-40")
	}
	if len(passphrase) != 0 {
		if len(string(passphrase)) < MinPassLength || len(string(passphrase)) > MaxPassLength {
			return errors.New("user passphrase length only can be 8-20")
		}
	}

	valid, err := regexp.Match(pattern, []byte(name))
	if err != nil {
		return errors.New("regexp can't match,get name:" + name)
	}
	if !valid {
		return errors.New("regexp can't match,we want `^[a-zA-Z0-9_@.]+$`,but get name:" + name)
	}
	if keystoreDir != "" {
		exists, err := fs.PathExists(keystoreDir)
		if err != nil {
			return err
		}
		if !exists {
			fs.MakeDir(keystoreDir)
		}
	}
	acct, err := keys.NewAccountExTwo(name, keystoreDir)
	if err != nil {
		return err
	}
	//todo fingerprint 加入个字符串？？
	if acct != nil {
		acct.Save(passphrase, nil)
	}
	return nil
}

// 读取公钥
// name			账户名称
// passphrase	用来保护账户的口令
func GetPubKey(name string, passphrase []byte) ([]byte, error) {

	if name == "" || len(passphrase) == 0 {
		return nil, errors.New("user name and password cannot to te empty")
	}

	if name != "" && len(name) > 40 {
		return nil, errors.New("user name length only can be 1-40")
	}
	if len(passphrase) != 0 {
		if len(string(passphrase)) < MinPassLength || len(string(passphrase)) > MaxPassLength {
			return nil, errors.New("user passphrase length only can be 8-20")
		}
	}
	acct, err := keys.LoadAccount(keystoreDir+"/"+name+".wal", passphrase, nil)
	if acct == nil {
		return nil, err
	}

	pubkey := acct.PubKey.(crypto.PubKeyEd25519)
	return pubkey[:], nil
}

// 对数据签名
// name			账户名称
// passphrase	用来保护账户的口令（
// data			待签名数据
func SignData(name, passphrase string, data []byte) (*types.Ed25519Sig, error) {
	if SigMode == LocalMode {
		return LocalSignData(name, passphrase, data)
	}
	if SigMode == RemoteMode {
		return HttpsSignData(name, passphrase, data)
	}
	panic("sigMode error")
}

func LocalSignData(name, passphrase string, data []byte) (*types.Ed25519Sig, error) {

	if name == "" || len(passphrase) == 0 {
		return nil, errors.New("user name and password cannot to te empty")
	}

	if name != "" && len(name) > 40 {
		return nil, errors.New("user name length only can be 1-40")
	}
	if len(data) <= 0 {
		return nil, errors.New("user data which wants be signed length needs more than 0")
	}
	if len(passphrase) != 0 {
		if len(string(passphrase)) < MinPassLength || len(string(passphrase)) > MaxPassLength {
			return nil, errors.New("user passphrase length only can be 8-20")
		}
	}
	acct, err := keys.LoadAccount(keystoreDir+"/"+name+".wal", []byte(passphrase), nil)
	if acct == nil {
		return nil, err
	}

	sigInfo := types.Ed25519Sig{
		"ed25519",
		acct.PubKey.(crypto.PubKeyEd25519),
		acct.PrivKey.Sign(data).(crypto.SignatureEd25519),
	}

	return &sigInfo, nil
}

//to do 跳过证书验证，后续需要双向验证证书
func HttpsSignData(enPrivKey, passphrase string, data []byte) (*types.Ed25519Sig, error) {
	rpc := rpcclient.NewJSONRPCClientEx(SigUrl, CaPath, true)
	if rpc == nil {
		return nil, errors.New("NewJSONRPCClientForHTTPS failed, please check ca.crt's path")
	}

	coinType, err := GetCoinType()
	if err != nil {
		return nil, err
	}

	type signDataCoinParam struct {
		Tbsigndata string `json:"tbsigndata"`
	}
	coinParam := signDataCoinParam{Tbsigndata: "0x" + hex.EncodeToString(data)}

	type ResultSignRawData struct {
		Type     string `json:"type"`
		PubKey   string `json:"pubKey"`
		SignData string `json:"signData"`
	}

	result := new(ResultSignRawData)

	_, err = rpc.Call("bcb_signrawData", map[string]interface{}{"coinType": coinType, "encPrivateKey": enPrivKey, "password": string(passphrase), "coinParam": coinParam}, result)
	if err != nil {
		return nil, err
	}

	resPubKey := strings.TrimPrefix(result.PubKey, "0x")
	pubKeyData, err := hex.DecodeString(resPubKey)
	if err != nil {
		return nil, err
	}

	resSignData := strings.TrimPrefix(result.SignData, "0x")
	signData, err := hex.DecodeString(resSignData)
	if err != nil {
		return nil, err
	}

	var sigInfo types.Ed25519Sig
	sigInfo.SigType = result.Type
	copy(sigInfo.PubKey[:], pubKeyData)
	copy(sigInfo.SigValue[:], signData)

	return &sigInfo, nil
}

// 验证签名
// pubkey		公钥
// sign			签名
func VerifySign(pubkey, data, sign []byte) (bool, error) {
	if len(pubkey) == 0 || len(sign) == 0 {
		return false, errors.New("pubkey and sign cannot to te empty")
	}

	pubKey := crypto.PubKeyEd25519FromBytes(pubkey)
	signature := crypto.SignatureEd25519FromBytes(sign)

	return pubKey.VerifyBytes(data, signature), nil
}

// 验证签名
// rawFile		原始文件
// signFile	    签名文件
func VerifyFileSign(rawFile, signFile string) (bool, error) {
	rawBytes, err := ioutil.ReadFile(rawFile)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Read file \"%v\" failed, %v", rawFile, err.Error()))
	}
	sigBytes, err := ioutil.ReadFile(signFile)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Read file \"%v\" failed, %v", signFile, err.Error()))
	}

	type SignInfo struct {
		PubKey1   string `json:"pubkey"`
		PubKey2   string `json:"publicEccKey"`
		Signature string `json:"signature"`
	}
	si := new(SignInfo)
	err = json.Unmarshal(sigBytes, si)
	if err != nil {
		return false, errors.New(fmt.Sprintf("UnmarshalJSON from file \"%v\" failed, %v", signFile, err.Error()))
	}

	var pubkey []byte
	if si.PubKey1 != "" {
		pubkey, err = hex.DecodeString(si.PubKey1)
	} else if si.PubKey2 != "" {
		pubkey, err = hex.DecodeString(si.PubKey2)
	}
	if err != nil {
		return false, errors.New(fmt.Sprintf("UnmarshalJSON from file \"%v\" failed, %v", signFile, err.Error()))
	}

	signature, err := hex.DecodeString(si.Signature)
	if err != nil {
		return false, errors.New(fmt.Sprintf("UnmarshalJSON from file \"%v\" failed, %v", signFile, err.Error()))
	}

	ret, err := VerifySign(pubkey, rawBytes, signature)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Verify signature failed, %v", err.Error()))
	}
	if ret == false {
		return false, errors.New(fmt.Sprintf("Verify signature failed"))
	}

	return true, nil
}

// 获取地址
func GetAddress(coinType, enPrivKey, passphrase string) (string, error) {

	if SigMode == LocalMode {
		return LocalAddress(enPrivKey, passphrase)
	}
	if SigMode == RemoteMode {
		return HttpsAddress(coinType, enPrivKey, passphrase)
	}
	panic("sigMode error")

}

func LocalAddress(name, passphrase string) (string, error) {
	if name == "" || len(passphrase) == 0 {
		return "", errors.New("user name and password cannot to te empty")
	}

	if name != "" && len(name) > 40 {
		return "", errors.New("user name length only can be 1-40")
	}

	if len(passphrase) != 0 {
		if len(string(passphrase)) < MinPassLength || len(string(passphrase)) > MaxPassLength {
			return "", errors.New("user passphrase length only can be 8-20")
		}
	}
	//获取from账户信息
	acct, err := keys.LoadAccount("./.keystore/"+name+".wal", []byte(passphrase), nil)
	if err != nil {
		return "", err
	}
	if acct == nil {
		return "", errors.New("get" + name + "'s account info failed")
	}
	return acct.Address, nil
}

func HttpsAddress(coinType, enPrivKey, passphrase string) (string, error) {
	rpc := rpcclient.NewJSONRPCClientEx(SigUrl, CaPath, true)
	if rpc == nil {
		return "", errors.New("NewJSONRPCClientForHTTPS failed, please check ca.crt's path")
	}

	coinType, err := GetCoinType()
	if err != nil {
		return "", err
	}

	type ResultPrikeyToAddr struct {
		Addr string `json:"addr"`
	}

	result := new(ResultPrikeyToAddr)

	_, err = rpc.Call("bcb_prikeyToAddr", map[string]interface{}{"coinType": coinType, "encPrivateKey": enPrivKey, "password": string(passphrase)}, result)
	if err != nil {
		return "", err
	}

	return result.Addr, nil
}

func GetCoinType() (string, error) {
	var coinType string

	chainId := crypto.GetChainId()
	if chainId == "devtest" {
		coinType = "0x1000"
	} else if chainId == "bcbtest" {
		coinType = "0x1001"
	} else if chainId == "bcb" {
		coinType = "0x1002"
	} else if chainId == "local" {
		coinType = "0x1003"
	} else {
		return "", errors.New("Invalid chainId : " + chainId)
	}

	return coinType, nil
}
