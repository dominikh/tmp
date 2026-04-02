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

type Ellipse struct {
	inner Affine
}

var _ ClosedShape = Ellipse{}

// NewEllipse returns a new ellipse with a given center, radii, and rotation.
//
// The returned ellipse will be the result of taking a circle, stretching
// it by the radii along the x and y axes, then rotating it from the
// x axis by xRotation radians, before finally translating the center
// to center.
//
// Rotation is clockwise in a y-down coordinate system. For more on
// rotation, see [Rotate].
func NewEllipse(center Point, radii Vec2, xRotation Angle) Ellipse {
	rx, ry := radii.Splat()
	return newEllipse(Vec2(center), rx, ry, xRotation)
}

// NewEllipseFromRect returns the largest ellipse that can be bounded by the
// provided rectangle.
//
// This uses the absolute width and height of the rectangle.
//
// This ellipse is always axis-aligned; to apply rotation you can call
// [Ellipse.WithRotation] on the result.
func NewEllipseFromRect(rect Rect) Ellipse {
	center := Vec2(rect.Center())
	width, height := rect.Size().Scale(1.0 / 2.0).Splat()
	return newEllipse(center, width, height, 0.0)
}

// NewEllipseFromAffine returns an ellipse from an affine transformation of the unit
// circle.
func NewEllipseFromAffine(aff Affine) Ellipse {
	return Ellipse{inner: aff}
}

func NewEllipseFromCircle(c Circle) Ellipse {
	return NewEllipse(c.Center, Vec(c.Radius, c.Radius), 0)
}

// WithCenter returns a new ellipse centered on the provided point.
func (e Ellipse) WithCenter(center Point) Ellipse {
	return Ellipse{inner: e.inner.WithTranslation(Vec2(center))}
}

// WithRadii returns a new ellipse, with the radii replaced by the argument.
func (e Ellipse) WithRadii(radii Vec2) Ellipse {
	rotation := e.inner.svd1()
	translation := e.inner.Translation()
	return newEllipse(translation, radii.X, radii.Y, rotation)
}

// WithRotation returns a new ellipse, with the rotation replaced by the
// argument.
//
// The rotation is clockwise, for a y-down coordinate system. For more
// on rotation, See [Rotate].
func (e Ellipse) WithRotation(rotation Angle) Ellipse {
	scale := e.inner.svd0()
	translation := e.inner.Translation()
	return newEllipse(translation, scale.X, scale.Y, rotation)
}

func newEllipse(center Vec2, scaleX, scaleY float64, xRotation Angle) Ellipse {
	// Since the circle is symmetric about the x and y axes, using absolute values for the
	// radii results in the same ellipse. For simplicity we make this change here.
	return Ellipse{
		inner: Translate(center).
			Mul(Rotate(xRotation)).
			Mul(Scale(math.Abs(scaleX), math.Abs(scaleY))),
	}
}

// Contains implements ClosedShape.
func (e Ellipse) Contains(pt Point) bool {
	return e.Winding(pt) != 0
}

func (e Ellipse) IsInf() bool {
	return e.inner.IsInf()
}

func (e Ellipse) IsNaN() bool {
	return e.inner.IsNaN()
}

// Area implements ClosedShape.
func (e Ellipse) Area() float64 {
	// An ellipse is a unit circle transformed by an affine transformation. The
	// unsigned area of a transformed region is area × abs(det(affine)). The
	// area of the unit circle is π.
	return math.Pi * math.Abs(e.inner.Determinant())
}

// BoundingBox implements Shape.
func (e Ellipse) BoundingBox() Rect {
	// Compute a tight bounding box of the ellipse.
	//
	// See https://www.iquilezles.org/articles/ellipses/. We can get the two
	// radius vectors by applying the affine map to the two impulses (1, 0) and (0, 1) which gives
	// (a, b) and (c, d) if the affine map is
	//
	//  a | c | e
	// -----------
	//  b | d | f
	//
	//  We can then use the method in the link with the translation to get the bounding box.

	aff := e.inner.Coefficients()
	a2 := aff[0] * aff[0]
	b2 := aff[1] * aff[1]
	c2 := aff[2] * aff[2]
	d2 := aff[3] * aff[3]
	cx := aff[4]
	cy := aff[5]
	rangeX := math.Sqrt(a2 + c2)
	rangeY := math.Sqrt(b2 + d2)
	return Rect{
		X0: cx - rangeX,
		Y0: cy - rangeY,
		X1: cx + rangeX,
		Y1: cy + rangeY,
	}
}

func (e Ellipse) Path(tolerance float64) BezPath { return slices.Collect(e.PathElements(tolerance)) }

// PathElements implements Shape.
func (e Ellipse) PathElements(tolerance float64) iter.Seq[PathElement] {
	radii, xRotation := e.inner.svd()
	return Arc{
		Center:     e.Center(),
		Radii:      radii,
		StartAngle: 0.0,
		SweepAngle: 2 * math.Pi,
		XRotation:  xRotation,
	}.PathElements(tolerance)
}

// PathLength returns the approximated ellipse perimeter.
//
// This uses a numerical approximation. The absolute error between the calculated perimeter
// and the true perimeter is bounded by `accuracy` (modulo floating point rounding errors).
//
// For circular ellipses (equal horizontal and vertical radii), the calculated perimeter is
// exact.
func (e Ellipse) PathLength(accuracy float64) float64 {
	radii := e.Radii()

	if radii.IsInf() {
		return math.Inf(1)
	}

	// Check for the trivial case where the ellipse has one of its radii
	// equal to 0, i.e., where it describes a line, as the numerical method
	// used breaks down with this extreme.
	if radii.X == 0 || radii.Y == 0 {
		return 4 * max(radii.X, radii.Y)
	}

	// Evaluate an approximation based on a truncated infinite series. If it
	// returns a good enough value, we do not need to iterate.
	if kummerEllipticPerimeterRange(radii) <= accuracy {
		return kummerEllipticPerimeter(radii)
	}

	return agmEllipticPerimeter(accuracy, radii)
}

// kummerEllipticPerimeter calculates circumference C of an ellipse with radii
// (x, y) as the infinite series
//
// C = π (x+y) · ∑ binom(1/2, n)^2 * h^n from n = 0 to ∞
//
// with h = (x - y)^2 / (x + y)^2
// and binom(.,.) the binomial coefficient
//
// as described by Kummer ("Über die Hypergeometrische Reihe", 1837) and
// rediscovered by Linderholm and Segal
// ("An Overlooked Series for the Elliptic Perimeter", 1995).
//
// The series converges very quickly for ellipses with only moderate
// eccentricity (h not close to 1).
//
// The series is truncated to the sixth power, meaning a lower bound on the true
// value is returned. Adding the value of [kummerEllipticPerimeterRange] to the
// value returned by this function calculates an upper bound on the true value.
func kummerEllipticPerimeter(radii Vec2) float64 {
	x, y := radii.Splat()
	h := pow2((x - y) / (x + y))
	h2 := h * h
	h3 := h2 * h
	h4 := h3 * h
	h5 := h4 * h
	h6 := h5 * h

	lower := math.Pi +
		h*(math.Pi/4) +
		h2*(math.Pi/64) +
		h3*(math.Pi/256) +
		h4*(math.Pi*25/16384) +
		h5*(math.Pi*49/65536) +
		h6*(math.Pi*441/1048576)

	return (x + y) * lower
}

// kummerEllipticPerimeterRange this calculates the error range of
// [kummerEllipticPerimeter]. That function returns a lower bound on the true
// value, and though we do not know what the remainder of the infinite series
// sums to, we can calculate an upper bound:
//
// ∑ binom(1/2, n)^2 for n = 0 to inf
//
//	= 1 + (1 / 2‼)² + (1‼ / 4‼)² + (3‼ / 6‼)² + (5‼ / 8‼)² + …
//	= 4 / π
//	with ‼ the [double factorial]
//
// (equation 274 in "Summation of Series", L. B. W. Jolley, 1961).
//
// This means the remainder of the infinite series for C, assuming the series was truncated to the
// mᵗʰ term and h = 1, sums to
// 4 / π - ∑ binom(1/2, n)² for n = 0 to m-1
//
// As 0 ≤ h ≤ 1, this is an upper bound.
//
// [double factorial]: https://en.wikipedia.org/wiki/Double_factorial
func kummerEllipticPerimeterRange(radii Vec2) float64 {
	x, y := radii.Splat()
	h := pow2((x - y) / (x + y))

	const binomSquaredRemainder = 4.0/math.Pi -
		(1.0 +
			1.0/4.0 +
			1.0/64.0 +
			1.0/256.0 +
			25.0/16384.0 +
			49.0/65536.0 +
			441.0/1048576.0)

	return math.Pi * binomSquaredRemainder * pow7(h) * (x + y)
}

// agmEllipticPerimeter calculates circumference C of an ellipse with radii
// (x, y) using the arithmetic-geometric mean, as described in equation 19.8.6 of
// https://web.archive.org/web/20240926233336/https://dlmf.nist.gov/19.8#i.
func agmEllipticPerimeter(accuracy float64, radii Vec2) float64 {
	var x, y float64
	if radii.X >= radii.Y {
		x, y = radii.Splat()
	} else {
		y, x = radii.Splat()
	}

	accuracy = accuracy / (2 * math.Pi * radii.X)

	sum := 1.0
	a := 1.0
	g := y / x
	c := math.Sqrt(1 - pow2(g))
	mul := 0.5

	for {
		c2 := pow2(c)
		// term = 2^(n-1) c_n^2
		term := mul * c2
		sum -= term

		// We have c_(n+1) ≤ 1/2 c_n
		// (for a derivation, see e.g. section 2.1 of  "Elliptic integrals, the
		// arithmetic-geometric mean and the Brent-Salamin algorithm for π" by G.J.O. Jameson:
		// https://web.archive.org/web/20241002140956/https://www.maths.lancs.ac.uk/jameson/ellagm.pdf)
		//
		// Therefore
		// ∑ 2^(i-1) c_i^2 from i = 1 ≤ ∑ 2^(i-1) ((1/2)^i c_0)^2 from i = 1
		//                            = ∑ 2^-(i+1) c_0^2          from i = 1
		//                            = 1/2 c_0^2
		//
		// or, for arbitrary starting point i = n+1:
		// ∑ 2^(i-1) c_i^2 from i = n+1 ≤ ∑ 2^(i-1) ((1/2)^(i-n) c_n)^2 from i = n+1
		//                              = ∑ 2^(2n - i - 1) c_n^2        from i = n+1
		//                              = 2^(2n) ∑ 2^(-(i+1)) c_n^2     from i = n+1
		//                              = 2^(2n) 2^(-(n+1)) c_n^2
		//                              = 2^(n-1) c_n^2
		//
		// Therefore, the remainder of the series sums to less than or equal to 2^(n-1) c_n^2,
		// which is exactly the value of the nth term.
		//
		// Furthermore, a_m ≥ g_n, and g_n ≤ 1, for all m, n.
		if term <= accuracy*g {
			// sum currently overestimates the true value - subtract the upper
			// bound of the remaining series. We will then underestimate the
			// true value, but by no more than 'accuracy'.
			sum -= term
			break
		}

		mul *= 2
		// This is equal to c_next = c^2 / (4 * a_next)
		c = (a - g) / 2
		aNext := (a + g) / 2
		g = math.Sqrt(a * g)
		a = aNext
	}

	return 2 * math.Pi * radii.X / a * sum
}

// Winding implements ClosedShape.
func (e Ellipse) Winding(pt Point) int {
	// Strategy here is to apply the inverse map to the point and see if it is in the unit
	// circle.
	inv := e.inner.Invert()
	if Vec2(pt.Transform(inv)).Hypot2() < 1.0 {
		return 1
	} else {
		return 0
	}
}

// Center returns the center of the ellipse.
func (e Ellipse) Center() Point {
	return Point(e.inner.Translation())
}

// Radii returns the two radii of the ellipse.
//
// The first number is the horizontal radius and the second is the
// vertical radius, before rotation.
func (e Ellipse) Radii() Vec2 {
	radii := e.inner.svd0()
	return radii
}

// Rotation returns the ellipse's rotation, in radians.
func (e Ellipse) Rotation() Angle {
	rot := e.inner.svd1()
	return rot
}

// RadiiRotation returns the radii and the rotation of this ellipse.
//
// This is equivalent to, but more efficiant than, using [Ellipse.Radii] and
// [Ellipse.Rotation].
func (e Ellipse) RadiiRotation() (Vec2, Angle) {
	return e.inner.svd()
}

func (e Ellipse) Translate(v Vec2) Ellipse {
	return Ellipse{
		inner: Translate(v).Mul(e.inner),
	}
}

func (e Ellipse) Transform(aff Affine) Ellipse {
	return Ellipse{
		inner: e.inner.Mul(aff),
	}
}
