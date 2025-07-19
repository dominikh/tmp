// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package iterutil

import "iter"

func MutableValues[E any, S ~[]E](s S) iter.Seq[*E] {
	return func(yield func(*E) bool) {
		for i := range s {
			if !yield(&s[i]) {
				return
			}
		}
	}
}

func Dereference[E any, Ptr ~*E](seq iter.Seq[Ptr]) iter.Seq[E] {
	return func(yield func(E) bool) {
		for el := range seq {
			if !yield(*el) {
				return
			}
		}
	}
}

func WithIndex[E any](seq iter.Seq[E]) iter.Seq2[int, E] {
	return func(yield func(int, E) bool) {
		i := 0
		for el := range seq {
			if !yield(i, el) {
				return
			}
			i++
		}
	}
}
