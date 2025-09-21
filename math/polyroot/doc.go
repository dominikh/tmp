// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

// Package polyroot implements polynomial root finding.
//
// It is based on Cem Yuksel's [high-performance polynomial solver]. As such, it
// strikes a balance between performance and accuracy suitable for interactive
// graphics and delivers satisfactory results up to degree 20, or thereabouts.
//
// Unlike Yuksel's work, which uses a fairly trivial algorithm for solving
// quadratic polynomials, we use an implementation based on [Goualard]'s paper,
// which offers significantly better numerical robustness. As the solver for
// higher degrees works recursively, this improves the results for all degrees.
//
// # Limitations
//
// Roots with [multiplicity] greater than one may be missed, reported just once,
// or reported multiple times at slightly different coordinates, depending on
// rounding errors.
//
// [multiplicity]: https://en.wikipedia.org/wiki/Multiplicity_(mathematics)#Multiplicity_of_a_root_of_a_polynomial
// [high-performance polynomial solver]: https://www.cemyuksel.com/research/polynomials/
// [Goualard]: https://doi.org/10.22541/au.168635343.38524892/v1
package polyroot
