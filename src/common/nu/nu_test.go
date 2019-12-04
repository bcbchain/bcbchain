package nu // net utility

import (
	"fmt"
	"testing"
)

func Test(t *testing.T) {
	for i := 0; i < 100; i++ {
		p := GetIdlePort()
		fmt.Println("port:", p)
	}
}
