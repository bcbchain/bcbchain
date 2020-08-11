package statedb

import (
	"bytes"
	"errors"
	"sort"
	"sync"
)

type TxFunction func(tx *Tx, params ...interface{}) bool

type Tx struct {
	txID    int64
	wBuffer map[string][]byte
	rBuffer map[string][]byte
	wBits   *conflictBits
	rBits   *conflictBits

	txFunc        TxFunction
	txParams      []interface{}
	done          bool
	doneSuccess   bool
	doneEvent     sync.WaitGroup
	prevDoneEvent *sync.WaitGroup

	transaction *Transaction

	export_buffer1 []byte
	export_buffer2 map[string][]byte
}

func (t *Tx) ID() int64 {
	return t.txID
}

func (t *Tx) Transaction() *Transaction {
	return t.transaction
}

func (tx *Tx) Get(key string) []byte {
	value := make([]byte, 0)
	ok := false
	if value, ok = tx.wBuffer[key]; !ok {
		if value, ok = tx.rBuffer[key]; !ok {
			value = tx.transaction.Get(key)
			tx.rBuffer[key] = value
			tx.rBits.Set([]byte(key))
		}
	}
	return value
}

func (tx *Tx) Set(key string, value []byte) {
	tx.wBuffer[key] = value
	tx.wBits.Set([]byte(key))
}

func (tx *Tx) GetBuffer() ([]byte, map[string][]byte) {
	return tx.export_buffer1, tx.export_buffer2
}

func (tx *Tx) BatchSet(data map[string][]byte) {
	for key, val := range data {
		tx.wBuffer[key] = val
		tx.wBits.Set([]byte(key))
	}
}
func (tx *Tx) begin() {
	tx.doneEvent.Add(1)
}

func (tx *Tx) end() {
	if tx.prevDoneEvent != nil {
		tx.prevDoneEvent.Wait()
	}
	tx.done = true
	tx.doneEvent.Done()
}

func (tx *Tx) exec() {
	if tx.txFunc == nil {
		panic(errors.New("No tx function to execute!"))
	}

	//executing function of tx
	tx.doneSuccess = tx.txFunc(tx, tx.txParams...)
}

func (tx *Tx) commit() {
	// commit to transaction
	tx.transaction.BatchSet(tx.wBuffer)

	var keys []string
	for k, _ := range tx.wBuffer {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	for _, k := range keys {
		v := tx.wBuffer[k]
		buf.Write([]byte(k))
		buf.Write(v)
	}

	tx.export_buffer1 = buf.Bytes()
	tx.export_buffer2 = tx.wBuffer
	tx.reset()
}

func (tx *Tx) reset() {
	tx.wBuffer = make(map[string][]byte)
	tx.rBuffer = make(map[string][]byte)
	tx.wBits.Clear()
	tx.rBits.Clear()
}

func (tx *Tx) Commit() ([]byte, map[string][]byte) {
	tx.commit()
	return tx.export_buffer1, tx.export_buffer2
}

func (tx *Tx) Rollback() {
	tx.reset()
	tx.done = false
	tx.doneSuccess = false
}
