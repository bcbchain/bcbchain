package helper

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdkimpl"
)

// StateHelper state helper information
type StateHelper struct {
	smc sdk.ISmartContract //指向智能合约API对象指针
}

var _ sdk.IStateHelper = (*StateHelper)(nil)
var _ sdkimpl.IAcquireSMC = (*StateHelper)(nil)

// SMC get smart contract object
func (sh *StateHelper) SMC() sdk.ISmartContract { return sh.smc }

// SetSMC set smart contract object
func (sh *StateHelper) SetSMC(smc sdk.ISmartContract) { sh.smc = smc }

// Check check the db ever save object or not that map by key
func (sh *StateHelper) Check(key string) bool {
	if sh.Get(key, new(interface{})) == nil {
		return false
	}

	return true
}

// Get get value in db map by key, and then return nil if it's not exist
func (sh *StateHelper) Get(key string, defaultValue interface{}) interface{} {
	if !sh.checkKey(key) {
		return nil
	}

	fullKey := sh.smc.Message().Contract().KeyPrefix() + key
	return sh.smc.(*sdkimpl.SmartContract).LlState().Get(fullKey, defaultValue)
}

// GetEx get value in db map by key, and then return default if it's not exist
func (sh *StateHelper) GetEx(key string, defaultValue interface{}) interface{} {
	resp := sh.Get(key, defaultValue)
	if resp == nil {
		return defaultValue
	}

	return resp
}

// Set set value to db that map by key
func (sh *StateHelper) Set(key string, value interface{}) {
	if !sh.checkKey(key) {
		return
	}

	fullKey := sh.smc.Message().Contract().KeyPrefix() + key
	sh.smc.(*sdkimpl.SmartContract).LlState().Set(fullKey, value)
}

// Flush flush data in cache to bcchain
func (sh *StateHelper) Flush() {
	sh.SMC().(*sdkimpl.SmartContract).LlState().Flush()
}

// Delete data map by key
func (sh *StateHelper) Delete(key string) {
	if !sh.checkKey(key) {
		return
	}

	fullKey := sh.smc.Message().Contract().KeyPrefix() + key
	sh.smc.(*sdkimpl.SmartContract).LlState().Delete(fullKey)
}

// GetInt get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetInt(key string) int {
	return *sh.GetEx(key, new(int)).(*int)
}

// GetInt8 get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetInt8(key string) int8 {
	return *sh.GetEx(key, new(int8)).(*int8)
}

// GetInt16 get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetInt16(key string) int16 {
	return *sh.GetEx(key, new(int16)).(*int16)
}

// GetInt32 get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetInt32(key string) int32 {
	return *sh.GetEx(key, new(int32)).(*int32)
}

// GetInt64 get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetInt64(key string) int64 {
	return *sh.GetEx(key, new(int64)).(*int64)
}

// GetUint get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetUint(key string) uint {
	return *sh.GetEx(key, new(uint)).(*uint)
}

// GetUint8 get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetUint8(key string) uint8 {
	return *sh.GetEx(key, new(uint8)).(*uint8)
}

// GetUint16 get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetUint16(key string) uint16 {
	return *sh.GetEx(key, new(uint16)).(*uint16)
}

// GetUint32 get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetUint32(key string) uint32 {
	return *sh.GetEx(key, new(uint32)).(*uint32)
}

// GetUint64 get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetUint64(key string) uint64 {
	return *sh.GetEx(key, new(uint64)).(*uint64)
}

// GetByte get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetByte(key string) byte {
	return *sh.GetEx(key, new(byte)).(*byte)
}

// GetBool get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetBool(key string) bool {
	return *sh.GetEx(key, new(bool)).(*bool)
}

// GetString get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetString(key string) string {
	return *sh.GetEx(key, new(string)).(*string)
}

// GetInts get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetInts(key string) []int {
	return *sh.GetEx(key, new([]int)).(*[]int)
}

// GetInt8s get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetInt8s(key string) []int8 {
	return *sh.GetEx(key, new([]int8)).(*[]int8)
}

// GetInt16s get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetInt16s(key string) []int16 {
	return *sh.GetEx(key, new([]int16)).(*[]int16)
}

// GetInt32s get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetInt32s(key string) []int32 {
	return *sh.GetEx(key, new([]int32)).(*[]int32)
}

// GetInt64s get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetInt64s(key string) []int64 {
	return *sh.GetEx(key, new([]int64)).(*[]int64)
}

// GetUints get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetUints(key string) []uint {
	return *sh.GetEx(key, new([]uint)).(*[]uint)
}

// GetUint8s get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetUint8s(key string) []uint8 {
	return *sh.GetEx(key, new([]uint8)).(*[]uint8)
}

// GetUint16s get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetUint16s(key string) []uint16 {
	return *sh.GetEx(key, new([]uint16)).(*[]uint16)
}

// GetUint32s get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetUint32s(key string) []uint32 {
	return *sh.GetEx(key, new(uint32)).(*[]uint32)
}

// GetUint64s get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetUint64s(key string) []uint64 {
	return *sh.GetEx(key, new([]uint64)).(*[]uint64)
}

// GetBytes get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetBytes(key string) []byte {
	return *sh.GetEx(key, new([]byte)).(*[]byte)
}

// GetBools get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetBools(key string) []bool {
	return *sh.GetEx(key, new([]bool)).(*[]bool)
}

// GetStrings get value in db map by key, and then return default if it not exist
func (sh *StateHelper) GetStrings(key string) []string {
	return *sh.GetEx(key, new([]string)).(*[]string)
}

// SetInt set value to db that map by key
func (sh *StateHelper) SetInt(key string, v int) {
	sh.Set(key, &v)
}

// SetInt8 set value to db that map by key
func (sh *StateHelper) SetInt8(key string, v int8) {
	sh.Set(key, &v)
}

// SetInt16 set value to db that map by key
func (sh *StateHelper) SetInt16(key string, v int16) {
	sh.Set(key, &v)
}

// SetInt32 set value to db that map by key
func (sh *StateHelper) SetInt32(key string, v int32) {
	sh.Set(key, &v)
}

// SetInt64 set value to db that map by key
func (sh *StateHelper) SetInt64(key string, v int64) {
	sh.Set(key, &v)
}

// SetUint8 set value to db that map by key
func (sh *StateHelper) SetUint8(key string, v uint8) {
	sh.Set(key, &v)
}

// SetUint set value to db that map by key
func (sh *StateHelper) SetUint(key string, v uint) {
	sh.Set(key, &v)
}

// SetUint16 set value to db that map by key
func (sh *StateHelper) SetUint16(key string, v uint16) {
	sh.Set(key, &v)
}

// SetUint32 set value to db that map by key
func (sh *StateHelper) SetUint32(key string, v uint32) {
	sh.Set(key, &v)
}

// SetUint64 set value to db that map by key
func (sh *StateHelper) SetUint64(key string, v uint64) {
	sh.Set(key, &v)
}

// SetByte set value to db that map by key
func (sh *StateHelper) SetByte(key string, v byte) {
	sh.Set(key, &v)
}

// SetBool set value to db that map by key
func (sh *StateHelper) SetBool(key string, v bool) {
	sh.Set(key, &v)
}

// SetString set value to db that map by key
func (sh *StateHelper) SetString(key string, v string) {
	sh.Set(key, &v)
}

// SetInts set value to db that map by key
func (sh *StateHelper) SetInts(key string, v []int) {
	sh.Set(key, &v)
}

// SetInt8s set value to db that map by key
func (sh *StateHelper) SetInt8s(key string, v []int8) {
	sh.Set(key, &v)
}

// SetInt16s set value to db that map by key
func (sh *StateHelper) SetInt16s(key string, v []int16) {
	sh.Set(key, &v)
}

// SetInt32s set value to db that map by key
func (sh *StateHelper) SetInt32s(key string, v []int32) {
	sh.Set(key, &v)
}

// SetInt64s set value to db that map by key
func (sh *StateHelper) SetInt64s(key string, v []int64) {
	sh.Set(key, &v)
}

// SetUints set value to db that map by key
func (sh *StateHelper) SetUints(key string, v []uint) {
	sh.Set(key, &v)
}

// SetUint8s set value to db that map by key
func (sh *StateHelper) SetUint8s(key string, v []uint8) {
	sh.Set(key, &v)
}

// SetUint16s set value to db that map by key
func (sh *StateHelper) SetUint16s(key string, v []uint16) {
	sh.Set(key, &v)
}

// SetUint32s set value to db that map by key
func (sh *StateHelper) SetUint32s(key string, v []uint32) {
	sh.Set(key, &v)
}

// SetUint64s set value to db that map by key
func (sh *StateHelper) SetUint64s(key string, v []uint64) {
	sh.Set(key, &v)
}

// SetBytes set value to db that map by key
func (sh *StateHelper) SetBytes(key string, v []byte) {
	sh.Set(key, &v)
}

// SetBools set value to db that map by key
func (sh *StateHelper) SetBools(key string, v []bool) {
	sh.Set(key, &v)
}

// SetStrings set value to db that map by key
func (sh *StateHelper) SetStrings(key string, v []string) {
	sh.Set(key, &v)
}

// McCheck check the McCache or db ever save object or not that map by key
func (sh *StateHelper) McCheck(key string) bool {
	if sh.McGet(key, new(interface{})) == nil {
		return false
	}

	return true
}

// McGet get value in McCache or db that map by key, and then return nil if it not exist
func (sh *StateHelper) McGet(key string, defaultValue interface{}) interface{} {
	if !sh.checkKey(key) {
		return nil
	}

	// 从缓存读取
	fullKey := sh.smc.Message().Contract().KeyPrefix() + key
	return sh.smc.(*sdkimpl.SmartContract).LlState().McGet(fullKey, defaultValue)
}

// McGetEx get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetEx(key string, defaultValue interface{}) interface{} {
	resp := sh.McGet(key, defaultValue)
	if resp == nil {
		return defaultValue
	}

	return resp
}

// McSet set value to McCache and db that map by key
func (sh *StateHelper) McSet(key string, value interface{}) {
	if !sh.checkKey(key) {
		return
	}

	fullKey := sh.smc.Message().Contract().KeyPrefix() + key
	sh.smc.(*sdkimpl.SmartContract).LlState().McSet(fullKey, value)
}

// McGetInt get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetInt(key string) int {
	return *sh.McGetEx(key, new(int)).(*int)
}

// McGetInt8 get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetInt8(key string) int8 {
	return *sh.McGetEx(key, new(int8)).(*int8)
}

// McGetInt16 get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetInt16(key string) int16 {
	return *sh.McGetEx(key, new(int16)).(*int16)
}

// McGetInt32 get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetInt32(key string) int32 {
	return *sh.McGetEx(key, new(int32)).(*int32)
}

// McGetInt64 get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetInt64(key string) int64 {
	return *sh.McGetEx(key, new(int64)).(*int64)
}

// McGetUint get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetUint(key string) uint {
	return *sh.McGetEx(key, new(uint)).(*uint)
}

// McGetUint8 get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetUint8(key string) uint8 {
	return *sh.McGetEx(key, new(uint8)).(*uint8)
}

// McGetUint16 get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetUint16(key string) uint16 {
	return *sh.McGetEx(key, new(uint16)).(*uint16)
}

// McGetUint32 get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetUint32(key string) uint32 {
	return *sh.McGetEx(key, new(uint32)).(*uint32)
}

// McGetUint64 get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetUint64(key string) uint64 {
	return *sh.McGetEx(key, new(uint64)).(*uint64)
}

// McGetByte get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetByte(key string) byte {
	return *sh.McGetEx(key, new(byte)).(*byte)
}

// McGetBool get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetBool(key string) bool {
	return *sh.McGetEx(key, new(bool)).(*bool)
}

// McGetString get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetString(key string) string {
	return *sh.McGetEx(key, new(string)).(*string)
}

// McGetInts get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetInts(key string) []int {
	return *sh.McGetEx(key, new([]int)).(*[]int)
}

// McGetInt8s get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetInt8s(key string) []int8 {
	return *sh.McGetEx(key, new([]int8)).(*[]int8)
}

// McGetInt16s get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetInt16s(key string) []int16 {
	return *sh.McGetEx(key, new([]int16)).(*[]int16)
}

// McGetInt32s get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetInt32s(key string) []int32 {
	return *sh.McGetEx(key, new([]int32)).(*[]int32)
}

// McGetInt64s get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetInt64s(key string) []int64 {
	return *sh.McGetEx(key, new([]int64)).(*[]int64)
}

// McGetUints get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetUints(key string) []uint {
	return *sh.McGetEx(key, new([]uint)).(*[]uint)
}

// McGetUint8s get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetUint8s(key string) []uint8 {
	return *sh.McGetEx(key, new([]uint8)).(*[]uint8)
}

// McGetUint16s get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetUint16s(key string) []uint16 {
	return *sh.McGetEx(key, new([]uint16)).(*[]uint16)
}

// McGetUint32s get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetUint32s(key string) []uint32 {
	return *sh.McGetEx(key, new([]uint32)).(*[]uint32)
}

// McGetUint64s get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetUint64s(key string) []uint64 {
	return *sh.McGetEx(key, new([]uint64)).(*[]uint64)
}

// McGetBytes get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetBytes(key string) []byte {
	return *sh.McGetEx(key, new([]byte)).(*[]byte)
}

// McGetBools get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetBools(key string) []bool {
	return *sh.McGetEx(key, new([]bool)).(*[]bool)
}

// McGetStrings get value in McCache or db that map by key, and then return defaultData if it not exist
func (sh *StateHelper) McGetStrings(key string) []string {
	return *sh.McGetEx(key, new([]string)).(*[]string)
}

// McSetInt set value to McCache and db that map by key
func (sh *StateHelper) McSetInt(key string, v int) {
	sh.McSet(key, &v)
}

// McSetInt8 set value to McCache and db that map by key
func (sh *StateHelper) McSetInt8(key string, v int8) {
	sh.McSet(key, &v)
}

// McSetInt16 set value to McCache and db that map by key
func (sh *StateHelper) McSetInt16(key string, v int16) {
	sh.McSet(key, &v)
}

// McSetInt32 set value to McCache and db that map by key
func (sh *StateHelper) McSetInt32(key string, v int32) {
	sh.McSet(key, &v)
}

// McSetInt64 set value to McCache and db that map by key
func (sh *StateHelper) McSetInt64(key string, v int64) {
	sh.McSet(key, &v)
}

// McSetUint set value to McCache and db that map by key
func (sh *StateHelper) McSetUint(key string, v uint) {
	sh.McSet(key, &v)
}

// McSetUint8 set value to McCache and db that map by key
func (sh *StateHelper) McSetUint8(key string, v uint8) {
	sh.McSet(key, &v)
}

// McSetUint16 set value to McCache and db that map by key
func (sh *StateHelper) McSetUint16(key string, v uint16) {
	sh.McSet(key, &v)
}

// McSetUint32 set value to McCache and db that map by key
func (sh *StateHelper) McSetUint32(key string, v uint32) {
	sh.McSet(key, &v)
}

// McSetUint64 set value to McCache and db that map by key
func (sh *StateHelper) McSetUint64(key string, v uint64) {
	sh.McSet(key, &v)
}

// McSetByte set value to McCache and db that map by key
func (sh *StateHelper) McSetByte(key string, v byte) {
	sh.McSet(key, &v)
}

// McSetBool set value to McCache and db that map by key
func (sh *StateHelper) McSetBool(key string, v bool) {
	sh.McSet(key, &v)
}

// McSetString set value to McCache and db that map by key
func (sh *StateHelper) McSetString(key string, v string) {
	sh.McSet(key, &v)
}

// McSetInts set value to McCache and db that map by key
func (sh *StateHelper) McSetInts(key string, v []int) {
	sh.McSet(key, &v)
}

// McSetInt8s set value to McCache and db that map by key
func (sh *StateHelper) McSetInt8s(key string, v []int8) {
	sh.McSet(key, &v)
}

// McSetInt16s set value to McCache and db that map by key
func (sh *StateHelper) McSetInt16s(key string, v []int16) {
	sh.McSet(key, &v)
}

// McSetInt32s set value to McCache and db that map by key
func (sh *StateHelper) McSetInt32s(key string, v []int32) {
	sh.McSet(key, &v)
}

// McSetInt64s set value to McCache and db that map by key
func (sh *StateHelper) McSetInt64s(key string, v []int64) {
	sh.McSet(key, &v)
}

// McSetUints set value to McCache and db that map by key
func (sh *StateHelper) McSetUints(key string, v []uint) {
	sh.McSet(key, &v)
}

// McSetUint8s set value to McCache and db that map by key
func (sh *StateHelper) McSetUint8s(key string, v []uint8) {
	sh.McSet(key, &v)
}

// McSetUint16s set value to McCache and db that map by key
func (sh *StateHelper) McSetUint16s(key string, v []uint16) {
	sh.McSet(key, &v)
}

// McSetUint32s set value to McCache and db that map by key
func (sh *StateHelper) McSetUint32s(key string, v []uint32) {
	sh.McSet(key, &v)
}

// McSetUint64s set value to McCache and db that map by key
func (sh *StateHelper) McSetUint64s(key string, v []uint64) {
	sh.McSet(key, &v)
}

// McSetBytes set value to McCache and db that map by key
func (sh *StateHelper) McSetBytes(key string, v []byte) {
	sh.McSet(key, &v)
}

// McSetBools set value to McCache and db that map by key
func (sh *StateHelper) McSetBools(key string, v []bool) {
	sh.McSet(key, &v)
}

// McSetStrings set value to McCache and db that map by key
func (sh *StateHelper) McSetStrings(key string, v []string) {
	sh.McSet(key, &v)
}

// McClear dirty object in McCache that map by key
func (sh *StateHelper) McClear(key string) {
	if !sh.checkKey(key) {
		return
	}

	fullKey := sh.smc.Message().Contract().KeyPrefix() + key
	sdkimpl.McInst.Dirty(fullKey)
}

// McDelete dirty and delete data map by key
func (sh *StateHelper) McDelete(key string) {
	if !sh.checkKey(key) {
		return
	}

	fullKey := sh.smc.Message().Contract().KeyPrefix() + key
	sdkimpl.McInst.Dirty(fullKey)
	sh.smc.(*sdkimpl.SmartContract).LlState().Delete(fullKey)
}

func (sh *StateHelper) checkKey(key string) bool {
	if len(key) == 0 || key[0] != '/' {
		sdkimpl.Logger.Errorf("[sdk]The key=%s is not prefix \"/\"", key)
		return false
	}

	return true
}
