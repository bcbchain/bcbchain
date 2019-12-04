package main

import (
	"common/bcdb"
	"fmt"
	"testing"
)

func TestGILevelDB(t *testing.T) {

	//todo 保证多连接数据库
	db, err := bcdb.OpenDB("testdb", "127.0.0.1", "8888")
	if err != nil {
		fmt.Println(err)
	}

	key1 := []byte{0x01}
	//value1 := []byte{0x11}

	db.Set([]byte("/genesis/token"), []byte("/genesis/token"))
	fmt.Println(key1, ":", db.Get([]byte("/genesis/token")))

	//db2, err := gidb.OpenDB("testdb","127.0.0.1", "8888")
	//if err != nil{
	//	fmt.Println(err)
	//}
	//
	//key2 := []byte{0x02}
	//value2 := []byte{0x22}
	//
	//db2.Set(key2, value2)
	//fmt.Println(key1, ":", db.Get(key2))

}
