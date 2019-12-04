package sig

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/tendermint/go-crypto"
)

// 验证签名
// pubkey		公钥
// data         待验证数据
// sig			签名
func Verify(pubkey, data, sig []byte) (bool, error) {
	if len(pubkey) == 0 || len(sig) == 0 {
		return false, errors.New("pubkey and sign cannot to te empty")
	}

	pubKey := crypto.PubKeyEd25519FromBytes(pubkey)
	signature := crypto.SignatureEd25519FromBytes(sig)

	if !pubKey.VerifyBytes(data, signature) {
		return false, errors.New(fmt.Sprintf("Verify signature failed"))
	} else {
		return true, nil
	}
}

// 验证签名
// data         待验证数据
// sigFile	    签名文件
func VerifyFromSigFile(data []byte, sigFile string) (bool, error) {
	sigBytes, err := ioutil.ReadFile(sigFile)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Read file \"%v\" failed, %v", sigFile, err.Error()))
	}

	type SignInfo struct {
		PubKey1   string `json:"pubkey"`       //主链签名文件
		PubKey2   string `json:"publicEccKey"` //加密机签名文件
		Signature string `json:"signature"`
	}
	si := new(SignInfo)
	err = json.Unmarshal(sigBytes, si)
	if err != nil {
		return false, errors.New(fmt.Sprintf("UnmarshalJSON from file \"%v\" failed, %v", sigFile, err.Error()))
	}

	var pubkey []byte
	if si.PubKey1 != "" {
		pubkey, err = hex.DecodeString(si.PubKey1)
	} else if si.PubKey2 != "" {
		pubkey, err = hex.DecodeString(si.PubKey2)
	}
	if err != nil {
		return false, errors.New(fmt.Sprintf("UnmarshalJSON from file \"%v\" failed, %v", sigFile, err.Error()))
	}

	sig, err := hex.DecodeString(si.Signature)
	if err != nil {
		return false, errors.New(fmt.Sprintf("UnmarshalJSON from file \"%v\" failed, %v", sigFile, err.Error()))
	}

	return Verify(pubkey, data, sig)
}

// 验证二进制文件签名
// binFile		二进制文件名称
// sigFile	    签名文件名称
func VerifyBinFile(binFile, sigFile string) (bool, error) {
	binBytes, err := ioutil.ReadFile(binFile)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Read file \"%v\" failed, %v", binFile, err.Error()))
	}

	return VerifyFromSigFile(binBytes, sigFile)
}

// 验证文本文件签名
// textFile		文本文件名称
// sigFile	    签名文件名称
func VerifyTextFile(textFile, sigFile string) (bool, error) {
	textBytes, err := ioutil.ReadFile(textFile)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Read file \"%v\" failed, %v", textFile, err.Error()))
	}

	return VerifyFromSigFile(textBytes, sigFile)
}
