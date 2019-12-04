package transferAgency

import (
	"encoding/json"
)

func keyOfTokenFeeInfo(tokenName string) string {
	return "/" + Tag + "/tokenFeeInfo" + "/" + tokenName
}

func keyOfManager() string {
	return "/" + Tag + "/managerInfo"
}

func (tac *TransferAgency) SetManagerInfoDB(v *ManagerInfo) error {
	//存到状态数据库
	value, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return tac.State.Set(keyOfManager(), value)
}

func (tac *TransferAgency) getManagerInfoDB() (*ManagerInfo, error) {
	//从状态数据库读取
	temp := &ManagerInfo{}
	value, err := tac.State.Get(keyOfManager())
	if err != nil {
		panic(err)
	}
	if len(value) == 0 {
		temp.init()
		tac.SetManagerInfoDB(temp)
		return temp, nil
	}
	var manager ManagerInfo
	err = json.Unmarshal(value, &manager)
	if err != nil {
		panic(err)
	}

	return &manager, nil
}

func (tac *TransferAgency) setTokenFeeInfoDB(tokenName string, v *TokenFeeInfo) error {
	//存到状态数据库
	value, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return tac.State.Set(keyOfTokenFeeInfo(tokenName), value)
}
func (tac *TransferAgency) getTokenFeeInfoDB(tokenName string) (*TokenFeeInfo, error) {
	//从状态数据库读取
	temp := &TokenFeeInfo{}
	value, err := tac.State.Get(keyOfTokenFeeInfo(tokenName))
	if err != nil {
		panic(err)
	}
	if len(value) == 0 {
		temp.init()
		return temp, nil
	}
	var tokenFeeInfo TokenFeeInfo
	err = json.Unmarshal(value, &tokenFeeInfo)
	if err != nil {
		panic(err)
	}

	return &tokenFeeInfo, nil
}
