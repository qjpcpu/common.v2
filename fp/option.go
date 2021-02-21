package fp

import (
	"reflect"
)

var (
	optionType = reflect.TypeOf((*Option)(nil)).Elem()
	None       = NewOption(nil)
)

type Option interface {
	IsSome() bool
	IsNone() bool
	Val() interface{}
}

type optionImpl struct {
	val interface{}
}

func NewOption(v interface{}) Option {
	return optionImpl{val: v}
}

func (o optionImpl) IsNone() bool {
	if o.val == nil {
		return true
	}
	switch reflect.TypeOf(o.val).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan, reflect.Interface, reflect.Func, reflect.UnsafePointer:
		if reflect.ValueOf(o.val).IsNil() {
			return true
		}
	}
	return false
}

func (o optionImpl) IsSome() bool {
	return !o.IsNone()
}

func (o optionImpl) Val() interface{} {
	return o.val
}

// IsSome in option
func IsSome(o Option) bool { return o.IsSome() }

// IsNone nothing in option
func IsNone(o Option) bool { return o.IsNone() }

/* common type asserter */
var (
	StrTypeAsserter    = func(string) {}
	IntTypeAsserter    = func(int) {}
	Int64TypeAsserter  = func(int64) {}
	Int32TypeAsserter  = func(int32) {}
	UintTypeAsserter   = func(uint) {}
	Uint64TypeAsserter = func(uint64) {}
	Uint32TypeAsserter = func(uint32) {}
	ErrTypeAsserter    = func(error) {}
)

func TypeAsserter(v interface{}) interface{} {
	typ := reflect.TypeOf(v)
	ft := reflect.FuncOf([]reflect.Type{typ}, []reflect.Type{}, false)
	return reflect.MakeFunc(ft, func([]reflect.Value) []reflect.Value { return nil }).Interface()
}
