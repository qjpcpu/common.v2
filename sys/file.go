package sys

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/qjpcpu/common.v2/assert"
	"github.com/qjpcpu/common.v2/fp"
)

func MustWriteFile(filename string, data []byte) {
	dir := filepath.Dir(filename)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	} else {
		assert.ShouldBeNil(err, "write file %s %v", filename, err)
	}
	err = ioutil.WriteFile(filename, data, 0644)
	assert.ShouldBeNil(err, "write file %s %v", filename, err)
}

func MustReadFile(filename string) []byte {
	data, err := ioutil.ReadFile(filename)
	assert.ShouldBeNil(err, "read file %s %v", filename, err)
	return data
}

func IsExist(f string) bool {
	if _, err := os.Stat(f); err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func WalkDir(dir string) (dirList []string, fileList []string, err error) {
	dir = absOfDir(dir)
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			dirList = append(dirList, path)
		} else {
			fileList = append(fileList, path)
		}
		return nil
	})
	return
}

func MoveAndOverwrite(fromDir, toDir string) error {
	return MoveFiles(fromDir, toDir, nil)
}

func MoveFiles(fromDir, toDir string, shouldMove func(string, string) bool) error {
	return _MoveFiles(
		fromDir,
		toDir,
		shouldMove,
		func(d string) []string {
			_, f, _ := WalkDir(d)
			return f
		},
		func(d string) error {
			return os.MkdirAll(d, 0755)
		},
		IsExist,
		os.Rename,
	)
}

func _MoveFiles(fromDir, toDir string,
	shouldMove func(string, string) bool,
	walkDir func(string) []string,
	mkdirAll func(string) error,
	isFileExist func(string) bool,
	mvFile func(string, string) error,
) error {
	fromFileToFileMap := func(f string) string {
		return filepath.Join(toDir, strings.TrimPrefix(f, fromDir))
	}

	/* get all from files */
	fromDir = absOfDir(fromDir)
	toDir = absOfDir(toDir)
	files := walkDir(fromDir)

	/* drop files by fn */
	if shouldMove != nil {
		fp.ListOf(files).Filter(func(f string) bool {
			return shouldMove(f, fromFileToFileMap(f))
		}).Result(&files)
	}

	/* get all to files */
	toFileList := fp.ListOf(files).Map(fromFileToFileMap).MustGetResult().([]string)

	/* get all dest directories */
	dirList := fp.ListOf(toFileList).Map(func(f string) string {
		return filepath.Dir(f)
	}).Uniq().Sort().Strings()

	/* make directories for dest files */
	for i, dir := range dirList {
		if (i == len(dirList)-1 || !strings.HasPrefix(dirList[i+1], dir)) && !isFileExist(dir) {
			mkdirAll(dir)
		}
	}

	/* move files */
	for i, from := range files {
		if err := mvFile(from, toFileList[i]); err != nil {
			return err
		}
	}
	return nil
}

func absOfDir(dir string) string {
	dir, _ = filepath.Abs(dir)
	if !strings.HasSuffix(dir, `/`) {
		dir += `/`
	}
	return dir
}
