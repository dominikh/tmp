// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package curve

func pow2(d float64) float64 { return d * d }
func pow3(d float64) float64 { return d * d * d }
func pow4(d float64) float64 { dd := d * d; return dd * dd }
func pow6(d float64) float64 {
	dd := d * d
	return dd * dd * dd
}
func pow9(d float64) float64 {
	dd := d * d
	dddd := dd * dd
	return dddd * dddd * d
}
