package stubapi

import (
	"github.com/bcbchain/bcbchain/abciapp_v1.0/smc"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/statedb"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/types"
	abci "github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
)

type Account struct {
	Addr    smc.Address      // Account address
	TxState *statedb.TxState // StateDB
}

type InvokeParams struct {
	Ctx    *InvokeContext // invoke context
	Params []byte         //The Parameter that encoded in RLP
}

type InvokeContext struct {
	Sender      *Account         // Address of contract caller,
	Owner       *Account         // Address of contract owner
	TxState     *statedb.TxState // State DB
	BlockHash   []byte
	BlockHeader abci.Header // Header of current block
	Proposer    *Account    // Proposer of the block which containing the tx
	Rewarder    *Account    // Proposer of the block which containing the tx
	GasLimit    uint64      // The maximum gas allowed for this specific transaction
	Note        string
}

type InvokeParamsEx struct {
	TransID         int64       // Transaction ID
	Sender          smc.Address // Address of contract caller
	Owner           smc.Address // Address of contract owner
	BlockHash       []byte      // Hash bo current block
	BlockHeader     abci.Header // Header of current block
	Proposer        smc.Address // Proposer of the block containing the tx
	Rewarder        smc.Address // Proposer of the block containing the tx
	GasLimit        uint64      // The maximum gas allowed for transaction
	Note            string      // Note of transacton
	ContractAddress smc.Address // Address of contract
	Params          []byte      // The Parameter that encoded in RLP
}

type Token struct {
	Addr smc.Address    // Address of token
	Ctx  *InvokeContext // tx
}

const (
	RESPONSE_CODE_NEWTOKEN = 1 + iota
	RESPONSE_CODE_UPDATE_VALIDATORS
	RESPONSE_CODE_NEWBYBCONTRACT
	RESPONSE_CODE_NEWCGSCONTRACT
	RESPONSE_CODE_NEWCLTCONTRACT
	RESPONSE_CODE_NEWCOMCONTRACT
	RESPONSE_CODE_CGSQUERY
	RESPONSE_CODE_BYBCHROMO
	RESPONSE_CODE_NEWSXTCONTRACT
	RESPONSE_CODE_SXTQUERY
	RESPONSE_CODE_CLTQUERY
	RESPONSE_CODE_NEWBLMCONTRACT
	RESPONSE_CODE_NEWUNITEDTOKEN
	RESPONSE_CODE_NEWDICE2WINCONTRACT
	RESPONSE_CODE_ECQUERY
	RESPONSE_CODE_NEWEVERYCOLORCONTRACT
	RESPONSE_CODE_NEWDWDCCONTRACT
	RESPONSE_CODE_NEWDWUSDXCONTRACT
	RESPONSE_CODE_NEWECUSDXCONTRACT
	RESPONSE_CODE_NEWECDCCONTRACT
	RESPONSE_CODE_NEWDRAGONVSTIGERCONTRACT
	RESPONSE_CODE_NEWDTUSDXCONTRACT
	RESPONSE_CODE_NEWDTDCCONTRACT
	RESPONSE_CODE_NEWDWXTCONTRACT
	RESPONSE_CODE_NEWDCYUEBAOCONTRACT
	RESPONSE_CODE_NEWDTXTCONTRACT
	RESPONSE_CODE_NEWBACCARATCONTRACT
	RESPONSE_CODE_NEWDRAGONVSTIGERCONTRACT2_0
	RESPONSE_CODE_NEWDTUSDXCONTRACT2_0
	RESPONSE_CODE_NEWDTDCCONTRACT2_0
	RESPONSE_CODE_NEWDTXTCONTRACT2_0
	RESPONSE_CODE_NEWICTCONTRACT
	RESPONSE_CODE_NEWBACCARATCONTRACT2_0
	RESPONSE_CODE_NEWBACDCCONTRACT2_0
	RESPONSE_CODE_NEWBACXTCONTRACT2_0
	RESPONSE_CODE_NEWBACUSDXCONTRACT2_0
	RESPONSE_CODE_NEWTRANSFERAGENCY
	RESPONSE_CODE_NEWBACCARATCONTRACT3_0
	RESPONSE_CODE_NEWBACDCCONTRACT3_0
	RESPONSE_CODE_NEWBACXTCONTRACT3_0
	RESPONSE_CODE_NEWBACUSDXCONTRACT3_0
	RESPONSE_CODE_NEWICTCONTRACT2_0
	RESPONSE_CODE_NEWDICE2WINCONTRACT2_0
	RESPONSE_CODE_NEWEVERYCOLORCONTRACT2_0
	RESPONSE_CODE_NEWSICBOCONTRACT
	RESPONSE_CODE_NEWDCYUEBAOCONTRACT2_0
	RESPONSE_CODE_NEWTBCANCELLATIONCONTRACT
	RESPONSE_CODE_UPGRADE1TO2
	RESPONSE_CODE_RUNUPGRADE1TO2
	RESPONSE_CODE_NEWUSDYYUEBAOCONTRACT
	RESPONSE_CODE_NEWMININGCONTRACT
	RESPONSE_CODE_NEWTOKENBASICTEAM
	RESPONSE_CODE_NEWTOKENBASICFOUNDATION
)

// Response data for contract action
type Response struct {
	// RequestMethod is the request that was sent to obtain this Response
	// now, prototype is using for this field.
	RequestMethod string
	// GasUsed 	is the amount of gas used by this specific transaction alone
	GasUsed uint64
	// GasPrice provided by the token in Cong
	GasPrice uint64

	// Reward indicates reward amount for each reward address,
	// The value of uint64 is Fee in Cong
	RewardValues map[smc.Address]uint64

	//Code: See above definitions for details
	Code uint32

	// Data is the response data what the Contract Method returned.
	// For NewToken, it is the new contract address,
	// For Update Validators, it's the validator's address
	// For New Contract, it's contract address
	// For Query, it's response data
	// For others, it's nil for now.
	Data string

	// Tags is the receipt data what the Transaction's result
	// For Buy, it is the keys count of buy
	// For withDraw, it is the withdraw bcb number
	// For others, it's nil for now
	Tags common.KVPairs

	// error message
	Log string
}

type rewardStrategy struct {
	RewardStrategy []types.Rewarder `json:"rewardStrategy, omitempty"`
}

const (
	UDCState_Unmatured = "unmatured"
	UDCState_Matured   = "matured"
	UDCState_Expired   = "expired"
)
