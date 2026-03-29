// SPDX-FileCopyrightText: 2018 Raph Levien
// SPDX-FileCopyrightText: 2024 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT
// SPDX-FileAttributionText: https://github.com/linebender/kurbo

package curve

import (
	"iter"
	"math"
	"slices"
)

type Circle struct {
	Center Point
	Radius float64
}

var _ ClosedShape = Circle{}

// Contains implements ClosedShape.
func (c Circle) Contains(pt Point) bool {
	return c.Winding(pt) != 0
}

func (c Circle) Path(tolerance float64) BezPath { return slices.Collect(c.PathElements(tolerance)) }

func (c Circle) PathElements(tolerance float64) iter.Seq[PathElement] {
	return func(yield func(PathElement) bool) {
		scaledError := math.Abs(c.Radius) / tolerance
		var n int
		var armLength float64
		if scaledError < 1.0/1.9608e-4 {
			// Solution from http://spencermortensen.com/articles/bezier-circle/
			n = 4
			armLength = 0.551915024494
		} else {
			// This is empirically determined to fall within error tolerance.
			n = int(math.Ceil(math.Pow(1.1163*scaledError, 1.0/6.0)))
			// Note: this isn't minimum error, but it is simple and we can easily
			// estimate the error.
			armLength = (4.0 / 3.0) * math.Tan(math.Pi/2/(float64(n)))
		}

		x, y := c.Center.Splat()
		r := c.Radius
		if !yield(MoveTo(Pt(x+r, y))) {
			return
		}
		deltaTh := 2.0 * math.Pi / float64(n)
		for ix := 1; ix <= n; ix++ {
			a := armLength
			th1 := deltaTh * float64(ix)
			th0 := th1 - deltaTh
			s0, c0 := math.Sincos(th0)
			var s1, c1 float64
			if ix == n {
				s1 = 0.0
				c1 = 1.0
			} else {
				s1, c1 = math.Sincos(th1)
			}
			if !yield(CubicTo(
				Pt(x+r*(c0-a*s0), y+r*(s0+a*c0)),
				Pt(x+r*(c1+a*s1), y+r*(s1-a*c1)),
				Pt(x+r*c1, y+r*s1),
			)) {
				return
			}
		}
		if !yield(ClosePath()) {
			return
		}
	}
}

func (c Circle) Sector(startAngle, sweepAngle float64) CircleSector {
	return CircleSector{
		Center:     c.Center,
		Radius:     c.Radius,
		StartAngle: startAngle,
		SweepAngle: sweepAngle,
	}
}

func (c Circle) IsInf() bool {
	return c.Center.IsInf() || math.IsInf(c.Radius, 0)
}

func (c Circle) IsNaN() bool {
	return c.Center.IsNaN() || math.IsNaN(c.Radius)
}

func (c Circle) Translate(v Vec2) Circle {
	return Circle{
		Center: c.Center.Translate(v),
		Radius: c.Radius,
	}
}

func (c Circle) Area() float64 {
	return math.Pi * c.Radius * c.Radius
}

func (c Circle) BoundingBox() Rect {
	r := math.Abs(c.Radius)
	x := c.Center.X
	y := c.Center.Y
	return Rect{
		X0: x - r,
		Y0: y - r,
		X1: x + r,
		Y1: y + r,
	}
}

func (c Circle) Perimeter(accuracy float64) float64 {
	return math.Abs(2 * math.Pi * c.Radius)
}

func (c Circle) Winding(pt Point) int {
	if pt.Sub(c.Center).Hypot2() < c.Radius*c.Radius {
		return 1
	} else {
		return 0
	}
}

func (c Circle) Transform(aff Affine) Ellipse {
	return NewEllipseFromCircle(c).Transform(aff)
}

type CircleSector struct {
	Center     Point
	Radius     float64
	StartAngle float64
	SweepAngle float64
}

var _ ClosedShape = CircleSector{}

// Contains implements ClosedShape.
func (cs CircleSector) Contains(pt Point) bool {
	return cs.Winding(pt) != 0
}

func (cs CircleSector) Path(tolerance float64) BezPath {
	return slices.Collect(cs.PathElements(tolerance))
}

func (cs CircleSector) Arc() Arc {
	return Arc{
		Center:     cs.Center,
		Radii:      Vec2{cs.Radius, cs.Radius},
		StartAngle: cs.StartAngle,
		SweepAngle: cs.SweepAngle,
		XRotation:  0.0,
	}
}

// PathElements implements Shape.
func (cs CircleSector) PathElements(tolerance float64) iter.Seq[PathElement] {
	return func(yield func(PathElement) bool) {
		if !yield(MoveTo(cs.Center)) {
			return
		}
		if !yield(LineTo(pointOnCircle(cs.Center, cs.Radius, cs.StartAngle))) {
			return
		}
		a := cs.Arc()
		for el := range dropFirst(a.PathElements(tolerance)) {
			if !yield(el) {
				return
			}
		}
		if !yield(LineTo(cs.Center)) {
			return
		}
	}
}

func pointOnCircle(center Point, radius float64, angle float64) Point {
	sin, cos := math.Sincos(angle)
	return center.Translate(
		Vec2{
			X: cos * radius,
			Y: sin * radius,
		})
}

func (cs CircleSector) IsInf() bool {
	return cs.Center.IsInf() ||
		math.IsInf(cs.Radius, 0) ||
		math.IsInf(cs.StartAngle, 0) ||
		math.IsInf(cs.SweepAngle, 0)
}

func (cs CircleSector) IsNaN() bool {
	return cs.Center.IsNaN() ||
		math.IsNaN(cs.Radius) ||
		math.IsNaN(cs.StartAngle) ||
		math.IsNaN(cs.SweepAngle)
}

func (cs CircleSector) Translate(v Vec2) CircleSector {
	cs.Center = cs.Center.Translate(v)
	return cs
}

func (cs CircleSector) Area() float64 {
	return 0.5 * cs.Radius * cs.Radius * cs.SweepAngle
}

func (cs CircleSector) BoundingBox() Rect {
	// todo this is currently not tight
	r := cs.Radius
	x := cs.Center.X
	y := cs.Center.Y
	return Rect{
		X0: x - r,
		Y0: y - r,
		X1: x + r,
		Y1: y + r,
	}
}

func (cs CircleSector) Perimeter(accuracy float64) float64 {
	return 2.0*cs.Radius + cs.SweepAngle*cs.Radius
}

func (cs CircleSector) Winding(pt Point) int {
	angle := pt.Sub(cs.Center).Angle()
	if angle < cs.StartAngle || angle > cs.StartAngle+cs.SweepAngle {
		return 0
	}
	dist2 := pt.Sub(cs.Center).Hypot2()
	if dist2 < cs.Radius*cs.Radius && dist2 >= 0 {
		return 1
	} else {
		return 0
	}
}
