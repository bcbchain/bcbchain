package gen

import (
	"github.com/bcbchain/bcbchain/smccheck/parsecode"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCheck(t *testing.T) {
	const path = "/home/rustic/GIBlockChain/trunk/code/v2.0/bcsmc-sdk/src/contract/orgteststub/code/myplayerbook/v2.0/myplayerbook"
	res, err := parsecode.Check(path)
	fmt.Println(err)
	assert.Equal(t, err.ErrorCode, uint32(200))

	GenStore(path, res)

	GenSDK(path, res)

	GenReceipt(path, res)

	GenTypes(path, res)
}
