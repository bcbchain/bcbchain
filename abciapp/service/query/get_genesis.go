package query

import (
	"errors"
	"github.com/bcbchain/bcbchain/abciapp/common"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"github.com/bcbchain/bclib/fs"
	"github.com/bcbchain/bclib/jsoniter"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
)

func (app *QueryConnection) getGenesis() ([]byte, error) {
	if statedbhelper.GetChainGenesisVersion() == 0 {
		// genesis from v1, return empty but no error.
		return []byte{}, nil
	}

	genesisTarGZFile := filepath.Join(common.GlobalConfig.Path, "genesis.tar.gz")
	if exist, err := fs.PathExists(genesisTarGZFile); err != nil {
		return nil, errors.New("can not stat file path:" + genesisTarGZFile)
	} else if !exist {
		if err := targzGenesis(genesisTarGZFile); err != nil {
			return nil, err
		}
	}

	byt, err := ioutil.ReadFile(genesisTarGZFile)
	if err != nil {
		return nil, err
	}
	jsonBlob, err := jsoniter.Marshal(byt)
	if err != nil {
		return nil, err
	}

	return jsonBlob, nil
}

func targzGenesis(target string) error {
	fi, err := ioutil.ReadDir(common.GlobalConfig.Path)
	if err != nil {
		return err
	}

	minVersion := ""
	for _, v := range fi {
		if !v.IsDir() && strings.HasPrefix(v.Name(), "genesis-smcrunsvc_") && strings.HasSuffix(v.Name(), ".tar.gz") {
			version := strings.TrimPrefix(v.Name(), "genesis-smcrunsvc_")
			version = strings.TrimSuffix(version, ".tar.gz")
			if len(minVersion) == 0 {
				minVersion = version
				continue
			}

			if compareVersion(minVersion, version) > 0 {
				minVersion = version
			}
		}
	}

	if len(minVersion) == 0 {
		return errors.New("no genesis smcrunsvc files")
	}

	gFile := filepath.Join(common.GlobalConfig.Path, "genesis-smcrunsvc_"+minVersion+".tar.gz")
	return fs.TarGz(gFile, target, 1)
}

func compareVersion(v1, v2 string) int {
	if v1 == "" {
		return -1
	}
	v1s := strings.Split(v1, ".")
	v2s := strings.Split(v2, ".")
	if !(len(v1s) > 0 && len(v1s) == len(v2s)) {
		return 0
	}

	code := 0
	for i := 0; i < len(v1s); i++ {
		v1, err1 := strconv.Atoi(v1s[i])
		if err1 != nil {
			return 0
		}
		v2, err2 := strconv.Atoi(v2s[i])
		if err2 != nil {
			return 0
		}

		if v1 > v2 {
			code = 1
			return code
		} else if v1 < v2 {
			code = -1
			return code
		}
	}

	return code
}
