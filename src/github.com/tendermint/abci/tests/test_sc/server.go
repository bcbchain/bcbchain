/*bcchain v2.0重大问题和修订方案1.1解决方案3 服务端*/
package main

import (
	svr "github.com/tendermint/abci/server"
	"github.com/tendermint/abci/types"
	tmlog "github.com/tendermint/tmlibs/log"
	"os"
	"path/filepath"
)

var (
	logger tmlog.Loggerf
)

const (
	LINUX_S  ="tcp://192.168.80.150:8080"
	WINDOWS_S = "tcp://192.168.1.177:8080"
)

func main()  {
	var app    types.Application
	a:=make(chan bool)
	home := os.Getenv("HOME")
	logger = tmlog.NewTMLogger(filepath.Join(home, "log"), "bclib")
	logger.AllowLevel("debug")
	logger.SetOutputAsync(true)
	logger.SetOutputToFile(false)
	logger.SetOutputToScreen(true)
	logger.SetOutputFileSize(20000000)

	s:=svr.NewSocketServer(LINUX_S,app)
	s.SetLogger(logger)
	logger.Debug("start server")

	if err := s.OnStart(); err != nil {
		return
	}

	<-a
}
