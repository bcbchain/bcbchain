package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
)

func main() {
	go func() {
		if e := http.ListenAndServe(":2019", nil); e != nil {
			fmt.Println("pprof cannot start!!!")
		}
	}()

	err := Execute()
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}
