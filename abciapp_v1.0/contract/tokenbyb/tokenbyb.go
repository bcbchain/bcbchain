package tokenbyb

import (
	"math/big"
	"strconv"

	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/smcapi"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/types"
	. "github.com/bcbchain/bclib/bignumber_v1.0"
)

const (
	maxStockHolder = 100 // the max number of stockHolders
	BybName        = "BYB"
	bybSymbol      = "BYB"
	minTotalSupply = 1E9
)
const (
	addressRole_Committee = 1 + iota
	addressRole_StockHolder
	addressRole_Hole
	addressRole_User
)
const (
	white_Chromo = "0"
)

type TokenByb struct {
	*smcapi.SmcApi
}

/*
// Init byb token with totalSupply、addSupplyEnabled、burnEnabled
*/
func (byb *TokenByb) Init(totalSupply Number, addSupplyEnabled, burnEnabled bool) (smcError smc.Error) {
	if byb.Sender.Addr != byb.Owner.Addr {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}

	if totalSupply.Cmp_(minTotalSupply) < 0 {
		// totalSupply must great 1E9
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidSupply
		return
	}

	// check name and symbol
	if smcError = byb.checkBybToken(); smcError.ErrorCode != bcerrors.ErrCodeOK {
		return smcError
	}

	smcError = byb.initBybToken(*totalSupply.Value(), addSupplyEnabled, burnEnabled)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	return
}

func (byb *TokenByb) NewBlackHole(blackHole smc.Address) (smcError smc.Error) {
	smcError.ErrorCode = bcerrors.ErrCodeOK

	if byb.Sender.Addr != byb.Owner.Addr {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}

	addrRole, smcError := byb.checkAddress(blackHole)
	if addrRole == addressRole_Committee ||
		addrRole == addressRole_StockHolder ||
		addrRole == addressRole_Hole {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Cannot set it as black hole"
		return smcError
	}

	smcError = byb.updateBlackHole(blackHole)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}
	byb.bybReceipt_onNewBlackHole(blackHole)

	return
}

// add a new stockHolder of byb ,
// only the byb's owner can call the method,
// if the address is exist ,
// add new byb to the stockHolder
func (byb *TokenByb) NewStockHolder(stockHolder smc.Address, value Number) (chromo smc.Chromo, smcError smc.Error) {
	smcError.ErrorCode = bcerrors.ErrCodeOK
	//judge that sender is the owner of byb contract or not
	if byb.Sender.Addr != byb.Owner.Addr {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}
	// The value cannot be negative
	if value.Cmp_(0) <= 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid value: it cannot be a negative"
		return
	}
	//get the bybBalance of the owner
	ownerBalance, err := byb.State.GetBalance(byb.ContractAcct.Addr, byb.State.ContractAddress) //获取全部byb余额
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	// Value cannot to be bigger than the owner's balance
	if value.Cmp(NB(&ownerBalance)) >= 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid value: it is over the owner BYB balance"
		return
	}

	bybBalances, smcError := byb.makeStockHolderAndBybBalance(stockHolder, *value.Value())
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return chromo, smcError
	}

	//set the stockHolder's and sender's byb balances to stateDB
	smcError = byb.setOwnerAndStockHolderBalance(stockHolder, bybBalances, *value.Value(), ownerBalance)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}
	byb.bybReceipt_onNewStockHolder(stockHolder, bybBalances)

	return bybBalances[0].Chromo, smcError
}

func (byb *TokenByb) DelStockHolder(stockHolder smc.Address) (smcError smc.Error) {

	//judge that sender is the owner of byb contract or not
	if byb.Sender.Addr != byb.Owner.Addr {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}

	//get the bybBalance of the owner
	balance, err := byb.getBybBalance(stockHolder) //获取全部byb余额
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	if balance != nil && len(balance) >= 1 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsBybOwnedByb
		return
	}

	smcError = byb.delStockHolder(stockHolder)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}
	byb.bybReceipt_onDelStockHolder(stockHolder)

	return
}

// Ordinary transfer
// Common user calls this function to transfer byb
func (byb *TokenByb) Transfer(to smc.Address, value big.Int) (smcError smc.Error) {
	//The amount of the transfer is less than 0
	if Compare(value, Zero()) <= 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid value: it cannot be a negative"
		return
	}

	// The receiving address cannot be itself
	// And cannot transfer to contract account
	if byb.Sender.Addr == to || byb.ContractAcct.Addr == to {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsUnsupportTransToSelf
		return
	}
	// Checking "to" address role
	addrRole, smcError := byb.checkAddress(to)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return smcError
	}
	if addrRole == addressRole_StockHolder ||
		addrRole == addressRole_Committee {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid address, cannot transfer byb to this address"
		return
	}

	// Checking "Sender" address role
	addrRole, smcError = byb.checkAddress(byb.Sender.Addr)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return smcError
	}
	if addrRole == addressRole_StockHolder ||
		addrRole == addressRole_Committee {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid address, cannot transfer byb from this address"
		return
	}

	transByb, smcError := byb.calcAndSetSenderBalance(value, addrRole)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}
	// Pack receipt first
	byb.bybReceipt_onTransfer(byb.Sender.Addr, to, transByb)

	// Set the balance of the payee
	smcError = byb.setPayeeBalance(to, transByb)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}
	return
}

func (byb *TokenByb) TransferByChromo(chromo smc.Chromo, to smc.Address, value Number) (smcError smc.Error) {
	//The amount of the transfer is less than 0
	if value.Cmp_(0) <= 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid value: it cannot be a negative"
		return
	}

	//The receiving address cannot be itself
	// And cannot transfer to contract account
	if byb.Sender.Addr == to {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsUnsupportTransToSelf
		return
	}

	// Checking "to" address, cannot transfer to committee
	toRole, smcError := byb.checkAddress(to)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return smcError
	}
	if toRole == addressRole_Committee {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid address, cannot transfer byb to this address"
		return
	}
	// Cannot transfer to stock holder if it's not owned the chromo
	if toRole == addressRole_StockHolder {
		if !byb.isStockHolderOwnTheChromo(to, chromo) {
			smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			smcError.ErrorDesc = "Cannot transfer chromo byb to stockholder who does not own the chromo"
			return
		}
	}

	transByb, smcError := byb.checkAndSetSenderChromoBybBalance(chromo, *value.Value())
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return smcError
	}
	// Pack receipt first
	byb.bybReceipt_onTransfer(byb.Sender.Addr, to, transByb)

	smcError = byb.setPayeeBalance(to, transByb)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}
	return
}

/**
 * @dev set new owner account address
 */
func (byb *TokenByb) SetOwner(newOwnerAddr smc.Address) (smcError smc.Error) {
	// only contract owner just can set new owner
	if byb.Sender.Addr != byb.Owner.Addr {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}

	if newOwnerAddr == byb.Owner.Addr {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsUnsupportTransToSelf
		return
	}

	addrRole, smcError := byb.checkAddress(newOwnerAddr)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return smcError
	}
	if addrRole == addressRole_Committee ||
		addrRole == addressRole_StockHolder ||
		addrRole == addressRole_Hole {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid address"
		return
	}

	smcError = byb.setOwner(newOwnerAddr)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}
	byb.bybReceipt_onSetOwner(newOwnerAddr)
	return
}

func (byb *TokenByb) AddSupply(value Number) (smcError smc.Error) {

	// Value cannot be negative or ZERO
	if value.Cmp_(0) <= 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid value: it cannot be a negative"
		return
	}

	// Only token's owner can perform
	if byb.Sender.Addr != byb.Owner.Addr {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}

	newSupply, smcError := byb.addSupply(*value.Value())
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return smcError
	}
	byb.bybReceipt_onAddSupply(value, NB(&newSupply))
	return
}

func (byb *TokenByb) Burn(value Number) (smcError smc.Error) {

	// Value cannot be negative or ZERO
	if value.Cmp_(0) <= 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid value: it cannot be a negative"
		return
	}

	// Only token's owner can perform
	if byb.Sender.Addr != byb.Owner.Addr {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}

	newSupply, smcError := byb.burn(*value.Value())
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return smcError
	}
	byb.bybReceipt_onBurn(value, NB(&newSupply))
	return
}

// SetGasPrice is used to set gas price for specified token
func (byb *TokenByb) SetGasPrice(value uint64) (smcError smc.Error) {

	// Only token's owner can perform
	if byb.Sender.Addr != byb.Owner.Addr {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		return
	}

	// Token's gas price could not be smaller than GasBasePrice
	// and up to Max_Gas_Price (1E9 cong)
	if value < byb.State.GetBaseGasPrice() || value > smc.Max_Gas_Price {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidGasPrice
		return
	}

	smcError = byb.setGasPrice(value)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}
	byb.bybReceipt_onSetGasPrice(value)
	return
}

func (byb *TokenByb) ChangeChromoOwnership(chromo smc.Chromo, toStockHolder smc.Address) (smcError smc.Error) {

	if byb.Sender.Addr == toStockHolder {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsUnsupportTransToSelf
		smcError.ErrorDesc = "Don't transfer chromo to yourself"
		return
	}

	// Checking if sender is a stock holder
	addrRole, smcError := byb.checkAddress(byb.Sender.Addr)
	if addrRole != addressRole_StockHolder {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		smcError.ErrorDesc = "You are not a stock holder"
		return
	}

	addrRole, smcError = byb.checkAddress(toStockHolder)
	if addrRole != addressRole_StockHolder {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsNoAuthorization
		smcError.ErrorDesc = "Cannot transfer chromo to one who is not a stock holder"
		return
	}

	if !byb.isStockHolderOwnTheChromo(byb.Sender.Addr, chromo) {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsMinUserCode
		smcError.ErrorDesc = "Do not own the specified byb chromo"
		return
	}

	return byb.changeChromoOwnership(byb.Sender.Addr, toStockHolder, chromo)
}

/*
 * Private functions
 */
//Total calculation
func calcTotalBalance(chromoinfo []bybBalance) (totalBalace big.Int) {

	for _, x := range chromoinfo {
		totalBalace = Add(totalBalace, x.Value)
	}
	return

}

func (byb *TokenByb) updateBlackHole(blackHole smc.Address) (smcError smc.Error) {
	// get blackHole
	Holes, err := byb.getBlackHole()
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	Holes = append(Holes, blackHole)

	err = byb.setBlackHole(Holes)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	smcError.ErrorCode = bcerrors.ErrCodeOK
	return
}

// construct byb token object and save sender's balance
func (byb *TokenByb) initBybToken(totalSupply big.Int, addSupplyEnabled, burnEnabled bool) (smcError smc.Error) {
	smcError.ErrorCode = bcerrors.ErrCodeOK
	bybContract, err := byb.State.StateDB.GetContract(byb.State.ContractAddress)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	// construct token object and set it
	bybToken := types.IssueToken{
		Address:          bybContract.Address,
		Owner:            byb.Owner.Addr,
		Version:          bybContract.Version,
		Name:             BybName,
		Symbol:           bybSymbol,
		TotalSupply:      totalSupply,
		AddSupplyEnabled: addSupplyEnabled,
		BurnEnabled:      burnEnabled,
		GasPrice:         byb.State.GetBaseGasPrice(),
	}

	err = byb.State.SetToken(&bybToken)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	byb.bybReceipt_onInitToken(&bybToken, byb.ContractAcct.Addr)
	// construct sender's balance with totalSupply and set it
	balance := []bybBalance{
		{
			Chromo: getWhiteChromos(),
			Value:  totalSupply,
		},
	}

	err = byb.setBybBalance(byb.ContractAcct.Addr, balance)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
	}
	byb.bybReceipt_onTransfer("", byb.ContractAcct.Addr, balance)
	return
}

func (byb *TokenByb) getTokenAddrByName(name string) (address smc.Address, smcError smc.Error) {
	smcError.ErrorCode = bcerrors.ErrCodeOK
	address, err := byb.State.GetTokenAddrByName(BybName)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = smcError.Error()
		return
	}

	return
}

func (byb *TokenByb) getTokenAddrBySymbol(symbol string) (address smc.Address, smcError smc.Error) {
	smcError.ErrorCode = bcerrors.ErrCodeOK
	address, err := byb.State.GetTokenAddrBySymbol(bybSymbol)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = smcError.Error()
		return
	}

	return
}

// calculate the chromo for byb
func (byb *TokenByb) calcNewChromo() smc.Chromo {
	curChromo, err := byb.getCurChromo()
	if err != nil {
		return ""
	}
	var chromo uint64
	if curChromo == "" {
		chromo = 0
	} else {
		// Starting from 1 in statedb
		chromo, _ = strconv.ParseUint(curChromo, 10, 64)
		if chromo == 0 {
			return ""
		}
	}
	newChromo := strconv.FormatUint(chromo+1, 10)

	err = byb.setCurChromo(newChromo)

	return newChromo
}

func getWhiteChromos() smc.Chromo {
	return white_Chromo
}

// check byb token's paramters
func (byb *TokenByb) checkBybToken() (smcError smc.Error) {
	smcError.ErrorCode = bcerrors.ErrCodeOK

	// get token address by name and symbol
	addr1, bcerr1 := byb.getTokenAddrByName(BybName)
	addr2, bcerr2 := byb.getTokenAddrBySymbol(bybSymbol)
	if bcerr1.ErrorCode != bcerrors.ErrCodeOK {
		return bcerr1
	}
	if bcerr2.ErrorCode != bcerrors.ErrCodeOK {
		return bcerr2
	}

	if addr1 == "" && addr2 == "" {
		smcError.ErrorCode = bcerrors.ErrCodeOK
		return
	}

	// the address must be equal both addr1 and addr2
	if addr1 != addr2 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsBybInitialized
		return
	}

	// get token by address
	iToken, err := byb.State.GetToken(addr1)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	if iToken.Owner != byb.Sender.Addr {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsBybInitialized
		return
	}

	if Compare(iToken.TotalSupply, Zero()) != 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsBybInitialized
		return
	}

	// get token contract with contract address
	tContract, err := byb.State.StateDB.GetContract(iToken.Address)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	// can't set byb contract lose height
	if tContract.Address == byb.State.ContractAddress {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsBybInitialized
		return
	}

	// get world app state for block height
	worldAppState, err := byb.State.StateDB.GetWorldAppState()
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	// set byb token contract's lose height with block height
	tContract.LoseHeight = uint64(worldAppState.BlockHeight + 1)

	// update byb token contract
	err = byb.State.SetTokenContract(tContract)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = smcError.Error()
		return
	}

	return
}

func (byb *TokenByb) makeStockHolderAndBybBalance(stockHolder smc.Address, value big.Int) (bybt []bybBalance, smcError smc.Error) {
	//get the role of address
	var addrRole uint32
	addrRole, smcError = byb.checkAddress(stockHolder)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	if addrRole == addressRole_Committee || addrRole == addressRole_Hole {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid address, please ensure it's not the owner of contract or a hole"
		return
	}

	stockHolderAddrs, err := byb.getBybStockHolders()
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	if addrRole == addressRole_User && len(stockHolderAddrs)+1 > maxStockHolder {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsBeyondMaximumStockHolders
		return
	}

	stockHolderBalances, err := byb.getBybBalance(stockHolder)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	// A stock holder could not own byb before it becomes a real stock holder.
	if addrRole == addressRole_User && len(stockHolderBalances) != 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid address due to it has already owned byb token"
		return
	}

	if len(stockHolderBalances) == 0 {
		//make new byb
		bybBalanceNew, smcErr := byb.makeByb(value)
		if smcErr.ErrorCode != bcerrors.ErrCodeOK {
			return
		}
		stockHolderBalances = append(stockHolderBalances, bybBalanceNew)
	} else {
		item := stockHolderBalances[0]
		itmeValue := item.Value
		newValue := Add(itmeValue, value)
		newItem := bybBalance{item.Chromo, newValue}
		stockHolderBalances[0] = newItem
	}
	// Add into stock holder list if it's new
	if addrRole == addressRole_User {
		stockHolderAddrs = append(stockHolderAddrs, stockHolder)
		err := byb.setBybStockHolders(stockHolderAddrs)
		if err != nil {
			smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
			smcError.ErrorDesc = err.Error()
			return
		}
	}
	return stockHolderBalances, smcError
}

func (byb *TokenByb) calcAndSetSenderBalance(value big.Int, senderRole uint32) (out []bybBalance, smcError smc.Error) {
	smcError.ErrorCode = bcerrors.ErrCodeOK
	if Compare(value, Zero()) == 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		return
	}
	senderByb, err := byb.getBybBalance(byb.Sender.Addr)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	// Sender's balance could not be less than the transfer value
	totalBalance := calcTotalBalance(senderByb)
	if Compare(totalBalance, value) < 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInsufficientBalance
		return
	}
	// Sender's balance equal to transfer value
	if Compare(totalBalance, value) == 0 {
		if senderRole == addressRole_Hole {
			out = []bybBalance{{getWhiteChromos(), value}}
		} else {
			out = senderByb
		}
		senderNewByb := make([]bybBalance, 0)
		err = byb.setBybBalance(byb.Sender.Addr, senderNewByb)
		if err != nil {
			smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
			smcError.ErrorDesc = err.Error()
		}
		return
	}
	// Sender's balance is more than transfer value
	// transfer byb in proportion
	senderNewByb, payeeByb, smcError := calcTransferBybChromo(senderByb, value)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return nil, smcError
	}
	payeeTotal := calcTotalBalance(payeeByb)
	surplus := Sub(value, payeeTotal)
	if Compare(surplus, Zero()) == 0 {
		err = byb.setBybBalance(byb.Sender.Addr, senderNewByb)
		if err != nil {
			smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
			smcError.ErrorDesc = err.Error()
		}
		if senderRole == addressRole_Hole {
			out = []bybBalance{{getWhiteChromos(), value}}
		} else {
			out = payeeByb
		}
		return
	}
	// for Second round to deduct remainder
	sr_senderNewByb, sr_payeeByb, smcError := calcTransferBybChromo(senderNewByb, surplus)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return nil, smcError
	}
	err = byb.setBybBalance(byb.Sender.Addr, sr_senderNewByb)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	sr_payeeTotal := calcTotalBalance(sr_payeeByb)
	sr_surplus := Sub(surplus, sr_payeeTotal)
	if Compare(sr_surplus, Zero()) != 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidBalance
		return nil, smcError
	}

	if senderRole == addressRole_Hole {
		//Transfer from blackHole, the chromo is changed to white
		out = []bybBalance{{getWhiteChromos(), value}}
	} else {
		for _, ch := range payeeByb {
			for _, sr := range sr_payeeByb {
				if ch.Chromo == sr.Chromo {
					ch.Value = Add(ch.Value, sr.Value)
					break
				}
			}
			out = append(out, ch)
		}
	}
	return
}

func calcTransferBybChromo(senderByb []bybBalance, value big.Int) (senderNewByb, payeeByb []bybBalance, smcError smc.Error) {
	smcError.ErrorCode = bcerrors.ErrCodeOK
	var tempChromo bybBalance
	var totalTransValue big.Int

	totalBalance := calcTotalBalance(senderByb)
	surplus := Sub(value, totalTransValue)

	for _, chromo := range senderByb {
		// if balance = 0, skip it
		if Compare(chromo.Value, Zero()) == 0 {
			continue
		}

		// The remaining amount to be transferred
		surplus = Sub(value, totalTransValue)
		if Compare(surplus, Zero()) == 0 {
			senderNewByb = append(senderNewByb, chromo)
			continue // continue to append all chromo of sender to newChromo slice
		}
		if Compare(surplus, Zero()) < 0 { // Wrong
			smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidBalance
			return
		}

		// Calc transfer value for each chromo depends on its value
		// transValue = transfer value * balance of this chromo / total balance of sender
		// so: transValue = value * chromo.Value / totalBalance
		temp := SafeMul(value, chromo.Value)
		transValue := SafeDiv(temp, totalBalance)
		if Compare(transValue, chromo.Value) > 0 {
			smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidBalance
			return
		}
		if Compare(transValue, surplus) > 0 {
			transValue = *transValue.Set(&surplus)
		} else if Compare(transValue, Zero()) == 0 {
			transValue = *big.NewInt(1)
		}
		totalTransValue = Add(totalTransValue, transValue)

		// append "out" balance
		tempChromo.Value = transValue
		tempChromo.Chromo = chromo.Chromo
		payeeByb = append(payeeByb, tempChromo)

		// Update sender's balance
		tempChromo.Value = Sub(chromo.Value, transValue)
		if Compare(tempChromo.Value, Zero()) > 0 {
			senderNewByb = append(senderNewByb, tempChromo)
		}
	}
	return
}

/**
* Update the balance of payee
 */
func (byb *TokenByb) setPayeeBalance(payeeAddr smc.Address, addedByb []bybBalance) (smcError smc.Error) {
	smcError.ErrorCode = bcerrors.ErrCodeOK
	oldByb, err := byb.getBybBalance(payeeAddr)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	var newBybChromo = make([]bybBalance, 0)
	var bFound bool

	// Set old chromo first
	for _, chromo := range oldByb {
		bFound = false
		for i, newchromo := range addedByb {
			if chromo.Chromo == newchromo.Chromo {
				chromo.Value = Add(chromo.Value, newchromo.Value)
				newBybChromo = append(newBybChromo, chromo)
				//Reset to Zero
				addedByb[i].Value = *big.NewInt(0)
				bFound = true
				break
			}
		}
		// Old chromo byb, and its value did not change, append to slice
		if !bFound {
			newBybChromo = append(newBybChromo, chromo)
		}
	}

	// Add new chromo
	for _, newchromo := range addedByb {
		if Compare(newchromo.Value, Zero()) == 0 {
			continue
		}
		newBybChromo = append(newBybChromo, newchromo)
	}
	// Set payee's balance
	err = byb.setBybBalance(payeeAddr, newBybChromo)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
	}
	return
}

func (byb *TokenByb) checkAddress(addr smc.Address) (addrRole uint32, smcError smc.Error) {

	smcError.ErrorCode = bcerrors.ErrCodeOK
	if addr == byb.Owner.Addr ||
		addr == byb.ContractAcct.Addr ||
		addr == *byb.ContractAddr {
		addrRole = addressRole_Committee
		return
	}

	blackHoleAddrs, err := byb.getBlackHole()
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	for _, blackHole := range blackHoleAddrs {
		if addr == blackHole {
			addrRole = addressRole_Hole
			return
		}
	}

	//get the list of all stockHolders
	stockHolderAddrs, err := byb.getBybStockHolders()
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	for _, v := range stockHolderAddrs {
		if addr == v {
			// stock holder
			addrRole = addressRole_StockHolder
			return
		}
	}
	addrRole = addressRole_User

	return
}

func (byb *TokenByb) delStockHolder(stockHolder smc.Address) (smcError smc.Error) {
	smcError.ErrorCode = bcerrors.ErrCodeOK

	holders, err := byb.getBybStockHolders()
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	if holders == nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = "Failed to get stock holders list"
		return
	}

	var bFound bool = false
	for i, holder := range holders {
		if holder == stockHolder {
			holders = append(holders[:i], holders[i+1:]...)
			bFound = true
			break
		}
	}
	if !bFound {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsBybHolderNotFound
		return
	}

	err = byb.setBybStockHolders(holders)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	return
}

func (byb *TokenByb) checkAndSetSenderChromoBybBalance(chromo smc.Chromo, value big.Int) (bybt []bybBalance, smcError smc.Error) {

	// Checking "Sender" address, committee cannot transfer byb thru this function
	senderRole, smcError := byb.checkAddress(byb.Sender.Addr)
	if smcError.ErrorCode != bcerrors.ErrCodeOK {
		return nil, smcError
	}
	if senderRole == addressRole_Committee {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid address, cannot transfer byb from this address"
		return
	}

	// Check sender's balance
	senderByb, err := byb.getBybBalance(byb.Sender.Addr)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	if senderByb == nil || len(senderByb) == 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInsufficientBalance
		return
	}
	var bFound bool = false
	for _, token := range senderByb {
		if token.Chromo == chromo {
			bFound = true
			if Compare(token.Value, value) < 0 {
				smcError.ErrorCode = bcerrors.ErrCodeInterContractsInsufficientBalance
				return
			}
			break
		}
	}
	if !bFound {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInsufficientBalance
		return
	}

	//Set sender byb
	for i, token := range senderByb {
		if token.Chromo == chromo {
			senderByb[i].Value = Sub(token.Value, value)
			//For normal user, delete the chromo from list when the value is ZERO
			if senderRole != addressRole_StockHolder &&
				Compare(senderByb[i].Value, Zero()) == 0 {
				senderByb = append(senderByb[:i], senderByb[i+1:]...)
			}
			break
		}
	}
	err = byb.setBybBalance(byb.Sender.Addr, senderByb)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	var bybChromo = make([]bybBalance, 1)
	if senderRole == addressRole_Hole {
		bybChromo[0].Chromo = getWhiteChromos()
	} else {
		bybChromo[0].Chromo = chromo
	}
	bybChromo[0].Value = value

	smcError.ErrorCode = bcerrors.ErrCodeOK
	return bybChromo, smcError
}

func (byb *TokenByb) isStockHolderOwnTheChromo(stockHolder smc.Address, chromo smc.Chromo) bool {
	ownedByb, err := byb.getBybBalance(stockHolder)
	if err != nil {
		return false
	}

	for _, token := range ownedByb {
		if token.Chromo == chromo {
			return true
		}
	}
	return false
}

func (byb *TokenByb) setOwner(newOwnerAddr smc.Address) (smcError smc.Error) {
	// Set bybContract
	contract, err := byb.State.StateDB.GetContract(byb.State.ContractAddress)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	if contract == nil {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidAddr
		return
	}

	contract.Owner = newOwnerAddr
	err = byb.State.SetTokenContract(contract)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	// Delete contract from old owner's list
	err = byb.State.DeleteContractAddr(byb.Owner.Addr, contract.Address)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
	}

	// Set BYB Token
	token, lerror := byb.State.StateDB.GetToken(byb.State.ContractAddress)
	if lerror != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = lerror.Error()
		return
	}
	if token == nil {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidAddr
		return
	}

	token.Owner = newOwnerAddr
	lerror = byb.State.SetToken(token)
	if lerror != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = lerror.Error()
		return
	}

	smcError.ErrorCode = bcerrors.ErrCodeOK
	return
}

func (byb *TokenByb) addSupply(value big.Int) (totalSupply big.Int, smcError smc.Error) {

	// Get token's total supply and calculate the new total suppl.
	token, err := byb.State.GetToken(*byb.ContractAddr)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	if token == nil {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidAddr
		return
	}
	if !token.AddSupplyEnabled {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsUnsupportAddSupply
		return
	}

	newSupply := Add(token.TotalSupply, value)
	if Compare(newSupply, token.TotalSupply) < 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidSupply
		return
	}
	token.TotalSupply = newSupply
	err = byb.State.SetToken(token)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	// Get owner's balance and calculate new balance
	ownerByb, err := byb.getBybBalance(byb.ContractAcct.Addr)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	// This account only has one chromo byb
	ownerByb[0].Value = Add(ownerByb[0].Value, value)
	err = byb.setBybBalance(byb.ContractAcct.Addr, ownerByb)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	// transfer receipt
	newBybValue := []bybBalance{{getWhiteChromos(), value}}
	byb.bybReceipt_onTransfer("", byb.ContractAcct.Addr, newBybValue)

	smcError.ErrorCode = bcerrors.ErrCodeOK
	return newSupply, smcError
}

func (byb *TokenByb) burn(value big.Int) (totalSupply big.Int, smcError smc.Error) {

	// Get token's total supply and calculate the new total supply.
	token, err := byb.State.GetToken(*byb.ContractAddr)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	if token == nil {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidAddr
		return
	}
	if !token.BurnEnabled {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsUnsupportBurn
		return
	}
	// This account only has one chromo byb
	if Compare(token.TotalSupply, value) < 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInsufficientBalance
		return
	}

	// Get owner's balance and calculate new balance
	ownerByb, err := byb.getBybBalance(byb.ContractAcct.Addr)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	// This account only has one chromo byb
	if Compare(ownerByb[0].Value, value) < 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInsufficientBalance
		return
	}
	newBalance := Sub(ownerByb[0].Value, value)
	if Compare(newBalance, ownerByb[0].Value) >= 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidBalance
		return
	}
	ownerByb[0].Value = newBalance
	err = byb.setBybBalance(byb.ContractAcct.Addr, ownerByb)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	newSupply := Sub(token.TotalSupply, value)
	if Compare(newSupply, token.TotalSupply) >= 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidSupply
		return
	}
	token.TotalSupply = newSupply
	err = byb.State.SetToken(token)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	// transfer receipt
	newBybValue := []bybBalance{{getWhiteChromos(), value}}
	byb.bybReceipt_onTransfer(byb.ContractAcct.Addr, "", newBybValue)

	smcError.ErrorCode = bcerrors.ErrCodeOK
	return newSupply, smcError
}

func (byb *TokenByb) setGasPrice(value uint64) (smcError smc.Error) {

	//Set token's gas price
	// Get token's total supply and calculate the new total supply.
	token, err := byb.State.GetToken(*byb.ContractAddr)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	if token == nil {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidAddr
		return
	}

	token.GasPrice = value
	err = byb.State.SetToken(token)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	smcError.ErrorCode = bcerrors.ErrCodeOK
	return
}

func (byb *TokenByb) changeChromoOwnership(from, to smc.Address, chromo smc.Chromo) (smcError smc.Error) {

	ownedByb, err := byb.getBybBalance(from)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	var transByb bybBalance
	for i, bybt := range ownedByb {
		if bybt.Chromo == chromo {
			ownedByb = append(ownedByb[:i], ownedByb[i+1:]...)
			transByb = bybt
			break
		}
	}
	// Set sender's byb token
	err = byb.setBybBalance(byb.Sender.Addr, ownedByb)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	//Get stockholder's byb and set
	holderByb, err := byb.getBybBalance(to)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	// The stock holder would never owned this chromo unless by this function call
	holderByb = append(holderByb, transByb)
	err = byb.setBybBalance(to, holderByb)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	//Receipts
	trbyb := make([]bybBalance, 0)
	trbyb = append(trbyb, transByb)

	byb.bybReceipt_onChangeChromoOwnership(from, to, trbyb)

	smcError.ErrorCode = bcerrors.ErrCodeOK
	return
}

//set the stockHolder's and sender's byb balances to stateDB
func (byb *TokenByb) setOwnerAndStockHolderBalance(stockHolder string, bybBalances []bybBalance, value, sendBalanceBYB big.Int) (smcError smc.Error) {
	smcError.ErrorCode = bcerrors.ErrCodeOK
	//set the list of stockHolder's bybs  to stateDB
	err := byb.setBybBalance(stockHolder, bybBalances)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	sendBalanceBYBNew := Sub(sendBalanceBYB, value)
	ownerBalance, err := byb.getBybBalance(byb.ContractAcct.Addr)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	ownerBalance[0].Value = sendBalanceBYBNew
	err = byb.setBybBalance(byb.ContractAcct.Addr, ownerBalance)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	// the chromo for stock holder, it's always the first one whatever it's a new stock holder or not.
	transByb := make([]bybBalance, 0)
	transByb = append(transByb, bybBalances[0])
	if len(bybBalances) > 1 {
		// len() > 1 means it's not a new stockholder, and might be owning byb before.
		// reset the value for receipt.
		transByb[0].Value = value
	}
	byb.bybReceipt_onTransfer(byb.ContractAcct.Addr, stockHolder, bybBalances)
	return
}

//make bybBalance object
func (byb *TokenByb) makeByb(value big.Int) (balance bybBalance, smcError smc.Error) {
	smcError.ErrorCode = bcerrors.ErrCodeOK
	//Calculation chromos
	chromo := byb.calcNewChromo()
	if chromo == "" {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = "Error happened while calculating byb chromo"
		return
	}
	balance = bybBalance{
		chromo,
		value,
	}
	return
}
