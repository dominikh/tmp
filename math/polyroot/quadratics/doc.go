// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

// Package quadratics implements various quadratic equation solvers.
//
// All solvers in this package
//   - work with float64 precision for their inputs and outputs.
//   - only solve for real roots, not complex ones.
//
// Most solvers in this package have known limitations. They were implemented on
// the path to finding the best implementation, which is currently [Goualard].
//
// All solvers take as arguments the three coefficients 𝑎, 𝑏, and 𝑐 of the
// polynomial 𝑎𝑥² + 𝑏𝑥 + 𝑐 and return up to two roots and the number of roots
// returned. When fewer than two roots are found, the remaining roots will be
// set to NaN. The solvers do not differentiate between the following reasons
// for finding no roots and always return (NaN, NaN, 0):
//   - at least one of the coefficients was NaN or ±∞.
//   - the polynomial has no real roots.
//   - the polynomial has no roots.
//
// # Tests
//
// By default, the tests are expected to pass. They skip any solver × test pairs
// that are known to fail. By passing the -dont-skip flag, all tests will run
// unconditionally, revealing all of the known ways in which most solvers are
// broken.
//
// # Bibliogrpahy
//   - C. Lomont, “A better quadratic formula algorithm,” Lomont.org. Accessed: Sep. 23, 2025. [Online]. Available: https://lomont.org/posts/2022/a-better-quadratic-formula-algorithm/
//   - P. Panchekha, “An accurate quadratic formula.” Accessed: Oct. 05, 2025. [Online]. Available: https://pavpanchekha.com/blog/accurate-quadratic.html
//   - “Double-precision floating-point format,” Wikipedia. Sep. 23, 2025. Accessed: Oct. 05, 2025. [Online]. Available: https://en.wikipedia.org/w/index.php?title=Double-precision_floating-point_format&oldid=1312956125
//   - P. H. Sterbenz, Floating-point computation. in Prentice-Hall series in automatic computation. Englewood Cliffs ; Prentice-Hall, 1974.
//   - C.-P. Jeannerod, N. Louvet, and J.-M. Muller, “Further analysis of Kahan’s algorithm for the accurate computation of 2 x 2 determinants,” Mathematics of Computation, vol. 82, no. 284, p. 2245, 2013, doi: 10.1090/S0025-5718-2013-02679-8.
//   - C. Yuksel, “High-performance polynomial root finding for graphics,” Proc. ACM Comput. Graph. Interact. Tech., vol. 5, no. 3, pp. 1–15, Jul. 2022, doi: 10.1145/3543865.
//   - G. E. Forsythe, “How do you solve a quadratic equation?,” Stanford University, Stanford, CA, USA, Technical Report, May 1966.
//   - “Machine epsilon,” Wikipedia. Jul. 23, 2025. Accessed: Oct. 05, 2025. [Online]. Available: https://en.wikipedia.org/w/index.php?title=Machine_epsilon&oldid=1302054356
//   - “MPSolve,” Numerical Analysis Group, Pisa. Accessed: Oct. 05, 2025. [Online]. Available: https://numpi.dm.unipi.it/scientific-computing-libraries/mpsolve/
//   - W. Kahan, “On the cost of floating-point computation without extra-precise arithmetic”.
//   - “Quadratic formula,” Wikipedia. Sep. 11, 2025. Accessed: Oct. 05, 2025. [Online]. Available: https://en.wikipedia.org/w/index.php?title=Quadratic_formula&oldid=1310788225
//   - “Subnormal number,” Wikipedia. Jul. 20, 2025. Accessed: Oct. 05, 2025. [Online]. Available: https://en.wikipedia.org/w/index.php?title=Subnormal_number&oldid=1301507617
//   - F. Goualard, “The ins and outs of solving quadratic equations with floating-point arithmetic,” Jun. 09, 2023, Preprints. doi: 10.22541/au.168635343.38524892/v1.
//   - “Unit in the last place,” Wikipedia. Jul. 31, 2025. Accessed: Oct. 05, 2025. [Online]. Available: https://en.wikipedia.org/w/index.php?title=Unit_in_the_last_place&oldid=1303511421
package quadratics
