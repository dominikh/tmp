package mathutil

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"
)

func TestDet2x2(t *testing.T) {
	num := func() float64 {
		// const minExp = -1022 // no subnormals
		const minExp = -1073 // all subnormals
		const maxExp = 1023

		exp := rand.IntN(maxExp-minExp) + minExp
		m := (rand.Float64()*2 - 1)
		return math.Ldexp(m, exp)
	}

	var nans, infs int
	var n int
	for ; n-nans < 1_000_000; n++ {
		a, b, c, d := num(), num(), num(), num()
		ba := big.NewFloat(a).SetPrec(128)
		bb := big.NewFloat(b).SetPrec(128)
		bc := big.NewFloat(c).SetPrec(128)
		bd := big.NewFloat(d).SetPrec(128)

		want := new(big.Float).Sub(
			new(big.Float).Mul(ba, bd),
			new(big.Float).Mul(bb, bc))

		got := Det2x2(a, b, c, d)
		if math.IsNaN(got) {
			// inputs caused overflow
			nans++
			continue
		}
		if math.IsInf(got, 0) {
			infs++
		}
		const maxULP = 1.5
		if ulp := ReferenceULPDiff(want, got); ulp > maxULP {
			t.Fatalf("got %g, want %g, ULP difference %g exceeds tolerance",
				got, want, ulp)
		}
	}
	t.Logf("tested %d calls, got %d NaN and %d ±∞ results", n, nans, infs)
}
