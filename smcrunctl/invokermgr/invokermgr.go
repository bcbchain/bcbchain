package invokermgr

import (
	"github.com/bcbchain/bcbchain/abciapp/softforks"
	"github.com/bcbchain/bclib/algorithm"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bcbchain/smcdocker"
	"github.com/bcbchain/sdk/sdk/jsoniter"
	"github.com/bcbchain/sdk/sdk/std"
	"github.com/bcbchain/bclib/socket"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	types2 "github.com/bcbchain/bclib/tendermint/abci/types"

	"github.com/bcbchain/bclib/types"

	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
)

// UrlMap - url map
type UrlMap struct {
	Map map[string]struct{}
}

// TxID2UrlMap - txID to urls
type TxID2UrlMap struct {
	Map map[int64][]string
}

type TxID2ContractAddrMap struct {
	Map map[int64][]string
}

// InvokerMgr - class of invoke manager
type InvokerMgr struct {
	logger log.Logger

	standardMethods       map[uint32]struct{}
	transMap              sync.Map // map[transID]map[txID]urls
	dockerUrlMap          sync.Map // map[transID]map[url]struct{}
	dockerMapConnPool     sync.Map // map[url]*socket.ConnectionPool
	transIDToContractAddr sync.Map // map[transID]map[txID][]types.Address
	contractBuffer        sync.Map // map[contractAddr_methodID]gas/map[contract]acctAddr/map[contractToken]token
}

var (
	mgr          *InvokerMgr // instance of manager
	instanceOnce sync.Once
	initOnce     sync.Once
)

// GetInstance - construct manager and return it
func GetInstance() *InvokerMgr {
	instanceOnce.Do(func() {
		mgr = &InvokerMgr{}
	})

	return mgr
}

// Init - init manager members
func (im *InvokerMgr) Init(log log.Logger) {
	initOnce.Do(func() {
		im.logger = log
		im.standardMethods = map[uint32]struct{}{
			0x44D8CA60: {}, // prototype: Transfer(types.Address,bn.Number)
			0x6B7E4ED5: {}, // prototype: AddSupply(bn.Number)
			0xFBBD9DD3: {}, // prototype: Burn(bn.Number)
			0x810B995F: {}, // prototype: SetOwner(types.Address)
			0x9024DC9B: {}, // prototype: SetGasPrice(int64)
		}
	})
}

// DirtyURL - when the docker closed, then dirty url from map
func (im *InvokerMgr) DirtyURL(url string) {
	im.dockerMapConnPool.Delete(url)

	im.dockerUrlMap.Range(func(key, value interface{}) bool {
		urlMap := value.(*UrlMap)
		delete(urlMap.Map, url)
		return true
	})

	im.transMap.Range(func(key, value interface{}) bool {
		transMap := value.(*TxID2UrlMap)

		for txID, urls := range transMap.Map {
			for i, v := range urls {
				if v == url {
					urls = append(urls[:i], urls[i+1:]...)
				}
			}
			transMap.Map[txID] = urls
		}

		return true
	})
}

// CallMcDirtyTx - dirty tx if it failed
func (im *InvokerMgr) CallMcDirtyTx(urls []string, transId, txId int64) {

	for _, url := range urls {
		im.logger.Debug(url)
		pool := im.dockerConnPool(transId, url)
		cli, err := pool.GetClient()
		if err != nil {
			panic(err)
		}

		result, err := cli.Call("McDirtyTransTx", map[string]interface{}{"transID": transId, "txID": txId}, 60)
		if err != nil {
			panic(err)
		}
		pool.ReleaseClient(cli)

		if !result.(bool) {
			panic("CallMcDirtyTx result is false")
		}
	}
}

// InvokeTx - invoke tx's message one by one
func (im *InvokerMgr) InvokeTx(
	blockHeader types2.Header,
	transId, txId int64,
	sender types.Address,
	tx types.Transaction,
	pubKey types.PubKey,
	txHash types.Hash,
	blockHash types.Hash) (result *types.Response) {

	//从tx中解析出多个Message，InvokeMessage
	receipts := make([]common.KVPair, 0)
	result = new(types.Response)
	var gasUsed, fee int64
	var url string
	var err types.BcError
	urls := make([]string, 0, len(tx.Messages))
	addrOfNewContract := make([]string, 0, len(tx.Messages))
	for index, message := range tx.Messages {
		payer, e := im.getPayer(blockHeader.Height, sender, tx, index)
		if e != nil {
			result.Code = types.ErrLogicError
			result.Log = e.Error()
			return
		}

		url, result, err = im.invoke(blockHeader, transId, txId, tx.GasLimit-gasUsed, sender, payer, tx, message, result.Tags, pubKey, txHash, blockHash)

		// 无论失败与成功，均将收据和Fee等数据返回给调用者
		// 调用者是 checker将会把无用数据丢弃， deliver根据手续费收据从发送者账户扣除手续费
		if len(urls) != 0 && !inSlice(url, urls) {
			urls = append(urls, url)
		}
		// 在收据前部加上message序号
		for _, tag := range result.Tags {
			tag.Key = []byte(fmt.Sprintf("/%d%s", index, tag.Key))
			receipts = append(receipts, tag)
		}
		gasUsed = result.GasUsed
		fee = fee + result.Fee
		if result.Code != types.CodeOK || err.ErrorCode != types.CodeOK {
			//调用rpc清除该tx影响的Message的缓存
			if len(urls) != 0 {
				im.CallMcDirtyTx(urls, transId, txId)
			}

			break //跳出，不执行级联交易的下一条
		}

		newContractAddr := im.cleanDockerCache(result)
		if newContractAddr != "" {
			addrOfNewContract = append(addrOfNewContract, newContractAddr)
		}
	}

	result.GasLimit = tx.GasLimit
	result.Fee = fee
	result.GasUsed = gasUsed
	result.Tags = receipts

	// 记录调用过的url
	im.setValToTransMap(transId, txId, urls)
	if len(addrOfNewContract) > 0 {
		im.setValToTransCon(transId, txId, addrOfNewContract)
	}

	// if height in [23706999, forkHeight] then reset gas_used
	if softforks.V2_0_2_14654(blockHeader.Height) {
		im.resetGasUsed(blockHeader.Height, result, tx)
	}

	return
}

func inSlice(item string, s []string) bool {
	for _, v := range s {
		if item == v {
			return true
		}
	}

	return false
}

// TransferID - methodID of standard transfer method
var TransferID = algorithm.BytesToUint32(algorithm.CalcMethodId("Transfer(types.Address,bn.Number)"))

// invoke - invoke message in tx
func (im *InvokerMgr) invoke(
	blockHeader types2.Header,
	transId, txId, gasLeft int64,
	sender, payer types.Address,
	tx types.Transaction,
	message types.Message,
	receipts []common.KVPair,
	pubKey types.PubKey,
	txHash types.Hash,
	blockHash types.Hash) (url string, result *types.Response, error types.BcError) {

	error.ErrorCode = types.CodeOK
	tx.Messages = nil

	result = &types.Response{}
	var to types.Address

	//进行rpc调用
	contractAddr, url, err := smcdocker.GetInstance().GetContractInvokeURL(transId, txId, message.Contract)
	if err != nil {
		error.ErrorCode = types.ErrInternalFailed
		error.ErrorDesc = err.Error()

		result.Code = types.ErrInternalFailed
		result.Log = err.Error()
		im.logger.Error("GetContractInvokeURL()", "error", err.Error())
		return
	}
	message.Contract = contractAddr

	// 创世时不需要执行如下代码
	if message.Contract != std.GetGenesisContractAddr(statedbhelper.GetChainID()) {
		contract, e := im.getEffectContract(transId, txId, blockHeader.Height, message.Contract, message.MethodID)
		if e.ErrorCode != types.CodeOK {
			error.ErrorCode = e.ErrorCode
			error.ErrorDesc = e.ErrorDesc
			result.Code = e.ErrorCode
			result.Log = e.ErrorDesc
			return
		}
		message.Contract = contract.Address
	}

	//构造rpc参数
	invokeParam := types.RPCInvokeCallParam{
		Sender:          sender,
		Payer:           payer,
		To:              to,
		Tx:              tx,
		GasLeft:         gasLeft,
		TxHash:          txHash,
		Message:         message,
		Receipts:        receipts,
		SenderPublicKey: pubKey}
	if softforks.V2_0_1_13780(blockHeader.Height) {
		invokeParam.BlockHash = nil
	} else {
		invokeParam.BlockHash = blockHash
	}

	im.logger.Debug("GetContractInvokeURL", "url", url)
	im.logger.Trace("rpcCallInvoke", "invokeParamData", invokeParam)
	pool := im.dockerConnPool(transId, url)
	cli, err := pool.GetClient()
	if err != nil {
		panic(err)
	}
	defer pool.ReleaseClient(cli)

	timeout := time.Duration(160)
	if message.Contract == std.GetGenesisContractAddr(statedbhelper.GetChainID()) {
		timeout = 300
	}
	resp, err := cli.Call("Invoke", map[string]interface{}{"blockHeader": blockHeader, "transID": transId, "txID": txId, "callParam": invokeParam}, timeout)
	if err != nil {
		im.logger.Info("Client call error: " + err.Error())
		// 之前有在失败时再重试一次，现在改为失败了直接 panic
		panic(err)
	}
	result = new(types.Response)
	err = jsoniter.Unmarshal([]byte(resp.(string)), result)
	if err != nil {
		panic(err)
	}
	if message.Contract == std.GetGenesisContractAddr(statedbhelper.GetChainID()) {
		smcdocker.GetInstance().DirtyContractInvokeURL(0, 0, message.Contract)
	}
	im.logger.Debug("rpcCallInvoke", "returned code", result.Code)
	return
}

// Rollback - rollback transaction's data when it failed
func (im *InvokerMgr) Rollback(transID int64) {
	//依次获取url，进行rollback
	if v, ok := im.dockerUrlMap.Load(transID); ok {
		urlMap := v.(*UrlMap).Map
		for url := range urlMap {
			pool := im.dockerConnPool(transID, url)
			cli, err := pool.GetClient()
			if err != nil {
				panic(err)
			}

			_, err = cli.Call("McDirtyTrans", map[string]interface{}{"transID": transID}, 60)
			if err != nil {
				panic(err)
			}
			pool.ReleaseClient(cli)
		}
	}

	im.dockerUrlMap.Delete(transID)
	im.transMap.Delete(transID)
	im.transIDToContractAddr.Delete(transID)
}

// RollbackTx - rollback tx's data when it failed
func (im *InvokerMgr) RollbackTx(transID, txID int64) {
	v1, ok := im.transMap.Load(transID)
	if !ok {
		return
	}

	vEx := v1.(*TxID2UrlMap).Map
	urls, ok := vEx[txID]
	if !ok {
		return
	}

	for _, url := range urls {
		pool := im.dockerConnPool(transID, url)
		cli, err := pool.GetClient()
		if err != nil {
			panic(err)
		}

		_, err = cli.Call("McDirtyTransTx", map[string]interface{}{"transID": transID, "txID": txID}, 60)
		if err != nil {
			panic(err)
		}
		pool.ReleaseClient(cli)
	}

	delete(vEx, txID)

	v, ok := im.transIDToContractAddr.Load(transID)
	if !ok {
		return
	}
	vEx = v.(*TxID2ContractAddrMap).Map
	delete(vEx, txID)
}

// Commit - commit data when block finished
func (im *InvokerMgr) Commit(transId int64) {

	//依次获取url，进行commit
	if v, ok := im.dockerUrlMap.Load(transId); ok {
		urlMap := v.(*UrlMap).Map
		for url := range urlMap {
			pool := im.dockerConnPool(transId, url)
			cli, err := pool.GetClient()
			if err != nil {
				panic(err)
			}

			result, err := cli.Call("McCommitTrans", map[string]interface{}{"transID": transId}, 60)
			if err != nil {
				panic(err)
			}
			pool.ReleaseClient(cli)

			if !result.(bool) {
				panic("Commit result is false")
			}
		}
	}

	//判断transId对应的缓存中是否有需要通知dockermgr需要更新的合约地址
	if v, ok := im.transIDToContractAddr.Load(transId); ok {
		m := v.(*TxID2ContractAddrMap).Map
		for _, addrs := range m {
			for _, addr := range addrs {
				smcdocker.GetInstance().DirtyContractInvokeURL(0, 0, addr)
			}
		}
	}

	im.dockerUrlMap.Delete(transId)
	im.transMap.Delete(transId)
	im.transIDToContractAddr.Delete(transId)

	// 检查长时间没有发生交易的docker并杀掉
	smcdocker.GetInstance().CheckDockerLiveTime()
}

// McDirtyToken - dirty cache data of token, if any contract change it
func (im *InvokerMgr) McDirtyToken(tokenAddr types.Address) {
	im.dockerMapConnPool.Range(func(key, value interface{}) bool {
		pool := value.(*socket.ConnectionPool)
		cli, err := pool.GetClient()
		if err != nil {
			panic(err)
		}

		result, err := cli.Call("McDirtyToken", map[string]interface{}{"tokenAddr": tokenAddr}, 60)
		if err != nil {
			panic(err)
		}
		pool.ReleaseClient(cli)

		if !result.(bool) {
			panic("McDirtyToken result is false")
		}

		return true
	})
}

// McDirtyContract - dirty cache data of contract, if any contract change it.
func (im *InvokerMgr) McDirtyContract(contractAddr types.Address) {
	im.dockerMapConnPool.Range(func(key, value interface{}) bool {
		pool := value.(*socket.ConnectionPool)
		cli, err := pool.GetClient()
		if err != nil {
			panic(err)
		}

		result, err := cli.Call("McDirtyContract", map[string]interface{}{"contractAddr": contractAddr}, 60)
		if err != nil {
			panic(err)
		}
		pool.ReleaseClient(cli)

		if !result.(bool) {
			panic("McDirtyContract result is false")
		}

		return true
	})
}

// Health -
func (im *InvokerMgr) Health() *types.Health {
	return nil
}

// InitOrUpdateSMC - invoke InitChain/UpdateChain when any contract begin effect at this height.
func (im *InvokerMgr) InitOrUpdateSMC(transId, txId int64, header types2.Header, contractAddr, owner types.Address, inUpgrade bool) (result *types.Response) {
	result = new(types.Response)

	contractAddr, url, err := smcdocker.GetInstance().GetContractInvokeURL(transId, txId, contractAddr)
	if err != nil {
		panic(err)
	}
	m := types.Message{
		Contract: contractAddr,
	}
	invokeParam := types.RPCInvokeCallParam{Sender: owner, Message: m}

	method := ""
	if inUpgrade {
		method = "UpdateChain"
	} else {
		method = "InitChain"
	}

	pool := im.dockerConnPool(transId, url)
	cli, err := pool.GetClient()
	if err != nil {
		panic(err)
	}
	defer pool.ReleaseClient(cli)
	resp, err := cli.Call(method, map[string]interface{}{"blockHeader": header, "transID": transId, "txID": txId, "callParam": invokeParam}, 60)
	if err != nil {
		panic(err)
	}

	err = jsoniter.Unmarshal([]byte(resp.(string)), result)
	if err != nil {
		panic(err)
	}
	if im.isGenesisOrgContract(contractAddr) {
		im.McDirtyContract("*")
	}
	return
}

func (im *InvokerMgr) isGenesisOrgContract(contractAddr types.Address) bool {
	contract := statedbhelper.GetContract(contractAddr)
	if contract == nil {
		return false
	}
	return contract.OrgID == statedbhelper.GetGenesisOrgID(0, 0)
}

// cleanDockerCache - clean docker cache
func (im *InvokerMgr) cleanDockerCache(result *types.Response) (newContractAddr string) {
	for _, v := range result.Tags {
		var receipt std.Receipt
		err := jsoniter.Unmarshal(v.Value, &receipt)
		if err != nil {
			panic(err.Error())
		}

		/**
		根据收据判断，如果 token 或者 contract 的信息有修改，清空所有 docker 中对应的缓存
		目前针对 token 修改的信息包括：setGasPrice，burn，addSupply，setOwner
		如果合约被禁用或部署了新的合约，通知所有的 docker 清空缓存
		**/
		switch receipt.Name {
		case "std::setGasPrice":
			var obj std.SetGasPrice
			if err := jsoniter.Unmarshal(receipt.Bytes, &obj); err != nil {
				panic(err.Error())
			}
			im.McDirtyToken(obj.Token)

		case "std::burn":
			var obj std.Burn
			if err := jsoniter.Unmarshal(receipt.Bytes, &obj); err != nil {
				panic(err.Error())
			}
			im.McDirtyToken(obj.Token)

		case "std::addSupply":
			var obj std.AddSupply
			if err := jsoniter.Unmarshal(receipt.Bytes, &obj); err != nil {
				panic(err.Error())
			}
			im.McDirtyToken(obj.Token)

		case "std::setOwner", "smartcontract.forbidContract", "IBC.setGasPriceRatio": // 更新了出合约信息外的其它信息，如果合约有token，对应的token的owner也会转移
			im.McDirtyContract("*")

		case "smartcontract.deployContract": // 更新了出合约信息外的其它信息
			type deployContract struct {
				ContractAddr types.Address `json:"contractAddr"`
			}

			var obj deployContract
			if err := jsoniter.Unmarshal(receipt.Bytes, &obj); err != nil {
				panic(err.Error())
			}
			newContractAddr = obj.ContractAddr
			im.McDirtyContract("*")
		}
	}

	return
}

// dockerConnPool get connectionPool object from dockerMapConnPool if it's exist,
// or NewConnectionPool for create connection pool and object,
// Note: bNew means the docker pointed by url is new docker,
// if url exists in dockerMapConnPool, it will be NewConnectionPool also.
func (im *InvokerMgr) dockerConnPool(transID int64, url string) *socket.ConnectionPool {

	var pool *socket.ConnectionPool
	var err error
	value, ok := im.dockerMapConnPool.Load(url)
	if !ok {
		pool, err = socket.NewConnectionPool(url, 2, im.logger)
		if err != nil {
			panic(url + ":" + err.Error())
		}

		im.dockerMapConnPool.Store(url, pool)
	} else {
		pool = value.(*socket.ConnectionPool)
	}

	im.setValDockerMap(transID, url)

	return pool
}

// getEffectContract - return effect contract
func (im *InvokerMgr) getEffectContract(
	transId, txId, currentBlock int64,
	contractAddr types.Address,
	methodID uint32) (contract *std.Contract, error types.BcError) {

	error.ErrorCode = types.CodeOK

	// 先判断是不是 token，如果是，找到对应生效的合约地址，
	// 如果不是token，找到合约， 如果接口是五个标准接口并且合约发布了 token，则调用失败，否则继续。
	// 只有五个标准接口可以使用token地址调用
	token := statedbhelper.GetTokenByAddress(transId, txId, contractAddr)

	// If token != nil, it means calling a token address
	if token != nil {
		contract = statedbhelper.GetContract(contractAddr)
		if contract == nil {
			error.ErrorDesc = ""
			error.ErrorCode = types.ErrLogicError
			return
		}
		addrList := statedbhelper.GetContractsByName(transId, txId, contract.Name, contract.OrgID)
		if len(addrList) != 0 {
			for _, v := range addrList {
				con := statedbhelper.GetContract(v)
				if con != nil && con.EffectHeight <= currentBlock &&
					(con.LoseHeight == 0 || con.LoseHeight > currentBlock) {
					contract = con
					return
				}
			}
		}
		error.ErrorDesc = "The contract has expired"
		error.ErrorCode = types.ErrLogicError
		return
	} else {
		contract = statedbhelper.GetContract(contractAddr)
		if contract == nil {
			error.ErrorDesc = "Invalid contract address"
			error.ErrorCode = types.ErrLogicError
			return
		}
		_, ok := im.standardMethods[methodID]
		if ok && contract.Token != "" {
			error.ErrorDesc = "Can not call standard token method by contract address"
			error.ErrorCode = types.ErrLogicError
			return
		}

		if contract.EffectHeight > currentBlock {
			error.ErrorDesc = "The smart contract is not yet in effect"
			error.ErrorCode = types.ErrLogicError
			return
		}

		if contract.LoseHeight != 0 && contract.LoseHeight <= currentBlock {
			error.ErrorDesc = "The contract has expired"
			error.ErrorCode = types.ErrLogicError
			return
		}

		return
	}
}

// Mine - invoke method of mine
func (im *InvokerMgr) Mine(transId, txId int64, header types2.Header, contractAddr, owner types.Address) (result *types.Response) {
	result = new(types.Response)

	contractAddr, url, err := smcdocker.GetInstance().GetContractInvokeURL(transId, txId, contractAddr)
	if err != nil {
		panic(err)
	}
	m := types.Message{
		Contract: contractAddr,
	}
	invokeParam := types.RPCInvokeCallParam{Sender: owner, Message: m}

	pool := im.dockerConnPool(transId, url)
	cli, err := pool.GetClient()
	if err != nil {
		panic(err)
	}
	defer pool.ReleaseClient(cli)
	resp, err := cli.Call("Mine", map[string]interface{}{"blockHeader": header, "transID": transId, "txID": txId, "callParam": invokeParam}, 60)
	if err != nil {
		panic(err)
	}

	err = jsoniter.Unmarshal([]byte(resp.(string)), result)
	if err != nil {
		panic(err)
	}

	return
}

// setValToTransMap - set value to transMap
func (im *InvokerMgr) setValToTransMap(transID, txID int64, url []string) {
	var m *TxID2UrlMap
	if v, ok := im.transMap.Load(transID); !ok {
		m = &TxID2UrlMap{Map: make(map[int64][]string)}
	} else {
		m = v.(*TxID2UrlMap)
	}

	m.Map[txID] = append(m.Map[txID], url...)

	im.transMap.Store(transID, m)
}

// setValToTransCon - set value to transContractAddr
func (im *InvokerMgr) setValToTransCon(transID, txID int64, addrs []string) {
	var m *TxID2ContractAddrMap
	if v, ok := im.transIDToContractAddr.Load(transID); !ok {
		m = &TxID2ContractAddrMap{Map: make(map[int64][]string)}
	} else {
		m = v.(*TxID2ContractAddrMap)
	}

	m.Map[txID] = append(m.Map[txID], addrs...)

	im.transIDToContractAddr.Store(transID, m)
}

// setValDockerMap - set value to dockerUrlMap
func (im *InvokerMgr) setValDockerMap(transID int64, url string) {
	var m *UrlMap
	if v, ok := im.dockerUrlMap.Load(transID); !ok {
		m = &UrlMap{Map: make(map[string]struct{})}
	} else {
		m = v.(*UrlMap)
	}

	m.Map[url] = struct{}{}

	im.dockerUrlMap.Store(transID, m)
}

// getPayer - return payer address, the payer is sender if method gas is not negative,
// else return contract account address
func (im *InvokerMgr) getPayer(height int64, sender types.Address, tx types.Transaction, index int) (types.Address, error) {

	// 创世时走如下逻辑
	if len(tx.Messages) == 1 && tx.Messages[0].Contract == std.GetGenesisContractAddr(statedbhelper.GetChainID()) {
		return sender, nil
	}

	// load contract buffer
	contracts := make([]*std.Contract, 0)
	for _, msg := range tx.Messages {
		contractSplit := strings.Split(msg.Contract, ".")
		contractAddr := msg.Contract
		if len(contractSplit) == 2 {
			contractAddr = contractSplit[1]
		}
		contract, err := im.getEffectContract(0, 0, height, contractAddr, msg.MethodID)
		if err.ErrorCode != types.CodeOK {
			return "", errors.New(err.ErrorDesc)
		}
		contracts = append(contracts, contract)
	}

	if len(tx.Messages) == 2 && index == 0 {
		for _, method := range contracts[1].Methods {
			if method.MethodID == fmt.Sprintf("%x", tx.Messages[1].MethodID) {
				if method.Gas < 0 && tx.Messages[0].MethodID == 0x44d8ca60 && contracts[0].Token != "" {
					return contracts[1].Account, nil
				}
			}
		}
	}

	// construct key
	for _, method := range contracts[index].Methods {
		if method.MethodID == fmt.Sprintf("%x", tx.Messages[index].MethodID) && method.Gas < 0 {
			return contracts[index].Account, nil
		}
	}

	return sender, nil
}

func (im *InvokerMgr) loadContractInfo(height int64, contractAddr types.Address, methodID uint32) (*std.Contract, error) {
	contract, err := im.getEffectContract(0, 0, height, contractAddr, methodID)
	if err.ErrorCode != types.CodeOK {
		return nil, errors.New(err.ErrorDesc)
	}

	im.contractBuffer.Store(contract.Address, contract.Account)
	im.contractBuffer.Store(contract.Address+"Token", contract.Token)
	for _, method := range contract.Methods {
		key := contract.Address + "_" + method.MethodID

		im.contractBuffer.Store(key, method.Gas)
	}

	return contract, nil
}
