// unittestplatform
// account.go 实现测试账户管理功能，包括创建账户地址等

package utest

import (
	"blockchain/smcsdk/common/gls"
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl/object"
	"crypto/rand"

	"github.com/tendermint/go-crypto"
)

//FuncRecover recover panic by Assert
func FuncRecover(err *types.Error) {
	err.ErrorCode = types.CodeOK
	if rerr := recover(); rerr != nil {
		if _, ok := rerr.(types.Error); ok {
			err.ErrorCode = rerr.(types.Error).ErrorCode
			err.ErrorDesc = rerr.(types.Error).ErrorDesc
		} else {
			panic(rerr)
		}
	}
}

func newRandPubKey() []byte {
	tmp := make([]byte, 32)
	_, err := rand.Read(tmp)
	if err != nil {
		panic(err.Error())
	}
	return tmp
}

// CalcAddressFromPubKey calc address from public key
func CalcAddressFromPubKey(_pubKey []byte) types.Address {
	pk := crypto.PubKeyEd25519FromBytes(_pubKey)
	return pk.Address(utChainID)
}

// NewAccount generate a new account object with a given token and balance
func NewAccount(tokenName string, balance bn.Number) sdk.IAccount {
	var acct sdk.IAccount
	gls.Mgr.SetValues(gls.Values{gls.SDKKey: UTP.ISmartContract}, func() {
		addr := CalcAddressFromPubKey(newRandPubKey())
		UTP.accountList = append(UTP.accountList, addr)

		if balance.IsGreaterThanI(0) {
			Transfer(nil, addr, tokenName, balance)
		}

		acct = object.NewAccount(UTP.ISmartContract, addr)
	})

	return acct
}

// NewAccounts generate some new ccount objects with a given token and balance
func NewAccounts(tokenName string, balance bn.Number, count int) []sdk.IAccount {
	accounts := make([]sdk.IAccount, 0)

	gls.Mgr.SetValues(gls.Values{gls.SDKKey: UTP.ISmartContract}, func() {

		for i := 0; i < count; i++ {
			pubKey := newRandPubKey()
			addr := CalcAddressFromPubKey(pubKey)
			UTP.accountList = append(UTP.accountList, addr)

			if balance.IsGreaterThanI(0) {
				Transfer(nil, addr, tokenName, balance)
			}

			accounts = append(accounts, object.NewAccount(UTP.ISmartContract, addr))
			//commitState()
		}
	})

	return accounts
}

//GetAccount get an account
func (ut *UtPlatform) GetAccount(index int) types.Address {
	if index >= len(ut.accountList) {
		return ""
	}

	return ut.accountList[index]
}

func Transfer(sender sdk.IAccount, addr string, args ...interface{}) (err types.Error) {
	bTrans = true
	err = TransferEx(sender, addr, args...)
	if err.ErrorCode == types.CodeOK {
		Commit()
	}
	bTrans = false

	return err
}

//Transfer transfer token to account
func TransferEx(sender sdk.IAccount, addr string, args ...interface{}) (err types.Error) {
	defer FuncRecover(&err)

	gls.Mgr.SetValues(gls.Values{gls.SDKKey: UTP.ISmartContract}, func() {
		if len(args) == 1 {
			temps := make([]interface{}, 0)
			temps = append(temps, UTP.g.AppStateJSON.GnsToken.Name)
			temps = append(temps, args[0])
			args = temps
		} else if len(args)%2 != 0 { // 可变参数个数必须为偶数
			err.ErrorCode = types.ErrUserDefined
			err.ErrorDesc = "invalid args count"
			return
		}

		index := 0
		for index < len(args) {
			tokenName := args[index].(string)
			value := args[index+1].(bn.Number)

			if value.CmpI(0) > 0 {
				contract := UTP.Message().Contract()

				var ic sdk.IToken
				if tokenName == "" {
					//转本合约代币
					ic = UTP.Helper().TokenHelper().Token()
				} else {
					//代币，只能调用自己合约的代币
					ic = UTP.Helper().TokenHelper().TokenOfName(tokenName)
					if ic != nil {
						tempContract := UTP.Helper().ContractHelper().ContractOfToken(ic.Address())
						UTP.Message().(*object.Message).SetContract(tempContract)
					}
				}

				if ic == nil {
					err.ErrorCode = types.ErrUserDefined // 使用sdk中未使用的错误定义，避免干扰测试结果
					err.ErrorDesc = "Invalid token name=" + tokenName
					return
				}

				if sender == nil {
					sender = object.NewAccount(UTP.ISmartContract, ic.Owner().Address())
				}
				sender.TransferByToken(ic.Address(), addr, value)
				UTP.Message().(*object.Message).SetContract(contract)
			}

			index += 2
		}
	})

	return
}
