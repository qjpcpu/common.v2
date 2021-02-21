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

/* common type filters */
var (
	StrOptionFilter    = func(string) {}
	IntOptionFilter    = func(int) {}
	Int64OptionFilter  = func(int64) {}
	Int32OptionFilter  = func(int32) {}
	UintOptionFilter   = func(uint) {}
	Uint64OptionFilter = func(uint64) {}
	Uint32OptionFilter = func(uint32) {}
	ErrOptionFilter    = func(error) {}
)
