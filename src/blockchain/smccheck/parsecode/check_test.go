package parsecode

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	path1 = "/home/rustic/GIBlockChain/trunk/code/v2.0/bcsmc-sdk/src/contract/orgteststub/code/myplayerbook/v2.0/myplayerbook"
	path2 = "/home/rustic/GIBlockChain/trunk/code/v2.0/bcsmc-sdk/src/contract/orgexample/code/mydice2win/v1.0/mydice2win"
)

func hasTransfer(res *Result) bool {
	for _, fun := range res.Functions {
		if fun.GetTransferToMe {
			return true
		}
	}
	return false
}

func TestCheck(t *testing.T) {

	// path1
	res, err := Check(path1)
	fmt.Println(err)
	assert.Equal(t, err.ErrorCode, uint32(200))

	assert.Equal(t, hasTransfer(res), true)

	// path2
	res, err = Check(path2)
	fmt.Println(err)
	assert.Equal(t, err.ErrorCode, uint32(200))

	assert.Equal(t, hasTransfer(res), true)

}
