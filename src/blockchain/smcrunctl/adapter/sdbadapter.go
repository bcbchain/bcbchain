package adapter

import (
	"blockchain/smcsdk/sdk/std"
)

var (
	get   GetCallback
	set   SetCallback
	build BuildCallback
)

//GetCallback callback of get()
type GetCallback func(int64, int64, string) ([]byte, error)

//SetCallback callback of set()
type SetCallback func(int64, int64, map[string][]byte) (*bool, error)

//BuildCallback callback of build()
type BuildCallback func(int64, int64, std.ContractMeta) (*std.BuildResult, error)

//SetSdbCallback set sdb callback
func SetSdbCallback(getFunc GetCallback, setFunc SetCallback, buildCallback BuildCallback) {
	get = getFunc
	set = setFunc
	build = buildCallback
}

//Get get key's value from sdb
func Get(transID, txID int64, key string) ([]byte, error) {
	return get(transID, txID, key)
}

//Set set key and value to sdb
func Set(transID, txID int64, data map[string][]byte) (*bool, error) {
	return set(transID, txID, data)
}

//Build build contract and save to sdb
func Build(transID, txID int64, contractMeta std.ContractMeta) (*std.BuildResult, error) {

	return build(transID, txID, contractMeta)
}
