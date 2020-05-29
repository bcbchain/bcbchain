package tokenbasic_foundation

import (
	"encoding/json"
)

func keyOfGlobal() string {
	return "/TBF(BCB)/global"
}

// global info
func (t *TBFoundation) setGlobalDB(v *TBFGlobal) error {
	//存到状态数据库
	value, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return t.State.Set(keyOfGlobal(), value)
}

func (t *TBFoundation) getGlobalDB() (*TBFGlobal, error) {
	//从状态数据库读取
	temp := &TBFGlobal{}
	value, err := t.State.Get(keyOfGlobal())
	if err != nil {
		panic(err)
	}
	if len(value) == 0 {
		return temp, nil
	}
	var globalInfo TBFGlobal
	err = json.Unmarshal(value, &globalInfo)
	if err != nil {
		panic(err)
	}

	return &globalInfo, nil
}
