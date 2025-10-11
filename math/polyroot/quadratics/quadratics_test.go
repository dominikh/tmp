// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package quadratics

import (
	"flag"
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"honnef.co/go/stuff/math/mathutil"
)

var dontSkip = flag.Bool("dont-skip", false, "Don't skip test×solver pairs that are known to fail")

var solvers = []struct {
	name string
	fn   func(a, b, c float64) (r1, r2 float64, n int)
}{
	{"Yuksel", Yuksel},
	{"Yuksel2", Yuksel2},
	{"Yuksel3", Yuksel3},
	{"Panchekha", Panchekha},
	{"Lomont", Lomont},
	{"Goualard", Goualard},
	{"BigFloat", func(a, b, c float64) (r1 float64, r2 float64, n int) {
		nf := func(f float64) *big.Float {
			if math.IsNaN(f) {
				return nil
			}
			return big.NewFloat(f).SetPrec(128)
		}
		x1, x2, n := BigFloat(nf(a), nf(b), nf(c))
		switch n {
		case 0:
			return math.NaN(), math.NaN(), 0
		case 1:
			f1, _ := x1.Float64()
			return f1, math.NaN(), 1
		case 2:
			f1, _ := x1.Float64()
			f2, _ := x2.Float64()
			return f1, f2, 2
		default:
			panic("unreachable")
		}
	}},
}

func skipf(t *testing.T, f string, args ...any) {
	t.Helper()

	if !*dontSkip {
		t.Skipf(f, args...)
	}
}

func TestSpecialCases(t *testing.T) {
	type rule struct {
		skip           bool
		ignoreZeroSign bool
		maxUlp         float64
	}

	type test struct {
		coeffs [3]float64
		r1, r2 *big.Float
		rules  map[string]rule
	}

	bf := func(s string) *big.Float {
		b, _, err := big.ParseFloat(s, 10, 128, big.ToNearestEven)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	tests := []test{
		// Hard to believe but true, several "robust" implementations don't
		// handle degenerates correctly.
		{
			[3]float64{3, 0, 0},
			bf("0"),
			nil,
			map[string]rule{
				"Yuksel":  {ignoreZeroSign: true},
				"Yuksel2": {ignoreZeroSign: true},
				"Yuksel3": {ignoreZeroSign: true},
				// Panchekha returns no solutions for 3x².
				"Panchekha": {skip: true},
			},
		},

		{
			[3]float64{0, 3, 0},
			bf("0"),
			nil,
			map[string]rule{
				// Yuksel returns too many solutions for 3x.
				"Yuksel":    {skip: true},
				"Yuksel2":   {ignoreZeroSign: true},
				"Yuksel3":   {ignoreZeroSign: true},
				"Lomont":    {ignoreZeroSign: true},
				"Panchekha": {ignoreZeroSign: true},
			},
		},

		{
			[3]float64{0, 0, 3},
			nil,
			nil,
			map[string]rule{
				// Panchekha and Lomont return -∞ as a solution to 3=0.
				"Lomont":    {skip: true},
				"Panchekha": {skip: true},
			},
		},

		{
			[3]float64{0, 1, 2},
			bf("-2.0"),
			nil,
			map[string]rule{
				// Yuksel returns two roots, -∞ and -2.
				"Yuksel": {skip: true},
			},
		},

		{
			[3]float64{0, 3, 4},
			// -4.0 / 3.0
			bf("-1.333333333333333333333333333333333333335"),
			nil,
			map[string]rule{
				// Yuksel returns two roots, -∞ and -1.(3).
				"Yuksel": {skip: true},
			},
		},

		{
			[3]float64{0, 0, 0},
			bf("Inf"),
			bf("-Inf"),
			map[string]rule{
				// Only Goualard implements a way to indicate "all reals" as the solution.
				"Yuksel":    {skip: true},
				"Yuksel2":   {skip: true},
				"Yuksel3":   {skip: true},
				"Panchekha": {skip: true},
				"Lomont":    {skip: true},
			},
		},

		// This used to creash in Lomont (and still does upstream).
		{
			[3]float64{-1, 0, 0},
			bf("0"),
			nil,
			map[string]rule{
				// Panchekha gets it wrong and returns (0, NaN, 1).
				"Panchekha": {skip: true},
			},
		},

		{
			[3]float64{0, math.Ldexp(1, 600), math.Ldexp(-1, 600)},
			bf("1.0"),
			nil,
			map[string]rule{
				"Yuksel":  {skip: true},
				"Yuksel2": {skip: true},
				"Yuksel3": {skip: true},
			},
		},

		{
			[3]float64{0, math.Ldexp(1, 600), math.Ldexp(1, 600)},
			bf("-1.0"),
			nil,
			map[string]rule{
				"Yuksel":  {skip: true},
				"Yuksel2": {skip: true},
				"Yuksel3": {skip: true},
			},
		},

		{
			[3]float64{0, math.Ldexp(1, -600), math.Ldexp(1, 600)},
			bf("-Inf"),
			nil,
			map[string]rule{
				"Yuksel":  {skip: true},
				"Yuksel2": {skip: true},
				"Yuksel3": {skip: true},
			},
		},

		{
			[3]float64{0, math.Ldexp(1, 600), math.Ldexp(1, -600)},
			bf("-0"),
			nil,
			map[string]rule{
				"Yuksel":  {skip: true},
				"Yuksel2": {skip: true},
				"Yuksel3": {skip: true},
			},
		},

		{
			[3]float64{0, 2, -1.0e-323},
			bf("5.0e-324"),
			nil,
			map[string]rule{
				"Yuksel": {skip: true},
			},
		},

		{
			[3]float64{2, 0, -3},
			// -sqrt(1.5)
			bf("-1.224744871391589049098642037352945695983"),
			// sqrt(1.5)
			bf("1.224744871391589049098642037352945695983"),
			map[string]rule{
				"Yuksel":    {maxUlp: 1},
				"Yuksel2":   {maxUlp: 1},
				"Yuksel3":   {maxUlp: 1},
				"Panchekha": {maxUlp: 1},
			},
		},

		{
			[3]float64{math.Ldexp(1, 600), 0, math.Ldexp(-1, 600)},
			bf("-1.0"),
			bf("1.0"),
			map[string]rule{
				"Yuksel": {skip: true},
			},
		},

		{
			[3]float64{3, 2, 0},
			// -2/3
			bf("-0.666666666666666666666666666666666666668"),
			bf("0"),
			map[string]rule{
				"Yuksel":    {ignoreZeroSign: true},
				"Yuksel2":   {ignoreZeroSign: true},
				"Yuksel3":   {ignoreZeroSign: true},
				"Panchekha": {ignoreZeroSign: true},
			},
		},

		{
			[3]float64{math.Ldexp(1, 600), math.Ldexp(1, 700), 0},
			big.NewFloat(math.Ldexp(-1, 100)),
			bf("0"),
			map[string]rule{
				"Yuksel":    {skip: true},
				"Yuksel2":   {ignoreZeroSign: true},
				"Yuksel3":   {ignoreZeroSign: true},
				"Panchekha": {ignoreZeroSign: true},
			},
		},

		{
			[3]float64{math.Ldexp(1, -600), math.Ldexp(1, 700), 0},
			bf("-Inf"),
			bf("0"),
			map[string]rule{
				"Yuksel":    {ignoreZeroSign: true},
				"Yuksel2":   {ignoreZeroSign: true},
				"Yuksel3":   {ignoreZeroSign: true},
				"Panchekha": {ignoreZeroSign: true},
			},
		},

		{
			[3]float64{math.Ldexp(1, 600), math.Ldexp(1, -700), 0},
			bf("-0"),
			bf("0"),
			map[string]rule{
				"Yuksel":    {skip: true},
				"Yuksel2":   {skip: true},
				"Yuksel3":   {skip: true},
				"Panchekha": {ignoreZeroSign: true},
			},
		},

		// Test cases that all have two solutions, from Goualard.
		{
			[3]float64{1, 1.0000000000000002, 0.2500000000000001},
			bf("-0.5000000000000002"),
			bf("-0.5"),
			map[string]rule{
				"Yuksel":  {skip: true},
				"Yuksel2": {skip: true},
			},
		},

		{
			[3]float64{1, -1, -1},
			// 0.5 - math.Sqrt(5)/2,
			bf("-0.61803398874989484820458683436563811772"),
			// (1 + math.Sqrt(5)) / 2,
			bf("1.61803398874989484820458683436563811772"),
			map[string]rule{
				"Yuksel":    {maxUlp: 1},
				"Yuksel2":   {maxUlp: 1},
				"Yuksel3":   {maxUlp: 1},
				"Panchekha": {maxUlp: 1},
				"Lomont":    {maxUlp: 1},
				"Goualard":  {maxUlp: 1},
			},
		},

		{
			[3]float64{
				1,
				math.Ldexp(1, -511) + math.Ldexp(1, -563),
				math.Ldexp(1, -1024),
			},
			// According to mpsolve -as -Ga -o 20
			bf("-0.74583409066468525558025386e-154"),
			bf("-0.74583405557535644441974614e-154"),
			map[string]rule{
				// This test has excessive error with every solver. We don't
				// trust mpsolve's result.
				"Yuksel":    {skip: true},
				"Yuksel2":   {skip: true},
				"Yuksel3":   {skip: true},
				"Panchekha": {skip: true},
				"Lomont":    {skip: true},
				"Goualard":  {skip: true},
				"BigFloat":  {skip: true},
			},
		},

		{
			[3]float64{1, math.Ldexp(1, 27), 0.75},
			bf("-134217728.0"), bf("-5.587935447692871e-09"),
			nil,
		},
		{
			[3]float64{1, -1e9, 1},
			bf("1e-09"),
			bf("1000000000.0"),
			nil,
		},

		{
			[3]float64{
				1.3407807929942596e154,
				-1.3407807929942596e154,
				-1.3407807929942596e154,
			},
			// 0.5 - math.Sqrt(5)/2,
			bf("-0.61803398874989484820458683436563811772"),
			// (1 + math.Sqrt(5)) / 2,
			bf("1.61803398874989484820458683436563811772"),
			map[string]rule{
				"Yuksel":    {skip: true},
				"Yuksel2":   {maxUlp: 1},
				"Yuksel3":   {maxUlp: 1},
				"Panchekha": {maxUlp: 1},
				"Lomont":    {maxUlp: 1},
				"Goualard":  {maxUlp: 1},
			},
		},

		{
			[3]float64{
				math.Ldexp(1, 600),
				0.5,
				math.Ldexp(-1, -600),
			},
			bf("-3.086568504549085e-181"),
			bf("1.8816085719976428e-181"),
			nil,
		},

		{
			[3]float64{
				math.Ldexp(1, 600),
				0.5,
				math.Ldexp(-1, 600),
			},
			bf("-1.0"),
			bf("1.0"),
			map[string]rule{
				"Yuksel": {skip: true},
			},
		},

		{
			[3]float64{
				8.0,
				math.Ldexp(1, 800),
				math.Ldexp(-1, 500),
			},
			bf("-8.335018041099818e+239"),
			bf("4.909093465297727e-91"),
			map[string]rule{
				"Yuksel": {skip: true},
			},
		},

		{
			[3]float64{
				1.0,
				math.Ldexp(1, 26),
				-0.125,
			},
			bf("-67108864.0"),
			bf("1.862645149230957e-09"),
			nil,
		},

		{
			[3]float64{
				math.Ldexp(1, -1073),
				math.Ldexp(-1, -1073),
				math.Ldexp(-1, -1073),
			},
			// 0.5 - math.Sqrt(5)/2,
			bf("-0.61803398874989484820458683436563811772"),
			// (1 + math.Sqrt(5)) / 2,
			bf("1.61803398874989484820458683436563811772"),
			map[string]rule{
				"Yuksel":    {skip: true},
				"Yuksel2":   {skip: true},
				"Yuksel3":   {skip: true},
				"Panchekha": {skip: true},
				"Lomont":    {skip: true},
				"Goualard":  {maxUlp: 1},
			},
		},

		{
			[3]float64{
				math.Ldexp(1, 600),
				math.Ldexp(-1, -600),
				math.Ldexp(-1, -600),
			},
			bf("-2.409919865102884e-181"),
			bf("2.409919865102884e-181"),
			nil,
		},

		{
			[3]float64{
				-158114166017,
				316227766017,
				-158113600000,
			},
			bf("0.999996420200578744200505"),
			bf("1.0"),
			map[string]rule{
				"Yuksel":  {skip: true},
				"Yuksel2": {skip: true},
			},
		},

		{
			[3]float64{
				16, 0, -32,
			},
			bf("-0.141421356237309504880168872421e1"),
			bf("0.141421356237309504880168872421e1"),
			map[string]rule{
				"Yuksel":    {maxUlp: 1},
				"Yuksel2":   {maxUlp: 1},
				"Yuksel3":   {maxUlp: 1},
				"Panchekha": {maxUlp: 1},
			},
		},

		{
			[3]float64{
				6.096731e+118,
				1.3318949e+220,
				1.3318949e+220,
			},
			bf("-2.1846049956935938e+101"),
			bf("-1"),
			map[string]rule{
				"Yuksel": {skip: true},
			},
		},

		{
			[3]float64{
				// Randomly generated roots from fuzz test
				1.560291e-317,
				1.560291e-317,
				-3.769117243149701e-09,
			},
			// According to mpsolve -as -Ga -o 20
			bf("-0.155423620637973416914374380353e155"),
			bf("0.155423620637973416914374380353e155"),
			map[string]rule{
				// This test has excessive error with every solver. We don't
				// trust mpsolve's result.
				"Yuksel":    {skip: true},
				"Yuksel2":   {skip: true},
				"Yuksel3":   {skip: true},
				"Panchekha": {skip: true},
				"Lomont":    {skip: true},
				"Goualard":  {skip: true},
				"BigFloat":  {skip: true},
			},
		},

		{
			[3]float64{3, 2, -1},
			bf("-1"),
			// 1.0 / 3.0,
			bf("0.333333333333333333333333333333333333334"),
			nil,
		},
		{
			[3]float64{3, 0, -3},
			bf("-1"),
			bf("1"),
			map[string]rule{
				"Panchekha": {maxUlp: 1},
			}},

		{
			// https://github.com/ChrisLomont/BetterQuadraticRoots/issues/1
			[3]float64{
				2.3943585916821317e-289,
				-6.826544501426514e+83,
				-3.166675889418798e+94,
			},
			bf("-0.463876839703758057079512669838e11"),
			bf("2.85e372"),
			map[string]rule{
				"Yuksel":    {maxUlp: 1},
				"Yuksel2":   {skip: true},
				"Yuksel3":   {skip: true},
				"Panchekha": {maxUlp: 1},
				"Lomont":    {skip: true},
				"Goualard":  {maxUlp: 1},
				"BigFloat":  {maxUlp: 1},
			},
		},

		{
			[3]float64{1, 0, -5},
			// -sqrt(5)
			bf("-2.23606797749978969640917366873127623544"),
			// sqrt(5)
			bf("2.23606797749978969640917366873127623544"),
			nil,
		},
		{
			[3]float64{1, 0, 5},
			nil,
			nil,
			nil,
		},
		{
			[3]float64{0, 1, 5},
			bf("-5.0"),
			nil,
			map[string]rule{
				"Yuksel": {skip: true},
			},
		},
		{
			[3]float64{1, 2, 1},
			bf("-1.0"),
			nil,
			map[string]rule{
				// Lomont isn't all that wrong, returning (-1, -1)
				"Lomont": {skip: true},
			},
		},

		{
			[3]float64{
				-312499999999.0,
				707106781186.0,
				-400000000000.0,
			},
			// According to mpsolve -as -Ga -o 20
			bf("0.113136939602710943375e1"),
			bf("0.113137230377533133968937e1"),
			map[string]rule{
				"Yuksel":  {maxUlp: 94747.18},
				"Yuksel2": {maxUlp: 94747.18},
			},
		},
		{
			[3]float64{
				-67,
				134,
				-65,
			},
			// According to mpsolve -as -Ga -o 20
			bf("0.8272263148837279780066296998e0"),
			bf("0.11727736851162720219933703e1"),
			map[string]rule{
				"Yuksel":    {maxUlp: 1},
				"Yuksel2":   {maxUlp: 1},
				"Yuksel3":   {maxUlp: 1},
				"Panchekha": {maxUlp: 1},
				"Lomont":    {maxUlp: 1},
				"Goualard":  {maxUlp: 1},
			},
		},
		{
			[3]float64{
				0.247260273973,
				0.994520547945,
				-0.138627953316,
			},
			// According to mpsolve -as -Ga -o 20
			bf("-0.415703002703359216702368661657e1"),
			bf("0.134869362220940790806488106313e0"),
			nil,
		},
		{
			[3]float64{
				1,
				-2300000,
				2e11,
			},
			// According to mpsolve -as -Ga -o 20
			bf("0.90518994979145461414117052424e5"),
			bf("0.220948100502085453858588294758e7"),
			nil,
		},
		{
			// Yuksel and Pavel have a larger relative error for the first root,
			// 1.3597399555105182e-16.
			[3]float64{
				math.Ldexp(1.5, -1026),
				0,
				math.Ldexp(-1, 1022),
			},
			// -2^1024 * sqrt(6) / 3
			bf("-1.467810298172326429616711289610698310006e+308"),
			bf("1.467810298172326429616711289610698310006e+308"),
			map[string]rule{
				"Yuksel":    {maxUlp: 3},
				"Yuksel2":   {maxUlp: 3},
				"Yuksel3":   {maxUlp: 3},
				"Panchekha": {maxUlp: 3},
				"Lomont":    {maxUlp: 2},
				"Goualard":  {maxUlp: 2},
			},
		},
	}

	NaN := math.NaN()
	Inf := math.Inf(1)
	// Any combination of ∞, -∞, and NaN should fail to solve.
	for _, a := range []float64{-Inf, Inf, NaN, 1} {
		for _, b := range []float64{-Inf, Inf, NaN, 1} {
			for _, c := range []float64{-Inf, Inf, NaN, 1} {
				rules := map[string]rule{}
				switch {
				case a == -Inf && b == 1 && c == -Inf:
				case a == +Inf && b == 1 && c == +Inf:
				case a == +Inf && b == 1 && c == 1:
				case a == 1 && b == 1 && c == +Inf:
				case a == 1 && b == 1 && c == 1:
				default:
					rules["Yuksel"] = rule{skip: true}
				}
				switch {
				case a == -Inf && b == 1 && c == +Inf,
					a == -Inf && b == 1 && c == 1,
					a == +Inf && b == 1 && c == -Inf,
					a == 1 && b == -Inf && c == 1,
					a == 1 && b == +Inf && c == 1,
					a == 1 && b == 1 && c == -Inf:
					rules["Panchekha"] = rule{skip: true}
				}
				tests = append(tests, test{[3]float64{a, b, c}, nil, nil, rules})
			}
		}
	}

	for _, solver := range solvers {
		t.Run(fmt.Sprintf("solver=%s", solver.name), func(t *testing.T) {
			for _, tt := range tests {
				a := tt.coeffs[0]
				b := tt.coeffs[1]
				c := tt.coeffs[2]
				r1, r2, n := solver.fn(a, b, c)

				rule := tt.rules[solver.name]
				maxUlp := rule.maxUlp
				if maxUlp == 0 {
					maxUlp = 0.5
				}
				if rule.ignoreZeroSign {
					if tt.r1 != nil && tt.r1.Sign() == 0 && r1 == 0 {
						r1, _ = tt.r1.Float64()
					}
					if tt.r2 != nil && tt.r2.Sign() == 0 && r2 == 0 {
						r2, _ = tt.r2.Float64()
					}
				}

				ulp1 := mathutil.ReferenceULPDiff(tt.r1, r1)
				ulp2 := mathutil.ReferenceULPDiff(tt.r2, r2)

				var wantN int
				if tt.r2 != nil {
					wantN = 2
				} else if tt.r1 != nil {
					wantN = 1
				}
				if ulp1 > maxUlp || ulp2 > maxUlp || n != wantN {
					if rule.skip && !*dontSkip {
						// solver is known to fail this test
					} else {
						wantR1f, _ := tt.r1.Float64()
						wantR2f, _ := tt.r2.Float64()
						t.Errorf(`
coeffs:    %g, %g, %g
want:      %g, %g, %d
want(f64): %g, %g, %d
got:       %g, %g, %d
ulp:       %f, %f`,
							a, b, c,
							tt.r1, tt.r2, wantN,
							wantR1f, wantR2f, wantN,
							r1, r2, n,
							ulp1, ulp2)
					}
				} else {
					if rule.skip {
						// Test didn't fail, so why are we told to skip it?
						t.Errorf("solve(%g, %g, %g) succeeded, but test was marked as skip; why?", a, b, c)
					}
				}
			}
		})
	}
}

func TestFibonacci(t *testing.T) {
	// From "On the Cost of Floating-Point Computation Without Extra-Precise
	// Arithmetic" by Kahan, 2004. Doesn't actually seem all that useful, as
	// only the most basic implementations fail it.

	fibs := make([]int64, 77)
	fibs[1] = 1
	for i := 2; i < len(fibs); i++ {
		fibs[i] = fibs[i-1] + fibs[i-2]
	}

	for _, solver := range solvers {
		t.Run(fmt.Sprintf("solver=%s", solver.name), func(t *testing.T) {
			if solver.name == "Yuksel" || solver.name == "Yuksel2" {
				skipf(t, "solver %s isn't capable of passing this test", solver.name)
			}

			for i := 2; i < len(fibs); i += 2 {
				// 𝑀𝐹ₙx² − 2𝑀𝐹ₙ₋₁𝑥 + 𝑀𝐹ₙ₋₂
				min := int64(1 << 52)
				max := int64(1<<53 - 1)
				R := rand.Int64N(max-min) + min
				M := R / fibs[i]
				c := float64(M * fibs[i-2])
				b := float64(-2 * M * fibs[i-1])
				a := float64(M * fibs[i])

				r1, r2, n := solver.fn(a, b, c)
				want := []float64{
					float64(fibs[i-1]-1) / float64(fibs[i]),
					float64(fibs[i-1]+1) / float64(fibs[i]),
				}
				ulp1 := mathutil.ULPDiff(want[0], r1)
				ulp2 := mathutil.ULPDiff(want[1], r2)
				const maxULP = 1
				const wantN = 2
				if ulp1 > maxULP || ulp2 > maxULP || n != wantN {
					if isFin(r1) && isFin(r2) && isFin(want[0]) && isFin(want[1]) && n == wantN {
						t.Errorf("solve(%g, %g, %g) = (%g, %g, %d), want (%g, %g, %d); ulp=(%d, %d)",
							a, b, c, r1, r2, n, want[0], want[1], wantN, ulp1, ulp2)
					} else {
						t.Errorf("solve(%g, %g, %g) = (%g, %g, %d), want (%g, %g, %d)",
							a, b, c, r1, r2, n, want[0], want[1], wantN)
					}
				}
			}
		})
	}
}

func BenchmarkRandom(b *testing.B) {
	const N = 1_000_000
	quadratics := make([][3]float64, N)
	for i := range N {
		ca, cb, cc := Random()
		quadratics[i] = [3]float64{ca, cb, cc}
	}

	for _, solver := range solvers {
		b.Run(fmt.Sprintf("solver=%s", solver.name), func(b *testing.B) {
			for b.Loop() {
				for _, coeffs := range quadratics {
					solver.fn(coeffs[0], coeffs[1], coeffs[2])
				}
			}
			b.ReportMetric(float64(b.Elapsed())/float64(len(quadratics))/float64(b.N), "ns/poly")
		})
	}

	b.Run("solver=BigFloat", func(b *testing.B) {
		for b.Loop() {
			for _, coeffs := range quadratics {
				BigFloat(
					big.NewFloat(coeffs[0]).SetPrec(53),
					big.NewFloat(coeffs[1]).SetPrec(53),
					big.NewFloat(coeffs[2]).SetPrec(53),
				)
			}
		}
		b.ReportMetric(float64(b.Elapsed())/float64(len(quadratics))/float64(b.N), "ns/poly")
	})
}
