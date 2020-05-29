package smcapi

import (
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/statedb"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/types"
	"github.com/bcbchain/bclib/algorithm"
	. "github.com/bcbchain/bclib/bignumber_v1.0"
	"encoding/hex"
	"encoding/json"
	"github.com/docker/docker/api/types/versions"
	"github.com/pkg/errors"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	"golang.org/x/crypto/sha3"
	"math/big"
)

var logger log.Logger

type Block struct {
	BlockHash       []byte
	ChainID         string
	Height          int64
	Time            int64
	NumTxs          int32
	DataHash        []byte
	LastBlockHash   []byte
	LastCommitHash  []byte
	LastAppHash     []byte
	LastFee         uint64
	ProposerAddress string
	RewardAddress   string
	RandomeOfBlock  []byte
}

func (block *Block) Now_() int64 {
	return block.Time
}
func (block *Block) Now() Number {
	return N(block.Time)
}

type EventHandler struct {
	smcApi   *SmcApi
	receipts []smc.Receipt
}

type SmcApi struct {
	Sender       *stubapi.Account // Address of contract caller,
	Owner        *stubapi.Account // Address of contract owner
	ContractAcct *stubapi.Account // Address of contract account
	ContractAddr *smc.Address
	State        *statedb.TxState // State DB
	Block        *Block
	EventHandler *EventHandler
	Note         string
}

func SetLogger(nlog log.Logger) {
	logger = nlog
}

func GetLogger() log.Logger {
	return logger
}

func InitEventHandler(smcapi *SmcApi) {
	var evtHandler = EventHandler{}

	evtHandler.smcApi = smcapi
	evtHandler.receipts = make([]smc.Receipt, 0)

	smcapi.EventHandler = &evtHandler
	return
}

//Get receipt
func (evh *EventHandler) GetReceipts() []smc.Receipt {
	return evh.receipts
}

//Set receipt
func (evh *EventHandler) SetReceipts(receipts []smc.Receipt) error {
	evh.receipts = receipts
	return nil
}

//Emit a receipt
func (evh *EventHandler) EmitReceipt(name string, receipt smc.ReceiptBytes) {
	rd := smc.Receipt{
		Name:            name,
		ContractAddress: *evh.smcApi.ContractAddr,
		ReceiptBytes:    receipt,
		ReceiptHash:     smc.CalcReceiptHash(name, *evh.smcApi.ContractAddr, receipt), //hash
	}
	evh.receipts = append(evh.receipts, rd)
}

//Emit a transfer event，建议使用TransferToken函数指明代币名称，进行转账
func (rec *EventHandler) Transfer(from, to smc.Address, value Number) (bcerr smc.Error) {
	return rec.TransferToken(
		"",
		from,
		to,
		value)
}

//Emit a transfer event, tokenName=""，表示使用本币转账
func (rec *EventHandler) TransferToken(tokenName string, from, to smc.Address, value Number) (bcerr smc.Error) {
	if len(tokenName) == 0 {
		token, _ := rec.smcApi.State.GetGenesisToken()
		tokenName = token.Name
	}
	tokenAddr, _ := rec.smcApi.State.GetTokenAddrByName(tokenName)

	return rec.transferToken(tokenAddr, from, to, value)
}

//Emit a transfer event, tokenName=""，表示使用本币转账
func (rec *EventHandler) TransferByAddr(tokenAddr, from, to smc.Address, value Number) (bcerr smc.Error) {

	return rec.transferToken(tokenAddr, from, to, value)
}

//Emit a transfer event, tokenName=""，表示使用本币转账
func (rec *EventHandler) transferToken(tokenAddr, from, to smc.Address, value Number) (bcerr smc.Error) {
	tokenValue := value.Value()
	if Compare(*tokenValue, Zero()) <= 0 {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		bcerr.ErrorDesc = "Invalid value: it cannot be a negative"
		return
	}

	if from == to {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsUnsupportTransToSelf
		return
	}

	if len(tokenAddr) == 0 {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		bcerr.ErrorDesc = "Invalid value: you must specify a token to transfer"
		return
	}
	// Using a copy here in case the code modify original TxState
	newTxState := *rec.smcApi.State
	newTxState.ContractAddress = tokenAddr
	sender := stubapi.Account{from, &newTxState}
	// Get and check Sender's balance
	senderBal, bcerr := sender.BalanceOf(tokenAddr)
	if bcerr.ErrorCode != bcerrors.ErrCodeOK {
		return bcerr
	}

	if Compare(senderBal, *tokenValue) < 0 {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInsufficientBalance
		return
	}

	bcerr = sender.SetBalance(tokenAddr, Sub(senderBal, *tokenValue))
	if bcerr.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	// Get and Set payee's balance
	payee := stubapi.Account{to, sender.TxState}
	payeeBal, bcerr := payee.BalanceOf(tokenAddr)
	if bcerr.ErrorCode != bcerrors.ErrCodeOK {
		return bcerr
	}
	bcerr = payee.SetBalance(tokenAddr, Add(payeeBal, *tokenValue))
	if bcerr.ErrorCode != bcerrors.ErrCodeOK {
		//logger.Error("Failed to set new balance of payee")
		return
	}

	rec.PackReceiptOfTransfer(tokenAddr, from, to, value)

	return
}

const (
	ReceiptNameTransfer    = "transfer"
	ReceiptNameFee         = "fee"
	ReceiptNameSetOwner    = "setOwner"
	ReceiptNameAddSupply   = "addSupply"
	ReceiptNameBurn        = "burn"
	ReceiptNameSetGasPrice = "setGasPrice"
	ReceiptNameNewToken    = "newToken"
)

// -----------------------------------pack receipt-------------------------------------
func (rec *EventHandler) PackReceiptOfTransfer(token, from, to smc.Address, value Number) {
	receiptByte, err := json.Marshal(&smc.ReceiptOfTransfer{
		Token: token,
		From:  from,
		To:    to,
		Value: *value.Value()})
	if err != nil {
		return
	}

	rec.EmitReceipt(ReceiptNameTransfer, receiptByte)
}

func (rec *EventHandler) PackReceiptOfFee(token, from smc.Address, value uint64) {
	receiptByte, err := json.Marshal(&smc.ReceiptOfFee{
		Token: token,
		From:  from,
		Value: value})
	if err != nil {
		return
	}

	rec.EmitReceipt(ReceiptNameFee, receiptByte)
}

func (rec *EventHandler) PackReceiptOfSetOwner(address, newOwner smc.Address) {
	receiptByte, err := json.Marshal(&smc.ReceiptOfSetOwner{
		ContractAddr: address,
		NewOwner:     newOwner})
	if err != nil {
		return
	}

	rec.EmitReceipt(ReceiptNameSetOwner, receiptByte)
}

func (rec *EventHandler) PackReceiptOfAddSupply(token smc.Address, value, totalSupply Number) {
	receiptByte, err := json.Marshal(&smc.ReceiptOfAddSupply{
		Token:       token,
		Value:       *value.Value(),
		TotalSupply: *totalSupply.Value()})
	if err != nil {
		return
	}

	rec.EmitReceipt(ReceiptNameAddSupply, receiptByte)
}

func (rec *EventHandler) PackReceiptOfBurn(token smc.Address, value, totalSupply Number) {
	receiptByte, err := json.Marshal(&smc.ReceiptOfBurn{
		Token:       token,
		Value:       *value.Value(),
		TotalSupply: *totalSupply.Value()})
	if err != nil {
		return
	}

	rec.EmitReceipt(ReceiptNameBurn, receiptByte)
}

func (rec *EventHandler) PackReceiptOfSetGasPrice(token smc.Address, gasPrice uint64) {
	receiptByte, err := json.Marshal(&smc.ReceiptOfSetGasPrice{
		Token:    token,
		GasPrice: gasPrice})
	if err != nil {
		return
	}

	rec.EmitReceipt(ReceiptNameSetGasPrice, receiptByte)
}

func (rec *EventHandler) PackReceiptOfNewToken(token *types.IssueToken, acctAddr smc.Address) {

	receiptByte, err := json.Marshal(&smc.ReceiptOfNewToken{
		TokenAddress:     token.Address,
		ContractAddress:  token.Address,
		AccountAddress:   acctAddr,
		Owner:            token.Owner,
		Version:          token.Version,
		Name:             token.Name,
		Symbol:           token.Symbol,
		TotalSupply:      token.TotalSupply,
		AddSupplyEnabled: token.AddSupplyEnabled,
		BurnEnabled:      token.BurnEnabled,
		GasPrice:         token.GasPrice})
	if err != nil {
		return
	}

	rec.EmitReceipt(ReceiptNameNewToken, receiptByte)
}

//Sha3_256
func (api *SmcApi) SHA3256(datas ...[]byte) []byte {

	hasherSHA3256 := sha3.New256()
	for _, data := range datas {
		hasherSHA3256.Write(data)
	}
	return hasherSHA3256.Sum(nil)
}

func (api *SmcApi) GetCurrentBlockHeight() (int64, error) {
	app, err := api.State.StateDB.GetWorldAppState()
	if err != nil {
		return 0, err
	}
	if app == nil {
		return 0, errors.New("Failed to get AppState")
	}

	return app.BlockHeight + 1, nil
}

//GetAccount creates Account structure if addr is an external account address,
// or Get its external account if addr is a contract address
func (api *SmcApi) GetAccount(addr smc.Address) *stubapi.Account {
	contract, _ := api.State.StateDB.GetContract(addr)
	if contract != nil {
		return api.GetContractAcct(contract.Name)
	}
	return &stubapi.Account{addr, api.State}
}

func (api *SmcApi) GetContractAcct(name string) *stubapi.Account {

	addr := algorithm.CalcContractAddress(
		api.State.GetChainID(),
		"",
		name,
		"")

	return &stubapi.Account{addr, nil}
}

func (api *SmcApi) GetContractAddr(name string) smc.Address {

	appState, _ := api.State.StateDB.GetWorldAppState()
	if appState == nil {
		return ""
	}

	contractList, _ := api.State.GetContractsListByName(name)
	for _, addr := range contractList {
		contract, _ := api.State.StateDB.GetContract(addr)

		if contract.EffectHeight <= uint64(appState.BlockHeight+1) &&
			(contract.LoseHeight == 0 || contract.LoseHeight > uint64(appState.BlockHeight+1)) {
			return contract.Address
		}
	}
	return ""
}

//CalcAddress calculates address of smart contract with contract name and other flags
func (api *SmcApi) CalcContractAddress(name, version string) smc.Address {
	return smc.Address(
		algorithm.CalcContractAddress(
			api.State.GetChainID(),
			crypto.Address(api.Owner.Addr),
			name,
			version))
}

func (api *SmcApi) GetContract() (*types.Contract, error) {
	return api.State.StateDB.GetContract(*api.ContractAddr)
}

func (api *SmcApi) CreateUnitedToken(addr smc.Address,
	name string,
	symbol string,
	totalsupply big.Int,
	addSupplyEnabled bool,
	burnEnabled bool,
	gasprice uint64) smc.Error {

	ctx := stubapi.InvokeContext{Sender: api.Sender, Owner: api.Owner, TxState: api.State}

	bcerr := ctx.NewUnitedToken(addr, name, symbol, totalsupply, addSupplyEnabled, burnEnabled, gasprice)
	if bcerr.ErrorCode == bcerrors.ErrCodeOK {
		// Receipts
		token, _ := ctx.GetToken(addr)

		api.EventHandler.PackReceiptOfNewToken(token, api.GetContractAcct(name).Addr)

		api.EventHandler.PackReceiptOfTransfer(token.Address, "", token.Owner, NB(&token.TotalSupply))
	}

	return bcerr
}

func (api *SmcApi) CheckParameterGasAndPrototype(protoTypes []string, gasList []uint64) (smcError smc.Error) {
	// Check gas, gas cannot be smaller than 1 (for now)
	// Warning: Commit out due to contract united-token requires to set gas of transfer to 0
	//for _, gas := range gasList {
	//	if gas < 1 {
	//		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidGasPrice
	//		return
	//	}
	//}

	// each prototype must has its own gas
	if len(protoTypes) != len(gasList) {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidGasPrice
		return
	}
	smcError.ErrorCode = bcerrors.ErrCodeOK
	return
}

func (api *SmcApi) ForbidSpecificContract(addr smc.Address, loseHeight uint64) (smcError smc.Error) {
	smcError.ErrorCode = bcerrors.ErrCodeOK

	contract, err := api.State.StateDB.GetContract(addr)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}
	if contract == nil {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "The specified contract address does not exist"
		return
	}

	if contract.LoseHeight != 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "It's already be set, don't repeat"
		return
	}

	if loseHeight <= contract.EffectHeight {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "The specified block height is invalid"
		return
	}

	contract.LoseHeight = loseHeight
	err = api.State.SetTokenContract(contract)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	return
}

func (api *SmcApi) CheckAndForbidOldVersionContract(name, version string, effectHeight uint64) (bForbid bool, smcError smc.Error) {
	smcError.ErrorCode = bcerrors.ErrCodeOK
	oldcontracts, err := api.State.GetContractsListByName(name)
	if err != nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = err.Error()
		return
	}

	bForbid = false
	for _, contractAddr := range oldcontracts {
		bForbid = true
		contract, err := api.State.StateDB.GetContract(contractAddr)
		if err != nil {
			smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
			smcError.ErrorDesc = err.Error()
			return
		}
		if versions.GreaterThan(contract.Version, version) {
			smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			smcError.ErrorDesc = "Contract's version is incorrect"
			return
		}
		if effectHeight <= contract.EffectHeight {
			smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
			smcError.ErrorDesc = "Contract's effectHeight is incorrect"
			return
		}
		if contract.LoseHeight == 0 {
			bForbid = false
			contract.LoseHeight = effectHeight
			err = api.State.SetTokenContract(contract)
			if err != nil {
				smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
				smcError.ErrorDesc = err.Error()
				return
			}
		}
	}

	return
}

func (api *SmcApi) SetNewContract(
	contractAddr smc.Address,
	name, version string,
	protoTypes []string,
	gasList []uint64,
	codeHash smc.Hash,
	effectHeight uint64) error {

	methods := make([]types.Method, 0)
	for i, protoType := range protoTypes {
		method := types.Method{
			MethodId:  hex.EncodeToString(algorithm.CalcMethodId(protoType)),
			Gas:       int64(gasList[i]),
			Prototype: protoType,
		}
		methods = append(methods, method)
	}
	contract := types.Contract{
		Address:      contractAddr,
		Owner:        api.Owner.Addr,
		Name:         name,
		Version:      version,
		CodeHash:     hex.EncodeToString(codeHash),
		Methods:      methods,
		EffectHeight: uint64(effectHeight),
		LoseHeight:   0,
	}

	return api.State.SetTokenContract(&contract)
}

func (api *SmcApi) CheckTokenNameAndSybmol(name, symbol string) smc.Error {
	ic := stubapi.InvokeContext{Sender: api.Sender, Owner: api.Owner, TxState: api.State}
	return ic.CheckNameAndSybmol(name, symbol)
}

func (api *SmcApi) GetTokenTotalSupply() (big.Int, smc.Error) {
	ic := stubapi.InvokeContext{Sender: api.Sender, Owner: api.Owner, TxState: api.State}
	return ic.TokenSupply()
}

func (api *SmcApi) SetTokenGasPrice(value uint64) smc.Error {
	ic := stubapi.InvokeContext{Sender: api.Sender, Owner: api.Owner, TxState: api.State}
	return ic.SetGasPrice(value)
}

func (api *SmcApi) SetTokenSupply(value big.Int) smc.Error {
	ic := stubapi.InvokeContext{Sender: api.Sender, Owner: api.Owner, TxState: api.State}
	return ic.SetTokenSupply(value, true, false)
}

func (api *SmcApi) TokenBurn(value big.Int) smc.Error {
	ic := stubapi.InvokeContext{Sender: api.Sender, Owner: api.Owner, TxState: api.State}
	return ic.SetTokenSupply(value, false, true)
}

func (api *SmcApi) SetTokenNewOwner(owner smc.Address) smc.Error {
	ic := stubapi.InvokeContext{Sender: api.Sender, Owner: api.Owner, TxState: api.State}
	return ic.SetTokenNewOwner(owner)
}

func (api *SmcApi) GetOwnerOfGenesisToken() (addr smc.Address, smcError smc.Error) {
	// Get genesis token
	gt, _ := api.State.GetGenesisToken()
	if gt == nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = "Failed to get genesis token"
		return
	}
	// Get current token information in case it was set new owner
	token, _ := api.State.GetToken(gt.Address)
	if token == nil {
		smcError.ErrorCode = bcerrors.ErrCodeLowLevelError
		smcError.ErrorDesc = "Failed to get genesis token"
		return
	}
	addr = token.Owner

	smcError.ErrorCode = bcerrors.ErrCodeOK
	return
}

func (api *SmcApi) GetBalance(tokenName, accAddress string) (value big.Int, bcerr smc.Error) {
	if len(tokenName) == 0 {
		token, _ := api.State.GetGenesisToken()
		tokenName = token.Name
	}

	tokenAddr, _ := api.State.GetTokenAddrByName(tokenName)
	if len(tokenAddr) == 0 {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		bcerr.ErrorDesc = "Invalid value: you must specify a token to query"
		return
	}
	// Using a copy here in case the code modify original TxState
	newTxState := *api.State
	newTxState.ContractAddress = tokenAddr
	sender := stubapi.Account{accAddress, &newTxState}
	// Get and check Sender's balance
	value, bcerr = sender.BalanceOf(tokenAddr)

	return
}

// SetTokenNewOwner sets a new owner for the specified token
func (api *SmcApi) TransferToken(from, to, token smc.Address, value big.Int) (bcerr smc.Error) {
	if Compare(value, Zero()) < 0 {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		return
	}

	newTxState := statedb.TxState{
		api.State.StateDB,
		token,
		from,
		api.State.Tx,
	}

	// sender's balance
	accSender := stubapi.Account{from, &newTxState}
	senderBalance, _ := accSender.BalanceOf(token)
	if Compare(senderBalance, value) < 0 {
		bcerr.ErrorCode = bcerrors.ErrCodeInterContractsInsufficientBalance
		return
	}

	bcerr = accSender.SetBalance(token, Sub(senderBalance, value))
	if bcerr.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	//to's balance
	accTo := stubapi.Account{to, &newTxState}
	toBalance, _ := accTo.BalanceOf(token)

	return accTo.SetBalance(token, Add(toBalance, value))
}
