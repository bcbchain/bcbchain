package main

import (
	"github.com/bcbchain/bclib/wal"
	"fmt"
)

func main() {
	acc, err := wal.NewAccount("wal", "owner", "Ab1@Cd3$")
	if err != nil {
		panic(err)
	}
	fmt.Println(acc.Address("local"))
}
