package core

import (
	"fmt"

	"encoding/hex"

	abci "github.com/tendermint/abci/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	sm "github.com/tendermint/tendermint/state"
	"github.com/tendermint/tendermint/state/txindex/null"
	"github.com/tendermint/tendermint/types"
	cmn "github.com/tendermint/tmlibs/common"
	tmquery "github.com/tendermint/tmlibs/pubsub/query"
)

// Tx allows you to query the transaction results. `nil` could mean the
// transaction is in the mempool, invalidated, or was not sent in the first
// place.
//
// ```shell
// curl "localhost:46657/tx?hash=0x2B8EC32BA2579B3B8606E42C06DE2F7AFA2556EF"
// ```
//
// ```go
// client := client.NewHTTP("tcp://0.0.0.0:46657", "/websocket")
// tx, err := client.Tx([]byte("2B8EC32BA2579B3B8606E42C06DE2F7AFA2556EF"), true)
// ```
//
// > The above command returns JSON structured like this:
//
// ```json
// {
// 	"error": "",
// 	"result": {
// 		"proof": {
// 			"Proof": {
// 				"aunts": []
// 			},
// 			"Data": "YWJjZA==",
// 			"RootHash": "2B8EC32BA2579B3B8606E42C06DE2F7AFA2556EF",
// 			"Total": 1,
// 			"Index": 0
// 		},
// 		"tx": "YWJjZA==",
// 		"tx_result": {
// 			"log": "",
// 			"data": "",
// 			"code": 0
// 		},
// 		"index": 0,
// 		"height": 52,
//		"hash": "2B8EC32BA2579B3B8606E42C06DE2F7AFA2556EF"
// 	},
// 	"id": "",
// 	"jsonrpc": "2.0"
// }
// ```
//
// Returns a transaction matching the given transaction hash.
//
// ### Query Parameters
//
// | Parameter | Type   | Default | Required | Description                                               |
// |-----------+--------+---------+----------+-----------------------------------------------------------|
// | hash      | []byte | nil     | true     | The transaction hash                                      |
// | prove     | bool   | false   | false    | Include a proof of the transaction inclusion in the block |
//
// ### Returns
//
// - `proof`: the `types.TxProof` object
// - `tx`: `[]byte` - the transaction
// - `tx_result`: the `abci.Result` object
// - `index`: `int` - index of the transaction
// - `height`: `int` - height of the block where this transaction was in
// - `hash`: `[]byte` - hash of the transaction
//func Tx(hash []byte, prove bool) (*ctypes.ResultTx, error) {
//
//	// if index is disabled, return error
//	if _, ok := txIndexer.(*null.TxIndex); ok {
//		return nil, fmt.Errorf("Transaction indexing is disabled")
//	}
//
//	r, err := txIndexer.Get(hash)
//	if err != nil {
//		return nil, err
//	}
//
//	if r == nil {
//		return nil, fmt.Errorf("Tx (%X) not found", hash)
//	}
//
//	height := r.Height
//	index := r.Index
//
//	var proof types.TxProof
//	if prove {
//		block := blockStore.LoadBlock(height)
//		proof = block.Data.Txs.Proof(int(index)) // XXX: overflow on 32-bit machines
//	}
//
//	return &ctypes.ResultTx{
//		Hash:     hash,
//		Height:   height,
//		Index:    uint32(index),
//		TxResult: r.Result,
//		Tx:       r.Tx,
//		Proof:    proof,
//	}, nil
//}

/*
  stateCode = 1   checkTx中
  stateCode = 2   checkTx完成
*/
func Tx(hash string, prove bool) (*ctypes.ResultTx, error) {

	var stateCode uint32
	var height int64
	check := true
	deTx, _ := hex.DecodeString(hash)
	dResult, err := sm.LoadABCITxResponses(stateDB, cmn.HexBytes(deTx))
	if err == nil {
		height = dResult.Height
	}
	var checkResult abci.ResponseCheckTx
	checkRes, errCheck := mempool.GiTxSearch(hash)
	if errCheck == nil {
		if checkRes != nil {
			checkResult = *checkRes
		}
	} else {
		check = false
	}

	if dResult.Height == 0 {
		if checkRes == nil {
			if !check {
				return nil, errCheck //没有查到交易 或者交易check失败且缓存失效
			} else {
				stateCode = 1
				checkResult = abci.ResponseCheckTx{}
			}
		} else if checkRes.Code == 2018 {
			checkResult = abci.ResponseCheckTx{}
			stateCode = 1 //正在check 返回返回checkRes为空
		} else {
			stateCode = 2 //check完成   返回checkRes
		}
	} else {
		if checkRes == nil {
			if !check {
				stateCode = 3 // deliver成功   返回deliver 不返回返回checkRes    check缓存失效
				checkResult = abci.ResponseCheckTx{}
			} else {
				stateCode = 5 //不存在的情况  出现则为bug
			}
		} else {
			stateCode = 4 //deliver和check结构都返回
		}

	}

	return &ctypes.ResultTx{
		Hash:   string(hash),
		Height: height,
		//Index:    uint32(index),
		DeliverResult: dResult,
		CheckResult:   checkResult,
		StateCode:     stateCode,
	}, nil
}

// TxSearch allows you to query for multiple transactions results.
//
// ```shell
// curl "localhost:46657/tx_search?query=\"account.owner='Ivan'\"&prove=true"
// ```
//
// ```go
// client := client.NewHTTP("tcp://0.0.0.0:46657", "/websocket")
// q, err := tmquery.New("account.owner='Ivan'")
// tx, err := client.TxSearch(q, true)
// ```
//
// > The above command returns JSON structured like this:
//
// ```json
// {
//   "result": [
//     {
//       "proof": {
//         "Proof": {
//           "aunts": [
//             "J3LHbizt806uKnABNLwG4l7gXCA=",
//             "iblMO/M1TnNtlAefJyNCeVhjAb0=",
//             "iVk3ryurVaEEhdeS0ohAJZ3wtB8=",
//             "5hqMkTeGqpct51ohX0lZLIdsn7Q=",
//             "afhsNxFnLlZgFDoyPpdQSe0bR8g="
//           ]
//         },
//         "Data": "mvZHHa7HhZ4aRT0xMDA=",
//         "RootHash": "F6541223AA46E428CB1070E9840D2C3DF3B6D776",
//         "Total": 32,
//         "Index": 31
//       },
//       "tx": "mvZHHa7HhZ4aRT0xMDA=",
//       "tx_result": {},
//       "index": 31,
//       "height": 12,
//       "hash": "2B8EC32BA2579B3B8606E42C06DE2F7AFA2556EF"
//     }
//   ],
//   "id": "",
//   "jsonrpc": "2.0"
// }
// ```
//
// Returns transactions matching the given query.
//
// ### Query Parameters
//
// | Parameter | Type   | Default | Required | Description                                               |
// |-----------+--------+---------+----------+-----------------------------------------------------------|
// | query     | string | ""      | true     | Query                                                     |
// | prove     | bool   | false   | false    | Include proofs of the transactions inclusion in the block |
//
// ### Returns
//
// - `proof`: the `types.TxProof` object
// - `tx`: `[]byte` - the transaction
// - `tx_result`: the `abci.Result` object
// - `index`: `int` - index of the transaction
// - `height`: `int` - height of the block where this transaction was in
// - `hash`: `[]byte` - hash of the transaction
func TxSearch(query string, prove bool) ([]*ctypes.ResultTx, error) {
	// if index is disabled, return error
	if _, ok := txIndexer.(*null.TxIndex); ok {
		return nil, fmt.Errorf("Transaction indexing is disabled")
	}

	q, err := tmquery.New(query)
	if err != nil {
		return nil, err
	}

	results, err := txIndexer.Search(q)
	if err != nil {
		return nil, err
	}

	// TODO: we may want to consider putting a maximum on this length and somehow
	// informing the user that things were truncated.
	apiResults := make([]*ctypes.ResultTx, len(results))
	var proof types.TxProof
	for i, r := range results {
		height := r.Height
		index := r.Index

		if prove {
			block := blockStore.LoadBlock(height)
			proof = block.Data.Txs.Proof(int(index)) // XXX: overflow on 32-bit machines
		}

		apiResults[i] = &ctypes.ResultTx{
			Hash:          string(r.Tx.Hash()),
			Height:        height,
			Index:         index,
			DeliverResult: r.Result,
			Tx:            r.Tx,
			Proof:         proof,
		}
	}

	return apiResults, nil
}
