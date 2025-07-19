// SPDX-FileCopyrightText: 2023 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package rbtree

import (
	"fmt"
	"io"
)

type Direction uint8
type Color bool

const (
	Left  Direction = 0
	Right Direction = 1
)

const (
	Black Color = false
	Red   Color = true
)

type Comparable[T any] interface {
	Compare(T) int
}

type Tree[K Comparable[K], V any] struct {
	Root            *Node[K, V]
	NumValues       int
	AllowDuplicates bool

	Rotated func(node *Node[K, V])
}

type Node[K Comparable[K], V any] struct {
	Parent   *Node[K, V]
	Children [2]*Node[K, V]
	Key      K
	Values   []V
	color    Color
}

func NewNode[K Comparable[K], V any](k K, v V) *Node[K, V] {
	return &Node[K, V]{
		Key:    k,
		Values: []V{v},
	}
}

func (t *Tree[K, V]) Inorder(yield func(K, V) bool) {
	t.Root.Inorder(yield)
}

func (t *Node[K, V]) Inorder(yield func(K, V) bool) bool {
	if t == nil {
		return true
	}
	if !t.Children[0].Inorder(yield) {
		return false
	}
	for _, v := range t.Values {
		if !yield(t.Key, v) {
			return false
		}
	}
	return t.Children[1].Inorder(yield)
}

func (t *Tree[K, V]) Search(k K) (node *Node[K, V], found bool, dir Direction) {
	if t.Root == nil {
		return nil, false, 0
	}

	x := t.Root
	for {
		switch k.Compare(x.Key) {
		case -1:
			dir = Left
		case 0:
			return x, true, 0
		case 1:
			dir = Right
		}

		child := x.Children[dir]
		if child == nil {
			return x, false, dir
		}
		x = child
	}
}

func (t *Tree[K, V]) rotate(p *Node[K, V], dir Direction) *Node[K, V] {
	g := p.Parent
	s := p.Children[1-dir]
	c := s.Children[dir]
	p.Children[1-dir] = c
	if c != nil {
		c.Parent = p
	}
	s.Children[dir] = p
	p.Parent = s
	s.Parent = g
	if g != nil {
		var child Direction
		if p == g.Children[Right] {
			child = Right
		} else {
			child = Left
		}
		g.Children[child] = s
	} else {
		t.Root = s
	}

	if t.Rotated != nil {
		t.Rotated(p)
	}

	return s
}

func (t *Tree[K, V]) Insert(k K, v V) *Node[K, V] {
	if t.Root == nil {
		t.NumValues++
		n := NewNode(k, v)
		t.insert(n, nil, 0)
		return n
	}

	p, ok, dir := t.Search(k)
	if ok {
		if t.AllowDuplicates {
			t.NumValues++
			p.Values = append(p.Values, v)
		} else {
			// OPT(dh): we could write to p.Values[0] instead, but then we must forbid users from retaining
			// Node.Values even when !t.AllowDuplicates
			p.Values = []V{v}
		}
		return p
	} else {
		t.NumValues++
		n := NewNode(k, v)
		t.insert(n, p, dir)
		return n
	}
}

func (t *Tree[K, V]) insert(n *Node[K, V], p *Node[K, V], dir Direction) {
	var g *Node[K, V]
	var u *Node[K, V]

	n.color = Red
	n.Children[Left] = nil
	n.Children[Right] = nil
	n.Parent = p
	if p == nil {
		t.Root = n
		return
	}
	p.Children[dir] = n

	for {
		if p.color == Black {
			return
		}

		g = p.Parent
		if g == nil {
			p.color = Black
			return
		}

		dir = p.childDir()
		u = g.Children[1-dir]
		if u == nil || u.color == Black {
			if n == p.Children[1-dir] {
				t.rotate(p, dir)
				n = p
				p = g.Children[dir]
			}

			t.rotate(g, 1-dir)
			p.color = Black
			g.color = Red
			return
		}

		p.color = Black
		u.color = Black
		g.color = Red
		n = g

		p = n.Parent
		if p == nil {
			break
		}
	}
}

func (n *Node[K, V]) childDir() Direction {
	if n.Parent.Children[Right] == n {
		return Right
	} else {
		return Left
	}
}

func (n *Node[K, V]) Dot(w io.Writer, meta func(n *Node[K, V]) string) {
	p := func(s string) {
		w.Write([]byte(s))
		w.Write([]byte("\n"))
	}
	pf := func(f string, vs ...any) {
		fmt.Fprintf(w, f, vs...)
		w.Write([]byte("\n"))
	}

	var node func(n *Node[K, V])
	node = func(n *Node[K, V]) {
		if n == nil {
			return
		}

		var c string
		if n.color == Black {
			c = "black"
		} else {
			c = "red"
		}
		label := fmt.Sprintf("%v = %v", n.Key, n.Values)
		if meta != nil {
			label += "\n" + meta(n)
		}
		pf(`p%p [label="%s", color=%s];`, n, label, c)

		for i, child := range n.Children {
			if child == nil {
				pf("p%pc%d [label=nil, style=invis];", n, i)
				pf("p%p -> p%pc%d [style=invis];", n, n, i)
			} else {
				node(child)
				pf("p%p -> p%p;", n, child)
			}
		}

	}

	p("digraph {")
	p("graph [ordering=out];")

	node(n)

	p("}")
}
