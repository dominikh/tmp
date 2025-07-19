// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package mathutil

import (
	"math"
	"reflect"

	"golang.org/x/exp/constraints"
)

func Rescale[T constraints.Float](oldStart, oldEnd, newStart, newEnd, v T) T {
	slope := (newEnd - newStart) / (oldEnd - oldStart)
	output := newStart + slope*(v-oldStart)
	return output
}

func Clamp[T constraints.Integer | constraints.Float](x, minv, maxv T) T {
	return min(max(x, minv), maxv)
}

// Lerp linearly interpolates between integer and float types.
func Lerp[T constraints.Integer | constraints.Float](start, end T, t float64) T {
	switch t {
	case 0:
		return start
	case 1:
		return end
	default:
		if rv := reflect.ValueOf(start); rv.CanInt() || rv.CanUint() {
			return (T(math.Round(float64(start) + float64(end-start)*t)))
		} else {
			return (T(float64(start) + float64(end-start)*t))
		}
	}
}
