package main

import (
	"common/bcdb"
	"fmt"
)

func main() {
	db, err := bcdb.OpenDB("testdb", "127.0.0.1", "8888")
	if err != nil {
		fmt.Println(err)
	}

	key1 := []byte{0x01}
	value1 := []byte{0x11}

	db.Set(key1, value1)
	valBytes, _ := db.Get(key1)
	fmt.Println(key1, ":", valBytes)

	batch := db.NewBatch()
	key2 := []byte{0x02}
	value2 := []byte{0x22}
	batch.Set(key2, value2)
	batch.Delete(key1)
	db.Print()
	batch.Commit()
	db.Print()

	db.Delete(key2)
	db.Close()
}
