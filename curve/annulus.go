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

type Annulus struct {
	Center      Point
	OuterRadius float64
	InnerRadius float64
}

var _ ClosedShape = Annulus{}

// Contains implements ClosedShape.
func (an Annulus) Contains(pt Point) bool {
	return an.Winding(pt) != 0
}

func (an Annulus) Path(tolerance float64) BezPath {
	return slices.Collect(an.PathElements(tolerance))
}

// PathElements implements Shape.
func (an Annulus) PathElements(tolerance float64) iter.Seq[PathElement] {
	return func(yield func(PathElement) bool) {
		for el := range (Circle{
			Center: an.Center,
			Radius: an.OuterRadius,
		}).PathElements(tolerance) {
			if !yield(el) {
				return
			}
		}

		// OPT(dh): we should implement a version of Circle.Path that emits the
		// path in reverse order so that we don't have to allocate a slice.
		for _, el := range (Circle{
			Center: an.Center,
			Radius: an.InnerRadius,
		}).Path(tolerance).ReverseSubpaths() {
			if !yield(el) {
				return
			}
		}
	}
}

func (an Annulus) IsInf() bool {
	return an.Center.IsInf() ||
		math.IsInf(an.OuterRadius, 0) ||
		math.IsInf(an.InnerRadius, 0)
}

func (an Annulus) IsNaN() bool {
	return an.Center.IsNaN() ||
		math.IsNaN(an.OuterRadius) ||
		math.IsNaN(an.InnerRadius)
}

func (an Annulus) Translate(v Vec2) Annulus {
	an.Center = an.Center.Translate(v)
	return an
}

func (an Annulus) Area() float64 {
	return math.Pi * (an.OuterRadius*an.OuterRadius - an.InnerRadius*an.InnerRadius)
}

func (an Annulus) BoundingBox() Rect {
	r := an.OuterRadius
	x := an.Center.X
	y := an.Center.Y
	return Rect{
		X0: x - r,
		Y0: y - r,
		X1: x + r,
		Y1: y + r,
	}
}

func (an Annulus) Perimeter(accuracy float64) float64 {
	return 2 * math.Pi * (an.OuterRadius + an.InnerRadius)
}

func (an Annulus) Winding(pt Point) int {
	dist2 := pt.Sub(an.Center).Hypot2()
	if dist2 < an.OuterRadius*an.OuterRadius && dist2 > an.InnerRadius*an.InnerRadius ||
		dist2 < an.InnerRadius*an.InnerRadius && dist2 > an.OuterRadius*an.OuterRadius {
		return 1
	} else {
		return 0
	}
}

// Sector returns a sector of the annulus. The start angle gets
// normalized to [0, 2π] and the sweep angle gets clamped to [-2π, 2π].
func (an Annulus) Sector(startAngle, sweepAngle Angle) AnnulusSector {
	return AnnulusSector{
		Center:      an.Center,
		OuterRadius: an.OuterRadius,
		InnerRadius: an.InnerRadius,
		StartAngle:  normalizeAngle(startAngle),
		SweepAngle:  clampAngle(sweepAngle),
	}
}

type AnnulusSector struct {
	Center      Point
	OuterRadius float64
	InnerRadius float64
	StartAngle  Angle
	SweepAngle  Angle
}

var _ ClosedShape = AnnulusSector{}

// Contains implements ClosedShape.
func (as AnnulusSector) Contains(pt Point) bool {
	return as.Winding(pt) != 0
}

func (as AnnulusSector) Path(tolerance float64) BezPath {
	return slices.Collect(as.PathElements(tolerance))
}

// OuterArc returns the arc representing the outer radius.
func (as AnnulusSector) OuterArc() Arc {
	return Arc{
		Center:     as.Center,
		Radii:      Vec2{as.OuterRadius, as.OuterRadius},
		StartAngle: normalizeAngle(as.StartAngle),
		SweepAngle: clampAngle(as.SweepAngle),
		XRotation:  0.0,
	}
}

// InnerArc returns the arc representing the inner radius.
//
// This is in the opposite direction of the outer arc, so that it is in the same
// direction as the arc that would be drawn (as the path elements for this
// annulus sector produce a closed path). See [Arc.Reverse] for reversing the
// arc.
func (as AnnulusSector) InnerArc() Arc {
	sweepAngle := clampAngle(as.SweepAngle)
	return Arc{
		Center:     as.Center,
		Radii:      Vec2{as.InnerRadius, as.InnerRadius},
		StartAngle: normalizeAngle(as.StartAngle + sweepAngle),
		SweepAngle: -sweepAngle,
		XRotation:  0.0,
	}
}

// PathElements implements Shape.
func (as AnnulusSector) PathElements(tolerance float64) iter.Seq[PathElement] {
	if math.Abs(as.SweepAngle) < 2*math.Pi {
		return func(yield func(PathElement) bool) {
			if !yield(MoveTo(pointOnCircle(as.Center, as.InnerRadius, as.StartAngle))) {
				return
			}

			// First radius
			if !yield(LineTo(pointOnCircle(as.Center, as.OuterRadius, as.StartAngle))) {
				return
			}

			// Outer arc
			a := as.OuterArc()
			for el := range dropFirst(a.PathElements(tolerance)) {
				if !yield(el) {
					return
				}
			}

			// Second radius
			if !yield(LineTo(pointOnCircle(as.Center, as.InnerRadius, as.StartAngle+as.SweepAngle))) {
				return
			}

			// Inner arc
			a = as.InnerArc()
			for el := range dropFirst(a.PathElements(tolerance)) {
				if !yield(el) {
					return
				}
			}
		}
	} else {
		return func(yield func(PathElement) bool) {
			// Outer arc
			a := as.OuterArc()
			for el := range a.PathElements(tolerance) {
				if !yield(el) {
					return
				}
			}

			// Inner arc
			a = as.InnerArc()
			for el := range a.PathElements(tolerance) {
				if !yield(el) {
					return
				}
			}
		}
	}
}

func (as AnnulusSector) IsInf() bool {
	return as.Center.IsInf() ||
		math.IsInf(as.OuterRadius, 0) ||
		math.IsInf(as.InnerRadius, 0) ||
		math.IsInf(as.StartAngle, 0) ||
		math.IsInf(as.SweepAngle, 0)
}

func (as AnnulusSector) IsNaN() bool {
	return as.Center.IsNaN() ||
		math.IsNaN(as.OuterRadius) ||
		math.IsNaN(as.InnerRadius) ||
		math.IsNaN(as.StartAngle) ||
		math.IsNaN(as.SweepAngle)
}

func (as AnnulusSector) Translate(v Vec2) AnnulusSector {
	as.Center = as.Center.Translate(v)
	return as
}

func (as AnnulusSector) Area() float64 {
	return 0.5 * math.Abs(as.OuterRadius*as.OuterRadius-as.InnerRadius*as.InnerRadius) * as.SweepAngle
}

func (as AnnulusSector) BoundingBox() Rect {
	// todo this is currently not tight
	r := as.OuterRadius
	x := as.Center.X
	y := as.Center.Y
	return Rect{
		X0: x - r,
		Y0: y - r,
		X1: x + r,
		Y1: y + r,
	}
}

func (as AnnulusSector) Perimeter(accuracy float64) float64 {
	return 2.0*
		(as.OuterRadius-as.InnerRadius) +
		clampAngle(as.SweepAngle)*(as.InnerRadius+as.OuterRadius)
}

func (as AnnulusSector) Winding(pt Point) int {
	angle := pt.Sub(as.Center).Angle()
	if angle < normalizeAngle(as.StartAngle) ||
		angle > normalizeAngle(as.StartAngle+clampAngle(as.SweepAngle)) {
		return 0
	}
	dist2 := pt.Sub(as.Center).Hypot2()
	if dist2 < as.OuterRadius*as.OuterRadius && dist2 >= 0 {
		return 1
	} else {
		return 0
	}
}
