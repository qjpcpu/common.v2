package broadcast

import (
	"log"
	"runtime"
	"runtime/debug"
)

func allowPanic(tag string, f func()) {
	defer func() {
		if r := recover(); r != nil {
			logStack(tag, r)
		}
	}()
	f()
}

var logStack = func(tag string, r interface{}) {
	const size = 64 << 10
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	log.Printf("%s panic: %s: %s", tag, r, buf)
	debug.PrintStack()
}
