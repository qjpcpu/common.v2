package joint

import (
	"reflect"
)

type node struct {
	val  reflect.Value
	next *node
}

type linkedList struct {
	head *node
	rear *node
}

func newList() *linkedList {
	return &linkedList{}
}

func (l *linkedList) push(v reflect.Value) {
	n := &node{
		val: v,
	}
	if l.rear == nil {
		l.head = n
		l.rear = n
	} else {
		l.rear.next = n
		l.rear = n
	}
}

func (l *linkedList) pop() (reflect.Value, bool) {
	if l.head == nil {
		return reflect.Value{}, false
	} else {
		n := l.head
		if l.head == l.rear {
			l.head = nil
			l.rear = nil
		} else {
			l.head = l.head.next
		}
		return n.val, true
	}
}
