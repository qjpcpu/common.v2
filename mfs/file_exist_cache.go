package mfs

import "sync"

type fileExistCache struct {
	os    OS
	rw    sync.RWMutex
	files map[string]bool
}

func newFileExistCache(o OS) *fileExistCache {
	return &fileExistCache{os: o, files: make(map[string]bool)}
}

func (c *fileExistCache) Exist(file string) bool {
	c.rw.RLock()
	e, ok := c.files[file]
	if ok {
		c.rw.RUnlock()
		return e
	}
	c.rw.RUnlock()

	// real check
	c.rw.Lock()
	defer c.rw.Unlock()
	if e, ok = c.files[file]; ok {
		return e
	}
	e = c.os.Exist(file)
	c.files[file] = e
	return e
}
