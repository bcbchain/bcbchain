package statedb_test

import (
	"blockchain/statedb"
	"fmt"
	. "gopkg.in/check.v1"
	"strconv"
	"testing"
	"time"
)

type MySuite struct{}

var _ = Suite(&MySuite{})

func Test(t *testing.T) { TestingT(t) }

func (s *MySuite) TestGetForNotCommitAndRollback(c *C) {
	statedb.Init("test", "test", "test")
	transID := statedb.NewTransaction()

	randStr := strconv.Itoa(time.Now().Nanosecond())
	tempStr := "test" + randStr

	fmt.Println("transID: ", transID)

	statedb.Set(transID, 1, "test1", []byte(tempStr))

	c.Check(string(statedb.Get(0, 1, "test1")), Equals, "")

	c.Check(string(statedb.Get(transID, 1, "test1")), Equals, tempStr)
}

func (s *MySuite) TestGetForCommitTx(c *C) {
	statedb.Init("test", "test", "test")
	transID := statedb.NewTransaction()

	randStr := strconv.Itoa(time.Now().Nanosecond())

	valueMap := make(map[string][]byte)

	for i := 0; i < 100; i++ {
		valueMap[strconv.Itoa(i)+randStr] = []byte("test" + strconv.Itoa(i))
	}

	statedb.BatchSet(transID, 1, valueMap)

	commitRes, _ := statedb.CommitTx(transID, 1)
	fmt.Println("commitRes:", string(commitRes))
	fmt.Println([]byte("aaa"))

	for i := 0; i < 100; i++ {
		c.Check(string(statedb.Get(transID, 1, strconv.Itoa(i)+randStr)), Equals, "test"+strconv.Itoa(i))
		c.Check(string(statedb.Get(0, 1, strconv.Itoa(i)+randStr)), Equals, "")
	}
}

func (s *MySuite) TestGetForCommit(c *C) {
	statedb.Init("test", "test", "test")
	transID := statedb.NewTransaction()

	randStr := strconv.Itoa(time.Now().Nanosecond())

	defer func() {
		r := recover()
		c.Check(true, Equals, r != nil)
		c.Check(r.(string), Equals, "Invalid transID.")
	}()

	statedb.Set(transID, 1, randStr, []byte(randStr))

	statedb.CommitTx(transID, 1)
	c.Check(string(statedb.Get(0, 1, randStr)), Equals, "")

	statedb.Commit(transID)
	c.Check(string(statedb.Get(0, 1, randStr)), Equals, randStr)

	statedb.Get(transID, 1, randStr)
}

func (s *MySuite) TestGetForRollbackTx(c *C) {
	statedb.Init("test", "test", "test")
	transID := statedb.NewTransaction()

	randStr := strconv.Itoa(time.Now().Nanosecond())

	valueMap := make(map[string][]byte)

	for i := 0; i < 100; i++ {
		valueMap[strconv.Itoa(i)+randStr] = []byte("test" + strconv.Itoa(i))
	}

	statedb.BatchSet(transID, 1, valueMap)
	statedb.Set(transID, 2, randStr, []byte(randStr))

	statedb.RollbackTx(transID, 1)

	c.Check(string(statedb.Get(0, 1, randStr)), Equals, "")
	c.Check(string(statedb.Get(transID, 2, randStr)), Equals, randStr)

	for i := 0; i < 100; i++ {
		c.Check(string(statedb.Get(transID, 1, strconv.Itoa(i)+randStr)), Equals, "")
	}
}

func (s *MySuite) TestGetForRollback(c *C) {
	statedb.Init("test", "test", "test")
	transID := statedb.NewTransaction()

	randStr := strconv.Itoa(time.Now().Nanosecond())

	defer func() {
		r := recover()
		c.Check(true, Equals, r != nil)
		c.Check(r.(string), Equals, "Invalid transID.")
	}()

	statedb.Set(transID, 1, randStr, []byte(randStr))
	c.Check(string(statedb.Get(0, 1, randStr)), Equals, "")
	c.Check(string(statedb.Get(transID, 1, randStr)), Equals, randStr)

	statedb.Rollback(transID)
	c.Check(string(statedb.Get(0, 1, randStr)), Equals, "")

	statedb.Get(transID, 1, randStr)
}

func (s *MySuite) TestGetForTransZero(c *C) {
	statedb.Init("test", "test", "test")
	transID := statedb.NewTransaction()

	randStr := strconv.Itoa(time.Now().Nanosecond())

	statedb.Set(transID, 1, randStr, []byte(randStr))

	statedb.CommitTx(transID, 1)
	c.Check(string(statedb.Get(transID, 1, randStr)), Equals, randStr)

	statedb.Commit(transID)
	c.Check(string(statedb.Get(0, 1, randStr)), Equals, randStr)
}

func (s *MySuite) TestBatchSetAndCommit(c *C) {
	statedb.Init("test", "test", "test")
	transID := statedb.NewTransaction()

	randStr := strconv.Itoa(time.Now().Nanosecond())
	tempStr := "test" + randStr

	value := []byte(tempStr)

	valueMap := make(map[string][]byte)
	valueMap["test4"] = value

	for i := 0; i < 100; i++ {
		value := []byte(strconv.Itoa(i))
		valueMap[strconv.Itoa(i)] = value
	}

	statedb.BatchSet(transID, 1, valueMap)

	statedb.CommitTx(transID, 1)

	statedb.Commit(transID)

	for i := 0; i < 100; i++ {
		value := []byte(strconv.Itoa(i))
		c.Check(string(statedb.Get(0, 1, strconv.Itoa(i))), Equals, string(value))
	}
}

func (s *MySuite) TestCommitManyTime(c *C) {
	statedb.Init("test", "test", "test")
	transID := statedb.NewTransaction()

	randStr := strconv.Itoa(time.Now().Nanosecond())
	tempStr := "test" + randStr

	defer func() {
		r := recover()
		c.Check(true, Equals, r != nil)
	}()

	var value []byte
	value = make([]byte, 1)
	value[0] = 'b'
	fmt.Println("transID: ", transID)
	fmt.Println("value: ", value)

	for i := 1; i < 20; i++ {
		statedb.Set(transID, int64(i), tempStr+strconv.Itoa(i), []byte(tempStr+strconv.Itoa(i)))
	}

	for i := 1; i < 20; i++ {
		statedb.CommitTx(transID, int64(i))
		c.Check(string(statedb.Get(0, int64(i), tempStr+strconv.Itoa(i))), Equals, "")
		c.Check(string(statedb.Get(transID, int64(i), tempStr+strconv.Itoa(i))), Equals, tempStr+strconv.Itoa(i))
	}

	statedb.Commit(transID)

	statedb.Get(transID, 1, tempStr)
}

func (s *MySuite) TestConcurrent(c *C) {
	var resSlice []int64
	for i := 0; i < 40; i++ {
		go func() {
			transID := statedb.NewTransaction()
			fmt.Println(transID)
			resSlice = append(resSlice, transID)
		}()
	}

	time.Sleep(time.Duration(2) * time.Second)
	fmt.Println(resSlice)

	isOk := true
	for i := 0; i < len(resSlice); i++ {
		for j := i + 1; j < len(resSlice)-1; j++ {
			if resSlice[i] == resSlice[j] {
				isOk = false
			}
		}
	}

	c.Check(true, Equals, isOk)
}
