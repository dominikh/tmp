// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
// SPDX-FileAttributionText: doi:10.22541/au.168635343.38524892/v1
//
// SPDX-License-Identifier: MIT

package quadratics

import (
	"math"
)

// Goualard implements Frédéric Goualard's quadratic equation solver as
// described in "The Ins and Outs of Solving Quadratic Equations with
// Floating-Point Arithmetic", which describes it as a modern implementation of
// Pat Sterbenz's exposition from 1974.
//
// This is currently the most robust solver in this package that operates in
// 64-bit precision. It handles all possible inputs, including NaN, infinities,
// denormals, and zeros. If roots fit in float64 they will be found, even if
// intermediary computations would over- or underflow if executed naively.
//
// For 30 runs of 10 million randomly generated quadratic equations spanning the
// whole range of (non-NaN, non-∞) floating point numbers, 95.84% of roots were
// correct to within 0.5 ULP, 99.97% to within 1.5 ULP, and 99.99999% to within
// 2.5 ULP. 0.00001% of roots were wrong by 3 to 3.5 ULP. Run to run variance
// was negligible.
func Goualard(a, b, c float64) (r1, r2 float64, n int) {
	if !isFin(a) || !isFin(b) || !isFin(c) {
		return math.NaN(), math.NaN(), 0
	}

	// In several places, the paper uses a "keep_exponent_in_check" function
	// that splits an exponent 𝑀 into 𝑀 = 𝑀₁ + 𝑀₂, so that 𝑥 × 2^𝑀 can instead
	// be computed as 2^𝑀₁ × (𝑥 × 2^𝑀₂) so that 2^𝑀 doesn't over- or
	// underflow. For example, when x is small, none of the multiplications in
	// the new equation will overflow even if in 𝑥 × 2^𝑀, 2^𝑀 would've
	// overflowed.
	//
	// We instead use math.Ldexp(x, 𝑀), which computes 𝑥 × 2^𝑀 while taking
	// care of over- and underflow for us. Internally it does logic quite
	// similar to the paper's, working on the mantissa and exponent separately.

	// | a | b | c | formula                               |
	// |---+---+---+---------------------------------------|
	// | 0 | 0 | 0 | ℝ                                     |
	// | 0 | 0 | 1 | ∅                                    |
	// | 0 | 1 | 0 | 0                                     |
	// | 0 | 1 | 1 | -c/b                                  |
	// | 1 | 0 | 0 | 0                                     |
	// | 1 | 0 | 1 | ±√(−4ac) ∕ 2a if sign(a) ≠ sign(c) |
	// | 1 | 1 | 0 | −b∕a, 0                               |
	// | 1 | 1 | 1 | Quadratic formula                     |
	switch {
	case a == 0:
		if b == 0 {
			if c == 0 { // a == 0, b == 0, c == 0
				// All reals are valid solutions.
				return math.Inf(1), math.Inf(-1), 2
			} else { // a == 0, b == 0, c != 0
				// There are no solutions.
				return math.NaN(), math.NaN(), 0
			}
		} else {
			if c == 0 { // a == 0, b != 0, c == 0
				// Zero is the only solution
				return 0, math.NaN(), 1
			} else { // a == 0, b != 0, c != 0
				// The potential over-/underflow in -c / b is unavoidable and
				// means we cannot represent the solution.
				return -c / b, math.NaN(), 1
			}
		}
	case b == 0:
		if c == 0 { // a != 0, b == 0, c == 0
			// Zero is the only solution
			return 0, math.NaN(), 1
		} else { // a != 0, b == 0, c != 0
			if math.Signbit(a) == math.Signbit(c) {
				// The only solutions are two complex numbers. We don't
				// differentiate between "no roots" and "no real roots" and thus
				// simply return nothing.
				return math.NaN(), math.NaN(), 0
			} else {
				fracA, expA := math.Frexp(a)
				fracC, expC := math.Frexp(c)

				ecp := expC - expA
				dM := ecp &^ 1 // dM = floor(ecp/2)*2
				M := dM / 2
				c3 := fracC
				if ecp&1 == 1 {
					c3 *= 2
				}
				S := math.Sqrt(-c3 / fracA)
				x1 := math.Ldexp(S, M)
				return -x1, x1, 2
			}
		}
	case c == 0: // a != 0, b != 0, c == 0
		// The potential over-/underflow in −𝑏 ∕ 𝑎 is unavoidable and means we
		// cannot represent the solution.
		r := -b / a
		if math.Signbit(a) == math.Signbit(b) {
			return r, 0, 2
		} else {
			return 0, r, 2
		}

	default: // a != 0, b != 0, c != 0
		// (This is in the default case, not outside the switch, so that the
		// compiler will yell at us if any of the cases have paths where we
		// don't return from the function.)

		// Please see §5.4 of "The ins and outs of solving quadratic equations
		// with floating-point arithmetic" by Goualard for the detailed
		// derivation of the math that follows.
		//
		// To summarize: we split the coefficients into their mantissas and
		// exponents (via [math.Frexp]), scale our equation by 2^(expA − 2×expB),
		// substitute 𝑥 = 2^(expB - expA) × 𝑦 and solve the resulting equation
		// instead. In the new equation, only the constant term has potential
		// for underflow and overflow.
		//
		// The final equation: mantA×𝑦² + mantB×𝑦 + mantC×2^(expC + expA − 2×expB) = 0

		// mant, exp = math.Frexp(𝑥) => 𝑥 = mant × 2^exp
		mantA, expA := math.Frexp(a)
		mantB, expB := math.Frexp(b)
		mantC, expC := math.Frexp(c)
		ecp := expC + expA - 2*expB

		const (
			// The smallest (normal) float64 exponent
			Emin = -1022
			// The largest float64 exponent
			Emax = 1023
		)

		const (
			// In the paper, mantissas can be in the range [2^(1-𝑝), 2). As we
			// want to calculate 4 × mantA × mantC × 2^(expC + expA − 2×expB),
			// the exponent has to be within [Emin + 2p − 4, Emax − 3) to
			// avoid underflow and overflow. math.Frexp, however, returns
			// fractions in [0.5, 1), and we can avoid underflow and overflow by
			// staying within [Emin, Emax − 1).

			// The minimum (inclusive) and maximum (exclusive) range for the
			// first case. Case 2 is <case1Min and case 3 is >=case1Max
			case1Min = Emin
			// FIXME(dh): for some reason this doesn't fail even with Emax
			// instead of Emax - 1. why are we off by one?
			case1Max = Emax - 1
		)

		if ecp >= case1Min && ecp < case1Max {
			// No underflow or overflow.

			a2 := mantA
			b2 := mantB
			c2 := math.Ldexp(mantC, ecp)
			// disc = 𝑏′² − 4𝑎′𝑐′
			disc := discriminant(a2, b2, c2)
			if math.IsInf(disc, 0) || math.IsNaN(disc) {
				// This should be impossible if our case1Min and case1Max aren't
				// too wide.
				panic("unreachable")
			}
			switch {
			case disc < 0:
				// No real solutions. Similar to above, we return nothing and don't
				// differentiate reasons for the lack of real roots.
				return math.NaN(), math.NaN(), 0
			case disc > 0:
				// When using −𝑏′ ± √(Δ) ∕ 2𝑎′ to compute both roots, we will
				// encounter catastrophic cancellation whenever 𝑏′ ≈ ±√(Δ).
				//
				// For positive b′, we compute
				// −𝑏′ + √(Δ) = √(Δ) − b′ and
				// −𝑏′ - √(Δ) = −(𝑏′ + √(Δ))
				// the first of which can lead to cancellation.
				//
				// For negative b′, we compute
				// −𝑏′ + √(Δ) = |𝑏′| + √(Δ) and
				// −𝑏′ − √(Δ) = |𝑏′| − √(Δ)
				// the second of which can lead to cancellation.
				//
				// Another formula (from Fagnano's work) for computing the roots is
				// 2𝑐′ ∕ −𝑏′ ∓ √(Δ). This formula has the same problem. However,
				// the two formulas use opposite signs to compute the same root,
				// which means that if we use the same sign with both formulas,
				// we compute both roots. Since we can choose the sign based on
				// the sign of 𝑏′, we can always avoid catastrophic
				// cancellation. This does not require a branch, either, because
				// we can make use of [math.Copysign].
				//
				// For positive b′, we compute −𝑏′ − sign(𝑏′)×√(Δ) = −𝑏′ − √(Δ) = −(𝑏′ + √(Δ))
				// For negative b′, we compute −𝑏′ − sign(𝑏′)×√(Δ) = |𝑏′| + √(Δ)
				comp := -b2 - math.Copysign(math.Sqrt(disc), b2)
				y1 := (2 * c2) / comp // citardauq
				y2 := comp / (2 * a2) // quadratic
				x1 := math.Ldexp(y1, expB-expA)
				x2 := math.Ldexp(y2, expB-expA)
				return min(x1, x2), max(x1, x2), 2
			default: // disc == 0; or NaN, which shouldn't be possible
				return math.Ldexp((-b2 / (2 * a2)), expB-expA), math.NaN(), 1
			}

		} else {
			// m_c 2^(E_c + E_a - 2E_b) = m_c 2^(2M + E) = 2^2M m_c 2^E
			// 𝐸 ∈ {0, 1}

			a2 := mantA
			b2 := mantB
			dM := ecp &^ 1 // dM = floor(ecp/2)*2
			c3 := mantC
			if ecp&1 == 1 {
				c3 *= 2
			}
			if ecp < case1Min {
				// 𝑐′ underflows, so 4𝑎′𝑐′ underflows,
				// so √(𝑏′² − 4𝑎′𝑐′) ≈ √(𝑏′²) ≈ 𝑏′. Then
				// y₁ = −𝑏′−𝑏′ ∕ 2𝑎' = −(2𝑏′∕2𝑎′) = −(𝑏′∕𝑎′)
				y1 := -b2 / a2

				// For 𝑦₂ we use more scaling and Viete's formula. See page
				// 19 of "The ins and outs of solving quadratic equations with
				// floating-point arithmetic" by Goualard for the details.
				y2 := c3 / (a2 * y1)
				x1 := math.Ldexp(y1, expB-expA)
				x2 := math.Ldexp(y2, dM+expB-expA)
				return min(x1, x2), max(x1, x2), 2
			} else {
				// ecp >= case1Max
				if math.Signbit(a) == math.Signbit(c) {
					// No real solutions. Similar to above, we return nothing and don't
					// differentiate reasons for the lack of real roots.
					return math.NaN(), math.NaN(), 0
				} else {
					// 4𝑎′𝑐′ overflows, which means it is so large that we can
					// ignore the contribution of 𝑏′² and 𝑏′ and compute
					// ±√(|𝑐′∕𝑎′|) (If 𝑏′ were large enough to matter, ecp
					// wouldn't be big enough to end up in this branch.)
					//
					// Then we do more scaling as explained in the paper.
					S := math.Sqrt(math.Abs(c3 / a2))
					M := dM / 2
					r := math.Ldexp(S, M+expB-expA)
					return -r, r, 2
				}
			}
		}
	}
}
