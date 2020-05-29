package statedb

import (
	"bytes"
	"sort"
)

type Tx struct {
	txID        int64
	buffer      map[string][]byte
	transaction *Transaction
}

func (t *Tx) ID() int64 {
	return t.txID
}

func (t *Tx) Get(key string) []byte {
	return t.buffer[key]
}

func (t *Tx) Set(key string, value []byte) {
	t.buffer[key] = value
}

func (t *Tx) BatchSet(data map[string][]byte) {
	for k, v := range data {
		t.buffer[k] = v
	}
}

func (t *Tx) Commit() ([]byte, map[string][]byte) {
	var keys []string

	// commit to transaction
	for k, v := range t.buffer {
		keys = append(keys, k)
		t.transaction.buffer[k] = v
	}

	sort.Strings(keys)
	var buf bytes.Buffer
	for _, k := range keys {
		v := t.buffer[k]
		buf.Write([]byte(k))
		buf.Write(v)
	}

	bufMap := t.buffer
	t.buffer = make(map[string][]byte)

	return buf.Bytes(), bufMap
}

func (t *Tx) Rollback() {
	t.buffer = make(map[string][]byte)
}
