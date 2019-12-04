package genstub

import (
	"blockchain/smccheck/parsecode"
	"bytes"
	"os"
	"path/filepath"
	"text/template"
)

const tripleBackQuote = "`"

var templateText1 = `package types

import (
	"blockchain/smcsdk/sdk"
	"blockchain/types"
)

type IContractStub interface {
	InitChain(smcapi sdk.ISmartContract) types.Response
	UpdateChain(smcapi sdk.ISmartContract) types.Response
	Mine(smcapi sdk.ISmartContract) types.Response
	Invoke(smcapi sdk.ISmartContract) types.Response
	InvokeInternal(smcapi sdk.ISmartContract, invokeType int) types.Response
}

type IContractIntfcStub interface {
	Invoke(methodid string, p interface{}) types.Response
	GetSdk() sdk.ISmartContract
	SetSdk(smc sdk.ISmartContract)
}

type IContractIBCStub interface {
	Invoke(smc sdk.ISmartContract) types.Response
}
`

var templateText2 = `package common

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/crypto/sha3"
	"blockchain/smcsdk/sdk/jsoniter"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
	"blockchain/smcsdk/sdkimpl/object"
	types2 "blockchain/types"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/tendermint/tmlibs/common"
)

const (
	METHOD = iota+1
	INTERFACE
	IBC
)

//CreateResponse create response data
func CreateResponse(sdk sdk.ISmartContract, oriTags []common.KVPair, data string, fee, gasUsed, gasLimit int64, err types.Error) (response types2.Response) {
	response.Code = err.ErrorCode
	response.Data = data
	response.Fee = fee
	response.Log = err.Error()
	response.GasLimit = gasLimit
	response.GasUsed = response.GasLimit - sdk.Tx().GasLeft()
	if oriTags != nil{
		response.Tags = oriTags
	}

	for _, v := range sdk.Message().(*object.Message).OutputReceipts() {
		tag := common.KVPair{}
		tag.Value = v.Value
		keySuffix := string(v.Key[1:])[strings.Index(string(v.Key)[1:], "/")+1:]
		tag.Key = []byte(fmt.Sprintf("/%d/%s", len(response.Tags), keySuffix))
		response.Tags = append(response.Tags, tag)
	}
	return
}

//FeeAndReceipt pay fee for the calling and emit fee receipt
func FeeAndReceipt(smc sdk.ISmartContract, invokeType int) (fee, gasUsed int64, receipt types.KVPair, err types.Error) {

	err.ErrorCode = types.CodeOK
	//Get gas price
	var gasprice int64
    gasPriceRatio := int64(smc.Helper().GenesisHelper().GasPriceRatio())
	if smc.Message().Contract().Token() == "" {
		tokenAddr := smc.Helper().GenesisHelper().Token().Address()
		gasprice = smc.Helper().TokenHelper().TokenOfAddress(tokenAddr).GasPrice() * gasPriceRatio / 1000
	} else {
		gasprice = smc.Helper().TokenHelper().Token().GasPrice() * gasPriceRatio / 1000
	}
	//calculate fee
	var methods []std.Method
	switch invokeType {
	case METHOD:
		methods = smc.Message().Contract().Methods()
	case INTERFACE:
		methods = smc.Message().Contract().Interfaces()
	case IBC:
		methods = smc.Message().Contract().IBCs()
	default:
		err.ErrorCode = types.ErrStubDefined
		err.ErrorDesc = "undefined invoke type"
		return
	}

	var gas int64
	for _, m := range methods {
		if m.MethodID == smc.Message().MethodID() {
			gas = m.Gas
			break
		}
	}
	gasAbs := int64(math.Abs(float64(gas))) //abs number

	gasLeft := smc.Tx().GasLeft()
	if gasLeft < gasAbs {
		gasUsed = gasLeft
		err.ErrorCode = types.ErrGasNotEnough
	} else {
		gasUsed = gasAbs
	}
	fee = gasprice * gasUsed

	payer := smc.Message().Payer()
	token := smc.Helper().GenesisHelper().Token().Address()
	balance := payer.BalanceOfToken(token)
	if balance.IsLessThanI(fee) {
		fee = balance.V.Int64()
		balance = bn.N(0)
		gasUsed = fee/gasprice
		err.ErrorCode = types.ErrFeeNotEnough
	} else {
		balance = balance.SubI(fee)
	}
	payer.(*object.Account).SetBalanceOfToken(token, balance)

	//Set gasLeft to tx
	gasLeft = gasLeft - gasUsed
	smc.Tx().(*object.Tx).SetGasLeft(gasLeft)
	//emit receipt
	feeReceipt := std.Fee{
		Token: smc.Helper().GenesisHelper().Token().Address(),
		From:  payer.Address(),
		Value: fee,
	}
	receipt = emitFeeReceipt(smc, feeReceipt)

	return
}

func CalcKey(name, version string) string {
	if strings.HasPrefix(name, "token-templet-") {
		name = "token-issue"
	}
	return name + strings.Replace(version, ".", "", -1)
}

func emitFeeReceipt(smc sdk.ISmartContract,receipt std.Fee) types.KVPair {
	bz, err := jsoniter.Marshal(receipt)
	if err != nil {
		sdkimpl.Logger.Fatalf("[sdk]Cannot marshal receipt data=%v", receipt)
		sdkimpl.Logger.Flush()
		panic(err)
	}

	rcpt := std.Receipt{
		Name:         receiptName(receipt),
		ContractAddr: smc.Message().Contract().Address(),
		Bytes:        bz,
		Hash:         nil,
	}
	rcpt.Hash = sha3.Sum256([]byte(rcpt.Name), []byte(rcpt.ContractAddr), bz)
	resBytes, _ := jsoniter.Marshal(rcpt) // nolint unhandled

	result := types.KVPair{
		Key:   []byte(fmt.Sprintf("/%d/%s", len(smc.Message().(*object.Message).OutputReceipts()), rcpt.Name)),
		Value: resBytes,
	}

	return result
}

func receiptName(receipt interface{}) string {
	typeOfInterface := reflect.TypeOf(receipt).String()

	if strings.HasPrefix(typeOfInterface, "std.") {
		prefixLen := len("std.")
		return "std::" + strings.ToLower(typeOfInterface[prefixLen:prefixLen+1]) + typeOfInterface[prefixLen+1:]
	}

	return typeOfInterface
}
`

var templateText3 = `package softforks

import "common/jsoniter"

//具体含义请参考 bcchain.yaml
type ForkInfo struct {
	Tag               string ` + tripleBackQuote + `json:"tag,omitempty"` + tripleBackQuote + `       // Tag, contains the former released version"
	BugBlockHeight    int64  ` + tripleBackQuote + `json:"bugblockheight,omitempty"` + tripleBackQuote + `    // bug block height
	EffectBlockHeight int64  ` + tripleBackQuote + `json:"effectblockheight,omitempty"` + tripleBackQuote + ` // Effect Block Height
	Description       string ` + tripleBackQuote + `json:"description,omitempty"` + tripleBackQuote + `       // Description for the fork
}

var (
	TagToForkInfo map[string]ForkInfo
)

func Init(forksBytes []byte) {
	err := jsoniter.Unmarshal(forksBytes, &TagToForkInfo)
	if err != nil {
		panic(err)
	}
}

// Fixs bug #4281, sdk block hash not equal tendermint block hahs.
// Adds the softfork to reset sdk block hash
func V2_0_1_13780(blockHeight int64) bool {
	if forkInfo, ok := TagToForkInfo["fork-abci#2.0.1.13780"]; ok {
		return blockHeight < forkInfo.EffectBlockHeight
	}

	return false
}`

// GenStubCommon - generate the stub common go source
func GenStubCommon(rootDir string) {

	err := genTypes(rootDir)
	if err != nil {
		panic(err)
	}

	err = genCommon(rootDir)
	if err != nil {
		panic(err)
	}

	err = genSoftforks(rootDir)
	if err != nil {
		panic(err)
	}

}

func genTypes(rootDir string) error {
	newPath := filepath.Join(rootDir, "types")
	if err := os.MkdirAll(newPath, os.FileMode(0750)); err != nil {
		return err
	}
	filename := filepath.Join(newPath, "types.go")

	tmpl, err := template.New("types").Parse(templateText1)
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	if err = tmpl.Execute(&buf, nil); err != nil {
		return err
	}

	if err := parsecode.FmtAndWrite(filename, buf.String()); err != nil {
		return err
	}

	return nil
}

func genCommon(rootDir string) error {
	newPath := filepath.Join(rootDir, "common")
	if err := os.MkdirAll(newPath, os.FileMode(0750)); err != nil {
		return err
	}
	filename := filepath.Join(newPath, "common.go")

	tmpl, err := template.New("common").Parse(templateText2)
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	if err = tmpl.Execute(&buf, nil); err != nil {
		return err
	}

	if err := parsecode.FmtAndWrite(filename, buf.String()); err != nil {
		return err
	}

	return nil
}

func genSoftforks(rootDir string) error {
	newPath := filepath.Join(rootDir, "softforks")
	if err := os.MkdirAll(newPath, os.FileMode(0750)); err != nil {
		return err
	}
	filename := filepath.Join(newPath, "softforks.go")

	tmpl, err := template.New("softforks").Parse(templateText3)
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	if err = tmpl.Execute(&buf, nil); err != nil {
		return err
	}

	if err := parsecode.FmtAndWrite(filename, buf.String()); err != nil {
		return err
	}

	return nil
}
