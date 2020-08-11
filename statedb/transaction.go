package statedb

import (
	"bytes"
	"sort"

	"fmt"
	"github.com/bcbchain/bclib/jsoniter"
	"sync/atomic"
)

type Transaction struct {
	transactionID  int64
	stateDB        *StateDB
	committable    bool
	maxTxCount     int
	goRoutineCount int
	wBuffer        map[string][]byte
	wBitsMerged    *conflictBits
	rBuffer        *kvBuffer
	lastTxID       int64
}

func (trans *Transaction) ID() int64 {
	return trans.transactionID
}

func (trans *Transaction) NewTx(f TxFunction, params ...interface{}) (tx *Tx) {
	tx = &Tx{
		txID:        trans.calcTxID(),
		wBuffer:     make(map[string][]byte),
		rBuffer:     make(map[string][]byte),
		wBits:       newConflictBits(trans.maxTxCount * 256),
		rBits:       newConflictBits(trans.maxTxCount * 256),
		txFunc:      f,
		txParams:    params,
		transaction: trans,
	}
	return
}

func (trans *Transaction) calcTxID() int64 {
	return atomic.AddInt64(&trans.lastTxID, 1)
}

func (trans *Transaction) Get(key string) []byte {
	var err error
	value := make([]byte, 0)
	ok := false
	if value, ok = trans.wBuffer[key]; !ok {
		if value, ok = trans.rBuffer.get(key); !ok {
			value, err = trans.stateDB.sdb.Get([]byte(key))
			if err != nil {
				panic(err)
			}
			trans.rBuffer.set(key, value)
		}
	}
	return value
}

func (trans *Transaction) Set(key string, value []byte) {
	trans.wBuffer[key] = value
}

func (trans *Transaction) BatchSet(data map[string][]byte) {
	for k, v := range data {
		trans.wBuffer[k] = v
	}
}

func (trans *Transaction) Exec(tx *Tx) {
	txs := make([]*Tx, 0)
	txs = append(txs, tx)
	trans.GoBatchExec(txs)
}

func (trans *Transaction) GoBatchExec(txs []*Tx) {
	for txs != nil {
		txs = trans.exec(txs)
	}
	return
}

func _run_tx(tx *Tx) {
	tx.exec()
	tx.end()
}

func run_tx(tx *Tx) {
	tx.begin()
	go _run_tx(tx)
}

func (trans *Transaction) exec(txs []*Tx) []*Tx {
	subtxs := make([]*Tx, 0)
	go_num := 0
	for _, tx := range txs {
		subtxs = append(subtxs, tx)
		go_num++
		if go_num >= trans.goRoutineCount {
			break
		}
	}
	if go_num == 0 {
		return nil
	}

	trans.wBitsMerged = subtxs[0].wBits
	for subtxs != nil {
		subtxs = trans._exec(subtxs)
	}

	return append(make([]*Tx, 0), txs[go_num:]...)
}

func (trans *Transaction) _exec(txs []*Tx) []*Tx {
	if txs == nil || len(txs) == 0 {
		return nil
	}

	gotxs := make([]*Tx, 0)
	for _, tx := range txs {
		if tx.done == false {
			gotxs = append(gotxs, tx)
		}
	}

	for i, tx := range gotxs {
		if i > 0 {
			tx.prevDoneEvent = &(gotxs[i-1].doneEvent)
		}
		run_tx(tx)
	}
	last := len(gotxs) - 1
	gotxs[last].doneEvent.Wait()

	return trans.mergeTxResult(txs)
}

func (trans *Transaction) mergeTxResult(txs []*Tx) []*Tx {
	trans.wBitsMerged = trans.wBitsMerged.Merge(txs[0].wBits)
	last_no_conflict := 0

	for i := 1; i < len(txs); i++ {
		tx := txs[i]
		if tx.rBits.IsConflictTo(trans.wBitsMerged) {
			//conflict tx
			tx.Rollback()
			break
		} else if tx.doneSuccess {
			trans.wBitsMerged = trans.wBitsMerged.Merge(tx.wBits)
		} else {
			// tx exec failed
		}
		last_no_conflict++
	}

	for i := 0; i <= last_no_conflict; i++ {
		tx := txs[i]
		if tx.doneSuccess {
			// only commit the tx exec succeed
			tx.commit()
		} else {
			tx.reset()
		}
	}
	if last_no_conflict == len(txs)-1 {
		return nil
	} else {
		return append(make([]*Tx, 0), txs[last_no_conflict+1:]...)
	}
}

func (trans *Transaction) Commit() {
	if !trans.committable {
		panic("can not commit rollback transaction")
	}

	// check current transaction ID
	trans.checkID()

	batch := trans.stateDB.sdb.NewBatch()
	originData := make(map[string][]byte, len(trans.wBuffer))

	for k, v := range trans.wBuffer {

		// get origin data
		if value, err := trans.stateDB.sdb.Get([]byte(k)); err != nil {
			panic(err)
		} else {
			originData[k] = value
		}

		// set new data to state db
		if len(v) == 0 {
			batch.Delete([]byte(k))
		} else {
			batch.Set([]byte(k), v)
		}
	}

	// snapshot
	trans.stateDB.snapshot.commit(trans.transactionID, originData, trans.wBuffer)

	// set last transaction ID
	value, err := jsoniter.Marshal(trans.transactionID)
	if err != nil {
		panic(err)
	}
	batch.Set([]byte(keyOfLastTransactionID()), value)

	// commit state db
	err = batch.Commit()
	if err != nil {
		panic(err)
	}

	trans.rBuffer.reset()
	trans.stateDB.committableTransaction = nil
}

func (trans *Transaction) Rollback() {
	trans.wBuffer = make(map[string][]byte)
	trans.rBuffer = newKVbuffer(trans.rBuffer.maxCacheSize)

	if trans.committable {
		trans.stateDB.committableTransaction = nil
	}
}

func (trans *Transaction) GetBuffer() []byte {
	var keys []string
	for k := range trans.wBuffer {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	for _, k := range keys {
		v := trans.wBuffer[k]
		buf.Write([]byte(k))
		buf.Write(v)
	}
	return buf.Bytes()
}

func (trans *Transaction) checkID() {
	value, err := trans.stateDB.sdb.Get([]byte(keyOfLastTransactionID()))
	if err != nil {
		panic(err)
	}

	if len(value) == 0 {
		if trans.transactionID != 1 {
			panic("first transaction ID must be 1")
		}

	} else {
		var lastID int64
		if err := jsoniter.Unmarshal(value, &lastID); err != nil {
			panic(err)
		}
		if trans.transactionID != lastID+1 {
			panic(fmt.Sprintf("transaction ID must be last transaction ID plus one, ID:%d, last ID %d", trans.transactionID, lastID))
		}
	}
}
