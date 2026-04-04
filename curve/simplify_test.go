// SPDX-FileCopyrightText: 2018 Raph Levien
// SPDX-FileCopyrightText: 2024 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT
// SPDX-FileAttributionText: https://github.com/linebender/kurbo

package curve

import (
	"testing"
)

func TestSimplifyLinesCorner(t *testing.T) {
	var p BezPath
	p.MoveTo(Pt(1.0, 2.0))
	p.LineTo(Pt(3.0, 4.0))
	p.LineTo(Pt(10.0, 5.0))
	simplified := Simplify(p, 1.0, DefaultSimplifyOptions, nil)
	diff(t, p, simplified)
}
