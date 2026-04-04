// SPDX-FileCopyrightText: 2018 Raph Levien
// SPDX-FileCopyrightText: 2024 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT
// SPDX-FileAttributionText: https://github.com/linebender/kurbo

package curve

import (
	"math"
	"testing"
)

func TestRectAreaSign(t *testing.T) {
	approxEqual := func(x, y float64) bool {
		return math.Abs(x-y) < 1e-8
	}

	r := Rect{0.0, 0.0, 10.0, 10.0}
	center := r.Center()
	if a := r.Area(); !approxEqual(a, 100) {
		t.Errorf("got area %v, want %v", a, 100.0)
	}
	if w := r.Winding(center); w != 1 {
		t.Errorf("got winding %v, want %v", w, 1)
	}

	p := (r.Path(1e-9, nil))
	if ra, pa := r.Area(), p.Area(); !approxEqual(ra, pa) {
		t.Errorf("expected r's and p's areas to be approximately equal, got %v and %v", ra, pa)
	}
	if rw, pw := r.Winding(center), p.Winding(center); rw != pw {
		t.Errorf("expected r's and p's winding numbers to be equal, got %v and %v", rw, pw)
	}

	rFlip := Rect{0.0, 10.0, 10.0, 0.0}
	if a := rFlip.Area(); !approxEqual(a, -100) {
		t.Errorf("got area %v, want %v", a, -100.0)
	}

	if w := rFlip.Winding(Pt(5, 5)); w != -1 {
		t.Errorf("got winding %v, want %v", w, -1)
	}

	pFlip := (rFlip.Path(1e09, nil))
	if ra, pa := rFlip.Area(), pFlip.Area(); !approxEqual(ra, pa) {
		t.Errorf("expected r's and p's areas to be approximately equal, got %v and %v", ra, pa)
	}
	if rw, pw := rFlip.Winding(center), pFlip.Winding(center); rw != pw {
		t.Errorf("expected r's and p's winding numbers to be equal, got %v and %v", rw, pw)
	}
}

func TestContainedRectWithAspectRatio(t *testing.T) {
	f := func(outer Rect, aspectRatio float64, want Rect) {
		t.Helper()
		if got := outer.ContainedRectWithAspectRatio(aspectRatio); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	// squares (different point orderings)
	f(Rect{0.0, 0.0, 10.0, 20.0}, 1.0, Rect{0.0, 5.0, 10.0, 15.0})
	f(Rect{0.0, 20.0, 10.0, 0.0}, 1.0, Rect{0.0, 5.0, 10.0, 15.0})
	f(Rect{10.0, 0.0, 0.0, 20.0}, 1.0, Rect{10.0, 15.0, 0.0, 5.0})
	f(Rect{10.0, 20.0, 0.0, 0.0}, 1.0, Rect{10.0, 15.0, 0.0, 5.0})
	// non-square
	f(Rect{0.0, 0.0, 10.0, 20.0}, 1.0/0.5, Rect{0.0, 7.5, 10.0, 12.5})
	// same aspect ratio
	f(Rect{0.0, 0.0, 10.0, 20.0}, 1.0/2.0, Rect{0.0, 0.0, 10.0, 20.0})
	// negative aspect ratio
	f(Rect{0.0, 0.0, 10.0, 20.0}, -1.0, Rect{0.0, 15.0, 10.0, 5.0})
	// infinite aspect ratio
	f(Rect{0.0, 0.0, 10.0, 20.0}, math.Inf(1), Rect{0.0, 10.0, 10.0, 10.0})
	// zero aspect ratio
	f(Rect{0.0, 0.0, 10.0, 20.0}, 0, Rect{5.0, 0.0, 5.0, 20.0})
	// zero width rect
	f(Rect{0.0, 0.0, 0.0, 20.0}, 1.0, Rect{0.0, 10.0, 0.0, 10.0})
	// many zeros
	f(Rect{0.0, 0.0, 0.0, 20.0}, 0.0, Rect{0.0, 0.0, 0.0, 20.0})
	// everything zero
	f(Rect{0.0, 0.0, 0.0, 0.0}, 0.0, Rect{0.0, 0.0, 0.0, 0.0})
}

func TestAspectRatio(t *testing.T) {
	r := Rect{0, 0, 1, 1}
	if ratio := r.AspectRatio(); math.Abs(ratio-1) > 1e-6 {
		t.Errorf("got ratio %v, want 1.0", ratio)
	}
}

func TestRoundedRectArea(t *testing.T) {
	const epsilon = 1e-9

	// Extremum: 0.0 radius corner -> rectangle
	rect := Rect{0.0, 0.0, 100.0, 100.0}
	roundedRect := RoundedRect{Rect: rect}
	if ra, rra := rect.Area(), roundedRect.Area(); math.Abs(ra-rra) > epsilon {
		t.Errorf("got areas %v and %v, expected them to be equal", ra, rra)
	}

	// Extremum: half-size radius corner -> circle
	circle := Circle{Pt(0.0, 0.0), 50.0}
	roundedRect = NewRoundedRect(0, 0, 100, 100, 50)
	if ca, rra := circle.Area(), roundedRect.Area(); math.Abs(ca-rra) > epsilon {
		t.Errorf("got areas %v and %v, expected them to be equal", ca, rra)
	}
}

func TestRoundedRectWinding(t *testing.T) {
	type point struct {
		point   Point
		winding int
	}
	tests := []struct {
		rect   RoundedRect
		points []point
	}{
		{
			RoundedRect{Rect{-5.0, -5.0, 10.0, 20.0}, RoundedRectRadii{5, 5, 5, 0}},
			[]point{
				{Pt(0.0, 0.0), 1},
				{Pt(-5.0, 0.0), 1},  // left edge
				{Pt(0.0, 20.0), 1},  // bottom edge
				{Pt(10.0, 20.0), 0}, // bottom-right corner
				{Pt(-5.0, 20.0), 1}, // bottom-left corner (has a radius of 0)
				{Pt(-10.0, 0.0), 0},
			},
		},
		{
			NewRoundedRect(-10.0, -20.0, 10.0, 20.0, 0.0), // rectangle
			[]point{
				{Pt(10, 20), 1}, // bottom-right corner
			},
		},
	}
	for i, tt := range tests {
		for j, p := range tt.points {
			if w := tt.rect.Winding(p.point); w != p.winding {
				t.Errorf("test %d, point %d (zero-indexed): got winding %d, expected %d", i, j, w, p.winding)
			}
		}
	}
}

func TestRoundedRectBeziers(t *testing.T) {
	rect := NewRoundedRect(-5, -5, 10, 20, 5)
	p := rect.Path(1e-9, nil)
	// Note: could be more systematic about tolerance tightness.
	const epsilon = 1e-7
	if ra, pa := rect.Area(), p.Area(); math.Abs(ra-pa) > epsilon {
		t.Errorf("got areas %v and %v, expected them to be the same", ra, pa)
	}
	if w := p.Winding(Point{}); w != 1 {
		t.Errorf("got winding %d, expected 1", w)
	}
}

func TestRect_Overlaps(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		r    Rect
		o    Rect
		want bool
	}{
		// Abutting rectangles do not overlap. (Test names assume a y-down
		// coordinate system)
		{
			"abutting-top",
			Rect{0, 0, 1, 1},
			Rect{0, -1, 1, 0},
			false,
		},
		{
			"abutting-right",
			Rect{0, 0, 1, 1},
			Rect{1, 0, 2, 1},
			false,
		},
		{
			"abutting-bottom",
			Rect{0, 0, 1, 1},
			Rect{0, 1, 1, 2},
			false,
		},
		{
			"abutting-left",
			Rect{0, 0, 1, 1},
			Rect{-1, 0, 0, 1},
			false,
		},
		{
			"abutting-top-left",
			Rect{0, 0, 1, 1},
			Rect{-1, -1, 0, 0},
			false,
		},
		{
			"abutting-top-right",
			Rect{0, 0, 1, 1},
			Rect{1, -1, 2, 0},
			false,
		},
		{
			"abutting-bottom-right",
			Rect{0, 0, 1, 1},
			Rect{1, 1, 2, 2},
			false,
		},
		{
			"abutting-bottom-left",
			Rect{0, 0, 1, 1},
			Rect{-1, 1, 0, 2},
			false,
		},

		// o inside r
		{
			"o-inside-r",
			Rect{0, 0, 1, 1},
			Rect{0.25, 0.25, 0.75, 0.75},
			true,
		},

		// r inside o
		{
			"r-inside-o",
			Rect{0, 0, 1, 1},
			Rect{-1, -1, 2, 2},
			true,
		},

		// o inside r with a shared edge
		{
			"o-inside-r-shared-edge",
			Rect{0, 0, 1, 1},
			Rect{0, 0.25, 0.5, 0.75},
			true,
		},

		// Identical rects
		{
			"identical",
			Rect{0, 0, 1, 1},
			Rect{0, 0, 1, 1},
			true,
		},

		// All the ways of overlapping
		{
			"overlap-top",
			Rect{0, 0, 1, 1},
			Rect{0.25, -0.25, 0.75, 0.25},
			true,
		},
		{
			"overlap-right",
			Rect{0, 0, 1, 1},
			Rect{0.75, 0.25, 1.25, 0.75},
			true,
		},
		{
			"overlap-bottom",
			Rect{0, 0, 1, 1},
			Rect{0.25, 0.75, 0.75, 1.25},
			true,
		},
		{
			"overlap-left",
			Rect{0, 0, 1, 1},
			Rect{-0.25, 0.25, 0.25, 0.75},
			true,
		},
		{
			"overlap-top-left",
			Rect{0, 0, 1, 1},
			Rect{-0.25, -0.25, 0.25, 0.25},
			true,
		},
		{
			"overlap-top-right",
			Rect{0, 0, 1, 1},
			Rect{0.75, -0.25, 1.25, 0.25},
			true,
		},
		{
			"overlap-bottom-right",
			Rect{0, 0, 1, 1},
			Rect{0.75, 0.75, 1.25, 1.25},
			true,
		},
		{
			"overlap-bottom-left",
			Rect{0, 0, 1, 1},
			Rect{-0.25, 0.75, 0.25, 1.25},
			true,
		},

		// A point does not overlap a rectangle and vice versa
		{
			"singularity",
			Rect{0, 0, 1, 1},
			Rect{0.5, 0.5, 0.5, 0.5},
			false,
		},

		// Non-existent rectangles don't overlap
		{
			"mixed-signs",
			Rect{0, 0, 1, 1},
			Rect{0.25, 0.75, 0.75, 0.25},
			false,
		},
		{
			"both-negative",
			Rect{0, 1, 1, 0},
			Rect{0.25, 0.75, 0.75, 0.25},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.Overlaps(tt.o)
			if got != tt.want {
				t.Errorf("%s: %v.Overlaps(%v) = %v, want %v", tt.name, tt.r, tt.o, got, tt.want)
			}
			if tt.o.Overlaps(tt.r) != got {
				t.Errorf("%s: r.Overlaps(o) != o.Overlaps(r)", tt.name)
			}
		})
	}
}
