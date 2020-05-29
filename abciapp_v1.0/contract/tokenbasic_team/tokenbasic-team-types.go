package tokenbasic_team

import (
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/smcapi"
	. "github.com/bcbchain/bclib/bignumber_v1.0"
)

const timeLayout = "2006-01-02 15:04:05"

var (
	totalBCB    = N(13200000E9)
	addressList = []string{
		"bcbJkkFcqarbCyuMnVWsnR6MDRtGg56TokJp",
		"bcbPLrAYFYRrxnzM4vbJmam7hperGWYa6q6p",
		"bcb62xxXHUsTT1NpZ5ZEXnSf3E1j9PbN7nNf",
		"bcbHSUwdkmqdQPq5Roi9d95mRTthkz94a6i2",
		"bcbMJGB58Ys4PsYVt3GqAkd9scYVbXDGh6qu",
	}
)

type TBTeam struct {
	*smcapi.SmcApi
	global__ *TBTGlobal
}

type UnLock struct {
	UnlockTime string
	Amount     Number
	Settled    bool
}

type TBTGlobal struct {
	UnlockInfo []UnLock `json:"unlockInfo"`
}

func (t *TBTGlobal) init() {
	t.UnlockInfo = []UnLock{
		//团队总资金池数量：13,200,000bcb,按比例解锁5年
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
