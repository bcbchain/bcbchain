package sig

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/tendermint/go-crypto"
	cmd "github.com/tendermint/tmlibs/common"
)

// 对交易数据签名
// privateKey	私钥对象
// data			待签名交易数据
func Sign(privateKey crypto.PrivKey, data []byte) (*Ed25519Sig, error) {

	if len(data) <= 0 {
		return nil, errors.New("user data which wants be signed length needs more than 0")
	}

	sigInfo := Ed25519Sig{
		"ed25519",
		privateKey.PubKey().(crypto.PubKeyEd25519),
		privateKey.Sign(data).(crypto.SignatureEd25519),
	}

	return &sigInfo, nil
}

// 对裸数据数据签名
// privateKey	私钥对象
// data			待签名数据
// sigFile		输出的签名文件
func Sign2File(privateKey crypto.PrivKey, data []byte, sigFile string) error {

	if len(data) <= 0 {
		return errors.New("user data which wants be signed length needs more than 0")
	}

	pubKeyBytes := privateKey.PubKey().(crypto.PubKeyEd25519)
	sigBytes := privateKey.Sign(data).(crypto.SignatureEd25519)

	sigInfo := FileSig{
		PubKey1:   hex.EncodeToString(pubKeyBytes[:]),
		Signature: strings.ToUpper(hex.EncodeToString(sigBytes[:])),
	}

	// get the json byte
	sigJsonByte, err := json.MarshalIndent(sigInfo, "", "  ")
	if err != nil {
		return err
	}

	// write to signature file
	return cmd.WriteFileAtomic(sigFile, sigJsonByte, 0600)
}

// 对二进制文件进行签名
// privateKey	私钥对象
// binFile		二进制文件名称
// sigFile		输出的签名文件名称
func SignBinFile(privateKey crypto.PrivKey, binFile, sigFile string) error {

	binBytes, err := ioutil.ReadFile(binFile)
	if err != nil {
		return errors.New(fmt.Sprintf("Read file \"%v\" failed, %v", binFile, err.Error()))
	}

	return Sign2File(privateKey, binBytes, sigFile)
}

// 对文本文件进行签名
// privateKey	私钥对象
// textFile		文本文件名称
// sigFile		输出的签名文件名称
func SignTextFile(privateKey crypto.PrivKey, textFile, sigFile string) error {

	textBytes, err := ioutil.ReadFile(textFile)
	if err != nil {
		return errors.New(fmt.Sprintf("Read file \"%v\" failed, %v", textFile, err.Error()))
	}

	return Sign2File(privateKey, textBytes, sigFile)
}
