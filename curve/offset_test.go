// SPDX-FileCopyrightText: 2018 Raph Levien
// SPDX-FileCopyrightText: 2024 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT
// SPDX-FileAttributionText: https://github.com/linebender/kurbo

package curve

import (
	"testing"
)

// This test tries combinations of parameters that have caused problems in the past.
func TestOffsetPathologicalCurves(t *testing.T) {
	curve := CubicBez{
		P0: Pt(-1236.3746269978635, 152.17981429574826),
		P1: Pt(-1175.18662093517, 108.04721798590596),
		P2: Pt(-1152.142883879584, 105.76260301083356),
		P3: Pt(-1151.842639804639, 105.73040758939104),
	}
	const offset = 3603.7267536453924
	const accuracy = 0.1

	path := offsetCubic(curve, offset, accuracy, true, nil)
	if len(path) == 0 || path[0].Kind != MoveToKind {
		t.Fatalf("did not get valid path")
	}
}

// Cubic offset that used to trigger infinite recursion.
func TestOffsetInfiniteRecursion(t *testing.T) {
	const tolerance = 0.1
	const offset = -0.5
	c := CubicBez{
		Pt(1096.2962962962963, 593.90243902439033),
		Pt(1043.6213991769548, 593.90243902439033),
		Pt(1030.4526748971193, 593.90243902439033),
		Pt(1056.7901234567901, 593.90243902439033),
	}
	// Test that we terminate
	offsetCubic(c, offset, tolerance, true, nil)
}

func TestCubicOffsetSimpleLine(t *testing.T) {
	cubic := CubicBez{
		Pt(0.0, 0.0),
		Pt(10.0, 0.0),
		Pt(20.0, 0.0),
		Pt(30.0, 0.0),
	}
	// Test that we terminate
	offsetCubic(cubic, 5, 1e-6, true, nil)
}
