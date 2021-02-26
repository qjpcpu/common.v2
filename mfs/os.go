package mfs

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

type OS interface {
	Walk(rootDir string, includingHidden bool, cb func(string))
	Open(file string) (io.ReadCloser, error)
	MkdirAll(dir string) error
	WriteToFile(string, io.Reader) error
	Remove(string) error
	Exist(string) bool
}

type osImpl struct{}

func newOs() OS {
	return osImpl{}
}

func (self osImpl) Walk(rootDir string, includingHidden bool, cb func(string)) {
	_ = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if parent := filepath.Dir(path); !includingHidden && self.isHidden(parent) && !self.equal(rootDir, parent) {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			if includingHidden || !self.isHidden(path) {
				cb(path)
			}
		}
		return nil
	})
}

func (osImpl) isHidden(f string) bool {
	base := filepath.Base(f)
	return base != "." && base != ".." && strings.HasPrefix(base, ".")
}

func (osImpl) equal(f1, f2 string) bool {
	f1, _ = filepath.Abs(f1)
	f2, _ = filepath.Abs(f2)
	return f1 == f2
}

func (osImpl) Exist(f string) bool {
	if _, err := os.Stat(f); err != nil && os.IsNotExist(err) {
		return false
	}
	return true
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
