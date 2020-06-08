package statedb

import (
	"bytes"
	"github.com/bcbchain/bclib/jsoniter"
	"fmt"
	"sort"
	"sync"
)

var muts sync.Mutex

type Transaction struct {
	transactionID int64
	stateDB       *StateDB
	buffer        map[string][]byte
	committable   bool
	lastTxID      int64
}

func (t *Transaction) ID() int64 {
	return t.transactionID
}

func (t *Transaction) NewTx() *Tx {
	return &Tx{
		txID:        t.calcTxID(),
		buffer:      make(map[string][]byte),
		transaction: t,
	}
}

func (t *Transaction) calcTxID() int64 {
	muts.Lock()
	defer muts.Unlock()

	t.lastTxID++
	return t.lastTxID
}

func (t *Transaction) Get(key string) []byte {
	if value, ok := t.buffer[key]; ok {
		return value
	}

	value, err := t.stateDB.sdb.Get([]byte(key))
	if err != nil {
		panic(err)
	}
	return value
}

func (t *Transaction) Set(key string, value []byte) {
	t.buffer[key] = value
}

func (t *Transaction) BatchSet(data map[string][]byte) {
	for k, v := range data {
		t.buffer[k] = v
	}
}

func (t *Transaction) Commit() {
	if !t.committable {
		panic("can not commit rollback transaction")
	}

	// check current transaction ID
	t.checkID()

	batch := t.stateDB.sdb.NewBatch()
	originData := make(map[string][]byte, len(t.buffer))

	for k, v := range t.buffer {

		// get origin data
		if value, err := t.stateDB.sdb.Get([]byte(k)); err != nil {
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
	t.stateDB.snapshot.commit(t.transactionID, originData, t.buffer)

	// set last transaction ID
	value, err := jsoniter.Marshal(t.transactionID)
	if err != nil {
		panic(err)
	}
	batch.Set([]byte(keyOfLastTransactionID()), value)

	// commit state db
	err = batch.Commit()
	if err != nil {
		panic(err)
	}

	t.stateDB.committableTransaction = nil
}

func (t *Transaction) Rollback() {
	t.buffer = make(map[string][]byte)
}

func (t *Transaction) GetBuffer() []byte {
	var keys []string

	for k := range t.buffer {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer

	for _, k := range keys {
		v := t.buffer[k]
		buf.Write([]byte(k))
		buf.Write(v)
	}

	return buf.Bytes()
}

func (t *Transaction) checkID() {
	value, err := t.stateDB.sdb.Get([]byte(keyOfLastTransactionID()))
	if err != nil {
		panic(err)
	}

	if len(value) == 0 {
		if t.transactionID != 1 {
			panic("first transaction ID must be 1")
		}

	} else {
		var lastID int64
		if err := jsoniter.Unmarshal(value, &lastID); err != nil {
			panic(err)
		}
		if t.transactionID != lastID+1 {
			panic(fmt.Sprintf("transaction ID must be last transaction ID plus one, ID:%d, last ID %d", t.transactionID, lastID))
		}
	}
}
