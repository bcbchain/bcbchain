package upgrade1to2

import (
	"blockchain/abciapp_v1.0/smc"
	"blockchain/abciapp_v1.0/types"
	"blockchain/common/statedbhelper"
	"blockchain/smcsdk/sdk/std"
	"common/jsoniter"
	abci "github.com/tendermint/abci/types"
)

func (u *Upgrade1to2) getContract(address smc.Address) (contract *std.Contract) {
	key := std.KeyOfContract(address)
	value, err := u.State.Get(key)
	if err != nil {
		panic(err.Error())
	}

	contract = new(std.Contract)
	if err = jsoniter.Unmarshal(value, contract); err != nil {
		panic(err.Error())
	}
	return
}

func (u *Upgrade1to2) getAllContract() (allAddress *[]smc.Address) {
	allAddress = new([]smc.Address)
	key := std.KeyOfAllContracts()

	if value, err := u.State.Get(key); err != nil {
		panic(err.Error())
	} else if err := jsoniter.Unmarshal(value, allAddress); err != nil {
		panic(err.Error())
	}
	return
}

func (u *Upgrade1to2) setAllContract(allAddress *[]smc.Address) {
	value, err := jsoniter.Marshal(allAddress)
	if err != nil {
		panic(err.Error())
	}
	if err := u.State.Set(std.KeyOfAllContracts(), value); err != nil {
		panic(err.Error())
	}
}

func (u *Upgrade1to2) setContract(contract *std.Contract) {
	value, err := jsoniter.Marshal(contract)
	if err != nil {
		panic(err.Error())
	}
	if err := u.State.Set(std.KeyOfContract(contract.Address), value); err != nil {
		panic(err.Error())
	}
}

func (u *Upgrade1to2) setContractMeta(meta *std.ContractMeta) error {
	value, err := jsoniter.Marshal(meta)
	if err != nil {
		return err
	}
	key := "/contract/code/" + meta.ContractAddr
	return u.State.Set(key, value)
}

func (u *Upgrade1to2) setContractVersionInfo(info *std.ContractVersionList, orgID string) {
	value, err := jsoniter.Marshal(info)
	if err != nil {
		panic(err.Error())
	}
	key := "/contract/" + orgID + "/" + info.Name
	if err := u.State.Set(key, value); err != nil {
		panic(err.Error())
	}
}

func (u *Upgrade1to2) setOrganization(org *std.Organization) {
	value, err := jsoniter.Marshal(org)
	if err != nil {
		panic(err.Error())
	}
	key := "/organization/" + org.OrgID
	if err := u.State.Set(key, value); err != nil {
		panic(err.Error())
	}
}

func (u *Upgrade1to2) getAppState() (*abci.AppState, error) {
	appState := new(abci.AppState)
	value, err := u.State.Get(std.KeyOfAppState())
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(value, appState)
	return appState, err
}

func (u *Upgrade1to2) setOrgAuthDeployContract(address *smc.Address) error {
	value, err := jsoniter.Marshal(address)
	if err != nil {
		return err
	}
	return u.State.Set("/organization/"+u.GenesisOrg.OrgID+"/auth", value)
}

func (u *Upgrade1to2) setAccountContractAddrs(accountAddr *smc.Address, addrs *[]smc.Address) {
	key := "/account/ex/" + *accountAddr + "/contracts"
	value, err := jsoniter.Marshal(addrs)
	if err != nil {
		panic(err)
	}
	if err = u.State.Set(key, value); err != nil {
		panic(err.Error())
	}
}

func (u *Upgrade1to2) getAccountContractAddrs(accountAddr *smc.Address) *[]smc.Address {
	key := "/account/ex/" + *accountAddr + "/contracts"
	value, err := u.State.Get(key)
	if err != nil {
		panic(err.Error())
	}
	allAddrs := new([]smc.Address)
	if err = jsoniter.Unmarshal(value, allAddrs); err != nil {
		panic(err)
	}
	return allAddrs
}
func (u *Upgrade1to2) setGenesisContracts(addrs *[]smc.Address) error {
	value, err := jsoniter.Marshal(addrs)
	if err != nil {
		return err
	}
	return u.State.Set(std.KeyOfGenesisContractAddrList(), value)
}

func (u *Upgrade1to2) setGenesisContract(contract *std.Contract) {
	value, err := jsoniter.Marshal(contract)
	if err != nil {
		panic(err.Error())
	}
	if err := u.State.Set(std.KeyOfGenesisContract(contract.Address), value); err != nil {
		panic(err.Error())
	}
}
func (u *Upgrade1to2) getGenesisContract(contractAddr smc.Address) *std.Contract {
	value, err := u.State.Get(std.KeyOfGenesisContract(contractAddr))
	if err != nil {
		panic(err.Error())
	}
	contract := new(std.Contract)
	err = jsoniter.Unmarshal(value, contract)
	if err != nil {
		panic(err.Error())
	}
	return contract
}

func (u *Upgrade1to2) getGenesisContracts() *[]smc.Address {
	addrs := new([]smc.Address)
	value, err := u.State.Get(std.KeyOfGenesisContractAddrList())
	if err != nil {
		panic(err.Error())
	}
	err = jsoniter.Unmarshal(value, addrs)
	return addrs
}

func (u *Upgrade1to2) newV2TransactionID() int64 {
	return statedbhelper.NewTransactionID()
}

func (u *Upgrade1to2) setContractToV2Cache(transID, txID int64, contract *std.Contract) {
	statedbhelper.SetContract(transID, txID, contract)
}

func (u *Upgrade1to2) setContractMetaToV2Cache(transID, txID int64, contract *std.ContractMeta) {
	statedbhelper.SetContractMeta(transID, txID, contract)
}

func (u *Upgrade1to2) setContractVersionListToV2Cache(transID, txID int64, orgID string, v *std.ContractVersionList) {
	statedbhelper.SetContractVersionList(transID, txID, orgID, v)
}

func (u *Upgrade1to2) setOrganizationToV2Cache(transID, txID int64, org *std.Organization) {
	statedbhelper.SetOrganization(transID, txID, org)
}

func (u *Upgrade1to2) rollbackV2(transID int64) {
	statedbhelper.RollbackBlock(transID)
}

func (u *Upgrade1to2) setGenesisOrgID(orgID string) {
	key := "/genesis/orgid"
	if value, err := jsoniter.Marshal(orgID); err != nil {
		panic(err.Error())
	} else if err := u.State.Set(key, value); err != nil {
		panic(err.Error())
	}
}

func (u *Upgrade1to2) getGenesisToken() *types.IssueToken {
	token, err := u.State.GetGenesisToken()
	if err != nil {
		panic(err.Error())
	}
	return token
}

func (u *Upgrade1to2) getChainID() string {
	value, err := u.State.Get(std.KeyOfChainID())
	if err != nil {
		panic(err.Error())
	}
	return string(value)
}

func (u *Upgrade1to2) setMineContract(m []std.MineContract) {
	if value, err := jsoniter.Marshal(m); err != nil {
		panic(err.Error())
	} else if err := u.State.Set(std.KeyOfMineContracts(), value); err != nil {
		panic(err.Error())
	}
}

func (u *Upgrade1to2) getEffectHeightContractAddrs(height string) (contractWithHeight []std.ContractWithEffectHeight) {
	value, err := u.State.Get(std.KeyOfContractWithEffectHeight(height))
	if err != nil {
		panic(err.Error())
	}

	if len(value) == 0 {
		return
	}

	err = jsoniter.Unmarshal(value, &contractWithHeight)
	if err != nil {
		panic(err.Error())
	}
	return
}

func (u *Upgrade1to2) setEffectHeightContractAddrs(height string, contractWithHeight []std.ContractWithEffectHeight) {
	key := std.KeyOfContractWithEffectHeight(height)
	value, err := jsoniter.Marshal(contractWithHeight)
	if err != nil {
		panic(err)
	}
	if err = u.State.Set(key, value); err != nil {
		panic(err.Error())
	}
}

func (u *Upgrade1to2) getAllValidator() []types.Validator {
	value, err := u.State.Get(keyOfValidators())
	if err != nil {
		panic(err.Error())
	}

	if len(value) == 0 {
		return nil
	}

	var nodeAddrs []string
	err = jsoniter.Unmarshal(value, &nodeAddrs)
	if err != nil {
		panic(err)
	}
	var validators = make([]types.Validator, 0)
	var validator types.Validator
	for _, nodeAddr := range nodeAddrs {
		val, err := u.State.Get(keyOfValidator(nodeAddr))
		if err != nil {
			panic(err)
		}
		err = jsoniter.Unmarshal(val, &validator)
		if err != nil {
			panic(err)
		}

		validators = append(validators, validator)
	}
	return validators
}

func (u *Upgrade1to2) setValidator(validator types.Validator) {
	key := keyOfValidator(validator.NodeAddr)

	value, err := jsoniter.Marshal(validator)
	if err != nil {
		panic(err)
	}

	err = u.State.Set(key, value)
	if err != nil {
		panic(err)
	}
}

func keyOfValidators() string {
	return "/validators/all/0"
}

func keyOfValidator(nodeAddr string) string {
	return "/validator/" + nodeAddr
}
