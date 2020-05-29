package mining

import (
	"encoding/json"
)

func keyOfStartHeight() string  { return "/mining/start/height" }
func keyOfRewardAmount() string { return "/mining/reward/amount" }

//getMiningStartHeight This is a get mining start height method of Mining
func (m *Mining) getMiningStartHeight() int64 {
	//从状态数据库读取
	value, err := m.State.Get(keyOfStartHeight())
	if err != nil {
		panic(err)
	}

	if len(value) == 0 {
		return 0
	}

	var v int64
	err = json.Unmarshal(value, &v)
	if err != nil {
		panic(err)
	}

	return v
}

//setMiningStartHeight This is a set mining start height method of Mining
func (m *Mining) setMiningStartHeight(v int64) {
	value, err := json.Marshal(&v)
	if err != nil {
		panic(err)
	}

	if err = m.State.Set(keyOfStartHeight(), value); err != nil {
		panic(err)
	}
}

//getMiningRewardAmount This is a get mining start height method of Mining
func (m *Mining) getMiningRewardAmount() uint64 {
	//从状态数据库读取
	value, err := m.State.Get(keyOfRewardAmount())
	if err != nil {
		panic(err)
	}

	if len(value) == 0 {
		return 0
	}

	var v uint64
	err = json.Unmarshal(value, &v)
	if err != nil {
		panic(err)
	}

	return v
}

//setMiningRewardAmount This is a set mining start height method of Mining
func (m *Mining) setMiningRewardAmount(v uint64) {
	value, err := json.Marshal(&v)
	if err != nil {
		panic(err)
	}
	if err = m.State.Set(keyOfRewardAmount(), value); err != nil {
		panic(err)
	}
}
