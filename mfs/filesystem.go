package mfs

import (
	"io"
)

// FileSystem in mem
type FileSystem interface {
	Events() FileEventRegister
	Persist(rootDir string) error
	ListFile() []File
	ReadDir(dir string, recursive bool) []File
	CreateFile(name string) File
	Remove(string)
	Erase(string)
	GetFile(string) File
	Exist(string) bool
	Rename(string, string)
	Copy(string, string)
	Walk(string, WalkFunc)
	Mount(string, FileSystem)
}

// WalkFunc walk fs
type WalkFunc func(File)

// ContentMapFunc mapper stream
type ContentMapFunc func(io.Reader, io.Writer)

// ReadFunc read content
type ReadFunc func(io.Reader)

// File in mem
type File interface {
	OriginalName() string
	Name() string
	Content() []byte
	SetContent([]byte)
	Read(ReadFunc)
	Map(ContentMapFunc)
	IsDirty() bool
}
