package fp

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ObjectComprehensionTestSuite struct {
	suite.Suite
}

func (suite *ObjectComprehensionTestSuite) SetupTest() {
}

func TestObjectComprehensionTestSuite(t *testing.T) {
	suite.Run(t, new(ObjectComprehensionTestSuite))
}

func (suite *ObjectComprehensionTestSuite) TestForeach() {
	m := map[string]int{
		"a": 1,
		"b": 2,
	}
	var keys []string
	var vals []int
	KVListOf(m).Foreach(func(key string, val int) {
		keys = append(keys, key)
		vals = append(vals, val)
	})
	suite.ElementsMatch([]string{"a", "b"}, keys)
	suite.ElementsMatch([]int{1, 2}, vals)
}

func (suite *ObjectComprehensionTestSuite) TestKeys() {
	m := map[string]int{
		"a": 1,
		"b": 2,
	}
	var keys []string
	var vals []int
	KVListOf(m).Keys().Filter(func(s string) bool { return s == "a" }).Result(&keys)
	suite.ElementsMatch([]string{"a"}, keys)
	KVListOf(m).Values().Filter(func(v int) bool { return v == 2 }).Result(&vals)
	suite.ElementsMatch([]int{2}, vals)
}

func (suite *ObjectComprehensionTestSuite) TestMap() {
	m := map[string]int{
		"a": 1,
		"b": 2,
	}
	vk := KVListOf(m).Map(func(k string, v int) (int, string) {
		return v, k
	}).MustGetResult().(map[int]string)
	suite.Equal("a", vk[1])
	suite.Equal("b", vk[2])
}

func (suite *ObjectComprehensionTestSuite) TestMapValue() {
	m := map[string]int{
		"a": 1,
		"b": 2,
	}
	vk := KVListOf(m).MapValue(func(k string, v int) string {
		return k
	}).MustGetResult().(map[string]string)
	suite.Equal("a", vk["a"])
	suite.Equal("b", vk["b"])

	vk = KVListOf(m).MapValue(func(v int) string {
		return fmt.Sprint(v)
	}).MustGetResult().(map[string]string)
	suite.Equal("1", vk["a"])
	suite.Equal("2", vk["b"])
}

func (suite *ObjectComprehensionTestSuite) TestMapKey() {
	m := map[string]int{
		"a": 1,
		"b": 2,
	}
	vk := KVListOf(m).MapKey(func(k string) string {
		return strings.ToUpper(k)
	}).MustGetResult().(map[string]int)
	suite.Equal(1, vk["A"])
	suite.Equal(2, vk["B"])

	vk = KVListOf(m).MapKey(func(k string, v int) string {
		return strings.ToUpper(k)
	}).MustGetResult().(map[string]int)
	suite.Equal(1, vk["A"])
	suite.Equal(2, vk["B"])
}

func (suite *ObjectComprehensionTestSuite) TestFilter() {
	m := map[string]int{
		"a": 1,
		"b": 2,
	}
	suite.ElementsMatch(
		[]int{1},
		KVListOf(m).Filter(func(k string, v int) bool {
			return v == 1
		}).Values().MustGetResult().([]int),
	)
	suite.ElementsMatch(
		[]int{1},
		KVListOf(m).Reject(func(k string, v int) bool {
			return v == 2
		}).Values().MustGetResult().([]int),
	)
}

func (suite *ObjectComprehensionTestSuite) TestConstructor() {
	m := map[string]int{
		"a": 1,
		"b": 2,
	}
	suite.ElementsMatch(
		[]int{1},
		CreateKVList(func() map[string]int { return m }).Filter(func(k string, v int) bool {
			return v == 1
		}).Values().MustGetResult().([]int),
	)
	suite.ElementsMatch(
		[]int{1},
		CreateKVList(func() map[string]int { return m }).Reject(func(k string, v int) bool {
			return v == 2
		}).Values().MustGetResult().([]int),
	)
}
