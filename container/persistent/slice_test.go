package persistent

import (
	"fmt"
	"slices"
	"testing"
)

func makeSampleVector() Vector[int] {
	n := &interiorNode[int]{
		cumSums: &[maxBranch]uint{
			14, 22, 22, 22,
		},
		children: [maxBranch]node[int]{
			&interiorNode[int]{
				cumSums: &[maxBranch]uint{
					4, 7, 11, 14,
				},
				children: [maxBranch]node[int]{
					&leafNode[int]{4, [maxBranch]int{0, 1, 2, 3}},
					&leafNode[int]{3, [maxBranch]int{4, 5, 6}},
					&leafNode[int]{4, [maxBranch]int{7, 8, 9, 10}},
					&leafNode[int]{3, [maxBranch]int{11, 12, 13}},
				},
			},
			&interiorNode[int]{
				cumSums: &[maxBranch]uint{
					3, 7, 8, 8,
				},
				children: [maxBranch]node[int]{
					&leafNode[int]{3, [maxBranch]int{14, 15, 16}},
					&leafNode[int]{4, [maxBranch]int{17, 18, 19, 20}},
					&leafNode[int]{1, [maxBranch]int{21}},
				},
			},
		},
	}
	const shift = 2 * maxBranchExp
	return Vector[int]{
		root:  n,
		n:     22,
		shift: shift,
	}
}

func BenchmarkAppendPop(b *testing.B) {
	values := make([]int, maxBranch-1)
	v := NewVector(values)
	for b.Loop() {
		v2 := v.Append(0)
		_ = v2.Pop()
	}
}

func BenchmarkAppendPopStraddle(b *testing.B) {
	values := make([]int, maxBranch)
	v := NewVector(values)
	for b.Loop() {
		v2 := v.Append(0)
		_ = v2.Pop()
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

func TestGet(t *testing.T) {
	v := makeSampleVector()

	for i := range 22 {
		if got := v.Get(i); got != i {
			t.Fatalf("v.Get(%d) = %d, want %d", i, got, i)
		}
	}
}

func TestUpdate(t *testing.T) {
	v1 := makeSampleVector()
	v2 := v1.Update(21, 42)

	got := slices.Collect(v2.All())
	want := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 42}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestEqualFunc(t *testing.T) {
	a := &interiorNode[int]{}
	b := &interiorNode[int]{}
	c := &interiorNode[int]{}

	l0 := &leafNode[int]{n: 2, values: [maxBranch]int{1, 2}}
	l1 := &leafNode[int]{n: 2, values: [maxBranch]int{3, 4}}
	l2 := &leafNode[int]{n: 2, values: [maxBranch]int{5, 6}}
	l3 := &leafNode[int]{n: 2, values: [maxBranch]int{7, 8}}

	a.children = [maxBranch]node[int]{b, c}
	b.children = [maxBranch]node[int]{l0, l1}
	c.children = [maxBranch]node[int]{l2, l3}

	x := &interiorNode[int]{}
	y := &interiorNode[int]{}
	z := &interiorNode[int]{}

	l4 := &leafNode[int]{n: 3, values: [maxBranch]int{1, 2, 3}}
	l5 := &leafNode[int]{n: 1, values: [maxBranch]int{4}}

	x.children = [maxBranch]node[int]{y}
	y.children = [maxBranch]node[int]{z, c}
	z.children = [maxBranch]node[int]{l4, l5}

	v0 := Vector[int]{
		root: a,
	}
	v1 := Vector[int]{
		root: x,
	}

	if !v0.EqualFunc(v1, func(l int, r int) bool {
		return l == r
	}) {
		t.Fatalf("vectors did not compare equal")
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

	log := []node[int]{
		a, b, e, l0, c, f, l1, g, l2, d, l3,
	}
	p := newPreorder(a)
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
	p = newPreorder(a)
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
