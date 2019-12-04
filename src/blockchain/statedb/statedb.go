package statedb

import (
	"bytes"
	"common/bcdb"
	"fmt"
	"sort"
	"sync"
)

var (
	sdb           *bcdb.GILevelDB
	once          sync.Once
	mu            sync.Mutex
	transactionID int64
	idToTrans     sync.Map
)

type Transaction struct {
	TransBuffer map[string][]byte
	TxIDToTx    map[int64]*Tx
}

type Tx struct {
	TxBuffer map[string][]byte
	TxID     int64
}

func Init(name string, ip string, port string) (*bcdb.GILevelDB, bool) {
	var err error
	once.Do(func() {
		sdb, err = bcdb.OpenDB(name, ip, port)
	})

	return sdb, err == nil && sdb != nil
}

func NewTransaction() int64 {
	// Transaction could be concurrently.
	var transID int64
	{
		mu.Lock()
		transactionID += 1
		transID = transactionID
		defer mu.Unlock()
	}

	idToTrans.Store(transID, &Transaction{
		TransBuffer: make(map[string][]byte),
		TxIDToTx:    make(map[int64]*Tx)})

	return transID
}

func Get(transID int64, txID int64, key string) []byte {
	if transID == 0 {
		//Get data from sdb directly
		value, err := sdb.Get([]byte(key))
		if err != nil {
			panic(err)
		}
		return value
	}

	transTemp, ok := idToTrans.Load(transID)
	if !ok {
		panic(fmt.Sprintf("Invalid transID: %d", transID))
	}
	// Get data from tx buffer
	trans := transTemp.(*Transaction)
	if tx, ok := trans.TxIDToTx[txID]; ok {
		if value, ok := tx.TxBuffer[key]; ok {
			return value
		}
	}
	// Get data from trans buffer
	if value, ok := trans.TransBuffer[key]; ok {
		return value
	}
	//Get data from sdb directly
	value, err := sdb.Get([]byte(key))
	if err != nil {
		panic(err)
	}
	return value
}

func Set(transID int64, txID int64, key string, value []byte) {
	transTemp, ok := idToTrans.Load(transID)
	if !ok {
		panic(fmt.Sprintf("Invalid transID: %d", transID))
	}

	trans := transTemp.(*Transaction)

	tx, ok := trans.TxIDToTx[txID]
	if ok {
		tx.TxBuffer[key] = value
	} else {
		var tx Tx

		tx.TxBuffer = make(map[string][]byte)
		tx.TxID = txID

		tx.TxBuffer[key] = value

		trans.TxIDToTx[txID] = &tx
	}
}

// SetToTrans set value to trans cache, RollbackTx func won't rollback this value.
// Set account nonce using this func.
func SetToTrans(transID int64, key string, value []byte) {
	transTemp, ok := idToTrans.Load(transID)
	if !ok {
		panic("Invalid transID.")
	}

	trans := transTemp.(*Transaction)
	trans.TransBuffer[key] = value
}

func BatchSet(transID int64, txID int64, data map[string][]byte) {
	transTemp, ok := idToTrans.Load(transID)
	if !ok {
		panic(fmt.Sprintf("Invalid transID: %d", transID))
	}

	trans := transTemp.(*Transaction)

	tx, ok := trans.TxIDToTx[txID]
	if ok {
		for k, v := range data {
			tx.TxBuffer[k] = v
		}
	} else {
		var tx Tx

		tx.TxBuffer = make(map[string][]byte)
		tx.TxID = txID

		for k, v := range data {
			tx.TxBuffer[k] = v
		}

		trans.TxIDToTx[txID] = &tx
	}
}

func RollbackTx(transID int64, txID int64) {
	transTemp, ok := idToTrans.Load(transID)
	if !ok {
		panic(fmt.Sprintf("Invalid transID: %d", transID))
	}

	trans := transTemp.(*Transaction)

	_, ok = trans.TxIDToTx[txID]
	if !ok {
		panic(fmt.Sprintf("Invalid txID: %d", txID))
	}

	delete(trans.TxIDToTx, txID)
}

func CommitTx(transID int64, txID int64) ([]byte, map[string][]byte) {
	transTemp, ok := idToTrans.Load(transID)
	if !ok {
		panic(fmt.Sprintf("Invalid transID: %d", transID))
	}
	trans := transTemp.(*Transaction)

	tx, ok := trans.TxIDToTx[txID]
	if !ok {
		//panic(fmt.Sprintf("Invalid txID: %d", txID))
		return nil, nil
	}

	var keys []string
	for k, v := range tx.TxBuffer {
		trans.TransBuffer[k] = v
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var buf bytes.Buffer
	for _, k := range keys {
		v := tx.TxBuffer[k]
		buf.Write([]byte(k))
		buf.Write(v)
	}

	delete(trans.TxIDToTx, txID)

	return buf.Bytes(), tx.TxBuffer
}

func CommitTx2V1(transID int64, txBuffer map[string][]byte) {
	transTemp, ok := idToTrans.Load(transID)
	if !ok {
		panic(fmt.Sprintf("Invalid transID: %d", transID))
	}
	trans := transTemp.(*Transaction)

	for k, v := range txBuffer {
		trans.TransBuffer[k] = v
	}
}

func Commit(transID int64) {

	transTemp, ok := idToTrans.Load(transID)
	if !ok {
		panic(fmt.Sprintf("Invalid transID: %d", transID))
	}

	trans := transTemp.(*Transaction)

	batch := sdb.NewBatch()

	for k, v := range trans.TransBuffer {
		if len(v) == 0 {
			batch.Delete([]byte(k))
		} else {
			batch.Set([]byte(k), v)
		}
	}

	err := batch.Commit()
	if err != nil {
		panic(err)
	}

	idToTrans.Delete(transID)
}

func Rollback(transID int64) {
	transaction, ok := idToTrans.Load(transID)
	if !ok {
		panic(fmt.Sprintf("Invalid transID: %d", transID))
	}

	_, ok = transaction.(*Transaction)
	if !ok {
		panic(fmt.Sprintf("Invalid transID: %d", transID))
	}

	idToTrans.Delete(transID)
}
