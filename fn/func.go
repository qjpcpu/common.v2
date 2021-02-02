package fn

import (
	"fmt"
	"reflect"
)

// Decorator for function
type Decorator interface {
	// AddMW append a middleware
	AddMW(interface{}) Decorator
	// PrependMW insert a middleware
	PrependMW(interface{}) Decorator
	// MWCount size of middlewares
	MWCount() int
	// InsertMW into list
	InsertMW(int, interface{}) Decorator
	// Export as new function
	Export(fnPtr interface{})
}

// DecoratorOf given function
func DecoratorOf(fn interface{}) Decorator {
	fnTyp := reflect.TypeOf(fn)
	mustBeFunc(fnTyp)
	midTyp := reflect.FuncOf([]reflect.Type{fnTyp}, []reflect.Type{fnTyp}, false)
	return &fnDecorator{
		funcTyp:       fnTyp,
		middlewareTyp: midTyp,
		origFn:        reflect.ValueOf(fn),
	}
}

type fnDecorator struct {
	funcTyp, middlewareTyp reflect.Type
	origFn                 reflect.Value
	middlewares            []reflect.Value
}

// AddMW append a middleware
func (self *fnDecorator) AddMW(m interface{}) Decorator {
	return self.InsertMW(len(self.middlewares), m)
}

// InsertMW into list
func (self *fnDecorator) InsertMW(i int, m interface{}) Decorator {
	if i < 0 || i > len(self.middlewares) {
		panic("bad index")
	}
	self.mustBeMiddleware(m)
	val := reflect.ValueOf(m)
	self.middlewares = append(self.middlewares, val)
	for j := len(self.middlewares) - 1; j > i; j-- {
		self.middlewares[j] = self.middlewares[j-1]
	}
	self.middlewares[i] = val
	return self
}

// MWCount size of middlewares
func (self *fnDecorator) MWCount() int {
	return len(self.middlewares)
}

// PrependMW insert a middleware
func (self *fnDecorator) PrependMW(m interface{}) Decorator {
	return self.InsertMW(0, m)
}

// Export as new function
func (self *fnDecorator) Export(fnPtr interface{}) {
	typ := reflect.TypeOf(fnPtr)
	if typ.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("%v is not func ptr", typ))
	}
	typ = typ.Elem()
	mustBeFunc(typ)
	mustBeSameFuncType(self.funcTyp, typ)
	reflect.ValueOf(fnPtr).Elem().
		Set(self.build().Convert(typ))
}

func (self *fnDecorator) mustBeMiddleware(m interface{}) {
	typ := reflect.TypeOf(m)
	mustBeFunc(typ)
	mustBeSameFuncType(self.middlewareTyp, typ)
}

func (self *fnDecorator) build() reflect.Value {
	next := self.origFn
	for i := 0; i < len(self.middlewares); i++ {
		next = self.middlewares[i].Call([]reflect.Value{
			next.Convert(self.middlewares[i].Type().In(0)),
		})[0]
	}
	return next
}

/* utils functions */
func mustBeFunc(typ reflect.Type) {
	if typ.Kind() != reflect.Func {
		panic(fmt.Sprintf("%v is not function", typ))
	}
}

func mustBeSameFuncType(f1, f2 reflect.Type) {
	if f1.NumIn() != f2.NumIn() || f1.NumOut() != f2.NumOut() {
		panic(fmt.Sprintf("%v and %v are not same type", f1, f2))
	}
	compareArgument(f1, f2, f1.NumIn(), inType)
	compareArgument(f1, f2, f1.NumOut(), outType)
}

func compareArgument(f1, f2 reflect.Type, size int, inout func(reflect.Type, int) reflect.Type) {
	for i := 0; i < size; i++ {
		f1a, f2a := inout(f1, i), inout(f2, i)
		if f1a != f2a {
			if k := f1a.Kind(); k == reflect.Func && k == f2a.Kind() && f1a.ConvertibleTo(f2a) && f2a.ConvertibleTo(f1a) {
			} else {
				panic(fmt.Sprintf("%v and %v are not same type", f1, f2))
			}
		}
	}
}

func inType(typ reflect.Type, i int) reflect.Type {
	return typ.In(i)
}

func outType(typ reflect.Type, i int) reflect.Type {
	return typ.Out(i)
}
