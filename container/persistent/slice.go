//go:build go1.27

package persistent

import (
	"fmt"
	"io"
	"iter"
	"math/bits"
	"slices"
	"strings"
)

const (
	maxSearchError = 2
	maxBranchExp   = 2 // 2 during debugging, 5 in production

	// maxBranch must be 2**maxBranchExp.
	maxBranch = 1 << maxBranchExp
)

var _ [maxSearchError]struct{} // maxSearchError >= 0

type Vector[T any] struct {
	root node[T]
	n    uint
	// shift is maxBranchExp * height. A vector of height h can store up to
	// maxBranch<<shift (= maxBranch * 2**shift = 2**(maxBranchExp*(h+1))
	// elements. h is zero-based, a vector with a single leaf node has height
	// 0.
	shift uint8
	// rrb is true once concatenation or left slicing has occurred and the
	// vector can no longer be assumed to be leftwise dense.
	rrb bool
}

func NewVector[T any](elems []T) Vector[T] {
	if len(elems) == 0 {
		return Vector[T]{}
	}

	var shift int
	if len(elems) > 1 {
		// Highest index is n-1. Each trie level consumes maxBranchExp bits.
		shift = ((bits.Len(uint(len(elems)-1)) - 1) / maxBranchExp) * maxBranchExp
	}

	return Vector[T]{
		n:     uint(len(elems)),
		root:  buildTrie(elems, shift),
		shift: uint8(shift),
	}
}

func buildTrie[T any](elems []T, shift int) node[T] {
	if shift == 0 {
		leaf := &leafNode[T]{n: uint8(len(elems))}
		copy(leaf.values[:], elems)
		return leaf
	}

	childWidth := 1 << shift

	n := &interiorNode[T]{}
	for i := 0; i < len(elems); i += childWidth {
		end := min(i+childWidth, len(elems))
		n.children[n.n] = buildTrie(elems[i:end], shift-maxBranchExp)
		n.n++
	}

	return n
}

func (v Vector[T]) Append(value T) Vector[T] {
	if !v.rrb {
		// Fast path
		full := maxBranch<<v.shift == v.n
		vc := v
		vc.n++
		if full {
			vc.root = &interiorNode[T]{
				n:        1,
				children: [maxBranch]node[T]{v.root},
			}
			vc.shift += maxBranchExp
		} else {
			vc.root = cloneNode(v.root)
		}

		if vc.root == nil {
			vc.root = &leafNode[T]{
				n:      1,
				values: [maxBranch]T{value},
			}
			return vc
		} else {
			t := vc.root
			for shift := vc.shift; shift >= maxBranchExp; shift -= maxBranchExp {
				i := (v.n >> shift) % maxBranch
				child := cloneNode(t.(*interiorNode[T]).children[i])
				if child == nil {
					if shift == maxBranchExp {
						child = &leafNode[T]{}
					} else {
						child = &interiorNode[T]{}
					}
					t.(*interiorNode[T]).n++
				}
				t.(*interiorNode[T]).children[i] = child
				t = child
			}
			i := v.n % maxBranch
			t.(*leafNode[T]).values[i] = value
			t.(*leafNode[T]).n++
			return vc
		}
	} else {
		// XXX implement as a concatenation
		return Vector[T]{}
	}
}

func (v Vector[T]) Pop() Vector[T] {
	if v.n == 0 {
		panic("tried to pop from empty vector")
	}

	if v.n == 1 {
		return Vector[T]{}
	}

	if !v.rrb {
		vc := v
		vc.n--
		if v.shift > 0 && vc.n == 1<<v.shift {
			// After popping, the vector is fully dense. This implies that the
			// current root has two children and that the entire right arm of
			// the tree can be popped, because it only stores a single value.
			vc.shift -= maxBranchExp
			vc.root = v.root.(*interiorNode[T]).children[0]
			return vc
		} else {
			vc.root = cloneNode(v.root)
			t := node[T](vc.root)
			for shift := vc.shift; ; shift -= maxBranchExp {
				ic := (vc.n >> shift) % maxBranch
				if vc.n%(1<<shift) == 0 {
					switch t := t.(type) {
					case *interiorNode[T]:
						// The index to delete is the first index in subtree
						// t.children[ic]. Because the index is also the last index
						// in the vector, the subtree cannot have any other
						// children, which means we can drop it.
						t.n--
						t.children[ic] = nil
					case *leafNode[T]:
						t.n--
						t.values[ic] = *new(T)
					default:
						panic("unreachable")
					}
					return vc
				}
				t_ := t.(*interiorNode[T])
				t_.children[ic] = cloneNode(t_.children[ic])
				t = t_.children[ic]
			}
		}
	} else {
		// XXX implement
		return Vector[T]{}
	}
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

	if !okv {
		// v.root is a leaf, vo.root is an interior node
		leaf := v.root.(*leafNode[T])
		i := 0
		return rootvo.every(func(v T) bool {
			if i >= int(leaf.n) {
				return false
			}
			b := eq(leaf.values[i], v)
			i++
			return b
		})
	} else if !okvo {
		// v.root is an interior node, vo.root is a leaf
		leaf := vo.root.(*leafNode[T])
		i := 0
		return rootv.every(func(v T) bool {
			if i >= int(leaf.n) {
				return false
			}
			b := eq(v, leaf.values[i])
			i++
			return b
		})
	}

	// Both roots are interior nodes.
	pv := newPreorder(rootv)
	pvo := newPreorder(rootvo)

	var left, right []T
outer:
	for {
		bv := pv.next()
		bvo := pvo.next()
		if bv != bvo {
			return false
		}
		if !bv {
			return true
		}

		if pv.current() == pvo.current() {
			// Skip common subtree
			pv.ascend()
			pvo.ascend()
			continue outer
		}

		cv, okv := pv.current().(*leafNode[T])
		cvo, okvo := pvo.current().(*leafNode[T])
		if okv && okvo {
			left = cv.values[:cv.n]
			right = cvo.values[:cvo.n]
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
			continue outer
		}

		if len(left) == 0 {
			panic("internal error: left side is empty")
		}
		if len(right) == 0 {
			panic("internal error: right side is empty")
		}
		for {
			n := min(len(left), len(right))
			if !slices.EqualFunc(left[:n], right[:n], eq) {
				return false
			}
			left = left[n:]
			right = right[n:]
			if len(left) == 0 && len(right) == 0 {
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
	// OPT(dh): instead of all this logic, set a field when the current node
	// changes.

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

	if v.Length() == 0 {
		return true
	}

	next, cancel := iter.Pull(v.leaves())
	defer cancel()
	nexto, cancelo := iter.Pull(vo.leaves())
	defer cancelo()

	nv, _ := next()
	nvo, _ := nexto()

	left := nv.values[:nv.n]
	right := nvo.values[:nvo.n]

	for {
		n := min(len(left), len(right))
		if !slices.EqualFunc(left[:n], right[:n], eq) {
			return false
		}
		left = left[n:]
		right = right[n:]

		if len(left) == 0 {
			nv, ok := next()
			if ok {
				left = nv.values[:nv.n]
			}
		}
		if len(right) == 0 {
			nvo, ok := nexto()
			if ok {
				right = nvo.values[:nvo.n]
			}
		}

		if len(left) == 0 {
			return len(right) == 0
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
			var dfs func(node *interiorNode[T]) bool
			dfs = func(node *interiorNode[T]) bool {
				for _, child := range node.children {
					switch child := child.(type) {
					case *interiorNode[T]:
						if !dfs(child) {
							return false
						}
					case *leafNode[T]:
						if !yield(child) {
							return false
						}
					case nil:
					}
				}
				return true
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
	if uint(idx) >= v.n {
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
	if uint(i) >= v.n {
		panic(fmt.Sprintf("index out of range [%d] with length %d", i, v.n))
	}

	switch root := v.root.(type) {
	case *interiorNode[T]:
		rootc := *root
		parent := &rootc
		for addr, n := range root.traverse(uint(i), v.shift) {
			switch n := n.(type) {
			case *interiorNode[T]:
				nc := &interiorNode[T]{
					cumSums:  n.cumSums,
					children: n.children, // this is a copy
				}
				parent.children[addr.slot] = nc
				parent = nc
			case *leafNode[T]:
				nc := *n
				nc.values[addr.idx] = value
				parent.children[addr.slot] = &nc
			}
		}

		return Vector[T]{
			root:  &rootc,
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

//lint:ignore U1000 Debug helper
func (v Vector[T]) dot(name string, w io.Writer) {
	fmt.Fprintln(w, "strict digraph {")
	fmt.Fprintf(w, "v%s [label=%q, shape=box];\n", name, name)
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

func cloneNode[T any](n node[T]) node[T] {
	switch n := n.(type) {
	case *leafNode[T]:
		nc := *n
		return &nc
	case *interiorNode[T]:
		nc := *n
		return &nc
	case nil:
		return nil
	default:
		panic("unreachable")
	}
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
	// they can be reused between nodes. Nil when all children are leftwise
	// dense.
	cumSums *[maxBranch]uint
	// children are either *[interiorNode] or *[leafNode].
	//
	// OPT(dh): we could use unsafe.Pointer instead of an interface, saving
	// some storage and some instructions on use. We can infer the type from
	// the node's level. Of course it'd be a lot less safe.
	children [maxBranch]node[T]
	n        uint8
}

type slotAndIndex struct {
	slot uint8
	idx  uint
}

func (n *interiorNode[T]) traverse(idx uint, shift uint8) iter.Seq2[slotAndIndex, node[T]] {
	// Part of the reason why we return an iterator instead of accepting a
	// callback is that iterators have more generous inlining budgets.
	return func(yield func(slotAndIndex, node[T]) bool) {
		for {
			slot := (idx >> shift) % maxBranch
			if n.cumSums != nil {
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
			} else {
				idx -= slot << shift
			}

			n2 := n.children[slot]

			// When shift == maxBranchExp, the next level is the last level and
			// therefore a leaf. All leaves are on the same--the last--level.
			if shift > maxBranchExp {
				n = n2.(*interiorNode[T])
				// Decrease height by one
				shift -= maxBranchExp
				if !yield(slotAndIndex{uint8(slot), idx}, n) {
					return
				}
			} else {
				yield(slotAndIndex{uint8(slot), idx}, n2)
				return
			}
		}
	}
}

func (n *interiorNode[T]) index(idx uint, shift uint8) T {
	for addr, n2 := range n.traverse(idx, shift) {
		if n, ok := n2.(*leafNode[T]); ok {
			return n.values[addr.idx]
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
