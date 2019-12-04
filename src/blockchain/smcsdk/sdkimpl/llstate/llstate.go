package llstate

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/jsoniter"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdkimpl"
	"blockchain/smcsdk/sdkimpl/llfunction"
	"blockchain/types"
)

// LowLevelSDB lowLevelSDB information
type LowLevelSDB struct {
	smc     sdk.ISmartContract // 指向智能合约API对象指针
	transID int64              // 事务ID
	txID    int64              // 子事务ID
	cache   map[string][]byte  // 数据缓存
}

var _ sdkimpl.ILowLevelSDB = (*LowLevelSDB)(nil)
var _ sdkimpl.IAcquireSMC = (*LowLevelSDB)(nil)

// SMC get smart contract object
func (ll *LowLevelSDB) SMC() sdk.ISmartContract { return ll.smc }

// SetSMC set smart contract object
func (ll *LowLevelSDB) SetSMC(smc sdk.ISmartContract) { ll.smc = smc }

var (
	sdbGet llfunction.GetCallback // 获取数据回调接口
	sdbSet llfunction.SetCallback // 设置数据回调接口
)

// Init initial LowLevelSDB callback function
func Init(setFunc llfunction.SetCallback, getFunc llfunction.GetCallback) {
	sdbSet = setFunc
	sdbGet = getFunc
}

// Init initial LowLevelSDB property of transID and txID
func (ll *LowLevelSDB) Init(transID, txID int64) {
	ll.transID = transID
	ll.txID = txID
}

// TransID get value of transID
func (ll *LowLevelSDB) TransID() int64 {
	return ll.transID
}

// TxID get value of txID
func (ll *LowLevelSDB) TxID() int64 {
	return ll.txID
}

// Get get the object in db map by key, and then return nil if it not exist
func (ll *LowLevelSDB) Get(key string, defaultValue interface{}) interface{} {
	resBytes, ok := ll.cache[key]
	if ok == false {
		resBytes = sdbGet(ll.transID, ll.txID, key)
		resBytes = ll.data(key, resBytes)
		if resBytes == nil || len(resBytes) == 0 {
			sdkimpl.Logger.Debugf("[sdk][transID=%d][txID=%d] Cannot find key=%s in stateDB", ll.transID, ll.txID, key)
			return nil
		}
	}

	if resBytes == nil {
		sdkimpl.Logger.Debugf("[sdk][transID=%d][txID=%d] Cannot find key=%s in stateDB", ll.transID, ll.txID, key)
		return nil
	}

	err := jsoniter.Unmarshal(resBytes, defaultValue)
	if err != nil {
		if key == "/genesis/chainid" {
			temp := string(resBytes)
			defaultValue = &temp
		} else {
			sdkimpl.Logger.Fatalf("[sdk][transID=%d][txID=%d] Cannot unmarshal from key=%s in stateDB, error=%v\nbytes=%v", ll.transID, ll.txID, key, err, resBytes)
			sdkimpl.Logger.Flush()
			panic(err)
		}
	}

	sdkimpl.Logger.Tracef("[sdk][transID=%d][txID=%d] Get key=%s from stateDB, value=%v", ll.transID, ll.txID, key, defaultValue)
	return defaultValue
}

// GetEx get the object in db map by key, and then return defaultData if it not exist
func (ll *LowLevelSDB) GetEx(key string, defaultValue interface{}) interface{} {
	getData := ll.Get(key, defaultValue)
	if getData == nil {
		return defaultValue
	}

	return getData
}

// GetInt64 get the object in db that map by key, and then return defaultData if it not exist
func (ll *LowLevelSDB) GetInt64(key string) int64 {
	return *ll.GetEx(key, new(int64)).(*int64)
}

// GetStrings get the object in db that map by key, and then return defaultData if it not exist
func (ll *LowLevelSDB) GetStrings(key string) []string {
	return *ll.GetEx(key, new([]string)).(*[]string)
}

// Set set value to db that map by key
func (ll *LowLevelSDB) Set(key string, value interface{}) {
	resBytes, mErr := jsoniter.Marshal(value)
	if mErr != nil {
		sdkimpl.Logger.Fatalf("[sdk][transID=%d][txID=%d]Cannot marshal data error=%v, \nvalue=%v", ll.transID, ll.txID, mErr, value)
		sdkimpl.Logger.Flush()
		panic(mErr)
	}

	// cache
	ll.cache[key] = resBytes

	sdkimpl.Logger.Tracef("[sdk][transID=%d][txID=%d] Set key=%s to stateDB, value=%v", ll.transID, ll.txID, key, value)
}

// McGet get the object in db map by key, and then return defaultData if it not exist
func (ll *LowLevelSDB) McGet(key string, defaultValue interface{}) interface{} {
	mc := sdkimpl.McInst.NewMc(ll.transID, key)
	if result := mc.Get(); result != nil {
		err := jsoniter.Unmarshal(result, defaultValue)
		if err != nil {
			sdkimpl.Logger.Fatalf("[sdk][transID=%d][txID=%d] Cannot unmarshal from key=%s in mc, error=%v\nbytes=%v", ll.transID, ll.txID, key, err, result)
			panic(err)
		}
		sdkimpl.Logger.Tracef("[sdk][transID=%d][txID=%d] Get key=%s from memory cache, value=%v", ll.transID, ll.txID, key, string(result))
		return defaultValue
	}

	if result := ll.Get(key, defaultValue); result != nil {
		sdkimpl.Logger.Tracef("[sdk][transID=%d][txID=%d] Get key=%s from stateDB, value=%v", ll.transID, ll.txID, key, result)

		value, err := jsoniter.Marshal(result)
		if err != nil {
			sdkimpl.Logger.Fatalf("[sdk][transID=%d][txID=%d] Cannot marshal set value struct, key=%s, error=%v", ll.transID, ll.txID, key, err)
			panic(err)
		}
		mc.Set(ll.txID, value)
		return result
	}
	sdkimpl.Logger.Tracef("[sdk][transID=%d][txID=%d] Get key=%s failed", ll.transID, ll.txID, key)

	return nil
}

// McGetEx get the object in db map by key, and then return defaultData if it not exist
func (ll *LowLevelSDB) McGetEx(key string, defaultValue interface{}) interface{} {
	getData := ll.McGet(key, defaultValue)
	if getData == nil {
		return defaultValue
	}

	return getData
}

// McSet set value to db that map by key
func (ll *LowLevelSDB) McSet(key string, value interface{}) {
	mc := sdkimpl.McInst.NewMc(ll.transID, key)

	ll.Set(key, value)

	valueByte, err := jsoniter.Marshal(value)
	if err != nil {
		sdkimpl.Logger.Fatalf("[sdk][transID=%d][txID=%d] Cannot marshal set value struct, key=%s, error=%v", ll.transID, ll.txID, key, err)
		panic(err)
	}

	mc.Set(ll.txID, valueByte)
	sdkimpl.Logger.Trace("Set Memory Cache", "transID", ll.transID, "txID", ll.txID, "key", key, "value", string(valueByte))
}

// Commit commit all set data
func (ll *LowLevelSDB) Commit() {
	sdbSet(ll.transID, ll.txID, ll.cache)
}

// Flush flush cache data to database
func (ll *LowLevelSDB) Flush() {
	sdbSet(ll.transID, ll.txID, ll.cache)
}

// Delete delete data map by key
func (ll *LowLevelSDB) Delete(key string) {
	ll.cache[key] = nil
}

// GetCache return cache data
func (ll *LowLevelSDB) GetCache() map[string][]byte {
	tmpCache := make(map[string][]byte)

	for key, value := range ll.cache {
		tmpCache[key] = value
	}

	return tmpCache
}

// GetCache return cache data
func (ll *LowLevelSDB) SetCache(cache map[string][]byte) {
	ll.cache = cache
}

// data input std.GetResult data, then return value if ok or nil
func (ll *LowLevelSDB) data(key string, resBytes []byte) []byte {
	var getResult std.GetResult
	err := jsoniter.Unmarshal(resBytes, &getResult)
	if err != nil {
		sdkimpl.Logger.Fatalf("[sdk][transID=%d][txID=%d] Cannot unmarshal get result struct, key=%s, error=%v\nbytes=%v", ll.transID, ll.txID, key, err, resBytes)
		sdkimpl.Logger.Flush()
		panic(err)
	} else if getResult.Code != types.CodeOK {
		sdkimpl.Logger.Debugf("[sdk][transID=%d][txID=%d] Cannot find key=%s in stateDB, error=%s", ll.transID, ll.txID, getResult.Msg)
		return nil
	}

	return getResult.Data
}
