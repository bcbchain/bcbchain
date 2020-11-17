package deliver

//nolint weak
import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	types4 "github.com/bcbchain/bcbchain/abciapp/service/types"
	"github.com/bcbchain/bcbchain/abciapp/softforks"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bcbchain/smcrunctl/adapter"
	"github.com/bcbchain/bcbchain/statedb"
	"github.com/bcbchain/bclib/algorithm"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
	tx2 "github.com/bcbchain/bclib/tx/v2"
	tx3 "github.com/bcbchain/bclib/tx/v3"
	types2 "github.com/bcbchain/bclib/types"
	"github.com/bcbchain/sdk/sdk/bn"
	"github.com/bcbchain/sdk/sdk/crypto/sha3"
	"github.com/bcbchain/sdk/sdk/jsoniter"
	"github.com/bcbchain/sdk/sdk/std"
	types3 "github.com/bcbchain/sdk/sdk/types"
	"sort"
	"strconv"
	"strings"
)

func (app *AppDeliver) deliverBCTx(tx []byte) (resDeliverTx types.ResponseDeliverTx, txBuffer map[string][]byte) {

	app.logger.Info("Recv ABCI interface: DeliverTx", "tx", string(tx))
	if app.chainID == "" {
		app.SetChainID(statedbhelper.GetChainID())
	}
	app.txID = statedbhelper.NewTx(app.transID)

	// for base58
	tx2.Init(app.chainID)
	transaction, pubKey, err := tx2.TxParse(string(tx))
	if err != nil {
		// for base64
		tx3.Init(app.chainID)
		transaction, pubKey, err = tx3.TxParse(string(tx))
		if err != nil {
			app.logger.Error("tx parse failed:", err)
			return app.ReportFailure(tx, types2.ErrDeliverTx, "tx parse failed"), nil
		}
	}
	app.logger.Debug("DELIVER.TX", "height", app.blockHeader.Height, "tx", transaction, "pubKey", pubKey, "addr", pubKey.Address(statedbhelper.GetChainID()))

	return app.runDeliverTx(tx, transaction, pubKey)
}

func (app *AppDeliver) deliverBCTxCurrency(tx []byte) (result types4.Result2) {

	app.logger.Info("Recv ABCI interface: DeliverTxCurrency", "tx", string(tx))
	if app.chainID == "" {
		app.SetChainID(statedbhelper.GetChainID())
	}
	//app.txID = statedbhelper.NewTx(app.transID)
	result.TxID = statedbhelper.NewTx(app.transID)

	result.Tx = tx
	// for base58
	tx2.Init(app.chainID)
	transaction, pubKey, err := tx2.TxParse(string(tx))
	if err != nil {
		// for base64
		tx3.Init(app.chainID)
		transaction, pubKey, err = tx3.TxParse(string(tx))
		result.TxV3Result.Transaction = transaction
		result.TxV3Result.Pubkey = pubKey
		result.TxVersion = "v3"
		if err != nil {
			app.logger.Error("tx parse failed:", err)
			result.ErrorLog = errors.New("tx parse failed")
			return result
			//return app.ReportFailure(tx, types2.ErrDeliverTx, "tx parse failed"), nil
		}
	}
	app.logger.Debug("DELIVER.TX", "height", app.blockHeader.Height, "tx", transaction, "pubKey", pubKey, "addr", pubKey.Address(statedbhelper.GetChainID()))

	result.TxV2Result.Transaction = transaction
	result.TxV2Result.Pubkey = pubKey
	result.TxVersion = "v2"

	return result
}

func (app *AppDeliver) runDeliverTx(tx []byte, transaction types2.Transaction, pubKey crypto.PubKeyEd25519) (resDeliverTx types.ResponseDeliverTx, txBuffer map[string][]byte) {
	resDeliverTx.Code = types2.CodeOK

	if len(transaction.Note) > types2.MaxSizeNote {
		return app.ReportFailure(tx, types2.ErrDeliverTx, "tx note is out of range"), nil
	}

	sender := pubKey.Address(statedbhelper.GetChainID())
	nonceBuffer, err := statedbhelper.SetAccountNonce(app.transID, app.txID, sender, transaction.Nonce)
	if err != nil {
		app.logger.Error("SetAccountNonce failed:", err)
		return app.ReportFailure(tx, types2.ErrDeliverTx, "SetAccountNonce failed"), nil
	}

	txHash := common.HexBytes(algorithm.CalcCodeHash(string(tx)))
	adp := adapter.GetInstance()
	response := adp.InvokeTx(app.blockHeader, app.transID, app.txID, sender, transaction, pubKey.Bytes(), txHash, app.blockHash)
	if response.Code != types2.CodeOK {
		app.logger.Error("docker invoke error.....", "error", response.Log)
		app.logger.Debug("docker invoke error.....", "response", response.String())
		statedbhelper.RollbackTx(app.transID, app.txID)
		adp.RollbackTx(app.transID, app.txID)
		resDeliverTx, txBuffer, totalFee := app.reportInvokeFailure(tx, transaction, response)
		resDeliverTx.Fee = uint64(totalFee)
		return resDeliverTx, combineBuffer(nonceBuffer, txBuffer)
	}
	app.logger.Debug("docker invoke response.....", "code", response.Code)

	// pack validators if update validator info
	if hasUpdateValidatorReceipt(response.Tags) {
		app.packValidators()
	}

	// pack side chain genesis info
	if t, ok := hasSideChainGenesisReceipt(response.Tags); ok {
		app.packSideChainGenesis(t)
	}

	//emit new summary fee  and transferFee receipts
	tags, totalFee := app.emitFeeReceipts(transaction, response.Tags, true)

	resDeliverTx.Code = response.Code
	resDeliverTx.Log = response.Log
	resDeliverTx.Tags = tags
	resDeliverTx.GasLimit = uint64(transaction.GasLimit)
	resDeliverTx.GasUsed = uint64(response.GasUsed)
	resDeliverTx.Fee = uint64(totalFee)
	resDeliverTx.Data = response.Data
	resDeliverTxStr := resDeliverTx.String()
	app.logger.Debug("deliverBCTx()", "resDeliverTx length", len(resDeliverTxStr), "resDeliverTx", resDeliverTxStr) // log value of async instance must be immutable to avoid data race

	stateTx, txBuffer := statedbhelper.CommitTx(app.transID, app.txID)
	app.calcDeliverHash(tx, &resDeliverTx, stateTx)
	app.logger.Debug("deliverBCTx() ", "stateTx length", len(stateTx), "stateTx ", string(stateTx))

	//calculate Fee
	app.fee = app.fee + response.Fee
	app.logger.Debug("deliverBCTx()", "app.fee", app.fee, "app.rewards", map2String(app.rewards))

	app.logger.Debug("end deliver invoke.....")
	return resDeliverTx, combineBuffer(nonceBuffer, txBuffer)
}

//nolint unhandled
func (app *AppDeliver) calcDeliverHash(tx []byte, response *types.ResponseDeliverTx, stateTx []byte) {
	md5TX := md5.New()
	if tx != nil {
		md5TX.Write(tx)
	}

	app.logger.Debug("deliverHash", "resp", response.String(), "stateTx", string(stateTx))
	if response != nil {
		md5TX.Write([]byte(response.String()))
	}

	if stateTx != nil {
		md5TX.Write(stateTx)
	}

	if tx != nil || response != nil || stateTx != nil {
		deliverHash := md5TX.Sum(nil)
		app.hashList.PushBack(deliverHash)
	}
}

func (app *AppDeliver) CalcDeliverHash(tx []byte, response *types.ResponseDeliverTx, stateTx []byte) {
	md5TX := md5.New()
	if tx != nil {
		md5TX.Write(tx)
	}

	app.logger.Debug("deliverHash", "resp", response.String(), "stateTx", string(stateTx))
	if response != nil {
		md5TX.Write([]byte(response.String()))
	}

	if stateTx != nil {
		md5TX.Write(stateTx)
	}

	if tx != nil || response != nil || stateTx != nil {
		deliverHash := md5TX.Sum(nil)
		app.hashList.PushBack(deliverHash)
	}
}

func (app *AppDeliver) ReportFailure(tx []byte, errorCode uint32, msg string) (response types.ResponseDeliverTx) {
	response.Code = errorCode
	response.Log = msg
	app.calcDeliverHash(tx, &response, nil)
	return
}

func (app *AppDeliver) reportInvokeFailure(tx []byte, transaction types2.Transaction, response *types2.Response) (
	resDeliverTx types.ResponseDeliverTx,
	txBuffer map[string][]byte,
	totalFee int64) {

	rcpts, totalFee := app.emitFeeReceipts(transaction, response.Tags, false)
	resDeliverTx = types.ResponseDeliverTx{
		Code:     response.Code,
		Log:      response.Log,
		GasLimit: uint64(transaction.GasLimit),
		GasUsed:  uint64(response.GasUsed),
		Fee:      uint64(response.Fee),
		Tags:     rcpts,
	}
	//commit transactions of fee
	var stateTx []byte
	if len(rcpts) > 0 {
		stateTx, txBuffer = statedbhelper.CommitTx(app.transID, app.txID)
	}
	app.calcDeliverHash(tx, &resDeliverTx, stateTx)

	return
}

func (app *AppDeliver) ReportInvokeFailure(tx []byte, transaction types2.Transaction, response *types2.Response) (
	resDeliverTx types.ResponseDeliverTx,
	txBuffer map[string][]byte,
	totalFee int64) {

	rcpts, totalFee := app.emitFeeReceipts(transaction, response.Tags, false)
	resDeliverTx = types.ResponseDeliverTx{
		Code:     response.Code,
		Log:      response.Log,
		GasLimit: uint64(transaction.GasLimit),
		GasUsed:  uint64(response.GasUsed),
		Fee:      uint64(response.Fee),
		Tags:     rcpts,
	}
	//commit transactions of fee
	var stateTx []byte
	if len(rcpts) > 0 {
		stateTx, txBuffer = statedbhelper.CommitTx(app.transID, app.txID)
	}
	app.calcDeliverHash(tx, &resDeliverTx, stateTx)

	return
}

func mapFee2String(m map[types2.Address]std.Fee) string {
	b := new(bytes.Buffer)
	b.WriteString("{")
	for key, value := range m {
		_, _ = fmt.Fprintf(b, "%s:'%s',", key, value.String())
	}
	b.WriteString("}")
	return b.String()
}

func map2String(m map[types2.Address]int64) string {
	b := new(bytes.Buffer)
	b.WriteString("{")
	for key, value := range m {
		_, _ = fmt.Fprintf(b, "%s:%d,", key, value)
	}
	b.WriteString("}")
	return b.String()
}

func (app *AppDeliver) emitFeeReceipts(transaction types2.Transaction, inPutTags []common.KVPair, isDlvOK bool) (tags []common.KVPair, totalFee int64) {
	fees, feetags, totalFee := gatherFeesByFromAddr(inPutTags, isDlvOK)
	app.logger.Debug("get fee receipts", "receipts", mapFee2String(fees))
	totalFeeReceipts, err := emitTotalFeeReceipt(fees)
	if err != nil {
		app.logger.Error("emit fee receipt failed", "error", err.Error())
		return nil, 0
	}

	// if transaction was succeed, save response tags
	if !isDlvOK {
		tags = feetags
	}

	//nolint
	keys := make([]types2.Address, 0)
	for k := range totalFeeReceipts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	methodID := transaction.Messages[len(transaction.Messages)-1].MethodID // todo 级联调用需要检查手续费，或者测试两种合约能否级联
	isBVM := methodID == 0 || methodID == 0xFFFFFFFF
	for _, k := range keys {
		transRcpt, _ := app.distributeFee(fees[k], app.rewarder, isDlvOK, isBVM)
		kv := common.KVPair{
			Key:   []byte(fmt.Sprintf("/%d/0/totalFee", len(transaction.Messages))),
			Value: totalFeeReceipts[k],
		}
		tags = append(tags, kv)
		for index, r := range transRcpt {
			kv := common.KVPair{
				Key:   []byte(fmt.Sprintf("/%d/%d/transferFee", len(transaction.Messages), index+1)),
				Value: r,
			}
			tags = append(tags, kv)
		}
	}

	return
}

func (app *AppDeliver) EmitFeeReceipts(transaction types2.Transaction, response *types2.Response, isDlvOK bool) (tags []common.KVPair, totalFee int64) {
	fees, feetags, totalFee := gatherFeesByFromAddr(response.Tags, isDlvOK)
	app.logger.Debug("get fee receipts", "receipts", mapFee2String(fees))
	totalFeeReceipts, err := emitTotalFeeReceipt(fees)
	if err != nil {
		app.logger.Error("emit fee receipt failed", "error", err.Error())
		return nil, 0
	}
	// if transaction was succeed, save response tags
	if isDlvOK {
		tags = response.Tags
	} else {
		//fill original fee receipts in deliver response even it's failed.
		tags = feetags
	}
	//nolint
	keys := make([]types2.Address, 0)
	for k := range totalFeeReceipts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	methodID := transaction.Messages[len(transaction.Messages)-1].MethodID // todo 级联调用需要检查手续费，或者测试两种合约能否级联
	isBVM := methodID == 0 || methodID == 0xFFFFFFFF
	for _, k := range keys {
		transRcpt, _ := app.distributeFee(fees[k], app.rewarder, isDlvOK, isBVM)
		kv := common.KVPair{
			Key:   []byte(fmt.Sprintf("/%d/0/totalFee", len(transaction.Messages))),
			Value: totalFeeReceipts[k],
		}
		tags = append(tags, kv)
		for index, r := range transRcpt {
			kv := common.KVPair{
				Key:   []byte(fmt.Sprintf("/%d/%d/transferFee", len(transaction.Messages), index+1)),
				Value: r,
			}
			tags = append(tags, kv)
		}
	}

	return
}

//distributeFee transfer fee from sender's balance to rewards address, and emit transferFee receipt accordingly
func (app *AppDeliver) distributeFee(fee std.Fee, proposerReward types2.Address, isDlvOK, isBVM bool) (receipts [][]byte, bcerr types2.BcError) {

	if !isDlvOK || isBVM {
		app.logger.Debug("DeliverTx was failed or bvmTx, pay fee from sender")
		//deliverTx fails, transaction be rollback, set sender's balance for Fee
		v := statedbhelper.BalanceOf(app.transID, app.txID, fee.From, fee.Token).SubI(fee.Value)
		statedbhelper.SetBalance(app.transID, app.txID, fee.From, fee.Token, v)
	}

	// Set rewards balance
	leftFee := fee.Value
	for i, reward := range app.rewardStrategy {
		app.logger.Debug("reward strategy", "reward", reward.String())
		addr := reward.Address
		//revard name "validators" writes into genesis file, cannot be modified.
		if reward.Address == "" && reward.Name == "validators" {
			addr = proposerReward
		}
		percent, err := strconv.ParseFloat(reward.RewardPercent, 64)
		if err != nil {
			bcerr.ErrorCode = types2.ErrDeliverTx
			bcerr.ErrorDesc = err.Error()
			app.logger.Error("Get reward percent failed", "error", err)
			return
		}
		// Using Number in case it's overflow by multiple 10000
		award := bn.N(fee.Value).MulI(int64(percent * 100)).DivI(10000)
		//the last reward, set all left Fee to him.
		if i == len(app.rewardStrategy) {
			award = bn.N(leftFee)
		}
		app.logger.Debug("The rewards of Fee", "reward", addr, "percent", percent, "value", award)
		v := statedbhelper.BalanceOf(app.transID, app.txID, addr, fee.Token)
		v = v.Add(award)
		statedbhelper.SetBalance(app.transID, app.txID, addr, fee.Token, v)
		if softforks.V2_0_1_13780(app.blockHeader.Height) {
			// 没有将资产信息添加到账户资产列表
		} else {
			statedbhelper.AddAccountToken(app.transID, app.txID, addr, fee.Token)
		}
		app.logger.Debug("The reward's balance", "reward", addr, "balance", v)

		r := emitTransferReceipt(fee.From, addr, fee.Token, award)
		receipts = append(receipts, r)

		ia, _ := strconv.ParseInt(award.String(), 10, 64)
		app.rewards[addr] = app.rewards[addr] + ia
		leftFee = leftFee - ia
	}
	bcerr.ErrorCode = types2.CodeOK
	return
}

// hasUpdateValidatorReceipt check if there is a receipt for updating validator
func hasUpdateValidatorReceipt(tags []common.KVPair) bool {
	isUpdtValidators := false
	for _, r := range tags {
		if strings.HasSuffix(string(r.Key), "governance.newValidator") ||
			strings.HasSuffix(string(r.Key), "governance.setPower") {
			//if strings.Contains(string(r.Key), "governance") {
			isUpdtValidators = true
		}
	}

	return isUpdtValidators
}

func HasUpdateValidatorReceipt(tags []common.KVPair) bool {
	isUpdtValidators := false
	for _, r := range tags {
		if strings.HasSuffix(string(r.Key), "governance.newValidator") ||
			strings.HasSuffix(string(r.Key), "governance.setPower") {
			//if strings.Contains(string(r.Key), "governance") {
			isUpdtValidators = true
		}
	}

	return isUpdtValidators
}

// packValidators pack validators info when governance contract update validator info
func (app *AppDeliver) packValidators() {
	app.udValidator = true
	var tempVal []types.Validator
	validators := statedbhelper.GetAllValidators(app.transID, app.txID)

	for _, validator := range validators {
		pkBytes := crypto.PubKeyEd25519FromBytes(validator.PubKey).Bytes()
		val := types.Validator{
			PubKey:     pkBytes,
			Power:      uint64(validator.Power),
			RewardAddr: validator.RewardAddr,
			Name:       validator.Name,
		}
		tempVal = append(tempVal, val)

	}
	app.validators = tempVal
	app.logger.Debug("deliverBCTx() update validators", "validators", app.validators)
}

func (app *AppDeliver) PackValidators() {
	app.udValidator = true
	var tempVal []types.Validator
	validators := statedbhelper.GetAllValidators(app.transID, app.txID)

	for _, validator := range validators {
		pkBytes := crypto.PubKeyEd25519FromBytes(validator.PubKey).Bytes()
		val := types.Validator{
			PubKey:     pkBytes,
			Power:      uint64(validator.Power),
			RewardAddr: validator.RewardAddr,
			Name:       validator.Name,
		}
		tempVal = append(tempVal, val)

	}
	app.validators = tempVal
	app.logger.Debug("deliverBCTx() update validators", "validators", app.validators)
}

// hasSideChainGenesisReceipt check if there is a receipt for side chain genesis
func hasSideChainGenesisReceipt(tags []common.KVPair) (common.KVPair, bool) {
	for _, r := range tags {
		if strings.Contains(string(r.Key), "netgovernance.genesisSideChain") {
			return r, true
		}
	}

	return common.KVPair{}, false
}

func HasSideChainGenesisReceipt(tags []common.KVPair) (common.KVPair, bool) {
	for _, r := range tags {
		if strings.Contains(string(r.Key), "netgovernance.genesisSideChain") {
			return r, true
		}
	}

	return common.KVPair{}, false
}

func (app *AppDeliver) packSideChainGenesis(tag common.KVPair) {

	type Validator struct {
		PubKey     types3.PubKey `json:"nodepubkey,omitempty"`  //节点公钥
		Power      int64         `json:"power,omitempty"`       //节点记账权重
		RewardAddr string        `json:"reward_addr,omitempty"` //节点接收奖励的地址
		Name       string        `json:"name,omitempty"`        //节点名称
		NodeAddr   string        `json:"nodeaddr,omitempty"`    //节点地址
	}

	type ContractData struct {
		Name     string          `json:"name"`
		Version  string          `json:"version"`
		CodeByte types3.HexBytes `json:"codeByte"`
	}

	type genesisSideChain struct {
		SideChainID  string         `json:"sideChainID"`
		OpenURLs     []string       `json:"openURLs"`
		GenesisInfo  string         `json:"genesisInfo"`
		ContractData []ContractData `json:"contractData"`
	}

	type genesisInfo struct {
		Validators []Validator `json:"validators"`
	}

	var r std.Receipt
	err := jsoniter.Unmarshal(tag.Value, &r)
	if err != nil {
		panic(err)
	}

	gsc := new(genesisSideChain)
	if err = jsoniter.Unmarshal(r.Bytes, gsc); err != nil {
		panic(err)
	}

	conDatas := make([]types.ContractData, len(gsc.ContractData))
	for i, v := range gsc.ContractData {
		conDatas[i] = types.ContractData{
			Name:     v.Name,
			Version:  v.Version,
			CodeData: v.CodeByte,
		}
	}

	gi := new(genesisInfo)
	if err = jsoniter.Unmarshal(bytes.NewBufferString(gsc.GenesisInfo).Bytes(), gi); err != nil {
		panic(err)
	}

	vals := make([]types.Validator, len(gi.Validators))
	for i, v := range gi.Validators {
		vals[i] = types.Validator{
			PubKey:     v.PubKey,
			Power:      uint64(v.Power),
			RewardAddr: v.RewardAddr,
			Name:       v.Name,
		}
	}

	scg := &types.SideChainGenesis{
		SideChainID:  gsc.SideChainID,
		GenesisInfo:  gsc.GenesisInfo,
		ContractData: conDatas,
		Validators:   vals,
	}
	app.scGenesis = []*types.SideChainGenesis{scg}
}

func (app *AppDeliver) PackSideChainGenesis(tag common.KVPair) {

	type Validator struct {
		PubKey     types3.PubKey `json:"nodepubkey,omitempty"`  //节点公钥
		Power      int64         `json:"power,omitempty"`       //节点记账权重
		RewardAddr string        `json:"reward_addr,omitempty"` //节点接收奖励的地址
		Name       string        `json:"name,omitempty"`        //节点名称
		NodeAddr   string        `json:"nodeaddr,omitempty"`    //节点地址
	}

	type ContractData struct {
		Name     string          `json:"name"`
		Version  string          `json:"version"`
		CodeByte types3.HexBytes `json:"codeByte"`
	}

	type genesisSideChain struct {
		SideChainID  string         `json:"sideChainID"`
		OpenURLs     []string       `json:"openURLs"`
		GenesisInfo  string         `json:"genesisInfo"`
		ContractData []ContractData `json:"contractData"`
	}

	type genesisInfo struct {
		Validators []Validator `json:"validators"`
	}

	var r std.Receipt
	err := jsoniter.Unmarshal(tag.Value, &r)
	if err != nil {
		panic(err)
	}

	gsc := new(genesisSideChain)
	if err = jsoniter.Unmarshal(r.Bytes, gsc); err != nil {
		panic(err)
	}

	conDatas := make([]types.ContractData, len(gsc.ContractData))
	for i, v := range gsc.ContractData {
		conDatas[i] = types.ContractData{
			Name:     v.Name,
			Version:  v.Version,
			CodeData: v.CodeByte,
		}
	}

	gi := new(genesisInfo)
	if err = jsoniter.Unmarshal(bytes.NewBufferString(gsc.GenesisInfo).Bytes(), gi); err != nil {
		panic(err)
	}

	vals := make([]types.Validator, len(gi.Validators))
	for i, v := range gi.Validators {
		vals[i] = types.Validator{
			PubKey:     v.PubKey,
			Power:      uint64(v.Power),
			RewardAddr: v.RewardAddr,
			Name:       v.Name,
		}
	}

	scg := &types.SideChainGenesis{
		SideChainID:  gsc.SideChainID,
		GenesisInfo:  gsc.GenesisInfo,
		ContractData: conDatas,
		Validators:   vals,
	}
	app.scGenesis = []*types.SideChainGenesis{scg}
}

//gatherFeesByFromAddr loop up tags of receipts and find out Fee receipts for each "From" address,
// and return fees' receipts also
func gatherFeesByFromAddr(tags []common.KVPair, isDlvOK bool) (fees map[types2.Address]std.Fee, feeTags []common.KVPair, totalFee int64) {
	fees = make(map[types2.Address]std.Fee)
	feeTags = make([]common.KVPair, 0)
	for _, t := range tags {
		if strings.Contains(string(t.Key), "std::fee") {
			receipt := std.Receipt{}
			err := jsoniter.Unmarshal(t.Value, &receipt)
			if err != nil {
				panic(err)
			}
			rf := std.Fee{}
			err = jsoniter.Unmarshal(receipt.Bytes, &rf)
			if err != nil {
				panic(err)
			}
			totalFee += rf.Value
			if v, ok := fees[rf.From]; ok {
				rf.Value = v.Value + rf.Value
			}
			fees[rf.From] = rf

			if !isDlvOK {
				//fill original fee receipt in
				feeTags = append(feeTags, t)
			}
		}
	}

	return
}

//emitTotalFeeReceipt generate new total Fee receipt for each of sender
func emitTotalFeeReceipt(fees map[types2.Address]std.Fee) (map[types2.Address][]byte, error) {
	bbrs := make(map[types2.Address][]byte)
	for _, fee := range fees {
		rbyte, err := jsoniter.Marshal(fee)
		if err != nil {
			return nil, err
		}

		receipt := types2.Receipt{
			Name:         "totalFee",
			ReceiptBytes: rbyte,
			ReceiptHash:  nil,
		}
		receipt.ReceiptHash = sha3.Sum256([]byte(receipt.Name), rbyte)

		br, err := jsoniter.Marshal(receipt)
		if err != nil {
			return nil, err
		}
		bbrs[fee.From] = br
	}

	return bbrs, nil
}

func emitTransferReceipt(sender, to, tokenAddr types2.Address, value bn.Number) []byte {
	trans := std.Transfer{
		Token: tokenAddr,
		From:  sender,
		To:    to,
		Value: value,
	}

	bz, err := jsoniter.Marshal(trans)
	if err != nil {
		return nil
	}
	receipt := types2.Receipt{
		Name:         "transferFee",
		ReceiptBytes: bz,
		ReceiptHash:  nil,
	}

	receipt.ReceiptHash = sha3.Sum256([]byte(receipt.Name), bz)
	bz, err = jsoniter.Marshal(receipt)
	if err != nil {
		return nil
	}
	return bz
}

func combineBuffer(nonceBuffer, txBuffer map[string][]byte) map[string][]byte {
	if txBuffer == nil {
		txBuffer = make(map[string][]byte)
	}

	for k, v := range nonceBuffer {
		txBuffer[k] = v
	}

	return txBuffer
}

func (app *AppDeliver) RunExecTx(tx *statedb.Tx, params ...interface{}) (doneSuccess bool, response interface{}) {

	//doneSuccess = true
	txHash := params[0].(common.HexBytes)
	transaction := params[1].(types2.Transaction)
	sender := params[2].(types2.Address)
	pubKey := params[3].(crypto.PubKeyEd25519)
	app.logger.Info("Recv ABCI interface: DeliverTx", "tx", tx.ID(), "txHash", txHash.String())

	adp := adapter.GetInstance()
	invokeRes := adp.InvokeTx(app.blockHeader, app.transID, tx.ID(), sender, transaction, pubKey.Bytes(), txHash, app.blockHash)
	if invokeRes.Code != types2.CodeOK {
		app.logger.Error("docker invoke error.....", "error", invokeRes.Log)
		app.logger.Debug("docker invoke error.....", "response", invokeRes.String())
		statedbhelper.RollbackTx(app.transID, tx.ID())
		adp.RollbackTx(app.transID, tx.ID())
	}
	app.logger.Debug("docker invoke response.....", "response", invokeRes.String())

	response = new(types2.Response)
	resDeliverTx := response.(*types2.Response)
	resDeliverTx.Code = invokeRes.Code
	resDeliverTx.Log = invokeRes.Log
	resDeliverTx.GasLimit = invokeRes.GasLimit
	resDeliverTx.GasUsed = invokeRes.GasUsed
	resDeliverTx.Fee = invokeRes.Fee
	resDeliverTx.Data = invokeRes.Data
	resDeliverTx.Tags = invokeRes.Tags
	resDeliverTx.Height = invokeRes.Height
	resDeliverTx.TxHash = invokeRes.TxHash

	return true, resDeliverTx
}

func (app *AppDeliver) HandleResponse(tx *statedb.Tx, txStr string, rawTxV2 *types2.Transaction, response *types2.Response) (resDeliverTx types.ResponseDeliverTx) {
	app.SetTxID(tx.ID())
	if response.Code == types2.ErrDeliverTx {
		resDeliverTx.Code = response.Code
		resDeliverTx.Log = response.Log
		return resDeliverTx
	}
	if response.Code != types2.CodeOK {
		var totalFee int64
		resDeliverTx, _, totalFee = app.reportInvokeFailure([]byte(txStr), *rawTxV2, response)
		resDeliverTx.Fee = uint64(totalFee)
		return resDeliverTx
	}

	// pack validators if update validator info
	if hasUpdateValidatorReceipt(response.Tags) {
		app.packValidators()
	}

	// pack side chain genesis info
	if t, ok := hasSideChainGenesisReceipt(response.Tags); ok {
		app.packSideChainGenesis(t)
	}
	//emit new summary fee  and transferFee receipts
	tags, totalFee := app.EmitFeeReceipts(*rawTxV2, response, true)

	resDeliverTx.Code = response.Code
	resDeliverTx.Log = response.Log
	resDeliverTx.Tags = tags
	resDeliverTx.GasLimit = uint64(rawTxV2.GasLimit)
	resDeliverTx.GasUsed = uint64(response.GasUsed)
	resDeliverTx.Fee = uint64(totalFee)
	resDeliverTx.Data = response.Data
	resDeliverTxStr := resDeliverTx.String()
	app.logger.Debug("deliverBCTx()", "resDeliverTx length", len(resDeliverTxStr), "resDeliverTx", resDeliverTxStr) // log value of async instance must be immutable to avoid data race

	stateTx, _ := statedbhelper.CommitTx(app.transID, tx.ID())
	//stateTx, _ := tx.GetBuffer()
	app.logger.Debug("deliverBCTx() ", "stateTx length", len(stateTx), "stateTx ", string(stateTx))

	app.calcDeliverHash([]byte(txStr), &resDeliverTx, stateTx)
	//calculate Fee
	app.fee = app.fee + response.Fee
	app.logger.Debug("deliverBCTx()", "app.fee", app.fee, "app.rewards", map2String(app.rewards))

	app.logger.Debug("end deliver invoke.....")

	return resDeliverTx
}
