package helper

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
	"blockchain/smcsdk/sdkimpl/object"
	"fmt"
)

// ContractHelper contract helper information
type ContractHelper struct {
	smc sdk.ISmartContract //指向智能合约API对象指针
}

var _ sdk.IContractHelper = (*ContractHelper)(nil)
var _ sdkimpl.IAcquireSMC = (*ContractHelper)(nil)

// SMC get smart contract object
func (ch *ContractHelper) SMC() sdk.ISmartContract { return ch.smc }

// SetSMC set smart contract object
func (ch *ContractHelper) SetSMC(smc sdk.ISmartContract) { ch.smc = smc }

// ContractOfAddress get contract object with address
func (ch *ContractHelper) ContractOfAddress(address types.Address) sdk.IContract {
	sdk.RequireAddress(address)

	return ch.contractOfAddress(address)
}

// ContractOfToken get contract object with token address
func (ch *ContractHelper) ContractOfToken(tokenAddr types.Address) sdk.IContract {
	sdk.RequireAddress(tokenAddr)

	// the tokenAddr is a token's address
	if ch.smc.Helper().TokenHelper().TokenOfAddress(tokenAddr) == nil {
		return nil
	}

	// Initialize to the first contract it has in case all of its contracts are unenforced
	contract := ch.smc.Message().Contract()
	if contract.Address() == tokenAddr {
		return contract
	}

	keyOfContract := std.KeyOfContract(tokenAddr)
	other := ch.smc.(*sdkimpl.SmartContract).LlState().McGet(keyOfContract, &std.Contract{})
	if other == nil {
		return nil
	}

	otherContract := other.(*std.Contract)
	if ch.smc.Block().Height() >= otherContract.EffectHeight && (ch.smc.Block().Height() < otherContract.LoseHeight || otherContract.LoseHeight == 0) {
		return object.NewContractFromSTD(ch.smc, otherContract)
	}

	keyOfContractsWithName := std.KeyOfContractsWithName(otherContract.OrgID, otherContract.Name)
	contractList := ch.smc.(*sdkimpl.SmartContract).LlState().McGetEx(keyOfContractsWithName,
		&std.ContractVersionList{}).(*std.ContractVersionList)
	addr := ""
	for i := len(contractList.ContractAddrList) - 1; i >= 0; i-- {
		if ch.smc.Block().Height() >= contractList.EffectHeights[i] {
			addr = contractList.ContractAddrList[i]
			break
		}
	}

	c := ch.contractOfAddress(addr)
	if c.LoseHeight() != 0 && c.LoseHeight() < ch.smc.Block().Height() {
		return nil
	}

	return c
}

// ContractOfName get contract object with name
func (ch *ContractHelper) ContractOfName(name string) sdk.IContract {
	versionList := ch.contractVersionList(name)
	if versionList == nil {
		return nil
	}

	for i := len(versionList.ContractAddrList) - 1; i >= 0; i-- {
		// return effective contract
		if ch.smc.Block().Height() >= versionList.EffectHeights[i] {
			contract := ch.contractOfAddress(versionList.ContractAddrList[i])
			if contract.LoseHeight() != 0 && contract.LoseHeight() < ch.smc.Block().Height() {
				return nil
			}

			return contract
		}
	}

	return nil
}

// UpdateContractsToken update contract's token item
func (ch *ContractHelper) UpdateContractsToken(tokenAddr types.Address) {
	versionList := ch.contractVersionList(ch.smc.Message().Contract().Name())
	sdk.Require(versionList != nil,
		types.ErrInvalidParameter,
		fmt.Sprintf("TokenAddr=%s cannot get any contract", tokenAddr))

	// calculate loseHeight of all version contract, and last contract's lostHeight is zero
	loseHeights := make([]int64, len(versionList.EffectHeights))
	for index := range versionList.EffectHeights {
		if len(versionList.EffectHeights) > index+1 {
			loseHeights[index] = versionList.EffectHeights[index+1] - 1
		} else {
			loseHeights[index] = 0
		}
	}

	for index, addr := range versionList.ContractAddrList {
		if loseHeights[index] != 0 && ch.smc.Block().Height() >= loseHeights[index] {
			continue
		}

		// update token's value
		key := std.KeyOfContract(addr)
		contract := ch.smc.(*sdkimpl.SmartContract).LlState().McGet(key, &std.Contract{})
		contract.(*std.Contract).Token = tokenAddr
		ch.smc.(*sdkimpl.SmartContract).LlState().McSet(key, contract)
	}

	// update smc obtain contract's token
	ch.smc.Message().Contract().(*object.Contract).SetToken(tokenAddr)
}

// contractOfAddress get contract object with address
func (ch *ContractHelper) contractOfAddress(address types.Address) sdk.IContract {
	key := std.KeyOfContract(address)
	stdContract := ch.smc.(*sdkimpl.SmartContract).LlState().McGet(key, &std.Contract{})
	if stdContract == nil {
		return nil
	}

	contract := object.NewContractFromSTD(ch.smc, stdContract.(*std.Contract))

	return contract
}

// contractVersionList get address list of contract that map by name
func (ch *ContractHelper) contractVersionList(name string) *std.ContractVersionList {
	// get current contract's address list
	key := std.KeyOfContractsWithName(ch.smc.Message().Contract().OrgID(), name)
	versionList := ch.smc.(*sdkimpl.SmartContract).LlState().McGet(key, &std.ContractVersionList{})
	if versionList == nil {
		key = std.KeyOfContractsWithName(ch.smc.Helper().GenesisHelper().Contracts()[0].OrgID(), name)
		versionList = ch.smc.(*sdkimpl.SmartContract).LlState().McGet(key, &std.ContractVersionList{})
	}

	if versionList == nil {
		return nil
	}

	return versionList.(*std.ContractVersionList)
}
