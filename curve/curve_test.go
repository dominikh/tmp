// SPDX-FileCopyrightText: 2018 Raph Levien
// SPDX-FileCopyrightText: 2024 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT
// SPDX-FileAttributionText: https://github.com/linebender/kurbo

package curve

import (
	"math"
	"slices"
	"testing"
)

func TestSolveITP(t *testing.T) {
	f := func(x float64) float64 { return x*x*x - x - 2.0 }
	x := SolveITP(f, 1.0, 2.0, 1e-12, 0, 0.2, f(1.0), f(2.0))
	if n := math.Abs(f(x)); n > 6e-12 {
		t.Errorf("%v > 6e-12", n)
	}
}

func TestSolveForArclen(t *testing.T) {
	c := CubicBez{
		Pt(0.0, 0.0),
		Pt(100.0/3.0, 0.0),
		Pt(200.0/3.0, 100.0/3.0),
		Pt(100.0, 100.0),
	}
	const target = 100.0
	SolveITP(
		func(t float64) float64 { return c.Subsegment(0.0, t).Arclen(1e-9) - target },
		0.0,
		1.0,
		1e-6,
		1,
		0.2,
		-target,
		c.Arclen(1e-9)-target,
	)
}

func TestSVGSingle(t *testing.T) {
	segments := []PathSegment{
		CubicBez{
			Pt(10.0, 10.0),
			Pt(20.0, 20.0),
			Pt(30.0, 30.0),
			Pt(40.0, 40.0),
		}.Seg(),
	}
	var path BezPath = slices.Collect(Elements(slices.Values(segments)))
	want := "M10,10 C20,20 30,30 40,40"
	got := path.SVG(SVGOptions{})
	diff(t, got, want)
}

func TestSVGTwoNoMove(t *testing.T) {
	segments := []PathSegment{
		CubicBez{
			Pt(10.0, 10.0),
			Pt(20.0, 20.0),
			Pt(30.0, 30.0),
			Pt(40.0, 40.0),
		}.Seg(),
		CubicBez{
			Pt(40.0, 40.0),
			Pt(30.0, 30.0),
			Pt(20.0, 20.0),
			Pt(10.0, 10.0),
		}.Seg(),
	}
	var path BezPath = slices.Collect(Elements(slices.Values(segments)))
	want := "M10,10 C20,20 30,30 40,40 C30,30 20,20 10,10"
	got := path.SVG(SVGOptions{})
	diff(t, got, want)
}

func TestSVGTwoMove(t *testing.T) {
	segments := []PathSegment{
		CubicBez{
			Pt(10.0, 10.0),
			Pt(20.0, 20.0),
			Pt(30.0, 30.0),
			Pt(40.0, 40.0),
		}.Seg(),
		CubicBez{
			Pt(50.0, 50.0),
			Pt(30.0, 30.0),
			Pt(20.0, 20.0),
			Pt(10.0, 10.0),
		}.Seg(),
	}
	var path BezPath = slices.Collect(Elements(slices.Values(segments)))
	want := "M10,10 C20,20 30,30 40,40 M50,50 C30,30 20,20 10,10"
	got := path.SVG(SVGOptions{})
	diff(t, got, want)
}
