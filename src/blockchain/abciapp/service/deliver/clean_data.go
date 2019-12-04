package deliver

import (
	"blockchain/common/statedbhelper"
	"common/dockerlib"
	"os"
	"path"
	"path/filepath"
)

func (app *AppDeliver) cleanData() error {
	chainID := statedbhelper.GetChainID()
	if chainID != "" {
		dockerlib.GetDockerLib().Reset(chainID + ".")
	}

	home := os.Getenv("HOME")
	var err error

	if err = os.RemoveAll(home + "/.build/bin"); err != nil {
		return err
	}

	if err = os.RemoveAll(home + "/.build/build"); err != nil {
		return err
	}

	if err = os.RemoveAll(home + "/.build/log"); err != nil {
		return err
	}

	if err = delForksFiles(); err != nil {
		return err
	}

	if err = os.RemoveAll(home + "/.appstate.db"); err != nil {
		return err
	}

	return nil
}

func delForksFiles() error {
	currentPath, err := os.Executable()
	if err != nil {
		return err
	}

	currentDir := path.Dir(currentPath)
	if err = os.RemoveAll(filepath.Join(currentDir, "abci-forks.json")); err != nil {
		return err
	}

	if err = os.RemoveAll(filepath.Join(currentDir, "abci-forks.json.sig")); err != nil {
		return err
	}
	return nil
}
