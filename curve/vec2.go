// SPDX-FileCopyrightText: 2018 Raph Levien
// SPDX-FileCopyrightText: 2024 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT
// SPDX-FileAttributionText: https://github.com/linebender/kurbo

package curve

import (
	"fmt"
	"math"
)

type Vec2 struct {
	X float64
	Y float64
}

// Vec returns the vector ⟨x, y⟩.
func Vec(x, y float64) Vec2 {
	return Vec2{
		X: x,
		Y: y,
	}
}

// Splat returns the vector's x and y coordinates.
func (v Vec2) Splat() (float64, float64) {
	return v.X, v.Y
}

func (v Vec2) String() string {
	return fmt.Sprintf("⟨%g, %g⟩", v.X, v.Y)
}

// Dot returns the dot product of v and o.
func (v Vec2) Dot(o Vec2) float64 {
	return v.X*o.X + v.Y*o.Y
}

// Cross returns the cross product of v and o.
//
// This is signed so that (1, 0) × (0, 1) = 1.
func (v Vec2) Cross(o Vec2) float64 {
	return v.X*o.Y - v.Y*o.X
}

// Hypot returns the magnitude of the vector.
func (v Vec2) Hypot() float64 {
	return math.Hypot(v.X, v.Y)
}

// Hypot2 returns the squared magnitude of the vector.
//
// This function is more efficient than squaring the result of [Vec2.Hypot].
func (v Vec2) Hypot2() float64 {
	return v.Dot(v)
}

// Angle returns the angle in radians between the vector and ⟨1, 0⟩ in the positive y
// direction. This is atan2(y, x).
func (v Vec2) Angle() float64 {
	return math.Atan2(v.Y, v.X)
}

// VecFromAngle returns a unit vector of the given angle, which is expressed in radians.
// With θ = 0, the result is the positive x unit vector. At π/2, it is the positive y unit
// vector.
//
// Thus, in a y-down coordinate system (as is common for graphics),
// it is a clockwise rotation, and in y-up (traditional for math), it
// is anti-clockwise.
func VecFromAngle(th float64) Vec2 {
	y, x := math.Sincos(th)
	return Vec2{
		X: x,
		Y: y,
	}
}

// Lerp linearly interpolates between two vectors.
func (v Vec2) Lerp(o Vec2, t float64) Vec2 {
	// v + t * (o-v)
	return v.Add(o.Sub(v).Mul(t))
}

// Normalize returns a vector of magnitude 1.0 with the same angle as o.
// This produces a NaN vector if the magnitude is 0.
func (v Vec2) Normalize() Vec2 {
	return v.Mul(1.0 / v.Hypot())
}

// Round returns a new vector with x and y rounded to the nearest integers.
func (v Vec2) Round() Vec2 {
	return Vec2{
		X: math.Round(v.X),
		Y: math.Round(v.Y),
	}
}

// Ceil returns a new vector with x and y rounded up to the nearest integers.
func (v Vec2) Ceil() Vec2 {
	return Vec2{
		X: math.Ceil(v.X),
		Y: math.Ceil(v.Y),
	}
}

// Floor returns a new vector with x and y rounded down to the nearest integers.
func (v Vec2) Floor() Vec2 {
	return Vec2{
		X: math.Floor(v.X),
		Y: math.Floor(v.Y),
	}
}

// Expand returns a new vector with x and y rounded away from zero to the nearest integers.
func (v Vec2) Expand() Vec2 {
	return Vec2{
		X: expand(v.X),
		Y: expand(v.Y),
	}
}

// Trunc returns a new vector with x and y rounded towards zero to the nearest integers.
func (v Vec2) Trunc() Vec2 {
	return Vec2{
		X: math.Trunc(v.X),
		Y: math.Trunc(v.Y),
	}
}

// IsInf reports whether at least one of x and y is infinite.
func (v Vec2) IsInf() bool {
	return math.IsInf(v.X, 0) || math.IsInf(v.Y, 0)
}

// IsNaN reports whether at least one of x and y is NaN.
func (v Vec2) IsNaN() bool {
	return math.IsNaN(v.X) || math.IsNaN(v.Y)
}

// Add adds two vectors and returns the resulting vector.
func (v Vec2) Add(o Vec2) Vec2 {
	return Vec2{
		X: v.X + o.X,
		Y: v.Y + o.Y,
	}
}

// Sub subtracts two vectors and returns the resulting vector.
func (v Vec2) Sub(o Vec2) Vec2 {
	return Vec2{
		X: v.X - o.X,
		Y: v.Y - o.Y,
	}
}

func (v Vec2) Mul(f float64) Vec2 {
	return Vec2{
		X: v.X * f,
		Y: v.Y * f,
	}
}

func (v Vec2) Div(f float64) Vec2 {
	return Vec2{
		X: v.X / f,
		Y: v.Y / f,
	}
}

// Negate returns a new vector with the signs of x and y flipped.
func (v Vec2) Negate() Vec2 {
	// XXX test that this works correctly for infinities
	return Vec2{
		X: -v.X,
		Y: -v.Y,
	}
}

// Turn90 turns the vector by 90 degrees.
//
// The rotation is clockwise in a Y-down coordinate system.
func (v Vec2) Turn90() Vec2 {
	return Vec2{-v.Y, v.X}
}

// rotateScale combines two vectors interpreted as rotation and scaling.
//
// Interpret both vectors as a rotation and a scale, and combine
// their effects, by adding the angles and multiplying the magnitudes.
// This operation is equivalent to multiplication when the vectors
// are interpreted as complex numbers. It is commutative.
func rotateScale(lhs, rhs Vec2) Vec2 {
	return Vec2{
		lhs.X*rhs.X - lhs.Y*rhs.Y,
		lhs.X*rhs.Y + lhs.Y*rhs.X,
	}
}
