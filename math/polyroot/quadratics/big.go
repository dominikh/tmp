// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package quadratics

import (
	"math/big"
)

// BigFloat is a solver that works on [big.Float] instead of float64. It uses
// the same logic as [Goualard] for handling edge cases, but uses
// straightforward big math instead of manually manipulating mantissas and
// exponents.
//
// Its main purpose is to produce reference values for tests.
//
// As big.Float doesn't support NaN, BigFloat returns nil big.Floats instead of
// NaN for invalid roots.
func BigFloat(a, b, c *big.Float) (r1, r2 *big.Float, n int) {
	if a == nil || b == nil || c == nil || a.IsInf() || b.IsInf() || c.IsInf() {
		return nil, nil, 0
	}

	switch {
	case a.Sign() == 0:
		if b.Sign() == 0 {
			if c.Sign() == 0 {
				// a == b == c == 0

				// All reals are valid solutions.
				return new(big.Float).SetInf(false), new(big.Float).SetInf(true), 2
			} else {
				// There are no solutions. We indicate this by returning
				// nothing, whereas the paper would return a single NaN.

				// a == b == 0, c != 0
				return nil, nil, 0
			}
		} else {
			if c.Sign() == 0 {
				// a == c == 0, b != 0

				// Zero is the only solution

				return new(big.Float), nil, 1
			} else {
				// a == 0, b != 0, c != 0
				r := new(big.Float)
				r.Neg(c)
				r.Quo(r, b)
				return r, nil, 1
			}
		}
	case b.Sign() == 0:
		if c.Sign() == 0 { // a != 0, b == c == 0

			// Zero is the only solution
			return new(big.Float), nil, 1
		} else { // a != 0, b == 0, c != 0
			if a.Signbit() == c.Signbit() {
				// The only solutions are two complex numbers. We don't
				// differentiate between "no roots" and "no real roots" and thus
				// simply return nothing. The paper would return (NaN, NaN) to
				// indicate that the two roots aren't real.

				return nil, nil, 0
			} else {
				// ±√(−4𝑎𝑐) ∕ 2𝑎
				r1 := new(big.Float).Quo(
					new(big.Float).Sqrt(
						new(big.Float).Mul(big.NewFloat(-4), new(big.Float).Mul(a, c))),
					new(big.Float).Mul(big.NewFloat(2), a))
				r2 := new(big.Float).Neg(r1)
				return r2, r1, 2
			}
		}
	case c.Sign() == 0:
		// a != 0, b != 0, c == 0

		// The over-/underflow in -b / a is unavoidable and means we
		// cannot represent the solution.
		if a.Signbit() == b.Signbit() {
			r1 := new(big.Float).Neg(b)
			r1.Quo(r1, a)
			return r1, new(big.Float), 2
		} else {
			r2 := new(big.Float).Neg(b)
			r2.Quo(r2, a)
			return new(big.Float), r2, 2
		}

	default:
		// Main logic for a != b != c != 0
		//
		// (This is in the default case, not outside the switch, so that the
		// compiler will yell at us if any of the cases have paths where we
		// don't return from the function.)

		// Use −𝑏 ± √(Δ) ∕ 2𝑎 and 2𝑐 ∕ −𝑏 ∓ √(Δ) with the same sign to compute
		// both roots while avoiding catastrophic cancellation.

		tmp := new(big.Float).Mul(big.NewFloat(4), a)
		disc := bigDet2x2(b, b, tmp, c)
		switch disc.Sign() {
		case -1:
			// No real solutions. Similar to above, we return nothing and don't
			// differentiate reasons for the lack of real roots.
			return nil, nil, 0
		case 1:
			sqrtDisc := new(big.Float).Sqrt(disc)
			var comp *big.Float
			if b.Signbit() {
				// −𝑏 − sign(𝑏)×√(Δ) = |𝑏| + √(Δ)
				comp = new(big.Float).Add(
					new(big.Float).Abs(b),
					sqrtDisc)
			} else {
				// −𝑏 − sign(𝑏)×√(Δ) = −𝑏 − √(Δ) = −(𝑏 + √(Δ))
				comp = new(big.Float).Add(b, sqrtDisc)
				comp.Neg(comp)
			}
			// r1 := (2 * c) / comp // citardauq
			r1 := new(big.Float).Quo(
				new(big.Float).Mul(big.NewFloat(2), c),
				comp)
			// r2 := comp / (2 * a) // quadratic
			r2 := new(big.Float).Quo(
				comp,
				new(big.Float).Mul(big.NewFloat(2), a))
			if r1.Cmp(r2) == -1 {
				return r1, r2, 2
			} else {
				return r2, r1, 2
			}
		default: // disc == 0
			r := new(big.Float).Quo(
				b,
				new(big.Float).Mul(
					big.NewFloat(2),
					a))
			r.Neg(r)
			return r, nil, 1
		}
	}
}

// bigDet2x2 computes a*b - c*d.
func bigDet2x2(a, b, c, d *big.Float) *big.Float {
	return new(big.Float).Sub(
		new(big.Float).SetPrec(a.Prec()+b.Prec()).Mul(a, b),
		new(big.Float).SetPrec(c.Prec()+d.Prec()).Mul(c, d),
	)
}
