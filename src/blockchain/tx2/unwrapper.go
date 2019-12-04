package tx2

import (
	"blockchain/common/statedbhelper"
	"blockchain/smcsdk/sdk/rlp"
	"blockchain/types"
	"bytes"
	"common/sig"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"github.com/pkg/errors"
	"github.com/tendermint/go-crypto"
)

// TxParse 解析一笔交易（包含签名验证）的接口函数，将结果填入Transaction数据结构，其中Data字段为RLP编码的合约调用参数
func TxParse(txString string) (tx types.Transaction, pubKey crypto.PubKeyEd25519, err error) {
	MAC := gChainID + "<tx>"
	Version := "v2"
	SignerNumber := "<1>"
	strs := strings.Split(txString, ".")

	if len(strs) != 5 {
		err = errors.New("tx data error")
		return
	}

	if strs[0] != MAC || strs[1] != Version || strs[3] != SignerNumber {
		err = errors.New("tx data error")
		return
	}

	txData := base58.Decode(strs[2])
	sigBytes := base58.Decode(strs[4])

	reader := bytes.NewReader(sigBytes)
	var siginfo sig.Ed25519Sig
	err = rlp.Decode(reader, &siginfo)
	if err != nil {
		return
	}

	if !siginfo.PubKey.VerifyBytes(txData, siginfo.SigValue) {
		err = errors.New("verify sig fail")
		return
	}
	pubKey = siginfo.PubKey

	//RLP解码Transaction结构
	reader = bytes.NewReader(txData)
	err = rlp.Decode(reader, &tx)
	if len(tx.Messages) > 2 {
		err = errors.New("Up to two messages at one time")
		return
	}
	if len(tx.Messages) == 0 {
		err = errors.New("no message in transaction")
		return
	}
	return
}

func QueryDataParse(chainID, txString string) (crypto.Address, types.Query, error) {
	MAC := chainID + "<qy>"
	Version := "v1"
	SignerNumber := "<1>"
	strs := strings.Split(txString, ".")

	if strs[0] != MAC || strs[1] != Version || strs[3] != SignerNumber {
		return "", types.Query{}, errors.New("tx data error")
	}

	txData := base58.Decode(strs[2])
	sigBytes := base58.Decode(strs[4])

	reader := bytes.NewReader(sigBytes)
	var siginfo types.Ed25519Sig
	err := rlp.Decode(reader, &siginfo)
	if err != nil {
		return "", types.Query{}, err
	}

	if !siginfo.PubKey.VerifyBytes(txData, siginfo.SigValue) {
		return "", types.Query{}, errors.New("verify sig fail")
	}
	crypto.SetChainId(chainID)
	siginfo.PubKey.Address(statedbhelper.GetChainID())

	//RLP解码Transaction结构
	reader = bytes.NewReader(txData)
	qy := new(types.Query)
	err = rlp.Decode(reader, qy)
	if err != nil {
		return "", types.Query{}, err
	}

	return siginfo.PubKey.Address(statedbhelper.GetChainID()), *qy, nil
}
