package tokenbyb

import (
	"encoding/json"
	"math/big"

	"blockchain/abciapp_v1.0/smc"
	"common/bignumber_v1.0"
)

type bybBalance struct {
	Chromo smc.Chromo `json:"chromo"` //byb染色体
	Value  big.Int    `json:"value"`  //byb余额
}

func keyOfBlackHole() string {
	return "/byb/blackholes"
}

func keyOfStockholders() string {
	return "/byb/stockholders"
}

func keyOfAccount(exAddress smc.Address) string {
	return "/account/ex/" + exAddress
}

func keyOfCurChromo() string {
	return "/byb/curchromo"
}

func (byb *TokenByb) setBlackHole(blackHoles []smc.Address) error {
	blackHolesData, err := json.Marshal(blackHoles)
	if err != nil {
		return err
	}
	return byb.State.Set(keyOfBlackHole(), []byte(blackHolesData))
}

func (byb *TokenByb) getBlackHole() ([]smc.Address, error) {
	value, err := byb.State.Get(keyOfBlackHole())
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}
	var blackHoles []smc.Address
	err = json.Unmarshal(value, &blackHoles)
	if err != nil {
		return nil, err
	}
	return blackHoles, nil
}

// 设置账户BYB
func (byb *TokenByb) setBybBalance(addr smc.Address, balances []bybBalance) error {
	bybBalanceData, err := json.Marshal(&balances)
	if err != nil {
		return err
	}

	key := keyOfAccount(addr)
	childKey := key + "/byb"
	data, err := byb.State.Get(childKey)
	if err != nil {
		return err
	}
	if data == nil {
		err = byb.State.AddChildKey(key, childKey)
		if err != nil {
			return err
		}
	}
	//保存byb账户的详细余额
	err = byb.State.Set(childKey, bybBalanceData)
	if err != nil {
		return err
	}

	//保存账户的byb总额
	var totalBalance big.Int
	for _, balance := range balances {
		totalBalance = bignumber.Add(totalBalance, balance.Value)
	}
	return byb.State.SetBalance(addr, byb.State.ContractAddress, totalBalance)
}

func (byb *TokenByb) getBybBalance(addr smc.Address) ([]bybBalance, error) {
	//根据合约地址，在内部构造出key
	key := keyOfAccount(addr) + "/byb"
	bybBalanceData, err := byb.State.Get(key)
	if err != nil {
		return nil, err
	}
	if bybBalanceData == nil {
		return nil, nil
	}

	var balances []bybBalance
	err = json.Unmarshal(bybBalanceData, &balances)
	if err != nil {
		return nil, err
	}
	return balances, nil
}

// 获取BYB股东地址列表
func (byb *TokenByb) getBybStockHolders() ([]smc.Address, error) {
	value, err := byb.State.Get(keyOfStockholders())
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}
	var StockHolders []smc.Address
	err = json.Unmarshal(value, &StockHolders)
	if err != nil {
		return nil, err
	}
	return StockHolders, nil
}

// 新增BYB股东
func (byb *TokenByb) setBybStockHolders(stockHolders []smc.Address) error {
	stockHoldersData, err := json.Marshal(stockHolders)
	if err != nil {
		return err
	}
	return byb.State.Set(keyOfStockholders(), []byte(stockHoldersData))
}

func (byb *TokenByb) setCurChromo(chromo smc.Chromo) error {
	return byb.State.Set(keyOfCurChromo(), []byte(chromo))
}

func (byb *TokenByb) getCurChromo() (smc.Chromo, error) {
	value, err := byb.State.Get(keyOfCurChromo())
	if err != nil {
		return "", err
	}
	if value == nil {
		return "", nil
	}

	return smc.Chromo(value), nil
}
