package tx2

import (
	"blockchain/smcsdk/sdk/rlp"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"blockchain/algorithm"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/types"

	"github.com/tendermint/go-crypto"
)

func TestTransaction_TxParse(t *testing.T) {
	Init("bcb")
	crypto.SetChainId("bcb")

	methodID1 := algorithm.BytesToUint32(algorithm.CalcMethodId("Transfer(types.Address,bn.Number)"))
	toContract1 := "bcbKvG4ayU644JD7BHhEVmP5sof2Lekopj5K"

	toAccount := "bcbKvG4ayU644JD7BHhEVmP5sof2Lekopj5K"
	value := bn.N(1000000000)
	itemInfo1 := WrapInvokeParams(toAccount, value)
	message1 := types.Message{
		Contract: toContract1,
		MethodID: methodID1,
		Items:    itemInfo1,
	}
	nonce := uint64(1)
	gasLimit := int64(500)
	note := "Example for cascade invoke smart contract."
	txPayloadBytesRlp := WrapPayload(nonce, gasLimit, note, message1)
	privKeyStr := "0x4a2c14697282e658b3ed7dd5324de1a102d216d6fa50d5937ffe89f35cbc12aa68eb9a09813bdf7c0869bf34a244cc545711509fe70f978d121afd3a4ae610e6"
	finalTx := WrapTx(txPayloadBytesRlp, privKeyStr)

	privKeyBytes, _ := hex.DecodeString(privKeyStr[2:])
	privKey := crypto.PrivKeyEd25519FromBytes(privKeyBytes)
	address := privKey.PubKey().Address()
	fmt.Println(finalTx)

	transaction, txPubKey, err := TxParse(string(finalTx))
	if err != nil {
		panic("TxParse error:" + err.Error())
	}
	if !reflect.DeepEqual(txPubKey.Address(), address) {
		fmt.Println("Parsed pubkey: ", txPubKey)
		fmt.Println("Wrapper pubkey:", privKey.PubKey())
		fmt.Println("Parsed pubkey address: ", txPubKey.Address())
		fmt.Println("Wrapper pubkey address:", address)

		panic("Sender address is wrong")
	}
	if !reflect.DeepEqual(transaction.Nonce, nonce) {
		panic("nonce is wrong")
	}
	if !reflect.DeepEqual(transaction.GasLimit, gasLimit) {
		panic("gaslimit is wrong")
	}

	if !reflect.DeepEqual(len(transaction.Messages), int(1)) {
		panic("Message size mismatch")
	}
	message := transaction.Messages[0]
	if !reflect.DeepEqual(message.Contract, message1.Contract) {
		panic("Contract in message mismatch")
	}
	if !reflect.DeepEqual(message.MethodID, message1.MethodID) {
		panic("MethodID in message mismatch")
	}
	if !reflect.DeepEqual(len(message.Items), len(message1.Items)) {
		panic("Length of items in message mismatch")
	}
	if !reflect.DeepEqual(len(message.Items), int(2)) {
		panic("Length of items in message is wrong")
	}
	to := ""
	err = rlp.DecodeBytes(message.Items[0], &to)
	if err != nil {
		panic(err)
	}
	if to != toAccount {
		panic("item to is wrong")
	}
	var v bn.Number
	err = rlp.DecodeBytes(message.Items[1], &v)
	if err != nil {
		panic(err)
	}
	if v.String() != value.String() {
		panic("item value is wrong")
	}
}

func TestRegisterOrg(t *testing.T) {
	Init("bcb")
	crypto.SetChainId("bcb")

	methodID1 := algorithm.BytesToUint32(algorithm.CalcMethodId("RegisterOrganization(string)string"))
	toContract1 := "bcb7ygeQKjX3373FhJtFrFpEtUz2Esf7viJi"

	//toAccount := "bcb7ygeQKjX3373FhJtFrFpEtUz2Esf7viJi"
	//value := bn.N(1000000000)
	itemInfo1 := WrapInvokeParams("test-org")
	message1 := types.Message{
		Contract: toContract1,
		MethodID: methodID1,
		Items:    itemInfo1,
	}
	nonce := uint64(1)
	gasLimit := int64(5000000)
	note := "Register organization."
	txPayloadBytesRlp := WrapPayload(nonce, gasLimit, note, message1)
	privKeyStr := "0x0b0d846a25e5461ad607e57ee99a59d472df3c30fef6499e9e2564dc529e190d4904f72f0e054ea3778c5a6480ba19650502bdf1ff8116abe18b8c86fe53d535"
	finalTx := WrapTx(txPayloadBytesRlp, privKeyStr)

	privKeyBytes, _ := hex.DecodeString(privKeyStr[2:])
	privKey := crypto.PrivKeyEd25519FromBytes(privKeyBytes)
	address := privKey.PubKey().Address()
	fmt.Println("addr:" + address)
	fmt.Println(finalTx)

	transaction, txPubKey, err := TxParse(string(finalTx))
	if err != nil {
		panic("TxParse error:" + err.Error())
	}
	if !reflect.DeepEqual(txPubKey.Address(), address) {
		fmt.Println("Parsed pubkey: ", txPubKey)
		fmt.Println("Wrapper pubkey:", privKey.PubKey())
		fmt.Println("Parsed pubkey address: ", txPubKey.Address())
		fmt.Println("Wrapper pubkey address:", address)

		panic("Sender address is wrong")
	}
	if !reflect.DeepEqual(transaction.Nonce, nonce) {
		panic("nonce is wrong")
	}
	if !reflect.DeepEqual(transaction.GasLimit, gasLimit) {
		panic("gaslimit is wrong")
	}

	if !reflect.DeepEqual(len(transaction.Messages), int(1)) {
		panic("Message size mismatch")
	}
	message := transaction.Messages[0]
	if !reflect.DeepEqual(message.Contract, message1.Contract) {
		panic("Contract in message mismatch")
	}
	if !reflect.DeepEqual(message.MethodID, message1.MethodID) {
		panic("MethodID in message mismatch")
	}
	if !reflect.DeepEqual(len(message.Items), len(message1.Items)) {
		panic("Length of items in message mismatch")
	}
	if !reflect.DeepEqual(len(message.Items), int(1)) {
		panic("Length of items in message is wrong")
	}
}

func TestSetOrgSigners(t *testing.T) {
	Init("bcb")
	crypto.SetChainId("bcb")

	methodID1 := algorithm.BytesToUint32(algorithm.CalcMethodId("SetSigners(string,[]types.PubKey)"))
	toContract1 := "bcb7ygeQKjX3373FhJtFrFpEtUz2Esf7viJi"

	//toAccount := "bcb7ygeQKjX3373FhJtFrFpEtUz2Esf7viJi"
	//value := bn.N(1000000000)
	//pk, _ := hex.DecodeString("0x12aa")
	//fmt.Println("pk", pk)
	privKeyStr := "0x0b0d846a25e5461ad607e57ee99a59d472df3c30fef6499e9e2564dc529e190d4904f72f0e054ea3778c5a6480ba19650502bdf1ff8116abe18b8c86fe53d535"

	privKeyBytes, _ := hex.DecodeString(privKeyStr[2:])
	privKey := crypto.PrivKeyEd25519FromBytes(privKeyBytes)
	address := privKey.PubKey().Address()
	fmt.Println("pk", hex.EncodeToString(privKey.PubKey().Bytes()))

	itemInfo1 := WrapInvokeParams("org9Ea4bGjATsmbGUXUDYnDDDDzxPkWUJnh2", [][]byte{privKey.PubKey().Bytes()[5:]})
	message1 := types.Message{
		Contract: toContract1,
		MethodID: methodID1,
		Items:    itemInfo1,
	}
	nonce := uint64(2)
	gasLimit := int64(5000000)
	note := "Register organization."
	txPayloadBytesRlp := WrapPayload(nonce, gasLimit, note, message1)
	finalTx := WrapTx(txPayloadBytesRlp, privKeyStr)

	fmt.Println("addr:" + address)
	//fmt.Println("pubkey:" + privKey.PubKey().Bytes())
	fmt.Println(finalTx)

	transaction, txPubKey, err := TxParse(string(finalTx))
	if err != nil {
		panic("TxParse error:" + err.Error())
	}
	if !reflect.DeepEqual(txPubKey.Address(), address) {
		fmt.Println("Parsed pubkey: ", txPubKey)
		fmt.Println("Wrapper pubkey:", privKey.PubKey())
		fmt.Println("Parsed pubkey address: ", txPubKey.Address())
		fmt.Println("Wrapper pubkey address:", address)

		panic("Sender address is wrong")
	}
	if !reflect.DeepEqual(transaction.Nonce, nonce) {
		panic("nonce is wrong")
	}
	if !reflect.DeepEqual(transaction.GasLimit, gasLimit) {
		panic("gaslimit is wrong")
	}

	if !reflect.DeepEqual(len(transaction.Messages), int(1)) {
		panic("Message size mismatch")
	}
	message := transaction.Messages[0]
	if !reflect.DeepEqual(message.Contract, message1.Contract) {
		panic("Contract in message mismatch")
	}
	if !reflect.DeepEqual(message.MethodID, message1.MethodID) {
		panic("MethodID in message mismatch")
	}
	if !reflect.DeepEqual(len(message.Items), len(message1.Items)) {
		panic("Length of items in message mismatch")
	}
	if !reflect.DeepEqual(len(message.Items), int(2)) {
		panic("Length of items in message is wrong")
	}
}

func TestAuthor(t *testing.T) {
	Init("bcb")
	crypto.SetChainId("bcb")

	methodID1 := algorithm.BytesToUint32(algorithm.CalcMethodId("Authorize(types.Address,string)"))
	toContract1 := "bcbKrNwxKaummdWtZ3dcqWEwFuW2YMDMDXBL" // smartcontract address

	//toAccount := "bcb7ygeQKjX3373FhJtFrFpEtUz2Esf7viJi"
	//value := bn.N(1000000000)
	//pk, _ := hex.DecodeString("0x12aa")
	//fmt.Println("pk", pk)
	privKeyStr := "0x0b0d846a25e5461ad607e57ee99a59d472df3c30fef6499e9e2564dc529e190d4904f72f0e054ea3778c5a6480ba19650502bdf1ff8116abe18b8c86fe53d535"

	//codeBytes,_ := ioutil.ReadFile("/Users/test/today/mystorage.tar.gz")
	privKeyBytes, _ := hex.DecodeString(privKeyStr[2:])
	privKey := crypto.PrivKeyEd25519FromBytes(privKeyBytes)
	address := privKey.PubKey().Address()
	fmt.Println("pk", hex.EncodeToString(privKey.PubKey().Bytes()))

	itemInfo1 := WrapInvokeParams(address, "org9Ea4bGjATsmbGUXUDYnDDDDzxPkWUJnh2")
	message1 := types.Message{
		Contract: toContract1,
		MethodID: methodID1,
		Items:    itemInfo1,
	}
	nonce := uint64(3)
	gasLimit := int64(5000000)
	note := "Register organization."
	txPayloadBytesRlp := WrapPayload(nonce, gasLimit, note, message1)
	finalTx := WrapTx(txPayloadBytesRlp, privKeyStr)

	fmt.Println("addr:" + address)
	//fmt.Println("pubkey:" + privKey.PubKey().Bytes())
	fmt.Println(finalTx)

	transaction, txPubKey, err := TxParse(string(finalTx))
	if err != nil {
		panic("TxParse error:" + err.Error())
	}
	if !reflect.DeepEqual(txPubKey.Address(), address) {
		fmt.Println("Parsed pubkey: ", txPubKey)
		fmt.Println("Wrapper pubkey:", privKey.PubKey())
		fmt.Println("Parsed pubkey address: ", txPubKey.Address())
		fmt.Println("Wrapper pubkey address:", address)

		panic("Sender address is wrong")
	}
	if !reflect.DeepEqual(transaction.Nonce, nonce) {
		panic("nonce is wrong")
	}
	if !reflect.DeepEqual(transaction.GasLimit, gasLimit) {
		panic("gaslimit is wrong")
	}

	if !reflect.DeepEqual(len(transaction.Messages), int(1)) {
		panic("Message size mismatch")
	}
	message := transaction.Messages[0]
	if !reflect.DeepEqual(message.Contract, message1.Contract) {
		panic("Contract in message mismatch")
	}
	if !reflect.DeepEqual(message.MethodID, message1.MethodID) {
		panic("MethodID in message mismatch")
	}
	if !reflect.DeepEqual(len(message.Items), len(message1.Items)) {
		panic("Length of items in message mismatch")
	}
	if !reflect.DeepEqual(len(message.Items), int(2)) {
		panic("Length of items in message is wrong")
	}
}

func TestDepLoyContract(t *testing.T) {
	Init("bcb")
	crypto.SetChainId("bcb")

	methodID1 := algorithm.BytesToUint32(algorithm.CalcMethodId("DeployContract(string,string,string,types.Hash,[]byte,string,string,int64,types.Address)types.Address"))
	toContract1 := "bcbKrNwxKaummdWtZ3dcqWEwFuW2YMDMDXBL" // smartcontract address

	//toAccount := "bcb7ygeQKjX3373FhJtFrFpEtUz2Esf7viJi"
	//value := bn.N(1000000000)
	//pk, _ := hex.DecodeString("0x12aa")
	//fmt.Println("pk", pk)
	//  name string,
	//	version string,
	//	orgID string,
	//	codeHash types.Hash,
	//	codeData []byte,
	//	codeDevSig string,
	//	codeOrgSig string,
	//	effectHeight int64,
	//	owner types.Address,
	privKeyStr := "0x0b0d846a25e5461ad607e57ee99a59d472df3c30fef6499e9e2564dc529e190d4904f72f0e054ea3778c5a6480ba19650502bdf1ff8116abe18b8c86fe53d535"

	codeBytes, _ := ioutil.ReadFile("/Users/test/today/mystorage.tar.gz")
	privKeyBytes, _ := hex.DecodeString(privKeyStr[2:])
	privKey := crypto.PrivKeyEd25519FromBytes(privKeyBytes)
	address := privKey.PubKey().Address()
	fmt.Println("pk", hex.EncodeToString(privKey.PubKey().Bytes()))

	itemInfo1 := WrapInvokeParams("mystorage", "1.0", "org9Ea4bGjATsmbGUXUDYnDDDDzxPkWUJnh2", []byte("codehash"), codeBytes,
		"{\"pubkey\":\"4904f72f0e054ea3778c5a6480ba19650502bdf1ff8116abe18b8c86fe53d535\",\"signature\":\"C5E9D78DED9307448F00DE2EDC50A6354A4259A05D43E03B5DE928ECD359FD78EFABFD9F942E32E3B5C7066D9CEFEF88D188E5F8AB51651F6E857DD3693A120F\"}",
		"{\"pubkey\":\"4904f72f0e054ea3778c5a6480ba19650502bdf1ff8116abe18b8c86fe53d535\",\"signature\":\"0709046E679FA19749F7025D78FF1A6F388177819B360EFEFB623D2CDA3BAF44C988CC937452A3AAABC5F90ACB9E46602D9E00FFEFF709EABDF708FD9F7C8909\"}",
		100, address)
	message1 := types.Message{
		Contract: toContract1,
		MethodID: methodID1,
		Items:    itemInfo1,
	}
	nonce := uint64(4)
	gasLimit := int64(5000000)
	note := "Register organization."
	txPayloadBytesRlp := WrapPayload(nonce, gasLimit, note, message1)
	finalTx := WrapTx(txPayloadBytesRlp, privKeyStr)

	fmt.Println("addr:" + address)
	//fmt.Println("pubkey:" + privKey.PubKey().Bytes())
	fmt.Println(finalTx)

	transaction, txPubKey, err := TxParse(string(finalTx))
	if err != nil {
		panic("TxParse error:" + err.Error())
	}
	if !reflect.DeepEqual(txPubKey.Address(), address) {
		fmt.Println("Parsed pubkey: ", txPubKey)
		fmt.Println("Wrapper pubkey:", privKey.PubKey())
		fmt.Println("Parsed pubkey address: ", txPubKey.Address())
		fmt.Println("Wrapper pubkey address:", address)

		panic("Sender address is wrong")
	}
	if !reflect.DeepEqual(transaction.Nonce, nonce) {
		panic("nonce is wrong")
	}
	if !reflect.DeepEqual(transaction.GasLimit, gasLimit) {
		panic("gaslimit is wrong")
	}

	if !reflect.DeepEqual(len(transaction.Messages), int(1)) {
		panic("Message size mismatch")
	}
	message := transaction.Messages[0]
	if !reflect.DeepEqual(message.Contract, message1.Contract) {
		panic("Contract in message mismatch")
	}
	if !reflect.DeepEqual(message.MethodID, message1.MethodID) {
		panic("MethodID in message mismatch")
	}
	if !reflect.DeepEqual(len(message.Items), len(message1.Items)) {
		panic("Length of items in message mismatch")
	}
	if !reflect.DeepEqual(len(message.Items), int(2)) {
		//panic("Length of items in message is wrong")
	}
}

func TestCallMyStorageSet(t *testing.T) {
	Init("bcb")
	crypto.SetChainId("bcb")

	methodID1 := algorithm.BytesToUint32(algorithm.CalcMethodId("Set(uint64)"))
	toContract1 := "bcbKNvanNbzNZmBFhgbZunFA1KW7ML7zzr6A"

	//toAccount := "bcb7ygeQKjX3373FhJtFrFpEtUz2Esf7viJi"
	//value := bn.N(1000000000)
	//pk, _ := hex.DecodeString("0x12aa")
	//fmt.Println("pk", pk)
	privKeyStr := "0x0b0d846a25e5461ad607e57ee99a59d472df3c30fef6499e9e2564dc529e190d4904f72f0e054ea3778c5a6480ba19650502bdf1ff8116abe18b8c86fe53d535"

	//codeBytes,_ := ioutil.ReadFile("/Users/test/today/mystorage.tar.gz")
	privKeyBytes, _ := hex.DecodeString(privKeyStr[2:])
	privKey := crypto.PrivKeyEd25519FromBytes(privKeyBytes)
	address := privKey.PubKey().Address()
	fmt.Println("pk", hex.EncodeToString(privKey.PubKey().Bytes()))

	itemInfo1 := WrapInvokeParams(uint64(50000000))
	message1 := types.Message{
		Contract: toContract1,
		MethodID: methodID1,
		Items:    itemInfo1,
	}
	nonce := uint64(5)
	gasLimit := int64(5000000)
	note := "Set mystorage"
	txPayloadBytesRlp := WrapPayload(nonce, gasLimit, note, message1)
	finalTx := WrapTx(txPayloadBytesRlp, privKeyStr)

	fmt.Println("addr:" + address)
	//fmt.Println("pubkey:" + privKey.PubKey().Bytes())
	fmt.Println(finalTx)

	transaction, txPubKey, err := TxParse(string(finalTx))
	if err != nil {
		panic("TxParse error:" + err.Error())
	}
	if !reflect.DeepEqual(txPubKey.Address(), address) {
		fmt.Println("Parsed pubkey: ", txPubKey)
		fmt.Println("Wrapper pubkey:", privKey.PubKey())
		fmt.Println("Parsed pubkey address: ", txPubKey.Address())
		fmt.Println("Wrapper pubkey address:", address)

		panic("Sender address is wrong")
	}
	if !reflect.DeepEqual(transaction.Nonce, nonce) {
		panic("nonce is wrong")
	}
	if !reflect.DeepEqual(transaction.GasLimit, gasLimit) {
		panic("gaslimit is wrong")
	}

	if !reflect.DeepEqual(len(transaction.Messages), int(1)) {
		panic("Message size mismatch")
	}
	message := transaction.Messages[0]
	if !reflect.DeepEqual(message.Contract, message1.Contract) {
		panic("Contract in message mismatch")
	}
	if !reflect.DeepEqual(message.MethodID, message1.MethodID) {
		panic("MethodID in message mismatch")
	}
	if !reflect.DeepEqual(len(message.Items), len(message1.Items)) {
		panic("Length of items in message mismatch")
	}
	if !reflect.DeepEqual(len(message.Items), int(2)) {
		//panic("Length of items in message is wrong")
	}
}

func TestCallMyStorageGet(t *testing.T) {
	Init("bcb")
	crypto.SetChainId("bcb")

	methodID1 := algorithm.BytesToUint32(algorithm.CalcMethodId("Get()"))
	toContract1 := "bcbKNvanNbzNZmBFhgbZunFA1KW7ML7zzr6A"

	//toAccount := "bcb7ygeQKjX3373FhJtFrFpEtUz2Esf7viJi"
	//value := bn.N(1000000000)
	//pk, _ := hex.DecodeString("0x12aa")
	//fmt.Println("pk", pk)
	privKeyStr := "0x0b0d846a25e5461ad607e57ee99a59d472df3c30fef6499e9e2564dc529e190d4904f72f0e054ea3778c5a6480ba19650502bdf1ff8116abe18b8c86fe53d535"

	//codeBytes,_ := ioutil.ReadFile("/Users/test/today/mystorage.tar.gz")
	privKeyBytes, _ := hex.DecodeString(privKeyStr[2:])
	privKey := crypto.PrivKeyEd25519FromBytes(privKeyBytes)
	address := privKey.PubKey().Address()
	fmt.Println("pk", hex.EncodeToString(privKey.PubKey().Bytes()))

	itemInfo1 := WrapInvokeParams()
	message1 := types.Message{
		Contract: toContract1,
		MethodID: methodID1,
		Items:    itemInfo1,
	}
	nonce := uint64(6)
	gasLimit := int64(5000000)
	note := "Set mystorage"
	txPayloadBytesRlp := WrapPayload(nonce, gasLimit, note, message1)
	finalTx := WrapTx(txPayloadBytesRlp, privKeyStr)

	fmt.Println("addr:" + address)
	//fmt.Println("pubkey:" + privKey.PubKey().Bytes())
	fmt.Println(finalTx)

	transaction, txPubKey, err := TxParse(string(finalTx))
	if err != nil {
		panic("TxParse error:" + err.Error())
	}
	if !reflect.DeepEqual(txPubKey.Address(), address) {
		fmt.Println("Parsed pubkey: ", txPubKey)
		fmt.Println("Wrapper pubkey:", privKey.PubKey())
		fmt.Println("Parsed pubkey address: ", txPubKey.Address())
		fmt.Println("Wrapper pubkey address:", address)

		panic("Sender address is wrong")
	}
	if !reflect.DeepEqual(transaction.Nonce, nonce) {
		panic("nonce is wrong")
	}
	if !reflect.DeepEqual(transaction.GasLimit, gasLimit) {
		panic("gaslimit is wrong")
	}

	if !reflect.DeepEqual(len(transaction.Messages), int(1)) {
		panic("Message size mismatch")
	}
	message := transaction.Messages[0]
	if !reflect.DeepEqual(message.Contract, message1.Contract) {
		panic("Contract in message mismatch")
	}
	if !reflect.DeepEqual(message.MethodID, message1.MethodID) {
		panic("MethodID in message mismatch")
	}
	if !reflect.DeepEqual(len(message.Items), len(message1.Items)) {
		panic("Length of items in message mismatch")
	}
	if !reflect.DeepEqual(len(message.Items), int(2)) {
		//panic("Length of items in message is wrong")
	}
}
