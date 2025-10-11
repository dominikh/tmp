// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package quadratics

import (
	"math/big"
)

// BigFloat is a copy of [Goualard] that uses [big.Float] instead of float64 for
// its inputs, outputs, and computations. Its main purpose is to produce
// reference values for tests.
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

				// The over-/underflow in -c / b is unavoidable and means we
				// cannot represent the solution.
				r := new(big.Float)
				r.Neg(c)
				r.Quo(r, b)
				return r, nil, 1
			}
		}
	case b.Sign() == 0:
		if c.Sign() == 0 {
			// a != 0, b == c == 0

			// Zero is the only solution
			return new(big.Float), nil, 1
		} else {
			if a.Signbit() == c.Signbit() {
				// The only solutions are two complex numbers. We don't
				// differentiate between "no roots" and "no real roots" and thus
				// simply return nothing. The paper would return (NaN, NaN) to
				// indicate that the two roots aren't real.

				return nil, nil, 0
			} else {
				fracA := new(big.Float)
				fracC := new(big.Float)
				expA := a.MantExp(fracA)
				expC := c.MantExp(fracC)

				ecp := expC - expA
				dM := ecp &^ 1 // dM = floor(ecp/2)*2
				M := dM / 2
				E := ecp & 1 // E = odd(ecp) ? 1 : 0

				c3 := new(big.Float).SetMantExp(fracC, E)
				S := new(big.Float)
				S.Neg(c3)
				S.Quo(S, fracA)
				S.Sqrt(S)
				x1 := new(big.Float).SetMantExp(S, M)
				return new(big.Float).Neg(x1), x1, 2
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
		fracA := new(big.Float)
		fracB := new(big.Float)
		fracC := new(big.Float)
		expA := a.MantExp(fracA)
		expB := b.MantExp(fracB)
		expC := c.MantExp(fracC)
		K := expB - expA
		L := expA - 2*expB
		ecp := expC + L
		if ecp >= -920 && ecp < 995 {
			c2 := new(big.Float).SetMantExp(fracC, ecp)
			// 4*fracA
			tmp := new(big.Float).Add(fracA, fracA)
			tmp.Add(tmp, tmp)
			disc := bigDet2x2(fracB, fracB, tmp, c2)
			switch disc.Sign() {
			case -1:
				// No real solutions. Similar to above, we return nothing and don't
				// differentiate reasons for the lack of real roots.
				return nil, nil, 0
			case 1:
				// y1 := -(2 * c2) / (fracB + math.Copysign(math.Sqrt(disc), b))
				xxx := new(big.Float).Sqrt(disc)
				if xxx.Signbit() != b.Signbit() {
					xxx.Neg(xxx)
				}
				xxx.Add(fracB, xxx)

				y1 := new(big.Float).Add(c2, c2)
				y1.Neg(y1)
				y1.Quo(y1, xxx)

				// y2 := -(fracB + math.Copysign(math.Sqrt(disc), b)) / (2 * fracA)
				y2 := new(big.Float).Neg(xxx)
				y2.Quo(y2, new(big.Float).Add(fracA, fracA))

				x1 := new(big.Float).SetMantExp(y1, K)
				x2 := new(big.Float).SetMantExp(y2, K)
				if x1.Cmp(x2) == -1 {
					return x1, x2, 2
				} else {
					return x2, x1, 2
				}
			default: // disc == 0
				return new(big.Float).SetMantExp(
					new(big.Float).Quo(
						new(big.Float).Neg(fracB),
						new(big.Float).Add(fracA, fracA),
					), K), nil, 1
			}
		} else {
			dM := ecp &^ 1 // dM = floor(ecp/2)*2
			M := dM / 2
			E := ecp & 1 // E = odd(ecp) ? 1 : 0
			c3 := new(big.Float).SetMantExp(fracC, E)
			S := new(big.Float)
			S.Quo(c3, fracA)
			S.Abs(S)
			S.Sqrt(S)
			if ecp < -920 {
				y1 := new(big.Float).Quo(
					new(big.Float).Neg(fracB),
					fracA,
				)
				y2 := new(big.Float).Quo(
					c3,
					new(big.Float).Mul(fracA, y1),
				)
				x1 := new(big.Float).SetMantExp(y1, K)
				x2 := new(big.Float).SetMantExp(y2, dM+K)
				if x1.Cmp(x2) == -1 {
					return x1, x2, 2
				} else {
					return x2, x1, 2
				}
			} else {
				// ecp >= 995
				if a.Signbit() == c.Signbit() {
					// No real solutions. Similar to above, we return nothing and don't
					// differentiate reasons for the lack of real roots.
					return nil, nil, 0
				} else {
					x1 := new(big.Float).SetMantExp(S, M+K)
					x2 := new(big.Float).Neg(x1)
					return x2, x1, 2
				}
			}
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
