package mc

import (
	"sync"
)

//Instance an Instance of memory cache
type Instance struct {
	llmt    sync.RWMutex
	llCache map[string][]byte //一级缓存，用key:value格式存储key对应的结构体数据

	mcmt    sync.RWMutex
	mcCache map[int64]map[string]*MemCache //二级缓存，保存transId对应的数据缓存
}

//MemCache an object
type MemCache struct {
	sync.RWMutex
	transID int64
	key     string
	tx      []txCache //三级缓存，保存txId对应的数据缓存
}

type txCache struct {
	id    int64
	value []byte
}

//NewMcInstance new a global Instance
func NewMcInstance() *Instance {
	mi := Instance{}
	mi.llCache = make(map[string][]byte)
	mi.mcCache = make(map[int64]map[string]*MemCache)

	return &mi
}

//NewMc new a MemCache for key
func (mi *Instance) NewMc(transID int64, key string) *MemCache {

	mi.mcmt.Lock()
	defer mi.mcmt.Unlock()
	if _, ok := mi.mcCache[transID]; !ok {
		mi.mcCache[transID] = make(map[string]*MemCache)
	}

	if _, ok := mi.mcCache[transID][key]; !ok {
		mi.mcCache[transID][key] = newmc(transID, key, mi)
	}

	return mi.mcCache[transID][key]
}

//Dirty dirty all cache of key
func (mi *Instance) Dirty(key string) {

	mi.mcmt.Lock()
	for transid, mcs := range mi.mcCache {
		for _, mc := range mcs {
			if mc.key == key {
				delete(mi.mcCache[transid], key)
			}
		}
	}
	mi.mcmt.Unlock()

	// delete "key" from llcache
	mi.llmt.Lock()
	if _, ok := mi.llCache[key]; ok {
		delete(mi.llCache, key)
	}
	mi.llmt.Unlock()
}

func (mi *Instance) Clear() {
	mi.llCache = make(map[string][]byte)
	mi.mcCache = make(map[int64]map[string]*MemCache)
}

//CommitTrans commit transaction data
func (mi *Instance) CommitTrans(transID int64) {
	mi.mcmt.Lock()
	if mcs, ok := mi.mcCache[transID]; ok {
		mi.llmt.Lock()
		for _, mc := range mcs {
			mi.llCache[mc.key] = mc.top()
		}
		mi.llmt.Unlock()

		//delete transid map
		delete(mi.mcCache, transID)
	}
	mi.mcmt.Unlock()
}

//DirtyTrans dirty transaction data
func (mi *Instance) DirtyTrans(transID int64) {
	mi.mcmt.Lock()
	delete(mi.mcCache, transID)
	mi.mcmt.Unlock()
}

//DirtyTransTx dirty tx data of transaction
func (mi *Instance) DirtyTransTx(transID, txID int64) {
	mi.mcmt.RLock()
	for _, mc := range mi.mcCache[transID] {
		mc.Dirty(txID)
	}
	mi.mcmt.RUnlock()
}

//Get get cached top value
func (mc *MemCache) Get() []byte {
	return mc.top()
}

// Set only push the data into txCache, will be set to llCache when commit
func (mc *MemCache) Set(txID int64, data []byte) {

	mc.push(txID, data)
}

//Dirty dirty tx data
func (mc *MemCache) Dirty(txID int64) {
	// there might be multiple txCache which id = txID, delete all of them
	mc.Lock()
	defer mc.Unlock()
	if i := mc.lastTx(); i >= 0 {
		if mc.tx[i].id == txID {
			mc.tx = append(mc.tx[:i])
		}
	}
}

func newmc(transid int64, key string, mi *Instance) *MemCache {
	mc := MemCache{transID: transid, key: key}
	mc.tx = make([]txCache, 0)
	// read and push cache to txCache if it has, and set txID to 0(zero) as default.
	mi.llmt.RLock()
	if v, ok := mi.llCache[key]; ok {
		mc.tx = append(mc.tx, txCache{0, v})
	}
	mi.llmt.RUnlock()
	return &mc
}

func (mc *MemCache) top() []byte {
	mc.RLock()
	defer mc.RUnlock()
	if mc.lastTx() >= 0 {
		return mc.tx[mc.lastTx()].value
	}
	return nil
}

func (mc *MemCache) push(txid int64, data []byte) {
	mc.Lock()
	defer mc.Unlock()
	//if the last txCache is for txID, cover it
	if len(mc.tx) > 0 && mc.tx[mc.lastTx()].id == txid {
		mc.tx[mc.lastTx()].value = data
		return
	}
	// push
	mc.tx = append(mc.tx, txCache{txid, data})
}

// Delete the latest txCache from slice
func (mc *MemCache) pop() interface{} {

	v := mc.top()
	if v != nil {
		mc.Lock()
		defer mc.Unlock()
		mc.tx = mc.tx[:mc.lastTx()]
	}
	return v
}

// check to see if txCache is empty
func (mc *MemCache) empty() bool {
	return mc.tx == nil || mc.lastTx() < 0
}

func (mc *MemCache) lastTx() int64 {
	return int64(len(mc.tx) - 1)
}
