package fp

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ListComprehensionTestSuite struct {
	suite.Suite
}

func (suite *ListComprehensionTestSuite) SetupTest() {
}

func TestListComprehensionTestSuite(t *testing.T) {
	suite.Run(t, new(ListComprehensionTestSuite))
}

type Item struct {
	Name *string
	Age  int
}

func stringPtr(s string) *string { return &s }

func (suite *ListComprehensionTestSuite) TestMap() {
	itemList := []*Item{
		{
			Name: stringPtr(`A`),
			Age:  10,
		},
		{
			Name: stringPtr(`B`),
			Age:  11,
		},
	}
	var out []string
	var ages []int
	err := ListOf(itemList).Map(func(item *Item) string {
		return *item.Name
	}).Result(&out)
	suite.Nil(err)
	suite.ElementsMatch([]string{`A`, `B`}, out)

	err = ListOf(itemList).Map(func(item *Item) int {
		return item.Age
	}).Result(&ages)
	suite.Nil(err)
	suite.ElementsMatch([]int{10, 11}, ages)
}

func (suite *ListComprehensionTestSuite) TestMapWithIndex() {
	itemList := []*Item{
		{
			Name: stringPtr(`A`),
			Age:  10,
		},
		{
			Name: stringPtr(`B`),
			Age:  11,
		},
	}
	var out []string
	var ages []int
	err := ListOf(itemList).Map(func(i int, item *Item) string {
		return *item.Name
	}).Result(&out)
	suite.Nil(err)
	suite.ElementsMatch([]string{`A`, `B`}, out)

	err = ListOf(itemList).Map(func(i int, item *Item) int {
		return item.Age
	}).Result(&ages)
	suite.Nil(err)
	suite.ElementsMatch([]int{10, 11}, ages)
}

func (suite *ListComprehensionTestSuite) TestGroupBy() {
	items := []*Item{
		{Name: stringPtr(`A`), Age: 10},
		{Name: stringPtr(`B`), Age: 11},
		{Name: stringPtr(`A`), Age: 11},
	}
	suite.ElementsMatch(
		[]string{"A", "B"},
		ListOf(items).GroupBy(func(v *Item) string {
			return *v.Name
		}).Keys().MustGetResult().([]string),
	)

	ListOf(items).GroupBy(func(v *Item) string {
		return *v.Name
	}).Foreach(func(key string, val []*Item) {
		if key == "A" {
			suite.ElementsMatch(
				[]int{10, 11},
				ListOf(val).Map(func(i *Item) int { return i.Age }).MustGetResult().([]int),
			)
		} else if key == "B" {
			suite.ElementsMatch(
				[]int{11},
				ListOf(val).Map(func(i *Item) int { return i.Age }).MustGetResult().([]int),
			)
		}
	})
}

func (suite *ListComprehensionTestSuite) TestSelect() {
	itemList := []*Item{
		{
			Name: stringPtr(`A`),
			Age:  10,
		},
		{
			Name: stringPtr(`B`),
			Age:  11,
		},
	}
	var out []int
	err := ListOf(itemList).Filter(func(item *Item) bool {
		return *item.Name == `B`
	}).Map(func(item *Item) int {
		return item.Age
	}).Result(&out)
	suite.Nil(err)
	suite.ElementsMatch([]int{11}, out)
}

func (suite *ListComprehensionTestSuite) TestReject() {
	itemList := []*Item{
		{
			Name: stringPtr(`A`),
			Age:  10,
		},
		{
			Name: stringPtr(`B`),
			Age:  11,
		},
	}
	var out []int
	err := ListOf(itemList).Reject(func(item *Item) bool {
		return *item.Name != `B`
	}).Map(func(item *Item) int {
		return item.Age
	}).Result(&out)
	suite.Nil(err)
	suite.ElementsMatch([]int{11}, out)
}

func (suite *ListComprehensionTestSuite) TestSimpleMap() {
	src := []int{1, 2, 3, 4}
	var out []string
	err := ListOf(src).Reject(func(item int) bool {
		return item > 1
	}).Map(func(item int) string {
		return fmt.Sprint(item)
	}).Result(&out)
	suite.Nil(err)
	suite.ElementsMatch([]string{"1"}, out)
}

func (suite *ListComprehensionTestSuite) TestForeach() {
	src := []int{1, 2, 3, 4}
	imap := make(map[int]int)
	ListOf(src).Foreach(func(v int) {
		imap[v] = v
	})
	suite.Len(imap, len(src))

	imap = make(map[int]int)
	ListOf(src).Foreach(func(i, v int) {
		imap[i] = v
	})
	suite.Len(imap, len(src))
}

func (suite *ListComprehensionTestSuite) TestReduce() {
	src := []string{"a", "a", "b", "c"}
	imap := make(map[string]int)
	err := ListOf(src).Reduce(make(map[string]int), func(memo map[string]int, ele string) map[string]int {
		memo[ele]++
		return memo
	}).Result(&imap)
	suite.Nil(err)

	suite.Equal(2, imap["a"])
	suite.Equal(1, imap["b"])
	suite.Equal(1, imap["c"])

	imap = make(map[string]int)
	err = ListOf(src).Reduce(map[string]int{"d": 100}, func(memo map[string]int, i int, ele string) map[string]int {
		memo[ele] = i
		return memo
	}).Result(&imap)
	suite.Nil(err)

	suite.Equal(1, imap["a"])
	suite.Equal(2, imap["b"])
	suite.Equal(3, imap["c"])
	suite.Equal(100, imap["d"])
}

func (suite *ListComprehensionTestSuite) TestSize() {
	src := []int{1, 2, 3, 4}
	suite.Equal(1, ListOf(src).Filter(func(v int) bool { return v > 3 }).Size())
	suite.Equal(4, ListOf(src).Size())
	suite.Equal([]int{1, 2, 3, 4}, src)
}

func (suite *ListComprehensionTestSuite) TestInplaceModify() {
	src := []int{1, 2, 3, 4}
	ListOf(src).Filter(func(v int) bool { return v == 2 }).Result(&src)
	suite.Equal([]int{2}, src)
}

func (suite *ListComprehensionTestSuite) TestUniq() {
	src := []int{1, 2, 3, 4, 1}
	var out []int
	ListOf(src).Uniq().Result(&out)
	suite.Equal([]int{1, 2, 3, 4}, out)

	itemList := []*Item{
		{
			Name: stringPtr(`A`),
			Age:  10,
		},
		{
			Name: stringPtr(`A`),
			Age:  10,
		},
	}
	suite.Equal(2, ListOf(itemList).Uniq().Size())

	a := stringPtr("A")
	itemList2 := []Item{
		{
			Name: a,
			Age:  10,
		},
		{
			Name: a,
			Age:  10,
		},
	}
	suite.Equal(1, ListOf(itemList2).Uniq().Size())

	itemList3 := []Item{
		{
			Name: stringPtr("A"),
			Age:  10,
		},
		{
			Name: stringPtr("A"),
			Age:  10,
		},
	}
	suite.Equal(2, ListOf(itemList3).Uniq().Size())

	itemList4 := []Item{
		{
			Name: stringPtr("A"),
			Age:  1,
		},
		{
			Name: stringPtr("A"),
			Age:  10,
		},
		{
			Name: stringPtr("B"),
			Age:  1,
		},
	}
	var outU []string
	ListOf(itemList4).
		UniqBy(func(t Item) string { return *t.Name }).
		Map(func(t Item) string { return *t.Name }).
		Result(&outU)
	suite.Equal([]string{"A", "B"}, outU)
}

func (suite *ListComprehensionTestSuite) TestSortBy() {
	src := []int{2, 3, 4, 1}
	var out []int
	ListOf(src).SortBy(func(i, j int) bool {
		return i < j
	}).Result(&out)
	suite.Equal([]int{1, 2, 3, 4}, out)

	src2 := []string{"b", "a", "d"}
	var out2 []string
	ListOf(src2).SortBy(func(i, j string) bool {
		return i < j
	}).Result(&out2)
	suite.Equal([]string{"a", "b", "d"}, out2)

	itemList := []*Item{
		{
			Name: stringPtr(`B`),
			Age:  20,
		},
		{
			Name: stringPtr(`A`),
			Age:  10,
		},
	}
	var outItems []*Item
	ListOf(itemList).SortBy(func(a, b *Item) bool {
		return a.Age < b.Age
	}).Result(&outItems)
	suite.Equal("A", *outItems[0].Name)
	suite.Equal("B", *outItems[1].Name)
}

func (suite *ListComprehensionTestSuite) TestSort() {
	src := []int{2, 3, 4, 1}
	var out []int
	ListOf(src).Sort().Result(&out)
	suite.Equal([]int{1, 2, 3, 4}, out)

	src2 := []string{"b", "a", "d"}
	var out2 []string
	ListOf(src2).Sort().Result(&out2)
	suite.Equal([]string{"a", "b", "d"}, out2)

	itemList := []*Item{
		{
			Name: stringPtr(`B`),
			Age:  20,
		},
		{
			Name: stringPtr(`A`),
			Age:  10,
		},
	}
	var outItems []*Item
	err := ListOf(itemList).Sort().Result(&outItems)
	suite.Nil(err)
}

func (suite *ListComprehensionTestSuite) TestAppend() {
	src := []int{1, 2, 3, 4}
	src2 := []int{3}
	ListOf(src).Append(ListOf(src2)).Result(&src)
	suite.Equal([]int{1, 2, 3, 4, 3}, src)
}

func (suite *ListComprehensionTestSuite) TestAppendElement() {
	src := []int{1, 2, 3, 4}
	ListOf(src).AppendElement(3).Result(&src)
	suite.Equal([]int{1, 2, 3, 4, 3}, src)
}

func (suite *ListComprehensionTestSuite) TestNilResult() {
	var src []int
	ListOf(src).Result(&src)
	var src2 []string
	suite.Nil(ListOf(src2).MustGetResult().([]string))

	suite.ElementsMatch([]string{"a"}, ListOf(src2).AppendElement("a").MustGetResult().([]string))
	suite.Nil(src2)
}

func (suite *ListComprehensionTestSuite) TestListSet() {
	items := []*Item{
		{Name: stringPtr(`A`), Age: 10},
		{Name: stringPtr(`A`), Age: 11},
		{Name: stringPtr(`B`), Age: 21},
	}
	suite.True(
		ListOf(items).AsSetBy(func(v *Item) string {
			return *v.Name
		}).Contains("A"),
	)

	m := make(map[string]*Item)
	ListOf(items).AsSetBy(func(v *Item) string {
		return *v.Name
	}).Result(&m)
	suite.Equal(2, len(m))
	suite.True(
		ListOf(items).AsSetBy(func(v *Item) string {
			return *v.Name
		}).Contains("A"),
	)
	suite.False(
		ListOf(items).AsSetBy(func(v *Item) string {
			return *v.Name
		}).Contains("C"),
	)
	suite.ElementsMatch(
		[]string{"A", "B"},
		ListOf(items).AsSetBy(func(v *Item) string {
			return *v.Name
		}).Keys().MustGetResult().([]string),
	)
}

func (suite *ListComprehensionTestSuite) TestAsSet() {
	set := ListOf([]string{"A", "C"}).AsSet()
	suite.True(set.Contains("A"))
	suite.True(set.Contains("C"))
	suite.False(set.Contains("E"))

	set = ListOf([]int8{1, 2, 3}).AsSet()
	suite.True(set.Contains(1))
	suite.False(set.Contains(5))
}

func (suite *ListComprehensionTestSuite) TestEqual() {
	set := ListOf([]string{"A", "C"}).Filter(Equal("A")).Strings()
	suite.ElementsMatch(set, []string{"A"})

	type obj struct{ N string }

	l := ListOf([]obj{
		{N: "1"},
		{N: "2"},
	}).Filter(Equal(obj{N: "2"}))
	suite.Equal(1, l.Size())
}

func (suite *ListComprehensionTestSuite) TestConstructor() {
	items := []*Item{
		{Name: stringPtr(`A`), Age: 10},
		{Name: stringPtr(`A`), Age: 11},
		{Name: stringPtr(`B`), Age: 21},
	}
	suite.Equal(
		3,
		CreateList(func() []*Item { return items }).Size(),
	)
}

func (suite *ListComprehensionTestSuite) TestPick() {
	items := []*Item{
		{Name: stringPtr(`A`), Age: 10},
		{Name: stringPtr(`A`), Age: 11},
		{Name: stringPtr(`B`), Age: 21},
	}

	first := ListOf(items).Map(func(v *Item) int { return v.Age }).First().MustGetResult().(int)
	suite.Equal(10, first)

	last := ListOf(items).Map(func(v *Item) int { return v.Age }).Last().MustGetResult().(int)
	suite.Equal(21, last)

	mid := ListOf(items).Map(func(v *Item) int { return v.Age }).Pick(1).MustGetResult().(int)
	suite.Equal(11, mid)

	items = nil
	var num int
	err := ListOf(items).Map(func(v *Item) int { return v.Age }).Pick(1).Result(&num)
	suite.Equal(0, num)
	suite.NotNil(err)

	e := &Item{}
	err = ListOf(items).First().Result(&e)
	suite.NotNil(err)
	suite.Nil(e)

	e = &Item{}
	items = []*Item{nil}
	err = ListOf(items).First().Result(&e)
	suite.Nil(err)
	suite.Nil(e)
}

func (suite *ListComprehensionTestSuite) TestFlatten() {
	words := []string{"hello", "world"}
	bytes := ListOf(words).Map(func(s string) []byte {
		return []byte(s)
	}).Flatten().MustGetResult().([]byte)
	out := string(bytes)
	suite.Equal("helloworld", out)
}

func (suite *ListComprehensionTestSuite) TestFlatMap() {
	words := []string{"hello", "world"}
	bytes := ListOf(words).FlatMap(func(s string) []byte {
		return []byte(s)
	}).MustGetResult().([]byte)
	out := string(bytes)
	suite.Equal("helloworld", out)
}

func (suite *ListComprehensionTestSuite) TestString() {
	src := []string{"a", "b"}
	src = ListOf(src).AppendElement("D").Strings()
	suite.Equal([]string{"a", "b", "D"}, src)
}

func (suite *ListComprehensionTestSuite) TestTake() {
	src := []string{"a", "b"}
	src = ListOf(src).Take(0).Strings()
	suite.Len(src, 0)

	src = []string{"a", "b"}
	src = ListOf(src).Take(1).Strings()
	suite.Len(src, 1)
}

func (suite *ListComprehensionTestSuite) TestReverse() {
	src := []string{"a", "b", "c"}
	src = ListOf(src).Reverse().Strings()
	suite.Equal(src[0], "c")
	suite.Equal(src[1], "b")
	suite.Equal(src[2], "a")
}

func (suite *ListComprehensionTestSuite) TestPartitionString() {
	src := []string{"a", "b", "c"}
	list := ListOf(src).Partition(1).MustGetResult().([][]string)
	suite.Equal([]string{"a"}, list[0])
	suite.Equal([]string{"b"}, list[1])
	suite.Equal([]string{"c"}, list[2])

	list = ListOf(src).Partition(2).MustGetResult().([][]string)
	suite.Equal([]string{"a", "b"}, list[0])
	suite.Equal([]string{"c"}, list[1])

	list = ListOf(src).Partition(200).MustGetResult().([][]string)
	suite.Equal([]string{"a", "b", "c"}, list[0])
}

func (suite *ListComprehensionTestSuite) TestInteract() {
	l1 := []string{"a", "b", "c", "b"}
	l2 := []string{"c", "b", "e", "b", "b"}
	l3 := ListOf(l1).Intersect(ListOf(l2)).Strings()
	suite.ElementsMatch([]string{"a", "b", "c", "b"}, l1)
	suite.ElementsMatch([]string{"c", "b", "e", "b", "b"}, l2)
	suite.ElementsMatch([]string{"b", "b", "c"}, l3)
}

func (suite *ListComprehensionTestSuite) TestSub() {
	l1 := []string{"a", "b", "c", "b"}
	l2 := []string{"c", "b"}
	l3 := ListOf(l1).Sub(ListOf(l2)).Strings()
	suite.ElementsMatch([]string{"a", "b", "c", "b"}, l1)
	suite.ElementsMatch([]string{"c", "b"}, l2)
	suite.ElementsMatch([]string{"a"}, l3)
}

func (suite *ListComprehensionTestSuite) TestOption() {
	l1 := []string{"a", "b", "c", "b"}
	l2 := ListOf(l1).Map(func(v string) Option {
		if v == "a" {
			return NewOption(1)
		}
		return None
	}).OptionValue(IntOptionFilter).MustGetResult().([]int)
	suite.ElementsMatch(l2, []int{1})
}

func (suite *ListComprehensionTestSuite) TestOptionEmptyList() {
	l1 := []string{"a", "b", "c", "b"}
	l2 := ListOf(l1).Map(func(v string) Option {
		return None
	}).OptionValue(Int64OptionFilter).MustGetResult().([]int64)
	suite.ElementsMatch(l2, []int64{})
}

func (suite *ListComprehensionTestSuite) TestOptionCustomType() {
	l1 := []string{"a", "b", "c", "b"}
	type st struct{ S string }
	l2 := ListOf(l1).Map(func(v string) Option {
		if v == "a" {
			return NewOption(st{S: v})
		}
		return None
	}).OptionValue(func(st) {}).MustGetResult().([]st)
	suite.Len(l2, 1)
	suite.Equal("a", l2[0].S)
}

func (suite *ListComprehensionTestSuite) TestOptionInterface() {
	l1 := []error{errors.New("test"), nil}
	l2 := ListOf(l1).Map(func(v error) Option {
		return NewOption(v)
	}).OptionValue(func(error) {}).MustGetResult().([]error)
	suite.Len(l2, 1)
	suite.Equal("test", l2[0].Error())
}

func (suite *ListComprehensionTestSuite) TestOptionCustomPtrType() {
	l1 := []string{"a", "b", "c", "b"}
	type st struct{ S string }
	l2 := ListOf(l1).Map(func(v string) Option {
		if v == "a" {
			return NewOption(&st{S: v})
		} else if v == "b" {
			var st1 *st
			return NewOption(st1)
		}
		return None
	}).OptionValue(func(*st) {}).MustGetResult().([]*st)
	suite.Len(l2, 1)
	suite.Equal("a", l2[0].S)
}
