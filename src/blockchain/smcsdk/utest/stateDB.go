package utest

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/crypto/sha3"
	"blockchain/smcsdk/sdk/jsoniter"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
	"blockchain/smcsdk/sdkimpl/object"
	"fmt"
	"math/big"
	"time"
)

var (
	//BlockHeight block height
	BlockHeight int64
	//LastNumTxs number of txs
	LastNumTxs int32

	prefix       = ""
	tx           = []types.HexBytes{types.HexBytes("YmNiPHR4Pi52MS4yVVpHSlJRZDYxYTdMa296MzJuRWJQQU5RQVJhWG9jZWQ1dDR5THRvZkVtaENWMkc3eWVrNVgzdU1UUzdlazU5ODZNTExRZ3ZRRllEb3ZQZjYyb3RjUjNLQ3p0VU5tcE1xU1l1SE5EREhHeGVCdUZZb0xyRk5LU3cxdEFZb2t1NGt4RlouPDE+LllUZ2lBMWdkREdpMkw4aG44enhlNWp5d2Y0bTFvZ3o4OXFRd0R1Y0ZCcERhQjZSOFkzOW5MUERyZ0FOZUxYVzNmZGdNd2o4WWFBUkRmTlZLTWlyTGRKQ1FLMkZham1ScFl4OHRZdEx0ZWlKdUhyUlgzakc3ZU1Nc01FVVdXeEdDaUxCNG5DMXkxVUs1NTk4ODg4WUEy")}
	block        = []byte(`{"chainID":"test","blockHash":"663666426932665034586B70685A5A323757424F786845473079493D","height":1,"time":1542436677,"numTxs":1,"dataHash":"4369496472305A546757635671336C3131326D31773539684557593D","proposerAddress":"testCUh7Zsb7PBgLwHJVok2QaMhbW64HNK4FU","rewardAddress":"testCUh7Zsb7PBgLwHJVok2QaMhbW64HNK4FU","randomNumber":"596D4E694E3074346546704E64314E3057556448626B74704D33684F546D3035625735744E33426B563352474F576445","version":"1.0","lastBlockHash":"4369496472305A546757635671336C3131326D31773539684557593D","lastCommitHash":"4D31724C4573745A68314B30624A644D6B4A4B53625774324450673D","lastAppHash":"36324446453643413937353539313437324435323143413943303845413243453242444646333546363231363230393132393937324636414133334633314234","lastFee":1500000}`)
	tokenBTC     = []byte(`{"address":"testKPkrvMkHZwJcmmaB9uXVNuWLjF6ssDDiB","owner":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","name":"BTC","symbol":"BTC","totalSupply":2000000000000000000,"addSupplyEnabled":false,"burnEnabled":false,"gasPrice":2500}`)
	tokenLTC     = []byte(`{"address":"testPyCmf1eWhGPzi8EZZ2aeZ7xBP43N52PmD","owner":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","name":"LTC","symbol":"LTC","totalSupply":2000000000000000000,"addSupplyEnabled":false,"burnEnabled":false,"gasPrice":2500}`)
	tokenETH     = []byte(`{"address":"test8kHEKgHzQLs3AG2J5T1HLEoC8HhUFt6Qv","owner":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","name":"ETH","symbol":"ETH","totalSupply":2000000000000000000,"addSupplyEnabled":false,"burnEnabled":false,"gasPrice":2500}`)
	tokenEOS     = []byte(`{"address":"test3kgRHcxDPWTgVRc3Kkvs3JQ1QQ5foE7bi","owner":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","name":"EOS","symbol":"EOS","totalSupply":2000000000000000000,"addSupplyEnabled":false,"burnEnabled":false,"gasPrice":2500}`)
	tokenUSDX    = []byte(`{"address":"testPsjtk4XqCsktM7gfL6Vm54tVPfeabFurV","owner":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","name":"USDX","symbol":"USDX","totalSupply":2000000000000000000,"addSupplyEnabled":false,"burnEnabled":false,"gasPrice":2500}`)
	tokenBCB     = []byte(`{"address":"test8s6oGjxdFVxzbVjQDikQiTs3EUbhPCtPo","owner":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","name":"BCB","symbol":"BCB","totalSupply":2000000000000000000,"addSupplyEnabled":false,"burnEnabled":false,"gasPrice":2500}`)
	tokenDC      = []byte(`{"address":"test6g6FXQjkSLmELnmatWcTwDUv4SigQA8wR","owner":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","name":"Diamond Coin","symbol":"DC","totalSupply":2000000000000000000,"addSupplyEnabled":false,"burnEnabled":false,"gasPrice":2500}`)
	contractBTC  = []byte(`{"address":"testKPkrvMkHZwJcmmaB9uXVNuWLjF6ssDDiB","account":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","owner":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","name":"token-templet-BTC","version":"2.0","codeHash":"563FAB3586B75D6831D313A14F45A1C23ABEB39B891D9FD726495EFF3A62E07A","effectHeight":1,"loseHeight":0,"keyPrefix":"","token":"testKPkrvMkHZwJcmmaB9uXVNuWLjF6ssDDiB","orgID":"orgJgaGConUyK81zibntUBjQ33PKctpk1K1G","chainVersion":0}`)
	contractLTC  = []byte(`{"address":"testPyCmf1eWhGPzi8EZZ2aeZ7xBP43N52PmD","account":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","owner":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","name":"token-templet-LTC","version":"2.0","codeHash":"563FAB3586B75D6831D313A14F45A1C23ABEB39B891D9FD726495EFF3A62E07A","effectHeight":1,"loseHeight":0,"keyPrefix":"","token":"testPyCmf1eWhGPzi8EZZ2aeZ7xBP43N52PmD","orgID":"orgJgaGConUyK81zibntUBjQ33PKctpk1K1G","chainVersion":0}`)
	contractETH  = []byte(`{"address":"test8kHEKgHzQLs3AG2J5T1HLEoC8HhUFt6Qv","account":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","owner":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","name":"token-templet-ETH","version":"2.0","codeHash":"563FAB3586B75D6831D313A14F45A1C23ABEB39B891D9FD726495EFF3A62E07A","effectHeight":1,"loseHeight":0,"keyPrefix":"","token":"test8kHEKgHzQLs3AG2J5T1HLEoC8HhUFt6Qv","orgID":"orgJgaGConUyK81zibntUBjQ33PKctpk1K1G","chainVersion":0}`)
	contractEOS  = []byte(`{"address":"test3kgRHcxDPWTgVRc3Kkvs3JQ1QQ5foE7bi","account":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","owner":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","name":"token-templet-EOS","version":"2.0","codeHash":"563FAB3586B75D6831D313A14F45A1C23ABEB39B891D9FD726495EFF3A62E07A","effectHeight":1,"loseHeight":0,"keyPrefix":"","token":"test3kgRHcxDPWTgVRc3Kkvs3JQ1QQ5foE7bi","orgID":"orgJgaGConUyK81zibntUBjQ33PKctpk1K1G","chainVersion":0}`)
	contractUSDX = []byte(`{"address":"testPsjtk4XqCsktM7gfL6Vm54tVPfeabFurV","account":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","owner":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","name":"token-templet-USDX","version":"2.0","codeHash":"563FAB3586B75D6831D313A14F45A1C23ABEB39B891D9FD726495EFF3A62E07A","effectHeight":1,"loseHeight":0,"keyPrefix":"","token":"testPsjtk4XqCsktM7gfL6Vm54tVPfeabFurV","orgID":"orgJgaGConUyK81zibntUBjQ33PKctpk1K1G","chainVersion":0}`)
	contractBCB  = []byte(`{"address":"test8s6oGjxdFVxzbVjQDikQiTs3EUbhPCtPo","account":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","owner":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","name":"token-templet-BCB","version":"2.0","codeHash":"563FAB3586B75D6831D313A14F45A1C23ABEB39B891D9FD726495EFF3A62E07A","effectHeight":1,"loseHeight":0,"keyPrefix":"","token":"test8s6oGjxdFVxzbVjQDikQiTs3EUbhPCtPo","orgID":"orgJgaGConUyK81zibntUBjQ33PKctpk1K1G","chainVersion":0}`)
	contractDC   = []byte(`{"address":"test6g6FXQjkSLmELnmatWcTwDUv4SigQA8wR","account":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","owner":"testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu","name":"token-templet-DC","version":"2.0","codeHash":"563FAB3586B75D6831D313A14F45A1C23ABEB39B891D9FD726495EFF3A62E07A","effectHeight":1,"loseHeight":0,"keyPrefix":"","token":"test6g6FXQjkSLmELnmatWcTwDUv4SigQA8wR","orgID":"orgJgaGConUyK81zibntUBjQ33PKctpk1K1G","chainVersion":0}`)
	addrBTC      = []byte(`"testKPkrvMkHZwJcmmaB9uXVNuWLjF6ssDDiB"`)
	addrLTC      = []byte(`"testPyCmf1eWhGPzi8EZZ2aeZ7xBP43N52PmD"`)
	addrETH      = []byte(`"test8kHEKgHzQLs3AG2J5T1HLEoC8HhUFt6Qv"`)
	addrEOS      = []byte(`"test3kgRHcxDPWTgVRc3Kkvs3JQ1QQ5foE7bi"`)
	addrUSDX     = []byte(`"testPsjtk4XqCsktM7gfL6Vm54tVPfeabFurV"`)
	addrBCB      = []byte(`"test8s6oGjxdFVxzbVjQDikQiTs3EUbhPCtPo"`)
	addrDC       = []byte(`"test6g6FXQjkSLmELnmatWcTwDUv4SigQA8wR"`)
	balBTC       = []byte(`{"address":"testKPkrvMkHZwJcmmaB9uXVNuWLjF6ssDDiB","balance":2000000000000000000}`)
	balLTC       = []byte(`{"address":"testPyCmf1eWhGPzi8EZZ2aeZ7xBP43N52PmD","balance":2000000000000000000}`)
	balETH       = []byte(`{"address":"test8kHEKgHzQLs3AG2J5T1HLEoC8HhUFt6Qv","balance":2000000000000000000}`)
	balEOS       = []byte(`{"address":"test3kgRHcxDPWTgVRc3Kkvs3JQ1QQ5foE7bi","balance":2000000000000000000}`)
	balUSDX      = []byte(`{"address":"testPsjtk4XqCsktM7gfL6Vm54tVPfeabFurV","balance":2000000000000000000}`)
	balBCB       = []byte(`{"address":"test8s6oGjxdFVxzbVjQDikQiTs3EUbhPCtPo","balance":2000000000000000000}`)
	balDC        = []byte(`{"address":"test6g6FXQjkSLmELnmatWcTwDUv4SigQA8wR","balance":2000000000000000000}`)

	stateDB   map[string][]byte
	stateBuff map[string][]byte
)

func initStateDB() {
	BlockHeight = 0
	LastNumTxs = 0

	stateDB = make(map[string][]byte)
	stateDB[std.KeyOfAppState()] = block
	stateDB["/genesis/chainid"] = []byte("test")
	stateDB["/token/testKPkrvMkHZwJcmmaB9uXVNuWLjF6ssDDiB"] = tokenBTC
	stateDB["/token/testPyCmf1eWhGPzi8EZZ2aeZ7xBP43N52PmD"] = tokenLTC
	stateDB["/token/test8kHEKgHzQLs3AG2J5T1HLEoC8HhUFt6Qv"] = tokenETH
	stateDB["/token/test3kgRHcxDPWTgVRc3Kkvs3JQ1QQ5foE7bi"] = tokenEOS
	stateDB["/token/testPsjtk4XqCsktM7gfL6Vm54tVPfeabFurV"] = tokenUSDX
	stateDB["/token/test8s6oGjxdFVxzbVjQDikQiTs3EUbhPCtPo"] = tokenBCB
	stateDB["/token/test6g6FXQjkSLmELnmatWcTwDUv4SigQA8wR"] = tokenDC
	stateDB["/contract/testKPkrvMkHZwJcmmaB9uXVNuWLjF6ssDDiB"] = contractBTC
	stateDB["/contract/testPyCmf1eWhGPzi8EZZ2aeZ7xBP43N52PmD"] = contractLTC
	stateDB["/contract/test8kHEKgHzQLs3AG2J5T1HLEoC8HhUFt6Qv"] = contractETH
	stateDB["/contract/test3kgRHcxDPWTgVRc3Kkvs3JQ1QQ5foE7bi"] = contractEOS
	stateDB["/contract/testPsjtk4XqCsktM7gfL6Vm54tVPfeabFurV"] = contractUSDX
	stateDB["/contract/test8s6oGjxdFVxzbVjQDikQiTs3EUbhPCtPo"] = contractBCB
	stateDB["/contract/test6g6FXQjkSLmELnmatWcTwDUv4SigQA8wR"] = contractDC
	stateDB["/token/name/btc"] = addrBTC
	stateDB["/token/name/ltc"] = addrLTC
	stateDB["/token/name/eth"] = addrETH
	stateDB["/token/name/eos"] = addrEOS
	stateDB["/token/name/usdx"] = addrUSDX
	stateDB["/token/name/bcb"] = addrBCB
	stateDB["/token/name/diamond coin"] = addrDC
	stateDB["/account/ex/testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu/token/testKPkrvMkHZwJcmmaB9uXVNuWLjF6ssDDiB"] = balBTC
	stateDB["/account/ex/testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu/token/testPyCmf1eWhGPzi8EZZ2aeZ7xBP43N52PmD"] = balLTC
	stateDB["/account/ex/testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu/token/test8kHEKgHzQLs3AG2J5T1HLEoC8HhUFt6Qv"] = balETH
	stateDB["/account/ex/testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu/token/test3kgRHcxDPWTgVRc3Kkvs3JQ1QQ5foE7bi"] = balEOS
	stateDB["/account/ex/testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu/token/testPsjtk4XqCsktM7gfL6Vm54tVPfeabFurV"] = balUSDX
	stateDB["/account/ex/testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu/token/test8s6oGjxdFVxzbVjQDikQiTs3EUbhPCtPo"] = balBCB
	stateDB["/account/ex/testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu/token/test6g6FXQjkSLmELnmatWcTwDUv4SigQA8wR"] = balDC

	stateBuff = make(map[string][]byte)
}

func commitState() {
	for k, v := range stateBuff {
		stateDB[k] = v
	}
	stateBuff = make(map[string][]byte)
}

func rollbackState() {
	stateBuff = make(map[string][]byte)
}

func setToDB(key string, value []byte) {
	stateBuff[key] = value
}

func getBlock(transID, height int64) std.Block {
	result := new(std.Block)

	if v, ok := stateBuff[std.KeyOfAppState()]; ok {
		err := jsoniter.Unmarshal(v, result)
		if err != nil {
			panic(err)
		}
	} else if v, ok := stateDB[std.KeyOfAppState()]; ok {
		err := jsoniter.Unmarshal(v, result)
		if err != nil {
			panic(err)
		}
	}

	return *result
}

func build(transID, txID int64, meta std.ContractMeta) (result std.BuildResult) {
	return
}

func sdbGet(transID, txID int64, key string) []byte {
	result := std.GetResult{Msg: "ok"}
	if data, ok := stateBuff[key]; ok == true {
		result.Code = types.CodeOK
		result.Data = data
	} else {
		if data, ok := stateDB[key]; ok == true {
			result.Code = types.CodeOK
			result.Data = data
		} else {
			result.Code = types.ErrInvalidParameter
			result.Msg = "invalid key"
		}
	}

	resBytes, _ := jsoniter.Marshal(result)

	return resBytes
}

func sdbSet(transID, txID int64, values map[string][]byte) {
	for k, v := range values {
		stateBuff[k] = v
	}
}

// SdbGet 供合约运行服务的测试程序使用
func SdbGet(transID, txID int64, key string) []byte {
	result := std.GetResult{Msg: "ok"}
	if data, ok := stateBuff[key]; ok == true {
		result.Code = types.CodeOK
		result.Data = data
	} else {
		if data, ok := stateDB[key]; ok == true {
			result.Code = types.CodeOK
			result.Data = data
		} else {
			result.Code = types.ErrInvalidParameter
			result.Msg = "invalid key"
		}
	}

	resBytes, _ := jsoniter.Marshal(result)

	return resBytes
}

//SdbSet set sdb
func SdbSet(transID, txID int64, values map[string][]byte) {
	for k, v := range values {
		stateBuff[k] = v
	}
}

//GetBlock get block data
func GetBlock(numTxs int32) []byte {
	BlockHeight++

	block := std.Block{
		ChainID:         utChainID,
		BlockHash:       sha3.Sum256(big.NewInt(BlockHeight).Bytes()),
		Height:          BlockHeight,
		Time:            time.Now().Unix(),
		NumTxs:          numTxs,
		DataHash:        sha3.Sum256(big.NewInt(BlockHeight + 150000000000000).Bytes()),
		ProposerAddress: CalcAccountFromPubKey([]byte("pp123456789012345678901234567890")),
		RewardAddress:   CalcAccountFromPubKey([]byte("rw123456789012345678901234567890")),
		RandomNumber:    sha3.Sum256(big.NewInt(BlockHeight + 983377333372898).Bytes()),
		Version:         "",
		LastBlockHash:   sha3.Sum256(big.NewInt(BlockHeight - 1).Bytes()),
		LastCommitHash:  sha3.Sum256(big.NewInt(BlockHeight + 100000000000000).Bytes()),
		LastAppHash:     sha3.Sum256(big.NewInt(BlockHeight + 200000000000000).Bytes()),
		LastFee:         int64(LastNumTxs) * 500 * 2500}

	resBytes, err := jsoniter.Marshal(block)
	if err != nil {
		panic(err.Error())
	}
	LastNumTxs = numTxs

	return resBytes
}

func TransferFunc(smc sdk.ISmartContract, tokenAddr, to types.Address, value bn.Number) ([]types.KVPair, types.Error) {

	sdk.Require(value.IsGreaterThanI(0),
		types.ErrInvalidParameter, "Value must greater than zero")

	// 1、smc sender
	sender := smc.Message().Contract().Account()

	from := sender.Address()
	sdk.Require(from != to,
		types.ErrInvalidParameter, "Cannot transfer to self")

	// 2、检查sender的tokenAddr余额够不够
	bal := sender.BalanceOfToken(tokenAddr)
	balTo := smc.Helper().AccountHelper().AccountOf(to).BalanceOfToken(tokenAddr)

	sdk.Require(bal.IsGE(value),
		types.ErrInsufficientBalance, "Insufficient balance")

	// 3、sender余额减去value
	bal = bal.Sub(value)
	balTo = balTo.Add(value)

	// 4、to的余额加上value
	balFrom := std.AccountInfo{Address: tokenAddr, Balance: bal}
	balToA := std.AccountInfo{Address: tokenAddr, Balance: balTo}

	UTP.ISmartContract.(*sdkimpl.SmartContract).LlState().Set(std.KeyOfAccountToken(sender.Address(), tokenAddr), balFrom)
	UTP.ISmartContract.(*sdkimpl.SmartContract).LlState().Set(std.KeyOfAccountToken(to, tokenAddr), balToA)

	//stateBuff[std.KeyOfAccountToken(sender.Address(), tokenAddr)] = balJson
	//stateBuff[std.KeyOfAccountToken(to, tokenAddr)] = balToJson

	old := smc.Message().(*object.Message).OutputReceipts()
	// 5、发收据
	smc.Helper().ReceiptHelper().Emit(std.Transfer{
		Token: tokenAddr,
		From:  from,
		To:    to,
		Value: value,
	})
	now := smc.Message().(*object.Message).OutputReceipts()

	receipts := now[len(old):]
	return receipts, types.Error{ErrorCode: types.CodeOK}
}

//NextBlock generate next block data
func NextBlock(_numTxs int32) []byte {

	BlockHeight++
	stdBlock := std.Block{
		ChainID:         utChainID,
		BlockHash:       sha3.Sum256(big.NewInt(BlockHeight).Bytes()),
		Height:          BlockHeight,
		Time:            time.Now().Unix(),
		NumTxs:          _numTxs,
		DataHash:        sha3.Sum256(big.NewInt(BlockHeight + 150000000000000).Bytes()),
		ProposerAddress: CalcAccountFromPubKey([]byte("pp123456789012345678901234567890")),
		RewardAddress:   CalcAccountFromPubKey([]byte("rw123456789012345678901234567890")),
		RandomNumber:    sha3.Sum256(big.NewInt(BlockHeight + 983377333372898).Bytes()),
		Version:         "",
		LastBlockHash:   sha3.Sum256(big.NewInt(BlockHeight - 1).Bytes()),
		LastCommitHash:  sha3.Sum256(big.NewInt(BlockHeight + 100000000000000).Bytes()),
		LastAppHash:     sha3.Sum256(big.NewInt(BlockHeight + 200000000000000).Bytes()),
		LastFee:         int64(LastNumTxs) * 500 * 2500,
	}
	block := object.NewBlockFromSTD(UTP.ISmartContract.(*sdkimpl.SmartContract), &stdBlock)

	resBytes, err := jsoniter.Marshal(stdBlock)
	if err != nil {
		panic(err.Error())
	}
	LastNumTxs = _numTxs

	smc := UTP.ISmartContract.(*sdkimpl.SmartContract)
	smc.SetBlock(block)

	key := fmt.Sprintf("/block/%d", block.Height())
	b := make(map[string][]byte)

	b[key], err = jsoniter.Marshal(block)
	if err != nil {
		panic(err.Error())
	}
	sdbSet(0, 0, b)
	return resBytes
}

//NextBlockEx generate next block data
func NextBlockEx(_numTxs int32,
	height, t, lastFee int64,
	blockHash, dataHash, lastBlockHash, lastCommitHash, lastAppHash types.Hash,
	proposerAddress, rewardAddress types.Address,
	randomNumber types.HexBytes) []byte {

	if height != 0 {
		BlockHeight = height
	} else {
		BlockHeight++
	}
	if t == 0 {
		t = time.Now().Unix()
	}
	if lastFee == 0 {
		lastFee = int64(LastNumTxs) * 500 * 2500
	}

	if blockHash == nil {
		blockHash = sha3.Sum256(big.NewInt(BlockHeight).Bytes())
	}
	if dataHash == nil {
		dataHash = sha3.Sum256(big.NewInt(BlockHeight + 150000000000000).Bytes())
	}
	if lastBlockHash == nil {
		lastBlockHash = sha3.Sum256(big.NewInt(BlockHeight + 983377333372898).Bytes())
	}
	if lastCommitHash == nil {
		lastCommitHash = sha3.Sum256(big.NewInt(BlockHeight - 1).Bytes())
	}
	if lastAppHash == nil {
		lastAppHash = sha3.Sum256(big.NewInt(BlockHeight + 100000000000000).Bytes())
	}
	if randomNumber == nil {
		randomNumber = sha3.Sum256(big.NewInt(BlockHeight + 200000000000000).Bytes())
	}

	if proposerAddress == "" {
		proposerAddress = CalcAccountFromPubKey([]byte("pp123456789012345678901234567890"))
	}
	if rewardAddress == "" {
		rewardAddress = CalcAccountFromPubKey([]byte("rw123456789012345678901234567890"))
	}

	stdBlock := std.Block{
		ChainID:         utChainID,
		BlockHash:       blockHash,
		Height:          BlockHeight,
		Time:            t,
		NumTxs:          _numTxs,
		DataHash:        dataHash,
		ProposerAddress: proposerAddress,
		RewardAddress:   rewardAddress,
		RandomNumber:    randomNumber,
		Version:         "",
		LastBlockHash:   lastBlockHash,
		LastCommitHash:  lastCommitHash,
		LastAppHash:     lastAppHash,
		LastFee:         lastFee,
	}
	block := object.NewBlockFromSTD(UTP.ISmartContract.(*sdkimpl.SmartContract), &stdBlock)

	resBytes, err := jsoniter.Marshal(stdBlock)
	if err != nil {
		panic(err.Error())
	}
	LastNumTxs = _numTxs

	smc := UTP.ISmartContract.(*sdkimpl.SmartContract)
	smc.SetBlock(block)

	key := fmt.Sprintf("/block/%d", block.Height())
	b := make(map[string][]byte)

	b[key], err = jsoniter.Marshal(block)
	if err != nil {
		panic(err.Error())
	}
	sdbSet(0, 0, b)
	return resBytes
}

//NextBlock generate next block data
func NextBlockOfHeight(height, _numTxs int32) []byte {

	BlockHeight += int64(height)
	block := object.NewBlock(UTP.ISmartContract.(*sdkimpl.SmartContract),
		utChainID,
		"",
		sha3.Sum256(big.NewInt(BlockHeight).Bytes()),
		sha3.Sum256(big.NewInt(BlockHeight+150000000000000).Bytes()),
		BlockHeight,
		time.Now().Unix(),
		_numTxs,
		CalcAccountFromPubKey([]byte("pp123456789012345678901234567890")),
		CalcAccountFromPubKey([]byte("rw123456789012345678901234567890")),
		sha3.Sum256(big.NewInt(BlockHeight+983377333372898).Bytes()),
		sha3.Sum256(big.NewInt(BlockHeight-1).Bytes()),
		sha3.Sum256(big.NewInt(BlockHeight+100000000000000).Bytes()),
		sha3.Sum256(big.NewInt(BlockHeight+200000000000000).Bytes()),
		int64(LastNumTxs)*500*2500)

	resBytes, err := jsoniter.Marshal(block)
	if err != nil {
		panic(err.Error())
	}
	LastNumTxs = _numTxs

	smc := UTP.ISmartContract.(*sdkimpl.SmartContract)
	smc.SetBlock(block)

	key := fmt.Sprintf("/block/%d", block.Height())
	b := make(map[string][]byte)

	b[key], err = jsoniter.Marshal(block)
	if err != nil {
		panic(err.Error())
	}
	sdbSet(0, 0, b)
	return resBytes
}

func data(key string, resBytes []byte) []byte {
	var getResult std.GetResult
	err := jsoniter.Unmarshal(resBytes, &getResult)
	if err != nil {
		sdkimpl.Logger.Fatalf("Cannot unmarshal get result struct, key=%s, error=%v\nbytes=%v", key, err, resBytes)
		sdkimpl.Logger.Flush()
		panic(err)
	} else if getResult.Code != types.CodeOK {
		sdkimpl.Logger.Debugf("Cannot find key=%s in stateBuff, error=%s", getResult.Msg)
		return nil
	}

	return getResult.Data
}
