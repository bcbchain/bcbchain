package builderhelper

import (
	"blockchain/smcbuilder"
	"blockchain/smcsdk/sdk/std"
)

func AdapterBuildCallBack(transID, txID int64, contractMeta std.ContractMeta) (result *std.BuildResult, err error) {
	b := smcbuilder.GetInstance()
	result1 := b.BuildContract(transID, txID, contractMeta)
	result = &result1

	return
}
