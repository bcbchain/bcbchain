package helper

import (
	"blockchain/algorithm"
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
	"blockchain/smcsdk/sdkimpl/object"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// standardTransferMethodID methodID for standard transfer
var standardTransferMethodID string
var transferWithNoteMethodID string

func init() {
	standardTransferMethodID = algorithm.ConvertMethodID(algorithm.CalcMethodId(std.TransferPrototype))
	transferWithNoteMethodID = algorithm.ConvertMethodID(algorithm.CalcMethodId(std.TransferWithNotePrototype))
}

// TokenHelper token helper information
type TokenHelper struct {
	smc sdk.ISmartContract //指向智能合约API对象指针
}

var _ sdk.ITokenHelper = (*TokenHelper)(nil)
var _ sdkimpl.IAcquireSMC = (*TokenHelper)(nil)

// SMC get smart contract object
func (th *TokenHelper) SMC() sdk.ISmartContract { return th.smc }

// SetSMC set smart contract object
func (th *TokenHelper) SetSMC(smc sdk.ISmartContract) { th.smc = smc }

// Define the minimum of token supply for issuing new token.
// It's one TOKEN, 1,000,000,000 cong
const minTokenSupply = 1000000000

// token name can be down to 3 characters
const minTokenNameSize = 3

// token name can be up to 40 characters
const maxTokenNameSize = 40

// token symbol can be down to 3 characters
const minTokenSymbolSize = 3

// token symbol can be up to 20 characters
const maxTokenSymbolSize = 20

// RegisterToken register new token to block chain with many property
func (th *TokenHelper) RegisterToken(
	name, symbol string,
	totalSupply bn.Number,
	addSupplyEnabled, burnEnabled bool) (token sdk.IToken) {

	sdk.RequireMainChain()
	// check sender
	sdk.RequireOwner()

	th.checkContract()

	// check contract's token
	sdk.Require(th.smc.Message().Contract().Token() == "",
		types.ErrInvalidParameter, "The contract has registered token already")

	// check token name
	th.checkName(name)

	// check token symbol
	th.checkSymbol(symbol)

	// totalSupply cannot be less than 1Token
	sdk.Require(totalSupply.IsGEI(minTokenSupply),
		types.ErrInvalidParameter, fmt.Sprintf("The totalSupply cannot be less than %d", minTokenSupply))

	tokenAddr := th.smc.Message().Contract().Address()
	token = object.NewToken(th.smc,
		tokenAddr,
		th.smc.Message().Contract().Owner().Address(),
		name,
		symbol,
		totalSupply,
		addSupplyEnabled,
		burnEnabled,
		th.BaseGasPrice())

	llState := th.smc.(*sdkimpl.SmartContract).LlState()
	// set token's data
	llState.McSet(std.KeyOfToken(token.Address()), token.(*object.Token).StdToken())
	// set token's name with address's value
	llState.McSet(std.KeyOfTokenWithName(name), &tokenAddr)
	// set token's symbol with address's value
	llState.McSet(std.KeyOfTokenWithSymbol(symbol), &tokenAddr)

	// get contract's list with current contract name
	th.smc.Helper().ContractHelper().(*ContractHelper).UpdateContractsToken(token.Address())

	// update owner's balance and account's token key
	ownerAcct := token.Owner()
	ownerAcct.(*object.Account).SetBalanceOfToken(token.Address(), token.TotalSupply())
	ownerAcct.(*object.Account).AddAccountTokenKey(std.KeyOfAccountToken(ownerAcct.Address(), tokenAddr))

	// update all token list
	th.updateAllTokenList(tokenAddr)

	// fire event of NewToken
	th.smc.Helper().ReceiptHelper().Emit(
		std.NewToken{
			TokenAddress:     token.Address(),
			ContractAddress:  th.smc.Message().Contract().Address(),
			Owner:            token.Owner().Address(),
			Name:             name,
			Symbol:           symbol,
			TotalSupply:      totalSupply,
			AddSupplyEnabled: addSupplyEnabled,
			BurnEnabled:      burnEnabled,
			GasPrice:         token.GasPrice(),
		},
	)

	// fire event of Transfer
	th.smc.Helper().ReceiptHelper().Emit(
		std.Transfer{
			Token: token.Address(),
			From:  "",
			To:    token.Owner().Address(),
			Value: totalSupply,
		},
	)

	return
}

func (th *TokenHelper) checkContract() {
	transferMethodExist := false
	transferInterExist := false
	transferWithNoteInterExist := false

	// check the contract that it must defined standard transfer method
	for _, method := range th.smc.Message().Contract().Methods() {
		if method.MethodID == standardTransferMethodID {
			transferMethodExist = true
			break
		}
	}

	// check the contract that it must defined standard transfer interface
	for _, inter := range th.smc.Message().Contract().Interfaces() {
		if inter.MethodID == standardTransferMethodID {
			transferInterExist = true
			break
		}
	}

	// check the contract that it must defined standard transfer interface
	for _, inter := range th.smc.Message().Contract().Interfaces() {
		if inter.MethodID == transferWithNoteMethodID {
			transferWithNoteInterExist = true
			break
		}
	}

	sdk.Require(transferMethodExist == true &&
		transferInterExist == true &&
		transferWithNoteInterExist == true,
		types.ErrInvalidParameter, "This contract never defined standard transfer method")
}

// Token get token that registered by current contract
func (th *TokenHelper) Token() sdk.IToken {
	if th.smc.Message().Contract().Token() == "" {
		return nil
	}

	return th.tokenOfAddress(th.smc.Message().Contract().Token())
}

// TokenOfAddress get token with address
func (th *TokenHelper) TokenOfAddress(tokenAddr types.Address) sdk.IToken {
	sdk.RequireAddress(tokenAddr)

	return th.tokenOfAddress(tokenAddr)
}

// TokenOfName get token with name
func (th *TokenHelper) TokenOfName(name string) sdk.IToken {
	tokenAddr := th.tokenAddressOfName(name)
	if tokenAddr == "" {
		return nil
	}

	return th.tokenOfAddress(tokenAddr)
}

// TokenOfSymbol get token with symbol
func (th *TokenHelper) TokenOfSymbol(symbol string) sdk.IToken {
	tokenAddr := th.tokenAddressOfSymbol(symbol)
	if tokenAddr == "" {
		return nil
	}

	return th.tokenOfAddress(tokenAddr)
}

// TokenOfContract get token with contract address
func (th *TokenHelper) TokenOfContract(contractAddr types.Address) sdk.IToken {
	// get contract
	contract := th.smc.Helper().ContractHelper().ContractOfAddress(contractAddr)
	if contract == nil {
		return nil
	}

	// check token address
	if contract.Token() == "" {
		return nil
	}

	// get token
	keyOfToken := std.KeyOfToken(contract.Token())
	stdToken := th.smc.(*sdkimpl.SmartContract).LlState().McGet(keyOfToken, &std.Token{})
	if stdToken == nil {
		sdkimpl.Logger.Error("[sdk]Cannot load contract's token, contractAddr=%s", contractAddr)
		return nil
	}

	return object.NewTokenFromSTD(th.smc, stdToken.(*std.Token))
}

// BaseGasPrice get base gasPrice
func (th *TokenHelper) BaseGasPrice() int64 {
	key := std.KeyOfTokenBaseGasPrice()

	return th.smc.(*sdkimpl.SmartContract).LlState().GetInt64(key)
}

// CheckActivate check the to chain have activated current token
func (th *TokenHelper) CheckActivate(address types.Address) error {
	tokenAddr := th.smc.Message().Contract().Token()
	if tokenAddr == "" {
		return errors.New("this contract not exist token")
	}

	toChainID := th.smc.Helper().BlockChainHelper().GetChainID(address)
	if toChainID != th.smc.Helper().BlockChainHelper().GetMainChainID() {
		key := std.KeyOfSupportSideChains(tokenAddr)
		supportSideChains := *th.smc.(*sdkimpl.SmartContract).LlState().GetEx(key, new([]string)).(*[]string)
		index := sort.SearchStrings(supportSideChains, toChainID)
		chainName := toChainID[strings.Index(toChainID, "[")+1 : len(toChainID)-1]
		if index == len(supportSideChains) || supportSideChains[index] != toChainID {
			return errors.New(fmt.Sprintf("chain: %s never activate current token", chainName))
		}
	}

	return nil
}

// TokenOfAddress get token with address
func (th *TokenHelper) tokenOfAddress(tokenAddr types.Address) sdk.IToken {
	key := std.KeyOfToken(tokenAddr)
	stdToken := th.smc.(*sdkimpl.SmartContract).LlState().McGet(key, &std.Token{})
	if stdToken == nil {
		return nil
	}

	return object.NewTokenFromSTD(th.smc, stdToken.(*std.Token))
}

// checkName check the token name is right or not
func (th *TokenHelper) checkName(name string) {
	sdk.Require(len(name) >= minTokenNameSize,
		types.ErrInvalidParameter,
		fmt.Sprintf("Token name cannot be less than %d characters", minTokenNameSize))

	sdk.Require(len(name) <= maxTokenNameSize,
		types.ErrInvalidParameter,
		fmt.Sprintf("The token name's length cannot great than %d", maxTokenNameSize))

	sdk.Require(th.tokenAddressOfName(name) == "",
		types.ErrInvalidParameter, "The token's name is exist")
}

// TokenAddressOfName get token address with name
func (th *TokenHelper) tokenAddressOfName(name string) types.Address {
	key := std.KeyOfTokenWithName(name)

	return *(th.smc.(*sdkimpl.SmartContract).LlState().McGetEx(key, new(types.Address)).(*types.Address))
}

// checkSymbol check the token symbol is right or not
func (th *TokenHelper) checkSymbol(symbol string) {
	sdk.Require(len(symbol) >= minTokenSymbolSize,
		types.ErrInvalidParameter,
		fmt.Sprintf("Token symbol cannot be less than %d characters", minTokenNameSize))

	sdk.Require(len(symbol) <= maxTokenSymbolSize,
		types.ErrInvalidParameter,
		fmt.Sprintf("The token symbol's length cannot great than %d", maxTokenSymbolSize))

	sdk.Require(th.tokenAddressOfSymbol(symbol) == "",
		types.ErrInvalidParameter, "The token's symbol is exist")
}

// TokenAddressOfSymbol get token address with symbol
func (th *TokenHelper) tokenAddressOfSymbol(symbol string) types.Address {
	key := std.KeyOfTokenWithSymbol(symbol)

	return *(th.smc.(*sdkimpl.SmartContract).LlState().McGetEx(key, new(types.Address)).(*types.Address))
}

func (th *TokenHelper) updateAllTokenList(tokenAddr types.Address) {
	key := std.KeyOfAllToken()

	llState := th.smc.(*sdkimpl.SmartContract).LlState()

	allTokens := make([]types.Address, 0)
	allTokens = *llState.GetEx(key, &allTokens).(*[]types.Address)

	for _, token := range allTokens {
		if token == tokenAddr {
			return
		}
	}

	allTokens = append(allTokens, tokenAddr)

	llState.Set(key, allTokens)
}
