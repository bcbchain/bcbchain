package genrpc

import (
	"blockchain/smccheck/parsecode"
	"bytes"
	"common/fs"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

const path = "/home/rustic/GIBlockChain/trunk/code/v2.0/bcsmc-sdk/src/contract/orgteststub/code/myplayerbook/v2.0/myplayerbook"

func TestVerifySign(t *testing.T) {
	b := VerifySign("testData/commitFile.tar.gz", "testData/signature.sn")
	assert.Equal(t, b, true)
}

func TestUnTarGz(t *testing.T) {
	e := os.MkdirAll("tmp", 0755)
	assert.Equal(t, e, nil)

	data, e := ioutil.ReadFile("testData/commitFile.tar.gz")
	assert.Equal(t, e, nil)

	e = fs.UnTarGz("tmp", bytes.NewReader(data), nil)
	assert.Equal(t, e, nil)

	e = os.RemoveAll("tmp")
	assert.Equal(t, e, nil)
}

func TestGenRPC(t *testing.T) {
	res, err := parsecode.Check(path)
	fmt.Println(err)
	assert.Equal(t, err.ErrorCode, uint32(200))

	e := GenRPC(res, 12321, "devtestNBH1CLAcg1eBpU78FK9fk1T9zYyYh11L5", "outdir")
	assert.Equal(t, e, nil)
}

func TestGenMarkdown(t *testing.T) {
	res, err := parsecode.Check(path)
	fmt.Println(err)
	assert.Equal(t, err.ErrorCode, uint32(200))

	e := GenMarkdown(res, 12321, "outdir")
	assert.Equal(t, e, nil)
}
