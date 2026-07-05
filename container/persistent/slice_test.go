package persistent

import (
	"fmt"
	"testing"
)

// func TestExample(t *testing.T) {
// 	v := NewVector([]int{})
// 	for i := range 5 {
// 		v = v.Append(i)
// 	}
// 	for range 5 {
// 		fmt.Println(v.Length())
// 		v = v.Pop()
// 	}
// 	fmt.Println(v.Length())
// }

func makeSampleVector() Vector[int] {
	values := make([]int, 22)
	for i := range values {
		values[i] = i
	}
	return NewVector(values)
}

func BenchmarkAppendPop(b *testing.B) {
	values := make([]int, maxBranch)
	v := NewVector(values)
	for b.Loop() {
		_ = v.Pop()
	}
}

func BenchmarkAppendPopLift(b *testing.B) {
	values := make([]int, maxBranch+1)
	v := NewVector(values)
	for b.Loop() {
		_ = v.Pop()
	}
}

func BenchmarkEqualFuncFirst(b *testing.B) {
	v1 := makeSampleVector()
	v2 := v1.Update(0, 42)
	for b.Loop() {
		v1.EqualFunc(v2, func(a, b int) bool { return a == b })
	}
}

func BenchmarkEqualFuncLast(b *testing.B) {
	v1 := makeSampleVector()
	v2 := v1.Update(21, 42)
	for b.Loop() {
		v1.EqualFunc(v2, func(a, b int) bool { return a == b })
	}
}

func BenchmarkEqualFuncIdentical(b *testing.B) {
	v1 := makeSampleVector()
	for b.Loop() {
		v1.EqualFunc(v1, func(a, b int) bool { return a == b })
	}
}

func BenchmarkEqualFuncStrictFirst(b *testing.B) {
	v1 := makeSampleVector()
	v2 := v1.Update(0, 42)
	for b.Loop() {
		v1.EqualFuncStrict(v2, func(a, b int) bool { return a == b })
	}
}

func BenchmarkEqualFuncStrictLast(b *testing.B) {
	v1 := makeSampleVector()
	v2 := v1.Update(21, 42)
	for b.Loop() {
		v1.EqualFuncStrict(v2, func(a, b int) bool { return a == b })
	}
}

func BenchmarkEqualFuncStrictIdentical(b *testing.B) {
	v1 := makeSampleVector()
	for b.Loop() {
		v1.EqualFuncStrict(v1, func(a, b int) bool { return a == b })
	}
}

func TestPreorder(t *testing.T) {
	a := &interiorNode[int]{}
	b := &interiorNode[int]{}
	c := &interiorNode[int]{}
	d := &interiorNode[int]{}
	e := &interiorNode[int]{}
	f := &interiorNode[int]{}
	g := &interiorNode[int]{}

	l0 := &leafNode[int]{}
	l1 := &leafNode[int]{}
	l2 := &leafNode[int]{}
	l3 := &leafNode[int]{}

	a.children = [maxBranch]node[int]{b, c, nil, d}
	b.children = [maxBranch]node[int]{e}
	c.children = [maxBranch]node[int]{f, g}
	d.children = [maxBranch]node[int]{l3}
	e.children = [maxBranch]node[int]{l0}
	f.children = [maxBranch]node[int]{l1}
	g.children = [maxBranch]node[int]{l2}

	names := map[node[int]]string{
		a:  "a",
		b:  "b",
		c:  "c",
		d:  "d",
		e:  "e",
		f:  "f",
		g:  "g",
		l0: "l0",
		l1: "l1",
		l2: "l2",
		l3: "l3",
	}

	// First we test that a simple preorder traversal without skips yields
	// nodes in the expected order.

	tail := &leafNode[int]{}
	log := []node[int]{
		a, b, e, l0, c, f, l1, g, l2, d, l3, tail,
	}
	p := newPreorder(a, tail)
	for i := range len(log) {
		cur := p.current()
		if cur != log[i] {
			got, ok := names[cur]
			if !ok {
				got = fmt.Sprintf("%p", cur)
			}
			want, ok := names[log[i]]
			if !ok {
				got = fmt.Sprintf("%p", log[i])
			}
			t.Fatalf("#%d: p.current() == %s, want %s", i, got, want)
		}
		if i < len(log)-1 {
			if !p.next() {
				t.Fatalf("#%d: p.next() returned false", i)
			}
		} else {
			if p.next() {
				t.Fatalf("#%d: p.next() returned true", i)
			}
		}
	}

	// Next we test that we can skip the children of c
	log = []node[int]{
		a, b, e, l0, c, d, l3,
	}
	p = newPreorder(a, nil)
	for i := range len(log) {
		cur := p.current()
		if cur != log[i] {
			got, ok := names[cur]
			if !ok {
				got = fmt.Sprintf("%p", cur)
			}
			want, ok := names[log[i]]
			if !ok {
				got = fmt.Sprintf("%p", log[i])
			}
			t.Fatalf("#%d: p.current() == %s, want %s", i, got, want)
		}
		if cur == c {
			p.ascend()
			p.next()
			continue
		}
		if i < len(log)-1 {
			if !p.next() {
				t.Fatalf("#%d: p.next() returned false", i)
			}
		} else {
			if p.next() {
				t.Fatalf("#%d: p.next() returned true", i)
			}
		}
	}
}
