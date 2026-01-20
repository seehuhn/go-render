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
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: 10, Y: 50}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: 20, Y: 30}}) {
			return
		}
		// Quadratic curve
		if !yield(path.CmdQuadTo, []vec.Vec2{{X: 32, Y: 10}, {X: 44, Y: 30}}) {
			return
		}
		// Line segment
		if !yield(path.CmdLineTo, []vec.Vec2{{X: 54, Y: 50}}) {
			return
		}
		// Cubic curve back to start area
		if !yield(path.CmdCubeTo, []vec.Vec2{{X: 48, Y: 60}, {X: 16, Y: 60}, {X: 10, Y: 50}}) {
			return
		}
		yield(path.CmdClose, nil)
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
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: cx + r, Y: cy}}) {
			return
		}
		// Top-right quadrant
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx + r, Y: cy - k},
			{X: cx + k, Y: cy - r},
			{X: cx, Y: cy - r},
		}) {
			return
		}
		// Top-left quadrant
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx - k, Y: cy - r},
			{X: cx - r, Y: cy - k},
			{X: cx - r, Y: cy},
		}) {
			return
		}
		// Bottom-left quadrant
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx - r, Y: cy + k},
			{X: cx - k, Y: cy + r},
			{X: cx, Y: cy + r},
		}) {
			return
		}
		// Bottom-right quadrant
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx + k, Y: cy + r},
			{X: cx + r, Y: cy + k},
			{X: cx + r, Y: cy},
		}) {
			return
		}
		// Stem going up
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + r, Y: 10}}) {
			return
		}
		// Across top
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + r - 6, Y: 10}}) {
			return
		}
		// Down to bowl
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + r - 6, Y: cy}}) {
			return
		}

		// Inner counter (hole in the bowl)
		ir := 8.0
		ik := ir * kappa

		// Line to start of inner circle (moving inward)
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + ir, Y: cy}}) {
			return
		}
		// Draw inner circle counter-clockwise (reverse winding for hole)
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx + ir, Y: cy + ik},
			{X: cx + ik, Y: cy + ir},
			{X: cx, Y: cy + ir},
		}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx - ik, Y: cy + ir},
			{X: cx - ir, Y: cy + ik},
			{X: cx - ir, Y: cy},
		}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx - ir, Y: cy - ik},
			{X: cx - ik, Y: cy - ir},
			{X: cx, Y: cy - ir},
		}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx + ik, Y: cy - ir},
			{X: cx + ir, Y: cy - ik},
			{X: cx + ir, Y: cy},
		}) {
			return
		}

		yield(path.CmdClose, nil)
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
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: startX, Y: startY}}) {
			return
		}

		// Draw spiral with line segments
		for i := 1; i <= steps; i++ {
			t := float64(i) / float64(steps)
			angle := t * totalAngle
			r := rMin + rGrowth*angle

			x := cx + r*math.Cos(angle)
			y := cy + r*math.Sin(angle)

			if !yield(path.CmdLineTo, []vec.Vec2{{X: x, Y: y}}) {
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
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: cx, Y: cy}}) {
			return
		}

		// Upper loop (clockwise)
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx + k, Y: cy - r/4},
			{X: cx + r, Y: topCy - k/2},
			{X: cx + r, Y: topCy},
		}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx + r, Y: topCy - k},
			{X: cx + k, Y: topCy - r},
			{X: cx, Y: topCy - r},
		}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx - k, Y: topCy - r},
			{X: cx - r, Y: topCy - k},
			{X: cx - r, Y: topCy},
		}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx - r, Y: topCy + k/2},
			{X: cx - k, Y: cy - r/4},
			{X: cx, Y: cy},
		}) {
			return
		}

		// Bottom circle center
		botCy := cy + r/2

		// Lower loop (counter-clockwise to cross)
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx - k, Y: cy + r/4},
			{X: cx - r, Y: botCy - k/2},
			{X: cx - r, Y: botCy},
		}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx - r, Y: botCy + k},
			{X: cx - k, Y: botCy + r},
			{X: cx, Y: botCy + r},
		}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx + k, Y: botCy + r},
			{X: cx + r, Y: botCy + k},
			{X: cx + r, Y: botCy},
		}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx + r, Y: botCy - k/2},
			{X: cx + k, Y: cy + r/4},
			{X: cx, Y: cy},
		}) {
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
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: cx - r, Y: cy - size}}) {
			return
		}
		// Down to curve start
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx - r, Y: cy}}) {
			return
		}
		// Tight U-turn using cubic curves
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx - r, Y: cy + k},
			{X: cx - k, Y: cy + r},
			{X: cx, Y: cy + r},
		}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx + k, Y: cy + r},
			{X: cx + r, Y: cy + k},
			{X: cx + r, Y: cy},
		}) {
			return
		}
		// Up to end
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + r, Y: cy - size}}) {
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

		if !yield(path.CmdMoveTo, []vec.Vec2{{X: x1, Y: cy}}) {
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
			if !yield(path.CmdLineTo, []vec.Vec2{{X: x, Y: y}}) {
				return
			}
		}
	}
}
