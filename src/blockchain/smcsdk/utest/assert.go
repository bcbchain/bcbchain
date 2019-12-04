/*
 * assert.go 实现各种断言方法，判断执行结果是否符合预期
 */

package utest

import (
	"blockchain/smcsdk/common/gls"
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/jsoniter"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"errors"
	"fmt"
	"gopkg.in/check.v1"
)

// Assert assert true
func Assert(b bool) {
	UTP.c.Assert(b, check.Equals, true)
}

//AssertEquals assert a equals b
func AssertEquals(a, b interface{}) {
	UTP.c.Assert(a, check.Equals, b)
}

//AssertError assert error code
func AssertError(err types.Error, expected uint32) {
	UTP.c.Assert(err.ErrorCode, check.Equals, expected)
}

//AssertOK assert errcode is CodeOK
func AssertOK(err types.Error) {
	UTP.c.Assert(err.ErrorCode, check.Equals, uint32(types.CodeOK))
}

//AssertErrorMsg assert error message
func AssertErrorMsg(err types.Error, msg string) {
	UTP.c.Assert(err.Error(), check.Matches, "*"+msg+"*")
}

//AssertBalance assert balance
func AssertBalance(account sdk.IAccount, tokenName string, value bn.Number) {
	gls.Mgr.SetValues(gls.Values{gls.SDKKey: UTP.ISmartContract}, func() {
		_token := UTP.Helper().TokenHelper().TokenOfName(tokenName)
		key := std.KeyOfAccountToken(account.Address(), _token.Address())
		b := sdbGet(0, 0, key)
		b = data(key, b)

		v := std.AccountInfo{}
		err := jsoniter.Unmarshal(b, &v)
		if err != nil {
			panic(err.Error())
		}
		UTP.c.Assert(_token.Address(), check.Equals, v.Address)
		UTP.c.Assert(value.V.String(), check.Equals, v.Balance.V.String())
	})
	//rollbackState()
}

//AssertSDB assert key's value in SDB
//判断状态数据库中某一Key的值，匹配完整格式，可以为结构体
func AssertSDB(key string, interf interface{}) {

	if err := checkKey(key); err != nil {
		panic(err)
	}

	_v, err := jsoniter.Marshal(interf)
	if err != nil {
		panic(err.Error())
	}

	fullKey := prefix + key

	b := sdbGet(0, 0, fullKey)
	b = data(fullKey, b)
	//rollbackState()

	if interf == nil {
		UTP.c.Assert(b, check.IsNil)
	} else {
		UTP.c.Assert(b, check.NotNil)
		UTP.c.Assert(b, check.DeepEquals, _v)
	}
}

//AssertSDB assert key's value in SDB
//判断状态数据库中某一Key的值，匹配完整格式，可以为结构体
func AssertSDBGlobal(fullkey string, interf interface{}) {

	_v, err := jsoniter.Marshal(interf)
	if err != nil {
		panic(err.Error())
	}

	b := sdbGet(0, 0, fullkey)
	b = data(fullkey, b)

	UTP.c.Assert(b, check.NotNil)
	UTP.c.Assert(b, check.DeepEquals, _v)
}

//AssertReceipt assert a receipt is existing
//判断测试结果包含某一特定收据，匹配完整收据格式
func AssertReceipt(interf interface{}) {
	_r := std.Receipt{}
	_s, err := jsoniter.Marshal(interf)
	if err != nil {
		panic(err.Error())
	}
	bMatch := false
Loop:
	for _, v := range UTP.Message().InputReceipts() {
		err := jsoniter.Unmarshal(v.Value, &_r)
		if err != nil {
			panic(err.Error())
		}
		if len(_r.Bytes) != len(_s) {
			continue Loop
		}

		for i := 0; i < len(_s); i++ {
			if _r.Bytes[i] != _s[i] {
				continue Loop
			}
		}
		//Find receipt
		bMatch = true
		break
	}

	UTP.c.Assert(true, check.Equals, bMatch)
}

//CheckError check error code is expected or not
func CheckError(err types.Error, expected int) {
	UTP.c.Check(err.ErrorCode, check.Equals, uint32(expected))
}

//CheckOK check error code is CodeOK or not
func CheckOK(err types.Error) {
	UTP.c.Check(err.ErrorCode, check.Equals, uint32(types.CodeOK))
}

//CheckErrorMsg check error message is expected or not
func CheckErrorMsg(err types.Error, msg string) {
	UTP.c.Check(err.Error(), check.Matches, "*"+msg+"*")
}

func checkKey(key string) error {
	if len(key) == 0 || key[0] != '/' {
		return errors.New(fmt.Sprintf("The key=%s is not prefix \"/\"", key))
	}

	return nil
}
