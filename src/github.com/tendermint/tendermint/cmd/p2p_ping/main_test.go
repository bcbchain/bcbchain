package main

import (
	"fmt"
	"testing"
)

func TestRegSplit(t *testing.T) {
	text := "192.168.1.1:222,192.111.1.1:2323;19.1.2.3:333 111"
	lst := RegSplit(text, "[,;]")
	for i, v := range lst {
		fmt.Println(i, v)
	}
}
