// SPDX-FileCopyrightText: 2026 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package curve

import (
	"math"
	"testing"
)

func TestArc_Reverse(t *testing.T) {
	a := Arc{
		Center:     Pt(0, 0),
		Radii:      Vec(1, 0),
		StartAngle: 0,
		SweepAngle: math.Pi,
		XRotation:  0,
	}

	got := a.Reverse()
	want := Arc{
		Center:     Pt(0, 0),
		Radii:      Vec(1, 0),
		StartAngle: math.Pi,
		SweepAngle: -math.Pi,
		XRotation:  0,
	}

	if got != want {
		t.Fatalf("reversing arc %v got %v, want %v", a, got, want)
	}

	got2 := got.Reverse()
	if got2 != a {
		t.Fatalf("reversing twice wasn't idempotent: %v -> %v -> %v", a, got, got2)
	}
}
