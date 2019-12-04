package smccheck

import (
	"blockchain/smccheck/gen"
	"blockchain/types"
	"fmt"
	"os"
	"testing"
)

func TestGen(t *testing.T) {
	err := os.RemoveAll("/Users/zerppen/GIBlockChain/trunk/code/v2.1/contract-debug/src/contract/stubcommon")
	if err != nil {
		fmt.Println(err)
	}
	path := "/Users/zerppen/GIBlockChain/trunk/code/v2.1/contract-debug/src/contract"

	var contractInfoList []gen.ContractInfo
	contractInfoList = append(contractInfoList, gen.ContractInfo{
		Name:         "myplayerbook",
		Version:      "1.0",
		EffectHeight: 50,
		LoseHeight:   1000,
	})

	contractInfoList = append(contractInfoList, gen.ContractInfo{
		Name:         "myplayerbook",
		Version:      "2.0",
		EffectHeight: 1000,
		LoseHeight:   0,
	})

	_, e := Gen(path, "ibc", "2.1", contractInfoList)
	if e.ErrorCode != types.CodeOK {
		fmt.Println(e.ErrorDesc)
	}
}
