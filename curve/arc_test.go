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

func checkPointNear(t *testing.T, got, want Point) {
	t.Helper()

	const epsilon = 1e-12
	if d := got.Distance(want); d > epsilon {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestArcEndpointsAndMidpoint(t *testing.T) {
	arc := Arc{Pt(3.0, -2.0), Vec(4.0, 1.5), math.Pi / 6.0, math.Pi, 0.0}
	checkPointNear(t, arc.Eval(0.0), arc.Start())
	checkPointNear(t, arc.Eval(1.0), arc.End())
	checkPointNear(
		t,
		arc.Eval(0.5),
		arc.Center.Translate(sampleEllipse(arc.Radii, arc.XRotation, arc.AngleAt(0.5))),
	)
}

func TestArcRotatedEllipse(t *testing.T) {
	arc := Arc{Pt(5.0, 7.0), Vec(3.0, 2.0), math.Pi / 6.0, math.Pi / 3.0, math.Pi / 4}
	checkPointNear(
		t,
		arc.Start(),
		arc.Center.Translate(sampleEllipse(arc.Radii, arc.XRotation, arc.StartAngle)),
	)
	checkPointNear(
		t,
		arc.End(),
		arc.Center.Translate(sampleEllipse(arc.Radii, arc.XRotation, arc.StartAngle+arc.SweepAngle)),
	)
}

func TestArcReverse(t *testing.T) {
	arc := Arc{
		Center:     Pt(2.0, 3.0),
		Radii:      Vec(4.0, 1.0),
		StartAngle: math.Pi / 6,
		SweepAngle: math.Pi / 2,
		XRotation:  math.Pi / 4,
	}
	reversed := arc.Reverse()
	checkPointNear(t, reversed.Start(), arc.End())
	checkPointNear(t, reversed.End(), arc.Start())
	checkPointNear(t, reversed.Eval(0.5), arc.Eval(0.5))
}

func TestArcSubsegmentMatchesOriginal(t *testing.T) {
	for _, tt := range []struct {
		arc   Arc
		start float64
		end   float64
	}{
		{
			arc: Arc{
				Center:     Pt(1.0, -4.0),
				Radii:      Vec(3.0, 2.0),
				StartAngle: -math.Pi / 4.0,
				SweepAngle: math.Pi,
				XRotation:  math.Pi / 6.0,
			},
			start: 0.2,
			end:   0.7,
		},

		{
			// Negative sweep
			arc: Arc{
				Center:     Pt(1.0, 2.0),
				Radii:      Vec(5.0, 3.0),
				StartAngle: math.Pi,
				SweepAngle: -0.75 * math.Pi,
				XRotation:  math.Pi / 6,
			},
			start: 0.1,
			end:   0.8,
		},

		{
			// Tiny sweep
			arc: Arc{
				Center:     Pt(0.0, 0.0),
				Radii:      Vec(2.0, 1.0),
				StartAngle: 1.0,
				SweepAngle: 1e-9,
				XRotation:  math.Pi / 4,
			},
			start: 0.25,
			end:   0.75,
		},
	} {
		subsegment := tt.arc.Subsegment(tt.start, tt.end)
		for _, s := range []float64{0.0, 0.25, 0.5, 1.0} {
			expectedT := tt.start + (tt.end-tt.start)*s
			checkPointNear(t, subsegment.Eval(s), tt.arc.Eval(expectedT))
		}
		checkPointNear(t, subsegment.Start(), tt.arc.Eval(tt.start))
		checkPointNear(t, subsegment.End(), tt.arc.Eval(tt.end))
	}
}
