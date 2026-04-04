// SPDX-FileCopyrightText: 2018 Raph Levien
// SPDX-FileCopyrightText: 2024 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT
// SPDX-FileAttributionText: https://github.com/linebender/kurbo

package curve

import (
	"math"
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

func (c Circle) Path(tolerance float64, out BezPath) BezPath {
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

	out.MoveTo(Pt(x+r, y))
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
		out.CubicTo(
			Pt(x+r*(c0-a*s0), y+r*(s0+a*c0)),
			Pt(x+r*(c1+a*s1), y+r*(s1-a*c1)),
			Pt(x+r*c1, y+r*s1),
		)
	}
	out.ClosePath()
	return out
}

// Sector returns a sector of the circle. The start angle gets
// normalized to [0, 2π] and the sweep angle gets clamped to [-2π, 2π].
func (c Circle) Sector(startAngle, sweepAngle Angle) CircleSector {
	return CircleSector{
		Center:     c.Center,
		Radius:     c.Radius,
		StartAngle: normalizeAngle(startAngle),
		SweepAngle: clampAngle(sweepAngle),
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

func (c Circle) PathLength(accuracy float64) float64 {
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
	StartAngle Angle
	SweepAngle Angle
}

var _ ClosedShape = CircleSector{}
var _ ParametricCurve = CircleSector{}

// Contains implements ClosedShape.
func (cs CircleSector) Contains(pt Point) bool {
	return cs.Winding(pt) != 0
}

func (cs CircleSector) Arc() Arc {
	return Arc{
		Center:     cs.Center,
		Radii:      Vec2{cs.Radius, cs.Radius},
		StartAngle: normalizeAngle(cs.StartAngle),
		SweepAngle: clampAngle(cs.SweepAngle),
		XRotation:  0.0,
	}
}

// Path implements Shape.
func (cs CircleSector) Path(tolerance float64, out BezPath) BezPath {
	if math.Abs(cs.SweepAngle) < 2*math.Pi {
		out.MoveTo(cs.Center)

		out = appendPathReplaceMoveTo(
			out,
			LineTo(pointOnCircle(cs.Center, cs.Radius, cs.StartAngle)),
			tolerance,
			cs.Arc(),
		)
		out.LineTo(cs.Center)
		return out
	} else {
		return cs.Arc().Path(tolerance, out)
	}
}

func pointOnCircle(center Point, radius float64, angle Angle) Point {
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
	return 0.5 * cs.Radius * cs.Radius * clampAngle(cs.SweepAngle)
}

func (cs CircleSector) BoundingBox() Rect {
	sweepAngle := clampAngle(cs.SweepAngle)
	containsAngle := func(a Angle) bool {
		if sweepAngle >= 0 {
			return normalizeAngle(a-cs.StartAngle) <= sweepAngle
		}
		return normalizeAngle(cs.StartAngle-a) <= -sweepAngle
	}

	bbox := Rect{1, 1, 0, 0}.
		UnionPoint(cs.Center).
		UnionPoint(cs.Eval(0)).
		UnionPoint(cs.Eval(1))

	for _, a := range []Angle{0, math.Pi / 2, math.Pi, 3 * math.Pi / 2} {
		if containsAngle(a) {
			bbox = bbox.UnionPoint(pointOnCircle(cs.Center, cs.Radius, a))
		}
	}

	return bbox
}

func (cs CircleSector) PathLength(accuracy float64) float64 {
	return 2.0*cs.Radius + clampAngle(cs.SweepAngle)*cs.Radius
}

func (cs CircleSector) Winding(pt Point) int {
	angle := pt.Sub(cs.Center).Angle()
	if angle < normalizeAngle(cs.StartAngle) || angle > normalizeAngle(cs.StartAngle+clampAngle(cs.SweepAngle)) {
		return 0
	}
	dist2 := pt.Sub(cs.Center).Hypot2()
	if dist2 < cs.Radius*cs.Radius && dist2 >= 0 {
		return 1
	} else {
		return 0
	}
}

// Start implements [ParametricCurve].
func (cs CircleSector) Start() Point {
	return cs.Eval(0)
}

// End implements [ParametricCurve].
func (cs CircleSector) End() Point {
	return cs.Eval(1)
}

func (cs CircleSector) AngleAt(t float64) Angle {
	return normalizeAngle(cs.StartAngle + clampAngle(cs.SweepAngle)*t)
}

// Eval implements [ParametricCurve].
func (cs CircleSector) Eval(t float64) Point {
	a := cs.AngleAt(t)
	return pointOnCircle(cs.Center, cs.Radius, a)
}

func (cs CircleSector) Subdivide() (CircleSector, CircleSector) {
	return cs.Subsegment(0.0, 0.5), cs.Subsegment(0.5, 1.0)
}

// SubdivideCurve implements [ParametricCurve].
func (cs CircleSector) SubdivideCurve() (ParametricCurve, ParametricCurve) {
	return cs.Subdivide()
}

func (cs CircleSector) Subsegment(start float64, end float64) CircleSector {
	return CircleSector{
		Center:     cs.Center,
		Radius:     cs.Radius,
		StartAngle: cs.AngleAt(start),
		SweepAngle: clampAngle(cs.SweepAngle) * (end - start),
	}
}

// SubsegmentCurve implements [ParametricCurve].
func (cs CircleSector) SubsegmentCurve(start float64, end float64) ParametricCurve {
	return cs.Subsegment(start, end)
}
