package helper

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
	"blockchain/smcsdk/sdkimpl/object"
	"bytes"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/tendermint/go-crypto"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"
)

// BlockChainHelper block chain helper information
type BlockChainHelper struct {
	smc sdk.ISmartContract
}

var _ sdk.IBlockChainHelper = (*BlockChainHelper)(nil)
var _ sdkimpl.IAcquireSMC = (*BlockChainHelper)(nil)

// SMC get smartContract object
func (bh *BlockChainHelper) SMC() sdk.ISmartContract { return bh.smc }

// SetSMC set smartContract object
func (bh *BlockChainHelper) SetSMC(smc sdk.ISmartContract) { bh.smc = smc }

// IsPeerChainAddress check address if not local return true, else return false
func (bh *BlockChainHelper) IsPeerChainAddress(address types.Address) bool {
	addrChainID := bh.GetChainID(address)
	sdk.RequireAddressEx(addrChainID, address)

	chainID := bh.smc.Block().ChainID()
	if chainID != addrChainID {
		return true
	}

	return false
}

// IsSideChain check chainID if contains '<' then return true, else return false
func (bh *BlockChainHelper) IsSideChain() bool {
	return strings.Contains(bh.smc.Block().ChainID(), "[")
}

// CalcSideChainID calculate chainID from chainName,
func (bh *BlockChainHelper) CalcSideChainID(chainName string) string {
	mainChainID := bh.GetMainChainID()

	r, _ := regexp.Compile("^[A-Za-z][a-zA-Z0-9_]{1,39}$")
	sdk.Require(r.MatchString(chainName),
		types.ErrInvalidParameter, "invalid chainName")

	return mainChainID + "[" + chainName + "]"
}

// FormatTime format tm to layout defined string
func (bh *BlockChainHelper) FormatTime(tm int64, layout string) string {
	return time.Unix(tm, 0).Format(layout)
}

// CalcAccountFromPubKey calculate account address from pubKey
func (bh *BlockChainHelper) CalcAccountFromPubKey(pubKey types.PubKey) types.Address {
	sdk.Require(pubKey != nil && len(pubKey) == 32,
		types.ErrInvalidParameter, "invalid pubKey")

	pk := crypto.PubKeyEd25519FromBytes(pubKey)
	return pk.Address(bh.smc.Helper().GenesisHelper().ChainID())
}

// CalcAccountFromName calculate account address from name
func (bh *BlockChainHelper) CalcAccountFromName(name string, orgID string) types.Address {
	return bh.CalcContractAddress(name, "", orgID)
}

// nolint unhandled
// CalcContractAddress calculate contract address from name、version and owner
func (bh *BlockChainHelper) CalcContractAddress(name string, version string, orgID string) types.Address {
	mainChainID := bh.GetMainChainID()

	hasherSHA3256 := sha3.New256()
	hasherSHA3256.Write([]byte(mainChainID))
	hasherSHA3256.Write([]byte(name))
	hasherSHA3256.Write([]byte(version))
	hasherSHA3256.Write([]byte(orgID))
	sha := hasherSHA3256.Sum(nil)

	hasherRIPEMD160 := ripemd160.New()
	hasherRIPEMD160.Write(sha) // does not error
	rpd := hasherRIPEMD160.Sum(nil)

	hasher := ripemd160.New()
	hasher.Write(rpd)
	md := hasher.Sum(nil)

	addr := make([]byte, 0, len(rpd)+len(md[:4]))
	addr = append(addr, rpd...)
	addr = append(addr, md[:4]...)

	return bh.smc.Helper().GenesisHelper().ChainID() + base58.Encode(addr)
}

// RecalcAddress recalculate address with chainID
func (bh *BlockChainHelper) RecalcAddress(address types.Address, chainID string) types.Address {
	addrChainID := bh.GetChainID(address)

	return chainID + address[len(addrChainID):]
}

// GetChainID get chainID from address
func (bh *BlockChainHelper) GetChainID(address types.Address) string {
	chainID := bh.GetMainChainID()
	if strings.Contains(address, "[") {
		chainID = address[:strings.Index(address, "]")+1]
	}

	sdk.RequireAddressEx(chainID, address)

	return chainID
}

// GetMainChainID get mainChainID from local chainID
func (bh *BlockChainHelper) GetMainChainID() string {
	chainID := bh.smc.Block().ChainID()
	if strings.Contains(chainID, "[") {
		return chainID[:strings.Index(chainID, "[")]
	}

	return chainID
}

// nolint unhandled
// CalcOrgID calculate organization ID
func (bh *BlockChainHelper) CalcOrgID(name string) string {
	hasherSHA3256 := sha3.New256()
	hasherSHA3256.Write([]byte(name))
	sha := hasherSHA3256.Sum(nil)

	hasherRIPEMD160 := ripemd160.New()
	hasherRIPEMD160.Write(sha) // does not error
	rpd := hasherRIPEMD160.Sum(nil)

	hasher := ripemd160.New()
	hasher.Write(rpd)
	md := hasher.Sum(nil)

	addr := make([]byte, 0, len(rpd)+len(md[:4]))
	addr = append(addr, rpd...)
	addr = append(addr, md[:4]...)

	return "org" + base58.Encode(addr)
}

// GetBlock get block data with height
func (bh *BlockChainHelper) GetBlock(height int64) sdk.IBlock {
	if height <= 0 || height > bh.smc.Block().Height() {
		return nil
	}

	transID := bh.smc.(*sdkimpl.SmartContract).LlState().TransID()
	v := sdkimpl.GetBlockFunc(transID, height)

	block := object.NewBlock(
		bh.smc,
		v.ChainID,
		v.Version,
		v.BlockHash,
		v.DataHash,
		v.Height,
		v.Time,
		v.NumTxs,
		v.ProposerAddress,
		v.RewardAddress,
		v.RandomNumber,
		v.LastBlockHash,
		v.LastCommitHash,
		v.LastAppHash,
		v.LastFee,
	)

	return block
}

// nolint unhandled
// CheckAddress check address and return result
func (bh *BlockChainHelper) CheckAddress(addr types.Address) error {
	chainID := bh.smc.Block().ChainID()

	return bh.CheckAddressEx(chainID, addr)
}

// nolint unhandled
// CheckAddressEx check address and return result
func (bh *BlockChainHelper) CheckAddressEx(chainID string, address types.Address) error {
	if strings.HasPrefix(address, chainID) == false {
		return errors.New("Address chainID is error! ")
	}

	base58Addr := strings.Replace(address, chainID, "", 1)
	addrData := base58.Decode(base58Addr)
	addrLen := len(addrData)
	if addrLen < 4 {
		return errors.New("Base58Addr parse error! ")
	}

	r160 := ripemd160.New()
	r160.Write(addrData[:addrLen-4])
	md := r160.Sum(nil)

	if bytes.Compare(md[:4], addrData[addrLen-4:]) != 0 {
		return errors.New("Address checksum is error! ")
	}
	return nil
}

// GetCurrentBlock get current block data
func GetCurrentBlock(smc sdk.ISmartContract) sdk.IBlock {
	// get current block if height is 0
	v := sdkimpl.GetBlockFunc(smc.(*sdkimpl.SmartContract).LlState().TransID(), 0)

	block := object.NewBlock(
		smc,
		v.ChainID,
		v.Version,
		v.BlockHash,
		v.DataHash,
		v.Height,
		v.Time,
		v.NumTxs,
		v.ProposerAddress,
		v.RewardAddress,
		v.RandomNumber,
		v.LastBlockHash,
		v.LastCommitHash,
		v.LastAppHash,
		v.LastFee,
	)

	return block
}
