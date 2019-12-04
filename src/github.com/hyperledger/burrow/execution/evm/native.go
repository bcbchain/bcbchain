// Copyright 2017 Monax Industries Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package evm

import (
	"crypto/sha256"
	"math/big"

	"github.com/tendermint/tmlibs/common"

	"github.com/hyperledger/burrow/execution/evm/ecrypto"

	"github.com/hyperledger/burrow/crypto"
	"github.com/tendermint/tmlibs/log"

	. "github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/execution/errors"

	"golang.org/x/crypto/ripemd160"
)

const SignatureLength = 65

var registeredNativeContracts = make(map[crypto.EVMAddress]NativeContract)

func IsRegisteredNativeContract(address crypto.EVMAddress) bool {
	_, ok := registeredNativeContracts[address]
	return ok
}

func init() {
	registerNativeContracts()
}

func registerNativeContracts() {
	registeredNativeContracts[crypto.EVMAddress(Int64ToWord256(1).Word160())] = ecRecoverFunc
	registeredNativeContracts[crypto.EVMAddress(Int64ToWord256(2).Word160())] = sha256Func
	registeredNativeContracts[crypto.EVMAddress(Int64ToWord256(3).Word160())] = ripemd160Func
	registeredNativeContracts[crypto.EVMAddress(Int64ToWord256(4).Word160())] = identityFunc
}

//-----------------------------------------------------------------------------

func ExecuteNativeContract(address crypto.EVMAddress, st Interface, caller crypto.EVMAddress, input []byte, gas *uint64,
	logger log.Logger) ([]byte, errors.CodedError) {

	contract, ok := registeredNativeContracts[address]
	if !ok {
		return nil, errors.ErrorCodef(errors.ErrorCodeNativeFunction,
			"no native contract registered at address: %v", address)
	}
	output, err := contract(st, caller, input, gas, logger)
	if err != nil {
		return nil, errors.NewException(errors.ErrorCodeNativeFunction, err.Error())
	}
	return output, nil
}

type NativeContract func(state Interface, caller crypto.EVMAddress, input []byte, gas *uint64,
	logger log.Logger) (output []byte, err error)

func ecRecoverFunc(state Interface, caller crypto.EVMAddress, input []byte, gas *uint64, logger log.Logger) (output []byte, err error) {
	// Deduct gas
	gasRequired := GasEcRecover
	if *gas < gasRequired {
		return nil, errors.ErrorCodeInsufficientGas
	} else {
		*gas -= gasRequired
	}
	// Recover
	const ecRecoverInputLength = 128

	input = common.RightPadBytes(input, ecRecoverInputLength)
	// "input" is (hash, v, r, s), each 32 bytes
	// but for ecrecover we want (r, s, v)

	r := new(big.Int).SetBytes(input[64:96])
	s := new(big.Int).SetBytes(input[96:128])
	v := input[63] - 27

	// tighter sig s values input homestead only apply to tx sigs
	if !allZero(input[32:63]) || !ecrypto.ValidateSignatureValues(v, r, s, false) {
		return nil, nil
	}

	hash := input[:32]
	// We must make sure not to modify the 'input', so placing the 'v' along with
	// the signature needs to be done on a new allocation
	sig := make([]byte, 65)
	copy(sig, input[64:128])
	sig[64] = v

	pub, err := ecrypto.EcRecover(hash, sig)
	// make sure the public key is a valid one
	if err != nil {
		return nil, nil
	}
	// the first byte of pubKey is bitcoin heritage
	return common.LeftPadBytes(ecrypto.Keccak256(pub[1:])[12:], 32), nil
}

func sha256Func(state Interface, caller crypto.EVMAddress, input []byte, gas *uint64,
	logger log.Logger) (output []byte, err error) {
	// Deduct gas
	gasRequired := uint64((len(input)+31)/32)*GasSha256Word + GasSha256Base
	if *gas < gasRequired {
		return nil, errors.ErrorCodeInsufficientGas
	} else {
		*gas -= gasRequired
	}
	// Hash
	hasher := sha256.New()
	// CONTRACT: this does not err
	hasher.Write(input)
	return hasher.Sum(nil), nil
}

func ripemd160Func(state Interface, caller crypto.EVMAddress, input []byte, gas *uint64,
	logger log.Logger) (output []byte, err error) {
	// Deduct gas
	gasRequired := uint64((len(input)+31)/32)*GasRipemd160Word + GasRipemd160Base
	if *gas < gasRequired {
		return nil, errors.ErrorCodeInsufficientGas
	} else {
		*gas -= gasRequired
	}
	// Hash
	hasher := ripemd160.New()
	// CONTRACT: this does not err
	hasher.Write(input)
	return LeftPadBytes(hasher.Sum(nil), 32), nil
}

func identityFunc(state Interface, caller crypto.EVMAddress, input []byte, gas *uint64,
	logger log.Logger) (output []byte, err error) {
	// Deduct gas
	gasRequired := uint64((len(input)+31)/32)*GasIdentityWord + GasIdentityBase
	if *gas < gasRequired {
		return nil, errors.ErrorCodeInsufficientGas
	} else {
		*gas -= gasRequired
	}
	// Return identity
	return input, nil
}

func allZero(b []byte) bool {
	for _, byte := range b {
		if byte != 0 {
			return false
		}
	}
	return true
}
