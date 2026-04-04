// SPDX-FileCopyrightText: 2018 Raph Levien
// SPDX-FileCopyrightText: 2026 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT
// SPDX-FileAttributionText: https://github.com/linebender/kurbo

package curve

import (
	"math"
	"testing"
)

const triangleRelTol = 1e-12

func approxEqualTriangleRel(x, y, rel float64) bool {
	scale := max(1.0, max(math.Abs(x), math.Abs(y)))
	return math.Abs(x-y) <= scale*rel
}

func approxEqualTrianglePoint(x, y Point, rel float64) bool {
	return approxEqualTriangleRel(x.X, y.X, rel) && approxEqualTriangleRel(x.Y, y.Y, rel)
}

func TestTriangleAreaSign(t *testing.T) {
	approxEqual := func(x, y float64) bool {
		return math.Abs(x-y) < 1e-8
	}

	tri := Triangle{Pt(0, 0), Pt(10, 0), Pt(0, 10)}
	center := Pt(1, 1)
	if a := tri.Area(); !approxEqual(a, 50) {
		t.Errorf("got area %v, want %v", a, 50.0)
	}
	if w := tri.Winding(center); w != 1 {
		t.Errorf("got winding %v, want %v", w, 1)
	}

	p := tri.Path(1e-9, nil)
	if ta, pa := tri.Area(), p.Area(); !approxEqual(ta, pa) {
		t.Errorf("expected triangle and path areas to be approximately equal, got %v and %v", ta, pa)
	}
	if tw, pw := tri.Winding(center), p.Winding(center); tw != pw {
		t.Errorf("expected triangle and path winding numbers to be equal, got %v and %v", tw, pw)
	}

	triFlip := Triangle{tri.P1, tri.P0, tri.P2}
	if a := triFlip.Area(); !approxEqual(a, -50) {
		t.Errorf("got area %v, want %v", a, -50.0)
	}
	if w := triFlip.Winding(center); w != -1 {
		t.Errorf("got winding %v, want %v", w, -1)
	}

	pFlip := triFlip.Path(1e-9, nil)
	if ta, pa := triFlip.Area(), pFlip.Area(); !approxEqual(ta, pa) {
		t.Errorf("expected flipped triangle and path areas to be approximately equal, got %v and %v", ta, pa)
	}
	if tw, pw := triFlip.Winding(center), pFlip.Winding(center); tw != pw {
		t.Errorf("expected flipped triangle and path winding numbers to be equal, got %v and %v", tw, pw)
	}
}

func TestTriangleCentroid(t *testing.T) {
	got := Triangle{Pt(-90.02, 3.5), Pt(7.2, -9.3), Pt(8.0, 9.1)}.Centroid()
	want := Pt(-24.94, 1.1)
	if !approxEqualTrianglePoint(got, want, triangleRelTol) {
		t.Fatalf("got centroid %v, want %v", got, want)
	}
}

func TestTriangleOffsets(t *testing.T) {
	got := Triangle{Pt(-20.0, 180.2), Pt(1.2, 0.0), Pt(290.0, 100.0)}.Offsets()
	want := [3]Vec2{
		Vec(-110.4, 86.8),
		Vec(-89.2, -93.4),
		Vec(199.6, 6.6),
	}

	for i := range got {
		if !approxEqualTrianglePoint(Point(got[i]), Point(want[i]), triangleRelTol) {
			t.Fatalf("offset %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestTriangleArea(t *testing.T) {
	tri := Triangle{
		Pt(12123.423, 2382.7834),
		Pt(7892.729, 238.459),
		Pt(7820.2, 712.23),
	}
	want := 1079952.9157407999

	if !approxEqualTriangleRel(tri.Area(), -want, triangleRelTol) {
		t.Fatalf("got area %v, want %v", tri.Area(), -want)
	}

	tri = Triangle{tri.P1, tri.P0, tri.P2}
	if !approxEqualTriangleRel(tri.Area(), want, triangleRelTol) {
		t.Fatalf("got area %v, want %v", tri.Area(), want)
	}
}

func TestTriangleCircumcenter(t *testing.T) {
	got := EquilateralTriangle.CircumscribedCircle().Center
	want := Pt(0.5, 0.28867513459481288)
	if !approxEqualTrianglePoint(got, want, triangleRelTol) {
		t.Fatalf("got circumcenter %v, want %v", got, want)
	}
}

func TestTriangleInradius(t *testing.T) {
	got := EquilateralTriangle.InscribedCircle().Radius
	want := 0.28867513459481287
	if !approxEqualTriangleRel(got, want, triangleRelTol) {
		t.Fatalf("got inradius %v, want %v", got, want)
	}
}

func TestTriangleCircumradius(t *testing.T) {
	tri := EquilateralTriangle
	want := 0.57735026918962576
	if !approxEqualTriangleRel(tri.CircumscribedCircle().Radius, want, triangleRelTol) {
		t.Fatalf("got circumradius %v, want %v", tri.CircumscribedCircle().Radius, want)
	}

	tri = Triangle{tri.P1, tri.P0, tri.P2}
	if !approxEqualTriangleRel(tri.CircumscribedCircle().Radius, -want, triangleRelTol) {
		t.Fatalf("got circumradius %v, want %v", tri.CircumscribedCircle().Radius, -want)
	}
}

func TestTriangleInscribedCircle(t *testing.T) {
	tri := Triangle{Pt(-4, 1), Pt(-4, -1), Pt(10, 3)}
	got := tri.InscribedCircle()
	if !approxEqualTrianglePoint(got.Center, Pt(-3.0880178529263671, 0.20904207741504303), triangleRelTol) {
		t.Fatalf("got inscribed center %v", got.Center)
	}
	if !approxEqualTriangleRel(got.Radius, 0.91198214707363295, triangleRelTol) {
		t.Fatalf("got inscribed radius %v", got.Radius)
	}
}

func TestTriangleCircumscribedCircle(t *testing.T) {
	tri := Triangle{Pt(-4, 1), Pt(-4, -1), Pt(10, 3)}
	got := tri.CircumscribedCircle()
	if !approxEqualTrianglePoint(got.Center, Pt(3.2857142857142857, 0), triangleRelTol) {
		t.Fatalf("got circumscribed center %v", got.Center)
	}
	if !approxEqualTriangleRel(got.Radius, 7.3540215292764288, triangleRelTol) {
		t.Fatalf("got circumscribed radius %v", got.Radius)
	}
}
