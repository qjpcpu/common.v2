package fp

import "reflect"

// Equal return a function(v interface{}) bool{ return v == elem }
func Equal(elem interface{}) interface{} {
	typ := reflect.TypeOf(elem)
	ftyp := reflect.FuncOf([]reflect.Type{typ}, []reflect.Type{boolType}, false)
	return reflect.MakeFunc(ftyp, func(in []reflect.Value) []reflect.Value {
		eq := in[0].Interface() == elem
		return []reflect.Value{reflect.ValueOf(eq)}
	}).Interface()
}

// SelfString return self
func SelfString() func(string) string {
	return func(s string) string { return s }
}

// IsBlankStr str is blank
func IsBlankStr(s string) bool {
	return s == ""
}
