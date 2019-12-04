package tokenbasic_foundation

import (
	"blockchain/abciapp_v1.0/contract/smcapi"
	. "common/bignumber_v1.0"
)

const timeLayout = "2006-01-02 15:04:05"

var (
	totalBCB    = N(19800000E9)
	addressList = []string{
		"bcbDJSLrajxUwu8JWX54GKzUQQuDMBeMs3Z3",
		"bcbCv1zLTK1Mk2Lntuh9hYzgoM3yLwCFs3Rj",
		"bcb68adY4FTK5eqRfiXxxbm6SC2YoE2Nmajh",
		"bcb4WwQ7QLPvP1KavgrLSGVQH8dPV77W2eN3",
		"bcb3TE7Jii6KGrNpkiZFvMjREVa3nsaUXMCi",
	}
)

type TBFoundation struct {
	*smcapi.SmcApi
	global__ *TBFGlobal
}

type UnLock struct {
	UnlockTime string
	Amount     Number
	Settled    bool
}

type TBFGlobal struct {
	UnlockInfo []UnLock `json:"unlockInfo"`
}

func (t *TBFGlobal) init() {
	t.UnlockInfo = []UnLock{
		//BCB基金会总资金池数量：19,800,000bcb,按比例解锁5年
		//第1年解冻时间：2019年7月21日0点    解锁比例：10％
		//第2年解冻时间：2020年7月21日0点    解锁比例：20％
		//第3年解冻时间：2021年7月21日0点    解锁比例：40％
		//第4年解冻时间：2022年7月21日0点    解锁比例：20％
		//第5年解冻时间：2023年7月21日0点    解锁比例：10％
		{"2019-07-21 00:00:00", totalBCB.Mul_(10).Div_(100), false},
		{"2020-07-21 00:00:00", totalBCB.Mul_(20).Div_(100), false},
		{"2021-07-21 00:00:00", totalBCB.Mul_(40).Div_(100), false},
		{"2022-07-21 00:00:00", totalBCB.Mul_(20).Div_(100), false},
		{"2023-07-21 00:00:00", totalBCB.Mul_(10).Div_(100), false},
	}
}
