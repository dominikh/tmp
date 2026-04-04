// SPDX-FileCopyrightText: 2018 Raph Levien
// SPDX-FileCopyrightText: 2024 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT
// SPDX-FileAttributionText: https://github.com/linebender/kurbo

package curve

import (
	"math"
	"slices"

	"honnef.co/go/stuff/math/polyroot"
)

func makeWeights(rev bool) [nLSE]float64 {
	var result [nLSE]float64
	for i := range nLSE {
		t := float64(i+1) / (nLSE + 1)
		mt := 1.0 - t
		result[i] = 3 * mt * t * mt
	}
	if rev {
		slices.Reverse(result[:])
	}
	return result
}

var aWeights, bWeights = makeWeights(false), makeWeights(true)

func signum(f float64) float64 {
	if math.Signbit(f) {
		return -1
	} else {
		return 1
	}
}

// cubicOffset is state used for computing an offset curve of a single cubic.
type cubicOffset struct {
	// The regularized cubic being offset.
	c CubicBez
	// The derivative of c.
	q QuadBez
	// The offset distance.
	d float64
	// c₀ + c₁t + c₂t² is the cross product of second and first derivatives
	// of the underlying cubic, multiplied by the offset. This is used for
	// computing cusps on the offset curve.
	//
	// Note that given a curve c(t), its signed curvature is c″(t) × c′(t) /
	// ‖c′(t)‖³. See also [cubicOffset.cuspSign].
	c0 float64
	c1 float64
	c2 float64
	// The tolerance.
	tolerance float64
}

// We never let cusp values haven an absolute value smaller than
// this. When a cusp is found, determine its sign and use this value.
const cuspEpsilon = 1e-12

// Number of points for least-squares fit and error evaluation.
//
// This value is a tradeoff between accuracy and performance. The main risk in it
// being to small is under-sampling the error and thus letting excessive error
// slip through. That said, the "arc drawing" approach is designed to be robust
// and not generate approximate results with narrow error peaks, even in near-cusp
// "J" shape curves.
const nLSE = 8

// The proportion of transverse error that is blended in the least-squares logic.
const blend = 1e-3

// Maximum recursion depth.
//
// Recursion is bounded to this depth, so the total number of subdivisions will
// not exceed two to this power.
//
// This is primarily a "belt and suspenders" robustness guard. In normal operation,
// the recursion bound should never be reached, as accuracy improves very quickly
// on subdivision. For unreasonably large coordinate values or small tolerances, it
// is possible, and in those cases the result will be out of tolerance.
//
// Perhaps should be configurable.
const maxDepth = 8

// offsetRec is state local to a subdivision.
type offsetRec struct {
	t0 float64
	t1 float64
	// Unit tangent at t0
	utan0 Vec2
	// Unit tangent at t1
	utan1 Vec2
	cusp0 float64
	cusp1 float64
	// recursion depth
	depth int
}

// errEval is the result of error evaluation.
type errEval struct {
	// Maximum detected error.
	errSquared float64
	// Unit normals sampled uniformly across approximation.
	unorms [nLSE]Vec2
	// Difference between point on source curve and normal from approximation.
	errVecs [nLSE]Vec2
}

// subdivisionPoint is the result of subdivision.
type subdivisionPoint struct {
	// Source curve t value at subdivision point
	t float64
	// Unit tangent at subdivision point
	utan Vec2
}

// offsetCubic computes an approximate offset curve.
//
// There is a fair amount of attention to robustness, but this method is not suitable
// for degenerate cubics with entirely co-linear control points. Those cases should be
// handled before calling this function, by replacing them with linear segments.
func offsetCubic(
	c CubicBez,
	d float64,
	tolerance float64,
	pushMoveTo bool,
	out BezPath,
) BezPath {
	// A tuning parameter for regularization. A value too large may distort the curve,
	// while a value too small may fail to generate smooth curves. This is a somewhat
	// arbitrary value, and should be revisited.
	const dimTune = 0.25
	// We use regularization to perturb the curve to avoid *interior* zero-derivative
	// cusps. There is robustness logic in place to handle zero derivatives at the
	// endpoints.
	//
	// As a performance note, it might be a good idea to move regularization and
	// tangent determination to the caller, as those computations are the same for both
	// signs of `d`.
	cRegularized := c.regularizeCusp(tolerance * dimTune)
	co := newCubicOffset(cRegularized, d, tolerance)
	tan0, tan1 := c.Tangents()
	utan0 := tan0.Normalize()
	utan1 := tan1.Normalize()
	cusp0 := co.endpointCusp(co.q.P0, co.c0)
	cusp1 := co.endpointCusp(co.q.P2, co.c0+co.c1+co.c2)

	if pushMoveTo {
		out.MoveTo(c.P0.Translate(utan0.Turn90().Mul(d)))
	}

	rec := offsetRec{
		t0:    0,
		t1:    1,
		utan0: utan0,
		utan1: utan1,
		cusp0: cusp0,
		cusp1: cusp1,
		depth: 0,
	}
	return co.offsetRec(&rec, out)
}

// Create a new curve from Bézier segment and offset.
func newCubicOffset(c CubicBez, d float64, tolerance float64) cubicOffset {
	q := c.Differentiate()
	d2 := 2.0 * d
	p1xp0 := Vec2(q.P1).Cross(Vec2(q.P0))
	p2xp0 := Vec2(q.P2).Cross(Vec2(q.P0))
	p2xp1 := Vec2(q.P2).Cross(Vec2(q.P1))
	return cubicOffset{
		c:         c,
		q:         q,
		d:         d,
		c0:        d2 * p1xp0,
		c1:        d2 * (p2xp0 - 2.0*p1xp0),
		c2:        d2 * (p2xp1 - p2xp0 + p1xp0),
		tolerance: tolerance,
	}
}

// cuspSign computes curvature of the source curve times offset plus 1.
//
// This quantity is called "cusp" because cusps appear in the offset curve
// where this value crosses zero. This is based on the geometric property
// that the offset curve has a cusp when the radius of curvature of the
// source curve is equal to the offset curve's distance.
//
// Note: there is a potential division by zero when the derivative vanishes.
// We avoid doing so for interior points by regularizing the cubic beforehand.
// We avoid doing so for endpoints by calling `endpoint_cusp` instead.
func (co *cubicOffset) cuspSign(t float64) float64 {
	ds2 := Vec2(co.q.Eval(t)).Hypot2()
	return ((co.c2*t+co.c1)*t+co.c0)/(ds2*math.Sqrt(ds2)) + 1.0
}

// endpointCusp computes the cusp value of an endpoint.
//
// This is a special case of [cubicOffset.cuspSign]. For the start point, tan
// should be the start point tangent and y should be c0. For the end point,
// tan should be the end point tangent and y should be c0 + c1 + c2.
//
// This is just evaluating the polynomial at t=0 and t=1.
//
// See [cubicOffset.cuspSign] for a description of what "cusp value" means.
func (co *cubicOffset) endpointCusp(tan Point, y float64) float64 {
	// Robustness to avoid divide-by-zero when derivatives vanish.
	const epsilonTanDist = 1e-12
	tanDist := max(Vec2(tan).Hypot(), epsilonTanDist)
	rsqrt := 1.0 / tanDist
	return y*(rsqrt*rsqrt*rsqrt) + 1.0
}

// offsetRec is the primary entry point for recursive subdivision.
//
// At a high level, this method determines whether subdivision is necessary. If
// so, it determines a subdivision point and then recursively calls itself on
// both subdivisions. If not, it computes a single cubic Bézier to approximate
// the offset curve.
func (co *cubicOffset) offsetRec(rec *offsetRec, out BezPath) BezPath {
	// First, determine whether the offset curve contains a cusp. If the sign
	// of the cusp value (curvature times offset plus 1) is different at the
	// subdivision endpoints, then there is definitely a cusp inside. Find it and
	// subdivide there.
	//
	// Note that there's a possibility the curve has two (or, potentially, any
	// even number). We don't rigorously check for this case; if the measured
	// error comes in under the tolerance, we simply accept it. Otherwise, in
	// the common case we expect to detect a sign crossing from the new
	// subdivision point.
	if rec.cusp0*rec.cusp1 < 0.0 {
		a := rec.t0
		b := rec.t1
		s := signum(rec.cusp1)
		f := func(t float64) float64 { return s * co.cuspSign(t) }
		k1 := 0.2 / (b - a)
		const epsilonItp = 1e-12
		t := SolveITP(f, a, b, epsilonItp, 1, k1, s*rec.cusp0, s*rec.cusp1)
		// TODO(robustness): If we're unlucky, there will be 3 cusps between t0
		// and t1, and the solver will land on the middle one. In that case, the
		// derivative on cusp will be the opposite sign as expected.
		//
		// If we're even more unlucky, there is a second-order cusp, both zero
		// cusp value and zero derivative.
		utanT := Vec2(co.q.Eval(t)).Normalize()
		cuspTMinus := math.Copysign(cuspEpsilon, rec.cusp0)
		cuspTPlus := math.Copysign(cuspEpsilon, rec.cusp1)
		return co.subdivide(rec, out, t, utanT, cuspTMinus, cuspTPlus)
	}
	// We determine the first approximation to the offset curve.
	a, b := co.drawArc(rec)
	dt := (rec.t1 - rec.t0) * (1.0 / (nLSE + 1))
	// These represent t values on the source curve.
	var ts [nLSE]float64
	for i := range ts {
		ts[i] = rec.t0 + float64(i+1)*dt
	}
	cApprox := co.apply(rec, a, b)
	errVal := co.evalErr(rec, cApprox, &ts)
	// Number of least-squares refinement steps. More gives a smaller error, but
	// takes more time.
	const nRefine = 2
	for range nRefine {
		if errVal.errSquared <= co.tolerance*co.tolerance {
			break
		}
		a2, b2 := co.refineLeastSquares(rec, a, b, &errVal)
		cApprox2 := co.apply(rec, a2, b2)
		errVal2 := co.evalErr(rec, cApprox2, &ts)
		if errVal2.errSquared >= errVal.errSquared {
			break
		}
		errVal = errVal2
		a, b = a2, b2
		cApprox = cApprox2
	}
	if rec.depth < maxDepth && errVal.errSquared > (co.tolerance*co.tolerance) {
		subdivPt := co.findSubdivisionPoint(rec)
		t, utan := subdivPt.t, subdivPt.utan
		// TODO(robustness): if cusp is extremely near zero, then assign epsilon
		// with alternate signs based on derivative of cusp.
		cusp := co.cuspSign(t)
		out = co.subdivide(rec, out, t, utan, cusp, cusp)
	} else {
		out.CubicTo(cApprox.P1, cApprox.P2, cApprox.P3)
	}
	return out
}

// subdivide recursively subdivides.
//
// In the case of subdividing at a cusp, the cusp value at the subdivision point
// is mathematically zero, but in those cases we treat it as a signed infinitesimal
// value representing the values at t minus epsilon and t plus epsilon.
//
// Note that unit tangents are passed down explicitly. In the general case, they
// are equal to the derivative (evaluated at that t value) normalized to unit
// length, but in cases where the derivative is near-zero, they are computed more
// robustly.
func (co *cubicOffset) subdivide(
	rec *offsetRec,
	into BezPath,
	t float64,
	utanT Vec2,
	cuspTMinus float64,
	cuspTPlus float64,
) BezPath {
	rec0 := offsetRec{
		t0:    rec.t0,
		t1:    t,
		utan0: rec.utan0,
		utan1: utanT,
		cusp0: rec.cusp0,
		cusp1: cuspTMinus,
		depth: rec.depth + 1,
	}
	into = co.offsetRec(&rec0, into)
	rec1 := offsetRec{
		t0:    t,
		t1:    rec.t1,
		utan0: utanT,
		utan1: rec.utan1,
		cusp0: cuspTPlus,
		cusp1: rec.cusp1,
		depth: rec.depth + 1,
	}
	return co.offsetRec(&rec1, into)
}

// apply converts from (a, b) parameter space to the approximate cubic Bézier.
//
// The offset approximation can be considered B(t) + d * D(t), where D(t)
// is roughly a unit vector in the direction of the unit normal of the source
// curve. (The word "roughly" is appropriate because transverse error may
// cancel out normal error, resulting in a lower error than either alone).
// The endpoints of D(t) must be the unit normals of the source curve, and
// the endpoint tangents of D(t) must tangent to the endpoint tangents of
// the source curve, to ensure G1 continuity.
//
// The (a, b) parameters refer to the magnitude of the vector from the endpoint
// to the corresponding control point in D(t), the direction being determined
// by the unit tangent.
//
// When the candidate solution would lead to negative distance from the
// endpoint to the control point, that distance is clamped to zero. Otherwise
// such solutions should be considered invalid, and have the unpleasant
// property of sometimes passing error tolerance checks.
func (co *cubicOffset) apply(rec *offsetRec, a float64, b float64) CubicBez {
	// wondering if p0 and p3 should be in rec
	// Scale factor from derivatives to displacements
	s := (1.0 / 3.0) * (rec.t1 - rec.t0)
	p0 := co.c.Eval(rec.t0).Translate(rec.utan0.Turn90().Mul(co.d))
	l0 := s*Vec2(co.q.Eval(rec.t0)).Hypot() + a*co.d
	p1 := p0
	if l0*rec.cusp0 > 0.0 {
		p1 = p1.Translate(rec.utan0.Mul(l0))
	}
	p3 := co.c.Eval(rec.t1).Translate(rec.utan1.Turn90().Mul(co.d))
	p2 := p3
	l1 := s*Vec2(co.q.Eval(rec.t1)).Hypot() - b*co.d
	if l1*rec.cusp1 > 0.0 {
		p2 = p2.Translate(rec.utan1.Mul(l1).Negate())
	}
	return CubicBez{p0, p1, p2, p3}
}

// drawArc computes an arc approximation.
//
// This is called "arc drawing" because if we just look at the delta
// vector, it describes an arc from the initial unit normal to the final unit
// normal, with "as smooth as possible" parametrization. This approximation
// is not necessarily great, but is very robust, and in particular, accuracy
// does not degrade for J shaped near-cusp source curves or when the offset
// distance is large (with respect to the source curve arc length).
//
// It is a pretty good approximation overall and has very clean O(n⁴) scaling.
// Its worst performance is on curves with a large cubic component, where it
// undershoots. The theory is that the least squares refinement improves those
// cases.
func (co *cubicOffset) drawArc(rec *offsetRec) (float64, float64) {
	// OPT: this can probably be done with vectors rather than arctangent
	th := math.Atan2(rec.utan1.Cross(rec.utan0), rec.utan1.Dot(rec.utan0))
	a := (2.0 / 3.0) / (1 + math.Cos(0.5*th)) * 2 * math.Sin(0.5*th)
	b := -a
	return a, b
}

// Evaluate error and also refine t values
//
// Returns evaluation of error including error vectors and (squared)
// maximum error.
//
// The vector of t values represents points on the source curve; the logic
// here is a Newton step to bring those points closer to the normal ray of
// the approximation.
func (co *cubicOffset) evalErr(rec *offsetRec, cApprox CubicBez, ts *[nLSE]float64) errEval {
	qa := cApprox.Differentiate()
	errSquared := 0.0
	var unorms [nLSE]Vec2
	var errVecs [nLSE]Vec2
	for i := range ts {
		ta := float64(i+1) * (1.0 / (nLSE + 1))
		t := ts[i]
		p := co.c.Eval(t)
		// Newton step to refine ta value
		pa := cApprox.Eval(ta)
		tana := Vec2(qa.Eval(ta))
		t += tana.Dot(pa.Sub(p)) / tana.Dot(Vec2(co.q.Eval(t)))
		t = min(max(t, rec.t0), rec.t1)
		ts[i] = t
		cusp := signum(rec.cusp0)
		unorm := tana.Normalize().Turn90().Mul(cusp)
		unorms[i] = unorm
		pNew := co.c.Eval(t).Translate(unorm.Mul(co.d))
		errVec := pa.Sub(pNew)
		errVecs[i] = errVec
		distErrSquared := errVec.Hypot2()
		if math.IsInf(distErrSquared, 0) {
			// A hack to make sure we reject bad refinements
			distErrSquared = 1e12
		}
		// Note: consider also incorporating angle_error
		errSquared = max(errSquared, distErrSquared)
	}
	return errEval{
		errSquared: errSquared,
		unorms:     unorms,
		errVecs:    errVecs,
	}
}

// refineLeastSquares refines an approximation, minimizing least squares error.
//
// Compute the approximation that minimizes least squares error, based on the given error
// evaluation.
//
// The effectiveness of this refinement varies. Basically, if the curve has a large cubic
// component, then the arc drawing will undershoot systematically and this refinement will
// reduce error considerably. In other cases, it will eventually converge to a local
// minimum, but convergence is slow.
//
// The blend parameter controls a tradeoff between robustness and speed of convergence.
// In the happy case, convergence is fast and not very sensitive to this parameter. If the
// parameter is too small, then in near-parabola cases the determinant will be small and
// the result not numerically stable.
//
// A value of 1 for blend corresponds to essentially the Hoschek method, minimizing
// Euclidean distance (which tends to over-anchor on the given t values). A value of 0 would
// minimize the dot product of error wrt the normal vector, ignoring the cross product
// component.
//
// A possible future direction would be to tune the parameter adaptively.
func (co *cubicOffset) refineLeastSquares(
	rec *offsetRec,
	a float64,
	b float64,
	err *errEval,
) (float64, float64) {
	aa := 0.0
	ab := 0.0
	ac := 0.0
	bb := 0.0
	bc := 0.0
	for i := range nLSE {
		n := err.unorms[i]
		errVec := err.errVecs[i]
		cn := errVec.Dot(n)
		ct := errVec.Cross(n)
		an := aWeights[i] * rec.utan0.Dot(n)
		at := aWeights[i] * rec.utan0.Cross(n)
		bn := bWeights[i] * rec.utan1.Dot(n)
		bt := bWeights[i] * rec.utan1.Cross(n)
		aa += an*an + blend*(at*at)
		ab += an*bn + blend*at*bt
		ac += an*cn + blend*at*ct
		bb += bn*bn + blend*(bt*bt)
		bc += bn*cn + blend*bt*ct
	}
	idet := 1.0 / (co.d * (aa*bb - ab*ab))
	deltaA := idet * (ac*bb - ab*bc)
	deltaB := idet * (aa*bc - ac*ab)
	return a - deltaA, b - deltaB
}

// findSubdivisionPoint decides where to subdivide when error is exceeded.
//
// For curves not containing an inflection point, subdivide at the tangent
// bisecting the endpoint tangents. The logic is that for a near cusp in the
// source curve, you want the subdivided sections to be approximately
// circular arcs with progressively smaller angles.
//
// When there is an inflection point (or, more specifically, when the curve
// crosses its chord), bisecting the angle can lead to very lopsided arc
// lengths, so just subdivide by t in that case.
func (co *cubicOffset) findSubdivisionPoint(rec *offsetRec) subdivisionPoint {
	t := 0.5 * (rec.t0 + rec.t1)
	qt := Vec2(co.q.Eval(t))
	x0 := math.Abs(rec.utan0.Cross(qt))
	x1 := math.Abs(rec.utan1.Cross(qt))
	const subdivideThreshold = 0.1
	if x0 > subdivideThreshold*x1 && x1 > subdivideThreshold*x0 {
		utan := qt.Normalize()
		return subdivisionPoint{t, utan}
	}

	// Note: do we want to track p0 & p3 in rec, to avoid repeated eval?
	chord := co.c.Eval(rec.t1).Sub(co.c.Eval(rec.t0))
	if chord.Cross(rec.utan0)*chord.Cross(rec.utan1) < 0.0 {
		tan := rec.utan0.Add(rec.utan1)
		if subdivision, ok := co.subdivideForTangent(rec.utan0, rec.t0, rec.t1, tan, false); ok {
			return subdivision
		}
	}
	// Curve definitely has an inflection point
	// Try to subdivide based on integral of absolute curvature.

	// Tangents at recursion endpoints and inflection points.
	tangents := make([]Vec2, 0, 4)
	ts := make([]float64, 0, 4)
	tangents = append(tangents, rec.utan0)
	ts = append(ts, rec.t0)
	inflections, numInflections := co.c.Inflections()
	for _, t := range inflections[:numInflections] {
		if t > rec.t0 && t < rec.t1 {
			tangents = append(tangents, Vec2(co.q.Eval(t)))
			ts = append(ts, t)
		}
	}
	tangents = append(tangents, rec.utan1)
	ts = append(ts, rec.t1)
	arcAngles := make([]float64, 0, 3)
	var sum float64
	for i := range len(tangents) - 1 {
		tan0 := tangents[i]
		tan1 := tangents[i+1]
		th := math.Atan2(tan0.Cross(tan1), tan0.Dot(tan1))
		sum += math.Abs(th)
		arcAngles = append(arcAngles, th)
	}
	target := sum * 0.5
	var i int
	for math.Abs(arcAngles[i]) < target {
		target -= math.Abs(arcAngles[i])
		i++
	}
	rotation := VecFromAngle(math.Copysign(target, arcAngles[i]))
	base := tangents[i]
	tan := rotateScale(base, rotation)
	var utan0 Vec2
	if i == 0 {
		utan0 = rec.utan0
	} else {
		utan0 = base.Normalize()
	}
	ret, ok := co.subdivideForTangent(utan0, ts[i], ts[i+1], tan, true)
	if !ok {
		panic("unreachable")
	}
	return ret
}

// subdivideForTangent finds a subdivision point, given a tangent vector.
//
// When subdividing by bisecting the angle (or, more generally, subdividing by
// the L1 norm of curvature when there are inflection points), we find the
// subdivision point by solving for the tangent matching, specifically the
// cross-product of the tangent and the curve's derivative being zero. For
// internal cusps, subdividing near the cusp is a good thing, but there is
// still a robustness concern for vanishing derivative at the endpoints.
func (co *cubicOffset) subdivideForTangent(utan0 Vec2, t0, t1 float64, tan Vec2, force bool) (subdivisionPoint, bool) {
	t := 0.0
	// set up quadratic equation for matching tangents
	z0 := tan.Cross(Vec2(co.q.P0))
	z1 := tan.Cross(Vec2(co.q.P1))
	z2 := tan.Cross(Vec2(co.q.P2))
	c0 := z0
	c1 := 2.0 * (z1 - z0)
	c2 := (z2 - z1) - (z1 - z0)
	roots := polyroot.NewPolynomial(c0, c1, c2).Roots(t0, t1, 0, nil)
	if len(roots) > 0 {
		t = roots[len(roots)-1]
	}
	nSoln := len(roots)
	if nSoln != 1 {
		if !force {
			return subdivisionPoint{}, false
		}
		// Numerical failure, try to subdivide at cusp; we pick the
		// smaller derivative.
		if Vec2(co.q.Eval(t0)).Hypot2() > Vec2(co.q.Eval(t1)).Hypot2() {
			t = t1
		} else {
			t = t0
		}
	}
	q := Vec2(co.q.Eval(t))
	const epsilonUtan = 1e-12
	var utan Vec2
	if nSoln == 1 && q.Hypot2() >= epsilonUtan {
		utan = q.Normalize()
	} else if tan.Hypot2() >= epsilonUtan {
		// Curve has a zero-derivative cusp but angles well defined
		utan = tan.Normalize()
	} else {
		// 180 degree U-turn, arbitrarily pick a direction.
		// If we get to this point, there will probably be a failure.
		utan = utan0.Turn90()
	}
	return subdivisionPoint{t, utan}, true
}
