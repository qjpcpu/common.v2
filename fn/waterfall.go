/*
 Package waterfall Inspired by `https://caolan.github.io/async/v3/docs.html#waterfall`
*/
package fn

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
)

type abortFunction func()

// Function would be invoked in waterfall one by one
type Function interface{} // first parameter should be Context
// AlwaysFunction would be invoked at last
type AlwaysFunction func()

// ErrorFunction would be invoked when err occurs
type ErrorFunction func(error)

// ConditionFuntion execute when condition
type ConditionFuntion func() bool

// FunctionRunner execute functions one by one
type FunctionRunner interface {
	Then(Function) FunctionRunner
	// IfThenBy condition, the then function should have same input/output
	IfThenBy(ConditionFuntion, Function) FunctionRunner
	IfThen(bool, Function) FunctionRunner
	OnErr(ErrorFunction) FunctionRunner
	Always(AlwaysFunction) FunctionRunner
	Run(...interface{}) error
}

// Do create a FunctionRunner with first function f
func Do(f Function) FunctionRunner {
	return makeRunner().addFunc(f)
}

func DoIf(c bool, f Function) FunctionRunner {
	return makeRunner().IfThen(c, f)
}

func DoIfBy(c ConditionFuntion, f Function) FunctionRunner {
	return makeRunner().IfThenBy(c, f)
}

type pipeFunction func([]reflect.Value) []reflect.Value

type funcRunner struct {
	funcList        []Function
	outToInFuncList []pipeFunction
	alwaysFunc      AlwaysFunction
	errFunc         ErrorFunction
	isAborted       func() bool
	abortFn         reflect.Value
}

func (r *funcRunner) Then(f Function) FunctionRunner {
	return r.addFunc(f)
}

func (r *funcRunner) IfThen(c bool, f Function) FunctionRunner {
	return r.IfThenBy(func() bool { return c }, f)
}

func (r *funcRunner) IfThenBy(cf ConditionFuntion, f Function) FunctionRunner {
	typ := reflect.TypeOf(f)
	mustBeConditionThenFunction(typ)
	hasAbortFn := hasAbortFnInput(typ)
	if !hasAbortFn {
		typ = appendAbortFn(typ)
	}

	fn := reflect.MakeFunc(typ, func(in []reflect.Value) (out []reflect.Value) {
		if cf() {
			if hasAbortFn {
				out = reflect.ValueOf(f).Call(in)
			} else {
				out = reflect.ValueOf(f).Call(in[:len(in)-1])
			}
			// abort fn
			in[len(in)-1].Call(nil)
			return out
		}
		return in[:len(in)-1]
	})
	return r.addFunc(fn.Interface())
}

func (r *funcRunner) OnErr(f ErrorFunction) FunctionRunner {
	r.errFunc = f
	return r
}

func (r *funcRunner) Always(f AlwaysFunction) FunctionRunner {
	r.alwaysFunc = f
	return r
}

func (r *funcRunner) Run(args ...interface{}) (err error) {
	// check function list definition
	if err = r.checkFunctions(); err != nil {
		return
	}

	if err = r.genPipelineFunctions(); err != nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			err = panicAsErr(r)
		}
	}()

	// execute
	err = r.run(args...)

	if r.alwaysFunc != nil {
		f := reflect.ValueOf(r.alwaysFunc)
		f.Call(nil)
	}

	return
}

func makeRunner() *funcRunner {
	var abortFlag bool
	return &funcRunner{
		isAborted: func() bool {
			return abortFlag
		},
		abortFn: reflect.ValueOf(func() {
			abortFlag = true
		}),
	}
}

func (r *funcRunner) addFunc(f Function) *funcRunner {
	r.funcList = append(r.funcList, f)
	return r
}

func (r *funcRunner) pickErr(out []reflect.Value) (error, bool) {
	if len(out) > 0 && isErrType(out[len(out)-1].Type()) {
		if e := out[len(out)-1].Interface(); e == nil || e.(error) == nil {
			return nil, true
		} else {
			return e.(error), true
		}
	}
	return nil, false
}

func (r *funcRunner) run(args ...interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = panicAsErr(r)
		}
	}()

	var input []reflect.Value
	if len(args) > 0 {
		for i := range args {
			input = append(input, reflect.ValueOf(args[i]))
		}
	}
	if len(r.funcList) > 0 {
		if tp := reflect.TypeOf(r.funcList[0]); tp.NumIn() == len(input)+1 && isAbortFnType(tp.In(tp.NumIn()-1)) {
			input = append(input, r.abortFn)
		}
	}
	for i := range r.funcList {
		f := reflect.ValueOf(r.funcList[i])
		out := f.Call(input)
		if er, ok := r.pickErr(out); ok && er != nil {
			err = er
			break
		}
		if r.isAborted() {
			break
		}
		input = r.outToInFuncList[i](out)
	}
	if err != nil && r.errFunc != nil {
		f := reflect.ValueOf(r.errFunc)
		f.Call([]reflect.Value{reflect.ValueOf(err)})
	}

	return
}

func (r *funcRunner) genPipelineFunctions() (err error) {
	for i, f := range r.funcList {
		if i == len(r.funcList)-1 {
			r.outToInFuncList = append(r.outToInFuncList, func(out []reflect.Value) []reflect.Value { return nil })
		} else {
			if t, err := r.genTransFunc(f, r.funcList[i+1]); err != nil {
				return err
			} else {
				r.outToInFuncList = append(r.outToInFuncList, t)
			}
		}
	}
	return
}

func (r *funcRunner) checkFunctions() (err error) {
	for i, f := range r.funcList {
		if i == 0 {
			if err = checkDoFunction(f); err != nil {
				return err
			}
		} else {
			if err = checkThenFunction(f); err != nil {
				return err
			}
		}
	}
	return
}

func checkDoFunction(f Function) error {
	if err := checkThenFunction(f); err != nil {
		return err
	}
	return nil
}

func checkThenFunction(f Function) error {
	if f == nil {
		return errors.New("function should not be nil")
	}
	fn := reflect.TypeOf(f)
	if fn.Kind() != reflect.Func {
		return fmt.Errorf("%v should be function", fn.String())
	}
	return nil

}

func (f *funcRunner) genTransFunc(this, next Function) (pipeFunction, error) {
	var lastOutputIsErr, appendAbortFn bool
	thisfn := reflect.TypeOf(this)
	nextfn := reflect.TypeOf(next)
	var outArgs, inArgs []reflect.Type
	for i := 0; i < thisfn.NumOut(); i++ {
		outArgs = append(outArgs, thisfn.Out(i))
	}
	for i := 0; i < nextfn.NumIn(); i++ {
		inArgs = append(inArgs, nextfn.In(i))
	}
	if len(outArgs) > 0 && isErrType(outArgs[len(outArgs)-1]) {
		lastOutputIsErr = true
		outArgs = outArgs[:len(outArgs)-1]
	}

	if len(inArgs) == len(outArgs)+1 && isAbortFnType(inArgs[len(inArgs)-1]) {
		appendAbortFn = true
		inArgs = inArgs[:len(inArgs)-1]
	}
	if len(inArgs) != len(outArgs) {
		return nil, errors.New("function pipeline type miss match")
	}
	for i := 0; i < len(inArgs); i++ {
		if inArgs[i] != outArgs[i] || !outArgs[i].AssignableTo(inArgs[i]) {
			return nil, errors.New("function pipeline type miss match")
		}
	}
	return func(vals []reflect.Value) []reflect.Value {
		if lastOutputIsErr {
			vals = vals[:len(vals)-1]
		}
		if appendAbortFn {
			vals = append(vals, f.abortFn)
		}
		return vals
	}, nil
}

var _errTyp = reflect.TypeOf((*error)(nil)).Elem()

func isErrType(typ reflect.Type) bool {
	return typ.AssignableTo(_errTyp)
}

func panicAsErr(r interface{}) error {
	const size = 64 << 10
	const omitKeyword = `github.com/qjpcpu/common.v2`
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	return fmt.Errorf("Panic %s: %s", r, buf)
}

func isAbortFnType(tp reflect.Type) bool {
	return tp.Kind() == reflect.Func && tp.NumIn() == 0 && tp.NumOut() == 0
}

func mustHaveSameInputOutputType(fn reflect.Type) {
	if fn.NumIn() != fn.NumOut() {
		panic("function must have same input output arguments")
	}
	for i := 0; i < fn.NumIn(); i++ {
		if fn.In(i) != fn.Out(i) {
			panic("function must have same input output arguments")
		}
	}
}

func mustBeConditionThenFunction(typ reflect.Type) {
	if typ.NumIn() == typ.NumOut() {
		mustHaveSameInputOutputType(typ)
	} else if typ.NumIn() == typ.NumOut()+1 && isAbortFnType(typ.In(typ.NumIn()-1)) {
		for i := 0; i < typ.NumOut(); i++ {
			if typ.In(i) != typ.Out(i) {
				panic("bad when function")
			}
		}
	} else {
		panic("bad when function")
	}
}

func hasAbortFnInput(typ reflect.Type) bool {
	return typ.NumIn() == typ.NumOut()+1 && isAbortFnType(typ.In(typ.NumIn()-1))
}

func appendAbortFn(typ reflect.Type) reflect.Type {
	abTyp := abortFunction(func() {})
	inTyps := make([]reflect.Type, typ.NumIn()+1)
	outTyps := make([]reflect.Type, typ.NumOut())
	for i := 0; i < typ.NumIn(); i++ {
		inTyps[i] = typ.In(i)
	}
	for i := 0; i < typ.NumOut(); i++ {
		outTyps[i] = typ.Out(i)
	}
	inTyps[len(inTyps)-1] = reflect.TypeOf(abTyp)
	return reflect.FuncOf(inTyps, outTyps, false)
}
