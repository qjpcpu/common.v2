package fn

import (
	"errors"
	"testing"
)

func TestSimple(t *testing.T) {
	var sum, a int
	err := Do(func() int {
		return 1
	}).Then(func(i int) int {
		return i + 1
	}).Then(func(i int) int {
		sum = i + 1
		return 0
	}).Always(func() {
		a = 100
	}).Run()
	if err != nil {
		t.Fatal(err)
	}
	if sum != 3 {
		t.Fatal("bad waterfall,sum=", sum)
	}
	if a != 100 {
		t.Fatal("bad waterfall,a=", a)
	}
}

func TestNilAssign(t *testing.T) {
	type A struct {
		Name string
	}
	err := Do(func() *A {
		return nil
	}).Then(func(a *A) {
		if a != nil {
			t.Fatal("should nil")
		}
	}).Run()
	if err != nil {
		t.Fatal("should no err")
	}
}

func TestArguments(t *testing.T) {
	var res int
	err := Do(func(a int) int {
		return a + 1
	}).Then(func(b int) {
		res = b
	}).Run(100)
	if err != nil {
		t.Fatal("should no err")
	}
	if res != 101 {
		t.Fatal("should not set v")
	}
}

func TestErr(t *testing.T) {
	var a int
	err := Do(func() (int, error) {
		return 0, errors.New("t")
	}).Then(func(i int) {
		a = 100
	}).OnErr(func(e error) {
		a = 1
	}).Run()
	if err == nil {
		t.Fatal("should  err")
	}
	if a != 1 {
		t.Fatal("should not set v", a)
	}

}

func TestAbort(t *testing.T) {
	var sum, a int
	err := Do(func() int {
		return 1
	}).Then(func(i int, abort func()) int {
		abort()
		return i + 1
	}).Then(func(i int) int {
		sum = i + 1
		return 0
	}).Always(func() {
		a = 100
	}).Run()
	if err != nil {
		t.Fatal(err)
	}
	if sum != 0 {
		t.Fatal("bad waterfall,sum=", sum)
	}
	if a != 100 {
		t.Fatal("bad waterfall,a=", a)
	}
}

func TestAbort2(t *testing.T) {
	var sum, a int
	err := Do(func(abort func()) int {
		abort()
		return 1
	}).Then(func(i int, abort func()) int {
		panic("should come here")
	}).Then(func(i int, a func()) int {
		sum = i + 1
		return 0
	}).Always(func() {
		a = 100
	}).Run()
	if err != nil {
		t.Fatal(err)
	}
	if sum != 0 {
		t.Fatal("bad waterfall,sum=", sum)
	}
	if a != 100 {
		t.Fatal("bad waterfall,a=", a)
	}
}

func TestWhen(t *testing.T) {
	var sum, a int
	err := Do(func() int {
		return 1
	}).IfThenBy(func() bool { return false }, func(i int) int {
		return i + 1
	}).Then(func(i int) int {
		sum = i + 1
		return 0
	}).Always(func() {
		a = 100
	}).Run()
	if err != nil {
		t.Fatal(err)
	}
	if sum != 2 {
		t.Fatal("bad waterfall,sum=", sum)
	}
	if a != 100 {
		t.Fatal("bad waterfall,a=", a)
	}
}

func TestIfThen(t *testing.T) {
	var sum, a int
	err := Do(func() int {
		return 1
	}).IfThen(false, func(i int) int {
		return i + 1
	}).Then(func(i int) int {
		sum = i + 1
		return 0
	}).Always(func() {
		a = 100
	}).Run()
	if err != nil {
		t.Fatal(err)
	}
	if sum != 2 {
		t.Fatal("bad waterfall,sum=", sum)
	}
	if a != 100 {
		t.Fatal("bad waterfall,a=", a)
	}
}

func TestIfThen2(t *testing.T) {
	var sum, a int
	err := Do(func() int {
		return 1
	}).IfThen(true, func(i int) int {
		return i + 1
	}).Then(func(i int) int {
		sum = i + 1
		return 0
	}).Always(func() {
		a = 100
	}).Run()
	if err != nil {
		t.Fatal(err)
	}
	if sum != 0 {
		t.Fatal("bad waterfall,sum=", sum)
	}
	if a != 100 {
		t.Fatal("bad waterfall,a=", a)
	}
}
