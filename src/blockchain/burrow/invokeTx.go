package burrow

import (
	"blockchain/burrow/burrowrpc"
	"blockchain/burrow/receipt"
	"blockchain/common/statedbhelper"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/jsoniter"
	"blockchain/smcsdk/sdk/rlp"
	"blockchain/smcsdk/sdk/std"
	"blockchain/types"
	sysbin "encoding/binary"
	"encoding/hex"
	"errors"
	crypto2 "github.com/hyperledger/burrow/crypto"
	"github.com/tendermint/go-wire/data/base58"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"

	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/execution/evm"
	types2 "github.com/tendermint/abci/types"
	"github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"
	"strings"
)

const (
	GasBvmRatio = 1
	GasPerByte  = 200
)

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

	gasPrice := receipt.GetGasPrice(transId, txId)
	result = new(types.Response)
	if IsCreate(tx.Messages) {
		bu.logger.Debug("evm: creating...")
		result = bu.Create(blockHeader, blockHash, transId, txId, sender, tx, pubKey, gasPrice)
		if e := checkBalanceForFee(transId, txId, result); e != nil {
			result.Code = types.ErrFeeNotEnough
			result.Log = e.Error()
			result.Data = ""
		}
		return

	} else if IsCall(tx.Messages) {
		bu.logger.Debug("evm: call...")
		st := NewState(transId, txId, bu.logger)
		contractAddr := tx.Messages[0].Contract
		bu.logger.Debug("evm:", "contractAddr", contractAddr)

		account, err := st.GetAccount(crypto2.ToEVM(contractAddr))
		if err != nil {
			result = new(types.Response)
			result.Code = types.ErrCodeEVMInvoke
			result.Log = err.Error()
			return
		}
		if account == nil {
			result = new(types.Response)
			result.Code = types.ErrCodeEVMInvoke
			result.Log = "no such account"
			return
		}
		bu.logger.Debug("evm:", "contractToken", account.EVMToken, "code", hex.EncodeToString(account.EVMCode))
		st.SetToken(account.EVMToken)
		state := evm.NewState(st, blockHashGetter)
		result = bu.Call(state, blockHeader, blockHash, gasPrice, sender, account.EVMCode, tx, bn.N(0))
		result.Tags = *receipt.Tags2Receipt(bu.logger, &bu.Tags, result.GasUsed*gasPrice, account.EVMToken, contractAddr, sender)

		if e := checkBalanceForFee(transId, txId, result); e != nil {
			result.Code = types.ErrFeeNotEnough
			result.Log = e.Error()
			result.Data = ""
		}
		return

	} else if IsCascadeCall(tx.Messages) {
		bu.logger.Debug("evm: cascadeCall...")
		result = bu.CascadeCall(blockHeader, blockHash, transId, txId, sender, tx, gasPrice)
		if e := checkBalanceForFee(transId, txId, result); e != nil {
			result.Code = types.ErrFeeNotEnough
			result.Log = e.Error()
			result.Data = ""
		}
		return
	} else {
		result.Code = types.ErrCodeEVMInvoke
		return
	}
}

// Create of EVM contract create
func (bu *Burrow) Create(blockHeader types2.Header, blockHash []byte, transId, txId int64, sender types.Address, tx types.Transaction, pubKey types.PubKey, gasPrice int64) (result *types.Response) {
	bu.logger.Debug("evm:", "transId", transId, "txId", txId, "gas", tx.GasLimit)
	bu.logger.Debug("evm:", "tx", tx.String())
	gas := uint64(tx.GasLimit) * GasBvmRatio
	result = new(types.Response)
	nonce := make([]byte, 8)
	sysbin.BigEndian.PutUint64(nonce, tx.Nonce)

	st := NewState(transId, txId, bu.logger)
	token := tx.Messages[0].Contract
	st.SetToken(token)
	bu.logger.Debug("evm:", "create contract token=", token)
	state := evm.NewState(st, blockHashGetter)

	var code []byte
	err := rlp.DecodeBytes(tx.Messages[0].Items[0], &code)
	if err != nil {
		result.Code = types.ErrRlpDecode
		result.Log = err.Error()
		return
	}
	bu.logger.Debug("evm:", "code", code)

	contractAddr := CalcContractAddr(sender, tx.Nonce, blockHeader.ChainID)
	evmAddr := crypto2.ToEVM(contractAddr)
	bu.logger.Debug("evm:", "contractAddr", contractAddr, "evmAddr", evmAddr)
	state.CreateAccount(evmAddr)
	bu.logger.Debug("evm", "contractAccount created=", evmAddr)

	senderEVMAddr := crypto2.ToEVM(sender)
	ourEvm := evm.NewVM(newParams(blockHeader, blockHash, gasPrice, tx.GasLimit), senderEVMAddr, nonce, bu.logger)
	bu.logger.Debug("evm:", "new evm", "succeed")
	out, err := ourEvm.Call(state, evm.NewBcEventSink(bu.logger, &bu.Tags), senderEVMAddr, evmAddr, code, nil, bn.N(0), &gas)
	if err != nil {
		result.Code = types.ErrCodeEVMInvoke
		result.Log = err.Error()
		return
	}

	state.InitCode(evmAddr, out)

	if err2 := state.Sync(); err2 != nil {
		result.Code = types.ErrCodeEVMInvoke
		result.Log = err2.String()
		return
	}

	result.Code = types.CodeOK
	result.Data = contractAddr
	result.GasUsed = (tx.GasLimit*GasBvmRatio - int64(gas)) / GasBvmRatio
	bu.logger.Debug("evm:", "Create-GasUsed: ", result.GasUsed)

	gasForCreate := int64(GasPerByte * len(out))
	result.Tags = *receipt.Tags2Receipt(bu.logger, &bu.Tags,
		(result.GasUsed+gasForCreate)*gasPrice,
		token, contractAddr, sender)

	return
}

// Call of EVM contract contract call
func (bu *Burrow) Call(state *evm.State, blockHeader types2.Header, blockHash []byte, gasPrice int64, sender types.Address, code []byte, tx types.Transaction, value bn.Number) (result *types.Response) {
	bu.logger.Debug("evm:", "tx", tx.String())
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

	bu.logger.Debug("evm:", "input", hex.EncodeToString(input))

	senderEVMAddr := crypto2.ToEVM(sender)
	ourEvm := evm.NewVM(newParams(blockHeader, blockHash, gasPrice, tx.GasLimit), senderEVMAddr, nonce, bu.logger)
	out, err := ourEvm.Call(state, evm.NewBcEventSink(bu.logger, &bu.Tags), senderEVMAddr, crypto2.ToEVM(tx.Messages[0].Contract), code, input, value, &gas)
	if err != nil {
		result.Code = types.ErrCodeEVMInvoke
		result.Log = err.Error()
		return
	}

	if err2 := state.Sync(); err2 != nil {
		result.Code = types.ErrCodeEVMInvoke
		result.Log = err2.String()
		return
	}

	result.Code = types.CodeOK
	result.Data = hex.EncodeToString(out)
	result.GasUsed = (tx.GasLimit*GasBvmRatio - int64(gas)) / GasBvmRatio
	bu.logger.Debug("evm:", "Call-GasUsed: ", result.GasUsed)

	return
}

// CascadeCall of EVM contract Cascade Call
func (bu *Burrow) CascadeCall(blockHeader types2.Header, blockHash []byte, transId, txId int64, sender types.Address, tx types.Transaction, gasPrice int64) (result *types.Response) {
	contractAddr := tx.Messages[1].Contract
	bu.logger.Debug("evm:", "contractAddr", contractAddr)
	st := NewState(transId, txId, bu.logger)
	account, err := st.GetAccount(crypto2.ToEVM(contractAddr))
	if err != nil {
		result = new(types.Response)
		result.Code = types.ErrCodeEVMInvoke
		result.Log = err.Error()
		return
	}
	if account == nil {
		result = new(types.Response)
		result.Code = types.ErrCodeEVMInvoke
		result.Log = "no such account"
		return
	}
	bu.logger.Debug("evm:", "contractToken", account.EVMToken, "code", hex.EncodeToString(account.EVMCode))
	st.SetToken(account.EVMToken)
	state := evm.NewState(st, blockHashGetter)

	var value []byte
	err = rlp.DecodeBytes(tx.Messages[0].Items[0], &value)
	if err != nil {
		result = new(types.Response)
		result.Code = types.ErrRlpDecode
		result.Log = err.Error()
		return
	}
	money := bn.NString(string(value)).MulI(1e9)
	bu.logger.Debug("evm:", "sender", sender, "paid", money)

	// Contract call
	tx.Messages = append(tx.Messages[1:])
	result = bu.Call(state, blockHeader, blockHash, gasPrice, sender, account.EVMCode, tx, money)

	result.Tags = *receipt.Tags2Receipt(bu.logger, &bu.Tags, result.GasUsed*gasPrice, account.EVMToken, contractAddr, sender)

	//tag0 := receipt.Emit(bu.logger,
	//	std.Transfer{CheckTx
	//		Token: st.tokenAddr,
	//		From:  sender,
	//		To:    contractAddr,
	//		Value: money,
	//	}, contractAddr, len(result.Tags))
	//result.Tags = append(result.Tags, *tag0)

	return
}

func newParams(blockHeader types2.Header, blockHash []byte, gasPrice, gasLimit int64) evm.Params {
	return evm.Params{
		BlockHeader:              blockHeader,
		GasLimit:                 uint64(gasLimit),
		BlockHash:                blockHash,
		GasPrice:                 gasPrice,
		CallStackMaxDepth:        0, //todo
		DataStackInitialCapacity: 0,
		DataStackMaxDepth:        0,
	}
}

func IsCreate(messages []types.Message) bool {
	if len(messages) != 1 || messages[0].MethodID != 0 {
		return false
	}

	return true
}

func IsCall(messages []types.Message) bool {
	if len(messages) != 1 || messages[0].MethodID != 0xFFFFFFFF {
		return false
	}

	return true
}

func IsCascadeCall(messages []types.Message) bool {
	if len(messages) != 2 || messages[1].MethodID != 0xFFFFFFFF {
		return false
	}

	return true
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

func CalcContractAddr(senderAddr types.Address, nonce uint64, chainID string) types.Address {

	nonceBin, err := rlp.EncodeToBytes(nonce)
	if err != nil {
		return ""
	}

	hasherSHA3256 := sha3.New256()
	hasherSHA3256.Write([]byte(senderAddr))
	hasherSHA3256.Write(nonceBin)
	sha := hasherSHA3256.Sum(nil)

	hasherRIPEMD160 := ripemd160.New()
	hasherRIPEMD160.Write(sha) // does not error
	rpd := hasherRIPEMD160.Sum(nil)

	hasher := ripemd160.New()
	hasher.Write(rpd)
	md := hasher.Sum(nil)

	addr := make([]byte, 0, len(rpd)+len(md[:4]))
	addr = append(addr, rpd...)
	addr = append(addr, md[:4]...)

	return chainID + base58.Encode(addr)
}

func checkBalanceForFee(transId, txID int64, response *types.Response) error {
	fee := getFee(response.Tags)
	senderBal := statedbhelper.BalanceOf(transId, txID, fee.From, fee.Token)

	if senderBal.IsLessThanI(response.Fee) {
		return errors.New("Insufficient balance to pay fee")
	}
	return nil
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
