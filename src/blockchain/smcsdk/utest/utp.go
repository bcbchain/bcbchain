//unittestplatform
// utp.go 创建单元测试对象，并执行初始化.

package utest

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
	"gopkg.in/check.v1"
)

//UtPlatform test object
type UtPlatform struct {
	sdk.ISmartContract
	c           *check.C
	g           *genesis
	accountList []types.Address
}

var (
	//UTP declare
	UTP       *UtPlatform
	utChainID string
	utOrgID   string
	bTrans    bool // 是否按事务执行标记
)

// Init init when starting test case
func Init(orgID string) types.Error {
	UTP = &UtPlatform{}

	utOrgID = orgID
	var err error
	UTP.g, err = readGenesisFile()
	if err != nil {
		panic(err.Error())
	}
	setChainID(UTP.g.ChainID)

	return initGenesis(UTP.g)
}

// DeployContract deploy contract
func DeployContract(c *check.C, contractName, orgID string, methods, interfaces []string) sdk.IAccount {
	logger := InitLog(contractName)
	UTP.c = c
	UTP.c.SetSkipLevel(3)

	return deployContract(contractName, orgID, methods, interfaces, logger)
}

// Commit commit
func Commit() {
	if UTP.ISmartContract != nil {
		UTP.ISmartContract.(sdkimpl.ISDB).Commit()
	}
	if bTrans == true {
		commitState()
	}
}

func Rollback() {
	UTP.ISmartContract.(sdkimpl.ISDB).Commit()
	rollbackState()
}

func GetFlag() bool {
	return bTrans
}

func SetFlag(b bool) {
	bTrans = b
}
