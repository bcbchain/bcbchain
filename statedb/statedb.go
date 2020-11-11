package statedb

import (
	"github.com/bcbchain/bclib/bcdb"
	"github.com/bcbchain/bclib/jsoniter"
	"runtime"
	"sync"
	"sync/atomic"
)

type StateDB struct {
	sdb      *bcdb.GILevelDB // state db
	snapshot *snapshot       // snapshot db

	committableTransaction *Transaction // current committable transaction

	lastCommittableTransactionID int64
	lastRollbackTransactionID    int64
}

func New(sdbName string, maxSnapshotCount int) *StateDB {
	if maxSnapshotCount < 0 {
		panic("invalid parameter")
	}

	// open state db
	sdb, err := bcdb.OpenDB(sdbName, "", "")
	if err != nil {
		panic(err)
	}

	// open snapshot db
	sndb, err := bcdb.OpenDB(sdbName+".snapshot", "", "")
	if err != nil {
		panic(err)
	}

	sn := &snapshot{snapshotDB: sndb}
	sn.setMaxSnapshotCount(maxSnapshotCount)

	var lastCommittableTransactionID int64

	// Get last committable transaction ID from db.
	value, err := sdb.Get([]byte(keyOfLastTransactionID()))
	if err != nil {
		panic(err)
	}

	if len(value) != 0 {
		if err := jsoniter.Unmarshal(value, &lastCommittableTransactionID); err != nil {
			panic(err)
		}
	}

	statedb := &StateDB{
		sdb:                          sdb,
		snapshot:                     sn,
		committableTransaction:       nil,
		lastCommittableTransactionID: lastCommittableTransactionID,
		lastRollbackTransactionID:    0,
	}
	sn.stateDB = statedb

	return statedb
}

func (s *StateDB) Get(key string) []byte {
	value, err := s.sdb.Get([]byte(key))
	if err != nil {
		panic(err)
	}
	return value
}

func (s *StateDB) NewCommittableTransaction(maxTxCount int) *Transaction {

	// There can only be one committable transaction at a time.
	if s.committableTransaction != nil {
		panic("must commit last transaction")
	}
	trans := &Transaction{
		transactionID:  s.calcTransactionID(true),
		stateDB:        s,
		maxTxCount:     maxTxCount,
		wBuffer:        new(sync.Map),
		rBuffer:        newKVbuffer(uint(64 * 256)),
		wBitsMerged:    newConflictBits(2000 * 256),
		committable:    true,
		goRoutineCount: (runtime.NumCPU() / 4) * 3,
	}
	s.committableTransaction = trans
	return trans
}

func (s *StateDB) NewRollbackTransaction() *Transaction {

	return &Transaction{
		transactionID: s.calcTransactionID(false),
		stateDB:       s,
		//wBuffer:        make(map[string][]byte),
		wBuffer:        new(sync.Map),
		rBuffer:        newKVbuffer(uint(1 * 256)), // TODO
		committable:    false,
		goRoutineCount: runtime.NumCPU() - 4,
	}
}

func (s *StateDB) Rollback(rollbackTransactions int) {

	if rollbackTransactions <= 0 {
		panic("invalid parameter")
	}

	// There cannot be uncommitted transactions when the database rollback
	if s.committableTransaction != nil {
		panic("cannot be rollback when a committable transaction is not committed")
	}

	s.snapshot.rollback(rollbackTransactions)
}

func (s *StateDB) Close() {

	// There cannot be uncommitted transactions when the database close
	if s.committableTransaction != nil {
		panic("there cannot be uncommitted transactions when the database is closed")
	}

	s.sdb.Close()      // close state db
	s.snapshot.close() // close snapshot db
}

func (s *StateDB) calcTransactionID(committable bool) int64 {
	if committable {
		return atomic.AddInt64(&s.lastCommittableTransactionID, 1)
	} else {
		return atomic.AddInt64(&s.lastRollbackTransactionID, -1)
	}
}

func keyOfLastTransactionID() string {
	return "$last_transaction_id"
}

func getInt64(db *bcdb.GILevelDB, key string) int64 {
	value, err := db.Get([]byte(key))
	if err != nil {
		panic(err)
	}

	if len(value) == 0 {
		return 0
	}

	var result int64
	if err := jsoniter.Unmarshal(value, &result); err != nil {
		panic(err)
	}

	return result
}

func setToBatch(batch *bcdb.GILevelDBBatch, key string, value interface{}) {
	valuerByte, err := jsoniter.Marshal(value)
	if err != nil {
		panic(err)
	}
	batch.Set([]byte(key), valuerByte)
}
