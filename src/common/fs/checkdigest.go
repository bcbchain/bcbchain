package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// sha2gen 已经改用sha256，因爲 md5 已經被認爲不安全， lint 工具一直報警
func sha2gen(fullPath string) (string, error) {
	data, err := ioutil.ReadFile(filepath.Clean(fullPath))
	if err != nil {
		return "", err
	}
	h := sha256.New()
	_, err = h.Write(data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Sha2Gen generate a sha256sum file for file "a" with filename "a.sha2"
func Sha2Gen(fullPath string) (bool, error) {
	md5, err := sha2gen(fullPath)
	if err != nil {
		return false, err
	}
	f, err := os.Create(fullPath + ".sha2")
	if err != nil {
		return false, err
	}
	defer func() {
		if e := f.Close(); e != nil {
			fmt.Println("Sha2Gen cause error:", e)
		}
	}()
	if _, err = f.WriteString(md5); err != nil {
		return false, err
	}
	if err = f.Sync(); err != nil {
		return false, err
	}

	return true, nil
}

// CheckSha2 verify the file's sha256sum, confirm it's not modified.
func CheckSha2(fullPath string) bool {
	sha2, err := sha2gen(fullPath)
	if err != nil {
		return false
	}

	fi, err := os.Open(filepath.Clean(fullPath + ".sha2"))
	if err != nil {
		return false
	}
	defer func() {
		if e := fi.Close(); e != nil {
			fmt.Println("CheckSha2 cause error:", e)
		}
	}()

	buf := make([]byte, 64)
	n, err := fi.Read(buf)
	if n != 64 || err != nil {
		return false
	}
	if strings.Compare(sha2, string(buf)) != 0 {
		return false
	}
	return true
}
