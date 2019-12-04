package mc

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"testing"
)

type testData struct {
	Code uint32
	Data string
	Log  string
	Info string
}

func TestMemCache_Set(t *testing.T) {
	var transid int64 = 1000
	key := "/test_/1000"
	var txid int64 = 1

	mi := NewMcInstance()
	mc := testNewMc(t, mi, transid, key)

	data := testData{200, "testSet", "OK", "Hello Cache"}
	testMcSet(t, mc, txid, data)
}

func testNewMc(t *testing.T, mi *Instance, transid int64, key string) *MemCache {
	mc := mi.NewMc(transid, key)
	t.Log("Starting testNewMc()")
	t.Log("    transId = ", transid, ", key = ", key)
	if _, ok := mi.mcCache[transid][key]; ok {
		t.Log("    MemCache Map: ", mi.mcCache)
		t.Log("    MemCache was created successfully")
	} else {
		t.Error("    MemCache was created failed")
	}
	return mc
}

func testMcSet(t *testing.T, mc *MemCache, txid int64, data interface{}) {
	t.Log("Starting testMcSet()")
	last := mc.lastTx()
	mc.Set(txid, data)
	t.Log("    txId = ", txid, ", data = ", data)
	if mc.lastTx() == last+1 {
		t.Log("    MemCache: ", mc)
		t.Log("    txCache was set successfully")
	} else {
		t.Error("    txCache was set failed")
	}
}

func testCommitTrans(t *testing.T, mi *Instance, transid int64) {
	t.Log("Starting testCommitTrans()")
	mcs := mi.mcCache[transid]
	keys := make([]string, 0)
	values := make([]interface{}, 0)
	for _, mc := range mcs {
		keys = append(keys, mc.key)
		values = append(values, mc.Get())
		t.Log("    Key:", mc.key, ", Value:", mc.Get())
	}
	mi.CommitTrans(transid)
	for i, key := range keys {
		if mi.llCache[key] != values[i] {
			t.Error("    Commit trans cache failed")
		}
	}
	t.Log("    Commit trans cache successfully")
}

func TestMcInstance_DirtyTransTx(t *testing.T) {
	goid := int64(getGoID())

	mi := NewMcInstance()
	transid := goid
	key := "/test_/" + strconv.FormatInt(transid, 10)

	mc := mi.NewMc(transid, key)
	txid := int64(1)
	for j := 1; j <= 4; j++ {
		txid = txid * int64(j)
		data := testData{uint32(200 * j), "testSet", "OK", "Hello Cache"}
		mc.Set(txid, data)
	}
	fmt.Println(mc)
	hasTx := false
	if tmc := mi.mcCache[transid][key]; tmc != nil {
		for _, t := range tmc.tx {
			if t.id == txid {
				hasTx = true
				break
			}
		}
	}
	if !hasTx {
		t.Error("None test case created")
	} else {
		hasTx = false
		mi.DirtyTransTx(transid, txid)
		fmt.Println(mc)

		for _, t := range mc.tx {
			if t.id == txid {
				hasTx = true
				break
			}
		}
		if hasTx {
			t.Error("DirtyTransTx() failed")
		}
	}

}
func TestActions(t *testing.T) {
	goid := int64(getGoID())

	mi := NewMcInstance()
	//Init test case
	num := 10
	for i := 1; i <= num; i++ {
		transid := goid * int64(i)
		key := "/test_/" + strconv.FormatInt(transid, 10)
		mc := testNewMc(t, mi, transid, key)

		for j := 1; j <= 4; j++ {
			txid := int64(1 * j)
			data := testData{uint32(200 * j), "testSet", "OK", "Hello Cache"}
			testMcSet(t, mc, txid, data)
		}
		testCommitTrans(t, mi, transid)
	}

	key := "/test_/" + strconv.FormatInt(goid, 10)
	transid := goid * 100
	mi.NewMc(transid, key)
	mi.Dirty("/test_/" + strconv.FormatInt(goid, 10))
	if _, ok := mi.llCache[key]; ok {
		t.Error("    Dirty(key) failed")
	}
	for _, mcs := range mi.mcCache {
		for _, mc := range mcs {
			if mc.key == key {
				t.Error("    Dirty(key) failed")
				break
			}
		}
	}
	//waitGroup.Done()
}

var waitGroup sync.WaitGroup

func TestGoRoutine(t *testing.T) {
	//waitGroup.Add(1)
	//go TestMcInstance2(t)

	//	waitGroup.Wait()
}

func getGoID() int64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseInt(string(b), 10, 63)
	return n
}

type testData1 struct {
	id    int64
	value string
	d     [10240000]int64
}

func TestMemoryDirtyTx(t *testing.T) {
	mi := NewMcInstance()

	mc2 := mi.NewMc(1, "test1")
	mc3 := mi.NewMc(2, "test2")
	i := int64(0)
	for {
		mc2.Set(1, testData1{id: i, value: "hello world"})
		mc2.Dirty(1)
		mc3.Set(1, testData1{id: i, value: "hello world"})
		mc3.Dirty(1)
		i++
		fmt.Println(mc2, mc3)
	}
}

func TestMemory_DirtyTrans(t *testing.T) {
	mi := NewMcInstance()
	i := int64(0)
	for {
		mc2 := mi.NewMc(i, "test1")
		mc2.Set(1, testData1{id: 333, value: "hello world"})
		mi.DirtyTrans(i)
		mc3 := mi.NewMc(100+i, "test2")
		mc3.Set(1, testData1{id: 444, value: "hello world"})
		mi.DirtyTrans(100 + i)
		i++
		fmt.Println(mi)
	}
}
