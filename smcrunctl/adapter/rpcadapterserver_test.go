package adapter

import (
	"fmt"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	. "gopkg.in/check.v1"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type MySuite struct{}

var _ = Suite(&MySuite{})

func Test(t *testing.T) { TestingT(t) }

func (s *MySuite) TestRpcStart(c *C) {
	home := os.Getenv("HOME")
	logger := log.NewTMLogger(filepath.Join(home, "log"), "smcsvc")
	go start(33881, logger)

	time.Sleep(time.Duration(3) * time.Second)
	err := httpSet()
	if err != nil {
		fmt.Println("Get ERROR:", err)
		return
	}
	c.Check(err, Equals, nil)

	err = httpGet()
	if err != nil {
		fmt.Println("Get ERROR:", err)
		return
	}
	c.Check(err, Equals, nil)
}

func httpSet() error {
	resp, err := http.Get("http://127.0.0.1:33881/set?transId=1&txId=1&data={\"test\":[97 97]}")
	if err != nil {
		fmt.Println("Set ERROR:", err)
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Get ERROR:", err)
		return err
	}
	fmt.Println("Set BODY:", string(body))
	return nil
}

func httpGet() error {
	resp, err := http.Get("http://127.0.0.1:33881/get?transId=1&txId=1&data=\"test\"")
	if err != nil {
		fmt.Println("Get ERROR:", err)
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Get ERROR:", err)
		return err
	}
	fmt.Println("Get BODY:", string(body))
	return nil
}
