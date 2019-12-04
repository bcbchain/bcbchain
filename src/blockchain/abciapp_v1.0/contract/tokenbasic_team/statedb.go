package tokenbasic_team

import (
	"encoding/json"
)

func keyOfGlobal() string {
	return "/TBT(BCB)/global"
}

// global info
func (t *TBTeam) setGlobalDB(v *TBTGlobal) error {
	//存到状态数据库
	value, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return t.State.Set(keyOfGlobal(), value)
}

func (t *TBTeam) getGlobalDB() (*TBTGlobal, error) {
	//从状态数据库读取
	temp := &TBTGlobal{}
	value, err := t.State.Get(keyOfGlobal())
	if err != nil {
		panic(err)
	}
	if len(value) == 0 {
		return temp, nil
	}
	var globalInfo TBTGlobal
	err = json.Unmarshal(value, &globalInfo)
	if err != nil {
		panic(err)
	}

	return &globalInfo, nil
}
