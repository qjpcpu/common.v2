package mfs

import (
	"sort"
	"sync"
)

type SortedFileSet interface {
	Set(string, File)
	Get(string) File
	Remove(string) bool
	Rename(string, string) bool
	Contains(string) bool
	Foreach(string, func(string, File))
}

type sortedFileSet struct {
	sync.RWMutex
	fileList []string
	fileMap  map[string]File
}

func NewSortedFileSet() SortedFileSet {
	return &sortedFileSet{
		fileMap: make(map[string]File),
	}
}

func (s *sortedFileSet) Foreach(dir string, cb func(name string, f File)) {
	var names []string
	var files []File
	s.RWMutex.RLock()
	for _, file := range s.fileList {
		if isFileInDir(file, dir) {
			names = append(names, file)
			files = append(files, s.fileMap[file])
		}
	}
	s.RWMutex.RUnlock()

	for i, name := range names {
		cb(name, files[i])
	}
}

func (s *sortedFileSet) Get(name string) File {
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()
	return s.fileMap[name]
}

func (s *sortedFileSet) Contains(name string) bool {
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()
	_, ok := s.fileMap[name]
	return ok
}

func (s *sortedFileSet) Set(name string, f File) {
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()
	if _, ok := s.fileMap[name]; ok {
		s.fileMap[name] = f
	}
	s.fileMap[name] = f
	s.insertName(name)
}

func (s *sortedFileSet) Remove(name string) bool {
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()
	if _, ok := s.fileMap[name]; !ok {
		return false
	}
	delete(s.fileMap, name)
	s.removeName(name)
	return true
}

func (s *sortedFileSet) Rename(from, to string) bool {
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()
	if _, ok := s.fileMap[from]; !ok {
		return false
	}
	if _, ok := s.fileMap[to]; !ok {
		s.insertName(to)
	}
	s.fileMap[to] = s.fileMap[from]
	delete(s.fileMap, from)
	s.removeName(from)
	return true
}

func (s *sortedFileSet) insertName(name string) {
	s.fileList = append(s.fileList, name)
	sort.Strings(s.fileList)
}

func (s *sortedFileSet) removeName(name string) {
	var found int
	for i := 0; i < len(s.fileList); i++ {
		if s.fileList[i] == name {
			found = 1
		} else if found != 0 {
			s.fileList[i-found] = s.fileList[i]
		}
	}
	s.fileList = s.fileList[:len(s.fileList)-found]
}
