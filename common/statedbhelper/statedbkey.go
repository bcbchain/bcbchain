package statedbhelper

import "github.com/bcbchain/bclib/types"

func keyOfWorldAppState() string {
	return "/world/appstate"
}

func keyOfGenesisChainID() string {
	return "/genesis/chainid"
}

func keyOfRewardStrategy() string {
	return "/rewardstrategys"
}

func KeyOfAccountNonce(exAddress types.Address) string {
	return "/account/ex/" + exAddress + "/account"
}

func KeyOfAccountToken(exAddress types.Address, contractAddr types.Address) string {
	return "/account/ex/" + exAddress + "/token/" + contractAddr
}

func KeyOfAccount(exAddress types.Address) string {
	return "/account/ex/" + exAddress
}

func keyOfContract(addr types.Address) string {
	return "/contract/" + addr
}

func keyOfValidators() string {
	return "/validators/all/0"
}

func keyOfValidator(nodeAddr types.Address) string {
	return "/validator/" + nodeAddr
}

func keyOfBlackList(addr types.Address) string {
	return "/blacklist/" + addr
}

func keyOfGenesisContract() string {
	return "/contract/genesis"
}

func keyOfContractOrgID(orgID, name string) string {
	return "/contract/" + orgID + "/" + name
}

func keyOfContractWithHeight(height string) string {
	return "/" + height
}

func keyOfContractMeta(addr types.Address) string {
	return "/contract/code/" + addr
}
func keyOfOrganization(orgID string) string {
	return "/organization/" + orgID
}

func keyOfMineContracts() string {
	return "/contract/mines"
}

func KeyOfToken(tokenAddr types.Address) string {
	return "/token/" + tokenAddr
}

func keyOfGasPriceRatio() string {
	return "/genesis/gaspriceratio"
}

func keyOfGenesisOrgID() string {
	return "/genesis/orgid"
}
