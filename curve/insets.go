// SPDX-FileCopyrightText: 2018 Raph Levien
// SPDX-FileCopyrightText: 2026 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT
// SPDX-FileAttributionText: https://github.com/linebender/kurbo

package curve

import "math"

// Insets describes the distances between the edges of two rectangles.
//
// Positive values increase the distance from the center of a rectangle to the
// relevant edge; negative values decrease it.
type Insets struct {
	X0, Y0 float64
	X1, Y1 float64
}

// NewUniformInsets returns insets with the same value on all sides.
func NewUniformInsets(d float64) Insets {
	return Insets{X0: d, Y0: d, X1: d, Y1: d}
}

// NewUniformInsetsXY returns insets with uniform values along each axis.
func NewUniformInsetsXY(x, y float64) Insets {
	return Insets{X0: x, Y0: y, X1: x, Y1: y}
}

func NewInsets(x0, y0, x1, y1 float64) Insets {
	return Insets{X0: x0, Y0: y0, X1: x1, Y1: y1}
}

// Delta returns the total delta represented by the insets.
func (in Insets) Delta() Size {
	return Sz(in.X0+in.X1, in.Y0+in.Y1)
}

// IsNonNegative reports whether all inset values are non-negative.
func (in Insets) IsNonNegative() bool {
	return in.X0 >= 0 && in.Y0 >= 0 && in.X1 >= 0 && in.Y1 >= 0
}

// ClampToNonNegative returns a copy of the insets with negative values replaced
// by 0.
func (in Insets) ClampToNonNegative() Insets {
	return Insets{
		X0: max(in.X0, 0),
		Y0: max(in.Y0, 0),
		X1: max(in.X1, 0),
		Y1: max(in.Y1, 0),
	}
}

// IsInf reports whether at least one inset value is infinite.
func (in Insets) IsInf() bool {
	return math.IsInf(in.X0, 0) || math.IsInf(in.Y0, 0) || math.IsInf(in.X1, 0) || math.IsInf(in.Y1, 0)
}

// IsNaN reports whether at least one inset value is NaN.
func (in Insets) IsNaN() bool {
	return math.IsNaN(in.X0) || math.IsNaN(in.Y0) || math.IsNaN(in.X1) || math.IsNaN(in.Y1)
}

// Min returns the component-wise minimum of two insets.
func (in Insets) Min(o Insets) Insets {
	return Insets{
		X0: min(in.X0, o.X0),
		Y0: min(in.Y0, o.Y0),
		X1: min(in.X1, o.X1),
		Y1: min(in.Y1, o.Y1),
	}
}

// Max returns the component-wise maximum of two insets.
func (in Insets) Max(o Insets) Insets {
	return Insets{
		X0: max(in.X0, o.X0),
		Y0: max(in.Y0, o.Y0),
		X1: max(in.X1, o.X1),
		Y1: max(in.Y1, o.Y1),
	}
}

// Negate returns a copy of the insets with all signs flipped.
func (in Insets) Negate() Insets {
	return Insets{X0: -in.X0, Y0: -in.Y0, X1: -in.X1, Y1: -in.Y1}
}

// Add adds two insets component-wise.
func (in Insets) Add(o Insets) Insets {
	return Insets{
		X0: in.X0 + o.X0,
		Y0: in.Y0 + o.Y0,
		X1: in.X1 + o.X1,
		Y1: in.Y1 + o.Y1,
	}
}

// Sub subtracts two insets component-wise.
func (in Insets) Sub(o Insets) Insets {
	return Insets{
		X0: in.X0 - o.X0,
		Y0: in.Y0 - o.Y0,
		X1: in.X1 - o.X1,
		Y1: in.Y1 - o.Y1,
	}
}

// Scale scales all inset values by f.
func (in Insets) Scale(f float64) Insets {
	return Insets{
		X0: in.X0 * f,
		Y0: in.Y0 * f,
		X1: in.X1 * f,
		Y1: in.Y1 * f,
	}
}

// Apply applies the insets to r.
//
// The operation is performed on r.Abs(), so it does not preserve negative width
// or height in the input rectangle.
func (in Insets) Apply(r Rect) Rect {
	r = r.Abs()
	return Rect{
		X0: r.X0 - in.X0,
		Y0: r.Y0 - in.Y0,
		X1: r.X1 + in.X1,
		Y1: r.Y1 + in.Y1,
	}
}
