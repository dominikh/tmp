// SPDX-FileCopyrightText: 2025 the Kurbo Authors
// SPDX-FileCopyrightText: 2025 Joe Neeman
// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package quadratics

import (
	"math"
)

// Yuksel implements Cem Yuksel's quadratics solver as used by his
// [High-Performance Polynomial Solver].
//
// [High-Performance Polynomial Solver]: https://www.cemyuksel.com/research/polynomials/
func Yuksel(a, b, c float64) (r1, r2 float64, n int) {
	delta := b*b - 4*a*c
	if delta > 0 {
		d := math.Sqrt(delta)
		q := -0.5 * (b + math.Copysign(d, b))
		rv0 := q / a
		rv1 := c / q
		return min(rv0, rv1), max(rv0, rv1), 2
	} else if delta < 0 {
		return math.NaN(), math.NaN(), 0
	}
	r := -0.5 * b / a
	n = 0
	if a != 0 {
		n = 1
	}
	return r, math.NaN(), n
}

// Yuksel2 implements an improved variant of Yuksel as found in [poly-cool] (as of 2025-10).
//
// [poly-cool]: https://github.com/jneem/poly-cool
func Yuksel2(a, b, c float64) (r1, r2 float64, n int) {
	disc := b*b - 4*a*c

	if !isFin(disc) {
		// At least one of the coefficients was too large and triggered
		// overflow.
		if !isFin(a) || !isFin(b) || !isFin(c) {
			// If we're infinite, just give up. (Otherwise, we'd stack overflow
			// by repeatedly trying to rescale.)
			return math.NaN(), math.NaN(), 0
		} else {
			// The exponent of f64 maxes out at 1023, so scaling down by
			// 2^{-512} is enough to ensure that squaring doesn't overflow. We
			// do an extra factor of 2^{-3} for some wiggle room. This can't
			// completely destroy all the coefficients: because of the overflow,
			// we know that at least one of them was big.
			scale := math.Pow(2, -515)
			return Yuksel(scale*a, scale*b, scale*c)
		}
	} else {
		if disc > 0.0 {
			q := -0.5 * (b + math.Copysign(math.Sqrt(disc), b))
			r0 := q / a
			r1 := c / q
			if isFin(r0) {
				if isFin(r1) {
					return min(r0, r1), max(r0, r1), 2
				} else {
					return r0, math.NaN(), 1
				}
			} else if isFin(r1) {
				return r1, math.NaN(), 1
			} else {
				return math.NaN(), math.NaN(), 0
			}
		} else if disc == 0.0 {
			root := -0.5 * b / a
			if isFin(root) {
				return root, math.NaN(), 1
			} else if c == 0.0 {
				// This is kurbo's behavior: the intention is that if the
				// whole thing is zero, return zero as a single root. I'm
				// not sure I love it.
				//
				// Bear in mind that this branch is not *only* for the
				// identically zero case: if a == c == 0.0 and b * b
				// underflows then we will end up here. In that case,
				// zero is the only root.
				return 0, math.NaN(), 1
			} else {
				return math.NaN(), math.NaN(), 0
			}
		} else {
			// No roots.
			return math.NaN(), math.NaN(), 0
		}
	}
}

// Yuksel3 implements an improved version of Yuksel2 that uses Kahan's method of
// computing the discriminant more robustly.
func Yuksel3(a, b, c float64) (r1, r2 float64, n int) {
	disc := discriminant(a, b, c)

	if !isFin(disc) {
		// At least one of the coefficients was too large and triggered
		// overflow.
		if !isFin(a) || !isFin(b) || !isFin(c) {
			// If we're infinite, just give up. (Otherwise, we'd stack overflow
			// by repeatedly trying to rescale.)
			return math.NaN(), math.NaN(), 0
		} else {
			// The exponent of f64 maxes out at 1023, so scaling down by
			// 2^{-512} is enough to ensure that squaring doesn't overflow. We
			// do an extra factor of 2^{-3} for some wiggle room. This can't
			// completely destroy all the coefficients: because of the overflow,
			// we know that at least one of them was big.
			scale := math.Pow(2, -515)
			return Yuksel(scale*a, scale*b, scale*c)
		}
	} else {
		if disc > 0.0 {
			q := -0.5 * (b + math.Copysign(math.Sqrt(disc), b))
			r0 := q / a
			r1 := c / q
			if isFin(r0) {
				if isFin(r1) {
					return min(r0, r1), max(r0, r1), 2
				} else {
					return r0, math.NaN(), 1
				}
			} else if isFin(r1) {
				return r1, math.NaN(), 1
			} else {
				return math.NaN(), math.NaN(), 0
			}
		} else if disc == 0.0 {
			root := -0.5 * b / a
			if isFin(root) {
				return root, math.NaN(), 1
			} else if c == 0.0 {
				// This is kurbo's behavior: the intention is that if the
				// whole thing is zero, return zero as a single root. I'm
				// not sure I love it.
				//
				// Bear in mind that this branch is not *only* for the
				// identically zero case: if a == c == 0.0 and b * b
				// underflows then we will end up here. In that case,
				// zero is the only root.
				return 0, math.NaN(), 1
			} else {
				return math.NaN(), math.NaN(), 0
			}
		} else {
			// No roots.
			return math.NaN(), math.NaN(), 0
		}
	}
}
