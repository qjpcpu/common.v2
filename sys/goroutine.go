package sys

import (
	"runtime"
	"sync"
)

var grtPool = sync.Pool{New: func() interface{} {
	return newByteSlice(256)
}}

type byteSlice struct {
	buf []byte
}

func newByteSlice(size int) *byteSlice {
	return &byteSlice{
		buf: make([]byte, size),
	}
}
func (bs *byteSlice) reset() {
	for i := 0; i < len(bs.buf); i++ {
		bs.buf[i] = 0
	}
}

// GetGoroutineID should only used for debug
func GetGoroutineID() string {
	bs := grtPool.Get().(*byteSlice)
	bs.reset()
	runtime.Stack(bs.buf, false)
	offset := 10 // goroutine 12 [running]:
	var gid string
	for i := offset; i < len(bs.buf); i++ {
		if bs.buf[i] == byte(32) {
			gid = string(bs.buf[offset:i])
			break
		}
	}
	grtPool.Put(bs)
	return gid

}
