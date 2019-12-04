package helper

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
	"blockchain/smcsdk/sdkimpl/object"
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

// GenesisHelper genesis helper information
type GenesisHelper struct {
	smc sdk.ISmartContract //指向智能合约API对象指针

	contractList []sdk.IContract //创世合约信息
	token        sdk.IToken      //创世通证（基础通证）的信息
}

var _ sdk.IGenesisHelper = (*GenesisHelper)(nil)
var _ sdkimpl.IAcquireSMC = (*GenesisHelper)(nil)

// SMC get smart contract object
func (gh *GenesisHelper) SMC() sdk.ISmartContract { return gh.smc }

// SetSMC set smart contract object
func (gh *GenesisHelper) SetSMC(smc sdk.ISmartContract) { gh.smc = smc }

// ChainID get chainID with current block chain
func (gh *GenesisHelper) ChainID() string {
	return *gh.smc.(*sdkimpl.SmartContract).LlState().McGetEx(std.KeyOfChainID(), new(string)).(*string)
}

// OrgID get organization identifier
func (gh *GenesisHelper) OrgID() string {
	return *gh.smc.(*sdkimpl.SmartContract).LlState().McGetEx(std.KeyOfOrgID(), new(string)).(*string)
}

// Contracts get genesis contract list
func (gh *GenesisHelper) Contracts() []sdk.IContract {
	if gh.contractList == nil {
		gh.contractList = gh.genesisContracts(&std.Contract{})
	}

	return gh.contractList
}

// Token get genesis token
func (gh *GenesisHelper) Token() sdk.IToken {
	if gh.token == nil {
		key := std.KeyOfGenesisToken()

		stdToken := gh.smc.(*sdkimpl.SmartContract).LlState().McGet(key, &std.Token{})
		if stdToken != nil {
			gh.token = object.NewTokenFromSTD(gh.smc, stdToken.(*std.Token))
		} else {
			err := errors.New("[sdk]Please genesis first")
			sdkimpl.Logger.Fatalf(err.Error())
			sdkimpl.Logger.Flush()
			panic(err)
		}
	}

	return gh.token
}

func (gh *GenesisHelper) GasPriceRatio() uint64 {

	key := std.KeyOfGasPriceRatio()
	gasPriceRatio := *gh.smc.(*sdkimpl.SmartContract).LlState().McGetEx(key, new(string)).(*string)
	if gasPriceRatio == "" {
		gasPriceRatio = "1.000"
	}

	gasPriceRatio = strings.Replace(gasPriceRatio, ".", "", -1)
	uGasPriceRatio, err := strconv.ParseUint(gasPriceRatio, 10, 64)
	if err != nil {
		panic(err)
	}

	return uGasPriceRatio
}

// genesisContractAddrList get genesis contract address list
func (gh *GenesisHelper) genesisContractAddrList() []types.Address {
	keyOfContractAddrs := std.KeyOfGenesisContractAddrList()

	return gh.smc.(*sdkimpl.SmartContract).LlState().GetStrings(keyOfContractAddrs)
}

// genesisContracts get genesis contract list
func (gh *GenesisHelper) genesisContracts(defaultVal interface{}) []sdk.IContract {
	addrList := gh.genesisContractAddrList()
	if addrList == nil {
		return nil
	}

	contracts := make([]sdk.IContract, len(addrList))
	for index, contractAddr := range addrList {
		keyOfContract := std.KeyOfGenesisContract(contractAddr)

		// get genesis contract object
		stdContract := gh.smc.(*sdkimpl.SmartContract).LlState().McGet(keyOfContract, defaultVal)
		if stdContract == nil {
			err := errors.New("[sdk]Please genesis first")
			sdkimpl.Logger.Fatalf(err.Error())
			sdkimpl.Logger.Flush()
			panic(err)
		}
		contracts[index] = object.NewContractFromSTD(gh.smc, stdContract.(*std.Contract))
	}

	return contracts
}
