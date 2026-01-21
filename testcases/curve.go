// seehuhn.de/go/render - a 2D rendering library
// Copyright (C) 2026  Jochen Voss <voss@seehuhn.de>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package testcases

import (
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/pdf/graphics"
)

// kappa for cubic Bezier approximation of a quarter circle
const kappa = 0.5522847498307936

var curveCases = []TestCase{
	// Existing tests
	{
		Name:   "quadratic",
		Path:   quadraticCurve(10, 50, 32, 10, 54, 50),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "cubic",
		Path:   cubicCurve(10, 50, 20, 10, 44, 10, 54, 50),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "circle",
		Path:   circle(32, 32, 25),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},

	// Section 4.1: Quadratic Bezier
	{
		Name:   "quadratic_shallow",
		Path:   quadraticCurve(10, 32, 32, 28, 54, 32), // control point near chord
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "quadratic_deep",
		Path:   quadraticCurve(10, 50, 32, 5, 54, 50), // control point far from chord
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "quadratic_below",
		Path:   quadraticCurve(10, 20, 32, 55, 54, 20), // control point below chord (curves down)
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "quadratic_s_shape",
		Path:   sCurveQuadratic(10, 32, 54, 32),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "quadratic_stroked",
		Path:   quadraticCurveOpen(10, 50, 32, 10, 54, 50),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},

	// Section 4.2: Cubic Bezier
	{
		Name:   "cubic_shallow",
		Path:   cubicCurve(10, 32, 22, 28, 42, 28, 54, 32), // control points near chord
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "cubic_deep",
		Path:   cubicCurve(10, 50, 15, 5, 49, 5, 54, 50), // control points far from chord
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "cubic_scurve",
		Path:   cubicCurve(10, 50, 10, 10, 54, 54, 54, 14), // S-curve with inflection
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "cubic_loop",
		Path:   cubicCurve(10, 32, 60, 5, 4, 59, 54, 32), // self-intersecting loop
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "cubic_cusp",
		Path:   cubicCurve(10, 50, 54, 10, 10, 10, 54, 50), // cusp (control points crossed)
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "cubic_nearly_straight",
		Path:   cubicCurve(10, 32, 24, 31, 40, 31, 54, 32), // almost a straight line
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "cubic_stroked",
		Path:   cubicCurveOpen(10, 50, 20, 10, 44, 10, 54, 50),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "cubic_scurve_stroked",
		Path:   cubicCurveOpen(10, 50, 10, 10, 54, 54, 54, 14),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},

	// Section 4.3: Circle/Ellipse
	{
		Name:   "circle_stroked",
		Path:   circle(32, 32, 25),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      3,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "circle_small",
		Path:   circle(32, 32, 5), // small radius
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "circle_large",
		Path:   circle(64, 64, 100), // large radius on 128x128 canvas
		Width:  128,
		Height: 128,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "ellipse",
		Path:   ellipse(32, 32, 28, 14), // stretched circle
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "arc",
		Path:   arc(32, 32, 25, 0, 0.75), // partial circle (3/4)
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},

	// Section 4.4: Curve Flattening Edge Cases
	{
		Name:   "curve_many_segments",
		Path:   cubicCurve(5, 60, 5, 5, 123, 5, 123, 60), // very detailed curve on large canvas
		Width:  128,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "curve_minimal_segments",
		Path:   cubicCurve(10, 32, 24, 31.5, 40, 31.5, 54, 32), // nearly flat
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "cubic_degenerate",
		Path:   cubicCurve(32, 32, 32, 32, 32, 32, 32, 32), // all control points coincident
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "quadratic_degenerate",
		Path:   quadraticCurve(10, 32, 10, 32, 54, 32), // control point on start endpoint
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
}

// quadraticCurve builds a closed shape with a quadratic Bezier curve.
func quadraticCurve(x1, y1, cx, cy, x2, y2 float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !moveTo(yield, x1, y1) {
			return
		}
		if !quadTo(yield, cx, cy, x2, y2) {
			return
		}
		closePath(yield)
	}
}

// quadraticCurveOpen builds an open path with a quadratic Bezier curve (for stroking).
func quadraticCurveOpen(x1, y1, cx, cy, x2, y2 float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !moveTo(yield, x1, y1) {
			return
		}
		quadTo(yield, cx, cy, x2, y2)
	}
}

// cubicCurve builds a closed shape with a cubic Bezier curve.
func cubicCurve(x1, y1, c1x, c1y, c2x, c2y, x2, y2 float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !moveTo(yield, x1, y1) {
			return
		}
		if !cubeTo(yield, c1x, c1y, c2x, c2y, x2, y2) {
			return
		}
		closePath(yield)
	}
}

// cubicCurveOpen builds an open path with a cubic Bezier curve (for stroking).
func cubicCurveOpen(x1, y1, c1x, c1y, c2x, c2y, x2, y2 float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !moveTo(yield, x1, y1) {
			return
		}
		cubeTo(yield, c1x, c1y, c2x, c2y, x2, y2)
	}
}

// sCurveQuadratic builds a closed S-shaped path from two quadratic Bezier curves.
func sCurveQuadratic(x1, y1, x2, y2 float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		midX := (x1 + x2) / 2
		midY := (y1 + y2) / 2

		if !moveTo(yield, x1, y1) {
			return
		}
		// First quadratic curves up
		if !quadTo(yield, (x1+midX)/2, y1-20, midX, midY) {
			return
		}
		// Second quadratic curves down
		if !quadTo(yield, (midX+x2)/2, y2+20, x2, y2) {
			return
		}
		closePath(yield)
	}
}

// circle builds an approximate circle using four cubic Bezier curves.
func circle(cx, cy, r float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		k := r * kappa

		// start at right
		if !moveTo(yield, cx+r, cy) {
			return
		}
		// top-right quadrant
		if !cubeTo(yield, cx+r, cy-k, cx+k, cy-r, cx, cy-r) {
			return
		}
		// top-left quadrant
		if !cubeTo(yield, cx-k, cy-r, cx-r, cy-k, cx-r, cy) {
			return
		}
		// bottom-left quadrant
		if !cubeTo(yield, cx-r, cy+k, cx-k, cy+r, cx, cy+r) {
			return
		}
		// bottom-right quadrant
		if !cubeTo(yield, cx+k, cy+r, cx+r, cy+k, cx+r, cy) {
			return
		}
		closePath(yield)
	}
}

// ellipse builds an approximate ellipse using four cubic Bezier curves.
func ellipse(cx, cy, rx, ry float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		kx := rx * kappa
		ky := ry * kappa

		// start at right
		if !moveTo(yield, cx+rx, cy) {
			return
		}
		// top-right quadrant
		if !cubeTo(yield, cx+rx, cy-ky, cx+kx, cy-ry, cx, cy-ry) {
			return
		}
		// top-left quadrant
		if !cubeTo(yield, cx-kx, cy-ry, cx-rx, cy-ky, cx-rx, cy) {
			return
		}
		// bottom-left quadrant
		if !cubeTo(yield, cx-rx, cy+ky, cx-kx, cy+ry, cx, cy+ry) {
			return
		}
		// bottom-right quadrant
		if !cubeTo(yield, cx+kx, cy+ry, cx+rx, cy+ky, cx+rx, cy) {
			return
		}
		closePath(yield)
	}
}

// arc builds a partial circle (pie slice) from startFraction to endFraction (0-1).
// The arc goes from startFraction to endFraction of a full circle, starting from the right.
func arc(cx, cy, r float64, startFraction, endFraction float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		k := r * kappa

		// Calculate how many quadrants we need
		totalFraction := endFraction - startFraction
		if totalFraction <= 0 {
			return
		}

		// For a 3/4 arc (0 to 0.75), we draw 3 quadrants
		numQuadrants := int(totalFraction * 4)
		if numQuadrants < 1 {
			numQuadrants = 1
		}
		if numQuadrants > 4 {
			numQuadrants = 4
		}

		// Start at center
		if !moveTo(yield, cx, cy) {
			return
		}

		// Line to start of arc (right side)
		if !lineTo(yield, cx+r, cy) {
			return
		}

		// Draw the quadrants
		// top-right quadrant (first)
		if numQuadrants >= 1 {
			if !cubeTo(yield, cx+r, cy-k, cx+k, cy-r, cx, cy-r) {
				return
			}
		}
		// top-left quadrant (second)
		if numQuadrants >= 2 {
			if !cubeTo(yield, cx-k, cy-r, cx-r, cy-k, cx-r, cy) {
				return
			}
		}
		// bottom-left quadrant (third)
		if numQuadrants >= 3 {
			if !cubeTo(yield, cx-r, cy+k, cx-k, cy+r, cx, cy+r) {
				return
			}
		}
		// bottom-right quadrant (fourth)
		if numQuadrants >= 4 {
			if !cubeTo(yield, cx+k, cy+r, cx+r, cy+k, cx+r, cy) {
				return
			}
		}

		closePath(yield)
	}
}
