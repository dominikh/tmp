// SPDX-FileCopyrightText: 2026 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package curve

import "testing"

func TestVec2CrossSign(t *testing.T) {
	v1 := Vec(1, 0)
	v2 := Vec(0, 1)
	c := v1.Cross(v2)
	if c != 1 {
		t.Fatalf("cross(%s, %s) = %g, want 1", v1, v2, c)
	}
}

func TestVec2Turn90(t *testing.T) {
	u := Vec(0.1, 0.2)
	turned := u.Turn90()
	if l0, l1 := u.Hypot(), turned.Hypot(); l0 != l1 {
		t.Fatalf("%g != %g", l0, l1)
	}
}
