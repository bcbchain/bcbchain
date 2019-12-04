package controllermgr

import (
	"blockchain/common/statedbhelper"
	"blockchain/smcbuilder"
	"blockchain/smcdocker"
	"blockchain/smcrunctl/invokermgr"
	"blockchain/types"
	"common/dockerlib"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/tendermint/tmlibs/log"
)

type ControllerMgr struct {
	logger    log.Logger
	rpcurl    string
	stTransId int
	stTxId    int
}

var (
	mgr    *ControllerMgr
	once   sync.Once
	health types.Health
)

func GetInstance() *ControllerMgr {
	once.Do(func() {
		mgr = &ControllerMgr{}
	})
	return mgr
}

func (ctl *ControllerMgr) Init(log log.Logger, rpcPort int) {
	ctl.logger = log
	//ctl.rpcurl = "tcp://" + common.GetDockerGateway() + ":" + strconv.Itoa(rpcPort)
	d := dockerlib.GetDockerLib()

	d.Init(log)
	chainID := statedbhelper.GetChainID()
	if chainID != "" {
		d.Reset(chainID + ".")
		d.SetPrefix(chainID + ".")
	}
	url := d.GetMyIntranetIP()
	ctl.rpcurl = "tcp://" + url + ":" + strconv.Itoa(rpcPort)

	im := invokermgr.GetInstance()
	im.Init(log)
	smcdocker.GetInstance().Init(log, ctl.rpcurl, im.DirtyURL)

	if runtime.GOOS == "windows" {
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		dir := filepath.Dir(ex)
		if dir == "" {
			panic(errors.New("failed to get path of forks file"))
		}
		smcbuilder.Init(log, dir+"\\.build")
	} else {
		smcbuilder.Init(log, os.Getenv("HOME")+"/.build")
	}

	go moniter()
}

func (ctl *ControllerMgr) Health() *types.Health {
	return &health
}

const (
	LoopStamp     = 10 //调用间隔 10 seconds
	keyDocker     = "docker"
	keyBuilder    = "builder"
	keyInvoker    = "invoker"
	keyController = "controller"
)

func moniter() {
	for {
		health.Tm = time.Now()
		//dkh := dockermgr.GetInstance().Health()
		//health = dkh
		//
		//loop for each 10 seconds
		time.Sleep(time.Second * LoopStamp)
	}
}

// checkHealth() 检查内存中记录的各个模块的健康状态，如果模块异常，调用模块的Init和Startup
func (ctl *ControllerMgr) checkHealth() {

	for k, v := range health.SubHealth {
		if health.Tm.Sub(v.Tm) > LoopStamp*3 {
			switch k {
			case keyBuilder:
				//buildermgr.GetInstance().Init(ctl.log, ctl.sdburl)

			case keyDocker:
				//dockermgr.GetInstance().Init(ctl.log, ctl.sdburl)

			case keyInvoker:
				//invokermgr.GetInstance().Init(ctl.log, ctl.sdburl)

			}
		}
	}
}
