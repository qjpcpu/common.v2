package mfs

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

type memFileSystem struct {
	origRootDir  string
	drv          FilesystemEventDriver
	fileSet      SortedFileSet
	removedFiles map[string]struct{}
	ios          OS
	doPersist    PersistFunc
	fc           *fileExistCache
}

type createOpt struct {
	Prefix          string
	IncludingHidden bool
}

// CreateOption of fs
type CreateOption func(*createOpt)

// WithPrefix for fs root
func WithPrefix(s string) CreateOption {
	return func(o *createOpt) {
		o.Prefix = prependSlash(s)
	}
}

// WithHidden with hidden file/dir
func WithHidden() CreateOption {
	return func(o *createOpt) {
		o.IncludingHidden = true
	}
}

// New fs mount rootDir to /
func New(rootDir string) FileSystem {
	return NewWithOptions(rootDir)
}

// NewWithOptions fs mount rootDir with prefix
func NewWithOptions(rootDir string, opts ...CreateOption) FileSystem {
	return Create(NewFSEventDriver(), newOs(), rootDir, opts...)
}

// Create mem fs
func Create(drv FilesystemEventDriver, ios OS, rootDir string, opts ...CreateOption) FileSystem {
	o := new(createOpt)
	for _, fn := range opts {
		fn(o)
	}
	rootDir = absOfFile(rootDir)
	ft := &memFileSystem{
		origRootDir:  rootDir,
		fileSet:      NewSortedFileSet(),
		removedFiles: make(map[string]struct{}),
		drv:          drv,
		ios:          ios,
	}
	ft.doPersist = ft._persitFile
	if isStrBlank(rootDir) {
		return ft
	}
	ios.Walk(rootDir, o.IncludingHidden, func(path string) {
		path = absOfFile(path)
		filename := join(o.Prefix, trimPrefix(path, ft.origRootDir))
		ft.fileSet.Set(filename, createFile(drv, ios, path, filename))
	})
	return ft
}

func (ft *memFileSystem) AddPersistHook(hook PersistHook) {
	ft.doPersist = hook(ft.doPersist)
}

func (ft *memFileSystem) Events() FileEventRegister {
	return ft.drv
}

func (ft *memFileSystem) Persist(rootDir string) error {
	if isStrBlank(rootDir) {
		return fmt.Errorf("empty dst directory %s", rootDir)
	}
	// set file exist checker
	ft.fc = newFileExistCache(ft.ios)

	rootDir = absOfFile(rootDir)
	ft.fileSet.Foreach("/", func(name string, f File) {
		fullname := join(rootDir, name)
		ft.doPersist(f, fullname)
	})
	for _, file := range ft.getMarkRemovedFiles() {
		fullname := join(rootDir, file)
		ft.ios.Remove(fullname)
	}
	return nil
}

func (ft *memFileSystem) ListFile() (list []File) {
	ft.fileSet.Foreach("/", func(name string, f File) {
		list = append(list, f)
	})
	return
}

func (ft *memFileSystem) ReadDir(dir string, recursive bool) (list []File) {
	dir = fmtDirWithoutSlash(prependSlash(dir))
	ft.fileSet.Foreach(dir, func(name string, f File) {
		if recursive || dir == filepath.Dir(name) {
			list = append(list, f)
		}
	})
	return
}

func (ft *memFileSystem) CreateFile(name string) File {
	name = prependSlash(name)
	if !ft.isValidFileName(name) {
		panic("invalid filename " + name)
	}
	return ft.createFileWithOrigFile(name, "")
}

func (ft *memFileSystem) createFileWithOrigFile(name string, origFile string) File {
	if ft.isValidFileName(name) {
		f := createFile(ft.drv, ft.ios, origFile, name)
		if ft.fileSet.Contains(name) {
			ft.fileSet.Remove(name)
		}
		ft.fileSet.Set(name, f)
		ft.unmarkFileRemoved(name)
		ft.drv.Trigger(FileCreatedEvent{Name: name})
		return f
	}
	return nil
}

func (ft *memFileSystem) Remove(name string) {
	name = prependSlash(name)
	if !ft.isValidFileName(name) {
		return
	}
	if ft.removeFile(name) {
		return
	}
	files := ft.ReadDir(name, true)
	for _, f := range files {
		ft.removeFile(f.Name())
	}
}

func (ft *memFileSystem) removeFile(name string) bool {
	if removed := ft.fileSet.Remove(name); removed {
		ft.markFileRemoved(name)
		ft.drv.Trigger(FileRemovedEvent{Name: name})
		return true
	}
	return false
}

func (ft *memFileSystem) Erase(name string) {
	name = prependSlash(name)
	if !ft.isValidFileName(name) {
		return
	}
	if ft.removeFile(name) {
		ft.unmarkFileRemoved(name)
		return
	}
	files := ft.ReadDir(name, true)
	for _, f := range files {
		name = f.Name()
		if ft.removeFile(name) {
			ft.unmarkFileRemoved(name)
		}
	}
}

func (ft *memFileSystem) DropIfExist(names ...string) {
	for _, name := range names {
		ft.dropIfExist(name)
	}
}

func (ft *memFileSystem) dropIfExist(name string) {
	name = prependSlash(name)
	ft.AddPersistHook(func(next PersistFunc) PersistFunc {
		return func(f File, absName string) error {
			targetDir := getDirByFile(absName, f.Name())
			if isFileOrSubFile(f.Name(), name) && ft.fc.Exist(filepath.Join(targetDir, name)) {
				return nil
			}
			return next(f, absName)
		}
	})
}

func (ft *memFileSystem) GetFile(name string) File {
	name = prependSlash(name)
	return ft.fileSet.Get(name)
}

func (ft *memFileSystem) Exist(name string) bool {
	name = prependSlash(name)
	return ft.fileSet.Contains(name)
}

func (ft *memFileSystem) Rename(from string, to string) {
	from = prependSlash(from)
	to = prependSlash(to)
	if ft.renameFile(from, to) {
		return
	}
	files := getFileNames(ft.ReadDir(from, true))
	for _, f := range files {
		ft.renameFile(f, join(to, trimPrefix(f, from)))
	}
}

func (ft *memFileSystem) renameFile(from string, to string) (ok bool) {
	if ft.isValidFileName(from) && ft.isValidFileName(to) && from != to {
		if !ft.Exist(from) {
			return
		}
		if ft.Exist(to) {
			ft.fileSet.Remove(to)
			ft.drv.Trigger(FileRemovedEvent{Name: to})
		}
		ok = true
		ft.fileSet.Rename(from, to)
		ft.drv.Trigger(FileRenamedEvent{Old: from, New: to})
		//ft.markFileRemoved(from)
		ft.unmarkFileRemoved(to)
	}
	return
}

func (ft *memFileSystem) Walk(dir string, visit WalkFunc) {
	dir = fmtDirWithoutSlash(prependSlash(dir))
	ft.fileSet.Foreach(dir, func(name string, f File) {
		visit(f)
	})
}

func (ft *memFileSystem) Mount(dir string, fs FileSystem) {
	if isStrBlank(dir) {
		return
	}
	dir = absOfFile(prependSlash(dir))
	files := fs.ListFile()
	mapPath := func(a string) string { return join(dir, a) }
	for _, file := range files {
		if file.IsDirty() {
			f := ft.CreateFile(mapPath(file.Name()))
			f.SetContent(file.Content())
		} else {
			ft.createFileWithOrigFile(mapPath(file.Name()), file.OriginalName())
		}
	}
}

func (ft *memFileSystem) Copy(from, to string) {
	from = prependSlash(from)
	to = prependSlash(to)
	if !ft.isValidFileName(from) || !ft.isValidFileName(to) {
		return
	}
	if ft.copyFile(from, to) {
		return
	}
	fromList := getFileNames(ft.ReadDir(from, true))
	for _, file := range fromList {
		ft.copyFile(file, strings.Replace(file, from, to, 1))
	}
}

func (ft *memFileSystem) copyFile(from, to string) bool {
	if !ft.Exist(from) {
		return false
	}
	if ft.Exist(to) {
		ft.removeFile(to)
	}
	file := ft.GetFile(from)
	if file.IsDirty() {
		f := ft.CreateFile(to)
		f.SetContent(file.Content())
	} else {
		ft.createFileWithOrigFile(to, file.OriginalName())
	}
	return true
}

func (ft *memFileSystem) isValidFileName(f string) bool {
	return strings.HasPrefix(f, "/") && !strings.HasSuffix(f, "/")
}

func (ft *memFileSystem) markFileRemoved(file string) {
	ft.removedFiles[file] = struct{}{}
}

func (ft *memFileSystem) unmarkFileRemoved(file string) {
	delete(ft.removedFiles, file)
}

func (ft *memFileSystem) getMarkRemovedFiles() []string {
	var files []string
	for f := range ft.removedFiles {
		files = append(files, f)
	}
	return files
}

func (ft *memFileSystem) mkAll(dir string) error {
	return ft.ios.MkdirAll(dir)
}

func (ft *memFileSystem) writeFile(filename string, f File) (err error) {
	f.Read(func(r io.Reader) {
		err = ft.ios.WriteToFile(filename, r)
	})
	return
}

func (ft *memFileSystem) _persitFile(f File, fullname string) error {
	if f.OriginalName() != fullname || f.IsDirty() {
		if dir := filepath.Dir(fullname); !ft.ios.Exist(dir) {
			if err := ft.mkAll(dir); err != nil {
				return err
			}
		}
		if err := ft.writeFile(fullname, f); err != nil {
			return err
		}
	}
	return nil
}
