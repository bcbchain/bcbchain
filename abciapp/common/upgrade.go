package common

import (
	"errors"
	"github.com/bcbchain/bcbchain/smcbuilder"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

// IsExistUpgradeFile check .upgrade file is exist
func IsExistUpgradeFile() (string, bool) {
	filePath := GlobalConfig.Path + "/.upgrade"

	if _, err := os.Stat(filePath); err == nil {
		return filePath, true
	} else if os.IsNotExist(err) {
		return "", false
	} else {
		panic(err)
	}
}

// UpgradeBin upgrade contract binary executable file
func UpgradeBin(filePath string, l log.Logger) {
	l.Info("upgrade docker binary file begin")

	buildPath := buildPath()

	orgIDs := make([]string, 0)
	binPath := buildPath + "/bin"
	rd, err := ioutil.ReadDir(binPath)
	if err != nil {
		if os.IsNotExist(err) {
			goto END
			//return
		}
		panic(err)
	}
	for _, fi := range rd {
		if fi.IsDir() && fi.Name() != "." && fi.Name() != ".." {
			orgIDs = append(orgIDs, fi.Name())
		}
	}

	err = os.RemoveAll(binPath)
	if err != nil {
		panic(err)
	}

	// init smcbuilder
	smcbuilder.Init(l, buildPath)

	// rebuild
	for _, orgID := range orgIDs {
		_, err = smcbuilder.GetInstance().GetContractDllPath(0, 0, orgID)
	}

END:
	err = os.RemoveAll(filePath)
	if err != nil {
		panic(err)
	}
	l.Info("upgrade docker binary file end")
}

func buildPath() string {
	var buildPath string
	if runtime.GOOS == "windows" {
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		dir := filepath.Dir(ex)
		if dir == "" {
			panic(errors.New("failed to get path of forks file"))
		}

		buildPath = dir + "\\.build"
	} else {
		buildPath = os.Getenv("HOME") + "/.build"
	}

	return buildPath
}
