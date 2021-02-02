package structs

import (
	"crypto/md5"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

// option for fill
type option struct {
	MaxLevel        int
	MaxSliceLen     int
	ForceSliceLen   map[string]*int
	MaxMapLen       int
	PathToValueFunc PathToValueFunc
}

// OptionFunc option
type OptionFunc func(*option)

// PathToValueFunc path to value
type PathToValueFunc func(string, reflect.Type) (interface{}, bool)

// MapKey of map mark
const MapKey = "$$KEY"

// FillStruct fill struct, obj must be pointer
func FillStruct(obj interface{}, optF ...OptionFunc) (err error) {
	stype := reflect.TypeOf(obj)
	if stype.Kind() != reflect.Ptr {
		return errors.New("should be pointer")
	}
	if reflect.ValueOf(obj).Elem().Type().Kind() == reflect.Ptr {
		return errors.New("should not pass pointer of pointer")
	}
	opt := defaultOpt()
	for _, fn := range optF {
		fn(&opt)
	}
	f := &filler{option: &opt, passed: make(map[string]bool)}
	f.initializeVal([]string{}, stype.Elem(), reflect.ValueOf(obj).Elem(), opt.MaxLevel)
	return
}

// SetMaxLevel dfs depth
func SetMaxLevel(lvl int) OptionFunc {
	return func(opt *option) {
		opt.MaxLevel = lvl
	}
}

// SetMaxSliceLen slice size
func SetMaxSliceLen(size int) OptionFunc {
	return func(opt *option) {
		opt.MaxSliceLen = size
	}
}

// SetMaxMapLen map size
func SetMaxMapLen(size int) OptionFunc {
	return func(opt *option) {
		opt.MaxMapLen = size
	}
}

// SetPathToValueFunc customize path to value function
func SetPathToValueFunc(fn PathToValueFunc) OptionFunc {
	return func(opt *option) {
		opt.PathToValueFunc = fn
	}
}

// SetSliceLen under certain path
func SetSliceLen(path string, size int) OptionFunc {
	return func(opt *option) {
		if size >= 0 {
			opt.ForceSliceLen[path] = &size
		}
	}
}

// InsertPathToValueFunc to the first
func InsertPathToValueFunc(fnList ...PathToValueFunc) OptionFunc {
	fn := mergeFuncs(fnList...)
	return func(opt *option) {
		if old := opt.PathToValueFunc; old != nil {
			opt.PathToValueFunc = func(p string, t reflect.Type) (interface{}, bool) {
				v, ok := fn(p, t)
				if !ok {
					return old(p, t)
				}
				return v, ok
			}
		} else {
			opt.PathToValueFunc = fn
		}
	}
}

// AppendPathToValueFunc to the tail
func AppendPathToValueFunc(fnList ...PathToValueFunc) OptionFunc {
	fn := mergeFuncs(fnList...)
	return func(opt *option) {
		if old := opt.PathToValueFunc; old != nil {
			opt.PathToValueFunc = func(p string, t reflect.Type) (interface{}, bool) {
				v, ok := old(p, t)
				if !ok {
					return fn(p, t)
				}
				return v, ok
			}
		} else {
			opt.PathToValueFunc = fn
		}
	}
}

// WithSysPVFunc disclose
func WithSysPVFunc(fn PathToValueFunc) OptionFunc {
	return SetPathToValueFunc(func(p string, t reflect.Type) (interface{}, bool) {
		v, ok := fn(p, t)
		if !ok {
			return defaultPathToValueFunc(p, t)
		}
		return v, ok
	})
}

// SplitFieldAndIndex a step like array[1] to (array,1)
func SplitFieldAndIndex(step string) (field string, idx int) {
	field, idx = step, -1
	data := []byte(step)
	size := len(data)
	if size < 4 || data[size-1] != ']' || data[size-2] < '0' || data[size-2] > '9' {
		return
	}
	// split fail if i==0
	for i := size - 3; i > 0; i-- {
		if data[i] >= '0' && data[i] <= '9' {
			// continue
		} else if data[i] == '[' {
			// filt field[00]
			if data[i+1] == '0' && data[i+2] == '0' {
				break
			}
			i64, err := strconv.ParseInt(string(data[i+1:size-1]), 10, 64)
			if err != nil {
				break
			}
			field = string(data[:i])
			idx = int(i64)
			break
		} else {
			break
		}
	}
	return
}

// TrimFieldIndexSuffix trim step[0] to step
func TrimFieldIndexSuffix(step string) string {
	f, _ := SplitFieldAndIndex(step)
	return f
}

func (fl *filler) initializeStruct(steps []string, t reflect.Type, v reflect.Value, level int) {
	if level <= 0 {
		return
	}
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		ft := t.Field(i)
		if !isExported(ft.Name) {
			continue
		}
		if level-1 < 0 {
			break
		}
		var offset int
		if ft.Anonymous {
			offset = 1
		}
		if ft.Type.Kind() == reflect.Ptr {
			vv, ok := fl.getPathValue(appendStep(steps, ft.Name), ft.Type)
			if ok {
				if vv != nilValue {
					fv := reflect.New(ft.Type.Elem())
					setContainerValue(fv.Elem(), vv)
					f.Set(fv)
				}
			} else {
				fv := reflect.New(ft.Type.Elem())
				fl.initializeVal(appendStep(steps, ft.Name), ft.Type.Elem(), fv.Elem(), level-1+offset)
				f.Set(fv)
			}
		} else {
			fl.initializeVal(appendStep(steps, ft.Name), ft.Type, f, level-1+offset)
		}
	}
}

func (fl *filler) initializeSlice(steps []string, t reflect.Type, elemt reflect.Type, level int) reflect.Value {
	size := fl.MaxSliceLen
	if l, ok := fl.ForceSliceLen[buildPath(steps)]; ok {
		size = *l
	}
	slicev := reflect.MakeSlice(t, size, size)
	if level < 0 {
		return slicev
	}
	if elemt.Kind() == reflect.Ptr {
		for i := 0; i < size; i++ {
			vv, ok := fl.getPathValue(appendStep(steps, "[", intToString(i), "]"), elemt)
			if ok {
				if vv != nilValue {
					ele := reflect.New(elemt.Elem())
					setContainerValue(ele.Elem(), vv)
					slicev.Index(i).Set(ele)
				}
			} else {
				ele := reflect.New(elemt.Elem())
				fl.initializeVal(appendStep(steps, "[", intToString(i), "]"), ele.Elem().Type(), ele.Elem(), level)
				slicev.Index(i).Set(ele)
			}
		}
	} else {
		for i := 0; i < size; i++ {
			ele := reflect.New(elemt)
			fl.initializeVal(appendStep(steps, "[", intToString(i), "]"), elemt, ele.Elem(), level)
			slicev.Index(i).Set(ele.Elem())
		}
	}
	return slicev
}

func (fl *filler) initializeArray(steps []string, elemt reflect.Type, arrayv reflect.Value, level int) reflect.Value {
	size := arrayv.Len()
	if level < 0 {
		return arrayv
	}
	if elemt.Kind() == reflect.Ptr {
		for i := 0; i < size; i++ {
			vv, ok := fl.getPathValue(appendStep(steps, "[", intToString(i), "]"), elemt)
			if ok {
				if vv != nilValue {
					ele := reflect.New(elemt.Elem())
					setContainerValue(ele.Elem(), vv)
					arrayv.Index(i).Set(ele)
				}
			} else {
				ele := reflect.New(elemt.Elem())
				fl.initializeVal(appendStep(steps, "[", intToString(i), "]"), ele.Elem().Type(), ele.Elem(), level)
				arrayv.Index(i).Set(ele)
			}
		}
	} else {
		for i := 0; i < size; i++ {
			ele := reflect.New(elemt)
			fl.initializeVal(appendStep(steps, "[", intToString(i), "]"), elemt, ele.Elem(), level)
			arrayv.Index(i).Set(ele.Elem())
		}
	}
	return arrayv
}

func (fl *filler) initializeMap(steps []string, tk, tv reflect.Type, mapv reflect.Value, level int) {
	if level <= 0 {
		return
	}
	for i := 0; i < fl.MaxMapLen; i++ {
		//key
		var key, val reflect.Value
		if tk.Kind() == reflect.Ptr {
			kptr := reflect.New(tk.Elem())
			fl.initializeVal(appendStep(steps, MapKey), kptr.Elem().Type(), kptr.Elem(), level-1)
			key = kptr
		} else {
			kptr := reflect.New(tk)
			fl.initializeVal(appendStep(steps, MapKey), tk, kptr.Elem(), level-1)
			key = kptr.Elem()
		}
		// value
		if tv.Kind() == reflect.Ptr {
			vv, ok := fl.getPathValue(appendStep(steps, key.String()), tv)
			if ok {
				if vv != nilValue {
					vptr := reflect.New(tv.Elem())
					setContainerValue(vptr, vv)
					val = vptr
				}
			} else {
				vptr := reflect.New(tv.Elem())
				fl.initializeVal(appendStep(steps, key.String()), vptr.Elem().Type(), vptr.Elem(), level-1)
				val = vptr
			}
		} else {
			vptr := reflect.New(tv)
			fl.initializeVal(appendStep(steps, key.String()), tv, vptr.Elem(), level-1)
			val = vptr.Elem()
		}
		mapv.SetMapIndex(key, val)
	}
}

func (fl *filler) getPathValue(steps []string, tp reflect.Type) (reflect.Value, bool) {
	path := buildPath(steps)
	if fl.isVisited(path) {
		return nilValue, false
	}
	fl.visit(path)
	if fl.PathToValueFunc == nil || len(steps) == 0 {
		return reflect.Value{}, false
	}
	obj, ok := fl.PathToValueFunc(path, tp)
	if !ok {
		return reflect.Value{}, false
	}
	var v reflect.Value
	if IsRefType(tp) {
		if obj == nil || (IsRefType(reflect.ValueOf(obj).Type()) && reflect.ValueOf(obj).IsNil()) {
			v = nilValue
		} else {
			v = reflect.ValueOf(obj)
			if v.Kind() == reflect.Ptr && tp.Kind() != reflect.Interface {
				return v.Elem(), true
			}
		}
	} else {
		v = reflect.ValueOf(obj)
		if v.Kind() == reflect.Ptr {
			return v.Elem(), true
		}
	}
	return v, true
}

func (fl *filler) initializeVal(steps []string, t reflect.Type, v reflect.Value, level int) {
	if level < 0 {
		return
	}
	switch t.Kind() {
	case reflect.String:
		if vv, ok := fl.getPathValue(steps, t); ok {
			v.SetString(vv.String())
		} else {
			v.SetString(randomString())
		}
	case reflect.Bool:
		if vv, ok := fl.getPathValue(steps, t); ok {
			v.SetBool(vv.Bool())
		} else {
			b := rand.Intn(100)%2 == 0
			v.SetBool(b)
		}
	case reflect.Int:
		if vv, ok := fl.getPathValue(steps, t); ok {
			v.SetInt(vv.Int())
		} else {
			v.SetInt(rand.Int63n(10000))
		}
	case reflect.Int16:
		if vv, ok := fl.getPathValue(steps, t); ok {
			v.SetInt(vv.Int())
		} else {
			v.SetInt(rand.Int63n(16))
		}
	case reflect.Int32:
		if vv, ok := fl.getPathValue(steps, t); ok {
			v.SetInt(vv.Int())
		} else {
			v.SetInt(rand.Int63n(32))
		}
	case reflect.Int64:
		if vv, ok := fl.getPathValue(steps, t); ok {
			v.SetInt(vv.Int())
		} else {
			v.SetInt(rand.Int63n(1000))
		}
	case reflect.Int8:
		if vv, ok := fl.getPathValue(steps, t); ok {
			v.SetInt(vv.Int())
		} else {
			v.SetInt(rand.Int63n(8))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if vv, ok := fl.getPathValue(steps, t); ok {
			v.SetUint(vv.Uint())
		} else {
			v.SetUint(rand.Uint64() % 100)
		}
	case reflect.Float32, reflect.Float64:
		if vv, ok := fl.getPathValue(steps, t); ok {
			v.SetFloat(vv.Float())
		} else {
			v.SetFloat(rand.Float64())
		}
	case reflect.Struct:
		if vv, ok := fl.getPathValue(steps, t); ok {
			v.Set(vv)
		} else {
			if t.String() == "time.Time" {
				v.Set(reflect.ValueOf(time.Now()))
			} else {
				fl.initializeStruct(steps, t, v, level)
			}
		}
	case reflect.Ptr:
		vv, ok := fl.getPathValue(steps, t)
		if ok {
			if vv != nilValue {
				fv := reflect.New(t)
				fv.Set(vv)
				v.Set(fv)
			}
		} else {
			fv := reflect.New(t)
			fl.initializeVal(steps, t.Elem(), fv.Elem(), level)
			v.Set(fv)
		}
	case reflect.Map:
		hash := reflect.MakeMap(t)
		if vv, ok := fl.getPathValue(steps, t); ok {
			if vv != nilValue {
				v.Set(vv)
			}
		} else {
			fl.initializeMap(steps, t.Key(), t.Elem(), hash, level)
			v.Set(hash)
		}
	case reflect.Slice:
		if vv, ok := fl.getPathValue(steps, t); ok {
			if vv != nilValue {
				v.Set(vv)
			}
		} else {
			array := fl.initializeSlice(steps, t, v.Type().Elem(), level)
			v.Set(array)
		}
	case reflect.Array:
		if vv, ok := fl.getPathValue(steps, t); ok {
			if vv != nilValue {
				v.Set(vv)
			}
		} else {
			fl.initializeArray(steps, v.Type().Elem(), v, level)
		}
	case reflect.Chan:
		v.Set(reflect.MakeChan(t, 0))
	case reflect.Interface:
		if vv, ok := fl.getPathValue(steps, t); ok {
			if vv != nilValue {
				setContainerValue(v, vv)
			}
		} else {
			v.Set(reflect.ValueOf("DYNAMIC"))
		}
	}
}

// LastNodeOfPath last step node of path
func LastNodeOfPath(path string) string {
	steps := strings.Split(path, ".")
	return steps[len(steps)-1]
}

// REmail random email
func REmail() string {
	return randomString() + "@fixture.com"
}

// RMobile random mobile
func RMobile() string {
	data := md5.Sum([]byte(fmt.Sprintf("%v:%v:%v", time.Now(), time.Now().Nanosecond(), r.Float32())))
	m := make([]byte, 0, 11)
	m = append(m, '1')
	if r.Uint32()&1 != 0 {
		m = append(m, '3')
	} else {
		m = append(m, '5')
	}
	for i := 0; i < 9; i++ {
		m = append(m, data[i]%10+'0')
	}
	return string(m)

}

// RLink random link
func RLink() string {
	l := "http://www.fixture.com"
	size := r.Int()%4 + 1
	for i := 0; i < size; i++ {
		l += "/" + randomString()
	}
	return l
}

// RNumber random number
func RNumber(left, right int64) int64 {
	return left + r.Int63n(right-left)
}

// RTimestamp random unix timestamp
func RTimestamp() int64 {
	return time.Now().Unix()
}

// RString random string
func RString() string {
	return randomString()
}

// IsPtr is pointer
func IsPtr(tp reflect.Type) bool {
	return tp.Kind() == reflect.Ptr
}

// IsIntegerType is integer
func IsIntegerType(tp reflect.Type) bool {
	kind := tp.Kind()
	return reflect.Int <= kind && kind <= reflect.Int64
}

// IsBoolType is bool
func IsBoolType(tp reflect.Type) bool {
	return tp.Kind() == reflect.Bool
}

// IsBoolPtrType is *bool
func IsBoolPtrType(tp reflect.Type) bool {
	return IsPtr(tp) && IsBoolType(tp.Elem())
}

// IsIntegerPtrType is *integer
func IsIntegerPtrType(tp reflect.Type) bool {
	return IsPtr(tp) && IsIntegerType(tp.Elem())
}

// IsUnsignedIntegerType is unsign integer
func IsUnsignedIntegerType(tp reflect.Type) bool {
	kind := tp.Kind()
	return reflect.Uint <= kind && kind <= reflect.Uintptr
}

// IsUnsignedIntegerPtrType is unsigined *integer
func IsUnsignedIntegerPtrType(tp reflect.Type) bool {
	return IsPtr(tp) && IsUnsignedIntegerType(tp.Elem())
}

// IsFloatType is float
func IsFloatType(tp reflect.Type) bool {
	return tp.Kind() == reflect.Float32 || tp.Kind() == reflect.Float64
}

// IsFloatPtrType is *float
func IsFloatPtrType(tp reflect.Type) bool {
	return IsPtr(tp) && IsFloatType(tp.Elem())
}

// IsTimeType is time.Time
func IsTimeType(tp reflect.Type) bool {
	return tp == reflect.TypeOf(time.Time{})
}

// IsTimePtrType is *time.Time
func IsTimePtrType(tp reflect.Type) bool {
	return IsPtr(tp) && IsTimeType(tp.Elem())
}

// IsStringType is string
func IsStringType(tp reflect.Type) bool {
	return tp.Kind() == reflect.String
}

// IsStringPtrType is *string
func IsStringPtrType(tp reflect.Type) bool {
	return IsPtr(tp) && IsStringType(tp.Elem())
}

// IsPrimitiveType is simple types, bool,intx,uintx,floatx
func IsPrimitiveType(tp reflect.Type) bool {
	kind := tp.Kind()
	return (reflect.Bool <= kind && kind <= reflect.Float64) || kind == reflect.String
}

// IsPrimitivePtrType is simple types, *bool,*intx,*uintx,*floatx
func IsPrimitivePtrType(tp reflect.Type) bool {
	return IsPtr(tp) && IsPrimitiveType(tp.Elem())
}

// IsRefType is ref type
func IsRefType(tp reflect.Type) bool {
	kd := tp.Kind()
	return kd == reflect.Slice || kd == reflect.Map || kd == reflect.Ptr || kd == reflect.Interface || kd == reflect.Func || kd == reflect.Chan || kd == reflect.UnsafePointer
}

/*
 random helper
*/
var r = rand.New(rand.NewSource(time.Now().UnixNano()))
var idGen = newSimpleIDGenerator()

func defaultPathToValueFunc(path string, tp reflect.Type) (interface{}, bool) {
	if tp.Kind() == reflect.Ptr {
		tp = tp.Elem()
	}
	list := strings.Split(path, ".")
	finalNode := strings.ToLower(list[len(list)-1])
	finalNode, _ = SplitFieldAndIndex(finalNode)
	switch {
	case strings.Contains(finalNode, "email") && IsStringType(tp):
		return REmail(), true
	case (strings.Contains(finalNode, "link") || strings.Contains(finalNode, "url")) && IsStringType(tp):
		return RLink(), true
	case strings.Contains(finalNode, "id"):
		if IsIntegerType(tp) {
			return idGen.Next(), true
		} else if IsUnsignedIntegerType(tp) {
			iv := idGen.Next()
			switch tp.Kind() {
			case reflect.Uint:
				return uint(iv), true
			case reflect.Uint8:
				return uint8(iv), true
			case reflect.Uint16:
				return uint16(iv), true
			case reflect.Uint32:
				return uint32(iv), true
			case reflect.Uint64:
				return uint64(iv), true
			}
		} else if IsStringType(tp) {
			return strconv.FormatInt(idGen.Next(), 10), true
		}
	case finalNode == "status":
		if IsIntegerType(tp) {
			return 1, true
		} else if IsUnsignedIntegerType(tp) {
			return uint8(1), true
		}
	case strings.Contains(finalNode, "num") && IsIntegerType(tp):
		return RNumber(1, 1000), true
	case strings.Contains(finalNode, "phone") && IsStringType(tp):
		return RMobile(), true
	case strings.Contains(finalNode, "mobile") && IsStringType(tp):
		return RMobile(), true
	case strings.Contains(finalNode, "time"):
		if IsTimeType(tp) {
			return time.Now(), true
		} else if IsIntegerType(tp) {
			return RNumber(time.Now().AddDate(0, -1, 0).Unix(), time.Now().Unix()), true
		}
	}
	return "", false
}

func defaultOpt() option {
	return option{
		MaxLevel:        10,
		MaxMapLen:       2,
		MaxSliceLen:     3,
		PathToValueFunc: defaultPathToValueFunc,
		ForceSliceLen:   make(map[string]*int),
	}
}

type filler struct {
	*option
	passed map[string]bool
}

func (fl *filler) visit(path string) {
	fl.passed[path] = true
}

func (fl *filler) isVisited(path string) bool {
	return fl.passed[path]
}

func isExported(fieldName string) bool {
	return len(fieldName) > 0 && (fieldName[0] >= 'A' && fieldName[0] <= 'Z')
}

func appendStep(steps []string, stepArgs ...string) []string {
	var step strings.Builder
	for _, arg := range stepArgs {
		step.WriteString(arg)
	}
	return append(steps, step.String())
}

type nilStruct struct{}

var nilValue = reflect.ValueOf(nilStruct{})

// is like [number]
func isIndexToken(s string) bool {
	token := stringToBytes(s)
	if len(token) < 3 || token[0] != '[' || token[len(token)-1] != ']' {
		return false
	}
	// filt [00????]
	if token[1] == '0' && token[2] == '0' {
		return false
	}
	for i := 1; i < len(token)-1; i++ {
		if token[i] < '0' || token[i] > '9' {
			return false
		}
	}
	return true
}

// convert steps [f1,f2,[1],f3] to path .f1.f2[1].f3
func buildPath(steps []string) string {
	var b strings.Builder
	for i, s := range steps {
		if isIndexToken(s) && i > 0 {
			b.WriteString(s)
		} else {
			b.WriteString(".")
			b.WriteString(s)
		}
	}
	return b.String()
}

func isNotRootPath(path string) bool {
	return len(path) > 0
}

// in case of type conversion fail
func setContainerValue(c reflect.Value, v reflect.Value) {
	switch v.Kind() {
	case reflect.String:
		c.SetString(v.String())
	case reflect.Bool:
		c.SetBool(v.Bool())
	case reflect.Int:
		c.SetInt(v.Int())
	case reflect.Int16:
		c.SetInt(v.Int())
	case reflect.Int32:
		c.SetInt(v.Int())
	case reflect.Int64:
		c.SetInt(v.Int())
	case reflect.Int8:
		c.SetInt(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		c.SetUint(v.Uint())
	case reflect.Float32, reflect.Float64:
		c.SetFloat(v.Float())
	default:
		c.Set(v)
	}
}

func mergeFuncs(fn ...PathToValueFunc) PathToValueFunc {
	return func(p string, t reflect.Type) (interface{}, bool) {
		for _, f := range fn {
			v, ok := f(p, t)
			if ok {
				return v, ok
			}
		}
		return nil, false
	}
}

func randomString() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%v:%v:%v", time.Now(), time.Now().Nanosecond(), r.Float32()))))[:8]
}

type idgenerator interface {
	Next() int64
}

type simpleIDGenerator struct{ seed int64 }

func newSimpleIDGenerator() idgenerator {
	return &simpleIDGenerator{seed: time.Now().Unix()}
}

func (g *simpleIDGenerator) Next() int64 {
	return atomic.AddInt64(&g.seed, 1)
}

func stringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}

func intToString(i int) string {
	return strconv.Itoa(i)
}
