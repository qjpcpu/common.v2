package broadcast

import (
	"reflect"
	"sync"
)

func runListenerSafe(rc chan message, stub *chanStub, fn reflect.Value) {
	allowPanic("broadcast.listener", func() {
		runListenerLoop(rc, stub, fn)
	})
}

func runListenerLoop(rc chan message, stub *chanStub, fn reflect.Value) {
	for {
		msg := <-rc
		v := msg.Payload
		rc <- msg
		rc = msg.Next
		if msg.isTerminateMessage() {
			stub.Close()
			return
		}
		allowPanic("broadcast.listener", func() {
			fn.Call([]reflect.Value{reflect.ValueOf(v.Ctx), reflect.ValueOf(v.Body)})
		})
	}
}

type chanStub struct {
	closeC chan struct{}
}

func newChanStub() *chanStub {
	return &chanStub{closeC: make(chan struct{}, 1)}
}

func (c *chanStub) Close() {
	close(c.closeC)
}

func (c *chanStub) Wait() {
	<-c.closeC
}

type chanStubList struct {
	list []*chanStub
	*sync.Mutex
}

func newChanStubList() *chanStubList {
	return &chanStubList{Mutex: new(sync.Mutex)}
}

func (cl *chanStubList) Append(s *chanStub) *chanStubList {
	cl.Lock()
	defer cl.Unlock()
	cl.list = append(cl.list, s)
	return cl
}

func (cl *chanStubList) Wait() {
	cl.Lock()
	defer cl.Unlock()
	for _, item := range cl.list {
		item.Wait()
	}
	// reset
	cl.list = cl.list[:0]
}
