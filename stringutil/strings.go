package stringutil

import (
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"time"
)

// InteractStrings string list
func InteractStrings(lists ...[]string) []string {
	for i := range lists {
		if len(lists[i]) == 0 {
			return nil
		}
		sort.Strings(lists[i])
	}
	return interactList(lists, 0, len(lists)-1)
}

// SubstractStrings string list
func SubstractStrings(list1 []string, list2 []string) []string {
	sort.Strings(list1)
	sort.Strings(list2)
	var res []string
	var i, j int
	for i < len(list1) && j < len(list2) {
		if list1[i] < list2[j] {
			res = append(res, list1[i])
			i++
		} else if list1[i] == list2[j] {
			i++
			j++
		} else {
			j++
		}
	}
	if i < len(list1) {
		res = append(res, list1[i:]...)
	}
	return res
}

// UnionStrings string list
func UnionStrings(lists ...[]string) []string {
	for i := range lists {
		sort.Strings(lists[i])
	}
	return andList(lists, 0, len(lists)-1)
}

// EqualStrings string lists have same elements
func EqualStrings(list1, list2 []string) bool {
	if len(list1) != len(list2) {
		return false
	}
	sort.Strings(list1)
	sort.Strings(list2)
	for i := 0; i < len(list1); i++ {
		if list1[i] != list2[i] {
			return false
		}
	}
	return true
}

// UniqStrings string list
func UniqStrings(list []string) []string {
	if len(list) <= 1 {
		return list
	}
	memo := make(map[string]int)
	for _, e := range list {
		memo[e] = 1
	}
	i := 0
	for str := range memo {
		list[i] = str
		i++
	}
	return list[:i]
}

// ContainString contain
func ContainString(list []string, target string) bool {
	for _, e := range list {
		if e == target {
			return true
		}
	}
	return false
}

// CopyStrings copy string list
func CopyStrings(list []string) []string {
	list2 := make([]string, len(list))
	copy(list2, list)
	return list2
}

// RemoveString remove string
func RemoveString(list []string, str string) []string {
	offset := 0
	for i, ele := range list {
		if ele == str {
			offset++
		} else if offset > 0 {
			list[i-offset] = ele
		}
	}
	return list[:len(list)-offset]
}

// ContainStrings contains string list
func ContainStrings(list []string, sub []string) bool {
	memo := make(map[string]int)
	for _, e := range list {
		memo[e] = 1
	}
	for _, e := range sub {
		if _, ok := memo[e]; !ok {
			return false
		}
	}
	return true
}

// ContainsAny char
func ContainsAny(text string, sub []string) bool {
	for _, str := range sub {
		if strings.Contains(text, str) {
			return true
		}
	}
	return false
}

// 是否包含，不区分大小写
func ContainStringIgnoreCase(list []string, target string) bool {
	target = strings.ToLower(target)
	for _, e := range list {
		if strings.ToLower(e) == target {
			return true
		}
	}
	return false
}

// 按大小分组
func PartitionStrings(list []string, size int) [][]string {
	var res [][]string
	for start := 0; start < len(list); start += size {
		end := start + size
		if end > len(list) {
			end = len(list)
		}
		res = append(res, list[start:end])
	}
	return res
}

// 转换元素
func MapStrings(list []string, f func(string) string) []string {
	for i, str := range list {
		list[i] = f(str)
	}
	return list
}

// 是否纯数字
func IsDigit(str string) bool {
	data := []byte(str)
	for _, b := range data {
		if b < 48 || b > 57 {
			return false
		}
	}
	return true
}

func IsBlankStr(s string) bool {
	return strings.TrimSpace(s) == ""
}

// 打乱数组
func ChaosArrays(arrays ...interface{}) {
	if len(arrays) == 0 {
		panic("no input")
	}
	var size int
	for _, array := range arrays {
		val := reflect.ValueOf(array)
		if size == 0 {
			size = val.Len()
		} else if size != val.Len() {
			panic("all array length should be equals")
		}
		if val.Kind() != reflect.Array && val.Kind() != reflect.Slice {
			panic("input parameters should be array")
		}
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < size-1; i++ {
		n := r.Intn(100) % (size - i - 1)
		for j := range arrays {
			exchangeArrayElem(arrays[j], size-i-1, n)
		}
	}
}

func exchangeArrayElem(array interface{}, i, j int) {
	if i == j {
		return
	}
	val := reflect.ValueOf(array)
	a, b := val.Index(i).Interface(), val.Index(j).Interface()
	val.Index(j).Set(reflect.ValueOf(a))
	val.Index(i).Set(reflect.ValueOf(b))
}

// below are helpers
func interactStrings(list1 []string, list2 []string) []string {
	if len(list1) == 0 || len(list2) == 0 {
		return nil
	}
	var res []string
	for i, j := 0, 0; i < len(list1) && j < len(list2); {
		if list1[i] == list2[j] {
			res = append(res, list1[i])
			i++
			j++
		} else if list1[i] > list2[j] {
			j++
		} else {
			i++
		}
	}
	return res
}

func interactList(lists [][]string, start, end int) []string {
	if start > end {
		return nil
	}
	mid := (start + end) / 2
	var left, right []string
	switch mid - start {
	case 0:
		left = lists[start]
	case 1:
		left = interactStrings(lists[start], lists[mid])
	default:
		left = interactList(lists, start, mid)
	}
	mid++
	if mid <= end {
		switch end - mid {
		case 0:
			right = lists[mid]
		case 1:
			right = interactStrings(lists[mid], lists[end])
		default:
			right = interactList(lists, mid, end)
		}
		return interactStrings(left, right)
	} else {
		return left
	}
}

func andList(lists [][]string, start, end int) []string {
	if start > end {
		return nil
	}
	mid := (start + end) / 2
	var left, right []string
	switch mid - start {
	case 0:
		left = lists[start]
	case 1:
		left = andStrings(lists[start], lists[mid])
	default:
		left = andList(lists, start, mid)
	}
	mid++
	if mid <= end {
		switch end - mid {
		case 0:
			right = lists[mid]
		case 1:
			right = andStrings(lists[mid], lists[end])
		default:
			right = andList(lists, mid, end)
		}
	}
	return andStrings(left, right)
}

func andStrings(list1 []string, list2 []string) []string {
	var res []string
	var i, j int
	for i < len(list1) && j < len(list2) {
		if list1[i] == list2[j] {
			res = append(res, list1[i])
			i++
			j++
		} else if list1[i] > list2[j] {
			res = append(res, list2[j])
			j++
		} else {
			res = append(res, list1[i])
			i++
		}
	}
	if i < len(list1) {
		res = append(res, list1[i:]...)
	}
	if j < len(list2) {
		res = append(res, list2[j:]...)
	}
	return res
}

// UnderlineLowercase 将大写单词转化成小写加下划线
func UnderlineLowercase(name string) string {
	data := []byte(name)
	res := make([]byte, 0, len(name))
	for i, b := range data {
		if b >= 'A' && b <= 'Z' {
			b += 32
			if i > 0 && (data[i-1] < 'A' || data[i-1] > 'Z') {
				res = append(res, byte(95))
			}
		}
		res = append(res, b)
	}
	return string(res)
}

// LowercaseWithSpace 将大写单词转化成小写以空格分割
func LowercaseWithSpace(name string) string {
	return strings.Replace(UnderlineLowercase(name), "_", " ", -1)
}
