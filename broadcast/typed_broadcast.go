package broadcast

import (
	"context"
	"reflect"
	"sync"
)

type TypedBroadcaster interface {
	Notify(context.Context, interface{})
	AddListener(function interface{}) ListenerStub
	Stop()
}

func NewTypedBroadcaster() TypedBroadcaster {
	return &typedbroadcasterInstance{
		RWMutex:      new(sync.RWMutex),
		broadcastMap: make(map[reflect.Type]*broadcasterInstance),
	}
}

type ListenerStub interface {
	Wait()
}

type typedbroadcasterInstance struct {
	*sync.RWMutex
	broadcastMap map[reflect.Type]*broadcasterInstance
}

func (tb *typedbroadcasterInstance) Notify(ctx context.Context, v interface{}) {
	bType := reflect.TypeOf(v)
	tb.RLock()
	defer tb.RUnlock()
	if broadcasterIns, ok := tb.broadcastMap[bType]; ok {
		broadcasterIns.Send(ctx, v)
	}
}

func (tb *typedbroadcasterInstance) Stop() {
	tb.Lock()
	defer tb.Unlock()
	wg := new(sync.WaitGroup)
	for _, b := range tb.broadcastMap {
		if b != nil {
			wg.Add(1)
			go func(bi *broadcasterInstance) {
				defer wg.Done()
				allowPanic("broadcast.main", bi.Stop)
			}(b)
		}
	}
	wg.Wait()
	/* reset map */
	tb.broadcastMap = make(map[reflect.Type]*broadcasterInstance)
}

// fn must like func(context.Context,Args)
func (tb *typedbroadcasterInstance) AddListener(fn interface{}) ListenerStub {
	fnV := reflect.ValueOf(fn)
	fnT := fnV.Type()
	bType := fnT.In(1)
	tb.Lock()
	defer tb.Unlock()
	if broadcasterIns, ok := tb.broadcastMap[bType]; ok {
		return broadcasterIns.AddListener(fnV)
	} else {
		ins := newInstance()
		stub := ins.AddListener(fnV)
		tb.broadcastMap[bType] = ins
		return stub
	}
}
