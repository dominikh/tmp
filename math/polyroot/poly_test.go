// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package polyroot

import (
	"fmt"
	"math"
	"math/big"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMul(t *testing.T) {
	p1 := NewPolynomial(-1, 1)
	p2 := NewPolynomial(-2, 1)
	p3 := NewPolynomial(-3, 1)
	p4 := NewPolynomial(-4, 1)
	p := p1.Mul(p2).Mul(p3).Mul(p4)
	if d := cmp.Diff(p.coeffs, []float64{24, -50, 35, -10, 1}); d != "" {
		t.Fatal(d)
	}
}

func TestDegree(t *testing.T) {
	tests := []struct {
		coeffs []float64
		degree int
	}{
		{[]float64{}, 0},
		{[]float64{0}, 0},
		{[]float64{0, 0}, 0},
		{[]float64{1}, 0},
		{[]float64{1, 0}, 0},
		{[]float64{1, 2, 3, 4}, 3},
		{[]float64{1, 2, 3, 4, 0}, 3},
		{[]float64{1, 2, 3, 4, 0, 0}, 3},
		{[]float64{1, 2, 3, 4, 0, 0, 7}, 6},
	}
	for _, tt := range tests {
		if got := NewPolynomial(tt.coeffs...).Degree(); got != tt.degree {
			t.Errorf("degree(%f) got %d, want %d", tt.coeffs, got, tt.degree)
		}
	}
}

func BenchmarkRoots(b *testing.B) {
	for i := 1; i < 10; i++ {
		b.Run(fmt.Sprintf("degree=%d", i), func(b *testing.B) {
			for _, xError := range []float64{0, 1e-6} {
				b.Run(fmt.Sprintf("xError=%g", xError), func(b *testing.B) {
					coeffs := make([]float64, i+1)
					for j := range coeffs {
						coeffs[j] = float64(j) + 1
					}
					p := NewPolynomial(coeffs...)

					b.Run("bounds=false", func(b *testing.B) {
						out := make([]float64, 0, p.Degree())
						for b.Loop() {
							out = p.Roots(math.Inf(-1), math.Inf(1), xError, out[:0])
						}
					})

					b.Run("bounds=true", func(b *testing.B) {
						out := make([]float64, 0, p.Degree())
						for b.Loop() {
							out = p.Roots(-1, 1, xError, out[:0])
						}
					})
				})
			}
		})
	}
}

func BenchmarkDerivative(b *testing.B) {
	p := NewPolynomial(
		0.8622054392093847,
		0.11789342389174473,
		0.1314766031277056,
		0.3435511173091106,
		0.9773809559880883,
	)
	for b.Loop() {
		p.Derivative()
	}
}

func TestRoots(t *testing.T) {
	type testCase struct {
		label        string
		coefficients []float64
		// Some test cases (such as orellana 1-22) start off with exactly
		// specified roots, from which we compute approximate coefficients. For
		// very ill-conditioned polynomials, the approximation won't yield roots
		// remotely close to the ones we started off, and testCase.roots will
		// reflect that by containing the best root for the approximate
		// coefficients, instead of the roots we started with. nominalRoots, on
		// the other hand, does contain the original roots, purely for
		// documentation purposes.
		nominalRoots []float64
		// The expected roots.
		roots []float64
		// Maximum relative error we tolerate for this polynomial.
		maxRelativeError float64
		// Forced range for the ranged test
		range_ [2]float64
		skip   bool
	}
	tests := []testCase{
		{
			// Four widely spaced roots
			label: "orellana-1",
			coefficients: []float64{
				// Exact coefficients
				1e+18,
				-1.001001001e+18,
				1.001002001001e+15,
				-1.001001001e+09,
				1,
			},
			roots: []float64{
				1, 1e3, 1e6, 1e9,
			},
		},

		{
			// Four closely spaced roots
			label: "orellana-2",
			coefficients: []float64{
				// Approximate coefficients computed from roots
				16.048044012, -32.072044006, 24.036011, -8.006, 1,
			},
			nominalRoots: []float64{
				2, 2.001, 2.002, 2.003,
			},
			roots: []float64{
				// According to mpsolve -as -Ga -o 20
				2, 2.001, 2.002, 2.003,
			},
			maxRelativeError: 2e-6,
		},

		{
			// Four large roots
			label: "orellana-3",
			coefficients: []float64{
				// Approximate coefficients computed from roots
				1e+199, -1.011001e+152, 1.1011011e+103, -1.001101e+53, 1,
			},
			nominalRoots: []float64{
				1e47, 1e49, 1e50, 1e53,
			},
			roots: []float64{
				// According to mpsolve -as -Ga -o 20
				1e47, 1e49, 1e50, 1e53,
			},
		},

		{
			// One large, three small roots
			label: "orellana-4",
			coefficients: []float64{
				// Exact coefficients
				-2e+14, 1.00000000000002e+14, 1.99999999999999e+14, -1.00000000000002e+14, 1,
			},
			roots: []float64{-1, 1, 2, 1e14},
		},

		{
			// Two large, two small roots
			label: "orellana-5",
			coefficients: []float64{
				// Exact coefficients
				2e+14, -1e+07, -2.00000000000001e+14, 1e+07, 1,
			},
			roots: []float64{
				-2e7, -1, 1, 1e7,
			},
		},

		{
			// Quadruple root
			label: "orellana-14",
			coefficients: []float64{
				// Exact coefficients
				1e+12, -4e+09, 6e+06, -4000, 1,
			},
			roots: []float64{1000, 1000, 1000, 1000},
		},

		{
			// Triple and one small root
			label: "orellana-15",
			coefficients: []float64{
				// Exactly specified coefficients for the given roots, but
				// approximate due to floating point rounding. The tiny terms
				// vanish.
				1e-6,
				-1e9 - 3e-9,
				3e6 + 3e-12,
				-3e3 - 1e-15,
				1,
			},
			nominalRoots: []float64{1e-15, 1000, 1000, 1000},
			roots: []float64{
				// According to mpsolve -as -Ga -o 20
				0.1000000000000000003e-14,
				0.999998999999666666333332901e3,
				0.999998999999666666333332901e3,
				0.999998999999666666333332901e3,
			},
		},

		{
			// Four closely spaced large roots
			label: "orellana-17",
			coefficients: []float64{
				// Exact coefficients
				1.011111101e+16, -4.033322201e+12, 6.0333111e+08, -40111, 1,
			},
			roots: []float64{
				10000, 10001, 10010, 10100,
			},
		},

		{
			// Four very widely spaced roots
			label: "orellana-19",
			coefficients: []float64{
				// Approximate coefficients
				1e+100, -1e+100, 2.0000000001e+70, -1.0000000002e+40, 1,
			},
			nominalRoots: []float64{
				1, 1e30, 1e30, 1e40,
			},
			roots: []float64{
				// According to mpsolve -as -Ga -o 20
				0.1e1,
				0.999999999999998999999999949999e30,
				0.100000000000000100000000005e31,
				0.1e41,
			},
			maxRelativeError: 1e-08,
		},

		{
			// Four widely spaced roots
			label: "orellana-20",
			coefficients: []float64{
				// Approximate coefficients
				1e+28, -1.00000020000001e+28, 2.00000020000002e+21, -1.00000020000001e+14, 1,
			},
			nominalRoots: []float64{1, 1e7, 1e7, 1e14},
			roots:        []float64{1, 1e7, 1e7, 1e14},
		},

		{
			// Four widely spaced roots
			label: "orellana-21",
			coefficients: []float64{
				// Approximate coefficients
				1e+29, -1.000000200000001e+29, 2.000000110000002e+22, -1.000000020000001e+15, 1,
			},
			nominalRoots: []float64{1, 1e7, 1e7, 1e15},
			roots: []float64{
				// According to mpsolve -as -Ga -o 20
				1, 1e7, 1e7, 1e15,
			},
			maxRelativeError: 2e-8,
		},

		{
			// Two very large roots
			label: "orellana-22",
			coefficients: []float64{
				// Approximate coefficients
				1e+307, -1.1e+307, 1e+306, -1.01e+154, 1,
			},
			nominalRoots: []float64{
				1, 10, 1e152, 1e154,
			},
			roots: []float64{
				// According to mpsolve -as -Ga -o 20
				1, 10, 1e152, 1e154,
			},
			maxRelativeError: 3e-16,
		},

		{
			label:        "orellana-23",
			coefficients: []float64{1e-3, 3.0 / 8.0, 1, 1, 1},
			roots: []float64{
				(1.0 / 20.0) * (-5 - math.Sqrt(2*math.Sqrt(5585)-125)),
				(1.0 / 20.0) * (math.Sqrt(2*math.Sqrt(5585)-125) - 5),
			},
			maxRelativeError: 2e-14,
		},

		{
			label:        "orellana-24",
			coefficients: []float64{-1e30, 1e60 + 1e30, 1e-30 - 1e60, -(1 + 1e-30), 1},
			roots: []float64{
				-1e30,
				1e-30,
				1,
				1e30,
			},
		},

		{
			label:        "polyroot-1",
			coefficients: []float64{-1.1e307, 2e306, -3.03e154, 4},
			roots: []float64{
				// According to mpsolve -as -Ga -o 20
				5.5,
				0.665920126936758663650655969375e152,
				0.750840798730632413363493440306e154,
			},
			maxRelativeError: 2e-16,
		},

		{
			label: "polyroot-3",
			coefficients: []float64{
				3.964039410107839e+305,
				1.824095849658912e+306,
				1.3240449630966948e+305,
				-1.8637852775024757e+306,
				3.4878340937076723e+305,
			},
			roots: []float64{
				// According to mpsolve -as -Ga -o 20
				-0.75026011514905249667961300481,
				-0.235212698531776171054072139963,
				1.2740243604880595647515231033,
				5.05512371820864611670372748899,
			},
			maxRelativeError: 2e-16,
		},

		{
			label:        "polyroot-4",
			coefficients: []float64{-6, 11, -6, 1},
			roots: []float64{
				// Exact roots
				1, 2, 3,
			},
		},

		{
			label:        "polyroot-5",
			coefficients: []float64{24, -50, 35, -10, 1},
			roots: []float64{
				// Exact roots
				1, 2, 3, 4,
			},
		},

		{
			// This one used to cause an infinite loop when not specifying
			// explicit bounds.
			label: "polyroot-6",
			// Coefficients found via fuzzing
			coefficients: []float64{
				1.398043286095289e-76,
				1.398043286095289e-76,
				1.967871024515285e+274,
				1.398043286095289e-76,
			},
			// Wolfram Alpha finds one root, -1.407589482e350, but that's
			// outside the range of float64.
			roots: []float64{},
		},

		{
			label: "polyroot-7",
			coefficients: []float64{
				// Exact coefficients
				-512, 2304, -4608, 5376, -4032, 2016, -672, 144, -18, 1,
			},
			roots: []float64{2},
		},

		{
			label: "polyroot-8",
			coefficients: []float64{
				-89.828194,
				-49.538208,
				-1.7485783,
				0.69357258,
			},
			nominalRoots: []float64{
				// Very approximate roots
				-5.87997, -2.09795, 10.499,
			},
			roots: []float64{
				// According to mpsolve -as -Ga -o 20
				-5.87997372938113902003781601538,
				-2.0979523063745154302925134943,
				10.49904401305516072550631758,
			},
			maxRelativeError: 5e-16,
		},

		{
			// At the moment we cannot find the first root. When we solve the
			// derivative (the quadratic), one of the roots is -Inf. That causes
			// us to skip the range that the cubic's first root lies in.
			label: "polyroot-12",
			coefficients: []float64{
				// Randomly generated
				3.581721448636721e+271,
				-9.338144821310265e+260,
				8.521976948294891e+145,
				8.323145503904757e-171,
			},
			roots: []float64{
				// According to mpsolve -as -Ga -o 20
				3.8355813892101944e+10,
				1.0957721286935277e+115,
			},
			skip: true,
		},

		{
			label:        "kurbo-1",
			coefficients: []float64{-5.0, 0.0, 0.0, 1.0},
			roots:        []float64{math.Cbrt(5.0)},
		},

		{
			label:        "kurbo-2",
			coefficients: []float64{-5.0, -1.0, 0.0, 1.0},
			roots:        []float64{1.9041608591349206},
		},

		{
			label:        "kurbo-3",
			coefficients: []float64{0.0, -1.0, 0.0, 1.0},
			roots:        []float64{-1.0, 0.0, 1.0},
		},

		{
			label:        "kurbo-4",
			coefficients: []float64{-2.0, -3.0, 0.0, 1.0},
			roots:        []float64{-1.0, 2.0},
		},

		{
			label:        "kurbo-5",
			coefficients: []float64{2.0, -3.0, 0.0, 1.0},
			roots:        []float64{-2.0, 1.0},
		},

		{
			label:            "kurbo-6",
			coefficients:     []float64{2.0 - 1e-12, 5.0, 4.0, 1.0},
			roots:            []float64{-1.9999999999989995, -1.0000010000848456, -0.9999989999161546},
			maxRelativeError: 4e-11,
		},

		{
			label:        "kurbo-7",
			coefficients: []float64{2.0 + 1e-12, 5.0, 4.0, 1.0},
			// The root at -1 is a double root. The Kurbo test specifies -2 as a
			// root, but -2.000000000001 is actually slightly more accurate (but
			// still approximate)
			roots: []float64{-1.0, -1.0, -2.000000000001},
		},

		{
			// https://github.com/linebender/kurbo/issues/446
			label:        "kurbo-issue-446",
			coefficients: []float64{-80, 100, -6e-13, 3e-27},
			roots: []float64{
				// According to mpsolve -as -Ga -o 20
				0.800000000000003840000000000022,
			},
		},

		// Test cases labeled tricky-cubic are cubic polynomials that the
		// poly-cool crate considers tricky. These mostly seem to affect Blinn,
		// though.
		{
			label: "tricky-cubic-1",
			coefficients: []float64{
				1.6149620090145706e-94,
				1.6149620090145634e-94,
				1.6149620090145663e-94,
				9.66803867245343e272,
			},
			// According to mpsolve -as -Ga -o 20
			roots: []float64{-0.550733266557532393246936613697e-122},
		},

		{
			label: "tricky-cubic-2",
			coefficients: []float64{
				-6.323283382275869e98,
				3.0957754283429482e-307,
				3.095775428342964e-307,
				3.095775428342951e-307,
			},
			// According to mpsolve -as -Ga -o 20
			roots: []float64{0.126879385184937912722547017134e136},
		},

		{
			label: "tricky-cubic-3",
			coefficients: []float64{
				-8.522348907129e-161,
				4.471145208374078e-67,
				-0.052026185927646074,
				-2.9441090045938734e-57,
			},
			// According to mpsolve -as -Ga -o 20
			roots: []float64{
				-0.1767128385751561275181e56,
				0.190607741640046917756640778981e-93,
				0.85940284275157032823119e-65,
			},
			maxRelativeError: 1.5e-16,
		},

		{
			label: "tricky-cubic-4",
			coefficients: []float64{
				-2.5162489269306657e-175,
				-2.516248926930655e-175,
				-2.5162489269306522e-175,
				-0.39205037382350466,
			},
			// According to mpsolve -as -Ga -o 20
			roots: []float64{-0.862588989160418052391143261255e-58},
		},

		{
			label: "tricky-cubic-5",
			coefficients: []float64{
				-6.428720163649757e103,
				-6.428720163649766e103,
				-3.3646756114322413e-74,
				-3.3646756114322547e-74,
			},
			// According to mpsolve -as -Ga -o 20
			roots:            []float64{-0.999999999999998600032390445434e0},
			maxRelativeError: 1.5e-16,
		},

		{
			label: "tricky-cubic-6",
			coefficients: []float64{
				-3.565233507454652e74,
				-3.5652335074546437e74,
				-1.2298855640101194e-17,
				9.133009604987547e-243,
			},
			// According to mpsolve -as -Ga -o 20
			roots: []float64{
				-0.289883352710472926894669722511e92,
				-0.10000000000000023280382568618e1,
				0.134663776477195149956099330217e226,
			},
			maxRelativeError: 1.5e-16,
		},

		{
			label: "tricky-cubic-7",
			coefficients: []float64{
				5.5174454041519107,
				-1.6144740273415798e-245,
				-3.892738574215212e-288,
				3.0860491510941517e-292,
			},
			// According to mpsolve -as -Ga -o 20
			roots: []float64{-0.26148396910966762037167940747e98},
		},

		{
			// This used to hit a bug in our Newton-Raphson implementation. It
			// used to abort if the new bound was the same as the old bound,
			// assuming this meant we ran out of precision. However, this can
			// happen naturally just due to the values the polynomial and its
			// derivative evaluate to.
			label: "simple-cubic-1",
			coefficients: []float64{
				-3,
				13,
				-24,
				16,
			},
			roots:  []float64{0.75},
			range_: [2]float64{0, 1},
		},
	}

	count := func(els []float64, el float64) int {
		n := 0
		for i := range els {
			if els[i] == el {
				n++
			}
		}
		return n
	}

	for _, tt := range tests {
		if tt.maxRelativeError >= 1 {
			t.Fatalf("suspiciously large accepted relative error: %g", tt.maxRelativeError)
		}

		want := slices.Clone(tt.roots)
		t.Run(tt.label, func(t *testing.T) {
			if tt.skip {
				t.SkipNow()
			}

			p := NewPolynomial(tt.coefficients...)
			roots := p.Roots(math.Inf(-1), math.Inf(1), 0, nil)
			t.Log(p)
			t.Log("want:          ", want)
			t.Log("got (no range):", roots)

			if len(roots) > len(tt.roots) {
				t.Fatalf("got %d roots but only expected %d", len(roots), len(want))
			}

			for _, r := range roots {
				if count(roots, r) != 1 {
					t.Fatalf("got repeated root %g", r)
				}
			}

			if _, msg, ok := validateRoots(t, roots, want, tt.maxRelativeError); !ok {
				t.Fatal(msg)
			}

			if len(tt.roots) > 0 {
				slices.Sort(tt.roots)
				x0, x1 := tt.range_[0], tt.range_[1]
				if x0 == 0 && x1 == 0 {
					if tt.roots[0] > 0 {
						x0 = -tt.roots[0]
					} else {
						x0 = tt.roots[0] * 2
					}
					if tt.roots[len(tt.roots)-1] > 0 {
						x1 = tt.roots[len(tt.roots)-1] * 2
					} else {
						x1 = -tt.roots[len(tt.roots)-1]
					}
				}
				rootsRange := p.Roots(x0, x1, 0, nil)
				t.Log("got (range):   ", rootsRange)
				if len(rootsRange) > len(tt.roots) {
					t.Fatalf("got %d roots but only expected %d", len(rootsRange), len(want))
				}

				for _, r := range rootsRange {
					if count(rootsRange, r) != 1 {
						t.Fatalf("got repeated root %g", r)
					}
				}

				if _, msg, ok := validateRoots(t, rootsRange, want, tt.maxRelativeError); !ok {
					t.Fatal(msg)
				}
			}
		})
	}
}

func quarticPolynomialWithRoots(x1, x2, x3, x4 *big.Rat) (p *Polynomial, exact bool) {
	// a-d are in the opposite order of Polynomial.Coefficients
	//
	// a = −(x1 + x2 + x3 + x4)
	// b = x1 (x2 + x3) + x2 (x3 + x4) + x4 (x1 + x3)
	// c = −x1x2 (x3 + x4) − x3x4 (x1 + x2)
	// d = x1x2x3x4,

	a := &big.Rat{}
	a.Add(x1, x2).Add(a, x3).Add(a, x4)
	a.Neg(a)

	b1 := &big.Rat{}
	b1.Add(x2, x3).Mul(b1, x1)

	b2 := &big.Rat{}
	b2.Add(x3, x4).Mul(b2, x2)

	b3 := &big.Rat{}
	b3.Add(x1, x3).Mul(b3, x4)

	b := &big.Rat{}
	b.Add(b1, b2).Add(b, b3)

	c1 := &big.Rat{}
	c1.Mul(x1, x2).Neg(c1)

	c2 := &big.Rat{}
	c2.Add(x3, x4)

	c3 := &big.Rat{}
	c3.Mul(x3, x4)

	c4 := &big.Rat{}
	c4.Add(x1, x2)

	c5 := &big.Rat{}
	c5.Mul(c1, c2)

	c6 := &big.Rat{}
	c6.Mul(c3, c4)

	c := &big.Rat{}
	c.Sub(c5, c6)

	d := &big.Rat{}
	d.Mul(x1, x2).Mul(d, x3).Mul(d, x4)

	df, dok := d.Float64()
	cf, cok := c.Float64()
	bf, bok := b.Float64()
	af, aok := a.Float64()
	return NewPolynomial(df, cf, bf, af, 1), dok && cok && bok && aok
}

func relError(got, want float64) float64 {
	if got == want {
		return 0
	}
	if want == 0 {
		want = math.SmallestNonzeroFloat64
	}
	absError := math.Abs(got - want)
	return absError / math.Abs(want)
}

func validateRoots(t testing.TB, got, want []float64, maxRelativeError float64) (errors []float64, msg string, ok bool) {
	t.Helper()

	count := func(els []float64, el float64) int {
		n := 0
		for i := range els {
			if els[i] == el {
				n++
			}
		}
		return n
	}

	var relErrors []float64
	j := 0
	handled := make([]bool, len(want))
	for i := range got {
		for {
			if j == len(want) {
				return nil, fmt.Sprintf("ran out of wanted roots, i=%d", i), false
			}

			if e := relError(got[i], want[j]); e <= maxRelativeError {
				relErrors = append(relErrors, e)
				handled[j] = true
				j++
				break
			}

			// At this point, the root we got doesn't match the root we
			// want. Check if this is due to multiple roots.
			if j > 0 && want[j] == want[j-1] {
				// The wanted root is a multiple root, and we've already
				// matched against the first occurence of it. Skip all
				// remaining copies of the multiple root.
				for j < len(want) && want[j] == want[j-1] {
					j++
				}
				continue
			}

			if j+1 < len(want) && want[j] == want[j+1] && count(want, want[j])%2 == 0 {
				// The wanted root is a multiple root and there is an
				// even number of them, which means they don't cross the
				// X axis, in which case we don't guarantee that we'll
				// find the root even once. Skip all copies of the
				// multiple root.
				j++
				for j < len(want) && want[j] == want[j-1] {
					j++
				}
				continue
			}

			msg := fmt.Sprintf("roots[%d] != want[%d] (%g != %g), relative error = %g",
				i, j, got[i], want[j], relError(got[i], want[j]))
			return nil, msg, false
		}
	}

	for i, b := range handled {
		if !b && i > 0 && want[i-1] == want[i] && handled[i-1] {
			// Multiple root, at least one of which was handled already.
			handled[i] = true
			continue
		}
		if !b && count(want, want[i])%2 == 0 {
			// Multiple root with an even number of occurrences. We
			// don't guarantee that we'll find roots that only touch the
			// X axis.
			continue
		}
		if !b {
			return nil, fmt.Sprintf("wanted root #%d wasn't matched with any found root", i), false
		}
	}

	return relErrors, "", true
}
