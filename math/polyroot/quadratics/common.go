// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package quadratics

import (
	"fmt"
	"math"
	"math/rand/v2"

	"honnef.co/go/stuff/math/mathutil"
)

func discriminant(a, b, c float64) float64 {
	// 𝑏² − 4𝑎𝑐
	return mathutil.Det2x2(b, 4*a, c, b)
}

// isFin reports wheter v is finite, i.e., neither infinite nor NaN.
func isFin(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// Random generates random coefficients for a quadratic equation. It separately
// generates a mantissa in (-1, 1) and an exponent in [-1073, 1023] and returns
// math.Ldexp(mantissa, exponent).
func Random() (a, b, c float64) {
	num := func() float64 {
		// const minExp = -1022 // no subnormals
		const minExp = -1073 // all subnormals
		const maxExp = 1023

		exp := rand.IntN(maxExp-minExp) + minExp
		m := (rand.Float64()*2 - 1)
		return math.Ldexp(m, exp)
	}
	a, b, c = num(), num(), num()

	if !isFin(a) || !isFin(b) || !isFin(c) {
		panic(fmt.Sprintf("generated invalid quadratic with coefficients %g, %g, %g", a, b, c))
	}

	return a, b, c
}
