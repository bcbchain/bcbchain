package helper

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdkimpl"
)

// Helper helper detail information
type Helper struct {
	smc sdk.ISmartContract //指向智能合约smc对象指针

	accountHelper    sdk.IAccountHelper    //账户相关的Helper对象
	blockChainHelper sdk.IBlockChainHelper //区块链相关的Helper对象
	contractHelper   sdk.IContractHelper   //合约相关的Helper对象
	receiptHelper    sdk.IReceiptHelper    //收据相关的Helper对象
	genesisHelper    sdk.IGenesisHelper    //创世相关的Helper对象
	stateHelper      sdk.IStateHelper      //状态相关的Helper对象
	tokenHelper      sdk.ITokenHelper      //通证相关的Helper对象
	buildHelper      sdk.IBuildHelper      //编译相关的Helper对象
	ibcHelper        sdk.IIBCHelper        //跨链数据相关Helper对象
	ibcStubHelper    sdk.IIBCStubHelper    //跨链执行相关Helper对象
}

var _ sdk.IHelper = (*Helper)(nil)

// SMC get smart contract object
func (h *Helper) SMC() *sdkimpl.SmartContract { return h.smc.(*sdkimpl.SmartContract) }

// SetSMC set smart contract object
func (h *Helper) SetSMC(smc sdk.ISmartContract) { h.smc = smc }

// AccountHelper get AccountHelper object
func (h *Helper) AccountHelper() sdk.IAccountHelper { return h.accountHelper }

// BlockChainHelper get BlockChainHelper object
func (h *Helper) BlockChainHelper() sdk.IBlockChainHelper { return h.blockChainHelper }

// ContractHelper get ContractHelper object
func (h *Helper) ContractHelper() sdk.IContractHelper { return h.contractHelper }

// ReceiptHelper get ReceiptHelper object
func (h *Helper) ReceiptHelper() sdk.IReceiptHelper { return h.receiptHelper }

// GenesisHelper get GenesisHelper object
func (h *Helper) GenesisHelper() sdk.IGenesisHelper { return h.genesisHelper }

// StateHelper get StateHelper object
func (h *Helper) StateHelper() sdk.IStateHelper { return h.stateHelper }

// TokenHelper get TokenHelper object
func (h *Helper) TokenHelper() sdk.ITokenHelper { return h.tokenHelper }

// BuildHelper get BuildHelper object
func (h *Helper) BuildHelper() sdk.IBuildHelper { return h.buildHelper }

// IBCHelper get IBCHelper object
func (h *Helper) IBCHelper() sdk.IIBCHelper { return h.ibcHelper }

// IBCStubHelper get IBCStubHelper object
func (h *Helper) IBCStubHelper() sdk.IIBCStubHelper { return h.ibcStubHelper }
