package mfs

import (
	"reflect"
	"sync"
)

type Stub interface {
	Unbind()
}

type FileEventRegister interface {
	OnEvent(fn interface{}) Stub
}

type FilesystemEventDriver interface {
	FileEventRegister
	Trigger(interface{})
}

func NewFSEventDriver() FilesystemEventDriver {
	return &filesystemEventDrv{
		typToHandler: make(map[reflect.Type][]handler),
	}
}

type stub struct {
	unbind func()
}

func (s *stub) Unbind() {
	s.unbind()
}

type handler struct {
	id int32
	fn reflect.Value
}

type filesystemEventDrv struct {
	sync.RWMutex
	hid             int32
	typToHandler    map[reflect.Type][]handler
	wildcardHandler []handler
}

func (drv *filesystemEventDrv) OnEvent(h interface{}) Stub {
	drv.RWMutex.Lock()
	defer drv.RWMutex.Unlock()
	typ := reflect.TypeOf(h).In(0)
	id := drv.genID()
	if drv.isWildType(typ) {
		drv.wildcardHandler = append(drv.wildcardHandler, handler{
			id: id,
			fn: reflect.ValueOf(h),
		})
	} else {
		drv.typToHandler[typ] = append(drv.typToHandler[typ], handler{
			id: id,
			fn: reflect.ValueOf(h),
		})
	}

	return &stub{
		unbind: drv.unEvent(typ, id),
	}
}

func (drv *filesystemEventDrv) genID() int32 {
	drv.hid += 1
	return drv.hid
}

func (drv *filesystemEventDrv) unEvent(typ reflect.Type, id int32) func() {
	return func() {
		drv.RWMutex.Lock()
		defer drv.RWMutex.Unlock()
		drv.typToHandler[typ] = removeHandlerByID(drv.typToHandler[typ], id)
		drv.wildcardHandler = removeHandlerByID(drv.wildcardHandler, id)
	}
}

func (drv *filesystemEventDrv) Trigger(evt interface{}) {
	var handlers []handler
	drv.RWMutex.RLock()
	typ := reflect.TypeOf(evt)
	if size := len(drv.typToHandler[typ]); size > 0 {
		handlers = make([]handler, len(drv.typToHandler[typ]))
		copy(handlers, drv.typToHandler[typ])
	}
	handlers = append(handlers, drv.wildcardHandler...)
	drv.RWMutex.RUnlock()

	for _, h := range handlers {
		h.fn.Call([]reflect.Value{reflect.ValueOf(evt)})
	}
}

func (drv *filesystemEventDrv) isWildType(typ reflect.Type) bool {
	return typ.Kind() == reflect.Interface
}

func removeHandlerByID(handlers []handler, id int32) []handler {
	var count int
	for i := 0; i < len(handlers); i++ {
		if handlers[i].id == id {
			count += 1
		} else if count != 0 {
			handlers[i-count] = handlers[i]
		}
	}
	return handlers[:len(handlers)-count]
}

type FileRenamedEvent struct {
	Old, New string
}

type FileRemovedEvent struct {
	Name string
}

type FileCreatedEvent struct {
	Name string
}

type FileModifiedEvent struct {
	Name string
}
