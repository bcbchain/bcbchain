package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

// PathExists returns whether the given file or directory exists or not
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func PathSplit(path string) []string {
	segs := make([]string, 0)
	parentDir := filepath.Dir(path)
	if parentDir == "." || parentDir == path {
		segs = append(segs, path)
	} else {
		parentDir := filepath.Dir(path)
		baseDir := filepath.Base(path)
		segs = append(segs, PathSplit(parentDir)...)
		segs = append(segs, baseDir)
	}
	return segs
}

// MakeDir make dir with Permission 0777
func MakeDir(dir string) (bool, error) {
	if dir == "" {
		return false, fmt.Errorf("can not make an empty dir")
	}

	parentDir := filepath.Dir(dir)
	if parentDir == "." || parentDir == dir {
		return makeDir(dir)
	} else {
		segs := PathSplit(dir)
		dir = ""
		for _, seg := range segs {
			dir = filepath.Join(dir, seg)
			if _, err := makeDir(dir); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func makeDir(dir string) (bool, error) {
	ok, err := PathExists(dir)
	if err != nil {
		return false, err
	}
	if !ok {
		err := os.Mkdir(dir, os.ModePerm)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}
