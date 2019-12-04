package prototype

const (
	System          = "system"
	TokenBasic      = "token-basic"
	TokenIssue      = "token-issue"
	TokenTemplet    = "token-templet"
	TokenTrade      = "token-trade"
	MINING          = "mining"
	TB_Cancellation = "token-basic-cancellation-1"
	UPGRADE1TO2     = "upgrade1to2"
	TokenBYB        = "token-byb"
	BlackList       = "black-list"
	TAC             = "transferAgency"
	TB_Team         = "token-basic-team"
	TB_Foundation   = "token-basic-foundation"

	SysNewValidator           = "NewValidator(string,smc.PubKey,smc.Address,uint64)smc.Error"
	SysSetPower               = "SetPower(smc.PubKey,uint64)smc.Error"
	SysSetRewardAddr          = "SetRewardAddr(smc.PubKey,smc.Address)smc.Error"
	SysForbidInternalContract = "ForbidInternalContract(smc.Address,uint64)smc.Error"
	SysDeployInternalContract = "DeployInternalContract(string,string,[]string,[]uint64,smc.Hash,uint64)(smc.Address,smc.Error)"
	SysSetRewardStrategy      = "SetRewardStrategy(string,uint64)smc.Error"

	TbTransfer        = "Transfer(smc.Address,big.Int)smc.Error"
	TbSetGasBasePrice = "SetGasBasePrice(uint64)smc.Error"
	TbSetGasPrice     = "SetGasPrice(uint64)smc.Error"

	TiNewToken = "NewToken(string,string,big.Int,bool,bool,uint64)(smc.Address,smc.Error)"

	TtTransfer      = "Transfer(smc.Address,big.Int)smc.Error"
	TtBatchTransfer = "BatchTransfer([]smc.Address,big.Int)smc.Error"
	TtAddSupply     = "AddSupply(big.Int)smc.Error"
	TtBurn          = "Burn(big.Int)smc.Error"
	TtSetGasPrice   = "SetGasPrice(uint64)smc.Error"
	TtSetOwner      = "SetOwner(smc.Address)smc.Error"
	TtUdcCreate     = "UdcCreate(smc.Address,big.Int,string)(smc.Hash,smc.Error)"
	TtUdcTransfer   = "UdcTransfer(smc.Hash,smc.Address,big.Int)(smc.Hash,smc.Hash,smc.Error)"
	TtUdcMature     = "UdcMature(smc.Hash)smc.Error"
	TtDealExchange  = "DealExchange(string,string)smc.Error"

	BYBInit                  = "Init(Number,bool,bool)smc.Error"
	BYBSetOwner              = "SetOwner(smc.Address)smc.Error"
	BYBSetGasPrice           = "SetGasPrice(uint64)smc.Error"
	BYBAddSupply             = "AddSupply(Number)smc.Error"
	BYBBurn                  = "Burn(Number)smc.Error"
	BYBNewBlackHole          = "NewBlackHole(smc.Address)smc.Error"
	BYBNewStockHolder        = "NewStockHolder(smc.Address,Number)(smc.Chromo,smc.Error)"
	BYBDelStockHolder        = "DelStockHolder(smc.Address)smc.Error"
	BYBChangeChromoOwnerShip = "ChangeChromoOwnership(smc.Chromo,smc.Address)smc.Error"
	BYBTransfer              = "Transfer(smc.Address,big.Int)smc.Error"
	BYBTransferByChromo      = "TransferByChromo(smc.Chromo,smc.Address,Number)smc.Error"

	BlmSetOwner   = "SetOwner(smc.Address)smc.Error"
	BlmAddAddress = "AddAddress([]smc.Address)smc.Error"
	BlmDelAddress = "DelAddress([]smc.Address)smc.Error"

	UPGRADE1TO2Upgrade = "Upgrade(string)(string,smc.Error)"

	TBCCancel = "Cancel()smc.Error"

	MNMine      = "Mine()smc.Error"
	TBTWithdraw = "Withdraw()smc.Error"
	TBFWithdraw = "Withdraw()smc.Error"

	TACSetManager    = "SetManager([]smc.Address)smc.Error"
	TACSetTokenFee   = "SetTokenFee(string)smc.Error"
	TACTransfer      = "Transfer(string,smc.Address,Number)smc.Error"
	TACWithdrawFunds = "WithdrawFunds(string,Number)smc.Error"
)
