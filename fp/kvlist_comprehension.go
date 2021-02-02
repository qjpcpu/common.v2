package fp

import (
	"reflect"
)

type _KVList struct {
	mapVal           reflect.Value
	keyType, valType reflect.Type
}

// CreateKVList create kvlist by constructor
// fn must be func() some_map
func CreateKVList(fn interface{}) *_KVList {
	return KVListOf(reflect.ValueOf(fn).Call(nil)[0].Interface())
}

// KVListOf map, prefer CreateKVList for compiler check
func KVListOf(m interface{}) *_KVList {
	if reflect.TypeOf(m).Kind() != reflect.Map {
		panic("argument should be map")
	}
	obj := newKvList()
	tp := reflect.TypeOf(m)
	obj.mapVal = reflect.ValueOf(m)
	obj.keyType = tp.Key()
	obj.valType = tp.Elem()
	return obj
}

// Foreach element of object
// fn should be func(key_type,element_type)
func (obj *_KVList) Foreach(fn interface{}) *_KVList {
	fnVal := reflect.ValueOf(fn)
	iter := obj.mapVal.MapRange()
	for iter.Next() {
		fnVal.Call([]reflect.Value{iter.Key(), iter.Value()})
	}
	return obj
}

// Map k-v pair
// fn should be func(key_type,element_type) (any_type,any_type)
func (obj *_KVList) Map(fn interface{}) *_KVList {
	fnVal := reflect.ValueOf(fn)
	iter := obj.mapVal.MapRange()
	keyTp, valTp := obj.parseMapFunction(fn)
	table := reflect.MakeMap(reflect.MapOf(keyTp, valTp))
	for iter.Next() {
		out := fnVal.Call([]reflect.Value{iter.Key(), iter.Value()})
		table.SetMapIndex(out[0], out[1])
	}
	return KVListOf(table.Interface())
}

// MapValue fn should be func(optional[key_type],element_type) (any_type)
func (obj *_KVList) MapValue(fn interface{}) *_KVList {
	fnVal := reflect.ValueOf(fn)
	iter := obj.mapVal.MapRange()
	keyTp, valTp, fnVal := obj.parseMapValueFunction(fn)
	table := reflect.MakeMap(reflect.MapOf(keyTp, valTp))
	for iter.Next() {
		out := fnVal.Call([]reflect.Value{iter.Key(), iter.Value()})
		table.SetMapIndex(iter.Key(), out[0])
	}
	return KVListOf(table.Interface())
}

// MapKey fn should be func(key_type,optional[element_type]) (any_type)
func (obj *_KVList) MapKey(fn interface{}) *_KVList {
	iter := obj.mapVal.MapRange()
	keyTp, fnVal := obj.parseMapKeyFunction(fn)
	table := reflect.MakeMap(reflect.MapOf(keyTp, obj.valType))
	for iter.Next() {
		out := fnVal.Call([]reflect.Value{iter.Key(), iter.Value()})
		table.SetMapIndex(out[0], iter.Value())
	}
	return KVListOf(table.Interface())
}

// Filter kv pair
func (obj *_KVList) Filter(fn interface{}) *_KVList {
	fnVal := reflect.ValueOf(fn)
	obj.parseFilterFunction(fn)

	table := reflect.MakeMap(obj.mapVal.Type())
	iter := obj.mapVal.MapRange()
	for iter.Next() {
		k, v := iter.Key(), iter.Value()
		if ok := fnVal.Call([]reflect.Value{k, v})[0].Bool(); ok {
			table.SetMapIndex(k, v)
		}
	}
	return KVListOf(table.Interface())
}

// Reject kv pair
func (obj *_KVList) Reject(fn interface{}) *_KVList {
	fnVal := reflect.ValueOf(fn)
	obj.parseFilterFunction(fn)

	table := reflect.MakeMap(obj.mapVal.Type())
	iter := obj.mapVal.MapRange()
	for iter.Next() {
		k, v := iter.Key(), iter.Value()
		if ok := fnVal.Call([]reflect.Value{k, v})[0].Bool(); !ok {
			table.SetMapIndex(k, v)
		}
	}
	return KVListOf(table.Interface())
}

// Contains key
func (obj *_KVList) Contains(key interface{}) bool {
	kval := reflect.ValueOf(key)
	if kval.Type() != obj.keyType && kval.Type().ConvertibleTo(obj.keyType) {
		kval = kval.Convert(obj.keyType)
	}
	if ele := obj.mapVal.MapIndex(kval); !ele.IsValid() {
		return false
	}
	return true
}

// Keys of object
func (obj *_KVList) Keys() *_List {
	keys := obj.mapVal.MapKeys()
	slice := reflect.MakeSlice(reflect.SliceOf(obj.keyType), len(keys), len(keys))
	for i := 0; i < len(keys); i++ {
		slice.Index(i).Set(keys[i])
	}
	return ListOf(slice.Interface())
}

// Values of object
func (obj *_KVList) Values() *_List {
	keys := obj.mapVal.MapKeys()
	slice := reflect.MakeSlice(reflect.SliceOf(obj.valType), len(keys), len(keys))
	for i := 0; i < len(keys); i++ {
		slice.Index(i).Set(obj.mapVal.MapIndex(keys[i]))
	}
	return ListOf(slice.Interface())
}

// Result of list
func (l *_KVList) Result(outPtr interface{}) error {
	return createResult(l.mapVal, nil).Result(outPtr)
}

// MustGetResult result
func (l *_KVList) MustGetResult() interface{} {
	return l.mapVal.Interface()
}

// Size of map
func (obj *_KVList) Size() int {
	return obj.mapVal.Len()
}

func newKvList() *_KVList {
	return &_KVList{}
}

func (obj *_KVList) parseMapFunction(fn interface{}) (keytyp reflect.Type, valTyp reflect.Type) {
	tp := reflect.TypeOf(fn)
	if tp.Kind() != reflect.Func {
		panic("should be function")
	}
	if tp.NumIn() != 2 || tp.NumOut() != 2 {
		panic("map function should be 2 intput 2 output")
	}
	if tp.In(0) != obj.keyType || tp.In(1) != obj.valType {
		panic("map function input/output shoule match")
	}
	return tp.Out(0), tp.Out(1)
}

func (obj *_KVList) parseMapValueFunction(fn interface{}) (keytyp reflect.Type, valTyp reflect.Type, fnVal reflect.Value) {
	tp := reflect.TypeOf(fn)
	if tp.Kind() != reflect.Func {
		panic("should be function")
	}
	if tp.NumOut() != 1 {
		panic("map function should be 1 output")
	}
	if tp.NumIn() == 0 || tp.NumIn() > 2 {
		panic("map function should be at least 1 input")
	}
	switch tp.NumIn() {
	case 1:
		if tp.In(0) != obj.valType {
			panic("map function bad type")
		}
		ft := reflect.FuncOf([]reflect.Type{obj.keyType, obj.valType}, []reflect.Type{tp.Out(0)}, false)
		return obj.keyType, tp.Out(0), reflect.MakeFunc(ft, func(in []reflect.Value) []reflect.Value {
			return reflect.ValueOf(fn).Call(in[1:])
		})
	case 2:
		if tp.In(0) != obj.keyType || tp.In(1) != obj.valType {
			panic("map function bad type")
		}
		return tp.In(0), tp.Out(0), reflect.ValueOf(fn)
	default:
		panic("map function should be at least 1 input")
	}
}

func (obj *_KVList) parseMapKeyFunction(fn interface{}) (reflect.Type, reflect.Value) {
	tp := reflect.TypeOf(fn)
	if tp.Kind() != reflect.Func {
		panic("should be function")
	}
	if tp.NumOut() != 1 || tp.NumIn() == 0 || tp.In(0) != obj.keyType {
		panic("map function should be 1 output")
	}
	if tp.NumIn() == 2 && tp.In(1) != obj.valType {
		panic("map function bad signature")
	}
	fnval := reflect.ValueOf(fn)
	if tp.NumIn() == 1 {
		ft := reflect.FuncOf([]reflect.Type{obj.keyType, obj.valType}, []reflect.Type{tp.Out(0)}, false)
		fnval = reflect.MakeFunc(ft, func(in []reflect.Value) []reflect.Value {
			return reflect.ValueOf(fn).Call(in[:1])
		})
	}
	return tp.Out(0), fnval
}

func (obj *_KVList) parseFilterFunction(fn interface{}) {
	tp := reflect.TypeOf(fn)
	if tp.Kind() != reflect.Func {
		panic("should be function")
	}
	if tp.NumIn() != 2 || tp.NumOut() != 1 {
		panic("filter function should be 2 intput 2 output")
	}
	if tp.In(0) != obj.keyType || tp.In(1) != obj.valType {
		panic("filter function input/output shoule match")
	}
	if tp.Out(0).Kind() != reflect.Bool {
		panic("filter function output shoule be boolean")
	}
}
