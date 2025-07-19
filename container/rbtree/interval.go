// SPDX-FileCopyrightText: 2023 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package rbtree

import (
	"cmp"
)

type Interval[T cmp.Ordered] struct {
	Min, Max T
}

type Value[T cmp.Ordered, V any] struct {
	MaxSubtree T
	Value      V
}

func (ival Interval[T]) Compare(oval Interval[T]) int {
	if ival.Min < oval.Min {
		return -1
	} else if ival.Min > oval.Min {
		return 1
	} else {
		if ival.Max < oval.Max {
			return -1
		} else if ival.Max > oval.Max {
			return 1
		} else {
			return 0
		}
	}
}

func (ival Interval[T]) Overlaps(oval Interval[T]) bool {
	ret := ival.Min <= oval.Max && ival.Max >= oval.Min

	return ret
}

func (ival Interval[T]) SupersetOf(oval Interval[T]) bool {
	return ival.Min <= oval.Min && ival.Max >= oval.Max
}

type IntervalTree[T cmp.Ordered, V any] struct {
	Tree[Interval[T], Value[T, V]]
}

func (t *IntervalTree[T, V]) Insert(min, max T, value V) *Node[Interval[T], Value[T, V]] {
	n := t.Tree.Insert(Interval[T]{min, max}, Value[T, V]{MaxSubtree: max, Value: value})
	t.updateAug(n)
	return n
}

func (t *IntervalTree[T, V]) updateAug(n *Node[Interval[T], Value[T, V]]) {
	if n == nil {
		return
	}

	max := n.Key.Max
	if c := n.Children[0]; c != nil && c.Values[0].MaxSubtree > max {
		max = c.Values[0].MaxSubtree
	}
	if c := n.Children[1]; c != nil && c.Values[0].MaxSubtree > max {
		max = c.Values[0].MaxSubtree
	}

	n.Values[0].MaxSubtree = max
	t.updateAug(n.Parent)
}

func (t *IntervalTree[T, V]) Find(
	min T,
	max T,
	out []*Node[Interval[T], Value[T, V]],
) []*Node[Interval[T], Value[T, V]] {
	return t.find(t.Root, min, max, out)
}

func (t *IntervalTree[T, V]) FindIter(
	min T,
	max T,
	cb func(node *Node[Interval[T], Value[T, V]]) bool,
) {
	t.findIter(t.Root, min, max, cb)
}

func (t *IntervalTree[T, V]) find(
	node *Node[Interval[T], Value[T, V]],
	min T,
	max T,
	out []*Node[Interval[T], Value[T, V]],
) []*Node[Interval[T], Value[T, V]] {
	if node == nil {
		return out
	}

	if min > node.Values[0].MaxSubtree {
		// This node and both subtrees are too small for our start point.
		return out
	}

	out = t.find(node.Children[Left], min, max, out)

	if node.Key.Overlaps(Interval[T]{min, max}) {
		out = append(out, node)
	}

	out = t.find(node.Children[Right], min, max, out)

	return out
}

func (t *IntervalTree[T, V]) findIter(
	node *Node[Interval[T], Value[T, V]],
	min T,
	max T,
	cb func(node *Node[Interval[T], Value[T, V]]) bool,
) bool {
	if node == nil {
		return false
	}

	if min > node.Values[0].MaxSubtree {
		// This node and both subtrees are too small for our start point.
		return false
	}

	if t.findIter(node.Children[Left], min, max, cb) {
		return true
	}

	if node.Key.Overlaps(Interval[T]{min, max}) {
		if cb(node) {
			return true
		}
	}

	if t.findIter(node.Children[Right], min, max, cb) {
		return true
	}

	return false
}

func NewIntervalTree[T cmp.Ordered, V any]() *IntervalTree[T, V] {
	t := &IntervalTree[T, V]{}
	t.Rotated = func(node *Node[Interval[T], Value[T, V]]) {
		t.updateAug(node)
	}
	return t
}
