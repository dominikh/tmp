// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package sliceutil

func Pop[E any, S ~[]E](s S) (E, S, bool) {
	if len(s) == 0 {
		return *new(E), s, false
	}
	e := s[len(s)-1]
	s = s[:len(s)-1]
	return e, s, true
}
