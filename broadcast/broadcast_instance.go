package broadcast

import (
	"context"
	"reflect"
)

type broadcasterInstance struct {
	listenc       chan chan (chan message)
	listenerStubs *chanStubList
	Sendc         chan<- payload
}

func newInstance() *broadcasterInstance {
	listenc := make(chan (chan (chan message)))
	sendc := make(chan payload)
	go runNotifierSafe(sendc, listenc)
	return &broadcasterInstance{
		listenc:       listenc,
		Sendc:         sendc,
		listenerStubs: newChanStubList(),
	}
}

func (b *broadcasterInstance) AddListener(fn reflect.Value) ListenerStub {
	c := make(chan chan message)
	b.listenc <- c
	rc := <-c
	stub := newChanStub()
	b.listenerStubs.Append(stub)
	go runListenerSafe(rc, stub, fn)
	return stub
}

func (b *broadcasterInstance) Send(ctx context.Context, v interface{}) {
	b.Sendc <- newPayload(ctx, v)
}

func (b *broadcasterInstance) Stop() {
	close(b.Sendc)
	b.listenerStubs.Wait()
}

func runNotifierSafe(sendc chan payload, listenc chan chan chan message) {
	allowPanic("broadcast.notifier", func() {
		runNotifierLoop(sendc, listenc)
	})
}

func runNotifierLoop(sendc chan payload, listenc chan chan chan message) {
	currc := make(chan message, 1)
	for {
		select {
		case v, ok := <-sendc:
			if !ok {
				currc <- terminationMsg
				return
			}
			c := make(chan message, 1)
			b := message{Next: c, Payload: v}
			currc <- b
			currc = c
		case r := <-listenc:
			r <- currc
		}
	}
}
