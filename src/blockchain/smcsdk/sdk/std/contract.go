package std

import (
	"blockchain/smcsdk/sdk/types"
	"fmt"

	"github.com/tendermint/go-crypto"
)

const (
	GenesisContract           = "bcb6Datuf5ww47ZmdhVbU1X4bjL6suohkmDU"
	TransferPrototype         = "Transfer(types.Address,bn.Number)"
	TransferWithNotePrototype = "TransferWithNote(types.Address,bn.Number,string)"
)

// ContractVersionList contract version information with addresses and effectHeights
type ContractVersionList struct {
	Name             string          `json:"name"`             // 合约名称
	ContractAddrList []types.Address `json:"contractAddrList"` // 合约地址列表
	EffectHeights    []int64         `json:"effectHeights"`    // 合约生效高度列表
}

// Organization organization information
type Organization struct {
	OrgID            string          `json:"orgID"`            // 组织机构ID
	Name             string          `json:"name"`             // 组织名字
	OrgOwner         types.Address   `json:"orgOwner"`         // 组织拥有者地址
	ContractAddrList []types.Address `json:"contractAddrList"` // 合约地址列表
	OrgCodeHash      []byte          `json:"orgCodeHash"`      // 组织机构代码hash
	Signers          []types.PubKey  `json:"signers"`          // 签名公钥列表
}

// OrgDeveloper the developer's pubKey related OrgID
type OrgDeveloper struct {
	PublicKey []byte   `json:"publicKey"` // 开发者公钥
	OrgID     []string `json:"orgID"`     // 开发者所属组织列表
}

// Method method information
type Method struct {
	MethodID  string `json:"methodId"`  //方法ID
	Gas       int64  `json:"gas"`       //方法需要消耗的燃料
	ProtoType string `json:"prototype"` //方法原型
}

// Contract contract detail information
type Contract struct {
	Address      types.Address `json:"address"`        //合约地址
	Account      types.Address `json:"account"`        //合约的账户地址
	Owner        types.Address `json:"owner"`          //合约拥有者的账户地址
	Name         string        `json:"name"`           //合约名称
	Version      string        `json:"version"`        //合约版本
	CodeHash     types.Hash    `json:"codeHash"`       //合约代码的哈希
	EffectHeight int64         `json:"effectHeight"`   //合约生效的区块高度
	LoseHeight   int64         `json:"loseHeight"`     //合约失效的区块高度
	KeyPrefix    string        `json:"keyPrefix"`      //合约在状态数据库中KEY值的前缀
	Methods      []Method      `json:"methods"`        //合约对外提供接口的方法列表
	Interfaces   []Method      `json:"interfaces"`     //合约提供的跨合约调用的方法列表
	Mine         []Method      `json:"mine"`           //合约提供的挖矿方法
	IBCs         []Method      `json:"ibcs,omitempty"` //合约提供的执行跨链业务的方法列表
	Token        types.Address `json:"token"`          //合约代币地址
	OrgID        string        `json:"orgID"`          //组织ID
	ChainVersion int64         `json:"chainVersion"`   //链版本
}

// BuildResult build result information
type BuildResult struct {
	Code        uint32   `json:"code"`
	Error       string   `json:"error"`
	Methods     []Method `json:"methods"`
	Interfaces  []Method `json:"interfaces"`
	Mine        []Method `json:"mine"`
	IBCs        []Method `json:"ibcs"`
	OrgCodeHash []byte   `json:"orgCodeHash"`
}

// GenResult - generate code's result
type GenResult struct {
	ContractName string   `json:"contractName"`
	Version      string   `json:"version"`
	OrgID        string   `json:"orgID"`
	Methods      []Method `json:"methods"`
	Interfaces   []Method `json:"interfaces"`
	Mine         []Method `json:"mine"`
	IBCs         []Method `json:"ibcs"`
}

// ContractMeta contract's meta information
type ContractMeta struct {
	Name         string        `json:"name"`
	ContractAddr types.Address `json:"contractAddr"`
	OrgID        string        `json:"orgID"`
	Version      string        `json:"version"`
	EffectHeight int64         `json:"effectHeight"`
	LoseHeight   int64         `json:"loseHeight"`
	CodeData     []byte        `json:"codeData"`
	CodeHash     []byte        `json:"codeHash"`
	CodeDevSig   []byte        `json:"codeDevSig"`
	CodeOrgSig   []byte        `json:"codeOrgSig"`
}

// ContractWithEffectHeight contract address and is upgrade or not for effect height
type ContractWithEffectHeight struct {
	ContractAddr types.Address `json:"contractAddr"`
	IsUpgrade    bool          `json:"isUpgrade"`
}

// MineContract contract address and height of mine
type MineContract struct {
	MineHeight int64         `json:"mineHeight"` // 开发挖矿高度
	Address    types.Address `json:"address"`    // 合约地址
}

// KeyOfMineContracts key of all MineContract list
// data for this key refer []MineContract
func KeyOfMineContracts() string { return "/contract/mines" }

// KeyOfContractWithEffectHeight the access key for effective contract with height
// data for this key refer ContractWithEffectHeight
func KeyOfContractWithEffectHeight(height string) string { return "/" + height }

// KeyOfAllContracts the access key for all contract
// data for this key refer []types.Address
func KeyOfAllContracts() string { return "/contract/all/0" }

// KeyOfContract the access key for contract in state database
// data for this key refer Contract
func KeyOfContract(contractAddr types.Address) string { return "/contract/" + contractAddr }

// KeyOfContractsWithName the access key for contract's addresses and effectHeights in state database
// data for this key refer ContractVersionList
func KeyOfContractsWithName(orgID, name string) string {
	return fmt.Sprintf("/contract/%s/%s", orgID, name)
}

// KeyOfContractCode the access key for contract's code in state database
// data for this key refer ContractCode
func KeyOfContractCode(contractAddr types.Address) string { return "/contract/code/" + contractAddr }

// KeyOfGenesisContract for create key with contract address of token
// data for this key refer Contract
func KeyOfGenesisContract(contractAddr string) string { return "/genesis/sc/" + contractAddr }

// KeyOfGenesisContractAddrList for create key of genesis contracts
// data for this key refer []types.Address
func KeyOfGenesisContractAddrList() string { return "/genesis/contracts" }

// GetGenesisContractAddr get genesis contract addr
func GetGenesisContractAddr(chainID string) string {
	pubKey := [32]byte{}
	p := crypto.PubKeyEd25519FromBytes(pubKey[:])
	addr := p.Address(chainID)
	return addr
}

// KeyOfOrg for create key of organization
// data for this key refer orgID
func GetOrganizaitionInfo(OrgID string) string { return "/organization/" + OrgID }
