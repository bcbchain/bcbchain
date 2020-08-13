package types

import "github.com/bcbchain/bclib/tendermint/abci/types"

type TxPool struct {
	TxChan chan []byte
}

func NewTxPool() *TxPool {
	return &TxPool{TxChan: make(chan []byte, 1000)}
}

type ResultPool struct {
	ResultChan chan types.Result
}

func NewResultPool() *ResultPool {
	return &ResultPool{ResultChan: make(chan types.Result, 1000)}
}

//记录Response的顺序，有序返回给socketsever
type ResponseChanOrder struct {
	Response types.ResponseCheckTx
	Index    int
}

type ResponsePool struct {
	ResponseOrder chan ResponseChanOrder
}

func NewResponsePool() *ResponsePool {
	return &ResponsePool{ResponseOrder: make(chan ResponseChanOrder, 1000)}
}
