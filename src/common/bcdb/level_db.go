package bcdb

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/tendermint/tmlibs/log"
)

type GILevelDB struct {
	db *leveldb.DB
}

// We defensively turn nil keys or values into []byte{} for
// most operations.
func nonNilBytes(bz []byte) []byte {
	if bz == nil {
		return []byte{}
	} else {
		return bz
	}
}

//第一版只实现本地数据库，当前目录下,如果数据库文件存在，直接打开，如果不存在，根据name创建数据库文件夹
func OpenDB(name string, ip string, port string) (*GILevelDB, error) {
	var dbPath string
	if strings.HasPrefix(name, "/") {
		dbPath = name + ".db"
	} else {
		home := os.Getenv("HOME")
		dbPath = filepath.Join(home, name+".db")
	}
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, err
	}
	database := &GILevelDB{
		db: db,
	}
	return database, nil
}

func (db *GILevelDB) Get(key []byte) ([]byte, error) {
	key = nonNilBytes(key)
	res, err := db.db.Get(key, nil)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return res, nil
}

func (db *GILevelDB) Has(key []byte) bool {
	v, _ := db.Get(key)
	return v != nil
}

func (db *GILevelDB) Set(key []byte, value []byte) error {
	key = nonNilBytes(key)
	value = nonNilBytes(value)
	return db.db.Put(key, value, nil)
}

func (db *GILevelDB) SetSync(key []byte, value []byte) error {
	key = nonNilBytes(key)
	value = nonNilBytes(value)
	return db.db.Put(key, value, &opt.WriteOptions{Sync: true})
}

func (db *GILevelDB) Delete(key []byte) error {
	key = nonNilBytes(key)
	return db.db.Delete(key, nil)
}

func (db *GILevelDB) DeleteSync(key []byte) error {
	key = nonNilBytes(key)
	return db.db.Delete(key, &opt.WriteOptions{Sync: true})
}

func (db *GILevelDB) Close() {
	db.db.Close()
}

func (db *GILevelDB) Print() {
	str, _ := db.db.GetProperty("leveldb.stats")
	fmt.Printf("%v\n", str)

	iter := db.db.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		fmt.Printf("%s:%s\n", string(key), string(value))
	}
}

//遍历db所有key
func (db *GILevelDB) GetAllKey() []byte {

	var data []string

	iter := db.db.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		data = append(data, string(key))
	}
	if len(data) == 0 {
		return nil
	}
	keysBytes := strings.Join(data, ";")

	return []byte(keysBytes)
}

func (db *GILevelDB) queryDB(w http.ResponseWriter, req *http.Request) {
	value, err := db.Get([]byte(req.RequestURI))
	if err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(value))
	}
}

func (db *GILevelDB) StartQueryDBServer(queryAddress string, logger log.Logger) {
	http.HandleFunc("/", db.queryDB)
	logger.Info("StartQueryDBServer", "address:", queryAddress)
	err := http.ListenAndServe(queryAddress, nil)
	if err != nil {
		logger.Error("ListenAndServe: ", "error", err)
	}
}

//----------------------------------------
// Batch

type GILevelDBBatch struct {
	db    *GILevelDB
	batch *leveldb.Batch
}

// Implements DB.
func (db *GILevelDB) NewBatch() *GILevelDBBatch {
	batch := new(leveldb.Batch)
	return &GILevelDBBatch{db, batch}
}

// Implements Batch.
func (mBatch *GILevelDBBatch) Set(key, value []byte) {
	mBatch.batch.Put(key, value)
}

// Implements Batch.
func (mBatch *GILevelDBBatch) Delete(key []byte) {
	mBatch.batch.Delete(key)
}

// Implements Batch.
func (mBatch *GILevelDBBatch) Commit() error {
	return mBatch.db.db.Write(mBatch.batch, nil)
}
