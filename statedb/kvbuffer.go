package statedb

import (
	"crypto/md5"
	"github.com/bcbchain/sdk/sdk/bn"
	"sync"
)

func calcIndex(key []byte, maxCacheSize uint) int {
	h := md5.New()
	h.Write(bn.N(int64(0x8f25cb36)).Bytes())
	h.Write(key)
	return int(bn.NBytes(h.Sum(nil)).ModI(int64(maxCacheSize)).Value().Int64())
}

type kvItem struct {
	vals map[string][]byte
	mt   sync.Mutex
}

type kvBuffer struct {
	buffer       []kvItem
	maxCacheSize uint
}

func newKVbuffer(maxCacheSize uint) *kvBuffer {
	buf := &kvBuffer{
		buffer:       make([]kvItem, maxCacheSize),
		maxCacheSize: maxCacheSize,
	}
	for i, _ := range buf.buffer {
		buf.buffer[i].vals = make(map[string][]byte)
	}
	return buf
}

func (buf *kvBuffer) reset() {
	buf.buffer = make([]kvItem, 0)
}

func (buf *kvBuffer) getItem(key string) *kvItem {
	return &buf.buffer[calcIndex([]byte(key), buf.maxCacheSize)]
}

func (buf *kvBuffer) get(key string) ([]byte, bool) {
	item := buf.getItem(key)
	item.mt.Lock()
	defer item.mt.Unlock()

	if value, ok := item.vals[key]; ok {
		return value, true
	}
	return nil, false
}

func (buf *kvBuffer) set(key string, value []byte) {
	data := make(map[string][]byte)
	data[key] = value
	buf._batchSet(data)
}

func (buf *kvBuffer) batchSet(data map[string][]byte) {
	buf._batchSet(data)
}

func (buf *kvBuffer) _batchSet(data map[string][]byte) {
	for key, value := range data {
		item := buf.getItem(key)
		item.mt.Lock()
		item.vals[key] = value
		item.mt.Unlock()
	}
}
