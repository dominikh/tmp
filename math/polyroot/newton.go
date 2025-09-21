// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package polyroot

import (
	"fmt"
	"math"
)

func findRoot(
	poly *Polynomial,
	deriv *Polynomial,
	x0 float64,
	x1 float64,
	yl float64,
	xError float64,
) (ret float64) {
	// If one of the ends is open we use a modified algorithm to find a finite
	// end before we switch to our main algorithm. Callers of findRoot guarantee
	// that only one end is open.
	if x0Inf, x1Inf := math.IsInf(x0, 0), math.IsInf(x1, 0); x0Inf || x1Inf {
		if x0Inf && x1Inf {
			// We only allow one end to be open. The other end should've been
			// determined by this point. That can fail, however, for example if
			// a calculation overflowed.
			return math.NaN()
		}
		if math.IsNaN(x0) || math.IsNaN(x1) {
			panic(fmt.Sprintf("internal error: x0=%g x1=%g", x0, x1))
		}
		isOpenMin := x0Inf
		var xl float64
		if isOpenMin {
			xl = x1
		} else {
			xl = x0
		}
		yl_ := yl

		// Taking big jumps will often lead to finding the missing bound after
		// one iteration, letting us switch to the more efficient algorithm
		// sooner.
		delta := max(math.Abs(xl), 1)
		if isOpenMin {
			delta = -delta
		}
		xr := xl + delta
		yr := poly.Evaluate(xr)

		for {
			if yr == 0 {
				return xr
			}

			if math.Signbit(yl_) != math.Signbit(yr) {
				// We've found a second limit and can switch to the algorithm for
				// bounded intervals.
				if isOpenMin {
					x0, x1, yl = xr, xl, yr
				} else {
					x0, x1, yl = xl, xr, yl_
				}
				break
			}

			// We're still on the same side of the root, but closer to it, which
			// means we can tighten our bounds.
			xl = xr
			yl_ = yr
			dyr := deriv.Evaluate(xr)

			dx := yr / dyr
			xn := xr - dx

			dirOk := isOpenMin && dx >= 0 && !isOpenMin && dx <= 0
			if dirOk && isFin(xn) {
				// Valid Newton step
				xr = xn
				if math.Abs(dx) <= xError {
					// We might have converged
					if xr == xl {
						return xr
					}
					xs := xn
					if isOpenMin {
						xs += xError
					} else {
						xs -= xError
					}
					ys := poly.Evaluate(xs)
					if math.Signbit(yl_) != math.Signbit(ys) {
						return xr
					}
					xr = xs
					yr = ys
				} else {
					yr = poly.Evaluate(xr)
				}
			} else {
				// Newton step failed
				xr += delta
				delta *= 2
				if math.IsInf(xr, 0) {
					return math.NaN()
				}
				yr = poly.Evaluate(xr)
			}
		}
	}

	// If called without open bounds, or after resolving an open bound, yl is
	// always poly.Evaluate(x0)
	y0 := yl

	// At least one of x0 and x1 tends to be a critical point of the polynomial.
	// If the polynomial has a double root, that root will be a critical point.
	// Check if we're dealing with a critical point that happens to be a root
	// and return early.
	//
	// This is not just a performance optimization but also tends to yield more
	// accurate results, especially as higher degree polynomials accrue more
	// error. For example, the true root of f(x) = 60480*x^3 - 362880*x^2 +
	// 725760*x - 483840 is 2, but even using the compensated Horner scheme,
	// f(1.9999999999677796) also evaluates to 0 and would be the root found
	// iteratively.
	if y0 == 0 {
		return x0
	} else if y1 := poly.Evaluate(x1); y1 == 0 {
		return x1
	}

	step := (x1 - x0) / 2.0
	xr := x0 + step

	if math.Abs(step) <= xError {
		return xr
	}

	// xb0 and xb1 are the bounds that will be tightened as we iterate while x0
	// and x1 will remain as the original bounds. The original bounds aren't
	// actually needed past this point, but by not reusing their names it is
	// clear that y0 is the y value for the original x0, not the updated xb0.
	xb0 := x0
	xb1 := x1

	// In Cem Yuksel's paper and implementation, there is a fixed number of
	// iterations of Newton's method that don't have the convergence guarantee
	// of the Newton+bisection approach, as it is supposed to be faster. In
	// testing we couldn't find this to be true.

	for j := uint(1); ; j++ {
		yr := poly.Evaluate(xr)
		dyr := deriv.Evaluate(xr)

		if yr == 0 {
			// Returning early should prevent xn from becoming NaN.
			//
			// TODO(dh): can yr != 0 && dyr == 0? Then xn can still become NaN
			//
			// TODO(dh): by returning early we skip over possible rounds of
			// bisection and xr won't be as precise as it could be.
			return xr
		}

		// Shrink the range from one end. If the new end is the same as the old
		// end, then we've run out of precision and are done.
		//
		// TODO(dh): make sure that only happens when xb0 and xb1 are adjacent.
		if math.Signbit(y0) != math.Signbit(yr) {
			if xb1 == xr {
				return xr
			}
			xb1 = xr
		} else {
			if xb0 == xr {
				return xr
			}
			xb0 = xr
		}

		if j%16 == 0 {
			// We've run a fair number of steps and still aren't done. Throw in
			// a round of bisection in case Newton is stuck progressing very
			// slowly.

			// OPT(dh): instead of doing this sporadically, is there a way to
			// determine which method gets closer to the result faster?
			xr = (xb0 + xb1) / 2
			continue
		}

		xn := xr - (yr / dyr)

		// There are two ways for the Newton step to fail: the new value might
		// be outside the existing bounds, or yr / dyr might be NaN due to
		// infinities. In both cases we fall back to bisection.
		if !(xn >= xb0 && xn <= xb1) {
			// TODO(dh): can't it theoretically happen that Newton keeps jumping
			// between xb0 and xb1, even when the difference between xb0 and xb1
			// is larger than the precision, and thus we should check for > and
			// < instead of >= and <=? but then how will that affect roots
			// falling on exactly the boundary
			xn = (xb0 + xb1) / 2
		}
		step := math.Abs(xr - xn)
		xr = xn

		if !(step > xError) {
			return xr
		}
	}
}
