<!--
// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT
-->

# Angles

We use the `Angle` type alias to represent angles. It is an alias, not a new
type, so using the math package doesn't become a pain. The alias still allows us
to attach documentation to the type, however.

When working with user-controllable angles, which includes all angles stored in
exported fields or passed as function arguments, make sure to use
`normalizeAngle` or `clampAngle` on them (depending if they represent a position
or an amount of rotation), unless the surrounding math or logic already handles
out-of-range values.

Also normalize and clamp angles in constructor-esque functions, so out of range
values are less likely to make it to the user.
