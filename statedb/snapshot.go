package statedb

import (
	"bytes"
	"github.com/bcbchain/bclib/bcdb"
	"github.com/bcbchain/bclib/jsoniter"
	"fmt"
)

type snapshot struct {
	stateDB    *StateDB
	snapshotDB *bcdb.GILevelDB
}

func (s *snapshot) rollback(rollbackTransactions int) {

	lastTransactionID := getInt64(s.snapshotDB, keyOfLastTransactionID())
	sdbLastTransactionID := getInt64(s.stateDB.sdb, keyOfLastTransactionID())

	if lastTransactionID != sdbLastTransactionID {
		panic(fmt.Sprintf("snapshot last transactionID=%d doesn't match state db last transactionID=%d.",
			lastTransactionID, sdbLastTransactionID))
	}

	if lastTransactionID-int64(rollbackTransactions) < 0 {
		panic(fmt.Sprintf("can not rollback, lastTransactionID=%d, rollbackTransactions=%d",
			lastTransactionID, rollbackTransactions))
	}

	targetID := lastTransactionID - int64(rollbackTransactions) + 1
	originData := s.getOriginData(targetID)
	if len(originData) == 0 {
		panic(fmt.Sprintf("no this transactionID snapshot, transactionID=%d", targetID))
	}

	batch := s.snapshotDB.NewBatch()
	for id := lastTransactionID; id >= targetID; id-- {
		s.rollbackStateDB(id)

		batch.Delete([]byte(keyOfNewData(id)))
		batch.Delete([]byte(keyOfOriginData(id)))
	}

	// update snapshot db last transaction ID
	setToBatch(batch, keyOfLastTransactionID(), targetID-1)

	if err := batch.Commit(); err != nil {
		panic(err)
	}
}

func (s *snapshot) rollbackStateDB(transactionID int64) {
	newData := make(map[string][]byte)
	s.snapshotDBGet(keyOfNewData(transactionID), &newData)
	if len(newData) == 0 {
		panic("invalid transactionID")
	}

	for k, v := range newData {
		sdbValue, err := s.stateDB.sdb.Get([]byte(k))
		if err != nil {
			panic(err)
		}

		if bytes.Compare(v, sdbValue) != 0 {
			panic("can not rollback, because snapshot data wrong.")
		}
	}

	originData := make(map[string][]byte)
	s.snapshotDBGet(keyOfOriginData(transactionID), &originData)
	if len(originData) == 0 {
		panic("can not rollback, because snapshot data wrong.")
	}

	sBatch := s.stateDB.sdb.NewBatch()
	for k, v := range originData {
		if len(v) == 0 {
			sBatch.Delete([]byte(k))
		} else {
			sBatch.Set([]byte(k), v)
		}
	}

	setToBatch(sBatch, keyOfLastTransactionID(), transactionID-1)

	if err := sBatch.Commit(); err != nil {
		panic(err)
	}
}

func (s *snapshot) commit(transactionID int64, originData, newData map[string][]byte) {

	var maxCount int

	if value, err := s.snapshotDB.Get([]byte(keyOfMaxSnapshotCount())); err != nil {
		panic(err)
	} else if len(value) == 0 {
		panic("must set max snapshot count")
	} else {
		if err := jsoniter.Unmarshal(value, &maxCount); err != nil {
			panic(err)
		}
	}

	lastTransactionID := getInt64(s.snapshotDB, keyOfLastTransactionID())

	// lastTransactionID == 0 means .snapshot.db deleted
	if lastTransactionID != 0 && transactionID != lastTransactionID+1 && transactionID != lastTransactionID {
		panic(fmt.Sprintf("invalid transactionID: %d, lastTransactionID: %d", transactionID, lastTransactionID))
	}

	batch := s.snapshotDB.NewBatch()
	s.checkMaxCount(transactionID, maxCount, batch)

	if maxCount != 0 {
		// set origin data
		setToBatch(batch, keyOfOriginData(transactionID), originData)

		// set new data
		setToBatch(batch, keyOfNewData(transactionID), newData)
	}

	// set last transaction ID
	setToBatch(batch, keyOfLastTransactionID(), transactionID)

	// commit to snapshot db
	if err := batch.Commit(); err != nil {
		panic(err)
	}
}

func (s *snapshot) checkMaxCount(transactionID int64, maxCount int, batch *bcdb.GILevelDBBatch) {

	minID := transactionID - int64(maxCount)
	if minID <= 0 {
		return
	}

	for id := minID; id > 0; id-- {

		originKey := []byte(keyOfOriginData(id))
		if !s.snapshotDB.Has(originKey) {
			return
		}

		batch.Delete(originKey)
		batch.Delete([]byte(keyOfNewData(id)))
	}
}

func (s *snapshot) getOriginData(transactionID int64) map[string][]byte {
	key := keyOfOriginData(transactionID)
	data := make(map[string][]byte)
	s.snapshotDBGet(key, &data)
	return data
}

func (s *snapshot) getNewData(transactionID int64) map[string][]byte {
	key := keyOfNewData(transactionID)
	data := make(map[string][]byte)
	s.snapshotDBGet(key, &data)
	return data
}

func (s *snapshot) setMaxSnapshotCount(count int) {
	value, err := jsoniter.Marshal(count)
	if err != nil {
		panic(err)
	}

	if err := s.snapshotDB.SetSync([]byte(keyOfMaxSnapshotCount()), value); err != nil {
		panic(err)
	}
}

func (s *snapshot) close() {
	s.snapshotDB.Close()
}

func (s *snapshot) stateDBGet(key string, obj interface{}) {
	value, err := s.stateDB.sdb.Get([]byte(key))
	if err != nil {
		panic(err)
	}

	if len(value) == 0 {
		return
	}

	if err := jsoniter.Unmarshal(value, obj); err != nil {
		panic(err)
	}
}

func (s *snapshot) snapshotDBGet(key string, obj interface{}) {
	value, err := s.snapshotDB.Get([]byte(key))
	if err != nil {
		panic(err)
	}

	if len(value) == 0 {
		return
	}

	if err := jsoniter.Unmarshal(value, obj); err != nil {
		panic(err)
	}
}

func keyOfMaxSnapshotCount() string {
	return "$max_snapshot_count"
}

func keyOfOriginData(transactionID int64) string {
	return fmt.Sprintf("$%d$origin_data", transactionID)
}

func keyOfNewData(transactionID int64) string {
	return fmt.Sprintf("$%d$new_data", transactionID)
}
