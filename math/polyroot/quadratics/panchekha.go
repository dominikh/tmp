// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
// SPDX-FileAttributionText: https://github.com/racket/math/blob/38ae0f4920de53aa18068ef7841ca285e40d9a9a/math-lib/math/private/number-theory/quadratic.rkt
//
// SPDX-License-Identifier: MIT

package quadratics

import (
	"math"
)

// Panchekha implements Pavel Panchekha's "accurate quadratic formula" as described
// in https://pavpanchekha.com/blog/accurate-quadratic.html and implemented in
// Racket's math library.
//
// # Modifications
//
// The original version may return roots in an unsorted order, while we expect
// them to be sorted in ascending order. Panchekha never claimed that they would be
// sorted so we don't consider this a shortcoming. We have, however, modified
// our implementation to sort the values.
func Panchekha(a, b, c float64) (r1, r2 float64, n int) {
	bDiv2 := b / 2
	sqrt_d := quadraticDiscriminant(a, bDiv2, c)

	switch {
	case a == 0:
		r := -c / b
		return r, math.NaN(), 1
	case math.IsNaN(sqrt_d):
		return math.NaN(), math.NaN(), 0
	case sqrt_d == 0:
		c := -bDiv2 / a
		return c, math.NaN(), 1
	case b < 0:
		r1 := c / (sqrt_d - bDiv2)
		r2 := (sqrt_d - bDiv2) / a
		return min(r1, r2), max(r1, r2), 2
	default:
		r1 := (bDiv2 + sqrt_d) / -a
		r2 := -c / (bDiv2 + sqrt_d)
		return min(r1, r2), max(r1, r2), 2
	}
}

func quadraticDiscriminant(a, b, c float64) float64 {
	absA := math.Abs(a)
	absB := math.Abs(b)
	absC := math.Abs(c)
	sa := a > 0
	sc := c > 0

	x := math.Sqrt(absA) * math.Sqrt(absC)

	if sa == sc {
		// Otherwise we have two cases depending on the sign of a*c
		// In this case a*c is positive and we want sqrt(b^2 - a*c)
		var acDivX, acDivXErr float64
		// Need to compute err(x) ~ (a*b / x - x) / 2
		if (absA > 1) == (x > 1) {
			// In this case do a / x first
			aDivXErr := math.FMA(absA/x, x, -absA) / x
			acDivX = (absA / x) * absC
			acDivXErr = math.FMA(absA/x, -absC, acDivX) + aDivXErr*absC
		} else {
			cDivXErr := math.FMA(absC/x, x, -absC) / x
			acDivX = absA * (absC / x)
			acDivXErr = math.FMA(-absA, absC/x, acDivX) + cDivXErr*a
		}
		// Now we have d* = |b| - sqrt(ac)
		d_ := absB - x - (acDivX-x-acDivXErr)/2
		switch {
		case d_ > 0:
			return math.Sqrt(d_) * math.Sqrt(absB+x)
		case d_ == 0:
			return 0
		default:
			return math.NaN()
		}
	} else {
		// In this case, a*c is negative and we want sqrt(b^2 + a*c)
		if absB > x {
			z := x / absB
			return absB * math.Sqrt(z*z+1.0)
		} else {
			z := absB / x
			return x * math.Sqrt(z*z+1.0)
		}
	}
}
