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

func checkRectNear(t *testing.T, got, want Rect) {
	t.Helper()

	const epsilon = 1e-12
	if math.Abs(got.X0-want.X0) > epsilon ||
		math.Abs(got.Y0-want.Y0) > epsilon ||
		math.Abs(got.X1-want.X1) > epsilon ||
		math.Abs(got.Y1-want.Y1) > epsilon {
		t.Fatalf("got %#v, want %#v", got, want)
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

func TestArcBoundingBoxAxisAlignedQuarter(t *testing.T) {
	arc := Arc{
		Center:     Pt(1, 2),
		Radii:      Vec(3, 4),
		StartAngle: 0,
		SweepAngle: math.Pi / 2,
		XRotation:  0,
	}

	got := arc.BoundingBox()
	want := Rect{X0: 1, Y0: 2, X1: 4, Y1: 6}
	checkRectNear(t, got, want)
}

func TestArcBoundingBoxFullArcMatchesEllipse(t *testing.T) {
	arc := Arc{
		Center:     Pt(5, 7),
		Radii:      Vec(3, 2),
		StartAngle: math.Pi / 6,
		SweepAngle: 2 * math.Pi,
		XRotation:  math.Pi / 4,
	}

	got := arc.BoundingBox()
	want := NewEllipse(arc.Center, arc.Radii, arc.XRotation).BoundingBox()
	checkRectNear(t, got, want)
}

func TestArcBoundingBoxContainsSamples(t *testing.T) {
	testCases := []Arc{
		{
			Center:     Pt(2, 3),
			Radii:      Vec(4, 1),
			StartAngle: math.Pi / 6,
			SweepAngle: math.Pi,
			XRotation:  math.Pi / 4,
		},
		{
			Center:     Pt(-1, 4),
			Radii:      Vec(5, 3),
			StartAngle: 1.7,
			SweepAngle: -1.3 * math.Pi,
			XRotation:  math.Pi / 7,
		},
	}

	const samples = 10000
	const epsilon = 1e-9
	for _, arc := range testCases {
		bbox := arc.BoundingBox()
		for i := range samples + 1 {
			pt := arc.Eval(float64(i) / samples)
			if pt.X < bbox.X0-epsilon || pt.X > bbox.X1+epsilon || pt.Y < bbox.Y0-epsilon || pt.Y > bbox.Y1+epsilon {
				t.Fatalf("BoundingBox() = %#v does not contain point %v on arc %#v", bbox, pt, arc)
			}
		}
	}
}

func TestArcPerimeterCircularArc(t *testing.T) {
	arc := Arc{
		Center:     Pt(0, 0),
		Radii:      Vec(3, 3),
		StartAngle: math.Pi / 7,
		SweepAngle: math.Pi / 2,
		XRotation:  math.Pi / 3,
	}

	got := arc.Perimeter(1e-12)
	want := 3 * math.Pi / 2
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("Perimeter() = %g, want %g", got, want)
	}
}

func TestArcPerimeterFullArcMatchesEllipse(t *testing.T) {
	arc := Arc{
		Center:     Pt(5, 7),
		Radii:      Vec(3, 2),
		StartAngle: math.Pi / 6,
		SweepAngle: 2 * math.Pi,
		XRotation:  math.Pi / 4,
	}

	got := arc.Perimeter(1e-12)
	want := NewEllipse(arc.Center, arc.Radii, arc.XRotation).Perimeter(1e-12)
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("Perimeter() = %g, want %g", got, want)
	}
}

func TestArcPerimeterSubsegmentsAddUp(t *testing.T) {
	arc := Arc{
		Center:     Pt(2, 3),
		Radii:      Vec(4, 1),
		StartAngle: math.Pi / 6,
		SweepAngle: -1.3 * math.Pi,
		XRotation:  math.Pi / 4,
	}

	whole := arc.Perimeter(1e-10)
	left := arc.Subsegment(0, 0.4).Perimeter(5e-11)
	right := arc.Subsegment(0.4, 1).Perimeter(5e-11)
	if math.Abs(whole-(left+right)) > 1e-10 {
		t.Fatalf("Perimeter() = %g, subsegments sum to %g", whole, left+right)
	}
}

func TestArcPerimeterMatchesBezierApproximation(t *testing.T) {
	arc := Arc{
		Center:     Pt(-1, 4),
		Radii:      Vec(5, 3),
		StartAngle: 1.7,
		SweepAngle: -1.3 * math.Pi,
		XRotation:  math.Pi / 7,
	}

	got := arc.Perimeter(1e-12)
	approx := arc.Path(1e-11).Arclen(1e-12)
	if math.Abs(got-approx) > 2e-11 {
		t.Fatalf("Perimeter() = %g, bezier approximation = %g", got, approx)
	}
}
