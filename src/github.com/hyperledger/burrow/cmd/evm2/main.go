package main

func main() {

}

//
//import (
//	"blockchain/smcsdk/sdk/bn"
//	"blockchain/smcsdk/sdk/types"
//	"fmt"
//
//	"github.com/tendermint/tmlibs/log"
//
//	"github.com/hyperledger/burrow/acm"
//	"github.com/hyperledger/burrow/binary"
//	"github.com/hyperledger/burrow/crypto"
//	"github.com/hyperledger/burrow/execution/evm"
//	"github.com/hyperledger/burrow/execution/evm/abi"
//	"github.com/tmthrgd/go-hex"
//)
//
//func main() {
//	logger := log.NewTMLogger("/dev/stdout", "")
//	ast := newAppState()
//	st := evm.NewState(ast, blockHashGetter)
//
//	//account1 := newAccount(st, "1, 2, 3")
//	//account2 := newAccount(st, "3, 2, 1")
//	//account3 := newAccount(st, "2, 3, 1")
//	//account4 := newAccount(st, "2, 1, 3")
//	//account5 := newAccount(st, "3, 1, 2")
//
//	account1 := "devtestJGF78XLCMUbCxu9yMFC1iztd4GsYEf7qA"
//	account2 := "devtestJGF78XLCMUbCxu9yMFC1iztd4GsYEf7qB"
//	account3 := "devtestJGF78XLCMUbCxu9yMFC1iztd4GsYEf7qC"
//	account4 := "devtestJGF78XLCMUbCxu9yMFC1iztd4GsYEf7qD"
//
//	fmt.Println(account1)
//	fmt.Println(account2)
//	fmt.Println(account3)
//	fmt.Println(account4)
//
//	ourVm := evm.NewVM(newParams(), crypto.ZeroAddress, nil, logger)
//
//	var gas uint64 = 100000
//
//	code := hex.MustDecodeString("608060405234801561001057600080fd5b506102b9806100206000396000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c80633de4eb171461004657806343ae80d31461008c5780638588b2c5146100fa575b600080fd5b61004e61013c565b6040518082601060200280838360005b8381101561007957808201518184015260208101905061005e565b5050505090500191505060405180910390f35b6100b8600480360360208110156100a257600080fd5b81019080803590602001909291905050506101bd565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6101266004803603602081101561011057600080fd5b81019080803590602001909291905050506101f0565b6040518082815260200191505060405180910390f35b610144610261565b60006010806020026040519081016040528092919082601080156101b3576020028201915b8160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019060010190808311610169575b5050505050905090565b600081601081106101ca57fe5b016000915054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b60008082101580156102035750600f8211155b61020c57600080fd5b336000836010811061021a57fe5b0160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550819050919050565b60405180610200016040528060109060208202803883398082019150509050509056fea265627a7a723158209023a73917b39201641210937a0d6629ebf913028762f04ef71c4b27184af2de64736f6c634300050b0032")
//
//	contractCode, err := ourVm.Call(st, evm.NewNoopEventSink(logger), account3, account4, code, nil, bn.N(0), &gas)
//	if err != nil {
//		panic(err) // 部署之后合约代码是否存下来了
//	}
//
//	by1 := st.GetEVMCode(account1)
//	by2 := st.GetEVMCode(account2)
//	by3 := st.GetEVMCode(account3)
//	by4 := st.GetEVMCode(account4)
//
//	fmt.Println(by1, by2, by3, by4)
//
//	st.InitCode(account4, contractCode)
//
//	code2 := hex.MustDecodeString("6080604052600860015530600260006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555034801561005657600080fd5b506040516102f13803806102f18339818101604052602081101561007957600080fd5b8101908080519060200190929190505050806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050610217806100da6000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c8063b8debfbf14610030575b600080fd5b61003861007a565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16638588b2c56001546040518263ffffffff1660e01b815260040180828152602001915050602060405180830381600087803b1580156100f257600080fd5b505af1158015610106573d6000803e3d6000fd5b505050506040513d602081101561011c57600080fd5b8101908080519060200190929190505050506000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166343ae80d36001546040518263ffffffff1660e01b81526004018082815260200191505060206040518083038186803b1580156101a257600080fd5b505afa1580156101b6573d6000803e3d6000fd5b505050506040513d60208110156101cc57600080fd5b810190808051906020019092919050505090509056fea265627a7a72315820dfd25b76e85474fbb06bdc8d780e7e2e8455f12a79c630ba227dcc683c90cbc064736f6c634300050b0032" + "000000000000000000000000a44b1b00979654baa13a4a656653b604e214a23d")
//
//	contractCode2, err := ourVm.Call(st, evm.NewNoopEventSink(logger), account1, account2, code2, nil, bn.N(0), &gas)
//	if err != nil {
//		panic(err)
//	}
//	st.InitCode(account2, contractCode2)
//
//	// input := hex.MustDecodeString("6d4ce63c")
//	input := abi.GetFunctionID("testGetAdopterAddressByPetId()").Bytes()
//
//	output, err := ourVm.Call(st, evm.NewNoopEventSink(logger), account1, account2, contractCode2, input, bn.N(0), &gas)
//	if err != nil {
//		panic(err)
//	}
//
//	fmt.Println("out")
//	fmt.Println(hex.EncodeToString(output))
//	fmt.Println("account")
//	fmt.Println(account2)
//	if account2 != string(output) {
//		panic("not equal")
//	}
//
//	if st.Error() != nil {
//		panic(st.Error())
//	}
//
//	fmt.Println("success")
//}
//
//func newAppState() *evm.FakeAppState {
//	st := &evm.FakeAppState{
//		Accounts: make(map[types.Address]*acm.Account),
//		Storage:  make(map[string][]byte),
//	}
//
//	return st
//}
//
//func blockHashGetter(height uint64) []byte {
//	return binary.LeftPadWord256([]byte(fmt.Sprintf("block_hash_%d", height))).Bytes()
//}
//
////func newAccount(st evm.Interface, name string) types.EVMAddress {
////	//address := newAddress(name)
////	//st.CreateAccount(address)
////
////	acc := new(acm.Account)
////	return acc.EVMAddress()
////}
//
////func newAddress(name string) types.EVMAddress {
////	hashBytes := ripemd160.New()
////	hashBytes.Write([]byte(name)) // nolint errcheck
////	return crypto.MustAddressFromBytes(hashBytes.Sum(nil))
////}
//
//func newParams() evm.Params {
//	return evm.Params{
//		BlockHeight: 0,
//		BlockTime:   0,
//		GasLimit:    0,
//	}
//}
