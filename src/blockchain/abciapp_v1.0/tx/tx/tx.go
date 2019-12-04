package tx

import (
	"blockchain/abciapp_v1.0/kms"
	"blockchain/abciapp_v1.0/types"
	"blockchain/smcsdk/sdk/rlp"
	"bytes"
	"errors"
	"github.com/btcsuite/btcutil/base58"
	"github.com/tendermint/go-crypto"
	"strings"
)

// 定义生成交易的接口函数，其中tx.Data已经按RLP进行编码
//返回构造好的交易数据，MAC.Version.Payload.<1>.Signature，Payload和Signature格式是RLP编码后的HexString
func (tx *Transaction) TxGen(chainID, name, passphrase string) (string, error) {
	//RLP编码tx
	size, r, err := rlp.EncodeToReader(tx)
	if err != nil {
		return "", err
	}
	txBytes := make([]byte, size)
	r.Read(txBytes)

	sigInfo, err := kms.SignData(name, passphrase, txBytes)
	if err != nil {
		return "", err
	}

	//RLP编码签名信息
	size, r, err = rlp.EncodeToReader(sigInfo)
	if err != nil {
		return "", err
	}
	sigBytes := make([]byte, size)
	r.Read(sigBytes) //转换为字节流

	txString := base58.Encode(txBytes)
	sigString := base58.Encode(sigBytes)

	MAC := string(chainID) + "<tx>"
	Version := "v1"
	SignerNumber := "<1>"

	return MAC + "." + Version + "." + txString + "." + SignerNumber + "." + sigString, nil
}

// 定义解析一笔交易（包含签名验证）的接口函数，将结果填入Transaction数据结构，其中Data字段为RLP编码的合约调用参数
func (tx *Transaction) TxParse(chainID, txString string) (crypto.Address, crypto.PubKeyEd25519, error) {
	MAC := chainID + "<tx>"
	Version := "v1"
	SignerNumber := "<1>"
	strs := strings.Split(txString, ".")

	if len(strs) != 5 {
		return "", crypto.PubKeyEd25519{}, errors.New("tx data error")
	}

	if strs[0] != MAC || strs[1] != Version || strs[3] != SignerNumber {
		return "", crypto.PubKeyEd25519{}, errors.New("tx data error")
	}

	txData := base58.Decode(strs[2])
	sigBytes := base58.Decode(strs[4])

	reader := bytes.NewReader(sigBytes)
	var siginfo types.Ed25519Sig
	err := rlp.Decode(reader, &siginfo)
	if err != nil {
		return "", crypto.PubKeyEd25519{}, err
	}

	if !siginfo.PubKey.VerifyBytes(txData, siginfo.SigValue) {
		return "", siginfo.PubKey, errors.New("verify sig fail")
	}

	//RLP解码Transaction结构
	reader = bytes.NewReader(txData)
	err = rlp.Decode(reader, tx)
	if err != nil {
		return "", siginfo.PubKey, err
	}

	//crypto.SetChainId(chainID)
	return siginfo.PubKey.Address(chainID), siginfo.PubKey, nil
}

func (qy *Query) QueryDataGen(chainID string, name, passphrase string) (string, error) {
	//RLP编码tx
	qyBytes, err := rlp.EncodeToBytes(qy)
	if err != nil {
		return "", err
	}

	sigInfo, err := kms.SignData(name, passphrase, qyBytes)
	if err != nil {
		return "", err
	}

	//RLP编码签名信息
	sigBytes, err := rlp.EncodeToBytes(sigInfo)
	if err != nil {
		return "", err
	}

	qyString := base58.Encode(qyBytes)
	sigString := base58.Encode(sigBytes)

	MAC := chainID + "<qy>"
	Version := "v1"
	SignerNumber := "<1>"

	return MAC + "." + Version + "." + qyString + "." + SignerNumber + "." + sigString, nil
}

func (qy *Query) QueryDataParse(chainID, txString string) (crypto.Address, error) {
	MAC := chainID + "<qy>"
	Version := "v1"
	SignerNumber := "<1>"
	strs := strings.Split(txString, ".")

	if strs[0] != MAC || strs[1] != Version || strs[3] != SignerNumber {
		return "", errors.New("tx data error")
	}

	txData := base58.Decode(strs[2])
	sigBytes := base58.Decode(strs[4])

	reader := bytes.NewReader(sigBytes)
	var siginfo types.Ed25519Sig
	err := rlp.Decode(reader, &siginfo)
	if err != nil {
		return "", err
	}

	if !siginfo.PubKey.VerifyBytes(txData, siginfo.SigValue) {
		return "", errors.New("verify sig fail")
	}
	//crypto.SetChainId(chainID)
	siginfo.PubKey.Address(chainID)

	//RLP解码Transaction结构
	reader = bytes.NewReader(txData)
	err = rlp.Decode(reader, qy)
	if err != nil {
		return "", err
	}

	return siginfo.PubKey.Address(chainID), nil
}
