package gen

import (
	"blockchain/smccheck/parsecode"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCheck(t *testing.T) {
	const path = "/home/rustic/GIBlockChain/trunk/code/v2.0/bcsmc-sdk/src/contract/orgteststub/code/myplayerbook/v2.0/myplayerbook"
	res, err := parsecode.Check(path)
	fmt.Println(err)
	assert.Equal(t, err.ErrorCode, uint32(200))

	e := GenStore(path, res)
	assert.Equal(t, e, nil)

	e = GenSDK(path, res)
	assert.Equal(t, e, nil)

	e = GenReceipt(path, res)
	assert.Equal(t, e, nil)

	e = GenTypes(path, res)
	assert.Equal(t, e, nil)
}
