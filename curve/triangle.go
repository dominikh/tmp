// SPDX-FileCopyrightText: 2018 Raph Levien
// SPDX-FileCopyrightText: 2026 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT
// SPDX-FileAttributionText: https://github.com/linebender/kurbo

package curve

import (
	"iter"
	"math"
	"slices"
)

type Triangle struct {
	P0 Point
	P1 Point
	P2 Point
}

// EquilateralTriangle is an equilateral triangle with the x-axis unit vector as
// its base.
var EquilateralTriangle = Triangle{
	P0: Pt(0.5, math.Sqrt(3)/2.0),
	P1: Pt(0.0, 0.0),
	P2: Pt(1.0, 0.0),
}

var _ ClosedShape = Triangle{}

func (t Triangle) Contains(pt Point) bool {
	return t.Winding(pt) != 0
}

func (t Triangle) Path(tolerance float64) BezPath {
	return slices.Collect(t.PathElements(tolerance))
}

func (t Triangle) PathElements(tolerance float64) iter.Seq[PathElement] {
	return func(yield func(PathElement) bool) {
		_ = yield(MoveTo(t.P0)) &&
			yield(LineTo(t.P1)) &&
			yield(LineTo(t.P2)) &&
			yield(ClosePath())
	}
}

// Centroid returns the triangle's centroid.
func (t Triangle) Centroid() Point {
	return Point(Vec2(t.P0).Add(Vec2(t.P1)).Add(Vec2(t.P2)).Mul(1.0 / 3.0))
}

// Offsets returns the offset of each vertex from the centroid.
func (t Triangle) Offsets() [3]Vec2 {
	centroid := Vec2(t.Centroid())
	return [3]Vec2{
		Vec2(t.P0).Sub(centroid),
		Vec2(t.P1).Sub(centroid),
		Vec2(t.P2).Sub(centroid),
	}
}

func (t Triangle) Area() float64 {
	return 0.5 * t.P1.Sub(t.P0).Cross(t.P2.Sub(t.P0))
}

// InscribedCircle returns the triangle's inscribed circle.
//
// This is defined as the greatest circle that lies within the triangle.
func (t Triangle) InscribedCircle() Circle {
	ab := t.P0.Distance(t.P1)
	bc := t.P1.Distance(t.P2)
	ac := t.P0.Distance(t.P2)

	perimeterRecip := 1.0 / (ab + bc + ac)
	incenter := Vec2(t.P0).Mul(bc).
		Add(Vec2(t.P1).Mul(ac)).
		Add(Vec2(t.P2).Mul(ab)).
		Mul(perimeterRecip)

	return Circle{
		Center: Point(incenter),
		Radius: 2.0 * t.Area() * perimeterRecip,
	}
}

// CircumscribedCircle returns the triangle's circumscribed circle.
//
// This is defined as the smallest circle that intercepts each vertex of the
// triangle.
func (t Triangle) CircumscribedCircle() Circle {
	b := t.P1.Sub(t.P0)
	c := t.P2.Sub(t.P0)
	bLen2 := b.Hypot2()
	cLen2 := c.Hypot2()
	dRecip := 0.5 / b.Cross(c)

	x := (c.Y*bLen2 - b.Y*cLen2) * dRecip
	y := (b.X*cLen2 - c.X*bLen2) * dRecip
	r := math.Sqrt(bLen2*cLen2) * c.Sub(b).Hypot() * dRecip

	return Circle{
		Center: t.P0.Translate(Vec(x, y)),
		Radius: r,
	}
}

// Inflate expands the triangle by a constant amount in all directions.
func (t Triangle) Inflate(scalar float64) Triangle {
	centroid := t.Centroid()
	return Triangle{
		P0: centroid.Translate(Vec(0.0, scalar)),
		P1: centroid.Translate(VecFromAngle(5.0 * math.Pi / 4).Mul(scalar)),
		P2: centroid.Translate(VecFromAngle(7.0 * math.Pi / 4).Mul(scalar)),
	}
}

func (t Triangle) IsInf() bool {
	return t.P0.IsInf() || t.P1.IsInf() || t.P2.IsInf()
}

func (t Triangle) IsNaN() bool {
	return t.P0.IsNaN() || t.P1.IsNaN() || t.P2.IsNaN()
}

func (t Triangle) Translate(v Vec2) Triangle {
	return Triangle{
		P0: t.P0.Translate(v),
		P1: t.P1.Translate(v),
		P2: t.P2.Translate(v),
	}
}

func (t Triangle) Transform(aff Affine) Triangle {
	return Triangle{
		P0: t.P0.Transform(aff),
		P1: t.P1.Transform(aff),
		P2: t.P2.Transform(aff),
	}
}

func (t Triangle) PathLength(accuracy float64) float64 {
	return t.P0.Distance(t.P1) + t.P1.Distance(t.P2) + t.P2.Distance(t.P0)
}

func (t Triangle) Winding(pt Point) int {
	s0 := signum(t.P1.Sub(t.P0).Cross(pt.Sub(t.P0)))
	s1 := signum(t.P2.Sub(t.P1).Cross(pt.Sub(t.P1)))
	s2 := signum(t.P0.Sub(t.P2).Cross(pt.Sub(t.P2)))

	if s0 == s1 && s1 == s2 {
		return int(s0)
	}
	return 0
}

func (t Triangle) BoundingBox() Rect {
	return Rect{
		X0: min(t.P0.X, min(t.P1.X, t.P2.X)),
		Y0: min(t.P0.Y, min(t.P1.Y, t.P2.Y)),
		X1: max(t.P0.X, max(t.P1.X, t.P2.X)),
		Y1: max(t.P0.Y, max(t.P1.Y, t.P2.Y)),
	}
}
