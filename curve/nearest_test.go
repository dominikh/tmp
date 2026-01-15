// SPDX-FileCopyrightText: 2018 Raph Levien
// SPDX-FileCopyrightText: 2026 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT
// SPDX-FileAttributionText: https://github.com/linebender/kurbo

package curve

import (
	"fmt"
	"testing"
)

func BenchmarkQuadraticNearest(b *testing.B) {
	q := QuadBez{
		P0: Pt(-1, -1),
		P1: Pt(0, 2),
		P2: Pt(1, -1),
	}
	p := Pt(0, 0)

	for _, acc := range []float64{1e-3, 1e-6, 1e-12} {
		b.Run(fmt.Sprintf("acc=%g", acc), func(b *testing.B) {
			for b.Loop() {
				q.Nearest(p, acc)
			}
		})
	}
}

func BenchmarkCubicNearest(b *testing.B) {
	c := CubicBez{
		P0: Pt(-1, -1),
		P1: Pt(0, 2),
		P2: Pt(1, -1),
		P3: Pt(2, 2),
	}
	p := Pt(0, 0)

	for _, acc := range []float64{1e-3, 1e-6, 1e-12} {
		b.Run(fmt.Sprintf("acc=%g", acc), func(b *testing.B) {
			for b.Loop() {
				c.Nearest(p, acc)
			}
		})
	}
}
