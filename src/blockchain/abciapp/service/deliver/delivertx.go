package deliver

//nolint weak
import (
	"blockchain/abciapp/softforks"
	"blockchain/algorithm"
	"blockchain/common/statedbhelper"
	"blockchain/smcrunctl/adapter"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/crypto/sha3"
	"blockchain/smcsdk/sdk/jsoniter"
	"blockchain/smcsdk/sdk/std"
	types3 "blockchain/smcsdk/sdk/types"
	"blockchain/tx2"
	types2 "blockchain/types"
	"bytes"
	"crypto/md5"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/tendermint/abci/types"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/tmlibs/common"
)

func (app *AppDeliver) deliverBCTx(tx []byte) (resDeliverTx types.ResponseDeliverTx, txBuffer map[string][]byte) {

	app.logger.Info("Recv ABCI interface: DeliverTx", "tx", string(tx))
	if app.chainID == "" {
		app.SetChainID(statedbhelper.GetChainID())
	}
	tx2.Init(app.chainID)
	app.txID++
	transaction, pubKey, err := tx2.TxParse(string(tx))
	if err != nil {
		app.logger.Error("tx parse failed:", err)
		return app.reportFailure(tx, types2.ErrDeliverTx, "tx parse failed"), nil
	}
	app.logger.Debug("DELIVER.TX", "height", app.blockHeader.Height, "tx", transaction.String(), "pubKey", pubKey, "addr", pubKey.Address(statedbhelper.GetChainID()))

	return app.runDeliverTx(tx, transaction, pubKey)
}

func (app *AppDeliver) runDeliverTx(tx []byte, transaction types2.Transaction, pubKey crypto.PubKeyEd25519) (resDeliverTx types.ResponseDeliverTx, txBuffer map[string][]byte) {
	resDeliverTx.Code = types2.CodeOK

	if len(transaction.Note) > types2.MaxSizeNote {
		return app.reportFailure(tx, types2.ErrDeliverTx, "tx note is out of range"), nil
	}

	nonceBuffer, err := statedbhelper.SetAccountNonce(app.transID, app.txID, pubKey.Address(statedbhelper.GetChainID()), transaction.Nonce)
	if err != nil {
		app.logger.Error("SetAccountNonce failed:", err)
		return app.reportFailure(tx, types2.ErrDeliverTx, "SetAccountNonce failed"), nil
	}

	txHash := common.HexBytes(algorithm.CalcCodeHash(string(tx)))
	adp := adapter.GetInstance()
	response := adp.InvokeTx(app.blockHeader, app.transID, app.txID, pubKey.Address(statedbhelper.GetChainID()), transaction, pubKey.Bytes(), txHash, app.blockHash)
	response.Fee = gatherFees(response.Tags)
	if response.Code != types2.CodeOK {
		app.logger.Error("docker invoke error.....", "error", response.Log)
		app.logger.Debug("docker invoke error.....", "response", response.String())
		statedbhelper.RollbackTx(app.transID, app.txID)
		adp.RollbackTx(app.transID, app.txID)
		resDeliverTx, txBuffer = app.reportInvokeFailure(tx, transaction, response)
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
	tags := app.emitFeeReceipts(transaction, response, true)

	resDeliverTx.Code = response.Code
	resDeliverTx.Log = response.Log
	resDeliverTx.Tags = tags
	resDeliverTx.GasLimit = uint64(transaction.GasLimit)
	resDeliverTx.GasUsed = uint64(response.GasUsed)
	resDeliverTx.Fee = uint64(response.Fee)
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

func (app *AppDeliver) reportFailure(tx []byte, errorCode uint32, msg string) (response types.ResponseDeliverTx) {
	response.Code = errorCode
	response.Log = msg
	app.calcDeliverHash(tx, &response, nil)
	return
}

func (app *AppDeliver) reportInvokeFailure(tx []byte, transaction types2.Transaction, response *types2.Response) (resDeliverTx types.ResponseDeliverTx, txBuffer map[string][]byte) {

	rcpts := app.emitFeeReceipts(transaction, response, false)
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

func (app *AppDeliver) emitFeeReceipts(transaction types2.Transaction, response *types2.Response, isDlvOK bool) (tags []common.KVPair) {
	fees, feetags := gatherFeesByFromAddr(response.Tags, isDlvOK)
	app.logger.Debug("get fee receipts", "receipts", mapFee2String(fees))
	totalFeeReceipts, err := emitTotalFeeReceipt(fees)
	if err != nil {
		app.logger.Error("emit fee receipt failed", "error", err.Error())
		return nil
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
	return tags
}

//distributeFee transfer fee from sender's balance to rewards address, and emit transferFee receipt accordingly
func (app *AppDeliver) distributeFee(fee std.Fee, proposerReward types2.Address, isDlvOK, isBVM bool) (receipts [][]byte, bcerr types2.BcError) {

	if !isDlvOK || isBVM {
		app.logger.Debug("DeliverTx was failed, pay fee from sender")
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

// hasSideChainGenesisReceipt check if there is a receipt for side chain genesis
func hasSideChainGenesisReceipt(tags []common.KVPair) (common.KVPair, bool) {
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
	app.logger.Info("侧链创世收据：", r.Name)

	gsc := new(genesisSideChain)
	if err = jsoniter.Unmarshal(r.Bytes, gsc); err != nil {
		panic(err)
	}
	app.logger.Info("侧链registerSideChain sideChainID：", gsc.SideChainID)
	app.logger.Info("侧链registerSideChain OpenURLs：", gsc.OpenURLs)

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
	app.logger.Info("侧链创世返回给 tendermint 的 validator ：", scg.Validators)
	app.scGenesis = []*types.SideChainGenesis{scg}
}

//gatherFeesByFromAddr loop up tags of receipts and find out Fee receipts for each "From" address,
// and return fees' receipts also
func gatherFeesByFromAddr(tags []common.KVPair, isDlvOK bool) (fees map[types2.Address]std.Fee, feeTags []common.KVPair) {
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

//gatherFees loop up tags of receipts and find out Fee receipts,
// and return totalFee
func gatherFees(tags []common.KVPair) (totalFee int64) {
	for _, t := range tags {
		if strings.Contains(string(t.Key), "std::fee") {
			receipt := std.Receipt{}
			err := jsoniter.Unmarshal(t.Value, &receipt)
			if err != nil {
				return totalFee
			}
			rf := std.Fee{}
			err = jsoniter.Unmarshal(receipt.Bytes, &rf)
			if err != nil {
				return totalFee
			}

			totalFee += rf.Value
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
