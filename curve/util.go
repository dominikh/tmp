// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package curve

import (
	"math"

	"honnef.co/go/stuff/math/mathutil"
)

func pow2(d float64) float64 { return d * d }
func pow3(d float64) float64 { return d * d * d }
func pow4(d float64) float64 { dd := d * d; return dd * dd }
func pow6(d float64) float64 {
	dd := d * d
	return dd * dd * dd
}
func pow7(d float64) float64 {
	dd := d * d
	return dd * dd * dd * d
}
func pow9(d float64) float64 {
	dd := d * d
	dddd := dd * dd
	return dddd * dddd * d
}

// normalizeAngle normalizes angles to [0, 2π].
func normalizeAngle(a Angle) Angle {
	a = math.Mod(a, 2*math.Pi)
	if a < 0 {
		a += 2 * math.Pi
	}
	return a
}

func clampAngle(a Angle) Angle {
	return mathutil.Clamp(a, -2*math.Pi, 2*math.Pi)
}
