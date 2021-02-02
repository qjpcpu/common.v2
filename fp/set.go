package fp

import (
	"reflect"
)

type _Set struct {
	smap           reflect.Value
	keyTyp, valTyp reflect.Type
}

func makeSet(keyTyp, valTyp reflect.Type) *_Set {
	return &_Set{
		smap:   reflect.MakeMap(reflect.MapOf(keyTyp, valTyp)),
		keyTyp: keyTyp,
		valTyp: valTyp,
	}
}

func (s *_Set) Add(k, v reflect.Value) bool {
	if s.Contains(k) {
		return false
	}
	s.smap.SetMapIndex(k, v)
	return true
}

func (s *_Set) Contains(k reflect.Value) bool {
	if ele := s.smap.MapIndex(k); !ele.IsValid() {
		return false
	}
	return true
}

type _ListSet struct {
	*_List
	set *_Set
}
