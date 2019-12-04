package common

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

var gopath string

// GoPath returns GOPATH env variable value. If it is not set, this function
// will try to call `go env GOPATH` subcommand.
func GoPath() string {
	if gopath != "" {
		return gopath
	}

	path := os.Getenv("GOPATH")
	if len(path) == 0 {
		goCmd := exec.Command("go", "env", "GOPATH")
		out, err := goCmd.Output()
		if err != nil {
			panic(fmt.Sprintf("failed to determine gopath: %v", err))
		}
		path = string(out)
	}
	gopath = path
	return path
}

// TrapSignal catches the SIGTERM and executes cb function. After that it exits
// with code 1.
func TrapSignal(cb func(sig os.Signal)) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	go func() {
		for sig := range c {
			fmt.Printf("captured %v, exiting...\n", sig)
			if cb != nil {
				cb(sig)
			}
			os.Exit(1)
		}
	}()
	select {}
}

// Kill the running process by sending itself SIGTERM.
func Kill() error {
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}
	return p.Signal(syscall.SIGTERM)
}

func Exit(s string) {
	fmt.Printf(s + "\n")
	os.Exit(1)
}

func EnsureDir(dir string, mode os.FileMode) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, mode)
		if err != nil {
			return fmt.Errorf("Could not create directory %v. %v", dir, err)
		}
	}
	return nil
}

func IsDirEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		if os.IsNotExist(err) {
			return true, err
		}
		// Otherwise perhaps a permission
		// error or some other error.
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func ReadFile(filePath string) ([]byte, error) {
	return ioutil.ReadFile(filePath)
}

func MustReadFile(filePath string) []byte {
	fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		Exit(Fmt("MustReadFile failed: %v", err))
		return nil
	}
	return fileBytes
}

func WriteFile(filePath string, contents []byte, mode os.FileMode) error {
	return ioutil.WriteFile(filePath, contents, mode)
}

func MustWriteFile(filePath string, contents []byte, mode os.FileMode) {
	err := WriteFile(filePath, contents, mode)
	if err != nil {
		Exit(Fmt("MustWriteFile failed: %v", err))
	}
}

// WriteFileAtomic creates a temporary file with data and the perm given and
// swaps it atomically with filename if successful.
func WriteFileAtomic(filename string, data []byte, perm os.FileMode) error {
	var (
		dir      = filepath.Dir(filename)
		tempFile = filepath.Join(dir, "write-file-atomic-"+RandStr(32))
		// Override in case it does exist, create in case it doesn't and force kernel
		// flush, which still leaves the potential of lingering disk cache.
		flag = os.O_WRONLY | os.O_CREATE | os.O_SYNC | os.O_TRUNC
	)

	f, err := os.OpenFile(tempFile, flag, perm)
	if err != nil {
		return err
	}
	// Clean up in any case. Defer stacking order is last-in-first-out.
	defer os.Remove(f.Name())
	defer f.Close()

	if n, err := f.Write(data); err != nil {
		return err
	} else if n < len(data) {
		return io.ErrShortWrite
	}
	// Close the file before renaming it, otherwise it will cause "The process
	// cannot access the file because it is being used by another process." on windows.
	f.Close()

	return os.Rename(f.Name(), filename)
}

//--------------------------------------------------------------------------------

func Tempfile(prefix string) (*os.File, string) {
	file, err := ioutil.TempFile("", prefix)
	if err != nil {
		PanicCrisis(err)
	}
	return file, file.Name()
}

func Tempdir(prefix string) (*os.File, string) {
	tempDir := os.TempDir() + "/" + prefix + RandStr(12)
	err := EnsureDir(tempDir, 0700)
	if err != nil {
		panic(Fmt("Error creating temp dir: %v", err))
	}
	dir, err := os.Open(tempDir)
	if err != nil {
		panic(Fmt("Error opening temp dir: %v", err))
	}
	return dir, tempDir
}

//--------------------------------------------------------------------------------

func Prompt(prompt string, defaultValue string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultValue, nil
	}
	return line, nil
}

func TarIt(source, target string) error {
	filename := filepath.Base(source)
	target = filepath.Join(target, fmt.Sprintf("%s.tar", filename))
	tarFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	tarball := tar.NewWriter(tarFile)
	defer tarball.Close()

	info, err := os.Stat(source)
	if err != nil {
		return nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	return filepath.Walk(source,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}

			if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
			}

			if err := tarball.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(tarball, file)
			return err
		})
}

func UnTar(tarball, target string) error {
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}

func GzipIt(source, target string) error {
	reader, err := os.Open(source)
	if err != nil {
		return err
	}

	filename := filepath.Base(source)
	target = filepath.Join(target, fmt.Sprintf("%s.gz", filename))
	writer, err := os.Create(target)
	if err != nil {
		return err
	}
	defer writer.Close()

	newWriter := gzip.NewWriter(writer)
	newWriter.Name = filename
	defer newWriter.Close()

	_, err = io.Copy(newWriter, reader)
	return err
}

func UnGzip(source, target string) error {
	reader, err := os.Open(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	archive, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer archive.Close()

	target = filepath.Join(target, archive.Name)
	writer, err := os.Create(target)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, archive)
	return err
}
