// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package main

import (
	"flag"
	"fmt"
	"maps"
	"math"
	"math/big"
	"os"
	"slices"
	"strings"

	"honnef.co/go/stuff/math/mathutil"
	"honnef.co/go/stuff/math/polyroot/quadratics"
)

func main() {
	flag.Usage = func() {
		o := flag.CommandLine.Output()
		fmt.Fprintln(o, "Usage: profile-quadratic-solver [flags] <solver>")
		fmt.Fprintln(o)
		fmt.Fprintln(o, "Flags:")
		flag.PrintDefaults()

		fmt.Fprintln(o)
		fmt.Fprintln(o, "Solvers: goualard, lomont, panchekha, yuksel, yuksel2, yuksel3")
	}
	maxULP := flag.Float64("max-ulp", 3, "Print details for every root that is wrong by more than `ulp` ULP")
	printMisidentified := flag.Bool("print-misidentified", true, "Print details for every misidentified root")
	printHist := flag.Bool("print-error-histogram", true, "Print histogram of errors of roots, measured in ULP")
	printStats := flag.Bool("print-stats", true, "Print overall statistics")
	n := flag.Int("n", 10_000_000, "How many quadratics to generate")

	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(flag.CommandLine.Output(), "must specify exactly one solver")
		flag.Usage()
		os.Exit(2)
	}

	var fn func(a, b, c float64) (r1, r2 float64, n int)
	switch name := strings.ToLower(flag.Arg(0)); name {
	case "yuksel":
		fn = quadratics.Yuksel
	case "yuksel2":
		fn = quadratics.Yuksel2
	case "yuksel3":
		fn = quadratics.Yuksel3
	case "panchekha":
		fn = quadratics.Panchekha
	case "lomont":
		fn = quadratics.Lomont
	case "goualard":
		fn = quadratics.Goualard
	default:
		fmt.Fprintf(flag.CommandLine.Output(), "unknown solver: %s\n", name)
		flag.Usage()
		os.Exit(2)
	}

	// The number of roots that were NaN when they shouldn't have been, or vice
	// versa.
	misidentified := 0
	ulpHist := map[float64]int{}

	for range *n {
		a, b, c := quadratics.Random()

		wantR1, wantR2, wantN := quadratics.BigFloat(
			big.NewFloat(a).SetPrec(128),
			big.NewFloat(b).SetPrec(128),
			big.NewFloat(c).SetPrec(128),
		)

		gotR1, gotR2, gotN := fn(a, b, c)

		wrong := false
		if (wantR1 == nil) != math.IsNaN(gotR1) || (wantR2 == nil) != math.IsNaN(gotR2) || gotN != wantN {
			wrong = true
			misidentified++
		}

		ulp1 := mathutil.ReferenceULPDiff(wantR1, gotR1)
		ulp2 := mathutil.ReferenceULPDiff(wantR2, gotR2)

		if !wrong {
			ulpHist[math.Round(ulp1/0.25)*0.25]++
			ulpHist[math.Round(ulp2/0.25)*0.25]++
		}

		if (*printMisidentified && wrong) || (!wrong && (ulp1 > *maxULP || ulp2 > *maxULP)) {
			fmt.Printf("wolfram:   Roots[FromDigits[{%g, %g, %g}, x] == 0, x]\n", a, b, c)
			fmt.Printf("coeffs:    %g, %g, %g\n", a, b, c)
			fmt.Printf("want:      %g, %g, %d\n", wantR1, wantR2, wantN)
			switch wantN {
			case 0:
			case 1:
				wantR1f, _ := wantR1.Float64()
				fmt.Printf("want(f64): %g, NaN, %d\n", wantR1f, wantN)
			case 2:
				wantR1f, _ := wantR1.Float64()
				wantR2f, _ := wantR2.Float64()
				fmt.Printf("want(f64): %g, %g, %d\n", wantR1f, wantR2f, wantN)
			}
			fmt.Printf("got:       %g, %g, %d\n", gotR1, gotR2, gotN)
			fmt.Printf("ulp:       %g, %g\n", ulp1, ulp2)
			fmt.Println()
		}
	}

	if *printHist {
		keys := slices.Collect(maps.Keys(ulpHist))
		slices.Sort(keys)
		for _, k := range keys {
			v := ulpHist[k]
			fmt.Printf("%g %d\n", k, v)
		}
	}
	if *printStats {
		fmt.Println(len(ulpHist), "histogram buckets")
		fmt.Println(misidentified, "misidentified roots")
	}
}
