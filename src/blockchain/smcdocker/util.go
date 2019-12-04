package smcdocker

import (
	"blockchain/abciapp/common"
	"blockchain/abciapp/softforks"
	"common/jsoniter"
	"common/socket"
	"github.com/tendermint/tmlibs/log"
	"time"
)

func waitSmcRunSvcReady(url string, logger log.Logger) bool {
	logger.Trace("WaitSmcSvcReady()", "url", url)
	beginTime := time.Now()
	timeOut := 10.00
	for {
		time.Sleep(time.Duration(time.Millisecond * 500))
		n := time.Now()
		sub := n.Sub(beginTime).Seconds()
		if sub > timeOut {
			logger.Error("WaitSmcSvcReady() timed out")
			return false
		}
		cli, err := socket.NewClient(url, true, logger)
		if err != nil {
			continue
		}
		value, err := cli.Call("Health", map[string]interface{}{"transID": 0}, 10)
		if err != nil || value == nil {
			continue
		}
		if value.(string) == "health" {
			logger.Debug("WaitSmcSvcReady()", "url", url, "checkHealth", value.(string))

			SetDockerLogLevel(url, common.GlobalConfig.LogLevel, logger)
			InitDockerSoftForks(url, logger)
			return true
		} else {
			panic("connect to smcrunsvc failed")
		}
	}
}

func SetDockerLogLevel(url, level string, logger log.Logger) {
	cli, err := socket.NewClient(url, true, logger)
	if err != nil {
		panic(err)
	}
	value, err := cli.Call("SetLogLevel", map[string]interface{}{"level": level}, 10)
	if err != nil {
		panic(err)
	}
	if !value.(bool) {
		panic("can not set docker log level")
	}
}

func InitDockerSoftForks(url string, logger log.Logger) {
	forksBytes, err := jsoniter.Marshal(softforks.TagToForkInfo)
	if err != nil {
		panic(err)
	}

	cli, err := socket.NewClient(url, true, logger)
	if err != nil {
		panic(err)
	}
	value, err := cli.Call("InitSoftForks", map[string]interface{}{"softforks": string(forksBytes)}, 10)
	if err != nil {
		panic(err)
	}
	if !value.(bool) {
		panic("can not init docker soft forks")
	}
}
