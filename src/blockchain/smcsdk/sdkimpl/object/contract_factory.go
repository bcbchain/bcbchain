package object

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
)

// NewContractFromAddress factory method for create contract with address
func NewContractFromAddress(smc sdk.ISmartContract, conAddr types.Address) sdk.IContract {
	var contract sdk.IContract
	contract = smc.Helper().ContractHelper().ContractOfAddress(conAddr)
	if contract != nil {
		contract.(*Contract).SetSMC(smc)
	}

	return contract
}

// NewContractFromSTD factory method for create contract with address
func NewContractFromSTD(smc sdk.ISmartContract, stdContract *std.Contract) sdk.IContract {
	contract := &Contract{ct: *stdContract}
	contract.SetSMC(smc)

	return contract
}

// NewContract factory method for create contract with all property
func NewContract(smc sdk.ISmartContract,
	orgID string,
	ownerAddr types.Address,
	name, version, keyPrefix string,
	codeHash types.Hash,
	effectHeight, loseHeight int64,
	methods, interfaces []std.Method,
	token types.Address) sdk.IContract {

	contract := &Contract{
		ct: std.Contract{
			Address:      smc.Helper().BlockChainHelper().CalcContractAddress(name, version, orgID),
			Account:      smc.Helper().BlockChainHelper().CalcAccountFromName(name, orgID),
			Owner:        ownerAddr,
			Name:         name,
			Version:      version,
			CodeHash:     codeHash,
			EffectHeight: effectHeight,
			LoseHeight:   loseHeight,
			KeyPrefix:    keyPrefix,
			Methods:      methods,
			Interfaces:   interfaces,
			//Initialized:  initialized,
			Token: token,
		},
	}
	contract.SetSMC(smc)

	return contract
}
