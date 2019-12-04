package fs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CopyDir copy files and sub dirs in srcPath dir to destPath dir,
// srcPath and destPath must exist.
// copy files that name match with matchRegexp and not match with excludeRegexp
// matchRegexp: (.go)$
// excludeRegexp: (_autogen_)
func CopyDir(srcPath, destPath, matchRegexp, excludeRegexp string) error {

	srcPath = strings.Replace(srcPath, "\\", "/", -1)
	destPath = strings.Replace(destPath, "\\", "/", -1)

	if srcInfo, err := os.Stat(srcPath); err != nil {
		return err
	} else {
		if !srcInfo.IsDir() {
			return errors.New("open " + srcPath + ": is not a directory")
		}
	}
	if destInfo, err := os.Stat(destPath); err != nil {
		return err
	} else {
		if !destInfo.IsDir() {
			return errors.New("open " + destPath + ": is not a directory")
		}
	}

	err := filepath.Walk(srcPath, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if matchRegexp == "" && excludeRegexp == "" {
			if !f.IsDir() {
				path = strings.Replace(path, "\\", "/", -1)
				destNewPath := strings.Replace(path, srcPath, destPath, -1)
				_, err := CopyFile(path, destNewPath)
				if err != nil {
					return err
				}
			}
		} else {
			match, errmc := regexp.MatchString(matchRegexp, f.Name())
			if errmc != nil {
				return errors.New(errmc.Error())
			}

			exclude, errex := regexp.MatchString(excludeRegexp, f.Name())
			if errex != nil {
				return errors.New(errmc.Error())
			}

			if !f.IsDir() && match && !exclude {
				path = strings.Replace(path, "\\", "/", -1)
				destNewPath := strings.Replace(path, srcPath, destPath, -1)
				_, err := CopyFile(path, destNewPath)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})

	return err
}

// CopyFile copy src file to dest
func CopyFile(src, dest string) (w int64, err error) {
	src = strings.Replace(src, "\\", "/", -1)
	dest = strings.Replace(dest, "\\", "/", -1)

	srcFile, err := os.Open(src)
	if err != nil {
		return
	}
	defer srcFile.Close()
	destSplitPathDirs := strings.Split(dest, "/")

	destSplitPath := ""
	for index, dir := range destSplitPathDirs {
		if index < len(destSplitPathDirs)-1 {
			destSplitPath = destSplitPath + dir + "/"
			b, e := PathExists(destSplitPath)
			if e != nil {
				err = errors.New(e.Error())
				return
			}
			if b == false {
				e := os.Mkdir(destSplitPath, os.ModePerm)
				if e != nil {
					err = errors.New(e.Error())
					return
				}
			}
		}
	}
	dstFile, err := os.Create(dest)
	if err != nil {
		return
	}
	defer dstFile.Close()

	return io.Copy(dstFile, srcFile)
}
