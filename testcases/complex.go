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
	"math"

	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/pdf/graphics"
)

var complexCases = []TestCase{
	// Section 5.2: Mixed Operations
	{
		Name:   "mixed_lines_curves",
		Path:   mixedLinesCurves(),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "stroked_mixed",
		Path:   mixedLinesCurves(),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      3,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "glyph_like",
		Path:   glyphLikeShape(),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},

	// Section 5.3: Stroke Self-Intersection
	{
		Name:   "spiral_overlap",
		Path:   spiralPath(32, 32, 5, 25, 3),
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
		Name:   "figure_eight",
		Path:   figureEightStroke(32, 32, 20),
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
		Name:   "thick_tight_curve",
		Path:   tightCurve(32, 32, 15),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      10,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "zigzag_thick",
		Path:   zigzagPath(10, 32, 54, 20),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
}

// mixedLinesCurves builds a path combining line segments and Bezier curves.
func mixedLinesCurves() path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// Start with a line segment
		if !moveTo(yield, 10, 50) {
			return
		}
		if !lineTo(yield, 20, 30) {
			return
		}
		// Quadratic curve
		if !quadTo(yield, 32, 10, 44, 30) {
			return
		}
		// Line segment
		if !lineTo(yield, 54, 50) {
			return
		}
		// Cubic curve back to start area
		if !cubeTo(yield, 48, 60, 16, 60, 10, 50) {
			return
		}
		closePath(yield)
	}
}

// glyphLikeShape builds a complex shape similar to a typographic glyph.
// This resembles a simplified lowercase 'a' or 'e' character.
func glyphLikeShape() path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		const kappa = 0.5522847498307936

		// Outer bowl (circular shape)
		cx, cy := 32.0, 38.0
		r := 18.0
		k := r * kappa

		// Start at right of bowl
		if !moveTo(yield, cx+r, cy) {
			return
		}
		// Top-right quadrant
		if !cubeTo(yield, cx+r, cy-k, cx+k, cy-r, cx, cy-r) {
			return
		}
		// Top-left quadrant
		if !cubeTo(yield, cx-k, cy-r, cx-r, cy-k, cx-r, cy) {
			return
		}
		// Bottom-left quadrant
		if !cubeTo(yield, cx-r, cy+k, cx-k, cy+r, cx, cy+r) {
			return
		}
		// Bottom-right quadrant
		if !cubeTo(yield, cx+k, cy+r, cx+r, cy+k, cx+r, cy) {
			return
		}
		// Stem going up
		if !lineTo(yield, cx+r, 10) {
			return
		}
		// Across top
		if !lineTo(yield, cx+r-6, 10) {
			return
		}
		// Down to bowl
		if !lineTo(yield, cx+r-6, cy) {
			return
		}

		// Inner counter (hole in the bowl)
		ir := 8.0
		ik := ir * kappa

		// Line to start of inner circle (moving inward)
		if !lineTo(yield, cx+ir, cy) {
			return
		}
		// Draw inner circle counter-clockwise (reverse winding for hole)
		if !cubeTo(yield, cx+ir, cy+ik, cx+ik, cy+ir, cx, cy+ir) {
			return
		}
		if !cubeTo(yield, cx-ik, cy+ir, cx-ir, cy+ik, cx-ir, cy) {
			return
		}
		if !cubeTo(yield, cx-ir, cy-ik, cx-ik, cy-ir, cx, cy-ir) {
			return
		}
		if !cubeTo(yield, cx+ik, cy-ir, cx+ir, cy-ik, cx+ir, cy) {
			return
		}

		closePath(yield)
	}
}

// spiralPath builds an Archimedean spiral that overlaps itself.
func spiralPath(cx, cy, rMin, rMax float64, turns float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		steps := int(turns * 32) // 32 segments per turn
		if steps < 8 {
			steps = 8
		}

		totalAngle := turns * 2 * math.Pi
		rGrowth := (rMax - rMin) / totalAngle

		// Start point
		startX := cx + rMin
		startY := cy
		if !moveTo(yield, startX, startY) {
			return
		}

		// Draw spiral with line segments
		for i := 1; i <= steps; i++ {
			t := float64(i) / float64(steps)
			angle := t * totalAngle
			r := rMin + rGrowth*angle

			x := cx + r*math.Cos(angle)
			y := cy + r*math.Sin(angle)

			if !lineTo(yield, x, y) {
				return
			}
		}
	}
}

// figureEightStroke builds a figure-eight (lemniscate-like) path for stroke testing.
func figureEightStroke(cx, cy, size float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// Using two circular arcs to form a figure-eight
		const kappa = 0.5522847498307936
		r := size / 2
		k := r * kappa

		// Top circle center
		topCy := cy - r/2

		// Start at center crossing point
		if !moveTo(yield, cx, cy) {
			return
		}

		// Upper loop (clockwise)
		if !cubeTo(yield, cx+k, cy-r/4, cx+r, topCy-k/2, cx+r, topCy) {
			return
		}
		if !cubeTo(yield, cx+r, topCy-k, cx+k, topCy-r, cx, topCy-r) {
			return
		}
		if !cubeTo(yield, cx-k, topCy-r, cx-r, topCy-k, cx-r, topCy) {
			return
		}
		if !cubeTo(yield, cx-r, topCy+k/2, cx-k, cy-r/4, cx, cy) {
			return
		}

		// Bottom circle center
		botCy := cy + r/2

		// Lower loop (counter-clockwise to cross)
		if !cubeTo(yield, cx-k, cy+r/4, cx-r, botCy-k/2, cx-r, botCy) {
			return
		}
		if !cubeTo(yield, cx-r, botCy+k, cx-k, botCy+r, cx, botCy+r) {
			return
		}
		if !cubeTo(yield, cx+k, botCy+r, cx+r, botCy+k, cx+r, botCy) {
			return
		}
		if !cubeTo(yield, cx+r, botCy-k/2, cx+k, cy+r/4, cx, cy) {
			return
		}
	}
}

// tightCurve builds a U-shaped curve where the inner radius is small
// relative to stroke width, causing the inner edge to cross.
func tightCurve(cx, cy, size float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		const kappa = 0.5522847498307936
		r := size
		k := r * kappa

		// Start at left
		if !moveTo(yield, cx-r, cy-size) {
			return
		}
		// Down to curve start
		if !lineTo(yield, cx-r, cy) {
			return
		}
		// Tight U-turn using cubic curves
		if !cubeTo(yield, cx-r, cy+k, cx-k, cy+r, cx, cy+r) {
			return
		}
		if !cubeTo(yield, cx+k, cy+r, cx+r, cy+k, cx+r, cy) {
			return
		}
		// Up to end
		if !lineTo(yield, cx+r, cy-size) {
			return
		}
	}
}

// zigzagPath builds a zigzag pattern where adjacent thick strokes overlap.
func zigzagPath(x1, cy, x2, amplitude float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		segments := 5
		width := x2 - x1
		segWidth := width / float64(segments)

		if !moveTo(yield, x1, cy) {
			return
		}

		for i := 1; i <= segments; i++ {
			x := x1 + float64(i)*segWidth
			var y float64
			if i%2 == 1 {
				y = cy - amplitude
			} else {
				y = cy + amplitude
			}
			if !lineTo(yield, x, y) {
				return
			}
		}
	}
}
