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

func TestInsetsConstructorsAndValues(t *testing.T) {
	diff(t, Insets{2, 2, 2, 2}, NewUniformInsets(2))
	diff(t, Insets{3, 8, 3, 8}, NewUniformInsetsXY(3, 8))

	in := NewInsets(5, 10, -12, 4)
	diff(t, Sz(-7, 14), in.Delta())
}

func TestInsetsNonNegative(t *testing.T) {
	in := NewInsets(-10, 3, -0.2, 4)
	if in.IsNonNegative() {
		t.Fatal("expected insets to contain negative values")
	}

	diff(t, Insets{0, 3, 0, 4}, in.ClampToNonNegative())

	if !NewInsets(0, 3, 0.2, 4).IsNonNegative() {
		t.Fatal("expected insets to be non-negative")
	}
}

func TestInsetsArithmetic(t *testing.T) {
	a := NewInsets(1, 2, 3, 4)
	b := NewInsets(4, 3, 2, 1)

	diff(t, Insets{5, 5, 5, 5}, a.Add(b))
	diff(t, Insets{-3, -1, 1, 3}, a.Sub(b))
	diff(t, Insets{-1, -2, -3, -4}, a.Negate())
	diff(t, Insets{2, 4, 6, 8}, a.Scale(2))
	diff(t, Insets{1, 2, 2, 1}, a.Min(b))
	diff(t, Insets{4, 3, 3, 4}, a.Max(b))
}

func TestInsetsApplyToRect(t *testing.T) {
	rect := NewRectFromOrigin(Pt(0, 0), Sz(10, 10))
	diff(t, Rect{-3, 0, 13, 10}, rect.Inset(NewUniformInsetsXY(3, 0)))
	diff(t, Rect{3, 0, 7, 10}, rect.Inset(NewUniformInsetsXY(-3, 0)))

	reversed := Rect{7, 11, 0, 0}
	diff(t, Rect{0, -1, 7, 12}, reversed.Inset(NewUniformInsetsXY(0, 1)))

	shrunk := Rect{0, 0, 3, 5}.Inset(NewUniformInsetsXY(0, -7))
	diff(t, Rect{0, 7, 3, -2}, shrunk)
	diff(t, -9.0, shrunk.Height())

	in := NewUniformInsetsXY(1, 7)
	insetRect := Rect{0, 0, 5, 11}.Inset(in)
	diff(t, in, insetRect.Sub(Rect{0, 0, 5, 11}))
	diff(t, insetRect, in.Apply(Rect{0, 0, 5, 11}))
}

func TestInsetsIsInfAndIsNaN(t *testing.T) {
	if !NewInsets(math.Inf(1), 0, 0, 0).IsInf() {
		t.Fatal("expected infinite insets to report IsInf")
	}
	if NewInsets(1, 2, 3, 4).IsInf() {
		t.Fatal("expected finite insets not to report IsInf")
	}

	if !NewInsets(math.NaN(), 0, 0, 0).IsNaN() {
		t.Fatal("expected NaN insets to report IsNaN")
	}
	if NewInsets(1, 2, 3, 4).IsNaN() {
		t.Fatal("expected non-NaN insets not to report IsNaN")
	}
}
