package builderhelper

import (
	"github.com/bcbchain/bcbchain/smcbuilder"
	"github.com/bcbchain/sdk/sdk/std"
)

func AdapterBuildCallBack(transID, txID int64, contractMeta std.ContractMeta) (result *std.BuildResult, err error) {
	b := smcbuilder.GetInstance()
	result1 := b.BuildContract(transID, txID, contractMeta)
	result = &result1

	return
}
