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
	"blockchain/smcsdk/sdk/bn"
	"blockchain/types"
	"testing"
	"time"

	gocrypto "github.com/tendermint/go-crypto"
	"github.com/tendermint/tmlibs/log"

	"golang.org/x/crypto/sha3"

	"fmt"

	"github.com/hyperledger/burrow/acm/acmstate"

	"github.com/hyperledger/burrow/acm"
	"github.com/hyperledger/burrow/binary"
	. "github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/execution/errors"
	. "github.com/hyperledger/burrow/execution/evm/asm"
	. "github.com/hyperledger/burrow/execution/evm/asm/bc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmthrgd/go-hex"
	"golang.org/x/crypto/ripemd160"
)

// Test output is a bit clearer if we /dev/null the logging, but can be re-enabled by uncommenting the below
//var logger, _, _ = lifecycle.NewStdErrLogger()
//
var logger = log.NewTMLogger("/dev/stdout", "")
var tags = make([]interface{}, 2)

type testState struct {
	*State
	BlockHashProvider func(blockNumber uint64) (Word256, error)
}

func NewTestState(st acmstate.ReaderWriter, blockHashGetter func(uint64) []byte) *testState {
	evmState := NewState(st, blockHashGetter)
	return &testState{
		State:             evmState,
		BlockHashProvider: evmState.GetBlockHash,
	}
}

func newAppState() *FakeAppState {
	fas := &FakeAppState{
		Accounts: make(map[crypto.EVMAddress]*acm.Account),
		Storage:  make(map[string][]byte),
	}

	return fas
}

func newParams() Params {
	return Params{
		BlockHeight: 0,
		BlockTime:   0,
		GasLimit:    0,
	}
}

func newAddress(name string) crypto.EVMAddress {
	hasherSHA3256 := sha3.New256()
	hasherSHA3256.Write([]byte(name))
	sha := hasherSHA3256.Sum(nil)
	fmt.Println("len sha=", len(sha))

	hasherRIPEMD160 := ripemd160.New()
	hasherRIPEMD160.Write(sha) // does not error
	rpd := hasherRIPEMD160.Sum(nil)
	fmt.Println("len rpd=", len(rpd))

	addr := crypto.EVMAddress{}
	copy(addr[:], rpd)
	return addr
}

func newAccount(st Interface, name string) crypto.EVMAddress {
	address := newAddress(name)
	st.CreateAccount(address)
	return address
}

func makeAccountWithCode(st Interface, name string, code []byte) crypto.EVMAddress {
	address := newAddress(name)
	st.CreateAccount(address)
	st.InitCode(address, code)
	st.AddToBalance(address, bn.N(9999999))
	return address
}

// Runs a basic loop
func TestVM(t *testing.T) {
	cache := NewState(newAppState(), blockHashGetter)
	ourVm := NewVM(newParams(), crypto.ZeroAddress, nil, logger)

	// Create Accounts
	account1 := newAccount(cache, "1")
	account2 := newAccount(cache, "101")

	var gas uint64 = 100000

	bytecode := MustSplice(PUSH1, 0x00, PUSH1, 0x20, MSTORE, JUMPDEST, PUSH2, 0x0F, 0x0F, PUSH1, 0x20, MLOAD,
		SLT, ISZERO, PUSH1, 0x1D, JUMPI, PUSH1, 0x01, PUSH1, 0x20, MLOAD, ADD, PUSH1, 0x20,
		MSTORE, PUSH1, 0x05, JUMP, JUMPDEST)

	start := time.Now()
	output, err := ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	t.Logf("Output: %v Error: %v\n", output, err)
	t.Logf("Call took: %v", time.Since(start))
	require.NoError(t, err)
	require.NoError(t, cache.Error())
}

func TestSHL(t *testing.T) {
	cache := NewState(newAppState(), blockHashGetter)
	ourVm := NewVM(newParams(), crypto.ZeroAddress, nil, logger)
	account1 := newAccount(cache, "1")
	account2 := newAccount(cache, "101")

	var gas uint64 = 100000

	//Shift left 0
	bytecode := MustSplice(PUSH1, 0x01, PUSH1, 0x00, SHL, return1())
	output, err := ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value := []byte{0x1}
	expected := LeftPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Alternative shift left 0
	bytecode = MustSplice(PUSH32, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, PUSH1, 0x00, SHL, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	expected = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Shift left 1
	bytecode = MustSplice(PUSH1, 0x01, PUSH1, 0x01, SHL, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value = []byte{0x2}
	expected = LeftPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Alternative shift left 1
	bytecode = MustSplice(PUSH32, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, PUSH1, 0x01, SHL, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	expected = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFE}

	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Alternative shift left 1
	bytecode = MustSplice(PUSH32, 0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, PUSH1, 0x01, SHL, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	expected = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFE}

	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Shift left 255
	bytecode = MustSplice(PUSH1, 0x01, PUSH1, 0xFF, SHL, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value = []byte{0x80}
	expected = RightPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Alternative shift left 255
	bytecode = MustSplice(PUSH32, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, PUSH1, 0xFF, SHL, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value = []byte{0x80}
	expected = RightPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Shift left 256 (overflow)
	bytecode = MustSplice(PUSH1, 0x01, PUSH2, 0x01, 0x00, SHL, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value = []byte{0x00}
	expected = LeftPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Alternative shift left 256 (overflow)
	bytecode = MustSplice(PUSH32, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, PUSH2, 0x01, 0x00, SHL,
		return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value = []byte{0x00}
	expected = LeftPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Shift left 257 (overflow)
	bytecode = MustSplice(PUSH1, 0x01, PUSH2, 0x01, 0x01, SHL, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value = []byte{0x00}
	expected = LeftPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	require.NoError(t, cache.Error())
}

func TestSHR(t *testing.T) {
	cache := NewState(newAppState(), blockHashGetter)
	ourVm := NewVM(newParams(), crypto.ZeroAddress, nil, logger)
	account1 := newAccount(cache, "1")
	account2 := newAccount(cache, "101")

	var gas uint64 = 100000

	//Shift right 0
	bytecode := MustSplice(PUSH1, 0x01, PUSH1, 0x00, SHR, return1())
	output, err := ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value := []byte{0x1}
	expected := LeftPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Alternative shift right 0
	bytecode = MustSplice(PUSH32, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, PUSH1, 0x00, SHR, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	expected = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Shift right 1
	bytecode = MustSplice(PUSH1, 0x01, PUSH1, 0x01, SHR, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value = []byte{0x00}
	expected = LeftPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Alternative shift right 1
	bytecode = MustSplice(PUSH32, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, PUSH1, 0x01, SHR, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value = []byte{0x40}
	expected = RightPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Alternative shift right 1
	bytecode = MustSplice(PUSH32, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, PUSH1, 0x01, SHR, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	expected = []byte{0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Shift right 255
	bytecode = MustSplice(PUSH32, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, PUSH1, 0xFF, SHR, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value = []byte{0x1}
	expected = LeftPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Alternative shift right 255
	bytecode = MustSplice(PUSH32, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, PUSH1, 0xFF, SHR, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value = []byte{0x1}
	expected = LeftPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Shift right 256 (underflow)
	bytecode = MustSplice(PUSH32, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, PUSH2, 0x01, 0x00, SHR,
		return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value = []byte{0x00}
	expected = LeftPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Alternative shift right 256 (underflow)
	bytecode = MustSplice(PUSH32, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, PUSH2, 0x01, 0x00, SHR,
		return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value = []byte{0x00}
	expected = LeftPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Shift right 257 (underflow)
	bytecode = MustSplice(PUSH32, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, PUSH2, 0x01, 0x01, SHR,
		return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value = []byte{0x00}
	expected = LeftPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	require.NoError(t, cache.Error())
}

func TestSAR(t *testing.T) {
	cache := NewState(newAppState(), blockHashGetter)
	ourVm := NewVM(newParams(), crypto.ZeroAddress, nil, logger)
	account1 := newAccount(cache, "1")
	account2 := newAccount(cache, "101")

	var gas uint64 = 100000

	//Shift arith right 0
	bytecode := MustSplice(PUSH1, 0x01, PUSH1, 0x00, SAR, return1())
	output, err := ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value := []byte{0x1}
	expected := LeftPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Alternative arith shift right 0
	bytecode = MustSplice(PUSH32, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, PUSH1, 0x00, SAR, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	expected = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Shift arith right 1
	bytecode = MustSplice(PUSH1, 0x01, PUSH1, 0x01, SAR, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value = []byte{0x00}
	expected = LeftPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Alternative shift arith right 1
	bytecode = MustSplice(PUSH32, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, PUSH1, 0x01, SAR, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value = []byte{0xc0}
	expected = RightPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Alternative shift arith right 1
	bytecode = MustSplice(PUSH32, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, PUSH1, 0x01, SAR, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	expected = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Shift arith right 255
	bytecode = MustSplice(PUSH32, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, PUSH1, 0xFF, SAR, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	expected = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Alternative shift arith right 255
	bytecode = MustSplice(PUSH32, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, PUSH1, 0xFF, SAR, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	expected = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Alternative shift arith right 255
	bytecode = MustSplice(PUSH32, 0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, PUSH1, 0xFF, SAR, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value = []byte{0x00}
	expected = RightPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Shift arith right 256 (reset)
	bytecode = MustSplice(PUSH32, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, PUSH2, 0x01, 0x00, SAR,
		return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	expected = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Alternative shift arith right 256 (reset)
	bytecode = MustSplice(PUSH32, 0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, PUSH2, 0x01, 0x00, SAR,
		return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	value = []byte{0x00}
	expected = LeftPadBytes(value, 32)
	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	//Shift arith right 257 (reset)
	bytecode = MustSplice(PUSH32, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, PUSH2, 0x01, 0x01, SAR,
		return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	expected = []uint8([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})

	assert.Equal(t, expected, output)

	t.Logf("Result: %v == %v\n", output, expected)

	if err != nil {
		t.Fatal(err)
	}

	require.NoError(t, cache.Error())
}

//Test attempt to jump to bad destination (position 16)
func TestJumpErr(t *testing.T) {
	cache := NewState(newAppState(), blockHashGetter)
	ourVm := NewVM(newParams(), crypto.ZeroAddress, nil, logger)

	// Create Accounts
	account1 := newAccount(cache, "1")
	account2 := newAccount(cache, "2")

	var gas uint64 = 100000

	bytecode := MustSplice(PUSH1, 0x10, JUMP)

	var err error
	ch := make(chan struct{})
	go func() {
		_, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
		ch <- struct{}{}
	}()
	tick := time.NewTicker(time.Second * 2)
	select {
	case <-tick.C:
		t.Fatal("VM ended up in an infinite loop from bad jump dest (it took too long!)")
	case <-ch:
		if err == nil {
			t.Fatal("Expected invalid jump dest err")
		}
	}
}

// Tests the code for a subcurrency contract compiled by serpent
func TestSubcurrency(t *testing.T) {
	st := newAppState()
	cache := NewState(st, blockHashGetter)
	// Create Accounts
	account1 := newAccount(cache, "1, 2, 3")
	account2 := newAccount(cache, "3, 2, 1")
	cache.Sync()

	ourVm := NewVM(newParams(), crypto.ZeroAddress, nil, logger)

	var gas uint64 = 1000

	bytecode := MustSplice(PUSH3, 0x0F, 0x42, 0x40, CALLER, SSTORE, PUSH29, 0x01, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, PUSH1,
		0x00, CALLDATALOAD, DIV, PUSH4, 0x15, 0xCF, 0x26, 0x84, DUP2, EQ, ISZERO, PUSH2,
		0x00, 0x46, JUMPI, PUSH1, 0x04, CALLDATALOAD, PUSH1, 0x40, MSTORE, PUSH1, 0x40,
		MLOAD, SLOAD, PUSH1, 0x60, MSTORE, PUSH1, 0x20, PUSH1, 0x60, RETURN, JUMPDEST,
		PUSH4, 0x69, 0x32, 0x00, 0xCE, DUP2, EQ, ISZERO, PUSH2, 0x00, 0x87, JUMPI, PUSH1,
		0x04, CALLDATALOAD, PUSH1, 0x80, MSTORE, PUSH1, 0x24, CALLDATALOAD, PUSH1, 0xA0,
		MSTORE, CALLER, SLOAD, PUSH1, 0xC0, MSTORE, CALLER, PUSH1, 0xE0, MSTORE, PUSH1,
		0xA0, MLOAD, PUSH1, 0xC0, MLOAD, SLT, ISZERO, ISZERO, PUSH2, 0x00, 0x86, JUMPI,
		PUSH1, 0xA0, MLOAD, PUSH1, 0xC0, MLOAD, SUB, PUSH1, 0xE0, MLOAD, SSTORE, PUSH1,
		0xA0, MLOAD, PUSH1, 0x80, MLOAD, SLOAD, ADD, PUSH1, 0x80, MLOAD, SSTORE, JUMPDEST,
		JUMPDEST, POP, JUMPDEST, PUSH1, 0x00, PUSH1, 0x00, RETURN)

	data := hex.MustDecodeString("693200CE0000000000000000000000004B4363CDE27C2EB05E66357DB05BC5C88F850C1A0000000000000000000000000000000000000000000000000000000000000005")
	output, err := ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, data, binary.N0, &gas)
	t.Logf("Output: %v Error: %v\n", output, err)
	if err != nil {
		t.Fatal(err)
	}
	require.NoError(t, cache.Error())
}

//This test case is taken from EIP-140 (https://github.com/ethereum/EIPs/blob/master/EIPS/eip-140.md);
//it is meant to test the implementation of the REVERT opcode
func TestRevert(t *testing.T) {
	cache := NewState(newAppState(), blockHashGetter)
	ourVm := NewVM(newParams(), crypto.ZeroAddress, nil, logger)

	// Create Accounts
	account1 := newAccount(cache, "1")
	account2 := newAccount(cache, "1, 0, 1")

	key, value := []byte{0x00}, []byte{0x00}
	cache.SetStorage(account1, LeftPadWord256(key), value)

	var gas uint64 = 100000

	bytecode := MustSplice(PUSH13, 0x72, 0x65, 0x76, 0x65, 0x72, 0x74, 0x65, 0x64, 0x20, 0x64, 0x61, 0x74, 0x61,
		PUSH1, 0x00, SSTORE, PUSH32, 0x72, 0x65, 0x76, 0x65, 0x72, 0x74, 0x20, 0x6D, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		PUSH1, 0x00, MSTORE, PUSH1, 0x0E, PUSH1, 0x00, REVERT)

	/*bytecode := MustSplice(PUSH32, 0x72, 0x65, 0x76, 0x65, 0x72, 0x74, 0x20, 0x6D, 0x65, 0x73, 0x73, 0x61,
	  0x67, 0x65, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	  0x00, 0x00, 0x00, PUSH1, 0x00, MSTORE, PUSH1, 0x0E, PUSH1, 0x00, REVERT)*/

	output, cErr := ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	assert.Error(t, cErr, "Expected execution reverted error")

	storageVal := cache.GetStorage(account1, LeftPadWord256(key))
	assert.Equal(t, value, storageVal)

	t.Logf("Output: %v\n", output)
}

// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1014.md
func TestCreate2(t *testing.T) {
	st := newAppState()
	cache := NewState(st, blockHashGetter)
	ourVm := NewVM(newParams(), crypto.ZeroAddress, nil, logger)

	// salt of 0s
	var salt [32]byte
	callee := makeAccountWithCode(cache, "callee", MustSplice(PUSH1, 0x0, PUSH1, 0x0, PUSH1, 0x0, PUSH32, salt[:], CREATE2, PUSH1, 0, MSTORE, PUSH1, 20, PUSH1, 12, RETURN))
	privateKey := gocrypto.GenPrivKeyEd25519()
	addr := privateKey.PubKey().Addr()

	var gas uint64 = 100000
	caller := newAccount(cache, "1, 2, 3")
	output, err := ourVm.Call(cache, NewBcEventSink(logger, &tags), caller, callee, cache.GetEVMCode(callee), []byte{}, binary.N0, &gas)
	assert.NoError(t, err, "Should return new address without error")
	assert.Equal(t, []byte(addr), output, "Returned value not equal to create2 address")
}

func TestMemoryBounds(t *testing.T) {
	cache := NewState(newAppState(), blockHashGetter)
	memoryProvider := func(err errors.Sink) Memory {
		return NewDynamicMemory(1024, 2048, err)
	}
	ourVm := NewVM(newParams(), crypto.ZeroAddress, nil, logger, MemoryProvider(memoryProvider))
	caller := makeAccountWithCode(cache, "caller", nil)
	callee := makeAccountWithCode(cache, "callee", nil)
	gas := uint64(100000)
	// This attempts to store a value at the memory boundary and return it
	word := One256
	output, err := ourVm.Call(cache, NewBcEventSink(logger, &tags), caller, callee,
		MustSplice(pushWord(word), storeAtEnd(), MLOAD, storeAtEnd(), returnAfterStore()),
		nil, binary.N0, &gas)
	assert.NoError(t, err)
	assert.Equal(t, word.Bytes(), output)

	// Same with number
	word = Int64ToWord256(232234234432)
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), caller, callee,
		MustSplice(pushWord(word), storeAtEnd(), MLOAD, storeAtEnd(), returnAfterStore()),
		nil, binary.N0, &gas)
	assert.NoError(t, err)
	assert.Equal(t, word.Bytes(), output)

	// Now test a series of boundary stores
	code := pushWord(word)
	for i := 0; i < 10; i++ {
		code = MustSplice(code, storeAtEnd(), MLOAD)
	}
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), caller, callee, MustSplice(code, storeAtEnd(), returnAfterStore()),
		nil, binary.N0, &gas)
	assert.NoError(t, err)
	assert.Equal(t, word.Bytes(), output)

	// Same as above but we should breach the upper memory limit set in memoryProvider
	code = pushWord(word)
	for i := 0; i < 100; i++ {
		code = MustSplice(code, storeAtEnd(), MLOAD)
	}
	require.NoError(t, cache.Error())
	_, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), caller, callee, MustSplice(code, storeAtEnd(), returnAfterStore()),
		nil, binary.N0, &gas)
	assert.Error(t, err, "Should hit memory out of bounds")
}

func TestMsgSender(t *testing.T) {
	st := newAppState()
	cache := NewState(st, blockHashGetter)
	account1 := newAccount(cache, "1, 2, 3")
	account2 := newAccount(cache, "3, 2, 1")

	ourVm := NewVM(newParams(), crypto.ZeroAddress, nil, logger)

	var gas uint64 = 100000

	/*
			pragma solidity ^0.5.4;

			contract SimpleStorage {
		                function get() public constant returns (address) {
		        	        return msg.sender;
		    	        }
			}
	*/

	// This bytecode is compiled from Solidity contract above using remix.ethereum.org online compiler
	code := hex.MustDecodeString("6060604052341561000f57600080fd5b60ca8061001d6000396000f30060606040526004361060" +
		"3f576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680636d4ce63c14604457" +
		"5b600080fd5b3415604e57600080fd5b60546096565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ff" +
		"ffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6000339050905600a165627a" +
		"7a72305820b9ebf49535372094ae88f56d9ad18f2a79c146c8f56e7ef33b9402924045071e0029")

	// Run the contract initialisation code to obtain the contract code that would be mounted at account2
	contractCode, err := ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, code, code, binary.N0, &gas)
	require.NoError(t, err)

	// Not needed for this test (since contract code is passed as argument to vm), but this is what an execution
	// framework must do
	cache.InitCode(account2, contractCode)

	// Input is the function hash of `get()`
	input := hex.MustDecodeString("6d4ce63c")

	output, err := ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, contractCode, input, binary.N0, &gas)
	require.NoError(t, err)

	assert.Equal(t, account1, string(LeftPadWord256(output).UnPadLeft()))

	require.NoError(t, cache.Error())
}

func TestInvalid(t *testing.T) {
	cache := NewState(newAppState(), blockHashGetter)
	ourVm := NewVM(newParams(), crypto.ZeroAddress, nil, logger)

	// Create Accounts
	account1 := newAccount(cache, "1")
	account2 := newAccount(cache, "1, 0, 1")

	var gas uint64 = 100000

	bytecode := MustSplice(PUSH32, 0x72, 0x65, 0x76, 0x65, 0x72, 0x74, 0x20, 0x6D, 0x65, 0x73, 0x73, 0x61,
		0x67, 0x65, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, PUSH1, 0x00, MSTORE, PUSH1, 0x0E, PUSH1, 0x00, INVALID)

	output, err := ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	assert.Equal(t, errors.ErrorCodeExecutionAborted, err.ErrorCode())
	t.Logf("Output: %v Error: %v\n", output, err)
}

func TestReturnDataSize(t *testing.T) {
	cache := NewState(newAppState(), blockHashGetter)
	ourVm := NewVM(newParams(), crypto.ZeroAddress, nil, logger)

	accountName := "account2addresstests"

	ret := "My return message"
	callcode := MustSplice(PUSH32, RightPadWord256([]byte(ret)), PUSH1, 0x00, MSTORE, PUSH1, len(ret), PUSH1, 0x00, RETURN)

	// Create Accounts
	account1 := newAccount(cache, "1")
	account2 := makeAccountWithCode(cache, accountName, callcode)

	var gas uint64 = 100000

	gas1, gas2 := byte(0x1), byte(0x1)
	value := byte(0x69)
	inOff, inSize := byte(0x0), byte(0x0) // no call data
	retOff, retSize := byte(0x0), byte(len(ret))

	bytecode := MustSplice(PUSH1, retSize, PUSH1, retOff, PUSH1, inSize, PUSH1, inOff, PUSH1, value,
		PUSH20, account2, PUSH2, gas1, gas2, CALL,
		RETURNDATASIZE, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN)

	expected := Uint64ToWord256(uint64(len(ret))).Bytes()

	output, err := ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	require.NoError(t, err)
	assert.Equal(t, expected, output)

	t.Logf("Output: %v Error: %v\n", output, err)

	if err != nil {
		t.Fatal(err)
	}
	require.NoError(t, cache.Error())
}

func TestReturnDataCopy(t *testing.T) {
	cache := NewState(newAppState(), blockHashGetter)
	ourVm := NewVM(newParams(), crypto.ZeroAddress, nil, logger)

	accountName := "account2addresstests"

	ret := "My return message"
	callcode := MustSplice(PUSH32, RightPadWord256([]byte(ret)), PUSH1, 0x00, MSTORE, PUSH1, len(ret), PUSH1, 0x00, RETURN)

	// Create Accounts
	account1 := newAccount(cache, "1")
	account2 := makeAccountWithCode(cache, accountName, callcode)

	var gas uint64 = 100000

	gas1, gas2 := byte(0x1), byte(0x1)
	value := byte(0x69)
	inOff, inSize := byte(0x0), byte(0x0) // no call data
	retOff, retSize := byte(0x0), byte(len(ret))

	bytecode := MustSplice(PUSH1, retSize, PUSH1, retOff, PUSH1, inSize, PUSH1, inOff, PUSH1, value,
		PUSH20, account2, PUSH2, gas1, gas2, CALL, RETURNDATASIZE, PUSH1, 0x00, PUSH1, 0x00, RETURNDATACOPY,
		RETURNDATASIZE, PUSH1, 0x00, RETURN)

	expected := []byte(ret)

	output, err := ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	require.NoError(t, err)
	assert.Equal(t, expected, output)

	t.Logf("Output: %v Error: %v\n", output, err)

	require.NoError(t, cache.Error())
}

func TestCallNonExistent(t *testing.T) {
	cache := NewState(newAppState(), blockHashGetter)
	account1 := newAccount(cache, "1")
	cache.AddToBalance(account1, bn.N(10000))
	unknownAddress := newAddress("nonexistent")
	ourVm := NewVM(newParams(), crypto.ZeroAddress, nil, logger)
	var gas uint64
	amt := int64(100)
	_, err := ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, unknownAddress, nil, nil, bn.N(amt), &gas)
	assertErrorCode(t, errors.ErrorCodeIllegalWrite, err,
		"Should not be able to call account before creating it (even before initialising)")
	assert.Equal(t, uint64(0), cache.GetBalance(unknownAddress))
}

func (ts testState) GetBlockHash(blockNumber uint64) (binary.Word256, error) {
	return ts.BlockHashProvider(blockNumber)
}

func TestGetBlockHash(t *testing.T) {
	cache := NewTestState(newAppState(), blockHashGetter)
	params := newParams()

	// Create Accounts
	account1 := newAccount(cache, "1")
	account2 := newAccount(cache, "101")

	var gas uint64 = 100000

	// Non existing block
	params.BlockHeight = 1
	ourVm := NewVM(params, crypto.ZeroAddress, nil, logger)
	bytecode := MustSplice(PUSH1, 2, BLOCKHASH)
	_, err := ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	assertErrorCode(t, errors.ErrorCodeInvalidBlockNumber, err, "attempt to get block hash of a non-existent block")

	// Excessive old block
	cache.error = nil
	params.BlockHeight = 258
	ourVm = NewVM(params, crypto.ZeroAddress, nil, logger)
	bytecode = MustSplice(PUSH1, 1, BLOCKHASH)

	_, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	assertErrorCode(t, errors.ErrorCodeBlockNumberOutOfRange, err, "attempt to get block hash of a block outside of allowed range")

	// Get block hash
	cache.error = nil
	params.BlockHeight = 257
	cache.BlockHashProvider = func(blockNumber uint64) (Word256, error) {
		// Should be just within range 257 - 2 = 255 (and the first and last block count in that range so add one = 256)
		assert.Equal(t, uint64(2), blockNumber)
		return One256, nil
	}
	ourVm = NewVM(params, crypto.ZeroAddress, nil, logger)
	bytecode = MustSplice(PUSH1, 2, BLOCKHASH, return1())

	output, err := ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	assert.NoError(t, err)
	assert.Equal(t, One256, LeftPadWord256(output))

	// Get block hash fail
	cache.error = nil
	params.BlockHeight = 3
	cache.BlockHashProvider = func(blockNumber uint64) (Word256, error) {
		assert.Equal(t, uint64(1), blockNumber)
		return Zero256, fmt.Errorf("unknown block %v", blockNumber)
	}
	ourVm = NewVM(params, crypto.ZeroAddress, nil, logger)
	bytecode = MustSplice(PUSH1, 1, BLOCKHASH, return1())

	_, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	assertErrorCode(t, errors.ErrorCodeInvalidBlockNumber, err, "attempt to get block hash failed")
}

// These code segment helpers exercise the MSTORE MLOAD MSTORE cycle to test
// both of the memory operations. Each MSTORE is done on the memory boundary
// (at MSIZE) which Solidity uses to find guaranteed unallocated memory.

// storeAtEnd expects the value to be stored to be on top of the stack, it then
// stores that value at the current memory boundary
func storeAtEnd() []byte {
	// Pull in MSIZE (to carry forward to MLOAD), swap in value to store, store it at MSIZE
	return MustSplice(MSIZE, SWAP1, DUP2, MSTORE)
}

func returnAfterStore() []byte {
	return MustSplice(PUSH1, 32, DUP2, RETURN)
}

// Store the top element of the stack (which is a 32-byte word) in memory
// and return it. Useful for a simple return value.
func return1() []byte {
	return MustSplice(PUSH1, 0, MSTORE, returnWord())
}

func returnWord() []byte {
	// PUSH1 => return size, PUSH1 => return offset, RETURN
	return MustSplice(PUSH1, 32, PUSH1, 0, RETURN)
}

// this is code to call another contract (hardcoded as addr)
func callContractCode(addr types.Address) []byte {
	gas1, gas2 := byte(0x1), byte(0x1)
	value := byte(0x69)
	inOff, inSize := byte(0x0), byte(0x0) // no call data
	retOff, retSize := byte(0x0), byte(0x20)
	// this is the code we want to run (send funds to an account and return)
	return MustSplice(PUSH1, retSize, PUSH1, retOff, PUSH1, inSize, PUSH1,
		inOff, PUSH1, value, PUSH20, addr, PUSH2, gas1, gas2, CALL, PUSH1, retSize,
		PUSH1, retOff, RETURN)
}

// Produce bytecode for a PUSH<N>, b_1, ..., b_N where the N is number of bytes
// contained in the unpadded word
func pushWord(word Word256) []byte {
	leadingZeros := byte(0)
	for leadingZeros < 32 {
		if word[leadingZeros] == 0 {
			leadingZeros++
		} else {
			return MustSplice(byte(PUSH32)-leadingZeros, word[leadingZeros:])
		}
	}
	return MustSplice(PUSH1, 0)
}

func TestPushWord(t *testing.T) {
	word := Int64ToWord256(int64(2133213213))
	assert.Equal(t, MustSplice(PUSH4, 0x7F, 0x26, 0x40, 0x1D), pushWord(word))
	word[0] = 1
	assert.Equal(t, MustSplice(PUSH32,
		1, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0x7F, 0x26, 0x40, 0x1D), pushWord(word))
	assert.Equal(t, MustSplice(PUSH1, 0), pushWord(Word256{}))
	assert.Equal(t, MustSplice(PUSH1, 1), pushWord(Int64ToWord256(1)))
}

// Kind of indirect test of Splice, but here to avoid import cycles
func TestBytecode(t *testing.T) {
	assert.Equal(t,
		MustSplice(1, 2, 3, 4, 5, 6),
		MustSplice(1, 2, 3, MustSplice(4, 5, 6)))
	assert.Equal(t,
		MustSplice(1, 2, 3, 4, 5, 6, 7, 8),
		MustSplice(1, 2, 3, MustSplice(4, MustSplice(5), 6), 7, 8))
	assert.Equal(t,
		MustSplice(PUSH1, 2),
		MustSplice(byte(PUSH1), 0x02))
	assert.Equal(t,
		[]byte{},
		MustSplice(MustSplice(MustSplice())))

	addr, _ := crypto.AddressFromBytes(Int64ToWord256(102).Bytes())
	gas1, gas2 := byte(0x1), byte(0x1)
	value := byte(0x69)
	inOff, inSize := byte(0x0), byte(0x0) // no call data
	retOff, retSize := byte(0x0), byte(0x20)
	contractCodeBytecode := MustSplice(PUSH1, retSize, PUSH1, retOff, PUSH1, inSize, PUSH1,
		inOff, PUSH1, value, PUSH20, addr, PUSH2, gas1, gas2, CALL, PUSH1, retSize,
		PUSH1, retOff, RETURN)
	contractCode := []byte{0x60, retSize, 0x60, retOff, 0x60, inSize, 0x60, inOff, 0x60, value, 0x73}
	contractCode = append(contractCode, addr[:]...)
	contractCode = append(contractCode, []byte{0x61, gas1, gas2, 0xf1, 0x60, 0x20, 0x60, 0x0, 0xf3}...)
	assert.Equal(t, contractCode, contractCodeBytecode)
}

func TestConcat(t *testing.T) {
	assert.Equal(t,
		[]byte{0x01, 0x02, 0x03, 0x04},
		Concat([]byte{0x01, 0x02}, []byte{0x03, 0x04}))
}

func TestSubslice(t *testing.T) {
	const size = 10
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		data[i] = byte(i)
	}
	err := errors.FirstOnly()
	for n := int64(0); n < size; n++ {
		data = data[:n]
		for offset := int64(-size); offset < size; offset++ {
			for length := int64(-size); length < size; length++ {
				err.Reset()
				subSlice(data, offset, length, err)
				if offset < 0 || length < 0 || n < offset {
					assert.NotNil(t, err.Error())
				} else {
					assert.Nil(t, err.Error())
				}
			}
		}
	}

	err.Reset()
	assert.Equal(t, []byte{
		5, 6, 7, 8, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	}, subSlice([]byte{1, 2, 3, 4, 5, 6, 7, 8}, 4, 32, err))
}

func TestDataStackOverflow(t *testing.T) {
	st := newAppState()
	cache := NewState(st, blockHashGetter)
	account1 := newAccount(cache, "1, 2, 3")
	account2 := newAccount(cache, "3, 2, 1")

	params := newParams()
	params.DataStackMaxDepth = 4
	ourVm := NewVM(params, crypto.ZeroAddress, nil, logger)

	var gas uint64 = 100000

	/*
		pragma solidity ^0.5.4;

		contract SimpleStorage {
			function get() public constant returns (address) {
				return get();
			}
		}
	*/

	// This bytecode is compiled from Solidity contract above using remix.ethereum.org online compiler
	code, err := hex.DecodeString("608060405234801561001057600080fd5b5060d18061001f6000396000f300608060405260043610" +
		"603f576000357c0100000000000000000000000000000000000000000000000000000000900463ff" +
		"ffffff1680636d4ce63c146044575b600080fd5b348015604f57600080fd5b5060566098565b6040" +
		"51808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffff" +
		"ffffffffffff16815260200191505060405180910390f35b600060a06098565b9050905600a16562" +
		"7a7a72305820daacfba0c21afacb5b67f26bc8021de63eaa560db82f98357d4e513f3249cf350029")
	require.NoError(t, err)

	// Run the contract initialisation code to obtain the contract code that would be mounted at account2
	eventSink := NewBcEventSink(logger, &tags)
	contractCode, err := ourVm.Call(cache, eventSink, account1, account2, code, code, binary.N0, &gas)
	require.NoError(t, err)

	// Input is the function hash of `get()`
	input, err := hex.DecodeString("6d4ce63c")
	require.NoError(t, err)

	_, err = ourVm.Call(cache, eventSink, account1, account2, contractCode, input, binary.N0, &gas)
	assertErrorCode(t, errors.ErrorCodeDataStackOverflow, err, "Should be stack overflow")
}

func TestExtCodeHash(t *testing.T) {
	cache := NewState(newAppState(), blockHashGetter)
	ourVm := NewVM(newParams(), crypto.ZeroAddress, nil, logger)
	account1 := newAccount(cache, "1")
	account2 := newAccount(cache, "101")

	var gas uint64 = 100000

	// 1. The EXTCODEHASH of the account without code is c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470
	//    what is the keccack256 hash of empty data.
	bytecode := MustSplice(PUSH20, account1, EXTCODEHASH, return1())
	output, err := ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	assert.NoError(t, err)
	assert.Equal(t, hex.MustDecodeString("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"), output)

	// 2. The EXTCODEHASH of non-existent account is 0.
	bytecode = MustSplice(PUSH1, 0x03, EXTCODEHASH, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	assert.NoError(t, err)
	assert.Equal(t, LeftPadBytes([]byte{0x00}, 32), output)

	// 3. EXTCODEHASH is the hash of an account code
	acc := makeAccountWithCode(cache, "trustedCode", MustSplice(BLOCKHASH, return1()))
	bytecode = MustSplice(PUSH20, acc, EXTCODEHASH, return1())
	output, err = ourVm.Call(cache, NewBcEventSink(logger, &tags), account1, account2, bytecode, []byte{}, binary.N0, &gas)
	assert.NoError(t, err)
	assert.Equal(t, hex.MustDecodeString("010da270094b5199d3e54f89afe4c66cdd658dd8111a41998714227e14e171bd"), output)
}

func assertErrorCode(t *testing.T, expectedCode errors.Code, err error, msgAndArgs ...interface{}) {
	if assert.Error(t, err, msgAndArgs...) {
		actualCode := errors.AsException(err).Code
		if !assert.Equal(t, expectedCode, actualCode, "expected error code %v", expectedCode) {
			t.Logf("Expected '%v' but got '%v'", expectedCode, actualCode)
		}
	}
}
