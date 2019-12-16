package core

import (
	"common/jsoniter"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"

	core_types "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/coreos/etcd/pkg/fileutil"
	"github.com/tendermint/tmlibs/common"

	"github.com/tendermint/tendermint/config"
)

var cfg *config.Config = nil
var lock sync.Mutex

func parseConfig() { // 相当不 DRY， 避免循环引用又撸一遍，有空把这个逻辑整合
	if cfg == nil {
		cfg = config.DefaultConfig()
		tmPath := os.Getenv("TMHOME")
		if tmPath == "" {
			home := os.Getenv("HOME")
			if home != "" {
				tmPath = filepath.Join(home, config.DefaultTendermintDir)
			}
		}
		if tmPath == "" {
			tmPath = "/" + config.DefaultTendermintDir
		}
		cfg.SetRoot(tmPath)
	}
}

func GetGenesisPkg() (*core_types.ResultConfFile, error) {
	if completeStarted == false {
		return nil, errors.New("wait application complete started")
	}

	lock.Lock()
	defer lock.Unlock()

	parseConfig()

	genesisDir := path.Join(cfg.RootDir, "genesis")
	chainDir := path.Join(genesisDir, genDoc.ChainID)
	targetFile := chainDir + ".tar.gz"

	if !fileutil.Exist(targetFile) {
		err := common.TarIt(chainDir, genesisDir)
		if err != nil {
			return nil, err
		}
		err = common.GzipIt(chainDir+".tar", genesisDir)
		if err != nil {
			return nil, err
		}
	}

	byt, err := ioutil.ReadFile(targetFile)
	if err != nil {
		return nil, err
	}
	jsonBlob, err := jsoniter.Marshal(byt)
	if err != nil {
		return nil, err
	}
	return &core_types.ResultConfFile{F: jsonBlob}, nil
}
