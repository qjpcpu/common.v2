package assert

import (
	"fmt"
	"reflect"

	fmt2 "github.com/qjpcpu/common.v2/fmt"
)

// ShouldBeNil would panic if err is not nil
func ShouldBeNil(err error, msgAndArgs ...interface{}) {
	if err == nil {
		return
	}
	v := reflect.ValueOf(err)
	if v.Kind() != reflect.Ptr || !v.IsNil() {
		printMsgArgs(msgAndArgs...)
		panic(fmt.Sprintf("[%v]%v", v.Type(), err))
	}
}

// ShouldBeTrue would panic if codition is false
func ShouldBeTrue(condition bool, msg ...interface{}) {
	if !condition {
		printMsgArgs(msg...)
		panic("should be true")
	}
}

// ShouldEqual would panic if not equal
func ShouldEqual(a, b interface{}, msg ...interface{}) {
	if !reflect.DeepEqual(a, b) {
		printMsgArgs(msg...)
		panic(fmt.Sprintf("%v != %v", a, b))
	}
}

// AllowPanic swallow panic
func AllowPanic(fn func()) (isPanicOccur bool) {
	defer func() {
		if r := recover(); r != nil {
			isPanicOccur = true
		}
	}()
	fn()
	return
}

// ShouldSuccessAtLeastOne excute functions one by one until success
func ShouldSuccessAtLeastOne(fnList ...func()) {
	for _, fn := range fnList {
		if !AllowPanic(fn) {
			return
		}
	}
	panic("all function failed")
}

func printMsgArgs(args ...interface{}) {
	switch len(args) {
	case 0:
	case 1:
		fmt2.Print("%+v\n", args[0])
	default:
		fmt2.Print(args[0].(string), args[1:]...)
	}
}
