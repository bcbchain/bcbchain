package gencmd

import (
	"blockchain/smccheck/parsecode"
	"bytes"
	"os"
	"path/filepath"
	"text/template"
)

var templateText = `package main

import (
	"blockchain/algorithm"
	"blockchain/smcsdk/common/gls"
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/jsoniter"
	"blockchain/smcsdk/sdk/rlp"
	"blockchain/smcsdk/sdk/std"
	sdkType "blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
	"blockchain/smcsdk/sdkimpl/helper"
	"blockchain/smcsdk/sdkimpl/llstate"
	"blockchain/smcsdk/sdkimpl/object"
	"blockchain/smcsdk/sdkimpl/sdkhelper"
	"blockchain/types"
	"common/socket"
	"contract/{{.OrgID}}/stub"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"

	"github.com/spf13/cobra"
	"contract/stubcommon/common"
	"contract/stubcommon/softforks"
	abci "github.com/tendermint/abci/types"
	tmcommon "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"
	"golang.org/x/crypto/sha3"
)

type Context struct {
	Header    *abci.Header
	BlockHash sdkType.Hash
}

var (
	logger          log.Loggerf
	flagRPCPort     int
	flagCallbackURL string
	p               *socket.ConnectionPool

	context sync.Map //map[transID]*Context
)

//Invoke invoke function
func Invoke(req map[string]interface{}) (interface{}, error) {
	logger.Tracef("Invoke starting")

	transID, txID, callParam := parseReq(req)
	smc := createSmc(transID, txID, callParam)

	logger.Infof("[transID=%d][txID=%d]invoke", transID, txID)

	response := invoke(smc)
	resBytes, _ := jsoniter.Marshal(response)

	return string(resBytes), nil
}

func invoke(smc sdk.ISmartContract) types.Response {

	var response types.Response
	gls.Mgr.SetValues(gls.Values{gls.SDKKey: smc}, func() {
		contractStub := stub.NewStub(smc, logger)
		if contractStub == nil {
			response.Code = sdkType.ErrInvalidAddress
			response.Log = fmt.Sprintf("Call contract=%s,version=%s is not exist or lost",
				smc.Message().Contract().Address(), smc.Message().Contract().Version())
		} else {
			logger.Debugf("contractStub Invoke")
			response = contractStub.Invoke(smc)

			logger.Debugf("contractStub Commit")
			smc.(*sdkimpl.SmartContract).Commit()
		}
	})

	return response
}

// InitChain initial smart contract
func InitChain(req map[string]interface{}) (interface{}, error) {
	logger.Info("contract InitChain")

	// prepare
	transID, txID, callParam := parseReq(req)
	smc := createSmcEx(transID, txID, callParam)

	response := initChain(smc)

	resBytes, _ := jsoniter.Marshal(response)
	return string(resBytes), nil
}

func initChain(smc sdk.ISmartContract) types.Response {

	var response types.Response
	gls.Mgr.SetValues(gls.Values{gls.SDKKey: smc}, func() {
		contractStub := stub.NewStub(smc, logger)
		if contractStub == nil {
			response.Code = sdkType.ErrInvalidAddress
			response.Log = fmt.Sprintf("Call contract=%s,version=%s is not exist or lost",
				smc.Message().Contract().Address(), smc.Message().Contract().Version())
		} else {
			logger.Debugf("Invoke contractStub InitChain")
			response = contractStub.InitChain(smc)

			logger.Debugf("Invoke contractStub Commit")
			smc.(*sdkimpl.SmartContract).Commit()
		}
	})

	return response
}

// UpdateChain initial smart contract when it's upgraded
func UpdateChain(req map[string]interface{}) (interface{}, error) {
	logger.Debug("contract UpdateChain")

	// prepare
	transID, txID, callParam := parseReq(req)
	smc := createSmcEx(transID, txID, callParam)

	response := updateChain(smc)

	resBytes, _ := jsoniter.Marshal(response)
	return string(resBytes), nil
}

func updateChain(smc sdk.ISmartContract) types.Response {

	var response types.Response
	gls.Mgr.SetValues(gls.Values{gls.SDKKey: smc}, func() {
		contractStub := stub.NewStub(smc, logger)
		if contractStub == nil {
			response.Code = sdkType.ErrInvalidAddress
			response.Log = fmt.Sprintf("Call contract=%s,version=%s is not exist or lost",
				smc.Message().Contract().Address(), smc.Message().Contract().Version())
		} else {
			logger.Debugf("Invoke contractStub UpdateChain")
			response = contractStub.UpdateChain(smc)

			logger.Debugf("Invoke contractStub Commit")
			smc.(*sdkimpl.SmartContract).Commit()
		}
	})

	return response
}

// Mine call mine of smart contract
func Mine(req map[string]interface{}) (interface{}, error) {
	logger.Debug("contract Mine")

	// prepare
	transID, txID, callParam := parseReq(req)
	smc := createSmcEx(transID, txID, callParam)

	response := mine(smc)

	resBytes, _ := jsoniter.Marshal(response)
	return string(resBytes), nil
}

func mine(smc sdk.ISmartContract) types.Response {

	var response types.Response
	gls.Mgr.SetValues(gls.Values{gls.SDKKey: smc}, func() {
		contractStub := stub.NewStub(smc, logger)
		if contractStub == nil {
			response.Code = sdkType.ErrInvalidAddress
			response.Log = fmt.Sprintf("Call contract=%s,version=%s is not exist or lost",
				smc.Message().Contract().Address(), smc.Message().Contract().Version())
		} else {
			logger.Debugf("Invoke contractStub UpdateChain")
			response = contractStub.Mine(smc)

			logger.Debugf("Invoke contractStub Commit")
			smc.(*sdkimpl.SmartContract).Commit()
		}
	})

	return response
}

func parseReq(req map[string]interface{}) (transID, txID int64, callParam types.RPCInvokeCallParam) {
	logger.Tracef("Invoke starting")

	transID = int64(req["transID"].(float64))
	txID = int64(req["txID"].(float64))

	// setup call parameter
	mCallParam := req["callParam"].(map[string]interface{})
	jsonStr, _ := jsoniter.Marshal(mCallParam)
	logger.Debugf("[transID=%d][txID=%d]callParam=%s", transID, txID, string(jsonStr))
	err := jsoniter.Unmarshal(jsonStr, &callParam)
	if err != nil {
		logger.Errorf("[transID=%d][txID=%d]callParam Unmarshal error", transID, txID, err.Error())
		panic(err)
	}

	// setup block header
	mBlockHeader := req["blockHeader"].(map[string]interface{})
	var blockHeader abci.Header
	jsonStr, _ = jsoniter.Marshal(mBlockHeader)
	logger.Debugf("[transID=%d][txID=%d]blockHeader=%s", transID, txID, string(jsonStr))
	err = jsoniter.Unmarshal(jsonStr, &blockHeader)
	if err != nil {
		logger.Errorf("[transID=%d][txID=%d]invoke error=%s", transID, txID, err.Error())
		panic(err)
	}

	var c *Context
	if v, ok := context.Load(transID); ok {
		c = v.(*Context)
		c.Header = &blockHeader
   		c.BlockHash = callParam.BlockHash
	} else {
		c = &Context{Header: &blockHeader, BlockHash: callParam.BlockHash}
		context.Store(transID, c)
	}

	logger.Infof("[transID=%d][txID=%d]invoke", transID, txID)

	return
}

//TransferFunc is used to transfer token for crossing contract invoking.
// nolint unhandled
func transfer(sdk sdk.ISmartContract, tokenAddr, to types.Address, value bn.Number, note string) ([]sdkType.KVPair, sdkType.Error) {
	logger.Debug("TransferFunc", "tokenAddress", tokenAddr, "to", to, "value", value)
	contract := sdk.Helper().ContractHelper().ContractOfToken(tokenAddr)
	logger.Info("Contract", "address", contract.Address(), "name", contract.Name(), "version", contract.Version())
	originMessage := sdk.Message()

	var items []sdkType.HexBytes
	var mID string
	if note != "" {
		mID = "88e0eb75"		// prototype: Transfer(types.Address,bn.Number,string)
		items = wrapInvokeParams(to, value, note)
	} else {
		mID = "44d8ca60"       // prototype: Transfer(types.Address,bn.Number)
		items = wrapInvokeParams(to, value)
	}
	newSdk := sdkhelper.OriginNewMessage(sdk, contract, mID, nil)

	newmsg := object.NewMessage(newSdk, newSdk.Message().Contract(), mID, items, originMessage.Contract().Account().Address(), 
		newSdk.Message().Payer().Address(), newSdk.Message().Origins(), nil)
	newSdk.(*sdkimpl.SmartContract).SetMessage(newmsg)
	contractStub := stub.NewStub(newSdk, logger)
	response := contractStub.InvokeInternal(newSdk, common.INTERFACE)
	logger.Debug("Invoke response", "code", response.Code, "tags", response.Tags)
	if response.Code != sdkType.CodeOK {
		return nil, sdkType.Error{ErrorCode: response.Code, ErrorDesc: response.Log}
	}

	// read receipts from response and append to original sdk message
	recKV := make([]sdkType.KVPair, len(response.Tags))
	for i, v := range response.Tags {
		recKV[i] = sdkType.KVPair{Key: v.Key, Value: v.Value}
	}
	newSdk.(*sdkimpl.SmartContract).SetMessage(originMessage)
	return recKV, sdkType.Error{ErrorCode: sdkType.CodeOK}
}

// ibcInvoke is used to ibc for crossing contract invoking.
// nolint unhandled
func ibcInvoke(sdk sdk.ISmartContract) (string, []sdkType.KVPair, sdkType.Error) {

	contractStub := stub.NewIBCStub(sdk, logger)
	response := contractStub.Invoke(sdk)
	logger.Debug("Invoke response", "code", response.Code, "tags", response.Tags)
	if response.Code != sdkType.CodeOK {
		return "", nil, sdkType.Error{ErrorCode: response.Code, ErrorDesc: response.Log}
	}

	// read receipts from response and append to original sdk message
	recKV := make([]sdkType.KVPair, 0)
	for _, v := range response.Tags {
		recKV = append(recKV, sdkType.KVPair{Key: v.Key, Value: v.Value})
	}

	return response.Data, recKV, sdkType.Error{ErrorCode: sdkType.CodeOK}
}

// wrapInvokeParams - wrap contract parameters
func wrapInvokeParams(params ...interface{}) []sdkType.HexBytes {
	paramsRlp := make([]sdkType.HexBytes, len(params))
	for i, param := range params {
		var paramRlp []byte
		var err error

		paramRlp, err = rlp.EncodeToBytes(param)
		if err != nil {
			panic(err)
		}
		paramsRlp[i] = paramRlp
	}
	return paramsRlp
}

func createSmc(transID, txID int64, callParam types.RPCInvokeCallParam) sdk.ISmartContract {
	sdkReceipts := make([]sdkType.KVPair, len(callParam.Receipts))
	for i, v := range callParam.Receipts {
		sdkReceipts[i] = sdkType.KVPair{Key: v.Key, Value: v.Value}
	}

	items := make([]sdkType.HexBytes, len(callParam.Message.Items))
	for i, item := range callParam.Message.Items {
		items[i] = []byte(item)
	}

	logger.Debugf("[transID=%d][txID=%d]invoke sdkhelper New", transID, txID)
	smc := sdkhelper.New(
		transID,
		txID,
		callParam.Sender,
		callParam.Payer,
		callParam.Tx.GasLimit,
		callParam.GasLeft,
		callParam.Tx.Note,
		callParam.TxHash,
		callParam.Message.Contract,
		fmt.Sprintf("%x", callParam.Message.MethodID),
		items,
		sdkReceipts,
	)

	return smc
}

func createSmcEx(transID, txID int64, callParam types.RPCInvokeCallParam) sdk.ISmartContract {
	sdkReceipts := make([]sdkType.KVPair, len(callParam.Receipts))
	for i, v := range callParam.Receipts {
		sdkReceipts[i] = sdkType.KVPair{Key: v.Key, Value: v.Value}
	}

	items := make([]sdkType.HexBytes, len(callParam.Message.Items))
	for i, item := range callParam.Message.Items {
		items[i] = []byte(item)
	}

	var header *abci.Header
	if v, ok := context.Load(transID); ok {
		c := v.(*Context)
		header = c.Header
	}

	smc := &sdkimpl.SmartContract{}
	gls.Mgr.SetValues(gls.Values{gls.SDKKey: smc}, func() {
		llState := llstate.NewLowLevelSDB(smc, transID, txID)
		smc.SetLlState(llState)

		block := object.NewBlock(smc, header.ChainID, header.Version, sdkType.Hash{}, header.DataHash,
			header.Height, header.Time, header.NumTxs, header.ProposerAddress, header.RewardAddress,
			header.RandomeOfBlock, header.LastBlockID.Hash, header.LastCommitHash, header.LastAppHash,
			int64(header.LastFee))
		smc.SetBlock(block)

		helperObj := helper.NewHelper(smc)
		smc.SetHelper(helperObj)

		contract := object.NewContractFromAddress(smc, callParam.Message.Contract)
		msg := object.NewMessage(smc, contract, "", items, callParam.Sender, callParam.Payer,
			nil, nil)
		smc.SetMessage(msg)
	})

	return smc
}

// ----- callback functions begin -----
func pool() *socket.ConnectionPool {
	if p == nil {
		var err error
		p, err = socket.NewConnectionPool(flagCallbackURL, 4, logger)
		if err != nil {
			panic(err)
		}
	}

	return p
}

//adapter回调函数
func set(transID, txID int64, value map[string][]byte) {
	var err error

	// for Marshal result can UnMarshal, it necessary
	data := make(map[string]string)
	for k, v := range value {
		if v == nil {
			data[k] = string([]byte{})
		} else {
			data[k] = string(v)
		}
	}

	logger.Debugf("[transID=%d][txID=%d]set data=%v", transID, txID, data)
	cli, err := pool().GetClient()
	if err != nil {
		msg := fmt.Sprintf("[transID=%d][txID=%d]socket set error: %s", transID, txID, err.Error())
		logger.Errorf(msg)
		panic(err)
	}
	defer pool().ReleaseClient(cli)

	result, err := cli.Call("set", map[string]interface{}{"transID": transID, "txID": txID, "data": data}, 10)
	if err != nil {
		msg := fmt.Sprintf("[transID=%d][txID=%d]socket set error: %s", transID, txID, err.Error())
		logger.Errorf(msg)
		panic("socket set error: " + err.Error())
	}
	logger.Debugf("[transID=%d][txID=%d]set return is %t", transID, txID, result.(bool))

	if !result.(bool) {
		msg := fmt.Sprintf("[transID=%d][txID=%d]socket set error: return false", transID, txID)
		logger.Errorf(msg)
		panic(msg)
	}
}

func get(transID, txID int64, key string) []byte {

	logger.Debugf("[transID=%d][txID=%d]get key=%s", transID, txID, key)
	cli, err := pool().GetClient()
	if err != nil {
		msg := fmt.Sprintf("[transID=%d][txID=%d]socket get error: %s", transID, txID, err.Error())
		logger.Errorf(msg)
		panic(err)
	}
	defer pool().ReleaseClient(cli)

	result, err := cli.Call("get", map[string]interface{}{"transID": transID, "txID": txID, "key": key}, 10)
	if err != nil {
		msg := fmt.Sprintf("[transID=%d][txID=%d]socket get error: %s", transID, txID, err.Error())
		logger.Errorf(msg)
		logger.Flush()
		panic(msg)
	}
	logger.Debugf("[transID=%d][txID=%d]get key=%s, result=%v", transID, txID, key, result)

	return []byte(result.(string))
}

func build(transID int64, txID int64, contractMeta std.ContractMeta) std.BuildResult {

	resBytes, _ := jsoniter.Marshal(contractMeta)
	logger.Debugf("[transID=%d][txID=%d]build orgID=%s contract=%s version=%s", transID, txID, contractMeta.OrgID, contractMeta.Name, contractMeta.Version)
	cli, err := pool().GetClient()
	if err != nil {
		msg := fmt.Sprintf("[transID=%d][txID=%d]socket build error: %s", transID, txID, err.Error())
		logger.Errorf(msg)
		panic(err)
	}
	defer pool().ReleaseClient(cli)

	result, err := cli.Call("build", map[string]interface{}{"transID": transID, "txID": txID, "contractMeta": string(resBytes)}, 180)
	if err != nil {
		msg := fmt.Sprintf("[transID=%d][txID=%d]socket build error: %s", transID, txID, err.Error())
		logger.Errorf(msg)
		panic(err)
	}
	logger.Debugf("[transID=%d][txID=%d]build result=%v", transID, txID, result)

	var buildResult std.BuildResult
	err = jsoniter.Unmarshal([]byte(result.(string)), &buildResult)
	if err != nil {
		panic(err)
	}

	return buildResult
}

func getBlock(transID, height int64) std.Block {
	if height == 0 {
		if v, ok := context.Load(transID); ok {
			c := v.(*Context)
			if c.Header != nil {
				header := c.Header
				block := std.Block{
					ChainID:         header.ChainID,
					Height:          header.Height,
					Time:            header.Time,
					NumTxs:          header.NumTxs,
					DataHash:        header.DataHash,
					ProposerAddress: header.ProposerAddress,
					RewardAddress:   header.RewardAddress,
					RandomNumber:    header.RandomeOfBlock,
					Version:         header.Version,
					LastBlockHash:   header.LastBlockID.Hash,
					LastCommitHash:  header.LastCommitHash,
					LastAppHash:     header.LastAppHash,
					LastFee:         int64(header.LastFee),
				}
				if len(c.BlockHash) == 0 {
					block.BlockHash = blockHash(block)
				} else {
					block.BlockHash = c.BlockHash
				}

				return block
			}
		}
	}

	logger.Debugf("get block height=%d", height)
	cli, err := pool().GetClient()
	if err != nil {
		msg := fmt.Sprintf("socket getBlock error: %s", err.Error())
		logger.Errorf(msg)
		panic(err)
	}
	defer pool().ReleaseClient(cli)

	result, err := cli.Call("block", map[string]interface{}{"height": height}, 10)
	if err != nil {
		msg := fmt.Sprintf("socket getBlock error: %s", err.Error())
		logger.Errorf(msg)
		panic(err)
	}

	var blockResult std.Block
	err = jsoniter.Unmarshal([]byte(result.(string)), &blockResult)

	return blockResult
}

func blockHash(block std.Block) sdkType.HexBytes {
	sha256 := sha3.New256()
	sha256.Write([]byte(block.ChainID))
	sha256.Write(algorithm.IntToBytes(int(block.Height)))
	sha256.Write(algorithm.IntToBytes(int(block.Time)))
	sha256.Write(algorithm.IntToBytes(int(block.NumTxs)))
	sha256.Write(block.DataHash)
	sha256.Write([]byte(block.ProposerAddress))
	sha256.Write([]byte(block.RewardAddress))
	sha256.Write(block.RandomNumber)
	sha256.Write([]byte(block.Version))
	sha256.Write(block.LastBlockHash)
	sha256.Write(block.LastCommitHash)
	sha256.Write(block.LastAppHash)
	sha256.Write(algorithm.IntToBytes(int(block.LastFee)))

	return sha256.Sum(nil)
}

// ----- callback functions end -----

//Routes routes map
//NOTE: Amino is registered in rpc/core/types/wire.go.
var Routes = map[string]socket.CallBackFunc{
	"Invoke":          Invoke,
	"McCommitTrans":   McCommitTrans,
	"McDirtyTrans":    McDirtyTrans,
	"McDirtyTransTx":  McDirtyTransTx,
	"McDirtyToken":    McDirtyToken,
	"McDirtyContract": McDirtyContract,
	"SetLogLevel":     SetLogLevel,
	"InitSoftForks":   InitSoftForks,
	"Health":          Health,
	"InitChain":       InitChain,
	"UpdateChain":     UpdateChain,
	"Mine":            Mine,
}

//RunRPC starts RPC service
func RunRPC(port int) error {
	logger = log.NewTMLogger(".", "smcsvc")
	logger.AllowLevel("debug")
	logger.SetOutputAsync(true)
	logger.SetOutputToFile(true)
	logger.SetOutputToScreen(false)
	logger.SetOutputFileSize(20000000)

	sdkhelper.Init(transfer, build, set, get, getBlock, ibcInvoke, &logger)

	// start server and wait forever
	svr, err := socket.NewServer("tcp://0.0.0.0:"+fmt.Sprintf("%d", port), Routes, 0, logger)
	if err != nil {
		tmcommon.Exit(err.Error())
	}

	// start server and wait forever
	err = svr.Start()
	if err != nil {
		tmcommon.Exit(err.Error())
	}

	return nil
}

//McCommitTrans commit transaction data of memory cache
func McCommitTrans(req map[string]interface{}) (interface{}, error) {

	transID := int64(req["transID"].(float64))
	logger.Infof("[transID=%d]McCommitTrans", transID)

	context.Delete(transID)
	sdkhelper.McCommit(transID)

	return true, nil
}

//McDirtyTrans dirty transaction data of memory cache
func McDirtyTrans(req map[string]interface{}) (interface{}, error) {

	transID := int64(req["transID"].(float64))
	logger.Infof("[transID=%d]McDirtyTrans", transID)

	context.Delete(transID)
	sdkhelper.McDirtyTrans(transID)

	return true, nil
}

//McDirtyTransTx dirty tx data of transaction of memory cache
func McDirtyTransTx(req map[string]interface{}) (interface{}, error) {

	transID := int64(req["transID"].(float64))
	txID := int64(req["txID"].(float64))
	logger.Infof("[transID=%d][txID=%d]McDirtyTransTx", transID, txID)

	sdkhelper.McDirtyTransTx(transID, txID)
	return true, nil
}

//McDirtyToken dirty token data of memory cache
func McDirtyToken(req map[string]interface{}) (interface{}, error) {

	tokenAddr := req["tokenAddr"].(string)
	logger.Infof("McDirtyToken tokenAddr=%s", tokenAddr)

	sdkhelper.McDirtyToken(tokenAddr)
	return true, nil
}

//McDirtyContract dirty contract data of memory cache
func McDirtyContract(req map[string]interface{}) (interface{}, error) {

	contractAddr := req["contractAddr"].(string)
	logger.Infof("McDirtyToken contractAddr=%s", contractAddr)

	sdkhelper.McDirtyContract(contractAddr)
	return true, nil
}

//SetLogLevel sets log level
func SetLogLevel(req map[string]interface{}) (interface{}, error) {

	level := req["level"].(string)
	logger.Infof("SetLogLevel level=%s", level)

	logger.AllowLevel(level)
	return true, nil
}

//InitSoftForks init soft forks information
func InitSoftForks(req map[string]interface{}) (interface{}, error) {

	forksBytes := req["softforks"].(string)
	logger.Infof("InitSoftForks softforks=%s", forksBytes)

	softforks.Init([]byte(forksBytes))
	return true, nil
}

// Health return health message
func Health(req map[string]interface{}) (interface{}, error) {
	return "health", nil
}

//RootCmd cmd
var RootCmd = &cobra.Command{
	Use:   "smcrunsvc",
	Short: "grpc",
	Long:  "smcsvc rpc console",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunRPC(flagRPCPort)
	},
}

func main() {
	go func() {
		if e := http.ListenAndServe(":2019", nil); e != nil {
			fmt.Println("pprof cannot start!!!")
		}
	}()

	err := excute()
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}

func excute() error {
	addFlags()
	addCommand()
	return RootCmd.Execute()
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start the smc_service",
	Long:  "start the smc_service",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunRPC(flagRPCPort)
	},
}

func addStartFlags() {
	startCmd.PersistentFlags().IntVarP(&flagRPCPort, "port", "p", 8080, "The port of the smc rpc service")
	startCmd.PersistentFlags().StringVarP(&flagCallbackURL, "callbackUrl", "c", "tcp://localhost:32333", "The url of the adapter callback")
}

func addFlags() {
	addStartFlags()
}

func addCommand() {
	RootCmd.AddCommand(startCmd)
}
`

type Cmd struct {
	OrgID string
}

// GenStubCommon - generate the stub common go source
func GenCmd(rootDir, orgID string) {

	newPath := filepath.Join(rootDir, "cmd/smcrunsvc")
	if err := os.MkdirAll(newPath, os.FileMode(0750)); err != nil {
		panic(err)
	}
	filename := filepath.Join(newPath, "smcrunsvc.go")

	tmpl, err := template.New("smcrunsvc").Parse(templateText)
	if err != nil {
		panic(err)
	}

	orgCmd := Cmd{OrgID: orgID}
	var buf bytes.Buffer

	if err = tmpl.Execute(&buf, orgCmd); err != nil {
		panic(err)
	}

	if err := parsecode.FmtAndWrite(filename, buf.String()); err != nil {
		panic(err)
	}
}
