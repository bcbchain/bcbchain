package invokermgr

import (
	"github.com/bcbchain/bcbchain/abciapp/softforks"
	"github.com/bcbchain/bclib/types"
	"fmt"
	"math"
)

// resetGasUsed if height in [23706999, forkHeight] then reset gas_used
func (im *InvokerMgr) resetGasUsed(height int64, result *types.Response, tx types.Transaction) {
	// reset gas_used
	for _, msg := range tx.Messages {
		//contract := statedbhelper.GetContract(msg.Contract)
		contract, err := im.getEffectContract(0, 0, height, msg.Contract, msg.MethodID)
		if err.ErrorCode != types.CodeOK {
			panic(err.ErrorDesc)
		}

		if softforks.FilterContracts_V2_0_2_14654(contract.OrgID, contract.Name) {
			return
		}
	}

	msg := tx.Messages[len(tx.Messages)-1]
	contract, err := im.getEffectContract(0, 0, height, msg.Contract, msg.MethodID)
	if err.ErrorCode != types.CodeOK {
		panic(err.ErrorDesc)
	}
	for _, method := range contract.Methods {
		methodID := fmt.Sprintf("%x", msg.MethodID)
		if method.MethodID == methodID {
			result.GasUsed = int64(math.Abs(float64(method.Gas)))
			break
		}
	}
}
