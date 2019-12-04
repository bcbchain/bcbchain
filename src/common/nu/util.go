package nu // net utility
import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// GetIdlePort 获取一个可用的空闲端口
func GetIdlePort() uint16 {
	for {
		addr := "127.0.0.1:"
		listener, err := net.Listen("tcp4", addr)
		if err != nil {
			fmt.Printf("%v is in use\n", addr)
			continue
		}
		port, _ := strconv.Atoi(strings.TrimPrefix(listener.Addr().String(), addr))
		listener.Close()

		return uint16(port)
	}
}
