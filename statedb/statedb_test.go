package statedb

import (
	"fmt"
	"github.com/bcbchain/bclib/jsoniter"
	. "gopkg.in/check.v1"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type MySuite struct{}

var _ = Suite(&MySuite{})

func Test(t *testing.T) { TestingT(t) }

func (s *MySuite) TestNewStateDB(c *C) {
	fmt.Println(c.TestName())
	testSameName(c)                     // 打开两个相同名字的数据库
	testMaxSnapshotCount(c)             // 测试最大快照数量
	testLastCommittableTransactionID(c) // 测试 GetStateDB 中 LastCommittableTransactionID 字段
}

func (s *MySuite) TestNewCommittableTransaction(c *C) {
	fmt.Println(c.TestName())
	testNewManyTransaction(c)         // 测试创建多个 Transaction
	testNotCommitAndNewTrnasaction(c) // 先创建 transaction，不 commit，再创建新的 transaction
}

// 测试创建 rollback transaction
func (s *MySuite) TestNewRollbackTransaction(c *C) {
	fmt.Println(c.TestName())

	sdb := New("trt", 100)
	defer sdb.Close()

	for i := -1; i > -20; i-- {
		ts := sdb.NewRollbackTransaction()

		c.Check(sdb.committableTransaction == nil, Equals, true)
		c.Check(ts.transactionID, Equals, sdb.lastRollbackTransactionID)
		c.Check(ts.transactionID, Equals, int64(i))
	}

}

func (s *MySuite) TestTransaction(c *C) {
	fmt.Println(c.TestName())

	// commit transaction
	testCommittableTsGetSet(c)
	testCommittableTsBatchSet(c)
	testCommitLateID(c)

	// rollback transaction
	testRollbackTsGetSet(c)
}

func (s *MySuite) TestTx(c *C) {
	fmt.Println(c.TestName())
	testTxGetSet(c)
}

func (s *MySuite) TestRollback(c *C) {
	fmt.Println(c.TestName())
	testRollbackNormal(c)
	testRollbackPanic(c)        // 测试回滚数量过大
	testRollbackPanicMaxZero(c) // 测试最大快照数为零
}

// 并发创建 rollback transaction
func (s *MySuite) TestConcurrentRollbackTransaction(c *C) {
	fmt.Println(c.TestName())

	sdb := New("crtrans", 100)
	defer sdb.Close()

	resMap := sync.Map{}
	for i := 0; i < 40; i++ {
		go func() {
			trans := sdb.NewRollbackTransaction()
			resMap.Store(trans.ID(), struct{}{})
		}()
	}

	time.Sleep(time.Duration(2) * time.Second)

	counter := 0
	resMap.Range(func(key, value interface{}) bool {
		counter++
		return true
	})
	fmt.Println("counter::", counter)

	c.Check(counter, Equals, 40)
}

func testNewManyTransaction(c *C) {
	sdb := New("tnmt", 100)
	defer sdb.Close()

	var lastID int64
	value := sdb.Get(keyOfLastTransactionID())
	if len(value) > 0 {
		if err := jsoniter.Unmarshal(value, &lastID); err != nil {
			panic(err)
		}
	}

	for i := 1; i < 20; i++ {
		ts := sdb.NewCommittableTransaction()

		c.Check(sdb.committableTransaction == ts, Equals, true)
		c.Check(ts.transactionID, Equals, sdb.lastCommittableTransactionID)
		c.Check(ts.transactionID, Equals, int64(i)+lastID)

		ts.Commit()
	}
}

func testNotCommitAndNewTrnasaction(c *C) {
	sdb := New("tct", 100)

	ts := sdb.NewCommittableTransaction()
	c.Check(ts, Equals, sdb.committableTransaction)

	ts.Commit()

	ts1 := sdb.NewCommittableTransaction()
	c.Check(ts1, Equals, sdb.committableTransaction)

	defer func() {
		err := recover()
		c.Check(err.(string), Equals, "must commit last transaction")
	}()

	// ts1 未 commit，再创建新的 ts
	_ = sdb.NewCommittableTransaction()
}

// 测试重新打开数据库后 StateDB 对象 lastCommittableTransactionID 字段信息
func testLastCommittableTransactionID(c *C) {
	sdb := New("msc", 10)
	for i := 0; i < 10; i++ {
		transaction := sdb.NewCommittableTransaction()
		transaction.Commit()
	}
	sdb.Close()

	sdb = New("msc", 10)

	var lastID int64
	value := sdb.Get(keyOfLastTransactionID())
	if len(value) != 0 {
		if err := jsoniter.Unmarshal(value, &lastID); err != nil {
			panic(err)
		}
	}
	c.Check(sdb.lastCommittableTransactionID, Equals, lastID)
}

func testMaxSnapshotCount(c *C) {

	defer func() {
		err := recover()
		c.Check(err.(string), Equals, "invalid parameter")
	}()

	type Case struct {
		MaxSnapshotCount int
		Desc             string
	}

	cases := []Case{
		{MaxSnapshotCount: 0, Desc: "最大快照数量为零"},
		{MaxSnapshotCount: 1, Desc: "最大快照数量为1"},
		{MaxSnapshotCount: 10, Desc: "最大快照数量为10"},
		{MaxSnapshotCount: 100, Desc: "最大快照数量为100"},
		{MaxSnapshotCount: 10000, Desc: "最大快照数量为10000"},
		{MaxSnapshotCount: -1, Desc: "最大快照数量为-1"},
	}

	for _, v := range cases {
		sdb := New("msc", v.MaxSnapshotCount)
		c.Check(sdb, NotNil)
		sdb.Close()
	}
}

func testSameName(c *C) {
	defer func() {
		err := recover()
		c.Check(err.(error).Error(), Equals, "resource temporarily unavailable")
	}()
	_ = New("same_name", 100)
	_ = New("same_name", 200)
}

func testRollbackTsGetSet(c *C) {
	sdb := New("testbatchset", 100)
	defer sdb.Close()

	ts := sdb.NewRollbackTransaction()

	for i := 0; i < 20; i++ {
		temp := strconv.Itoa(i)
		ts.Set("key11"+temp, []byte("value"+temp))
	}

	for i := 0; i < 20; i++ {
		temp := strconv.Itoa(i)
		value := ts.Get("key11" + temp)

		c.Check(string(value), Equals, "value"+temp)
	}

	ts.Rollback()

	for i := 0; i < 20; i++ {
		temp := strconv.Itoa(i)
		value := ts.Get("key11" + temp)

		c.Check(len(value), Equals, 0)
	}
}

func testCommittableTsBatchSet(c *C) {
	sdb := New("testbatchset", 100)
	defer sdb.Close()

	ts := sdb.NewCommittableTransaction()

	valueMap := make(map[string][]byte, 3)
	valueMap["1"] = []byte("11")
	valueMap["2"] = []byte("22")
	valueMap["3"] = []byte("33")
	ts.BatchSet(valueMap)

	for k, v := range valueMap {
		value := ts.Get(k)
		c.Check(string(value), Equals, string(v))
	}

	ts.Commit()

	for k, v := range valueMap {
		value := ts.Get(k)
		c.Check(string(value), Equals, string(v))
	}
}

func testCommittableTsGetSet(c *C) {
	sdb := New("tctset", 100)
	defer sdb.Close()

	ts := sdb.NewCommittableTransaction()

	for i := 0; i < 20; i++ {
		temp := strconv.Itoa(i)
		ts.Set("key"+temp, []byte("value"+temp))
	}

	for i := 0; i < 20; i++ {
		temp := strconv.Itoa(i)
		value := ts.Get("key" + temp)

		c.Check(string(value), Equals, "value"+temp)
	}
	ts.Commit()

	// commit 之后，依然可以查询到，transaction 缓存查询不到后，从数据库查询。
	for i := 0; i < 20; i++ {
		temp := strconv.Itoa(i)
		value := ts.Get("key" + temp)

		c.Check(string(value), Equals, "value"+temp)
	}
}

func testCommitLateID(c *C) {
	sdb := New("tctset", 100)
	defer func() {
		err := recover()
		c.Check(err.(string), Equals, "must commit last transaction")
	}()
	ts := sdb.NewCommittableTransaction()
	ts1 := sdb.NewCommittableTransaction()

	ts1.Commit()
	ts.Commit()
}

func testTxGetSet(c *C) {
	sdb := New("ttxset", 100)
	defer sdb.Close()

	ts := sdb.NewCommittableTransaction()
	tx := ts.NewTx(nil, nil)

	for i := 0; i < 20; i++ {
		temp := strconv.Itoa(i)
		tx.Set("key"+temp, []byte("value"+temp))
	}

	for i := 0; i < 20; i++ {
		temp := strconv.Itoa(i)
		value := tx.Get("key" + temp)

		c.Check(string(value), Equals, "value"+temp)
	}
	tx.Commit()

	for i := 0; i < 20; i++ {
		temp := strconv.Itoa(i)
		value := tx.Get("key" + temp)

		c.Check(len(value), Equals, 0)
	}

	for i := 0; i < 20; i++ {
		temp := strconv.Itoa(i)
		value := ts.Get("key" + temp)

		c.Check(string(value), Equals, "value"+temp)
	}

	ts.Commit()
}

func testRollbackPanicMaxZero(c *C) {
	sdb := New("testrollbackpaniczero", 0)

	defer func() {
		err := recover()
		flag := strings.HasPrefix(err.(string), "no this transactionID snapshot")
		c.Check(flag, Equals, true)
	}()

	for i := 0; i < 20; i++ {
		temp := strconv.Itoa(i)
		ts := sdb.NewCommittableTransaction()
		tx := ts.NewTx(nil, false, nil)

		tx.Set("key"+temp, []byte("value"+temp))
		tx.Commit()
		ts.Commit()
	}

	sdb.Rollback(15)
}

func testRollbackPanic(c *C) {
	sdb := New("testrollbackpanic", 10)

	defer func() {
		err := recover()
		flag := strings.HasPrefix(err.(string), "no this transactionID snapshot")
		c.Check(flag, Equals, true)
	}()

	for i := 0; i < 20; i++ {
		temp := strconv.Itoa(i)
		ts := sdb.NewCommittableTransaction()
		tx := ts.NewTx(nil, false, nil)

		tx.Set("key"+temp, []byte("value"+temp))
		tx.Commit()
		ts.Commit()
	}

	sdb.Rollback(15)
}

func testRollbackNormal(c *C) {
	sdb := New("testrollback", 10)
	defer sdb.Close()

	for i := 0; i < 20; i++ {
		temp := strconv.Itoa(i)
		ts := sdb.NewCommittableTransaction()
		tx := ts.NewTx(nil, false, nil)

		tx.Set("key"+temp, []byte("value"+temp))
		tx.Commit()
		ts.Commit()
	}

	var (
		lastID1, lastID2 int64
	)

	lastIDValue := sdb.Get(keyOfLastTransactionID())
	_ = jsoniter.Unmarshal(lastIDValue, &lastID1)

	sdb.Rollback(3)
	value := sdb.Get("key16")
	c.Check(string(value), Equals, "value16")

	lastIDValue = sdb.Get(keyOfLastTransactionID())
	_ = jsoniter.Unmarshal(lastIDValue, &lastID2)

	c.Check(lastID1, Equals, lastID2+3)
	value = sdb.Get("key17")
	c.Check(string(value), Equals, "")
}

func TestA(t *testing.T) {
	var a map[string]string
	_, ok := a["a"]
	fmt.Println(ok)
}
