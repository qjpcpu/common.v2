package structs

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

type Integer int
type A struct {
	Name   string
	B      *B
	M      map[string]*B
	MM     map[string]*int
	Nums   []*A
	Int    Integer
	Mobile string
	Tm     *time.Time
	ID     int
	Text   []*C
}

type B struct {
	Link   string
	URL    *string
	Phones []string
	Email  string
	C      *C
	Times  []*time.Time
}
type C struct {
	Link       string
	URL        *string
	MapInteger map[string]*Integer
}

func TestFillStruct(t *testing.T) {
	obj := &A{}
	now := time.Now()
	fn := func(path string, tp reflect.Type) (interface{}, bool) {
		switch path {
		case ".B.Link":
			return "http://www.github.com", true
		case ".B.Times[0]":
			return &now, true
		case ".B.C.Link":
			return "should drop this value", true
		case ".Int":
			return 1024, true
		case ".B.URL":
			return nil, true
		}
		return nil, false
	}
	if err := FillStruct(obj, SetMaxLevel(2), SetMaxMapLen(1), SetMaxSliceLen(1), WithSysPVFunc(fn)); err != nil {
		t.Fatal(err)
	}
	if obj.B.Times[0].UnixNano() != now.UnixNano() {
		t.Fatal("fill bad value")
	}
	if obj.B.Link != "http://www.github.com" {
		t.Fatal("fill bad value")
	}
	if obj.B.C.Link != "" {
		t.Fatal("fill bad value")
	}
	if int(obj.Int) != 1024 {
		t.Fatal("fill bad value")
	}
}

type node struct {
	Left  *node
	Right *node
	Val   int
}

func TestMaxSliceLen(t *testing.T) {
	b := &B{}
	FillStruct(b, SetMaxSliceLen(2), SetMaxMapLen(3))
	if len(b.Phones) != 2 || len(b.Times) != 2 || len(b.C.MapInteger) != 3 {
		t.Fatal("fill bad value")
	}
}

type MMap map[string]*MMap

func TestMaxLevel(t *testing.T) {
	root := &node{}
	lvl := 10
	FillStruct(root, SetMaxLevel(lvl), SetPathToValueFunc(func(p string, tp reflect.Type) (interface{}, bool) {
		if strings.HasSuffix(p, "Val") {
			return 5, true
		}
		return nil, false
	}))
	cur := root
	for i := 0; i < lvl; i++ {
		if i < lvl-1 {
			if cur.Val != 5 {
				t.Fatal("fill bad value")
			}
		}
		cur = cur.Left
	}
	if cur.Val != 0 {
		t.Fatal("fill bad value")
	}

	aa := &A{}
	FillStruct(aa, SetMaxLevel(1))
	if aa.B.Link != "" {
		t.Fatal("fill bad value")
	}
}

func TestFillNonStruct(t *testing.T) {
	var strs []string
	FillStruct(&strs)
	if len(strs) != 3 || strs[0] == "" {
		t.Fatal("fill bad value")
	}
	m := make(map[string]int)
	FillStruct(&m)
	if len(m) == 0 {
		t.Fatal("fill bad value")
	}

	type Address [5]byte
	var array Address
	if err := FillStruct(&array); err != nil {
		t.Fatal(err)
	}

	obj2 := struct {
		Addr  Address
		Addr2 *Address
	}{}
	if err := FillStruct(&obj2); err != nil {
		t.Fatal(err)
	}
}

func TestNil(t *testing.T) {
	a := struct {
		URL      *string
		Name     *string
		Nums     []*int
		Deep     *A
		FArr     [3]int
		Anything interface{}
	}{}
	FillStruct(&a, SetPathToValueFunc(func(p string, tt reflect.Type) (interface{}, bool) {
		if p == ".URL" {
			return nil, true
		}
		if p == ".Nums[1]" {
			return nil, true
		}
		if p == ".FArr" {
			return [3]int{0, 1, 2}, true
		}
		if p == ".Deep.B" {
			return nil, true
		}
		if p == ".Deep.M" {
			return nil, true
		}
		if p == ".Deep.Text" {
			return nil, true
		}
		str := "TEXT"
		if p == ".Anything" {
			return &str, true
		}
		return nil, false
	}))
	if a.URL != nil {
		t.Fatal("should be nil")
	}
	if a.Name == nil || *a.Name == "" {
		t.Fatal("should not be nil")
	}
	if a.Nums[1] != nil {
		t.Fatal("should be nil")
	}
	if a.Deep.B != nil {
		t.Fatal("should be nil")
	}
	if a.Deep.M != nil {
		t.Fatal("should be nil")
	}
	if a.Deep.Text != nil {
		t.Fatal("should be nil")
	}
	for i, val := range a.FArr {
		if i != val {
			t.Fatal("baa fill")
		}
	}
	if val, ok := a.Anything.(*string); !ok || *val != "TEXT" {
		t.Fatal("baa fill")
	}
}

func jsonObj(obj interface{}) string {
	data, _ := json.MarshalIndent(obj, "    ", "")
	return string(data)
}

type Chain struct {
	N    string
	Tail string
}

func TestFuncChain(t *testing.T) {
	f1 := func(p string, tp reflect.Type) (interface{}, bool) {
		if p == ".N" {
			return "func1", true
		}
		return nil, false
	}
	f2 := func(p string, tp reflect.Type) (interface{}, bool) {
		if p == ".N" {
			return "func2", true
		}
		return nil, false
	}
	f3 := func(p string, tp reflect.Type) (interface{}, bool) {
		if p == ".N" {
			return "func3", true
		}
		if p == ".Tail" {
			return "func3", true
		}
		return nil, false
	}
	c := Chain{}
	FillStruct(&c, SetPathToValueFunc(f1))
	if c.N != "func1" {
		t.Fatal("bad fill")
	}
	c = Chain{}
	FillStruct(&c, SetPathToValueFunc(f1), AppendPathToValueFunc(f3))
	if c.N != "func1" {
		t.Fatal("bad fill")
	}
	c = Chain{}
	FillStruct(&c, SetPathToValueFunc(f1), AppendPathToValueFunc(f3), InsertPathToValueFunc(f2))
	if c.N != "func2" {
		t.Fatal("bad fill")
	}
	if c.Tail != "func3" {
		t.Fatal("bad fill")
	}
	c = Chain{}
	FillStruct(&c, AppendPathToValueFunc(f2, f1, f3))
	if c.N != "func2" {
		t.Fatal("bad fill")
	}
	if c.Tail != "func3" {
		t.Fatal("bad fill")
	}
	c = Chain{}
	FillStruct(&c, InsertPathToValueFunc(f2, f1, f3))
	if c.N != "func2" {
		t.Fatal("bad fill")
	}
	if c.Tail != "func3" {
		t.Fatal("bad fill")
	}
}

func TestSplitFI(t *testing.T) {
	step := "jlkjlkj"
	f, i := SplitFieldAndIndex(step)
	if f != step || i != -1 {
		t.Fatal("split fail")
	}
	step = "element0]"
	f, i = SplitFieldAndIndex(step)
	if f != step || i != -1 {
		t.Fatal("split fail")
	}
	step = "element[0]"
	f, i = SplitFieldAndIndex(step)
	if f != "element" || i != 0 {
		t.Fatal("split fail")
	}
	step = "element[12]"
	f, i = SplitFieldAndIndex(step)
	if f != "element" || i != 12 {
		t.Fatal("split fail")
	}
	step = "element[00]"
	f, i = SplitFieldAndIndex(step)
	if f != step || i != -1 {
		t.Fatal("split fail")
	}
}

func TestFillSlice(t *testing.T) {
	type S struct {
		Slice  []*string
		FSlice []*string
	}
	s := &S{}
	FillStruct(s, SetMaxSliceLen(2), WithSysPVFunc(func(p string, tt reflect.Type) (interface{}, bool) {
		if p == ".FSlice" {
			str := "please go"
			return []*string{&str}, true
		}
		if p == ".Slice[1]" {
			return "please go2", true
		}
		return nil, false
	}))
	if len(s.Slice) != 2 {
		t.Fatal("bad fill")
	}
	if *s.Slice[1] != "please go2" {
		t.Fatal("bad fill")
	}
	if len(s.FSlice) != 1 {
		t.Fatal("bad fill")
	}
	if *s.FSlice[0] != "please go" {
		t.Fatal("bad fill")
	}
}

func TestBadParams(t *testing.T) {
	type BadArgs struct{}
	obj := &BadArgs{}
	err := FillStruct(&obj)
	if err == nil {
		t.Fatal("should has error")
	}
}

func TestForceSliceLen(t *testing.T) {
	type FSlice struct {
		Norm  []string
		Force []string
	}
	obj := &FSlice{}
	if err := FillStruct(obj, SetMaxSliceLen(3), SetSliceLen(".Force", 1)); err != nil {
		t.Fatal("bad fill")
	}
	if len(obj.Norm) != 3 {
		t.Fatal("bad fill")
	}
	if len(obj.Force) != 1 {
		t.Fatal("bad fill")
	}
}
