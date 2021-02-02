package mfs

import (
	"io"
	"os"
	"path/filepath"
)

type OS interface {
	Walk(rootDir string, cb func(string))
	Open(file string) (io.ReadCloser, error)
	MkdirAll(dir string) error
	WriteToFile(string, io.Reader) error
	Remove(string) error
}

type osImpl struct{}

func newOs() OS {
	return osImpl{}
}

func (osImpl) Walk(rootDir string, cb func(string)) {
	_ = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			cb(path)
		}
		return nil
	})
}

func (osImpl) Open(file string) (io.ReadCloser, error) {
	return os.Open(file)
}

func (osImpl) MkdirAll(dir string) error {
	return os.MkdirAll(dir, 0755)
}

func (osImpl) Remove(file string) error {
	return os.RemoveAll(file)
}

func (osImpl) WriteToFile(filename string, src io.Reader) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = io.Copy(f, src); err != nil {
		return err
	}
	return nil
}
