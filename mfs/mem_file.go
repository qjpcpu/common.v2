package mfs

import (
	"io"
	"io/ioutil"
	"sync"
)

type mFile struct {
	origFile    string
	name        string
	content     io.ReadCloser
	dirty       bool
	flock       sync.RWMutex
	loadOnce    sync.Once
	drv         FilesystemEventDriver
	unEventList []Stub
	ios         OS
}

func createFile(drv FilesystemEventDriver, ios OS, origFile string, name string) File {
	mf := &mFile{
		origFile: origFile,
		name:     name,
		drv:      drv,
		ios:      ios,
	}
	mf.registListener()
	return mf
}

func (f *mFile) registListener() {
	f.unEventList = append(f.unEventList, f.drv.OnEvent(f.rename))
	f.unEventList = append(f.unEventList, f.drv.OnEvent(f.unregistListener))
}

func (f *mFile) unregistListener(e FileRemovedEvent) {
	if e.Name == f.name {
		for _, fn := range f.unEventList {
			fn.Unbind()
		}
		f.unEventList = nil
	}
}

func (f *mFile) rename(e FileRenamedEvent) {
	if e.Old == f.name {
		f.name = e.New
		return
	}
}

func (f *mFile) Name() string {
	return f.name
}

func (f *mFile) OriginalName() string {
	return f.origFile
}

func (f *mFile) Map(fn ContentMapFunc) {
	f.flock.Lock()
	defer f.flock.Unlock()

	buf := NewBuffer(nil)
	src := f.loadContent()
	if src != nil {
		defer src.Close()
	}
	fn(src, buf)
	f.content = buf
	f.dirty = true

	f.drv.Trigger(FileModifiedEvent{Name: f.name})
}

func (f *mFile) Read(fn ReadFunc) {
	f.flock.RLock()
	defer f.flock.RUnlock()

	r1 := f.loadContent()
	if r1 == nil {
		return
	}
	defer r1.Close()
	fn(r1)
}

func (f *mFile) loadContent() io.ReadCloser {
	if f.content == nil {
		if f.origFile == "" {
		} else if file, err := f.ios.Open(f.origFile); err == nil {
			return file
		}
	}
	return f.content
}

func (f *mFile) IsDirty() bool {
	return f.dirty
}

func (f *mFile) Content() (body []byte) {
	f.Read(func(r1 io.Reader) {
		body, _ = ioutil.ReadAll(r1)
	})
	return
}

func (f *mFile) SetContent(data []byte) {
	f.Map(func(r io.Reader, w io.Writer) {
		w.Write(data)
	})
}
