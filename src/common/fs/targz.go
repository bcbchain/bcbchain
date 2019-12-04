package fs

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/tendermint/tmlibs/log"
)

// UnTarGz takes a destination path and a reader; a tar reader loops over the tar.gz file
// creating the file structure at 'dst' along the way, and writing any files
// nolint gocyclo // 這個方法鬼子寫的，抄來的，不改了
func UnTarGz(dst string, r io.Reader, l log.Logger) error {

	gzr, err := gzip.NewReader(r)
	if err != nil {
		if l != nil {
			l.Error("unTar", "err", err.Error())
		} else {
			fmt.Println("unTar err : " + err.Error())
		}
		return err
	}
	defer func() {
		if e := gzr.Close(); e != nil {
			if l != nil {
				l.Warn("UnTarGz close Reader Error", "err", err)
			} else {
				fmt.Println("UnTarGz close reader error:", err.Error())
			}
		}
	}()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

			// return any other error
		case err != nil:
			return err

			// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0750); err != nil {
					if l != nil {
						l.Error("unTar MkdirAll", "err", err.Error())
					} else {
						fmt.Println("unTar MkdirAll err : " + err.Error())
					}
					return err
				}
			}

			// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err = io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			if err = f.Close(); err != nil {
				if l != nil {
					l.Warn("file can't be closed", "f", target)
				} else {
					fmt.Println("file can't be closed: " + target)
				}
			}
		}
	}
}

// TarGz tar for srcPath and create destFile
// flag = 0,not contains srcPath dir, flag = 1, contains srcPath
func TarGz(srcPath string, destFile string, flag int) error {
	fw, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer fw.Close()

	// Gzip writer
	gw := gzip.NewWriter(fw)
	defer gw.Close()

	// Tar writer
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Check if it's a file or a directory
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	if fi.IsDir() {
		// handle source directory
		if flag == 0 {
			err := tarGzDir(srcPath, path.Base(""), tw)
			if err != nil {
				return err
			}
		} else if flag == 1 {
			err := tarGzFile(srcPath, path.Base(srcPath), tw, fi)
			if err != nil {
				return err
			}
			err = tarGzDir(srcPath, path.Base(srcPath), tw)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Invlaid flag")
		}

	} else {
		// handle file directly
		if flag == 0 {
			err := tarGzFile(srcPath, path.Base(""), tw, fi)
			if err != nil {
				return err
			}
		} else if flag == 1 {
			err := tarGzFile(srcPath, path.Base(srcPath), tw, fi)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Invlaid flag")
		}
	}

	return nil
}
func tarGzDir(srcDir string, recPath string, tw *tar.Writer) error {
	// Open source diretory
	dir, err := os.Open(srcDir)
	if err != nil {
		return err
	}
	defer dir.Close()

	// Get file info slice
	fis, err := dir.Readdir(0)
	if err != nil {
		return err
	}

	for _, fi := range fis {
		// Append path
		curPath := srcDir + "/" + fi.Name()

		err := tarGzFile(curPath, recPath+"/"+fi.Name(), tw, fi)
		if err != nil {
			return err
		}

		// Check it is directory or file
		if fi.IsDir() {
			// Directory
			// (Directory won't add unitl all subfiles are added)
			err := tarGzDir(curPath, recPath+"/"+fi.Name(), tw)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func tarGzFile(srcFile string, recPath string, tw *tar.Writer, fi os.FileInfo) error {
	if fi.IsDir() {
		// Create tar header
		hdr := new(tar.Header)
		// if last character of header name is '/' it also can be directory
		// but if you don't set Typeflag, error will occur when you untargz
		hdr.Name = recPath + "/"
		hdr.Typeflag = tar.TypeDir
		hdr.Size = 0
		hdr.Mode = int64(fi.Mode())
		hdr.ModTime = fi.ModTime()

		// Write hander
		err := tw.WriteHeader(hdr)
		if err != nil {
			return err
		}
	} else {
		// File reader
		fr, err := os.Open(srcFile)
		if err != nil {
			return err
		}
		defer fr.Close()

		// Create tar header
		hdr := new(tar.Header)
		hdr.Typeflag = tar.TypeReg
		hdr.Name = recPath
		hdr.Size = fi.Size()
		hdr.Mode = int64(fi.Mode())
		hdr.ModTime = fi.ModTime()

		// Write hander
		err = tw.WriteHeader(hdr)
		if err != nil {
			return err
		}

		// Write file data
		_, err = io.Copy(tw, fr)
		if err != nil {
			return err
		}
	}
	return nil
}
