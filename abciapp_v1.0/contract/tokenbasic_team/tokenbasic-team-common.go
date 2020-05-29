package tokenbasic_team

import (
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
)

func require(expr bool, errcode uint32, errinfo string, smcError *smc.Error) bool {
	if expr == false {
		smcError.ErrorCode = errcode
		smcError.ErrorDesc = errinfo
	}
	return expr
}
