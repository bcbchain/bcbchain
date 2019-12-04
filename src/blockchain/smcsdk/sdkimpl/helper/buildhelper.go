package helper

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
)

// BuildHelper build helper information
type BuildHelper struct {
	smc sdk.ISmartContract
}

var _ sdk.IBuildHelper = (*BuildHelper)(nil)
var _ sdkimpl.IAcquireSMC = (*BuildHelper)(nil)

const smartContractName = "smartcontract"
const genesisName = "genesis"

// SMC get smartContract object
func (bh *BuildHelper) SMC() sdk.ISmartContract { return bh.smc }

// SetSMC set smartContract object
func (bh *BuildHelper) SetSMC(smc sdk.ISmartContract) { bh.smc = smc }

// Build build smartContract code and return result
func (bh *BuildHelper) Build(metas std.ContractMeta) (buildResult std.BuildResult) {
	if bh.smc.Message().Contract().Address() != std.GetGenesisContractAddr(bh.smc.Block().ChainID()) {
		contractName := bh.smc.Message().Contract().Name()
		if contractName != smartContractName && contractName != genesisName {
			buildResult.Code = types.ErrNoAuthorization
			return
		}

		contractOrgID := bh.smc.Message().Contract().OrgID()
		genesisOrgID := bh.smc.Helper().GenesisHelper().OrgID()
		if contractOrgID != genesisOrgID {
			buildResult.Code = types.ErrNoAuthorization
			return
		}
	}

	return sdkimpl.BuildFunc(bh.smc.(*sdkimpl.SmartContract).LlState().TransID(), bh.smc.(*sdkimpl.SmartContract).LlState().TxID(), metas)
}
