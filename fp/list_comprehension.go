package fp

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
)

type _List struct {
	valList  reflect.Value
	elemType reflect.Type
}

// CreateList create list by constructor function
// constructor must be func() slice_type
func CreateList(constructor interface{}) *_List {
	fn := reflect.ValueOf(constructor)
	return ListOf(fn.Call(nil)[0].Interface())
}

// ListOf load slice as List, prefer CreateList for compiler check
func ListOf(slice interface{}) *_List {
	list := newList()
	if err := list.parseSlice(slice); err != nil {
		return list
	}
	return list.copy()
}

// Map convert List by function
// fn should be func(element_type) any_type or func(i int,element_type) any_type
// example: ListOf([]string{"a"}).Map(func(s string)int{return len(s)})
func (l *_List) Map(fn interface{}) *_List {
	outElemType, hasIndex, err := l.parseMapFunction(fn)
	if err != nil {
		panic(err)
	}
	size := l.valList.Len()
	out := reflect.MakeSlice(reflect.SliceOf(outElemType), size, size)
	fnVal := reflect.ValueOf(fn)
	for i := 0; i < size; i++ {
		if hasIndex {
			out.Index(i).Set(fnVal.Call([]reflect.Value{reflect.ValueOf(i), l.valList.Index(i)})[0])
		} else {
			out.Index(i).Set(fnVal.Call([]reflect.Value{l.valList.Index(i)})[0])
		}
	}
	l.valList = out
	l.elemType = outElemType
	return l
}

// FlatMap map and flatten
func (l *_List) FlatMap(fn interface{}) *_List {
	return l.Map(fn).Flatten()
}

// Filter element matches fn
// fn should be func(element_type) bool
// example: ListOf([]string{"a"}).Filter(func(s string)bool{return len(s)>0})
func (l *_List) Filter(fn interface{}) *_List {
	size := l.valList.Len()
	fnVal := reflect.ValueOf(fn)
	if err := l.parseFilterFunction(fn); err != nil {
		panic(err)
	}
	var delCnt int
	for i := 0; i < size; i++ {
		if !fnVal.Call([]reflect.Value{l.valList.Index(i)})[0].Bool() {
			delCnt++
		} else if delCnt > 0 {
			l.valList.Index(i - delCnt).Set(l.valList.Index(i))
		}
	}
	if delCnt > 0 {
		l.valList = l.valList.Slice(0, size-delCnt)
	}
	return l
}

// Reject element matches fn
// fn should be func(element_type) bool
// example: ListOf([]string{"a"}).Reject(func(s string)bool{return len(s)>0})
func (l *_List) Reject(fn interface{}) *_List {
	if err := l.parseFilterFunction(fn); err != nil {
		panic(err)
	}
	size := l.valList.Len()
	fnVal := reflect.ValueOf(fn)
	var delCnt int
	for i := 0; i < size; i++ {
		if fnVal.Call([]reflect.Value{l.valList.Index(i)})[0].Bool() {
			delCnt++
		} else if delCnt > 0 {
			l.valList.Index(i - delCnt).Set(l.valList.Index(i))
		}
	}
	if delCnt > 0 {
		l.valList = l.valList.Slice(0, size-delCnt)
	}
	return l
}

// Foreach iter for each element
// fn should be func(element_type) or func(index int,element_type)
// example: ListOf([]string{"a"}).Foreach(func(s string){})
// or ListOf([]string{"a"}).Foreach(func(i int,s string){})
func (l *_List) Foreach(fn interface{}) *_List {
	withIndex, err := l.parseForEachFunction(fn)
	if err != nil {
		panic(err)
	}
	fnVal := reflect.ValueOf(fn)
	for i := 0; i < l.valList.Len(); i++ {
		if withIndex {
			fnVal.Call([]reflect.Value{reflect.ValueOf(i), l.valList.Index(i)})
		} else {
			fnVal.Call([]reflect.Value{l.valList.Index(i)})
		}
	}
	return l
}

// Reduce with initval and reduce function
// fn should be func(memo_object,element_type) memo_object or func(memo_object,index int,element_type) memo_object
// example: ListOf([]string{"a"}).Reduce(0,func(count int,s string)int{return count+1})
// or ListOf([]string{"a"}).Reduce(0,func(count int,i int,s string)int{return count+1})
func (l *_List) Reduce(initval interface{}, fn interface{}) *ResultValue {
	withIndex, err := l.parseReduceFunction(fn)
	if err != nil {
		panic(err)
	}
	fnVal := reflect.ValueOf(fn)
	input := []reflect.Value{reflect.ValueOf(initval)}
	for i := 0; i < l.valList.Len(); i++ {
		if withIndex {
			input = append(input, reflect.ValueOf(i), l.valList.Index(i))
		} else {
			input = append(input, l.valList.Index(i))
		}
		input = fnVal.Call(input)
	}
	return createResult(input[0], nil)
}

// GroupBy group function
// fn should be func(element_type) any_type
// example: ListOf([]string{"a"}).GroupBy(func(s string)int{return len(s)})
func (l *_List) GroupBy(fn interface{}) *_KVList {
	groupIDType, err := l.parseGroupByFunction(fn)
	if err != nil {
		panic(err)
	}

	table := reflect.MakeMap(reflect.MapOf(groupIDType, reflect.SliceOf(l.elemType)))
	fnVal := reflect.ValueOf(fn)
	for i := 0; i < l.valList.Len(); i++ {
		v := l.valList.Index(i)
		key := fnVal.Call([]reflect.Value{v})[0]
		slice := table.MapIndex(key)
		if !slice.IsValid() {
			slice = reflect.MakeSlice(reflect.SliceOf(l.elemType), 0, l.valList.Len())
		}
		slice = reflect.Append(slice, v)
		table.SetMapIndex(key, slice)
	}
	return KVListOf(table.Interface())
}

// Partition list
func (l *_List) Partition(size int) *_List {
	nl := newList()
	listTyp := reflect.SliceOf(l.valList.Type())
	nl.elemType = listTyp
	list := reflect.MakeSlice(listTyp, 0, getGroupCount(l.valList.Len(), size))
	llen := l.valList.Len()
	if size <= 0 {
		list = reflect.Append(list, l.valList)
		nl.valList = list
		return nl
	}
	for start := 0; start < llen; start += size {
		end := start + size
		if end > llen {
			end = llen
		}
		list = reflect.Append(list, l.valList.Slice(start, end))
	}
	nl.valList = list
	return nl
}

// Size of list
func (l *_List) Size() int {
	return l.valList.Len()
}

// Uniq list
func (l *_List) Uniq() *_List {
	size := l.valList.Len()
	dupMap := make(map[interface{}]struct{})
	var delCnt int
	for i := 0; i < size; i++ {
		v := l.valList.Index(i).Interface()
		if _, ok := dupMap[v]; ok {
			delCnt++
		} else if delCnt > 0 {
			l.valList.Index(i - delCnt).Set(l.valList.Index(i))
		}
		dupMap[v] = struct{}{}
	}
	if delCnt > 0 {
		l.valList = l.valList.Slice(0, size-delCnt)
	}
	return l
}

// UniqBy key
// fn should be func(element_type) any_type
// example: ListOf([]string{"a"}).UniqBy(func(s string)string{return s})
func (l *_List) UniqBy(fn interface{}) *_List {
	l.uniqBy(fn)
	return l
}

// AsSetBy of key
// fn should be func(element_type) any_type
// example: ListOf([]string{"a"}).AsSet(func(s string)string{return s})
func (l *_List) AsSetBy(fn interface{}) *_KVList {
	set := l.uniqBy(fn)
	return KVListOf(set.smap.Interface())
}

// Append another list
func (l *_List) Append(l2 *_List) *_List {
	l.valList = reflect.AppendSlice(l.valList, l2.valList)
	return l
}

// Sub l2
func (l *_List) Sub(l2 *_List) *_List {
	s2 := l2.AsSet()
	fn := reflect.MakeFunc(l.filterFuncType(), func(in []reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(s2.Contains(in[0].Interface()))}
	})
	return l.Reject(fn.Interface())
}

// Intersect
func (l *_List) Intersect(l2 *_List) *_List {
	ft := reflect.FuncOf([]reflect.Type{l.elemType, l.elemType}, []reflect.Type{boolType}, false)
	cmpfn := l.compareFunc()
	fn := reflect.MakeFunc(ft, func(in []reflect.Value) []reflect.Value {
		ret := cmpfn(in[0], in[1]) <= 0
		return []reflect.Value{reflect.ValueOf(ret)}
	})
	l1 := l.SortBy(fn.Interface())
	l2 = l2.SortBy(fn.Interface())
	nl := newList()
	nl.elemType = l.elemType
	nl.valList = reflect.MakeSlice(
		l1.valList.Type(),
		0,
		min(l.valList.Len(), l2.valList.Len()),
	)

	for i, j := 0, 0; i < l1.valList.Len() && j < l2.valList.Len(); {
		if v := cmpfn(l1.valList.Index(i), l2.valList.Index(j)); v == 0 {
			nl.valList = reflect.Append(nl.valList, l1.valList.Index(i))
			i++
			j++
		} else if v == 1 {
			j++
		} else {
			i++
		}
	}
	return nl
}

// AsSet element
func (l *_List) AsSet() *_KVList {
	fnTyp := reflect.FuncOf([]reflect.Type{l.elemType}, []reflect.Type{l.elemType}, false)
	fn := reflect.MakeFunc(fnTyp, func(in []reflect.Value) []reflect.Value {
		return in
	})
	return l.AsSetBy(fn.Interface())
}

// Flatten [][]list
func (l *_List) Flatten() *_List {
	var total int
	size := l.valList.Len()
	for i := 0; i < size; i++ {
		total += l.valList.Index(i).Len()
	}
	arr := reflect.MakeSlice(l.elemType, 0, total)
	for i := 0; i < size; i++ {
		v := l.valList.Index(i)
		arr = reflect.AppendSlice(arr, v)
	}
	nl := newList()
	nl.valList = arr
	nl.elemType = l.elemType.Elem()
	return nl
}

// AppendElement to list
func (l *_List) AppendElement(ele interface{}) *_List {
	l.valList = reflect.Append(l.valList, reflect.ValueOf(ele))
	return l
}

// Sort list
func (l *_List) Sort() *_List {
	fnTyp := reflect.FuncOf([]reflect.Type{l.elemType, l.elemType}, []reflect.Type{boolType}, false)
	cmpfn := l.compareFunc()
	fn := reflect.MakeFunc(fnTyp, func(in []reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(cmpfn(in[0], in[1]) <= 0)}
	})
	return l.SortBy(fn.Interface())
}

// SortBy list fn is func(i,j element_type) bool
func (l *_List) SortBy(fn interface{}) *_List {
	if err := l.parseSortFunction(fn); err != nil {
		panic(err)
	}
	fnVal := reflect.ValueOf(fn)
	sort.SliceStable(l.valList.Interface(), func(i, j int) bool {
		return fnVal.Call([]reflect.Value{l.valList.Index(i), l.valList.Index(j)})[0].Bool()
	})
	return l
}

// Pick element by index
func (l *_List) Pick(idx int) *ResultValue {
	if idx >= 0 && idx < l.valList.Len() {
		return createResult(l.valList.Index(idx), nil)
	}
	return createResult(reflect.Zero(l.elemType), errors.New(`fp: slice index out of range`))
}

// Take N elements
func (l *_List) Take(n int) *_List {
	if n < 0 {
		n = 0
	}
	l.valList = l.valList.Slice(0, n)
	return l
}

// Reverse list
func (l *_List) Reverse() *_List {
	size := l.valList.Len()
	arr := reflect.MakeSlice(l.valList.Type(), size, size)
	for i := 0; i < size; i++ {
		arr.Index(i).Set(l.valList.Index(size - i - 1))
	}
	l.valList = arr
	return l
}

// First element
func (l *_List) First() *ResultValue {
	return l.Pick(0)
}

// Last element
func (l *_List) Last() *ResultValue {
	return l.Pick(l.valList.Len() - 1)
}

// OptionValue map Option list to its values, the list must be Option list
func (l *_List) OptionValue(typeAsserter interface{}) *_List {
	valTyp := reflect.TypeOf(typeAsserter).In(0)
	ft := reflect.FuncOf([]reflect.Type{optionType}, []reflect.Type{valTyp}, false)
	fv := reflect.MakeFunc(ft, func(in []reflect.Value) []reflect.Value {
		val := in[0].Interface().(Option).Val()
		return []reflect.Value{reflect.ValueOf(val).Convert(valTyp)}
	})
	return l.Filter(IsSome).Map(fv.Interface())
}

// Result of list
func (l *_List) Result(outPtr interface{}) error {
	return createResult(l.valList, nil).Result(outPtr)
}

// MustGetResult result
func (l *_List) MustGetResult() interface{} {
	return l.valList.Interface()
}

func (l *_List) Strings() (out []string) {
	l.Result(&out)
	return
}

func (l *_List) String() (out string) {
	l.Result(&out)
	return
}

func (l *_List) StringsList() (out [][]string) {
	l.Result(&out)
	return
}

func newList() *_List {
	return &_List{}
}

func (l *_List) copy() *_List {
	size := l.valList.Len()
	var out reflect.Value
	if l.valList.IsNil() {
		out = reflect.Zero(reflect.SliceOf(l.elemType))
	} else {
		out = reflect.MakeSlice(reflect.SliceOf(l.elemType), size, size)
		for i := 0; i < size; i++ {
			out.Index(i).Set(l.valList.Index(i))
		}
	}
	nl := newList()
	nl.valList = out
	nl.elemType = l.elemType
	return nl
}

func (l *_List) parseSlice(slice interface{}) error {
	tp := reflect.TypeOf(slice)
	if kind := tp.Kind(); kind != reflect.Slice && kind != reflect.Array {
		panic("argument must be slice")
	}
	tp = tp.Elem()
	v := reflect.ValueOf(slice)
	l.valList = v
	l.elemType = tp
	return nil
}

func (l *_List) parseMapFunction(fn interface{}) (outElemType reflect.Type, hasIndex bool, err error) {
	tp := reflect.TypeOf(fn)
	if tp.Kind() != reflect.Func {
		err = fmt.Errorf("%s should be function", tp)
		return
	}
	switch tp.NumIn() {
	case 1:
		if tp.In(0) != l.elemType {
			err = fmt.Errorf("input should be %v", l.elemType)
			return
		}
	case 2:
		if tp.In(0).Kind() != reflect.Int {
			err = fmt.Errorf("input should be %v", l.elemType)
			return
		}
		if tp.In(1) != l.elemType {
			err = fmt.Errorf("input should be %v", l.elemType)
			return
		}
		hasIndex = true
	default:
		err = errors.New("map function should have at least one input and one output")
		return
	}
	if tp.NumOut() != 1 {
		err = errors.New("map function should have at least one input and one output")
		return
	}
	outElemType = tp.Out(0)
	return
}

func (l *_List) parseFilterFunction(fn interface{}) (err error) {
	tp := reflect.TypeOf(fn)
	if tp.Kind() != reflect.Func {
		err = fmt.Errorf("%s should be function", tp)
		return
	}
	if tp.NumIn() != 1 || tp.NumOut() != 1 {
		err = errors.New("map function should have only one input and one output")
		return
	}
	if tp.In(0) != l.elemType {
		err = fmt.Errorf("input should be %v", l.elemType)
		return
	}
	if tp.Out(0).Kind() != reflect.Bool {
		err = errors.New("ouput should be bool")
		return
	}
	return
}

func (l *_List) parseForEachFunction(fn interface{}) (withIndex bool, err error) {
	tp := reflect.TypeOf(fn)
	if tp.Kind() != reflect.Func {
		err = fmt.Errorf("%s should be function", tp)
		return
	}
	if tp.NumIn() == 1 {
		if tp.In(0) != l.elemType {
			err = fmt.Errorf("foreach function argument shoule be %v", l.elemType)
			return
		}
	} else if tp.NumIn() == 2 {
		if tp.In(1) != l.elemType {
			err = fmt.Errorf("foreach function argument shoule be %v", l.elemType)
			return
		}
		if tp.In(0).Kind() != reflect.Int {
			err = fmt.Errorf("foreach function argument shoule be (int,%v)", l.elemType)
			return
		}
		withIndex = true
	} else {
		err = errors.New("foreach function should have at least one input")
		return
	}

	return
}

func (l *_List) parseSortFunction(fn interface{}) (err error) {
	tp := reflect.TypeOf(fn)
	if tp.Kind() != reflect.Func {
		err = fmt.Errorf("%s should be function", tp)
		return
	}
	if tp.NumOut() != 1 || tp.Out(0).Kind() != reflect.Bool {
		err = fmt.Errorf("sort function output shoule be boolean")
		return
	}
	if tp.NumIn() != 2 || tp.In(0) != l.elemType || tp.In(1) != l.elemType {
		err = fmt.Errorf("sort function argument shoule be %v", l.elemType)
		return
	}

	return
}

func (l *_List) parseUniqFunction(fn interface{}) (keyTyp reflect.Type) {
	tp := reflect.TypeOf(fn)
	if tp.Kind() != reflect.Func {
		panic(fmt.Sprintf("%s should be function", tp))
	}
	if tp.NumOut() != 1 {
		panic(fmt.Sprintf("uniq function should have one output"))
	}
	if tp.NumIn() != 1 || tp.In(0) != l.elemType {
		panic(fmt.Sprintf("uniq function argument shoule be %v", l.elemType))
	}

	return tp.Out(0)
}

func (l *_List) parseReduceFunction(fn interface{}) (withIndex bool, err error) {
	tp := reflect.TypeOf(fn)
	if tp.Kind() != reflect.Func {
		err = fmt.Errorf("%s should be function", tp)
		return
	}
	if tp.NumOut() != 1 {
		err = fmt.Errorf("reduce function should have 1 output")
		return
	}
	if tp.NumIn() == 2 {
		if tp.In(1) != l.elemType {
			err = fmt.Errorf("reduce function argument shoule be %v", l.elemType)
			return
		}
	} else if tp.NumIn() == 3 {
		if tp.In(2) != l.elemType {
			err = fmt.Errorf("reduce function argument shoule be %v", l.elemType)
			return
		}
		if tp.In(1).Kind() != reflect.Int {
			err = fmt.Errorf("reduce function argument shoule be (val,int,%v)", l.elemType)
			return
		}
		withIndex = true
	} else {
		err = errors.New("reduce function should be func(val,index,element) val")
		return
	}
	if tp.In(0) != tp.Out(0) {
		err = errors.New("reduce function should be func(val,index,element) val")
		return
	}

	return
}

func (l *_List) parseGroupByFunction(fn interface{}) (gtp reflect.Type, err error) {
	tp := reflect.TypeOf(fn)
	if tp.Kind() != reflect.Func {
		err = fmt.Errorf("%s should be function", tp)
		return
	}
	if tp.NumOut() != 1 || tp.NumIn() != 1 {
		err = fmt.Errorf("groupby function should have 1 output")
		return
	}
	if tp.In(0) != l.elemType {
		err = fmt.Errorf("groupby function input should be %v", l.elemType)
		return
	}
	gtp = tp.Out(0)
	return
}

func (l *_List) uniqBy(fn interface{}) *_Set {
	keyTyp := l.parseUniqFunction(fn)
	size := l.valList.Len()
	dupMap := makeSet(keyTyp, l.elemType)

	var delCnt int
	fnVal := reflect.ValueOf(fn)
	keyOfElem := func(i int) reflect.Value {
		return fnVal.Call([]reflect.Value{l.valList.Index(i)})[0]
	}
	for i := 0; i < size; i++ {
		key := keyOfElem(i)
		if dupMap.Add(key, l.valList.Index(i)) {
			l.valList.Index(i - delCnt).Set(l.valList.Index(i))
		} else {
			delCnt++
		}
	}
	if delCnt > 0 {
		l.valList = l.valList.Slice(0, size-delCnt)
	}
	return dupMap
}

func (l *_List) filterFuncType() reflect.Type {
	return reflect.FuncOf(
		[]reflect.Type{l.elemType},
		[]reflect.Type{boolType},
		false,
	)
}

func (l *_List) compareFunc() func(a, b reflect.Value) int {
	return func(a, b reflect.Value) int {
		switch l.elemType.Kind() {
		case reflect.String:
			if a.String() < b.String() {
				return -1
			} else if a.String() > b.String() {
				return 1
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if a.Int() < b.Int() {
				return -1
			} else if a.Int() > b.Int() {
				return 1
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if a.Uint() < b.Uint() {
				return -1
			} else if a.Uint() > b.Uint() {
				return 1
			}
		case reflect.Bool:
			if !a.Bool() && b.Bool() {
				return -1
			} else if a.Bool() && !b.Bool() {
				return 1
			}
		default:
			if !reflect.DeepEqual(a.Interface(), b.Interface()) {
				s1, s2 := fmt.Sprint(a.Interface()), fmt.Sprint(b.Interface())
				if s1 < s2 {
					return -1
				} else if s1 > s2 {
					return 1
				}
			}
		}
		return 0
	}
}

func getGroupCount(total, size int) int {
	if size >= total || size <= 0 {
		return 1
	}
	if total%size == 0 {
		return total / size
	}
	return total/size + 1
}

type ResultValue struct {
	v   reflect.Value
	err error
}

func createResult(v reflect.Value, err error) *ResultValue {
	return &ResultValue{v: v, err: err}
}

func (rv *ResultValue) Result(dst interface{}) error {
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr {
		return errors.New(`fp: dst must be pointer`)
	}
	if rv.err != nil {
		val.Elem().Set(reflect.Zero(val.Elem().Type()))
		return rv.err
	}
	val.Elem().Set(rv.v)
	return nil
}

func (rv *ResultValue) MustGetResult() interface{} {
	if rv.err != nil {
		panic(rv.err)
	}
	return rv.v.Interface()
}

func (rv *ResultValue) String() (s string) {
	rv.Result(&s)
	return
}

func (rv *ResultValue) Strings() (s []string) {
	rv.Result(&s)
	return
}

func (rv *ResultValue) StringsList() (s [][]string) {
	rv.Result(&s)
	return
}
