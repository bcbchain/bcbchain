package sdkimpl

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdkimpl/llfunction"
	"common/mc"
	"github.com/tendermint/tmlibs/log"
	"sync"
)

var (
	syncOnce sync.Once
)

// SmartContract smart contract information
type SmartContract struct {
	smc *SmartContract //自身的对象指针

	block   sdk.IBlock   //当前区块信息
	message sdk.IMessage //当前消息信息
	tx      sdk.ITx      //当前交易调用信息

	helper  sdk.IHelper //当前可用的Helper对象
	llState ILowLevelSDB
}

// IAcquireSMC Acquire interface information
type IAcquireSMC interface {
	SMC() sdk.ISmartContract       //获取智能合约smc对象指针
	SetSMC(smc sdk.ISmartContract) //设置智能合约smc对象指针
}

// ISDB sdb interface information
type ISDB interface {
	Commit() // 提交sdk缓存的更新数据
}

// ILowLevelSDB lowLevelSDB interface information
type ILowLevelSDB interface {
	TransID() int64                                           // 块事务ID
	TxID() int64                                              // 交易事务ID
	Get(key string, defaultValue interface{}) interface{}     // 根据key从数据库读取数据，不存在则返回nil
	GetEx(key string, defaultValue interface{}) interface{}   // 根据key从数据库读取数据，不存在则返回默认值
	GetInt64(key string) int64                                // 根据key从缓存读取int64数据，不存在则返回默认值
	GetStrings(key string) []string                           // 根据key从缓存读取[]string数据，不存在则返回默认值
	Set(key string, value interface{})                        // 保存数据
	McGet(key string, defaultValue interface{}) interface{}   // 根据key从缓存读取数据，不存在则返回nil
	McGetEx(key string, defaultValue interface{}) interface{} // 根据key从缓存读取数据，不存在则返回默认值
	McSet(key string, value interface{})                      // 保存数据到缓存和数据库
	Commit()                                                  // 提交sdk缓存的更新数据
	Flush()                                                   // 刷新sdk缓存数据到数据库
	Delete(key string)                                        // 删除指定的键值

	GetCache() map[string][]byte      // 返回缓存内容
	SetCache(cache map[string][]byte) // 设置缓存内容
}

var _ sdk.ISmartContract = (*SmartContract)(nil)
var _ ISDB = (*SmartContract)(nil)

var (
	// McInst instance of MCache
	McInst *mc.Instance

	// Logger object of log
	Logger log.Loggerf

	// TransferFuncMap transfer call back function
	TransferFunc llfunction.TransferCallBack

	// BuildFunc build call back function
	BuildFunc llfunction.BuildCallBack

	// GetBlockFunc get block data function
	GetBlockFunc llfunction.GetBlockCallBack

	// GetBlockFunc get block data function
	IBCInvokeFunc llfunction.IBCInvoke
)

// Init initial method for smc init
func Init(
	transferFunc llfunction.TransferCallBack,
	buildFunc llfunction.BuildCallBack,
	getBlockFunc llfunction.GetBlockCallBack,
	ibcInvokeFunc llfunction.IBCInvoke,
	loggerF *log.Loggerf) {

	syncOnce.Do(func() {
		GetBlockFunc = getBlockFunc
		TransferFunc = transferFunc
		BuildFunc = buildFunc
		IBCInvokeFunc = ibcInvokeFunc

		Logger = *loggerF
		McInst = mc.NewMcInstance()
	})
}

// Block get block object
func (smc *SmartContract) Block() sdk.IBlock { return smc.block }

// Message get message object
func (smc *SmartContract) Message() sdk.IMessage { return smc.message }

// Tx get tx object
func (smc *SmartContract) Tx() sdk.ITx { return smc.tx }

// Helper get hHelper object
func (smc *SmartContract) Helper() sdk.IHelper { return smc.helper }

// LlState get llState object
func (smc *SmartContract) LlState() ILowLevelSDB { return smc.llState }

// Commit invoke lowLevelSDB commit method
func (smc *SmartContract) Commit() { smc.llState.Commit() }

// Flush invoke lowLevelSDB flush method
func (smc *SmartContract) Flush() { smc.llState.Commit() }

// SetHelper set helper object
func (smc *SmartContract) SetHelper(v sdk.IHelper) { smc.helper = v }

// SetLlState set llState object
func (smc *SmartContract) SetLlState(v ILowLevelSDB) { smc.llState = v }

// SetBlock set block object
func (smc *SmartContract) SetBlock(v sdk.IBlock) { smc.block = v }

// SetMessage set message object
func (smc *SmartContract) SetMessage(v sdk.IMessage) { smc.message = v }

// SetTx set tx object
func (smc *SmartContract) SetTx(v sdk.ITx) { smc.tx = v }
