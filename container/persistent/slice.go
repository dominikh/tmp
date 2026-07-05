//go:build go1.27

package main

import (
	"fmt"
	"io"
	"iter"
	"log"
	"os"
	"slices"
	"strings"
)

func main() {
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

	v1 := Vector[int]{
		root:  n,
		n:     22,
		shift: shift,
	}

	v2 := v1.Update(13, 42)

	v1.dot("v1", "v1", os.Stdout)
	v2.dot("v2", "v2", os.Stdout)

	// const N = 1_000_000

	// {
	// 	t := time.Now()
	// 	for range N {
	// 		for i := range 22 {
	// 			_ = v1.Get(i)
	// 		}
	// 	}
	// 	fmt.Println(float64(time.Since(t)) / (N * 22))
	// }
}

const (
	maxSearchError = 2
	maxBranchExp   = 2 // 2 during debugging, 5 in production

	// maxBranch must be 2**maxBranchExp.
	maxBranch = 1 << maxBranchExp
)

var _ [maxSearchError]struct{} // maxSearchError >= 0

type Vector[T any] struct {
	root node[T]
	n    int
	// shift is maxBranchExp * height
	shift uint8
}

// EqualFunc reports whether two vectors are equal. If the lengths are
// different, EqualFunc returns false. eq gets called for every pair of values
// to determine equality, unless both vectors store the pair in the same
// backing array, in which case all values in the array are trivially equal.
// This optimization ignores the semantics of NaN; that is, NaN may equal NaN.
//
// For a function that calls eq for every pair, without optimizations, see
// [Vector.EqualFuncStrict]. Unlike EqualFunc, it supports comparing vectors
// with different element types.
func (v Vector[T]) EqualFunc(vo Vector[T], eq func(T, T) bool) bool {
	if v.Length() != vo.Length() {
		return false
	}

	// Trivially the same
	if v.root == vo.root {
		return true
	}

	rootv, okv := v.root.(*interiorNode[T])
	rootvo, okvo := vo.root.(*interiorNode[T])

	if !okv && !okvo {
		// Both roots are leaves. We already know that they aren't the same
		// node. Neither root can be nil. If only one of them were
		// nil, we'd have returned early due to mismatching vector lengths. If
		// both were nil, we'd have returned early because they're identical.
		rootv := v.root.(*leafNode[T])
		rootvo := vo.root.(*leafNode[T])
		return slices.EqualFunc(
			rootv.values[:rootv.n],
			rootvo.values[:rootvo.n],
			eq,
		)
	}

	// We have both equalLeafIter and equalIterLeaf to ensure that eq always
	// gets called with v's elements as the first argument.
	equalLeafIter := func(leaf *leafNode[T], inter *interiorNode[T]) bool {
		i := 0
		return inter.every(func(v T) bool {
			if i >= int(leaf.n) {
				return false
			}
			b := eq(leaf.values[i], v)
			i++
			return b
		})
	}
	equalIterLeaf := func(inter *interiorNode[T], leaf *leafNode[T]) bool {
		i := 0
		return inter.every(func(v T) bool {
			if i >= int(leaf.n) {
				return false
			}
			b := eq(v, leaf.values[i])
			i++
			return b
		})
	}

	if !okv {
		// v.root is a leaf, vo.root is an interior node
		return equalLeafIter(v.root.(*leafNode[T]), rootvo)
	} else if !okvo {
		// v.root is an interior node, vo.root is a leaf
		return equalIterLeaf(rootv, vo.root.(*leafNode[T]))
	}

	// Both roots are interior nodes.
	pv := newPreorder(rootv)
	pvo := newPreorder(rootvo)

	var left, right []T
outer:
	for {
		cv, okv := pv.current().(*leafNode[T])
		cvo, okvo := pvo.current().(*leafNode[T])
		if okv && okvo {
			if pv.current() == pvo.current() {
				log.Println("skipping identical leaves", pv.current())
				bv := pv.next()
				bvo := pvo.next()
				if !bv && !bvo {
					return true
				}
				continue outer
			} else {
				left = cv.values[:cv.n]
				right = cvo.values[:cvo.n]
			}
		} else if okv && !okvo {
			left = cv.values[:cv.n]

			// Advance pvo until it hits a leaf
			for {
				if !pvo.next() {
					return false
				}
				if c, ok := pvo.current().(*leafNode[T]); ok {
					right = c.values[:c.n]
					break
				}
			}
		} else if !okv && okvo {
			right = cvo.values[:cvo.n]

			// Advance pv until it hits a leaf
			for {
				if !pv.next() {
					return false
				}
				if c, ok := pv.current().(*leafNode[T]); ok {
					left = c.values[:c.n]
					break
				}
			}
		} else if !okv && !okvo {
			if pv.current() == pvo.current() {
				// Skip common subtree
				log.Println("skipping common subtree", pv.current())
				pv.ascend()
				pvo.ascend()
			}
			bv := pv.next()
			bvo := pvo.next()
			if bv != bvo {
				return false
			}
			if !bv {
				return true
			}
			continue outer
		}

		// Both are leaves. Compare and consume the shorter prefix, advance
		// the other side to the next leaf. If both leaves have been fully
		// consumed. advance both sides.
		if len(left) == 0 {
			panic("internal error: left side is empty")
		}
		if len(right) == 0 {
			panic("internal error: right side is empty")
		}
		for {
			n := min(len(left), len(right))
			log.Println("comparing", left[:n], right[:n])
			if !slices.EqualFunc(left[:n], right[:n], eq) {
				return false
			}
			left = left[n:]
			right = right[n:]
			if len(left) == 0 && len(right) == 0 {
				bv := pv.next()
				bvo := pvo.next()
				if !bv && !bvo {
					return true
				}
				continue outer
			} else if len(left) == 0 {
				// Advance pv until it hits a leaf
				for {
					if !pv.next() {
						return false
					}
					if c, ok := pv.current().(*leafNode[T]); ok {
						left = c.values[:c.n]
						break
					}
				}
			} else if len(right) == 0 {
				// Advance pvo until it hits a leaf
				for {
					if !pvo.next() {
						return false
					}
					if c, ok := pvo.current().(*leafNode[T]); ok {
						right = c.values[:c.n]
						break
					}
				}
			}
		}
	}
}

func newPreorder[T any](n *interiorNode[T]) *preorder[T] {
	return &preorder[T]{
		stack: []preorderStackEntry[T]{{n, -1}},
	}
}

type preorderStackEntry[T any] struct {
	node        *interiorNode[T]
	curChildIdx int
}

type preorder[T any] struct {
	stack []preorderStackEntry[T]
}

func (p *preorder[T]) current() node[T] {
	n := p.stack[len(p.stack)-1]
	if n.curChildIdx == -1 {
		return n.node
	}

	if n.curChildIdx >= len(n.node.children) {
		if len(p.stack) != 1 {
			panic("internal error")
		}
		// This has to be the root node.
		return n.node
	}

	// This will always be a leaf node.
	return n.node.children[n.curChildIdx]
}

// next executes the next step of a preorder traversal. It descends into the
// current node's next unvisited child. If no unvisited children are left, it
// first ascends to the first parent with unvisited children. If there are none
// left, next returns false.
func (p *preorder[T]) next() bool {
	if len(p.stack) == 0 {
		panic("unreachable")
	}
	cur := &p.stack[len(p.stack)-1]
	if len(p.stack) == 1 && cur.curChildIdx >= len(cur.node.children) {
		// No children left
		return false
	}
	cur.curChildIdx++
	for cur.curChildIdx < len(cur.node.children) && cur.node.children[cur.curChildIdx] == nil {
		cur.curChildIdx++
	}
	if cur.curChildIdx >= len(cur.node.children) {
		if p.ascend() {
			return p.next()
		} else {
			return false
		}
	}
	next := cur.node.children[cur.curChildIdx]
	if next, ok := next.(*interiorNode[T]); ok {
		p.stack = append(p.stack, preorderStackEntry[T]{
			node:        next,
			curChildIdx: -1,
		})
	}
	return true
}

// ascend pops an entry from the traversal stack. It returns false if the stack
// contained fewer than two entries.
func (p *preorder[T]) ascend() bool {
	if len(p.stack) < 2 {
		return false
	}
	p.stack = p.stack[:len(p.stack)-1]
	return true
}

// EqualFuncStrict reports whether two vectors are equal using an equality
// function on each pair of elements. If the lengths are different, EqualFunc
// returns false. Otherwise, the elements are compared in increasing index
// order, and the comparison stops at the first index for which eq returns
// false.
//
// See also [Vector.EqualFunc] for a function that optimizes away calls to eq
// for vectors that share memory.
func (v Vector[T]) EqualFuncStrict[TO any](vo Vector[TO], eq func(T, TO) bool) bool {
	if v.Length() != vo.Length() {
		return false
	}

	next, cancel := iter.Pull(v.All())
	defer cancel()
	nexto, cancelo := iter.Pull(vo.All())
	defer cancelo()

	for {
		a, oka := next()
		b, okb := nexto()
		if oka != okb {
			return false
		}
		if !oka {
			return true
		}
		if !eq(a, b) {
			return false
		}
	}
}

func (v Vector[T]) Length() int {
	return int(v.n)
}

func (v Vector[T]) All() iter.Seq[T] {
	if v.root == nil {
		return func(yield func(T) bool) {}
	}

	return func(yield func(T) bool) {
		_ = v.root.every(yield)
	}
}

func (v Vector[T]) Leaves() iter.Seq[[]T] {
	return func(yield func([]T) bool) {
		for l := range v.leaves() {
			values := l.values
			if !yield(values[:l.n]) {
				return
			}
		}
	}
}

func (v Vector[T]) leaves() iter.Seq[*leafNode[T]] {
	switch root := v.root.(type) {
	case *interiorNode[T]:
		return func(yield func(*leafNode[T]) bool) {
			var dfs func(node *interiorNode[T])
			dfs = func(node *interiorNode[T]) {
				for _, child := range node.children {
					switch child := child.(type) {
					case *interiorNode[T]:
						dfs(child)
					case *leafNode[T]:
						if !yield(child) {
							return
						}
					case nil:
					}
				}
			}
			dfs(root)
		}
	case *leafNode[T]:
		return func(yield func(*leafNode[T]) bool) {
			yield(root)
		}
	default: // nil
		return func(yield func(*leafNode[T]) bool) {}
	}
}

func (v Vector[T]) Get(idx int) T {
	if idx < 0 {
		panic(fmt.Sprintf("index out of range [%d]", idx))
	}
	if idx >= v.n {
		panic(fmt.Sprintf("index out of range [%d] with length %d", idx, v.n))
	}

	// Devirtualize call
	switch root := v.root.(type) {
	case *interiorNode[T]:
		return root.index(uint(idx), v.shift)
	case *leafNode[T]:
		return root.index(uint(idx), v.shift)
	default:
		// unreachable
		return *new(T)
	}
}

func (v Vector[T]) Update(i int, value T) Vector[T] {
	if i < 0 {
		panic(fmt.Sprintf("index out of range [%d]", i))
	}
	if i >= v.n {
		panic(fmt.Sprintf("index out of range [%d] with length %d", i, v.n))
	}

	switch root := v.root.(type) {
	case *interiorNode[T]:
		rootc := &interiorNode[T]{
			cumSums:  root.cumSums,
			children: root.children, // this is a copy
		}
		parent := rootc
		for slot, n := range root.traverse(uint(i), v.shift) {
			switch n := n.(type) {
			case *interiorNode[T]:
				nc := &interiorNode[T]{
					cumSums:  n.cumSums,
					children: n.children, // this is a copy
				}
				parent.children[slot] = nc
				parent = nc
			case *leafNode[T]:
				nc := *n
				nc.values[slot] = value
				parent.children[slot] = &nc
			}
		}

		return Vector[T]{
			root:  rootc,
			n:     v.n,
			shift: v.shift,
		}
	case *leafNode[T]:
		rootc := *root
		rootc.values[i] = value
		return Vector[T]{
			root:  &rootc,
			n:     v.n,
			shift: v.shift,
		}
	default:
		panic(fmt.Sprintf("internal error: unexpected root type %T", root))
	}
}

func (v Vector[T]) dot(name, desc string, w io.Writer) {
	fmt.Fprintln(w, "strict digraph {")
	fmt.Fprintf(w, "v%s [label=%q, shape=box];\n", name, desc)
	var dfs func(pid string, n node[T])
	dfs = func(pid string, n node[T]) {
		switch n := n.(type) {
		case *interiorNode[T]:
			id := fmt.Sprintf("i%p", n)
			var labels []string
			for i := range n.children {
				labels = append(labels, fmt.Sprintf("<f%d>", i))
			}
			label := strings.Join(labels, "|")
			fmt.Fprintf(w, "%s [shape=record, label=%q];\n", id, label)
			fmt.Fprintf(w, "%s -> %s;\n", pid, id)
			for i, child := range n.children {
				dfs(fmt.Sprintf("%s:f%d", id, i), child)
			}
		case *leafNode[T]:
			id := fmt.Sprintf("l%p", n)
			values := make([]string, len(n.values[:n.n]))
			for i := range values {
				values[i] = fmt.Sprint(n.values[i])
			}
			// FIXME(dh): this is buggy for value representations containing
			// pipes or angled brackets, among others.
			fmt.Fprintf(w, "%s [shape=record, label=%q];\n", id, strings.Join(values, "|"))
			fmt.Fprintf(w, "%s -> %s;\n", pid, id)
		case nil:
		}
	}
	dfs("v"+name, v.root)
	fmt.Fprintln(w, "}")
}

type node[T any] interface {
	index(idx uint, shift uint8) T
	every(f func(T) bool) bool
}

type leafNode[T any] struct {
	// n stores the number of populated values. This is redundant with the leaf
	// node's parent (either an interiorNode or a Vector), but simplifies
	// algorithms since they no longer need access to the parent.
	n      uint8
	values [maxBranch]T
}

func (n *leafNode[T]) index(idx uint, _ uint8) T {
	return n.values[idx]
}

func (n *leafNode[T]) every(f func(T) bool) bool {
	for _, v := range n.values[:n.n] {
		if !f(v) {
			return false
		}
	}
	return true
}

type interiorNode[T any] struct {
	// The number of values accessible via a branch. Not stored directly so
	// they can be reused between nodes.
	cumSums *[maxBranch]uint
	// children are either *[interiorNode] or *[leafNode].
	//
	// OPT(dh): we could use unsafe.Pointer instead of an interface, saving
	// some storage and some instructions on use. We can infer the type from
	// the node's level. Of course it'd be a lot less safe.
	children [maxBranch]node[T]
}

func (n *interiorNode[T]) traverse(idx uint, shift uint8) iter.Seq2[uint8, node[T]] {
	// Part of the reason why we return an iterator instead of accepting a
	// callback is that iterators have more generous inlining budgets.
	return func(yield func(uint8, node[T]) bool) {
		for {
			slot := (idx >> shift) & (maxBranch - 1)
			for slot < maxBranch && n.cumSums[slot] <= idx {
				slot++
			}
			if slot >= maxBranch {
				// unreachable for well-formed trees, but eliminates bounds checks.
				return
			}
			if slot > 0 {
				idx -= n.cumSums[slot-1]
			}

			n2 := n.children[slot]

			// When shift == maxBranchExp, the next level is the last level and
			// therefore a leaf. All leaves are on the same--the last--level.
			if shift > maxBranchExp {
				n = n2.(*interiorNode[T])
				// Decrease height by one
				shift -= maxBranchExp
				if !yield(uint8(slot), n) {
					return
				}
			} else {
				if idx >= maxBranch {
					// unreachable for well-formed trees, but eliminates bounds checks.
					return
				}
				n2 := n2.(*leafNode[T])
				if !yield(uint8(idx), n2) {
					return
				}
				return
			}
		}
	}
}

func (n *interiorNode[T]) index(idx uint, shift uint8) T {
	for slot, n2 := range n.traverse(idx, shift) {
		if n, ok := n2.(*leafNode[T]); ok {
			return n.values[slot]
		}
	}
	// unreachable
	return *new(T)
}

func (n *interiorNode[T]) every(f func(T) bool) bool {
	for _, child := range n.children {
		if child == nil {
			continue
		}
		if !child.every(f) {
			return false
		}
	}
	return true
}
