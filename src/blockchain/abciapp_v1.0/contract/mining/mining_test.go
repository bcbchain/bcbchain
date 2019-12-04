package mining

import (
	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/contract/smcapi"
	"blockchain/abciapp_v1.0/contract/stubapi"
	"blockchain/abciapp_v1.0/prototype"
	"blockchain/abciapp_v1.0/smc"
	"blockchain/abciapp_v1.0/statedb"
	"blockchain/algorithm"
	"common/bcdb"
	. "common/bignumber_v1.0"
	"fmt"
	"github.com/tendermint/abci/types"
	"math"
	"math/big"
	"testing"
)

var dbtest *bcdb.GILevelDB
var chainId string

func TestMing(t *testing.T) {
	mnCex, err := newContext()
	if err != nil {
		panic(err)
	}
	defer dbtest.Close()
	mn := Mining{mnCex}

	//给合约账户转钱
	token, e := mn.State.GetGenesisToken()
	if e != nil {
		t.Log("err:", err.Error())
		return
	}
	nerr := mn.EventHandler.TransferByAddr(token.Address, token.Owner, mn.ContractAcct.Addr, N(1.98*1E16))
	if nerr.ErrorCode != bcerrors.ErrCodeOK {
		t.Log("err:", err.Error())
		return
	}

	var testCases = []struct {
		mining Mining
		want   bool
	}{ //未转账进行部署
		{mn, true},
	}

	for _, c := range testCases {
		currentHeight, err := c.mining.GetCurrentBlockHeight()
		if err != nil {
			t.Log("err", err.Error())
		}

		startHeight := c.mining.miningStartHeight_()
		if startHeight == 0 {
			c.mining.setMiningStartHeight_(currentHeight)
			startHeight = c.mining.miningStartHeight_()
		}

		for {
			rewardAmount := calcRewardAmount(currentHeight, c.mining.miningStartHeight_())
			tokenaddr, err := c.mining.State.GetTokenAddrByName("LOC")
			if err != nil {
				t.Log("err", err.Error())
			}

			beforeBal, err := c.mining.State.GetBalance(c.mining.Block.RewardAddress, tokenaddr)
			if err != nil {
				t.Log("err", err.Error())
			}
			_, lerr := c.mining.Mine()
			afterBal, err := c.mining.State.GetBalance(c.mining.Block.RewardAddress, tokenaddr)
			if err != nil {
				t.Log("err", err.Error())
			}
			rewardBal := new(big.Int).Sub(&afterBal, &beforeBal).Int64()
			//fmt.Println("rewardBal:", rewardBal)

			if lerr.ErrorCode != bcerrors.ErrCodeOK {
				t.Errorf("lerr:%v", lerr)
				break
			} else {
				if rewardAmount != 0 {
					if rewardBal != rewardAmount {
						t.Errorf("fail :want=%v", c.want)
					}
				} else {
					if rewardBal == 1 {
						t.Logf("succed :want=%v", c.want)
						break
					} else {
						t.Errorf("fail :want=%v", c.want)
					}
				}
			}
			if currentHeight < 66000000 {
				currentHeight = c.mining.miningStartHeight_() + 66000000
			} else {
				currentHeight += 66000000
			}
			setHeight(mnCex, currentHeight-1)
		}

	}

}

func setHeight(smc *smcapi.SmcApi, height int64) {
	err := smc.State.StateDB.SetWorldAppState(&types.AppState{BlockHeight: height})
	if err != nil {
		fmt.Println("err:", err.Error())
	}

}

func newContext() (*smcapi.SmcApi, error) {
	// open db
	var err error
	dbtest, err = bcdb.OpenDB("D:\\blockchain1.0\\v1.0\\gichain\\bin\\.appstate", "127.0.0.1", "8888")
	//if err != nil {
	//	return nil, err
	//}
	//dbtest, err = bcdb.OpenDB("/Users/zerppen/.appstate", "127.0.0.1", "8888")
	if err != nil {
		return nil, err
	}
	stateDB := statedb.NewStateDB(dbtest)
	stateDB.BlockBuffer = make(map[string][]byte)

	chainId = "local"

	// 获取系统合约地址，填入txState
	var contractAddr, ownerAddr smc.Address
	//genContracts, _ := stateDB.GetContractAddrList()
	//for _, addr := range genContracts {
	//	contract, err := stateDB.GetContract(addr)
	//	if err != nil || contract == nil {
	//		panic(errors.New("Failed to get contract"))
	//	}
	//	if contract.Name == prototype.MINING {
	//		contractAddr = contract.Address
	//		ownerAddr = contract.Owner
	//	}
	//}
	var txState *statedb.TxState
	//Generate txState to operate stateDB
	senderAddress := "localJYuNvyAz1TcfRZNqUhB1afsAMuXE2RqjQ"
	// 系统合约的地址和拥有者
	txState = stateDB.NewTxState(contractAddr, senderAddress)
	txState.BeginTx()

	// Generate accounts and execute
	sender := &stubapi.Account{
		Addr:    senderAddress,
		TxState: txState,
	}
	// Get token basic and its contract
	tb, err := stateDB.GetGenesisToken()
	if err != nil || tb == nil {
		return nil, err
	}
	// Using tokenbasic owner as it doesn't be changed
	owner := &stubapi.Account{
		ownerAddr,
		txState,
	}

	invokeContext := smcapi.SmcApi{
		sender,
		owner,
		nil,
		&contractAddr,
		txState,
		nil,
		nil,
		"",
	}
	invokeContext.Block = &smcapi.Block{
		Height:        1,
		RewardAddress: "localDfqqUiWudBQBSg88kiu5Z7t96fcsBiPk2",
	}

	sct := stubapi.InvokeContext{Sender: invokeContext.Sender, Owner: invokeContext.Owner, TxState: invokeContext.State}
	invokeContext.ContractAcct = calcContractAcct(&sct, prototype.MINING)

	invokeContext.EventHandler = &smcapi.EventHandler{}
	smcapi.InitEventHandler(&invokeContext)
	return &invokeContext, nil
}
func calcContractAcct(ctx *stubapi.InvokeContext, contractName string) *stubapi.Account {

	addr := algorithm.CalcContractAddress(
		ctx.TxState.GetChainID(),
		"",
		contractName,
		"")

	genToken, _ := ctx.TxState.GetGenesisToken()
	if genToken == nil {
		return nil
	}

	return &stubapi.Account{addr,
		&statedb.TxState{ctx.TxState.StateDB,
			genToken.Address,
			ctx.TxState.SenderAddress,
			ctx.TxState.TxBuffer}}
}

func calcRewardAmount(cHeight, sHeight int64) (rewardAmount int64) {
	blockNum := cHeight - sHeight
	if blockNum == 0 {
		//给奖励地址150000000cong
		rewardAmount = int64(150000000)
	} else {
		rewardAmount = int64(150000000 / int64(math.Pow(2, float64(blockNum/66000000))))
	}

	return
}
