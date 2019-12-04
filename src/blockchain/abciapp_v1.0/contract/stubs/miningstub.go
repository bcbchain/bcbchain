package stubs

import (
	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/contract/mining"
	"blockchain/abciapp_v1.0/contract/smcapi"
	"blockchain/abciapp_v1.0/contract/stubapi"
	"blockchain/abciapp_v1.0/prototype"
	"blockchain/abciapp_v1.0/smc"
	"encoding/json"
	"fmt"
	"github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"
	"strconv"
)

const (
	MN_METHODID_MINE = iota

	//number
	MN_METHODID_TOTAL_COUNT
)

type MNStub struct {
	logger    log.Logger
	MNMethods []Method
}

func NewMN(ctx *stubapi.InvokeContext) *smcapi.SmcApi {
	newsmcapi := smcapi.SmcApi{Sender: ctx.Sender,
		Owner:        ctx.Owner,
		ContractAcct: CalcContractAcct(ctx, prototype.MINING),
		ContractAddr: &ctx.Owner.TxState.ContractAddress,
		State:        ctx.TxState,
		Block: &smcapi.Block{ctx.BlockHash,
			ctx.BlockHeader.ChainID,
			ctx.BlockHeader.Height,
			ctx.BlockHeader.Time,
			ctx.BlockHeader.NumTxs,
			ctx.BlockHeader.DataHash,
			ctx.BlockHeader.LastBlockID.Hash,
			ctx.BlockHeader.LastCommitHash,
			ctx.BlockHeader.LastAppHash,
			ctx.BlockHeader.LastFee,
			ctx.BlockHeader.ProposerAddress,
			ctx.BlockHeader.RewardAddress,
			ctx.BlockHeader.RandomeOfBlock}}

	smcapi.InitEventHandler(&newsmcapi)

	return &newsmcapi
}

// NewMNStub creates TokenMN stub and initialize it with Methods
func NewMNStub(logger log.Logger) *MNStub {
	// create methodID
	var stub MNStub
	stub.logger = logger
	stub.MNMethods = make([]Method, MN_METHODID_TOTAL_COUNT)
	stub.MNMethods[MN_METHODID_MINE].Prototype = prototype.MNMine

	for i, method := range stub.MNMethods {
		stub.MNMethods[i].MethodID = stubapi.ConvertPrototype2ID(method.Prototype)
		logger.Info("  method()",
			"id", strconv.FormatUint(uint64(stub.MNMethods[i].MethodID), 16),
			"prototype", stub.MNMethods[i].Prototype)
	}
	stubapi.SetLogger(logger)

	return &stub
}

func (mns *MNStub) Methods(addr smc.Address) []Method {
	return mns.MNMethods
}

func (mns *MNStub) Name(addr smc.Address) string {
	return prototype.MNMine
}

// Dispatcher decodes tx data that was sent by caller, and dispatch it to smart contract to execute.
// The response would be empty if there is error happens (err != nil)
func (mns *MNStub) Dispatcher(items *stubapi.InvokeParams, transID int64) (response stubapi.Response, bcerr bcerrors.BCError) {

	response.Data = "" // Don't have response data from contract method

	// construct mining object
	mnObj := mining.Mining{SmcApi: NewMN(items.Ctx)}

	// To decode method parameter with RLP API and call specified Method of smart contract depends on MethodID
	mns.logger.Debug("Dispatcher, Calling Mine() Function")
	response.RequestMethod = mns.MNMethods[MN_METHODID_MINE].Prototype

	rewardAmount, bcerr := mnObj.Mine()
	if bcerr.ErrorCode == bcerrors.ErrCodeOK {
		response.Data = fmt.Sprintf("%d", rewardAmount)
		addReceiptToMNResponse(&mnObj, &response)
	}
	return
}

// CodeHash gets smart contract code hash
func (mns *MNStub) CodeHash() []byte {
	//TBD
	return nil
}

func addReceiptToMNResponse(mnObj *mining.Mining, response *stubapi.Response) {
	for index, receipt := range mnObj.EventHandler.GetReceipts() {
		recByte, _ := json.Marshal(receipt)
		kvPair := common.KVPair{Key: []byte(fmt.Sprintf("%d", index)), Value: recByte}
		response.Tags = append(response.Tags, kvPair)
	}
}
