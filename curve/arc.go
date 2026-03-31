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

type Arc struct {
	Center     Point
	Radii      Vec2
	StartAngle Angle
	SweepAngle Angle
	XRotation  Angle
}

var _ Shape = Arc{}
var _ ParametricCurve = Arc{}

func (a Arc) Path(tolerance float64) BezPath { return slices.Collect(a.PathElements(tolerance)) }

func (a Arc) PathElements(tolerance float64) iter.Seq[PathElement] {
	return func(yield func(PathElement) bool) {
		if !yield(MoveTo(a.Start())) {
			return
		}

		scaledError := max(a.Radii.X, a.Radii.Y) / tolerance
		// Number of subdivisions per ellipse based on error tolerance.
		// Note: this may slightly underestimate the error for quadrants.
		nError := max(math.Pow(1.1163*scaledError, 1.0/6.0), 3.999_999)
		n := math.Ceil(nError * math.Abs(clampAngle(a.SweepAngle)) * (1.0 / (2.0 * math.Pi)))
		angleStep := clampAngle(a.SweepAngle) / n
		armLen := math.Copysign((4.0/3.0)*math.Tan(math.Abs(0.25*angleStep)), clampAngle(a.SweepAngle))
		angle0 := normalizeAngle(a.StartAngle)
		p0 := sampleEllipse(a.Radii, a.XRotation, angle0)

		for range int(n) {
			angle1 := angle0 + angleStep
			p1 := p0.Add(sampleEllipse(a.Radii, a.XRotation, angle0+math.Pi/2).Mul(armLen))
			p3 := sampleEllipse(a.Radii, a.XRotation, angle1)
			p2 := p3.Sub(sampleEllipse(a.Radii, a.XRotation, angle1+math.Pi/2).Mul(armLen))

			angle0 = angle1
			p0 = p3

			if !yield(CubicTo(
				a.Center.Translate(p1),
				a.Center.Translate(p2),
				a.Center.Translate(p3),
			)) {
				break
			}
		}
	}
}

// Take the ellipse radii, how the radii are rotated, and the sweep angle, and return a
// point on the ellipse.
func sampleEllipse(radii Vec2, xRotation, angle Angle) Vec2 {
	sin, cos := math.Sincos(angle)
	u := radii.X * cos
	v := radii.Y * sin
	return rotatePt(Vec2{u, v}, xRotation)
}

// Rotate pt about the origin by angle radians.
func rotatePt(pt Vec2, angle Angle) Vec2 {
	sin, cos := math.Sincos(angle)
	return Vec2{
		X: pt.X*cos - pt.Y*sin,
		Y: pt.X*sin + pt.Y*cos,
	}
}

func (a Arc) BoundingBox() Rect {
	panic("not implemented")
}

func (a Arc) Perimeter(accuracy float64) float64 {
	panic("not implemented")
}

func (a Arc) Translate(v Vec2) Arc {
	a.Center = a.Center.Translate(v)
	return a
}

// Reverse returns a copy of this arc in the opposite direction.
//
// The new arc will sweep towards the original arc's start angle.
func (a Arc) Reverse() Arc {
	return Arc{
		Center:     a.Center,
		Radii:      a.Radii,
		StartAngle: normalizeAngle(a.StartAngle + clampAngle(a.SweepAngle)),
		SweepAngle: -clampAngle(a.SweepAngle),
		XRotation:  a.XRotation,
	}
}

func (a Arc) AngleAt(t float64) float64 {
	return normalizeAngle(a.StartAngle + clampAngle(a.SweepAngle)*t)
}

func (a Arc) Eval(t float64) Point {
	return a.Center.Translate(sampleEllipse(a.Radii, a.XRotation, a.AngleAt(t)))
}

func (a Arc) Subsegment(start, end float64) Arc {
	return Arc{
		Center:     a.Center,
		Radii:      a.Radii,
		StartAngle: a.AngleAt(start),
		SweepAngle: clampAngle(clampAngle(a.SweepAngle) * (end - start)),
		XRotation:  a.XRotation,
	}
}

func (a Arc) SubsegmentCurve(start, end float64) ParametricCurve {
	return a.Subsegment(start, end)
}

func (a Arc) Subdivide() (Arc, Arc) {
	return a.Subsegment(0.0, 0.5), a.Subsegment(0.5, 1.0)
}

func (a Arc) SubdivideCurve() (ParametricCurve, ParametricCurve) {
	return a.Subdivide()
}

func (a Arc) Start() Point {
	return a.Eval(0)
}

func (a Arc) End() Point {
	return a.Eval(1)
}
