// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package math32

import "math"

func Floor(f float32) float32 {
	return float32(math.Floor(float64(f)))
}

func Ceil(f float32) float32 {
	return float32(math.Ceil(float64(f)))
}

func Abs(f float32) float32 {
	return math.Float32frombits(math.Float32bits(f) &^ (1 << 31))
}

func Sqrt(f float32) float32 {
	return float32(math.Sqrt(float64(f)))
}

func Sign(f float32) float32 {
	if math.Float32bits(f)&(1<<31) != 0 {
		// f is -0.0 or negative
		return -1
	} else {
		return 1
	}
}
func IsNaN(f float32) bool {
	return f != f
}

func Round(f float32) float32 {
	return float32(math.Round(float64(f)))
}
