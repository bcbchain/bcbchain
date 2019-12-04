package system

import (
	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/contract"
	"blockchain/abciapp_v1.0/contract/smcapi"
	"blockchain/abciapp_v1.0/smc"
	"blockchain/smcsdk/sdk/std"
	"encoding/json"
	"fmt"
	"strings"
)

// ValidatorManager is a reference of Contract structure
type System struct {
	*contract.Contract
}

//NewValidator set an observer as validator
func (contract *System) NewValidator(name string, pubKey smc.PubKey, rewardAddr smc.Address, power uint64) (smcError smc.Error) {

	if contract.Ctx.BlockHeader.ChainVersion != 0 {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = "this chain has upgraded, now chainVersion is 2"
		return
	}

	if len(name) == 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid name: validator's name cannot be empty"
		return
	} else if power == 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid power: validator's power cannot be ZERO"
		return
	}
	// Check sender's permission
	sender := contract.Sender()
	owner := contract.Owner()
	if sender.Address() != owner.Address() {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}

	// Check if validator is existing or not
	validatorMgr := contract.ValidatorMgr()
	if validatorMgr.Has(pubKey) {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid public key: validator's public key is already used"
		return
	}
	// Check duplicate name
	if smcError = validatorMgr.CheckNameDuplicate(name); smcError.ErrorCode != bcerrors.ErrCodeOK {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid name: validator's name is already used"
		return
	}

	if smcError = validatorMgr.CheckRewardAddress(rewardAddr); smcError.ErrorCode != bcerrors.ErrCodeOK {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid name: validator's reward address is already used"
		return
	}

	return validatorMgr.NewValidator(name, pubKey, rewardAddr, power)
}

//SetPower sets a validator's power, 0(zero) means setting it as an observer
func (contract *System) SetPower(pubKey smc.PubKey, power uint64) (smcError smc.Error) {

	if contract.Ctx.BlockHeader.ChainVersion != 0 {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = "this chain has upgraded, now chainVersion is 2"
		return
	}

	// Check sender's permission
	sender := contract.Sender()
	owner := contract.Owner()
	if sender.Address() != owner.Address() {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}
	// Check if validator is existing or not
	validatorMgr := contract.ValidatorMgr()
	if !validatorMgr.Has(pubKey) {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid validator: it doesn't exist"
		return
	}
	// Modify its power
	return validatorMgr.SetPower(pubKey, power)
}

//SetRewardAddr sets a validator's reward address, what is using to receive the rewards
func (contract *System) SetRewardAddr(pubKey smc.PubKey, rewardAddr smc.Address) (smcError smc.Error) {

	if contract.Ctx.BlockHeader.ChainVersion != 0 {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = "this chain has upgraded, now chainVersion is 2"
		return
	}

	// Check sender's permission
	sender := contract.Sender()
	owner := contract.Owner()
	if sender.Address() != owner.Address() {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}
	// Check if validator is existing or not
	validatorMgr := contract.ValidatorMgr()
	if !validatorMgr.Has(pubKey) {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid validator: it doesn't exist"
		return
	}
	// Modify its power
	return validatorMgr.SetRewardAddr(pubKey, rewardAddr)
}

// SetRewardStrategy sets a validator's reward strategy, only the owner of contract is allowed to execute
func (contract *System) SetRewardStrategy(strategy string, effectHeight uint64) (smcError smc.Error) {

	if contract.Ctx.BlockHeader.ChainVersion != 0 {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = "this chain has upgraded, now chainVersion is 2"
		return
	}

	// Check sender's permission
	sender := contract.Sender()
	owner := contract.Owner()
	if sender.Address() != owner.Address() {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}

	// Check if the inputted effectHeight is valid
	// the effectHeight should larger than current block height and setting of last strategy
	if smcError = contract.CheckEffectHeight(effectHeight); smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}
	// Check if the strategy is valid
	// the effectHeight should larger than current block height and setting of last strategy
	if smcError = contract.CheckRewardStrategy(strategy); smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}
	// Set strategy
	return contract.UpdateRewardStrategy(strategy, effectHeight)
}

func (contract *System) DeployInternalContract(
	name string,
	version string,
	protoTypes []string,
	gasList []uint64,
	codeHash smc.Hash,
	effectHeight uint64) (contractAddr smc.Address, smcError smc.Error) {

	// if chain upgraded then system cannot deploy new contract and
	// upgrade contract except that genesis
	if contract.isDisabled(name) {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = "contract is disabled"
		return
	}

	// Check sender's permission
	sender := contract.Sender()
	owner := contract.Owner()
	if sender.Address() != owner.Address() {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}

	// Checking parameters
	if len(name) == 0 || len(version) == 0 || effectHeight == 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		return "", smcError
	}

	sapi := smcapi.SmcApi{
		Sender: contract.Ctx.Sender,
		Owner:  contract.Ctx.Owner,
		State:  contract.Ctx.TxState,
		Block:  nil}
	smcError = sapi.CheckParameterGasAndPrototype(protoTypes, gasList)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	blockHeight, _ := sapi.GetCurrentBlockHeight()
	if int64(effectHeight) <= blockHeight {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "The specified block height is less than current height"
		return "", smcError
	}

	bForbid, smcError := sapi.CheckAndForbidOldVersionContract(name, version, effectHeight)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	if bForbid == true {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = "contract forbidden"
		return
	}

	// Calculate contract address, and set contract
	contractAddr = sapi.CalcContractAddress(name, version)

	// added for v2 chain
	orgID, conVerList := contract.getOrgIDAndConVerList(name)
	if contract.Ctx.BlockHeader.ChainVersion != 0 {
		conVerList.Name = name
		conVerList.ContractAddrList = append(conVerList.ContractAddrList, contractAddr)
		conVerList.EffectHeights = append(conVerList.EffectHeights, int64(effectHeight))

		resBytes, err := json.Marshal(conVerList)
		if err != nil {
			smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
			smcError.ErrorDesc = err.Error()
			return
		}

		err = contract.Ctx.TxState.Set(std.KeyOfContractsWithName(orgID, name), resBytes)
		if err != nil {
			smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
			smcError.ErrorDesc = err.Error()
			return
		}
	}
	err := sapi.SetNewContract(contractAddr, name, version, protoTypes, gasList, codeHash, effectHeight)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return "", smcError
	}
	smcError.ErrorCode = bcerrors.ErrCodeOK
	return
}

func (contract *System) ForbidInternalContract(contractAddr smc.Address, effectHeight uint64) (smcError smc.Error) {

	// Check sender's permission
	sender := contract.Sender()
	owner := contract.Owner()
	if sender.Address() != owner.Address() {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}

	con, err := contract.Ctx.TxState.StateDB.GetContract(contractAddr)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = err.Error()
		return
	}

	if con.ChainVersion != 0 {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = fmt.Sprintf("cannot forbid contract deploy by chain version %d", con.ChainVersion)
		return
	}

	sapi := smcapi.SmcApi{
		Sender: contract.Ctx.Sender,
		Owner:  contract.Ctx.Owner,
		State:  contract.Ctx.TxState,
		Block:  nil} //TODO

	blockHeight, _ := sapi.GetCurrentBlockHeight()
	if int64(effectHeight) <= blockHeight {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "The specified block height is less than current height"
		return
	}

	smcError = sapi.ForbidSpecificContract(contractAddr, effectHeight)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	smcError.ErrorCode = bcerrors.ErrCodeOK
	return
}

func (contract *System) isDisabled(name string) bool {
	// if chain upgraded then system cannot deploy new contract and
	// upgrade contract except that genesis
	if contract.Ctx.BlockHeader.ChainVersion != 0 {
		if strings.HasPrefix(name, "token-templet-") {
			return true
		}

		addrList, err := contract.Ctx.TxState.StateDB.GetGenesisContractList()
		if err != nil {
			panic(err)
		}

		for _, address := range addrList {
			genCon, err := contract.Ctx.TxState.StateDB.GetContract(address)
			if err != nil {
				panic(err)
			}

			if genCon.Name == name {
				return true
			}
		}
	}

	oldContracts, err := contract.Ctx.TxState.GetContractsListByName(name)
	if err != nil {
		panic(err)
	}

	forbidFlag := true
	if len(oldContracts) == 0 {
		forbidFlag = false
	}

	for _, conAddr := range oldContracts {
		con, err := contract.Ctx.TxState.StateDB.GetContract(conAddr)
		if err != nil {
			panic(err)
		}

		if con.ChainVersion != 0 {
			return true
		}

		if con.LoseHeight == 0 {
			forbidFlag = false
		}
	}

	return forbidFlag
}

// if chainVersion not zero, get orgID and contract version data
func (contract *System) getOrgIDAndConVerList(name string) (orgID string, conVerList std.ContractVersionList) {
	if contract.Ctx.BlockHeader.ChainVersion != 0 {
		value, err := contract.Ctx.TxState.Get(std.KeyOfOrgID())
		if err != nil {
			panic(err)
		}

		err = json.Unmarshal(value, &orgID)
		if err != nil {
			panic(err)
		}

		value, err = contract.Ctx.TxState.Get(std.KeyOfContractsWithName(orgID, name))
		if err != nil {
			panic(err)
		}

		if len(value) != 0 {
			err = json.Unmarshal(value, &conVerList)
			if err != nil {
				panic(err)
			}
		}
	}

	return
}
