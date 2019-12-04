package std

import (
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/types"
	"fmt"
)

// AccountInfo account information
type AccountInfo struct {
	Address       types.Address `json:"address"`
	Balance       bn.Number     `json:"balance"`
	BVMBalanceMod *bn.Number    `json:"bbMod,omitempty"` // 十八位精度的后九位
}

// KeyOfAccount the access key of account address
// data for this key is []Address
func KeyOfAccount(address types.Address) string { return "/account/ex/" + address }

// KeyOfAccountNonce the access key for account's nonce
func KeyOfAccountNonce(exAddress types.Address) string {
	return "/account/ex/" + exAddress + "/account"
}

// KeyOfAccountToken the access key for account information in state database
// data for this key refer AccountInfo
func KeyOfAccountToken(accountAddr, tokenAddr types.Address) string {
	return fmt.Sprintf("/account/ex/%v/token/%v", accountAddr, tokenAddr)
}

// KeyOfAccountContracts the access key for account's contract address list in state database
// data for this key is []types.Address
func KeyOfAccountContracts(accountAddr types.Address) string {
	return "/account/ex/" + accountAddr + "/contracts"
}
