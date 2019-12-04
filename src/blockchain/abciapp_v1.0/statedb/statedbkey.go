package statedb

import (
	"blockchain/abciapp_v1.0/smc"
	"encoding/hex"
	"github.com/tendermint/go-crypto"
	"strings"
)

func keyOfGenesisToken() string {
	return "/genesis/token"
}

func keyOfTokenAll() string {
	return "/token/all/0"
}

func keyOfGenesisContracts() string {
	return "/genesis/contracts"
}

func keyOfContractAll() string {
	return "/contract/all/0"
}

func keyOfGenesisChainId() string {
	return "/genesis/chainid"
}

func keyOfWorldAppState() string {
	return "/world/appstate"
}

func keyOfRewardStrategys() string {
	return "/rewardstrategys"
}

func keyOfGenesisContract(contractAddr smc.Address) string {
	return "/genesis/sc/" + contractAddr
}

func keyOfContract(contractAddr smc.Address) string {
	return "/contract/" + contractAddr
}

func keyOfToken(contractAddr smc.Address) string {
	return "/token/" + contractAddr
}

func keyOfAccountToken(exAddress smc.Address, contractAddr smc.Address) string {
	return "/account/ex/" + exAddress + "/token/" + contractAddr
}

func keyOfAccount(exAddress smc.Address) string {
	return "/account/ex/" + exAddress
}

func keyOfAccountNonce(exAddress smc.Address) string {
	return "/account/ex/" + exAddress + "/account"
}

func keyOfAccountContracts(exAddress smc.Address) string {
	return "/account/ex/" + exAddress + "/contracts"
}

func keyOfAccountUDCHashList(exAddress smc.Address) string {
	return "/account/ex/" + exAddress + "/udchashlist"
}

func keyOfTokenName(name string) string {
	return "/token/name/" + strings.ToLower(name)
}

func keyOfTokenSymbol(symbol string) string {
	return "/token/symbol/" + strings.ToLower(symbol)
}

func keyOfTokenBaseGasPrice() string {
	return "/token/basegasprice"
}

func keyOfUDCNonce() string {
	return "/udc/nonce"
}

func keyOfUDCOrder(udcHash crypto.Hash) string {
	return "/udc/nonce/" + hex.EncodeToString(udcHash)
}
func keyOfValidators() string {
	return "/validators/all/0"
}

func keyOfValidator(nodeAddr string) string {
	return "/validator/" + nodeAddr
}
