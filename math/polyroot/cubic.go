// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package polyroot

import (
	"math"
)

// cubicFirstRootBetween is a special case of [rootsBetweenWithBuffer] that only
// returns the first root it finds. Additionally, for unbounded ranges, it
// applies logic specific to cubics to find a bound when finding the roots of
// the derivative fails.
func (poly *Polynomial) cubicFirstRootBetween(deriv *Polynomial, x0, x1, xError float64) (float64, bool) {
	if r1, r2, n := deriv.quadraticRoots(); n > 0 {
		xa := r1
		xb := xa
		if n > 1 {
			xb = r2
		}
		if math.IsInf(x0, 0) || math.IsInf(x1, 0) {
			// FIXME(dh): this discards one of x0 or x1 if only one of them is inf
			ya := poly.Evaluate(xa)
			if math.Signbit(poly.coeffs[3]) != math.Signbit(ya) {
				yb := poly.Evaluate(xb)
				return findRoot(poly, deriv, xb, math.Inf(1), yb, xError), true
			} else {
				return findRoot(poly, deriv, math.Inf(-1), xa, ya, xError), true
			}
		} else {
			// This loop expands into the same kind of control flow as the highly
			// nested implementation of CubicRoots in Yuksel's code, except that it
			// executes a handful of superfluous comparisons. It is, however, much
			// more readable.
			possibleEndpoints := [3]float64{xa, xb, x1}
			xl := x0
			yl := poly.Evaluate(x0)
			for _, xu := range possibleEndpoints {
				if xu > xl && xu <= x1 {
					yu := poly.Evaluate(xu)
					if math.Signbit(yl) != math.Signbit(yu) {
						return findRoot(poly, deriv, xl, xu, yl, xError), true
					}

					xl = xu
					yl = yu
				}
			}
		}
	} else {
		// If the derivative doesn't have any roots, then the cubic has an
		// inflection point that isn't a stationary point.

		if math.IsInf(x0, 0) || math.IsInf(x1, 0) {
			// The roots of a cubic end with a -c/3d term, so is the idea here
			// that the roots cannot be smaller or larger than that term,
			// depending on the signs involved, and that's how we find one of
			// the missing bounds(?)
			xInf := -poly.coeffs[2] / (poly.coeffs[3] * 3)
			yInf := poly.Evaluate(xInf)
			if !isFin(xInf) {
				return 0, false
			}

			// FIXME(dh): handle cases where only one of x0 or x1 is an infinity
			if math.Signbit(poly.coeffs[3]) != math.Signbit(yInf) {
				return findRoot(poly, deriv, xInf, math.Inf(1), yInf, xError), true
			} else {
				return findRoot(poly, deriv, math.Inf(-1), xInf, yInf, xError), true
			}
		} else {
			y0 := poly.Evaluate(x0)
			y1 := poly.Evaluate(x1)
			if math.Signbit(y0) != math.Signbit(y1) {
				return findRoot(poly, deriv, x0, x1, y0, xError), true
			}
		}
	}
	return 0, false
}
