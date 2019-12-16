package sdk

import (
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/ibc"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
)

// IBlock the interface for block
type IBlock interface {
	ChainID() string                //链ID
	BlockHash() types.Hash          //区块哈希
	Height() int64                  //区块高度
	Time() int64                    //区块时间（单位为秒，始于1970-01-01 00:00:00）
	Now() bn.Number                 //将区块时间定义为区块链系统的当前时间
	NumTxs() int32                  //区块中包含的交易笔数
	DataHash() types.Hash           //区块中Data字段的哈希
	ProposerAddress() types.Address //区块提案者地址
	RewardAddress() types.Address   //接收区块奖励的地址
	RandomNumber() types.HexBytes   //区块随机数（取Linux系统的真随机数）
	Version() string                //获取当前区块提案人的软件版本
	LastBlockHash() types.Hash      //上一区块的区块哈希
	LastCommitHash() types.Hash     //上一区块的确认信息哈希
	LastAppHash() types.Hash        //上一区块的应用层哈希
	LastFee() int64                 //上一区块的手续费总和（单位为Cong）
}

// ITx the interface for tx
type ITx interface {
	Note() string     //交易的备注
	GasLimit() int64  //交易传入的最大燃料数
	GasLeft() int64   //剩余的燃料数
	TxHash() []byte   //交易hash
	Signer() IAccount //交易发签名者的账户信息
}

// IMessage the interface for Message
type IMessage interface {
	Contract() IContract           //消息调用的智能合约对象
	MethodID() string              //消息调用的智能合约方法ID
	Items() []types.HexBytes       //消息的参数数据字段的原始信息
	GasPrice() int64               //消息的燃料价格
	Sender() IAccount              //消息发送者的账户信息
	Payer() IAccount               //手续费支付者的账户信息
	Origins() []types.Address      //消息完整的调用链（用于记录跨合约调用的合约链）
	InputReceipts() []types.KVPair //级联消息中前一个消息输出的收据作为本次消息的输入

	GetTransferToMe() []*std.Transfer //获取级联消息中前一个消息中第一个转给现在这个合约的转账收据
}

// IAccount the interface for Account
type IAccount interface {
	Address() types.Address                                                                      //账户地址
	PubKey() types.PubKey                                                                        //账户公钥
	Balance() bn.Number                                                                          //账户当前合约注册的代币的余额（cong）
	BalanceOfToken(token types.Address) bn.Number                                                //根据地址获取代币或基础通证的余额（cong）
	BalanceOfName(name string) bn.Number                                                         //根据名称获取代币或基础通证的余额（cong）
	BalanceOfSymbol(symbol string) bn.Number                                                     //根据符号获取代币或基础通证的余额（cong）
	Transfer(to types.Address, value bn.Number)                                                  //执行当前合约注册代币的转账（cong）
	TransferWithNote(to types.Address, value bn.Number, note string)                             //执行当前合约注册代币的转账，收据中包含 note 信息（cong）
	TransferWithNoteEx(to types.Address, value bn.Number, note string) types.Error               //执行当前合约注册代币的转账（cong）
	TransferByToken(token types.Address, to types.Address, value bn.Number)                      //根据地址执行代币或基础通证的转账（cong）
	TransferByTokenWithNote(token types.Address, to types.Address, value bn.Number, note string) //根据地址执行代币或基础通证的转账，收据中包含 note 信息（cong）
	TransferByName(name string, to types.Address, value bn.Number)                               //根据名称执行代币或基础通证的转账（cong）
	TransferByNameWithNote(name string, to types.Address, value bn.Number, note string)          //根据名称执行代币或基础通证的转账，收据中包含 note 信息（cong）
	TransferBySymbol(symbol string, to types.Address, value bn.Number)                           //根据符号执行代币或基础通证的转账（cong）
	TransferBySymbolWithNote(symbol string, to types.Address, value bn.Number, note string)      //根据符号执行代币或基础通证的转账，收据中包含 note 信息（cong）
}

// IContract the interface for Contract
type IContract interface {
	Address() types.Address   //合约地址
	Account() IAccount        //合约的账户对象
	Owner() IAccount          //合约拥有者的账户对象
	Name() string             //合约名称
	Version() string          //合约版本
	CodeHash() types.Hash     //合约代码的哈希
	EffectHeight() int64      //合约生效的区块高度
	LoseHeight() int64        //合约失效的区块高度
	KeyPrefix() string        //合约在状态数据库中KEY值的前缀
	Methods() []std.Method    //合约对外提供接的方法列表
	Interfaces() []std.Method //合约对外提供的跨合约调用方法列表
	IBCs() []std.Method       //合约对外提供的跨合约调用方法列表
	Mine() []std.Method       //合约提供的挖矿方法
	Token() types.Address     //合约代币地址
	OrgID() string            //组织ID
	ChainVersion() int64      //链版本

	SetOwner(owner types.Address) //修改合约拥有者
}

// IToken the interface for Token
type IToken interface {
	Address() types.Address               //代币地址
	Owner() IAccount                      //代币拥有者的账户对象
	Name() string                         //代币的名称
	Symbol() string                       //代币的符号
	TotalSupply() bn.Number               //代币的总供应量
	AddSupplyEnabled() bool               //代币是否支持增发
	BurnEnabled() bool                    //代币是否支持燃烧
	GasPrice() int64                      //代币燃料价格
	SetTotalSupply(totalSupply bn.Number) //设置代币的总供应量
	SetGasPrice(gasPrice int64)           //设置代币燃料价格
}

// IHelper the interface for Helper
type IHelper interface {
	AccountHelper() IAccountHelper       //账户相关的Helper对象
	BlockChainHelper() IBlockChainHelper //区块链相关的Helper对象
	BuildHelper() IBuildHelper           //合约编译相关的Helper对象
	ContractHelper() IContractHelper     //合约相关的Helper对象
	GenesisHelper() IGenesisHelper       //创世相关的Helper对象
	ReceiptHelper() IReceiptHelper       //事件相关的Helper对象
	TokenHelper() ITokenHelper           //通证相关的Helper对象
	StateHelper() IStateHelper           //状态相关的Helper对象
	IBCHelper() IIBCHelper               //跨链数据相关的Helper对象
	IBCStubHelper() IIBCStubHelper       //跨链执行相关的Helper对象
}

// IAccountHelper the interface for account helper
type IAccountHelper interface {
	AccountOf(addr types.Address) IAccount        //根据账户地址构造账户信息对象
	AccountOfPubKey(pubkey types.PubKey) IAccount //根据账户公钥构造账户信息对象
}

// IBlockChainHelper the interface for block chain helper
type IBlockChainHelper interface {
	IsPeerChainAddress(address types.Address) bool //判断给定地址是否外链地址
	IsSideChain() bool                             //判断当前链是否是侧链

	CalcSideChainID(chainName string) string                                     //根据链名称计算链ID
	CalcAccountFromPubKey(pubKey types.PubKey) types.Address                     //根据用户公钥计算账户地址
	CalcAccountFromName(name string, orgID string) types.Address                 //根据合约名称和组织ID计算合约的账户地址
	CalcContractAddress(name string, version string, orgID string) types.Address //根据合约名称、版本与组织ID计算合约地址
	RecalcAddress(address types.Address, chainID string) types.Address           //根据给定地址和chainID重新计算新地址
	CalcOrgID(name string) string                                                //根据公钥计算组织ID
	CheckAddress(addr types.Address) error                                       //根据当前链chainID检查地址是否合法
	CheckAddressEx(chainID string, addr types.Address) error                     //根据给定chainID检查地址是否合法

	GetBlock(height int64) IBlock            //根据高度读取区块信息
	GetChainID(address types.Address) string // 从地址中获取链ID
	GetMainChainID() string                  // 从当前链ID中获取主链ID

	// time
	FormatTime(seconds int64, layout string) string // 输出格式化时间
	ParseTime(layout, value string) (int64, error)  // 输出格式化时间
}

// IBuildHelper the interface for build helper
type IBuildHelper interface {
	Build(meta std.ContractMeta) std.BuildResult
}

// IContractHelper the interface for contract helper
type IContractHelper interface {
	ContractOfAddress(addr types.Address) IContract    //根据合约地址构造合约信息对象
	ContractOfToken(tokenAddr types.Address) IContract //根据代币地址构造合约信息对象（当前区块可用）
	ContractOfName(name string) IContract              //根据合约名字返回当前有效合约对象
}

// IReceiptHelper the interface for receipt helper
type IReceiptHelper interface {
	Emit(receiptData interface{}) //发送一个事件
}

// IGenesisHelper the interface for genesis helper
type IGenesisHelper interface {
	ChainID() string        //读取创世时的链ID
	OrgID() string          //读取创世时的组织ID
	Contracts() []IContract //读取创世合约信息
	Token() IToken          //读取创世通证（基础通证）的信息
	GasPriceRatio() uint64  //燃料价格调整比例
}

// ITokenHelper the interface for token helper
type ITokenHelper interface {
	RegisterToken(name, symbol string, totalSupply bn.Number, addSupplyEnabled, burnEnabled bool) IToken //注册一个新的代币
	Token() IToken                                                                                       //获取合约代币的信息
	TokenOfAddress(tokenAddr types.Address) IToken                                                       //根据代币地址获取代币或基础通证的信息
	TokenOfName(name string) IToken                                                                      //根据代币名称获取代币或基础通证的信息
	TokenOfSymbol(symbol string) IToken                                                                  //根据代币符号获取代币或基础通证的信息
	TokenOfContract(contractAddr types.Address) IToken                                                   //根据合约地址获取代币或基础通证的信息
	BaseGasPrice() int64                                                                                 //基础燃料价格
	CheckActivate(address types.Address) error                                                           //判断指定地址对应的侧链是否已经激活当前合约代币
}

// IStateHelper the interface for state helper
type IStateHelper interface {
	Check(key string) bool   // 判断指定的key对应的数据是否存在
	McCheck(key string) bool // 判断指定的key对应的数据是否存在

	//Get
	Get(key string, defaultValue interface{}) interface{}   //从状态数据库中读取指定KEY对应的数据，不存在返回空
	GetEx(key string, defaultValue interface{}) interface{} //从状态数据库中读取指定KEY对应的数据，不存在返回默认值

	//GetXXX
	GetInt(key string) int
	GetInt8(key string) int8
	GetInt16(key string) int16
	GetInt32(key string) int32
	GetInt64(key string) int64
	GetUint(key string) uint
	GetUint8(key string) uint8
	GetUint16(key string) uint16
	GetUint32(key string) uint32
	GetUint64(key string) uint64
	GetByte(key string) byte
	GetBool(key string) bool
	GetString(key string) string

	//GetXXXs
	GetInts(key string) []int
	GetInt8s(key string) []int8
	GetInt16s(key string) []int16
	GetInt32s(key string) []int32
	GetInt64s(key string) []int64
	GetUints(key string) []uint
	GetUint8s(key string) []uint8
	GetUint16s(key string) []uint16
	GetUint32s(key string) []uint32
	GetUint64s(key string) []uint64
	GetBytes(key string) []byte
	GetBools(key string) []bool
	GetStrings(key string) []string

	//Set
	Set(key string, value interface{}) //向状态数据库设置指定KEY对应的数据

	//SetXXX
	SetInt(key string, v int)
	SetInt8(key string, v int8)
	SetInt16(key string, v int16)
	SetInt32(key string, v int32)
	SetInt64(key string, v int64)
	SetUint(key string, v uint)
	SetUint8(key string, v uint8)
	SetUint16(key string, v uint16)
	SetUint32(key string, v uint32)
	SetUint64(key string, v uint64)
	SetByte(key string, v byte)
	SetBool(key string, v bool)
	SetString(key string, v string)

	//SetXXXs
	SetInts(key string, v []int)
	SetInt8s(key string, v []int8)
	SetInt16s(key string, v []int16)
	SetInt32s(key string, v []int32)
	SetInt64s(key string, v []int64)
	SetUints(key string, v []uint)
	SetUint8s(key string, v []uint8)
	SetUint16s(key string, v []uint16)
	SetUint32s(key string, v []uint32)
	SetUint64s(key string, v []uint64)
	SetBytes(key string, v []byte)
	SetBools(key string, v []bool)
	SetStrings(key string, v []string)

	// Flush cache data to bcchain
	Flush()

	// Delete data map by key
	Delete(key string)

	//Memory cache McGet
	McGet(key string, defaultValue interface{}) interface{}   //从状态数据库或内存缓存中读取指定KEY对应的数据，不存在返回空
	McGetEx(key string, defaultValue interface{}) interface{} //从状态数据库或内存缓存中读取指定KEY对应的数据，不存在返回默认值

	//Memory cache McGetXXX
	McGetInt(key string) int
	McGetInt8(key string) int8
	McGetInt16(key string) int16
	McGetInt32(key string) int32
	McGetInt64(key string) int64
	McGetUint(key string) uint
	McGetUint8(key string) uint8
	McGetUint16(key string) uint16
	McGetUint32(key string) uint32
	McGetUint64(key string) uint64
	McGetByte(key string) byte
	McGetBool(key string) bool
	McGetString(key string) string

	//Memory cache McGetXXXs
	McGetInts(key string) []int
	McGetInt8s(key string) []int8
	McGetInt16s(key string) []int16
	McGetInt32s(key string) []int32
	McGetInt64s(key string) []int64
	McGetUints(key string) []uint
	McGetUint8s(key string) []uint8
	McGetUint16s(key string) []uint16
	McGetUint32s(key string) []uint32
	McGetUint64s(key string) []uint64
	McGetBytes(key string) []byte
	McGetBools(key string) []bool
	McGetStrings(key string) []string

	//Memory cache McSet
	McSet(key string, value interface{}) //向状态数据库和内存缓存设置指定KEY对应的数据

	//Memory cache McSetXXX
	McSetInt(key string, v int)
	McSetInt8(key string, v int8)
	McSetInt16(key string, v int16)
	McSetInt32(key string, v int32)
	McSetInt64(key string, v int64)
	McSetUint(key string, v uint)
	McSetUint8(key string, v uint8)
	McSetUint16(key string, v uint16)
	McSetUint32(key string, v uint32)
	McSetUint64(key string, v uint64)
	McSetByte(key string, v byte)
	McSetBool(key string, v bool)
	McSetString(key string, v string)

	//Memory cache McSetXXXs
	McSetInts(key string, v []int)
	McSetInt8s(key string, v []int8)
	McSetInt16s(key string, v []int16)
	McSetInt32s(key string, v []int32)
	McSetInt64s(key string, v []int64)
	McSetUints(key string, v []uint)
	McSetUint8s(key string, v []uint8)
	McSetUint16s(key string, v []uint16)
	McSetUint32s(key string, v []uint32)
	McSetUint64s(key string, v []uint64)
	McSetBytes(key string, v []byte)
	McSetBools(key string, v []bool)
	McSetStrings(key string, v []string)

	// McClear dirty Memory cache data map by key
	McClear(key string)

	// McDelete dirty and delete data map by key
	McDelete(key string)
}

// IIBCHelper the interface for IBCHelper
type IIBCHelper interface {
	IbcHash(toChainID string) types.Hash
	Run(f func()) IIBCHelper

	Register(toChainID string)
	Notify(toChainIDs []string)
	Broadcast()

	CalcBlockHash(h *ibc.Header) types.Hash
	CalcQueueHash(packet ibc.Packet, lastQueueHash types.Hash) types.Hash
	VerifyPrecommit(pubKey types.PubKey, precommit ibc.Precommit, chainID string, height int64) bool
}

// IIBCStubHelper the interface for IBCStubHelper
type IIBCStubHelper interface {
	Recast(ibcHash types.HexBytes, orgID, contractName string, receipts []types.KVPair) (bool, []types.KVPair, types.Error)
	Confirm(ibcHash types.HexBytes, orgID, contractName string, receipts []types.KVPair) ([]types.KVPair, types.Error)
	Cancel(ibcHash types.HexBytes, orgID, contractName string, receipts []types.KVPair) ([]types.KVPair, types.Error)
	TryRecast(ibcHash types.HexBytes, orgID, contractName string, receipts []types.KVPair) (bool, []types.KVPair, types.Error)
	ConfirmRecast(ibcHash types.HexBytes, orgID, contractName string, receipts []types.KVPair) ([]types.KVPair, types.Error)
	CancelRecast(ibcHash types.HexBytes, orgID, contractName string, receipts []types.KVPair) ([]types.KVPair, types.Error)
	Notify(ibcHash types.HexBytes, orgID, contractName string, receipts []types.KVPair) ([]types.KVPair, types.Error)
}

// ISmartContract the interface for SmartContract
type ISmartContract interface {
	Block() IBlock     // 返回区块信息对象
	Tx() ITx           // 返回交易信息对象
	Message() IMessage // 返回消息对象
	Helper() IHelper   // 返回当前可用的Helper对象
}
