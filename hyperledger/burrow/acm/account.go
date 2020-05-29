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

package acm

import (
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/sdk/sdk/bn"
	"github.com/bcbchain/sdk/sdk/types"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bcbchain/bcbchain/hyperledger/burrow/crypto"
	goCrypto "github.com/bcbchain/bclib/tendermint/go-crypto"

	"github.com/bcbchain/bcbchain/hyperledger/burrow/binary"

	"github.com/bcbchain/bcbchain/hyperledger/burrow/execution/errors"

	"github.com/bcbchain/bclib/tendermint/go-amino"

	"github.com/bcbchain/bcbchain/hyperledger/burrow/event/query"
)

var cdc = amino.NewCodec()

// Account - 账户，可以是 bvm 合约账户也可以是用户账户
// 作为用户账户的时候，只存取 balance(实际是balanceOfToken) ，因为已经确定了代币(BVMToken)(也可以是本币)
// 作为 bvm 合约账户需要存 BVMToken BVMCode/WASMCode
type Account struct {
	Address  crypto.BVMAddress `json:"address"` //账户地址
	PubKey   types.PubKey      `json:"pubKey"`  //账户公钥
	Balance  bn.Number         `json:"balance"`
	BVMToken goCrypto.Address  `json:"bvmToken"` // bvm 可操作的代币
	BVMCode  ByteCode          `json:"bvmCode"`
	WASMCode ByteCode          `json:"wasmCode,omitempty"`
}

func (a *Account) GetBalance() bn.Number {
	return a.Balance
}

func (a *Account) SetBalance(value bn.Number) {
	a.Balance = value
}

func (a *Account) AddToBalance(amount bn.Number) error {
	newBalance := a.Balance.Add(amount)
	if amount.IsLessThan(bn.N(0)) {
		return errors.ErrorCodef(errors.ErrorCodeIntegerOverflow,
			"uint256 overflow(bigger than %s): attempt to add %v to the balance of %s", binary.N256.String(), amount, a.Address)
	}
	a.SetBalance(newBalance)
	return nil
}

func (a *Account) SubtractFromBalance(amount bn.Number) error {
	if amount.IsLessThan(bn.N(0)) || amount.IsGreaterThan(a.GetBalance()) {
		return errors.ErrorCodef(errors.ErrorCodeInsufficientBalance,
			"insufficient funds: attempt to subtract %v from the balance of %s", amount, a.Address)
	}
	a.SetBalance(a.Balance.Sub(amount))
	return nil
}

func (a *Account) AddAccountTokenKey(transID, txID int64, keyOfAccountToken string) (err error) {

	key := statedbhelper.KeyOfAccount(crypto.ToAddr(a.Address))
	resSlice := make([]goCrypto.Address, 0)
	value, _ := statedbhelper.Get(transID, txID, key)
	if err = json.Unmarshal(value, &resSlice); err != nil {
		return
	}

	isExist := false
	for _, item := range resSlice {
		if item == keyOfAccountToken {
			isExist = true
			break
		}
	}
	if isExist == false {
		resSlice = append(resSlice, keyOfAccountToken)
	}

	result, err := json.Marshal(resSlice)
	if err != nil {
		return err
	}

	statedbhelper.Set(transID, txID, key, result)

	return
}

///---- Serialisation methods

func (a *Account) Encode() ([]byte, error) {
	return cdc.MarshalBinaryBare(a)
}

func Decode(accBytes []byte) (*Account, error) {
	ca := new(Account)
	err := cdc.UnmarshalBinaryBare(accBytes, ca)
	if err != nil {
		return nil, err
	}
	return ca, nil
}

// Copies all mutable parts of account
func (a *Account) Copy() *Account {
	if a == nil {
		return nil
	}
	accCopy := *a
	return &accCopy
}

func (a *Account) Equal(accOther *Account) bool {
	accEnc, err := a.Encode()
	if err != nil {
		return false
	}
	accOtherEnc, err := a.Encode()
	if err != nil {
		return false
	}
	return bytes.Equal(accEnc, accOtherEnc)
}

func (a Account) String() string {
	return fmt.Sprintf("Account{Address: %s; Balance: %v; Token: %v; CodeLength: %v}",
		a.Address, a.Balance, a.BVMToken, len(a.BVMCode))
}

func (a *Account) Tagged() query.Tagged {
	return &TaggedAccount{
		Account: a,
		Tagged:  query.MergeTags(query.MustReflectTags(a, "Address", "Balance", "BVMCode")),
	}
}

type TaggedAccount struct {
	*Account
	query.Tagged
}
