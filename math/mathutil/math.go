// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package mathutil

import (
	"math"
	"math/big"
	"reflect"

	"golang.org/x/exp/constraints"
)

func Rescale[T constraints.Float](oldStart, oldEnd, newStart, newEnd, v T) T {
	slope := (newEnd - newStart) / (oldEnd - oldStart)
	output := newStart + slope*(v-oldStart)
	return output
}

func Clamp[T constraints.Integer | constraints.Float](x, minv, maxv T) T {
	return min(max(x, minv), maxv)
}

// Lerp linearly interpolates between integer and float types.
func Lerp[T constraints.Integer | constraints.Float](start, end T, t float64) T {
	switch t {
	case 0:
		return start
	case 1:
		return end
	default:
		if rv := reflect.ValueOf(start); rv.CanInt() || rv.CanUint() {
			return (T(math.Round(float64(start) + float64(end-start)*t)))
		} else {
			return (T(float64(start) + float64(end-start)*t))
		}
	}
}

// ULP returns the [unit of least precision] of x.
//
// [unit of least precision]: https://en.wikipedia.org/wiki/Unit_in_the_last_place
func ULP(x float64) float64 {
	x = math.Abs(x)
	return math.Nextafter(x, math.Inf(1)) - x
}

// ULPDiff returns the absolute difference between f1 and f2 as an integer
// multiple of the [ULP].
//
// Two NaNs have a difference of zero. Two values of opposite sign have 1 added
// to their difference, so that -0 and +0 have a difference of 1.
//
// See [ReferenceULPDiff] for a function that returns the difference with
// fractional precision.
func ULPDiff(f1, f2 float64) uint64 {
	if math.IsNaN(f1) && math.IsNaN(f2) {
		return 0
	}

	if math.Signbit(f1) != math.Signbit(f2) {
		return ULPDiff(0, math.Abs(f1)) + ULPDiff(0, math.Abs(f2)) + 1
	}

	i1 := math.Float64bits(f1)
	i2 := math.Float64bits(f2)

	if i1 > i2 {
		return i1 - i2
	} else {
		return i2 - i1
	}
}

// ReferenceULPDiff returns the absolute difference between ref and b as a
// fractional multiple of the [ULP]. The fractional component arises from ref
// having higher precision than b. Otherwise, this function is identical to
// [ULPDiff].
//
// A nil ref is considered NaN, and two NaNs have a difference of zero.
func ReferenceULPDiff(ref *big.Float, b float64) float64 {
	if ref == nil && math.IsNaN(b) {
		return 0
	} else if ref == nil || math.IsNaN(b) {
		return math.Inf(1)
	}

	fref, _ := ref.Float64()
	iulp := ULPDiff(fref, b)

	if math.IsInf(fref, 0) || math.IsInf(b, 0) {
		return float64(iulp)
	}

	ref = new(big.Float).Abs(ref)
	fref = math.Abs(fref)
	b = math.Abs(b)

	frefDown, _ := ref.SetMode(big.ToNegativeInf).Float64()
	ulpRef := ULP(frefDown)

	diff := new(big.Float).Sub(ref, big.NewFloat(fref))
	fulp := new(big.Float).Quo(diff, big.NewFloat(ulpRef))
	if ref.Cmp(big.NewFloat(b)) == -1 {
		fulp.Neg(fulp)
	}

	ret, _ := fulp.Float64()
	return float64(iulp) + ret
}
