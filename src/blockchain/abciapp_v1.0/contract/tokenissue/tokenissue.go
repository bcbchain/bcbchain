package tokenissue

import (
	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/contract"
	"blockchain/abciapp_v1.0/smc"
	"common/bignumber_v1.0"
	"math/big"
)

// TokenIssue is a reference of Contract structure
type TokenIssue struct {
	*contract.Contract
}

// Define the minimum of token supply for issuing new token.
// It's one TOKEN, 1,000,000,000 cong
const MIN_TOKEN_SUPPLY = 1000000000

// token name can be up to 40 characters
const MAX_TOKEN_NAME_SIZE = 40

// token symbol can be up to 20 characters
const MAX_TOKEN_SYMBOL_SIZE = 20

//NewToken is a used to issue new token
func (contract *TokenIssue) NewToken(name string,
	symbol string,
	totalSupply big.Int,
	addSupplyEnabled bool,
	burnEnabled bool,
	gasprice uint64) (addr smc.Address, smcError smc.Error) {

	if len(name) == 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid name: token name cannot be empty"
		return
	} else if len(name) > MAX_TOKEN_NAME_SIZE {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid name: token name cannot be more than 40 characters"
		return
	} else if len(symbol) == 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid symbol: token symbol cannot be empty"
		return
	} else if len(symbol) > MAX_TOKEN_SYMBOL_SIZE {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid symbol: token symbol cannot be more than 20 characters"
		return
	} else if bignumber.Compare(totalSupply, *big.NewInt(MIN_TOKEN_SUPPLY)) < 0 &&
		bignumber.Compare(totalSupply, bignumber.Zero()) != 0 {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidParameter
		smcError.ErrorDesc = "Invalid supply: token supply is less than limit"
		return
	}

	// Checking specified token name and symbol, return error it it's duplicated.
	token := contract.Token()
	if smcError = token.CheckNameAndSybmol(name, symbol); smcError.ErrorCode != bcerrors.ErrCodeOK {
		return
	}

	// Token's gas price could not be smaller than GasBasePrice
	// and up to Max_Gas_Price (1E9 cong)
	if gasprice < contract.GasBasePrice() || gasprice > smc.Max_Gas_Price {
		smcError.ErrorCode = bcerrors.ErrCodeInterContractsInvalidGasPrice
		return
	}

	currency := contract.CalcAddress(name)
	return currency,
		contract.CreateToken(currency,
			name,
			symbol,
			totalSupply,
			addSupplyEnabled,
			burnEnabled,
			gasprice)

}
