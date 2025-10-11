// SPDX-FileCopyrightText: 2022 ChrisLomont
// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
// SPDX-FileAttributionText: https://lomont.org/posts/2022/a-better-quadratic-formula-algorithm/
// SPDX-FileAttributionText: https://github.com/ChrisLomont/BetterQuadraticRoots
//
// SPDX-License-Identifier: MIT

package quadratics

import (
	"math"
)

// Lomont implements Chris Lomont's "better quadratic formula algorithm" as
// described in
// https://lomont.org/posts/2022/a-better-quadratic-formula-algorithm/.
//
// # Known shortcomings
//
// If one of two roots doesn't fit in float64, the other root may not be found,
// even if it fits. This is
// https://github.com/ChrisLomont/BetterQuadraticRoots/issues/1.
//
// The original implementation will trigger an assertion when a < 0, b == 0, c
// == 0. Our version works around this.
//
// # Modifications
//
// The original version may return roots in an unsorted order, while we expect
// them to be sorted in ascending order. Lomont never claimed that they would be
// sorted so we don't consider this a shortcoming. We have, however, modified
// our implementation to sort the values.
func Lomont(a, b, c float64) (r1, r2 float64, n int) {
	r1, r2, n, isHandled := handleSpecialCasesFloat(a, b, c)
	if isHandled {
		return r1, r2, n
	}

	// TODO(dh): what's the point of returning root and rootE separately, when we
	// immediately scale, anyway?
	root, nonnegative, rootE := discriminantInfo(a, b, c)
	root = math.Ldexp(root, rootE)

	if nonnegative {
		if math.Abs(b) < math.MaxFloat64/2 {
			r1 = (-b - lomontCopysign(root, b)) / math.Ldexp(a, 1)
		} else {
			r1 = -b/math.Ldexp(a, 1) - lomontCopysign(root, b)/math.Ldexp(a, 1)
		}
		r2 = c / (r1 * a)
		// two reals
		return min(r1, r2), max(r1, r2), 2
	} else {
		// Complex roots
		return math.NaN(), math.NaN(), 0
	}
}

// Compute the discriminant D = b*b-4*a*c
// Return the (scaled) root r' = Sqrt(|D|), if d >= 0, and a scaling factor E
// such that the correct root is r = 2^E * r'
func discriminantInfo(a, b, c float64) (root float64, nonnegative bool, scale int) {
	aS, aE, aF := normalize(a)
	_, bE, bF := normalize(b)
	cS, cE, cF := normalize(c)

	root, scale, nonnegative = b, 0, true

	if 2*bE > aE+cE+53+5 {
		// +5 works, is derived, seems to work( +4, +0, -2, ) , -10 fails (-10, -5, -4, -3)
		root = bF
		scale = bE
		nonnegative = true
	} else if 2*bE < aE+cE-53-1 {
		// works: (-1,+2,+4), fails (+5, +7,+15,+40)
		scale = aE + cE
		if scale%2 != 0 {
			scale--
			aF = math.Ldexp(aF, 1)
		}
		//  +1 for the 4 in 4ac, then root, /2 is root
		scale = scale/2 + 1

		root = math.Sqrt(aF * cF)
		nonnegative = aS*cS < 0
	} else {
		deltaE := (aE - cE) / 2

		mid := (bE + bE + aE + cE) / 4
		aF = math.Ldexp(a, -mid-deltaE)
		bF = math.Ldexp(b, -mid)
		cF = math.Ldexp(c, -mid+deltaE)

		// d = 𝑏′² − 4𝑎′𝑐′
		d := discriminant(aF, bF, cF)

		root = math.Sqrt(math.Abs(d))
		nonnegative = d >= 0
		scale = mid
	}
	return root, nonnegative, scale
}

func normalize(value float64) (sign, exp int, frac float64) {
	isNormal := func(v float64) bool {
		return v != 0 && !math.IsNaN(v) && !math.IsInf(v, 0)
	}

	if value == 0 {
		return 1, 0, 0
	}

	if math.Signbit(value) {
		sign = -1
	} else {
		sign = 1
	}

	const expMask = (1 << 11) - 1
	const expBias = (1 << (11 - 1)) - 1

	i := math.Float64bits(value)

	exp = int(((i >> (53 - 1)) & expMask) - expBias)
	frac = float64(sign) * math.Ldexp(value, -exp)
	subnormal := exp <= -1023

	if !isNormal(value) || subnormal {
		s2, e2, f2 := normalize(frac)
		exp += e2
		frac = f2
		if s2 != 1 {
			panic("unreachable")
		}
	}

	return sign, exp, frac
}

func handleSpecialCasesFloat(a, b, c float64) (r1, r2 float64, n int, isHandled bool) {
	if math.IsNaN(a) || math.IsNaN(b) || math.IsNaN(c) {
		return math.NaN(), math.NaN(), 0, true
	}
	if math.IsInf(a, 0) || math.IsInf(b, 0) || math.IsInf(c, 0) {
		return math.NaN(), math.NaN(), 0, true
	}

	if a == 0 {
		// want bx+c = 0 gives x = -c/b
		if b == 0 && c == 0 {
			return math.NaN(), math.NaN(), 0, true
		}

		r1 := -c / b
		return r1, math.NaN(), 1, true
	}

	if b == 0 {
		// a != 0, want ax^2+c = 0, so x = +/- sqrt(-c/a)

		if c == 0 {
			return 0, math.NaN(), 1, true
		}

		sgn := lomontSign(a) * lomontSign(c)

		if sgn <= 0 {
			// real answers
			r1 := divRoot(-c, a)
			r2 := -r1
			return min(r1, r2), max(r1, r2), 2, true
		} else {
			// complex answers
			return math.NaN(), math.NaN(), 0, true
		}
	}

	if c == 0 {
		// a,b != 0, of form ax^2 + bx = 0, so roots are x=0 and x=-b/a
		r := -b / a
		return min(r, 0), max(r, 0), 2, true
	}

	return math.NaN(), math.NaN(), 0, false
}

// Compute sqrt(|x/y|), handling overflow and underflow if possible.
func divRoot(x, y float64) float64 {
	xS, xE, xF := normalize(x)
	yS, yE, yF := normalize(y)

	if xS*yS < 0 {
		panic("unreachable")
	}

	q := xF / yF
	e := xE - yE

	if ((xE + yE) & 1) == 1 {
		// exponent odd, scale so can easily update after root
		q = math.Ldexp(q, 1)
		e--
	}

	r := math.Sqrt(q)
	return math.Ldexp(r, e/2)
}

func lomontSign(x float64) float64 {
	if x < 0 {
		return -1
	} else if x == 0 {
		return 0
	} else if x > 0 {
		return 1
	} else {
		return math.NaN()
	}
}

func lomontCopysign(x, y float64) float64 {
	return math.Abs(x) * lomontSign(y)
}
