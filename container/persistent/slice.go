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
	maxBranchExp   = 5 // 2 during debugging, 5 in production

	// maxBranch must be 2**maxBranchExp.
	maxBranch = 1 << maxBranchExp
)

var _ [maxSearchError]struct{} // maxSearchError >= 0

type Vector[T any] struct {
	// Root of the trie. May be nil.
	root *interiorNode[T]
	// tail points to the trailing elements of the vector. Guaranteed to be
	// non-nil when the vector is non-empty.
	tail     *leafNode[T]
	metadata uint64
}

func shiftAndTreeN(shift uint8, treeN uint64) uint64 {
	return (treeN << 8) | uint64(shift)
}

// The number of values stored in the tree rooted at root. The total length
// of the vector is treeN + tail.n.
func (v Vector[T]) treeN() uint64 {
	return uint64(v.metadata >> 8)
}

// shift is maxBranchExp * height. A vector of height h can store up to
// maxBranch<<shift (= maxBranch * 2**shift = 2**(maxBranchExp*(h+1))
// elements. h is zero-based, a vector with a single leaf node has height
// 0 (the leaf is stored in the tail).
func (v Vector[T]) shift() uint8 {
	// TODO(dh): we don't need 8 bits for s hift
	return uint8(v.metadata & 0xFF)
}

func (v Vector[T]) rrb() bool {
	return false
}

func NewVector[T any](values []T) Vector[T] {
	if len(values) == 0 {
		return Vector[T]{}
	}

	// rem = len(values) % maxBranch
	// if rem == 0 { rem = maxBranch }
	rem := (len(values)-1)%maxBranch + 1
	tailValues := values[len(values)-rem:]
	tail := &leafNode[T]{
		n: uint8(len(tailValues)),
	}
	copy(tail.values[:], tailValues)

	values = values[:len(values)-len(tailValues)]
	if len(values) == 0 {
		return Vector[T]{
			tail: tail,
		}
	}

	// Highest index is n-1. Each trie level consumes maxBranchExp bits.
	depth := ((bits.Len(uint(len(values)-1)) - 1) / maxBranchExp)
	// We don't let root be a leaf node, so depth is at least 1.
	shift := max(1, depth) * maxBranchExp

	return Vector[T]{
		root:     buildTrie(values, shift).(*interiorNode[T]),
		tail:     tail,
		metadata: shiftAndTreeN(uint8(shift), uint64(len(values))),
	}
}

func buildTrie[T any](elems []T, shift int) node[T] {
	childWidth := 1 << shift

	if shift == 0 {
		leaf := &leafNode[T]{n: uint8(len(elems))}
		copy(leaf.values[:], elems)
		return leaf
	}

	n := &interiorNode[T]{}
	for i := 0; i < len(elems); i += childWidth {
		end := min(i+childWidth, len(elems))
		n.children[n.n] = buildTrie(elems[i:end], shift-maxBranchExp)
		n.n++
	}

	return n
}

func (v Vector[T]) Append(value T) Vector[T] {
	if !v.rrb() {
		if v.Length() == 0 {
			// Very fast path
			return Vector[T]{
				tail: &leafNode[T]{
					n:      1,
					values: [maxBranch]T{value},
				},
			}
		}

		if v.tail.n < maxBranch {
			vc := v
			// There is still room in the tail
			vc.tail = clone(vc.tail)
			vc.tail.values[vc.tail.n] = value
			vc.tail.n++
			return vc
		}

		// The tail is full. Insert it into the tree, then create a new tail.

		vc := v
		if v.root == nil {
			// Until now we only had a tail. Create the tree.
			vc.metadata = shiftAndTreeN(maxBranchExp, maxBranch)
			vc.root = &interiorNode[T]{
				n:        1,
				children: [maxBranch]node[T]{v.tail},
			}
			vc.tail = &leafNode[T]{
				n:      1,
				values: [maxBranch]T{value},
			}
			return vc
		}

		if full := maxBranch<<v.shift() == v.treeN(); full {
			// The tree is dense and full at its current depth. Insert a
			// new interior node at the top to hang new nodes off of.
			vc.root = &interiorNode[T]{
				n:        1,
				children: [maxBranch]node[T]{v.root},
			}
			vc.incShift()
		} else {
			vc.root = clone(v.root)
		}

		vc.metadata = shiftAndTreeN(vc.shift(), vc.treeN()+maxBranch)

		t := node[T](vc.root)
		for shift := vc.shift(); shift >= maxBranchExp; shift -= maxBranchExp {
			i := (v.treeN() >> shift) % maxBranch
			child := cloneNode(t.(*interiorNode[T]).children[i])
			if shift == maxBranchExp && child != nil {
				panic("unreachable")
			}
			if child == nil {
				if shift == maxBranchExp {
					child = v.tail
				} else {
					child = &interiorNode[T]{}
				}
				t.(*interiorNode[T]).n++
			}
			t.(*interiorNode[T]).children[i] = child
			t = child
		}
		if t != v.tail {
			panic("unreachable")
		}

		vc.tail = &leafNode[T]{
			n:      1,
			values: [maxBranch]T{value},
		}
		return vc
	} else {
		// XXX implement as a concatenation. Maybe we won't have to now that we
		// use the tail?
		return Vector[T]{}
	}
}

func clone[E any, P *E](ptr P) P {
	if ptr == nil {
		return nil
	}
	x := *ptr
	return &x
}

func (v Vector[T]) Pop() Vector[T] {
	if v.Length() == 0 {
		panic("tried to pop from empty vector")
	}

	if v.Length() == 1 {
		return Vector[T]{}
	}

	if v.tail.n > 1 {
		vc := v
		vc.tail = clone(vc.tail)
		vc.tail.n--
		vc.tail.values[vc.tail.n] = *new(T)
		return vc
	} else {
		// The tail will be empty after popping. Since the tail mustn't be
		// empty, we promote the rightmost leaf.

		if !v.rrb() {
			vc := v

			if vc.shift() < maxBranchExp {
				panic("internal error: shift smaller than expected")
			}

			var dfs func(n *interiorNode[T], shift uint8) node[T]
			dfs = func(n *interiorNode[T], shift uint8) node[T] {
				n = clone(n)
				ic := ((v.treeN() - 1) >> shift) % maxBranch
				if shift == maxBranchExp {
					vc.tail = n.children[ic].(*leafNode[T])
					vc.metadata = shiftAndTreeN(vc.shift(), vc.treeN()-uint64(vc.tail.n))
					if ic == 0 {
						assert(n.n == 1)
						return nil
					} else {
						n.n--
						n.children[ic] = nil
						return n
					}
				} else {
					new := dfs(n.children[ic].(*interiorNode[T]), shift-maxBranchExp)
					if ic == 0 && new == nil {
						assert(n.n == 1)
						return nil
					} else {
						n.children[ic] = new
						if new == nil {
							n.n--
						}
						return n
					}
				}
			}
			if nroot := dfs(vc.root, vc.shift()); nroot != nil {
				vc.root = nroot.(*interiorNode[T])
				if vc.root.n == 1 && is[*interiorNode[T]](vc.root.children[0]) {
					vc.decShift()
					vc.root = vc.root.children[0].(*interiorNode[T])
				}
			} else {
				// Only the tail is left
				vc.metadata = shiftAndTreeN(0, 0)
			}
			return vc
		} else {
			// XXX implement
			panic("unimplemented")
		}
	}
}

func (v Vector[T]) String() string {
	var sb strings.Builder
	sb.WriteString("Vector[")
	first := true
	for e := range v.All() {
		if !first {
			sb.WriteString(" ")
		}
		first = false
		fmt.Fprint(&sb, e)
	}
	sb.WriteString("]")
	return sb.String()
}

func is[T any](v any) bool {
	_, ok := v.(T)
	return ok
}

func assert(b bool) {
	if !b {
		panic("failed assertion")
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
	if v.root == vo.root && v.tail == vo.tail {
		return true
	}

	// Both roots are interior nodes.
	pv := newPreorder(v.root, v.tail)
	pvo := newPreorder(vo.root, v.tail)

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

func newPreorder[T any](n *interiorNode[T], tail *leafNode[T]) *preorder[T] {
	if n == nil {
		if tail == nil {
			return &preorder[T]{}
		}

		return &preorder[T]{
			stack: []preorderStackEntry[T]{{
				// We create an artificial interior node to hold the tail. The caller
				// will never see the interior node, because we set curChildIdx
				// to 0, so that preorder.current will return the tail.
				node: &interiorNode[T]{
					n:        1,
					children: [maxBranch]node[T]{tail},
				},
				curChildIdx: 0,
			}},
		}
	}

	return &preorder[T]{
		stack: []preorderStackEntry[T]{{n, -1}},
		tail:  tail,
	}
}

type preorderStackEntry[T any] struct {
	node        *interiorNode[T]
	curChildIdx int
}

type preorder[T any] struct {
	stack []preorderStackEntry[T]
	tail  *leafNode[T]
}

func (p *preorder[T]) current() node[T] {
	// OPT(dh): instead of all this logic, set a field when the current node
	// changes.

	if len(p.stack) == 0 {
		return nil
	}

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
		return false
	}

	cur := &p.stack[len(p.stack)-1]
	if len(p.stack) == 1 && cur.curChildIdx >= len(cur.node.children) {
		// No children left in the tree
		if p.tail == nil {
			return false
		}

		// We create an artificial interior node to hold the tail. The
		// caller will never see the interior node, because we immediately
		// increment the child index to zero.
		p.stack = append(p.stack, preorderStackEntry[T]{
			node: &interiorNode[T]{
				n:        1,
				children: [maxBranch]node[T]{p.tail},
			},
			curChildIdx: -1,
		})
		p.tail = nil
		cur = &p.stack[len(p.stack)-1]
	}
	cur.curChildIdx++
	for cur.curChildIdx < len(cur.node.children) && cur.node.children[cur.curChildIdx] == nil {
		cur.curChildIdx++
	}
	if cur.curChildIdx >= len(cur.node.children) {
		if p.ascend() || p.tail != nil {
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
	// XXX handle tail

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
	if v.tail == nil {
		return int(v.treeN())
	} else {
		return int(v.treeN() + uint64(v.tail.n))
	}
}

func (v Vector[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		_ = v.root.every(yield) && v.tail.every(yield)
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
	if v.Length() == 0 {
		return func(yield func(*leafNode[T]) bool) {}
	}

	if v.root == nil {
		return func(yield func(*leafNode[T]) bool) {
			yield(v.tail)
		}
	}

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
		assert(v.tail != nil)
		_ = dfs(v.root) && yield(v.tail)
	}
}

func (v Vector[T]) Get(idx int) T {
	if idx < 0 {
		panic(fmt.Sprintf("index out of range [%d]", idx))
	}
	if uint(idx) >= uint(v.Length()) {
		panic(fmt.Sprintf("index out of range [%d] with length %d", idx, v.treeN))
	}

	if v.isInTail(uint(idx)) {
		return v.tail.values[uint64(idx)-v.treeN()]
	}

	return v.root.index(uint(idx), v.shift())
}

func (v Vector[T]) Update(i int, value T) Vector[T] {
	if i < 0 {
		panic(fmt.Sprintf("index out of range [%d]", i))
	}
	if uint64(i) >= v.treeN() {
		panic(fmt.Sprintf("index out of range [%d] with length %d", i, v.treeN))
	}

	// XXX handle tail

	rootc := *v.root
	parent := &rootc
	for addr, n := range v.root.traverse(uint(i), v.shift()) {
		switch n := n.(type) {
		case *interiorNode[T]:
			nc := clone(n)
			parent.children[addr.slot] = nc
			parent = nc
		case *leafNode[T]:
			nc := clone(n)
			nc.values[addr.idx] = value
			parent.children[addr.slot] = nc
		}
	}

	return Vector[T]{
		root:     &rootc,
		metadata: v.metadata,
	}
}

//lint:ignore U1000 Debug helper
func (v Vector[T]) dot(name string, w io.Writer) {
	fmt.Fprintln(w, "strict digraph {")
	fmt.Fprintf(w, "v%s [label=%q, shape=box];\n", name, name)

	printLeaf := func(n *leafNode[T], id string) {
		values := make([]string, len(n.values[:n.n]))
		for i := range values {
			values[i] = fmt.Sprint(n.values[i])
		}
		// FIXME(dh): this is buggy for value representations containing
		// pipes or angled brackets, among others.
		fmt.Fprintf(w, "%s [shape=record, label=%q];\n", id, strings.Join(values, "|"))
	}
	var dfs func(pid string, n node[T])
	dfs = func(pid string, n node[T]) {
		switch n := n.(type) {
		case *interiorNode[T]:
			var labels []string
			if n == nil {
				// Malformed tree
				id := fmt.Sprintf("inil%s", pid)
				fmt.Fprintf(w, "%s [label=nil, color=red, shape=box];\n", id)
				fmt.Fprintf(w, "%s -> %s;\n", pid, id)
			} else {
				id := fmt.Sprintf("i%p", n)
				for i := range n.children {
					labels = append(labels, fmt.Sprintf("<f%d>", i))
				}
				label := strings.Join(labels, "|")
				fmt.Fprintf(w, "%s [shape=record, label=%q];\n", id, label)
				fmt.Fprintf(w, "%s -> %s;\n", pid, id)
				for i, child := range n.children {
					dfs(fmt.Sprintf("%s:f%d", id, i), child)
				}
			}
		case *leafNode[T]:
			id := fmt.Sprintf("l%p", n)
			printLeaf(n, id)
			fmt.Fprintf(w, "%s -> l%p;\n", pid, n)
		case nil:
		}
	}
	if v.root != nil {
		dfs("v"+name, v.root)
	}
	if v.tail != nil {
		tailID := fmt.Sprintf("l%p", v.tail)
		printLeaf(v.tail, tailID)
		fmt.Fprintf(w, "v%s -> %s [style=dashed];\n", name, tailID)
	}
	fmt.Fprintln(w, "}")
}

func (v Vector[T]) isInTail(idx uint) bool {
	return uint64(idx) >= v.treeN()
}

func (v *Vector[T]) incShift() {
	nshift := v.shift() + maxBranchExp
	if nshift < v.shift() {
		// This is impossible to hit unless there is a bug or memory corruption.
		// v.shift is 'maxBranchExp * tree height' and a uint8. The max height
		// of the tree is ln(2**64) / ln(2**maxBranchExp) and the max value of
		// v.shift therefore '(ln(2**64) / ln(2**maxBranchExp)) * maxBranchExp'.
		// This simplifies to 64.

		// TODO(dh): can RRB introduce additional levels?

		panic("internal error: increasing tree height would overflow")
	}
	v.metadata = shiftAndTreeN(nshift, v.treeN())
}

func (v *Vector[T]) decShift() {
	if v.shift() < maxBranchExp {
		// This should clearly not happen.
		panic("internal error: decreasing tree height would overflow")
	}
	v.metadata = shiftAndTreeN(maxBranchExp, v.treeN())
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
	if n == nil {
		return true
	}

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
	if n == nil {
		return true
	}

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
