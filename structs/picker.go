package structs

import (
	"errors"
	"reflect"
	"strings"
	"time"
)

var (
	ErrValueNotExist = errors.New("value not exists")
)

// PathHitter return true if you want get the value of certain path
type PathHitter func(string) bool

// ValuePtr of real value
type ValuePtr = reflect.Value

// VisitCtx context
type VisitCtx struct {
	shouldSkipSiblings bool
	shouldSkipChildren bool
	shouldSkipAll      bool
}

// Visitor func
type Visitor func(ctx *VisitCtx, path string, tp reflect.Type, v ValuePtr)

type value struct {
	v   ValuePtr
	err error
}

// Values store pick result
type Values map[string]value

func (v Values) setError(path string, err error) {
	v[path] = value{err: err}
}

func (v Values) setVal(path string, val ValuePtr) {
	if _, ok := v[path]; !ok {
		v[path] = value{v: val}
	}
}

// Paths of results
func (v Values) Paths() []string {
	var list []string
	for k := range v {
		list = append(list, k)
	}
	return list
}

// Get value of the path
func (v Values) Get(path string) (ValuePtr, error) {
	val, ok := v[path]
	if !ok {
		return reflect.Value{}, ErrValueNotExist
	}
	return val.v, val.err
}

func (v Values) MustGet(path string) ValuePtr {
	vv, err := v.Get(path)
	if err != nil {
		panic(err)
	}
	return vv
}

func (v Values) MustGetInterface(path string) interface{} {
	vv, err := v.Get(path)
	if err != nil {
		panic(err)
	}
	return vv.Elem().Interface()
}

func (v Values) MustGetString(path string) string {
	vv, err := v.Get(path)
	if err != nil {
		panic(err)
	}
	return vv.Elem().String()
}

func (v Values) MustGetStringPtr(path string) *string {
	vv, err := v.Get(path)
	if err != nil {
		panic(err)
	}
	return vv.Elem().Interface().(*string)
}

func (v Values) MustGetInt64(path string) int64 {
	vv, err := v.Get(path)
	if err != nil {
		panic(err)
	}
	return vv.Elem().Int()
}

func (v Values) MustGetInt64Ptr(path string) *int64 {
	vv, err := v.Get(path)
	if err != nil {
		panic(err)
	}
	return vv.Elem().Interface().(*int64)
}

func (v Values) MustGetInt(path string) int {
	vv, err := v.Get(path)
	if err != nil {
		panic(err)
	}
	return vv.Elem().Interface().(int)
}

func (v Values) MustGetIntPtr(path string) *int {
	vv, err := v.Get(path)
	if err != nil {
		panic(err)
	}
	return vv.Elem().Interface().(*int)
}

func (v Values) MustGetUint64(path string) uint64 {
	vv, err := v.Get(path)
	if err != nil {
		panic(err)
	}
	return vv.Elem().Uint()
}

func (v Values) MustGetUint64Ptr(path string) *uint64 {
	vv, err := v.Get(path)
	if err != nil {
		panic(err)
	}
	return vv.Elem().Interface().(*uint64)
}

func (v Values) MustGetTime(path string) time.Time {
	vv, err := v.Get(path)
	if err != nil {
		panic(err)
	}
	return vv.Elem().Interface().(time.Time)
}

func (v Values) MustGetTimePtr(path string) *time.Time {
	vv, err := v.Get(path)
	if err != nil {
		panic(err)
	}
	return vv.Elem().Interface().(*time.Time)
}

func newValues() Values {
	return make(Values)
}

// PickValuesByLastNode pick by last field name
func PickValuesByLastNode(obj interface{}, fields ...string) Values {
	fieldsMap := make(map[string]bool)
	for _, f := range fields {
		fieldsMap[f] = true
	}
	fn := func(p string) bool {
		return fieldsMap[LastNodeOfPath(p)]
	}
	return PickValues(obj, fn)
}

// PickValuesByPath pick by full path
func PickValuesByPath(obj interface{}, paths ...string) Values {
	fieldsMap := make(map[string]bool)
	for _, f := range paths {
		fieldsMap[f] = true
	}
	fn := func(p string) bool {
		return fieldsMap[p]
	}
	return PickValues(obj, fn)
}

// PickValues pick by path function
func PickValues(obj interface{}, pathFn PathHitter) (vals Values) {
	vals = newValues()
	if obj == nil {
		return
	}
	v := reflect.ValueOf(obj)
	walkVal(newCtx(), []string{}, v.Type(), v, visitOnce(pathHitterToVisitor(pathFn, vals)))
	return
}

// Walk object
func Walk(obj interface{}, visitFn Visitor) {
	if obj == nil {
		return
	}
	/* make root so we can change root fields */
	v := reflect.ValueOf(obj)
	root := reflect.MakeSlice(reflect.SliceOf(v.Type()), 1, 1)
	root.Index(0).Set(v)

	walkVal(newCtx(), []string{}, root.Type(), root, trimRoot(visitOnce(visitFn), true))
}

// WalkLeaf call visitFn only when primitive tyeps
func WalkLeaf(obj interface{}, visitFn Visitor) {
	fn := func(ctx *VisitCtx, path string, tp reflect.Type, v ValuePtr) {
		if IsPrimitiveType(tp) || IsPrimitivePtrType(tp) || IsTimePtrType(tp) || IsTimeType(tp) {
			visitFn(ctx, path, tp, v)
		}
	}
	Walk(obj, fn)
}

func visitOnce(visit Visitor) Visitor {
	onceMap := make(map[string]bool)
	return func(ctx *VisitCtx, path string, tp reflect.Type, v ValuePtr) {
		if _, ok := onceMap[path]; ok {
			return
		}
		onceMap[path] = true
		visit(ctx, path, tp, v)
	}
}

func trimRoot(visit Visitor, trim bool) Visitor {
	return func(ctx *VisitCtx, path string, tp reflect.Type, v ValuePtr) {
		path = strings.TrimPrefix(path, rootPrefix)
		visit(ctx, path, tp, v)
	}
}

func pathHitterToVisitor(pathFn PathHitter, vals Values) Visitor {
	return func(ctx *VisitCtx, path string, tp reflect.Type, v ValuePtr) {
		if pathFn(path) {
			vals.setVal(path, v)
		}
	}
}

func walkVal(ctx *VisitCtx, steps []string, t reflect.Type, v reflect.Value, visit Visitor) {
	path := buildPath(steps)
	switch t.Kind() {
	case reflect.String, reflect.Bool, reflect.Int64, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64:
		if isNotRootPath(path) {
			visit(ctx, path, t, v.Addr())
			if ctx.shouldSkipAll {
				return
			}
		}
	case reflect.Struct:
		if isNotRootPath(path) {
			visit(ctx, path, t, v.Addr())
			if ctx.shouldSkipAll || ctx.shouldSkipChildren {
				ctx.shouldSkipChildren = false
				return
			}
		}
		walkStruct(ctx, steps, t, v, visit)
	case reflect.Ptr:
		if isNotRootPath(path) {
			visit(ctx, path, t, v.Addr())
			if ctx.shouldSkipAll || ctx.shouldSkipChildren {
				ctx.shouldSkipChildren = false
				return
			}
		}
		if !v.IsNil() {
			walkVal(ctx, steps, t.Elem(), v.Elem(), visit)
			if ctx.shouldSkipAll || ctx.shouldSkipChildren {
				ctx.shouldSkipChildren = false
				return
			}
		}
	case reflect.Map:
		if isNotRootPath(path) {
			visit(ctx, path, t, v.Addr())
			if ctx.shouldSkipAll || ctx.shouldSkipChildren {
				ctx.shouldSkipChildren = false
				return
			}
		}
		if !v.IsNil() {
			walkMap(ctx, steps, t.Key(), t.Elem(), v, visit)
			if ctx.shouldSkipAll || ctx.shouldSkipChildren {
				ctx.shouldSkipChildren = false
				return
			}

		}
	case reflect.Slice, reflect.Array:
		if isNotRootPath(path) {
			visit(ctx, path, t, v.Addr())
			if ctx.shouldSkipAll || ctx.shouldSkipChildren {
				ctx.shouldSkipChildren = false
				return
			}
		}
		if !v.IsNil() {
			walkSlice(ctx, steps, t.Elem(), v, visit)
			if ctx.shouldSkipAll || ctx.shouldSkipChildren {
				ctx.shouldSkipChildren = false
				return
			}
		}
	case reflect.Interface:
		if isNotRootPath(path) {
			visit(ctx, path, t, v.Addr())
			if ctx.shouldSkipAll || ctx.shouldSkipChildren {
				ctx.shouldSkipChildren = false
				return
			}
		}
		if !v.IsNil() {
			if v.Elem().Kind() == reflect.Ptr && !v.Elem().IsNil() {
				walkVal(ctx, steps, v.Elem().Elem().Type(), v.Elem().Elem(), visit)
				if ctx.shouldSkipAll || ctx.shouldSkipChildren {
					ctx.shouldSkipChildren = false
					return
				}
			} else if v.Elem().Kind() != reflect.Ptr {
				/* create reference for non ptr type so that i can modify */
				ref := reflect.New(v.Elem().Type())
				ref.Elem().Set(v.Elem())
				walkVal(ctx, steps, ref.Elem().Type(), ref.Elem(), visit)
				v.Set(ref.Elem())
				if ctx.shouldSkipAll || ctx.shouldSkipChildren {
					ctx.shouldSkipChildren = false
					return
				}
			}
		}
	}
}

func walkMap(ctx *VisitCtx, steps []string, kt, vt reflect.Type, v reflect.Value, fn Visitor) {
	keys := v.MapKeys()
	for _, key := range keys {
		vv := v.MapIndex(key)
		/* create addressable value */
		newVal := reflect.New(vt)
		newVal.Elem().Set(vv)

		walkVal(ctx, append(steps, key.String()), vv.Type(), newVal.Elem(), fn)
		if ctx.shouldSkipAll || ctx.shouldSkipSiblings {
			break
		}
	}
	ctx.shouldSkipSiblings = false
}

func walkStruct(ctx *VisitCtx, steps []string, t reflect.Type, v reflect.Value, fn Visitor) {
	for i := 0; i < v.NumField(); i++ {
		fv := v.Field(i)
		ft := t.Field(i)
		if !isExported(ft.Name) {
			continue
		}
		walkVal(ctx, append(steps, ft.Name), ft.Type, fv, fn)
		if ctx.shouldSkipAll || ctx.shouldSkipSiblings {
			break
		}
	}
	ctx.shouldSkipSiblings = false
}

func walkSlice(ctx *VisitCtx, steps []string, et reflect.Type, v reflect.Value, fn Visitor) {
	for i := 0; i < v.Len(); i++ {
		walkVal(ctx, appendStep(steps, "[", intToString(i), "]"), et, v.Index(i), fn)
		if ctx.shouldSkipAll || ctx.shouldSkipSiblings {
			break
		}
	}
	ctx.shouldSkipSiblings = false
}

const (
	rootPrefix = ".[0]"
)

func newCtx() *VisitCtx {
	return new(VisitCtx)
}
func (ctx *VisitCtx) Stop() {
	ctx.shouldSkipAll = true
}

func (ctx *VisitCtx) SkipSiblings() {
	ctx.shouldSkipSiblings = true
}

func (ctx *VisitCtx) SkipChildren() {
	ctx.shouldSkipChildren = true
}
