//go:build go1.27

package persistent

import (
	"bytes"
	"math/bits"
	"slices"
	"testing"
)

func FuzzNewVectorRoundTrip(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		{0},
		{0, 1, 2, 3},
		{0, 1, 2, 3, 4},
		bytesOfLen(15),
		bytesOfLen(16),
		bytesOfLen(17),
		bytesOfLen(63),
		bytesOfLen(64),
		bytesOfLen(65),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		const maxLen = 256
		if len(data) > maxLen {
			data = data[:maxLen]
		}

		want := intsFromBytes(data)
		v := NewVector(want)
		checkVector(t, v, want)
	})
}

func FuzzVectorOps(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		// Append a few values.
		{0, 0, 1, 0, 0, 2, 0, 0, 3, 0, 0, 4},
		// Cross a leaf boundary, then pop back.
		{0, 0, 0, 0, 0, 1, 0, 0, 2, 0, 0, 3, 0, 0, 4, 1, 0, 0},
		// Build by NewVector from an existing model, then compare versions.
		{0, 0, 1, 0, 0, 2, 0, 0, 3, 5, 0, 0, 6, 0, 1},
		// Boundary-heavy append/pop sequence.
		append(bytes.Repeat([]byte{0, 0, 7}, 17), bytes.Repeat([]byte{1, 0, 0}, 3)...),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		const (
			maxOps      = 512
			maxLen      = 256
			maxVersions = 64
		)

		versions := []Vector[int]{{}}
		models := [][]int{nil}
		current := 0

		pc := 0
		next := func() byte {
			if pc >= len(data) {
				return 0
			}
			b := data[pc]
			pc++
			return b
		}

		addVersion := func(v Vector[int], model []int, selector byte) {
			checkVector(t, v, model)
			if len(versions) < maxVersions {
				versions = append(versions, v)
				models = append(models, slices.Clone(model))
				current = len(versions) - 1
				return
			}

			current = int(selector) % maxVersions
			versions[current] = v
			models[current] = slices.Clone(model)
		}

		for opCount := 0; pc < len(data) && opCount < maxOps; opCount++ {
			op := next()
			selector := next()
			arg := next()

			base := current
			if selector != 0 {
				base = int(selector) % len(versions)
			}
			v := versions[base]
			model := models[base]

			switch op % 7 {
			case 0: // Append to an arbitrary existing version.
				if len(model) >= maxLen {
					continue
				}
				model2 := append(slices.Clone(model), int(arg))
				v2 := v.Append(int(arg))
				checkVector(t, v, model) // persistence: the base version is unchanged
				addVersion(v2, model2, selector)

			case 1: // Pop from an arbitrary existing version.
				if len(model) == 0 {
					continue
				}
				model2 := slices.Clone(model[:len(model)-1])
				v2 := v.Pop()
				checkVector(t, v, model)
				addVersion(v2, model2, selector)

			case 2: // Update an arbitrary existing version.
				if len(model) == 0 {
					continue
				}
				idx := int(arg) % len(model)
				value := int(next())
				model2 := slices.Clone(model)
				model2[idx] = value
				v2 := v.Update(idx, value)
				checkVector(t, v, model)
				addVersion(v2, model2, selector)

			case 3: // Get.
				if len(model) == 0 {
					continue
				}
				idx := int(arg) % len(model)
				if got := v.Get(idx); got != model[idx] {
					t.Fatalf("Get(%d) = %d, want %d", idx, got, model[idx])
				}

			case 4: // Switch current version.
				current = base
				checkVector(t, versions[current], models[current])

			case 5: // Rebuild current model via NewVector, creating a different shape/history.
				v2 := NewVector(slices.Clone(model))
				checkVector(t, v, model)
				addVersion(v2, model, selector)

			case 6: // Compare two arbitrary versions.
				other := int(arg) % len(versions)
				want := slices.Equal(models[base], models[other])
				if got := versions[base].EqualFunc(versions[other], intEqual); got != want {
					t.Fatalf("EqualFunc(%d, %d) = %v, want %v", base, other, got, want)
				}
				if got := versions[base].EqualFuncStrict(versions[other], intEqual); got != want {
					t.Fatalf("EqualFuncStrict(%d, %d) = %v, want %v", base, other, got, want)
				}
			}
		}

		for i := range versions {
			checkVector(t, versions[i], models[i])
		}
	})
}

func checkVector(t *testing.T, v Vector[int], want []int) {
	t.Helper()

	if got := v.Length(); got != len(want) {
		t.Fatalf("Length() = %d, want %d", got, len(want))
	}

	if got := slices.Collect(v.All()); !slices.Equal(got, want) {
		t.Fatalf("All() = %v, want %v", got, want)
	}

	var gotLeaves []int
	for leaf := range v.Leaves() {
		gotLeaves = append(gotLeaves, leaf...)
	}
	if !slices.Equal(gotLeaves, want) {
		t.Fatalf("Leaves() flattened = %v, want %v", gotLeaves, want)
	}

	for i, wantElem := range want {
		if got := v.Get(i); got != wantElem {
			t.Fatalf("Get(%d) = %d, want %d", i, got, wantElem)
		}
	}

	rebuilt := NewVector(slices.Clone(want))
	if !v.EqualFunc(rebuilt, intEqual) {
		t.Fatalf("EqualFunc(NewVector(want)) = false")
	}
	if !rebuilt.EqualFunc(v, intEqual) {
		t.Fatalf("NewVector(want).EqualFunc(v) = false")
	}
	if !v.EqualFuncStrict(rebuilt, intEqual) {
		t.Fatalf("EqualFuncStrict(NewVector(want)) = false")
	}

	checkDenseInvariants(t, v, len(want))
}

func checkDenseInvariants(t *testing.T, v Vector[int], length int) {
	t.Helper()

	if v.rrb {
		// The fuzz tests only construct vectors via dense operations.
		t.Fatalf("vector is unexpectedly not dense")
	}
	if v.n != uint(length) {
		t.Fatalf("v.n = %d, want %d", v.n, length)
	}
	if length == 0 {
		if v.root != nil {
			t.Fatalf("empty vector has non-nil root %T", v.root)
		}
		if v.shift != 0 {
			t.Fatalf("empty vector shift = %d, want 0", v.shift)
		}
		return
	}

	wantShift := denseShiftForLen(length)
	if v.shift != wantShift {
		t.Fatalf("shift = %d, want %d for length %d", v.shift, wantShift, length)
	}
	if got := checkDenseNode(t, v.root, v.shift); got != uint(length) {
		t.Fatalf("tree stores %d values, want %d", got, length)
	}
}

func checkDenseNode(t *testing.T, n node[int], shift uint8) uint {
	t.Helper()

	if shift == 0 {
		leaf, ok := n.(*leafNode[int])
		if !ok {
			t.Fatalf("node at shift 0 has type %T, want *leafNode[int]", n)
		}
		if leaf.n == 0 || leaf.n > maxBranch {
			t.Fatalf("leaf.n = %d, want 1..%d", leaf.n, maxBranch)
		}
		return uint(leaf.n)
	}

	in, ok := n.(*interiorNode[int])
	if !ok {
		t.Fatalf("node at shift %d has type %T, want *interiorNode[int]", shift, n)
	}
	if in.cumSums != nil {
		t.Fatalf("dense node at shift %d has non-nil cumSums", shift)
	}

	var total uint
	var nonNil uint8
	seenNil := false
	for i, child := range in.children {
		if child == nil {
			seenNil = true
			continue
		}
		if seenNil {
			t.Fatalf("dense node at shift %d has non-left-packed child at slot %d", shift, i)
		}
		nonNil++
		total += checkDenseNode(t, child, shift-maxBranchExp)
	}
	if nonNil == 0 || nonNil > maxBranch {
		t.Fatalf("interior.n/non-nil child count = %d, want 1..%d", nonNil, maxBranch)
	}
	if in.n != nonNil {
		t.Fatalf("interior.n = %d, want %d", in.n, nonNil)
	}
	return total
}

func denseShiftForLen(length int) uint8 {
	if length <= 1 {
		return 0
	}
	return uint8(((bits.Len(uint(length-1)) - 1) / maxBranchExp) * maxBranchExp)
}

func intsFromBytes(data []byte) []int {
	out := make([]int, len(data))
	for i, b := range data {
		out[i] = int(b)
	}
	return out
}

func bytesOfLen(n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(i)
	}
	return out
}

func intEqual(a, b int) bool {
	return a == b
}
