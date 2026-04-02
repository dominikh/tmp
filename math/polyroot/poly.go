// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package polyroot

import (
	"fmt"
	"math"
	"slices"
	"strconv"

	"honnef.co/go/stuff/math/polyroot/quadratics"
	"honnef.co/go/stuff/safeish"
)

type Polynomial struct {
	// Coefficients in increasing order of degree.
	coeffs []float64
}

func (poly *Polynomial) quadraticRoots() (r1, r2 float64, n int) {
	a := poly.coeffs[2]
	b := poly.coeffs[1]
	c := poly.coeffs[0]
	return quadratics.Goualard(a, b, c)
}

func (poly *Polynomial) String() string {
	if len(poly.coeffs) == 0 {
		return "0"
	}

	// supers := []rune("⁰¹²³⁴⁵⁶⁷⁸⁹")

	var out string
	for i := len(poly.coeffs) - 1; i >= 0; i-- {
		coeff := poly.coeffs[i]
		if coeff == 0 {
			continue
		}
		if i == 0 {
			if coeff > 0 {
				out += fmt.Sprintf("+ %g", coeff)
			} else if coeff < 0 {
				out += fmt.Sprintf("- %g", math.Abs(coeff))
			}
		} else {
			power := "^"
			for _, c := range strconv.Itoa(i) {
				// power += string(supers[c-'0'])
				power += string(c)
			}
			if power == "^1" {
				power = ""
			}

			switch coeff {
			case 0:
			case -1:
				out += fmt.Sprintf("- x%s ", power)
			case 1:
				if i == len(poly.coeffs)-1 {
					out += fmt.Sprintf("x%s ", power)
				} else {
					out += fmt.Sprintf("+ x%s ", power)
				}
			default:
				if i == len(poly.coeffs)-1 {
					out += fmt.Sprintf("%g*x%s ", coeff, power)
				} else {
					ourSign := "+"
					if coeff < 0 {
						ourSign = "-"
					}
					out += fmt.Sprintf("%s %g*x%s ", ourSign, math.Abs(coeff), power)
				}
			}
		}
	}
	return out
}

// NewPolynomial constructs a new polynomial from coefficients.
//
// The first coefficient provided will be the constant term, the second will be
// the linear term, and so on. That is, passing the coefficients [c, b, a] will
// return the polynomial ax² + bx + c.
//
// Trailing zero coefficients will be ignored and not count towards the
// polynomial's degree.
func NewPolynomial(coeffs ...float64) *Polynomial {
	for len(coeffs) > 0 && coeffs[len(coeffs)-1] == 0 {
		coeffs = coeffs[:len(coeffs)-1]
	}
	return &Polynomial{coeffs}
}

func (poly *Polynomial) Mul(opoly *Polynomial) *Polynomial {
	coeffs := make([]float64, max(len(poly.coeffs)+len(opoly.coeffs)-1, 0))
	for i, c := range poly.coeffs {
		for j, d := range opoly.coeffs {
			coeffs[i+j] += c * d
		}
	}
	return NewPolynomial(coeffs...)
}

func (poly *Polynomial) Scale(scale float64) *Polynomial {
	out := &Polynomial{slices.Clone(poly.coeffs)}
	for i := range out.coeffs {
		out.coeffs[i] *= scale
	}
	return out
}

// Coefficients returns the coefficients of the polynomial, in the same order as
// passed to [NewPolynomial].
func (poly *Polynomial) Coefficients() []float64 {
	return poly.coeffs
}

func (poly *Polynomial) isFin() bool {
	for _, coeff := range poly.coeffs {
		if !isFin(coeff) {
			return false
		}
	}
	return true
}

func (poly *Polynomial) Derivative() *Polynomial {
	// The Go compiler won't trust that the length of poly.coeffs doesn't change
	// between uses...
	pcoeffs := poly.coeffs
	if len(pcoeffs) < 2 {
		return &Polynomial{}
	}
	dcoeffs := make([]float64, len(pcoeffs)-1)
	j := 1.0
	for i := 1; i < len(pcoeffs); i++ {
		// The Go compiler couldn't be convinced that dcoeffs[i-1] is safe.
		*safeish.Index(dcoeffs, i-1) = j * pcoeffs[i]
		// Adding to a float is cheaper than converting a float to an int
		// (higher throughput, more available ports.)
		//
		// OPT(dh): we couldn't get Go to dedicate a register to the '1.0'
		// needed for the increment. That would save us a MOVSD.
		j++
	}

	return &Polynomial{dcoeffs}
}

// Deflate divides this polynomial by the polynomial 'x - root', returning the
// quotient (as a polynomial with one fewer coefficient) and ignoring
// the remainder.
//
// If root is actually a root of poly (as the name suggests
// it should be, but this is not actually required), the
// remainder will be zero. In general, the remainder will be
// poly.Evaluate(root).
func (poly *Polynomial) Deflate(root float64) *Polynomial {
	if len(poly.coeffs) == 0 {
		return &Polynomial{}
	}

	acc := 0.0
	coeffs := make([]float64, len(poly.coeffs)-1)
	// TODO(dh): can we compensate for the error?
	for i := len(poly.coeffs) - 1; i >= 1; i-- {
		c := poly.coeffs[i]
		acc = math.FMA(acc, root, c)
		coeffs[i-1] = acc
	}
	return &Polynomial{coeffs}
}

// Evaluate evaluates the polynomial at point x.
func (poly *Polynomial) Evaluate(x float64) float64 {
	// This function implements the Compensated Horner Scheme. See
	// doi:10.4230/DagSemProc.05391.3 and doi:10.1145/1141277.1141585.

	if len(poly.coeffs) == 0 {
		return 0
	}

	s := poly.coeffs[len(poly.coeffs)-1]
	var e float64
	for i := len(poly.coeffs) - 2; i >= 0; i-- {
		p, e1 := twoProduct(s, x)
		var e2 float64
		s, e2 = twoSum(p, poly.coeffs[i])

		// The errors form the coefficients of two new polynomials that compute
		// the total error. We compute it right away. This would be "HornerSum"
		// or "HornerSumFMA" in the mentioned papers.
		e = math.FMA(e, x, e1+e2)
	}
	if math.IsInf(s, 0) {
		return s
	}

	return s + e
}

func twoProduct(a, b float64) (float64, float64) {
	// "Superfluous" float64 conversion for the same reason as in twoSum.
	x := float64(a * b)
	y := math.FMA(a, b, -x)
	return x, y
}

func twoSum(a, b float64) (float64, float64) {
	// "Superfluous" float64 conversions to prevent automatic use of FMA. Even
	// though there is no multiplication in this function, it could get inlined
	// into other math; if 'a' or 'b' are the result of a multiplication, the
	// compiler could combine that multiplication and the additions done here.
	x := float64(a + b)
	z := float64(x - a)
	y := float64((a - (x - z)) + (b - z))
	return x, y
}

// Degree returns the degree of the polynomial.
//
// A polynomial with no non-zero coefficients has a degree of 0, as does a
// polynomial with one non-zero coefficient.
func (poly *Polynomial) Degree() int {
	return max(0, len(poly.coeffs)-1)
}

// Roots finds all the roots in an interval, appends them to the provided slice,
// and returns the resulting slice.
//
// Either or both ends of the interval can be unbounded by specifying (-)∞.
// However, specifying tighter bounds increases both performance and accuracy
// and should be preferred.
//
// If poly.Evaluate(x0) or poly.Evaluate(x1) (for finite x0 and x1)
// evaluate to infinity or NaN, some roots may be missed.
//
// xError specifies the maximum allowed error in the returned roots. Specifying
// 0 attempts to compute the best roots possible, while non-zero errors allow
// trading accuracy for speed. Note that for large values of xError (say 1e-6)
// and well-conditioned polynomials, results will often be significantly more
// accurate than asked for.
func (poly *Polynomial) Roots(x0, x1, xError float64, roots []float64) (roots_ []float64) {
	origRootsLen := len(roots)
	defer func() {
		if poly.Degree() == 2 {
			// We get ideal results for quadratics.
			return
		}
		newRoots := roots_[origRootsLen:]
		slices.Sort(newRoots)
		newRoots = slices.DeleteFunc(newRoots, func(v float64) bool {
			return !isFin(v)
		})
		newRoots = slices.Compact(newRoots)
		roots_ = roots_[:origRootsLen+len(newRoots)]
	}()

	switch poly.Degree() {
	case 0:
		return roots
	case 1:
		root := -poly.coeffs[0] / poly.coeffs[1]
		if root >= x0 && root <= x1 {
			roots = append(roots, root)
		}
		return roots
	case 2:
		r1, r2, n := poly.quadraticRoots()
		switch n {
		case 1:
			roots = append(roots, r1)
		case 2:
			roots = append(roots, r1, r2)
		}
		roots = slices.DeleteFunc(roots, func(el float64) bool {
			return el < x0 || el > x1
		})
		return roots
	case 3:
		deriv := poly.Derivative()
		if r, ok := poly.cubicFirstRootBetween(deriv, x0, x1, xError); ok {
			if poly.Evaluate(r) <= xError {
				roots = append(roots, r)
				quad := poly.Deflate(r)
				if r1, r2, n := quad.quadraticRoots(); n > 0 {
					if r1 >= x0 && r1 <= x1 {
						roots = append(roots, r1)
					}
					if n >= 2 && r2 >= x0 && r2 <= x1 {
						roots = append(roots, r2)
					}
				}
			} else {
				// We don't trust the root we've found for deflation. Find the
				// remaining roots iteratively. We don't have to find the first
				// root again, however; it was found using the same iterative
				// process and won't improve.
				roots = rootsBetweenWithBuffer(poly, deriv, math.Nextafter(r, x1), x1, xError, roots)
				roots = append(roots, r)
				return roots
			}
		}
		return roots
	default:
		return rootsBetweenWithBuffer(poly, poly.Derivative(), x0, x1, xError, roots)
	}
}

func rootsBetweenWithBuffer(
	poly *Polynomial,
	deriv *Polynomial,
	x0 float64,
	x1 float64,
	xError float64,
	roots []float64,
) []float64 {
	if !deriv.isFin() {
		return roots
	}

	// We use the backing memory of 'roots' for the derivative's roots as well
	// as the roots of this polynomial. For each root of the derivative we
	// produce at most one actual root, so this is fine.
	oldRootsLen := len(roots)
	roots = deriv.Roots(x0, x1, xError, roots)
	if math.IsInf(x0, -1) && len(roots[oldRootsLen:]) == 0 {
		roots = append(roots, 0)
	}
	roots = append(roots, x1)
	// Appending to 'roots', then doing this reslicing ensures that future uses
	// of 'roots' benefit from any slice growth that may have happened due to
	// appending the derivative's roots.
	derivRoots := roots[oldRootsLen:]
	roots = roots[:oldRootsLen]

	// If the polynomial has an odd degree then it must have at least one root,
	// because the left and right arms point in opposite Y directions. But it
	// also might have multiple roots if it's not monotonic, in which case it
	// will have critical points.
	//
	// If the polynomial has an even degree then it must have critical points,
	// because the left and right arm point in the same Y direction. It might
	// not have any roots, though. (And we might not be able to find the
	// critical points due to numerical limitations.)
	oddDegree := poly.Degree()%2 == 1

	// If there are no critical points then we have exactly two brackets to
	// check: [-∞, 0] and [0, +∞]. If there is one critical point, this
	// changes to [-∞, c], [c, +∞]. For multiple critical points, we check
	// all brackets [-∞, c₀], [c₀, c₁], ..., [cₙ₋₁, cₙ], [cₙ, +∞].
	//
	// Each bracket may contain between zero and one roots. When bracket end
	// points fall on roots we may count such roots twice. Such duplicate
	// roots will get filtered out higher up the call chain.

	xa := x0
	ya := poly.Evaluate(xa)

	for _, xb := range derivRoots {
		if xb >= xa && xb <= x1 {
			yb := poly.Evaluate(xb)
			if yb == 0 {
				roots = append(roots, xb)
			} else if math.IsInf(xa, -1) && (math.Signbit(poly.coeffs[len(poly.coeffs)-1]) != math.Signbit(yb)) != oddDegree {
				roots = append(roots, findRoot(poly, deriv, xa, xb, yb, xError))
			} else if math.IsInf(xb, 1) && math.Signbit(ya) != math.Signbit(poly.coeffs[len(poly.coeffs)-1]) {
				roots = append(roots, findRoot(poly, deriv, xa, xb, ya, xError))
			} else if math.Signbit(ya) != math.Signbit(yb) {
				roots = append(roots, findRoot(poly, deriv, xa, xb, ya, xError))
			}
			xa = xb
			ya = yb
		}
	}

	return roots
}

// isFin reports wheter v is finite, i.e., neither infinite nor NaN.
func isFin(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
