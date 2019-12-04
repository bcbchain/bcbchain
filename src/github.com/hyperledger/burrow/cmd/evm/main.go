package main

import (
	"blockchain/burrow"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/statedb"
	"fmt"

	"github.com/hyperledger/burrow/execution/errors"

	goCrypto "github.com/tendermint/go-crypto"
	"github.com/tendermint/tmlibs/log"
	"golang.org/x/crypto/sha3"

	"github.com/hyperledger/burrow/acm/acmstate"

	"github.com/hyperledger/burrow/execution/evm/abi"

	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/execution/evm"

	"github.com/tmthrgd/go-hex"
	"golang.org/x/crypto/ripemd160"
)

var transId int64

func main() {
	goCrypto.SetChainId("devtest")
	logger := log.NewTMLogger("/dev/stdout", "")
	ast := newAppState(logger)
	ast.SetToken("myToken")
	st := evm.NewState(ast, blockHashGetter)
	tags := make([]interface{}, 2)

	account1 := newAccount(st, "1, 2, 3, 4, 5")
	account2 := newAccount(st, "2, 3, 4, 5, 1")
	account3 := newAccount(st, "3, 4, 5, 1, 2")
	account4 := newAccount(st, "4, 5, 1, 2, 3")
	fmt.Println("1: ", account1)
	fmt.Println("2: ", account2)
	fmt.Println("3: ", account3)
	fmt.Println("4: ", account4)

	if st.Error() != nil && st.Error().ErrorCode() == errors.ErrorCodeDuplicateAddress {
		account, err := ast.GetAccount(account4)
		if err != nil {
			panic(err)
		}
		fmt.Println("saved token:", account.EVMToken)
		fmt.Println("saved code:", hex.EncodeToString(account.EVMCode))
		return
	}
	ourVm := evm.NewVM(newParams(), crypto.ZeroAddress, nil, logger)

	var gas uint64 = 100000

	code := hex.MustDecodeString("608060405234801561001057600080fd5b506102b9806100206000396000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c80633de4eb171461004657806343ae80d31461008c5780638588b2c5146100fa575b600080fd5b61004e61013c565b6040518082601060200280838360005b8381101561007957808201518184015260208101905061005e565b5050505090500191505060405180910390f35b6100b8600480360360208110156100a257600080fd5b81019080803590602001909291905050506101bd565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6101266004803603602081101561011057600080fd5b81019080803590602001909291905050506101f0565b6040518082815260200191505060405180910390f35b610144610261565b60006010806020026040519081016040528092919082601080156101b3576020028201915b8160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019060010190808311610169575b5050505050905090565b600081601081106101ca57fe5b016000915054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b60008082101580156102035750600f8211155b61020c57600080fd5b336000836010811061021a57fe5b0160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550819050919050565b60405180610200016040528060109060208202803883398082019150509050509056fea265627a7a723158209023a73917b39201641210937a0d6629ebf913028762f04ef71c4b27184af2de64736f6c634300050b0032")

	contractCode, err := ourVm.Call(st, evm.NewBcEventSink(logger, &tags), account3, account4, code, nil, bn.N(0), &gas)
	if err != nil {
		panic(err)
	}
	st.InitCode(account4, contractCode)

	code2 := hex.MustDecodeString("608060405234801561001057600080fd5b506104c2806100206000396000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c8063081215bc146100465780635ed8b37d1461009057806377d32e94146100da575b600080fd5b61004e6101df565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6100986102c5565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b61019d600480360360408110156100f057600080fd5b81019080803590602001909291908035906020019064010000000081111561011757600080fd5b82018360208201111561012957600080fd5b8035906020019184600183028401116401000000008311171561014b57600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f8201169050808301925050505050505091929192905050506103ab565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6000807fce0677bb30baa8cf067c88db9811f4333d131bf8bcf12fe7065d211dce97100890506000601c905060007f90f27b8b488db00b00606796d2987f6a5f59ae62ea05effe84fef5b8b0e54998905060007f4a691139ad57a3f0b906637673aa2f63d1f55cb1a69199d4009eea23ceaddc93905060018484848460405160008152602001604052604051808581526020018460ff1660ff1681526020018381526020018281526020019450505050506020604051602081039080840390855afa1580156102b2573d6000803e3d6000fd5b5050506020604051035194505050505090565b6000807f4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c4590506000601b905060007ff4128988cbe7df8315440adde412a8955f7f5ff9a5468a791433727f82717a67905060007f53bd71882079522207060b681fbd3f5623ee7ed66e33fc8e581f442acbcf6ab8905060018484848460405160008152602001604052604051808581526020018460ff1660ff1681526020018381526020018281526020019450505050506020604051602081039080840390855afa158015610398573d6000803e3d6000fd5b5050506020604051035194505050505090565b60008060008060418551146103c65760009350505050610487565b602085015192506040850151915060ff6041860151169050601b8160ff1610156103f157601b810190505b601b8160ff16141580156104095750601c8160ff1614155b1561041a5760009350505050610487565b60018682858560405160008152602001604052604051808581526020018460ff1660ff1681526020018381526020018281526020019450505050506020604051602081039080840390855afa158015610477573d6000803e3d6000fd5b5050506020604051035193505050505b9291505056fea265627a7a723158201fc6215913a3b303f3072d928c45e4a7463d1202e0a0d5725e490f60f33477bb64736f6c634300050c0032")

	contractCode2, err := ourVm.Call(st, evm.NewBcEventSink(logger, &tags), account1, account2, code2, nil, bn.N(0), &gas)
	if err != nil {
		panic(err)
	}
	st.InitCode(account2, contractCode2)

	// input := hex.MustDecodeString("6d4ce63c")
	input := abi.GetFunctionID("myTest()").Bytes()

	output, err := ourVm.Call(st, evm.NewBcEventSink(logger, &tags), account1, account2, contractCode2, input, bn.N(0), &gas)
	if err != nil {
		panic(err)
	}

	stringAddr := binary.LeftPadWord256(output).Word160()
	fmt.Println("out")
	fmt.Println(stringAddr)
	fmt.Println("account")
	fmt.Println(account2)
	if !account2.Equal(crypto.EVMAddress(stringAddr)) {
		panic("not equal")
	}

	if st.Error() != nil {
		panic(st.Error())
	}

	if e := st.Sync(); e != nil {
		panic(e)
	}

	statedb.CommitTx(transId, 1)
	statedb.Commit(transId)
	fmt.Println("success")
}

func newAppState(logger log.Logger) acmstate.ReaderWriter {
	statedb.Init("evm-state", "", "")
	transId = statedb.NewTransaction()
	st := burrow.NewState(transId, 1, logger)
	return st
}

func blockHashGetter(height uint64) []byte {
	return binary.LeftPadWord256([]byte(fmt.Sprintf("block_hash_%d", height))).Bytes()
}

func newAccount(st evm.Interface, name string) crypto.EVMAddress {
	address := newAddress(name)
	st.CreateAccount(address)
	return address
}

func newAddress(name string) crypto.EVMAddress {
	hasherSHA3256 := sha3.New256()
	hasherSHA3256.Write([]byte(name))
	sha := hasherSHA3256.Sum(nil)
	// fmt.Println("len sha=", len(sha))

	hasherRIPEMD160 := ripemd160.New()
	hasherRIPEMD160.Write(sha) // does not error
	rpd := hasherRIPEMD160.Sum(nil)
	// fmt.Println("len rpd=", len(rpd))

	addr := crypto.EVMAddress{}
	copy(addr[:], rpd)
	return addr
}

func newParams() evm.Params {
	return evm.Params{}
}
