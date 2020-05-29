package burrow

import (
	"github.com/bcbchain/bcbchain/burrow/burrowrpc"
	"github.com/bcbchain/bcbchain/burrow/receipt"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/sdk/sdk/bn"
	"github.com/bcbchain/sdk/sdk/jsoniter"
	"github.com/bcbchain/sdk/sdk/rlp"
	"github.com/bcbchain/sdk/sdk/std"
	"github.com/bcbchain/bclib/types"
	sysbin "encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/bcbchain/bcbchain/hyperledger/burrow/binary"
	crypto2 "github.com/bcbchain/bcbchain/hyperledger/burrow/crypto"
	"github.com/bcbchain/bcbchain/hyperledger/burrow/execution/bvm"
	types2 "github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	"strconv"
	"strings"
)

const (
	GasBvmRatio = 1
	GasPerByte  = 200
)

var flag = false

//Burrow object of burrow
type Burrow struct {
	logger log.Logger
	Tags   []interface{}
}

//GetInstance get or create burrow instance
func GetInstance(log log.Logger) *Burrow {
	bu := &Burrow{}
	bu.logger = log
	bu.Tags = make([]interface{}, 0)
	return bu
}

func (bu *Burrow) InvokeTx(blockHeader types2.Header, blockHash []byte, transId, txId int64, sender types.Address, tx types.Transaction, pubKey types.PubKey) (result *types.Response) {
	result = bu.InvokeTxEx(blockHeader, blockHash, transId, txId, sender, tx, pubKey)

	if e := checkBalanceForFee(transId, txId, result); e != nil {
		result.Code = types.ErrFeeNotEnough
		result.Log = e.Error()
		result.Data = ""
	}

	return
}

func (bu *Burrow) InvokeTxEx(blockHeader types2.Header, blockHash []byte, transId, txId int64, sender types.Address, tx types.Transaction, pubKey types.PubKey) (result *types.Response) {

	result = new(types.Response)
	if IsCreate(tx.Messages) {
		bu.logger.Debug("bvm: creating...")
		gasPrice := receipt.GetGasPrice(transId, txId, true)
		result = bu.Create(blockHeader, blockHash, transId, txId, sender, tx, pubKey, gasPrice)

		return

	} else if IsCall(tx.Messages) {
		bu.logger.Debug("bvm: call...")
		gasPrice := receipt.GetGasPrice(transId, txId, false)
		st := NewState(transId, txId, bu.logger)
		contractAddr := tx.Messages[0].Contract
		bu.logger.Debug("bvm:", "contractAddr", contractAddr)
		if len(contractAddr) == 0 {
			result = new(types.Response)
			result.Code = types.ErrCodeBVMInvoke
			result.Log = "contract address should not be empty "
			return
		}

		account, err := st.GetAccount(crypto2.ToBVM(contractAddr))
		if err != nil {
			result = new(types.Response)
			result.Code = types.ErrCodeBVMInvoke
			result.Log = err.Error()
			return
		}
		if account == nil {
			result = new(types.Response)
			result.Code = types.ErrCodeBVMInvoke
			result.Log = "no such account"
			return
		}
		bu.logger.Debug("bvm:", "contractToken", account.BVMToken)
		st.SetToken(account.BVMToken)
		state := bvm.NewState(st, blockHashGetter)
		result = bu.Call(state, blockHeader, blockHash, gasPrice, sender, account.BVMCode, tx, bn.N(0))
		tags, err := receipt.Tags2Receipt(bu.logger, &bu.Tags, transId, txId, result.GasUsed*gasPrice, statedbhelper.GetGenesisToken().Address, contractAddr, sender, "", false)
		if err != nil {
			result = new(types.Response)
			result.Code = types.ErrCodeBVMInvoke
			result.Log = err.Error()
			return
		}

		result.Tags = *tags

		return

	} else if IsCascadeCall(tx.Messages) {
		bu.logger.Debug("bvm: cascadeCall...")
		gasPrice := receipt.GetGasPrice(transId, txId, false)
		result = bu.CascadeCall(blockHeader, blockHash, transId, txId, sender, tx, gasPrice)

		return
	} else {
		result.Code = types.ErrCodeBVMInvoke
		result.Log = "invalid call"
		return
	}
}

// Create of BVM contract create
func (bu *Burrow) Create(blockHeader types2.Header, blockHash []byte, transId, txId int64, sender types.Address, tx types.Transaction, pubKey types.PubKey, gasPrice int64) (result *types.Response) {
	bu.logger.Debug("bvm:", "transId", transId, "txId", txId, "gas", tx.GasLimit)
	bu.logger.Debug("bvm:", "tx", tx)
	gas := uint64(tx.GasLimit) * GasBvmRatio
	result = new(types.Response)
	nonce := make([]byte, 8)
	sysbin.BigEndian.PutUint64(nonce, tx.Nonce)

	st := NewState(transId, txId, bu.logger)
	token := tx.Messages[0].Contract

	tokenInfo, _ := statedbhelper.Get(transId, txId, std.KeyOfToken(token))
	if len(tokenInfo) == 0 {
		result.Code = types.ErrInvalidParam
		result.Log = "token addr is not exits"
		return
	}

	st.SetToken(token)
	bu.logger.Debug("bvm:", "create contract token=", token)
	state := bvm.NewState(st, blockHashGetter)

	var code []byte
	err := rlp.DecodeBytes(tx.Messages[0].Items[0], &code)
	if err != nil {
		result.Code = types.ErrInvalidParam
		result.Log = err.Error()
		return
	} else if len(code) == 0 {
		result.Code = types.ErrInvalidParam
		result.Log = "bvm contract code could not be empty"
		return
	}

	contractAddr := bvm.CalcContractAddr(sender, nonce, blockHeader.ChainID)
	BVMAddr := crypto2.ToBVM(contractAddr)
	state.CreateAccount(BVMAddr)
	bu.logger.Debug("bvm:", "contractAddr", contractAddr, "bvmAddr", BVMAddr)

	senderBVMAddr := crypto2.ToBVM(sender)
	ourBVM := bvm.NewVM(newParams(blockHeader, blockHash, gasPrice, tx.GasLimit), senderBVMAddr, nonce, bu.logger)
	out, err := ourBVM.Call(state, bvm.NewBcEventSink(bu.logger, &bu.Tags), senderBVMAddr, BVMAddr, code, nil, bn.N(0), &gas)
	if err != nil {
		result.Code = types.ErrCodeBVMInvoke
		result.Log = err.Error()
		return
	}

	var abiCode []byte
	err = rlp.DecodeBytes(tx.Messages[0].Items[1], &abiCode)
	if err != nil {
		result.Code = types.ErrInvalidParam
		result.Log = err.Error()
		return
	} else if len(abiCode) == 0 {
		result.Code = types.ErrInvalidParam
		result.Log = "contract's ABI info could not be empty"
		return
	}

	abiStr := string(abiCode)
	err = st.SetContractInfo(transId, txId, blockHeader.ChainVersion, token, crypto2.ToAddr(BVMAddr), crypto2.ToAddr(senderBVMAddr), abiStr)
	if err != nil {
		result.Code = types.ErrCodeBVMInvoke
		result.Log = err.Error()
		return
	}

	state.InitCode(BVMAddr, out)

	if err2 := state.Sync(); err2 != nil {
		result.Code = types.ErrCodeBVMInvoke
		result.Log = err2.String()
		return
	}

	result.Code = types.CodeOK
	result.Data = contractAddr
	result.GasUsed = (tx.GasLimit*GasBvmRatio - int64(gas)) / GasBvmRatio
	bu.logger.Debug("bvm:", "Create-GasUsed: ", result.GasUsed)

	gasForCreate := int64(GasPerByte * len(out))
	bu.logger.Debug("bvm:", "bu.tags: ", bu.Tags)

	tags, err := receipt.Tags2Receipt(bu.logger, &bu.Tags, transId, txId,
		(result.GasUsed+gasForCreate)*gasPrice,
		statedbhelper.GetGenesisToken().Address, contractAddr, sender, abiStr, false)
	if err != nil {
		result = new(types.Response)
		result.Code = types.ErrCodeBVMInvoke
		result.Log = err.Error()
		return
	}

	result.Tags = *tags

	return
}

// Call of BVM contract contract call
func (bu *Burrow) Call(state *bvm.State, blockHeader types2.Header, blockHash []byte, gasPrice int64, sender types.Address, code []byte, tx types.Transaction, value bn.Number) (result *types.Response) {
	bu.logger.Debug("bvm:", "tx", tx)
	gas := uint64(tx.GasLimit) * GasBvmRatio
	result = new(types.Response)
	nonce := make([]byte, 8)
	sysbin.BigEndian.PutUint64(nonce, 0)

	var input []byte
	err := rlp.DecodeBytes(tx.Messages[0].Items[0], &input)
	if err != nil {
		result.Code = types.ErrRlpDecode
		result.Log = err.Error()
		return
	}

	bu.logger.Debug("bvm:", "input", hex.EncodeToString(input))

	senderBVMAddr := crypto2.ToBVM(sender)
	ourBVM := bvm.NewVM(newParams(blockHeader, blockHash, gasPrice, tx.GasLimit), senderBVMAddr, nonce, bu.logger)
	out, err := ourBVM.Call(state, bvm.NewBcEventSink(bu.logger, &bu.Tags), senderBVMAddr, crypto2.ToBVM(tx.Messages[0].Contract), code, input, value, &gas)

	if err != nil {
		result.Code = types.ErrCodeBVMInvoke
		result.Log = err.Error()
		return
	}

	if err2 := state.Sync(); err2 != nil {
		result.Code = types.ErrCodeBVMInvoke
		result.Log = err2.String()
		return
	}

	methodID := input[:4]
	isView, outData, err := JudgeFuncType(bu.logger, tx.Messages[0].Contract, methodID, out)
	if err != nil {
		result.Code = types.ErrCodeBVMInvoke
		result.Log = err.Error()
		return
	}

	if isView {
		result.Code = types.CodeBVMQueryOK
		result.Data = string(outData)
		result.GasUsed = 0

	} else {
		result.Code = types.CodeOK
		result.Data = string(outData)
		result.GasUsed = (tx.GasLimit*GasBvmRatio - int64(gas)) / GasBvmRatio
		bu.logger.Debug("bvm:", "Call-GasUsed: ", result.GasUsed)
	}

	return
}

// CascadeCall of BVM contract Cascade Call
func (bu *Burrow) CascadeCall(blockHeader types2.Header, blockHash []byte, transId, txId int64, sender types.Address, tx types.Transaction, gasPrice int64) (result *types.Response) {
	contractAddr := tx.Messages[1].Contract
	bu.logger.Debug("bvm:", "contractAddr", contractAddr)
	st := NewState(transId, txId, bu.logger)
	account, err := st.GetAccount(crypto2.ToBVM(contractAddr))
	if err != nil {
		result = new(types.Response)
		result.Code = types.ErrCodeBVMInvoke
		result.Log = err.Error()
		return
	}
	if account == nil {
		result = new(types.Response)
		result.Code = types.ErrCodeBVMInvoke
		result.Log = "no such account"
		return
	}

	if tx.Messages[0].Contract != account.BVMToken {
		result = new(types.Response)
		result.Code = types.ErrInvalidParam
		result.Log = "invalid tokenAddr"
		return
	}
	bu.logger.Debug("bvm:", "contractToken", account.BVMToken)
	st.SetToken(account.BVMToken)
	state := bvm.NewState(st, blockHashGetter)

	if tx.Messages[0].MethodID != 0x44d8ca60 {
		result = new(types.Response)
		result.Code = types.ErrInvalidParam
		result.Log = "invalid methodID"
		return
	}

	value := bn.N(0)
	if len(tx.Messages[0].Items) == 2 {
		var to string
		err = rlp.DecodeBytes(tx.Messages[0].Items[0], &to)
		if err != nil {
			result = new(types.Response)
			result.Code = types.ErrRlpDecode
			result.Log = err.Error()
			return
		} else if to != contractAddr {
			result = new(types.Response)
			result.Code = types.ErrInvalidParam
			result.Log = "target address must be contractAddr"
			return
		}

		err = rlp.DecodeBytes(tx.Messages[0].Items[1], &value)
		if err != nil {
			result = new(types.Response)
			result.Code = types.ErrRlpDecode
			result.Log = err.Error()
			return
		}

	} else {
		result = new(types.Response)
		result.Code = types.ErrInvalidParam
		result.Log = "tx format error"
		return
	}

	bu.logger.Debug("bvm:", "sender", sender, "paid", value)

	// Contract call
	tx.Messages = append(tx.Messages[1:])
	result = bu.Call(state, blockHeader, blockHash, gasPrice, sender, account.BVMCode, tx, value)
	tags, err := receipt.Tags2Receipt(bu.logger, &bu.Tags, transId, txId, result.GasUsed*gasPrice, statedbhelper.GetGenesisToken().Address, contractAddr, sender, "", true)
	if err != nil {
		result = new(types.Response)
		result.Code = types.ErrCodeBVMInvoke
		result.Log = err.Error()
		return
	}

	result.Tags = *tags

	return
}

func newParams(blockHeader types2.Header, blockHash []byte, gasPrice, gasLimit int64) bvm.Params {
	return bvm.Params{
		BlockHeader:              blockHeader,
		GasLimit:                 uint64(gasLimit),
		BlockHash:                blockHash,
		GasPrice:                 gasPrice,
		CallStackMaxDepth:        10,
		DataStackInitialCapacity: 0,
		DataStackMaxDepth:        0,
	}
}

func IsCreate(messages []types.Message) bool {
	if len(messages) == 1 && messages[0].MethodID == 0 {
		return true
	}

	return false
}

func IsCall(messages []types.Message) bool {
	if len(messages) == 1 && messages[0].MethodID == 0xFFFFFFFF {
		return true
	}

	return false
}

func IsCascadeCall(messages []types.Message) bool {
	if len(messages) == 2 && messages[1].MethodID == 0xFFFFFFFF {
		return true
	}

	return false
}

func blockHashGetter(height uint64) []byte {
	req := make(map[string]interface{})
	req["height"] = int64(height)
	resp, err := burrowrpc.GetBlock(req)
	if err != nil {
		return nil
	}

	block := new(std.Block)
	err = jsoniter.Unmarshal([]byte(resp.(string)), block)
	if err != nil {
		return nil
	}
	hash := block.BlockHash

	return binary.LeftPadWord256(hash).Bytes()
}

// Determines whether the function call type is view
func JudgeFuncType(log log.Logger, address types.Address, methodID []byte, out []byte) (IsView bool, outData []byte, err error) {

	contract := GetContractInfo(address)

	Abi, err := receipt.GetAbiObject(nil, contract.BvmAbi)
	if err != nil {
		return
	}

	method, err := Abi.MethodById(methodID)
	if err != nil {
		return
	}
	IsView = method.Const

	if len(out) == 0 {
		return
	}

	vMap := make(map[string]interface{})
	vSlice := make([]interface{}, 0)

	err = Abi.UnpackIntoMap(vMap, method.Name, out)
	if err != nil {
		log.Debug("bvm", "JudgeFuncType fail", "Data unpack failed!")
		return
	}

	var result interface{}
	for k, iv := range method.Outputs {
		tMap := receipt.GetTypeMap(iv.Type.String(), iv.Type)
		result, err = receipt.DetermineType(iv.Type.String(), vMap[strconv.Itoa(k)], tMap)
		if err != nil {
			return
		}

		vSlice = append(vSlice, result)
	}

	outData, err = jsoniter.Marshal(vSlice)
	if err != nil {
		return
	}

	return
}

func getFee(tags []common.KVPair) std.Fee {
	for _, t := range tags {
		if strings.Contains(string(t.Key), "std::fee") {
			receipt := std.Receipt{}
			err := jsoniter.Unmarshal(t.Value, &receipt)
			if err != nil {
				panic(err)
			}
			f := std.Fee{}
			err = jsoniter.Unmarshal(receipt.Bytes, &f)
			if err != nil {
				panic(err)
			}
			return f
		}
	}
	//panic("no fee receipt")
	return std.Fee{}
}

func checkBalanceForFee(transId, txID int64, response *types.Response) error {
	fee := getFee(response.Tags)
	senderBal := statedbhelper.BalanceOf(transId, txID, fee.From, fee.Token)
	if senderBal.IsLessThanI(fee.Value) {
		return errors.New("Insufficient balance to pay fee ")
	}
	return nil
}
