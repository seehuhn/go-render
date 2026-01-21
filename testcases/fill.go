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
)

var fillCases = []TestCase{
	{
		Name:   "triangle_nonzero",
		Path:   triangle(10, 50, 32, 10, 54, 50),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "triangle_evenodd",
		Path:   triangle(10, 50, 32, 10, 54, 50),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: EvenOdd},
	},
	{
		Name:   "star_nonzero",
		Path:   fivePointStar(32, 32, 25),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "star_evenodd",
		Path:   fivePointStar(32, 32, 25),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: EvenOdd},
	},
	{
		Name:   "rectangle",
		Path:   rectangle(10, 10, 44, 44),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},

	// Section 1.1 Fill Rules
	{
		Name:   "concentric_rect_nonzero",
		Path:   concentricRectangles(32, 32, 25, 12),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "concentric_rect_evenodd",
		Path:   concentricRectangles(32, 32, 25, 12),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: EvenOdd},
	},
	{
		Name:   "overlapping_circles_nonzero",
		Path:   overlappingCircles(24, 32, 44, 32, 16),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "overlapping_circles_evenodd",
		Path:   overlappingCircles(24, 32, 44, 32, 16),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: EvenOdd},
	},
	{
		Name:   "figure_eight_nonzero",
		Path:   figureEight(32, 32, 20, 10),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "figure_eight_evenodd",
		Path:   figureEight(32, 32, 20, 10),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: EvenOdd},
	},
	{
		Name:   "high_winding_nonzero",
		Path:   highWindingRect(32, 32, 20, 3),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "high_winding_evenodd",
		Path:   highWindingRect(32, 32, 20, 3),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: EvenOdd},
	},
	{
		Name:   "alternating_winding_nonzero",
		Path:   alternatingWinding(32, 32, 25, 12),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "alternating_winding_evenodd",
		Path:   alternatingWinding(32, 32, 25, 12),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: EvenOdd},
	},

	// Section 1.2 Edge Cases
	{
		Name:   "horizontal_edges",
		Path:   rectangle(10, 20, 54, 44),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "vertical_edges",
		Path:   rectangle(28, 5, 36, 59),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "diagonal_45deg",
		Path:   diamond(32, 32, 20),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "near_horizontal",
		Path:   nearHorizontalQuad(10, 30, 54, 30.4),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "near_vertical",
		Path:   nearVerticalQuad(30, 10, 30.4, 54),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "single_pixel",
		Path:   triangle(30, 32, 32, 30, 34, 32),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "subpixel_shape",
		Path:   triangle(31.2, 31.8, 31.5, 31.2, 31.8, 31.8),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},

	// Section 1.3 Boundary Conditions
	{
		Name:   "touching_edge",
		Path:   rectangle(0, 10, 54, 54),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "partially_clipped",
		Path:   rectangle(-10, 20, 40, 74),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "fully_outside",
		Path:   rectangle(70, 70, 100, 100),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "pixel_aligned",
		Path:   rectangle(10, 10, 50, 50),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "half_pixel_offset",
		Path:   rectangle(10.5, 10.5, 50.5, 50.5),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
}

// triangle builds a triangular path.
func triangle(x1, y1, x2, y2, x3, y3 float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !moveTo(yield, x1, y1) {
			return
		}
		if !lineTo(yield, x2, y2) {
			return
		}
		if !lineTo(yield, x3, y3) {
			return
		}
		closePath(yield)
	}
}

// fivePointStar builds a five-pointed star (self-intersecting).
func fivePointStar(cx, cy, r float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// five points, connecting every second point
		var pts [5]vec.Vec2
		for i := range 5 {
			angle := float64(i)*2*math.Pi/5 - math.Pi/2
			pts[i] = vec.Vec2{
				X: cx + r*math.Cos(angle),
				Y: cy + r*math.Sin(angle),
			}
		}

		// draw star: 0 -> 2 -> 4 -> 1 -> 3 -> 0
		order := [5]int{0, 2, 4, 1, 3}
		if !moveTo(yield, pts[order[0]].X, pts[order[0]].Y) {
			return
		}
		for _, i := range order[1:] {
			if !lineTo(yield, pts[i].X, pts[i].Y) {
				return
			}
		}
		closePath(yield)
	}
}

// rectangle builds a rectangular path.
func rectangle(x1, y1, x2, y2 float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !moveTo(yield, x1, y1) {
			return
		}
		if !lineTo(yield, x2, y1) {
			return
		}
		if !lineTo(yield, x2, y2) {
			return
		}
		if !lineTo(yield, x1, y2) {
			return
		}
		closePath(yield)
	}
}

// concentricRectangles builds two nested rectangles (outer clockwise, inner counter-clockwise).
func concentricRectangles(cx, cy, outerSize, innerSize float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// Outer rectangle (clockwise)
		if !moveTo(yield, cx-outerSize, cy-outerSize) {
			return
		}
		if !lineTo(yield, cx+outerSize, cy-outerSize) {
			return
		}
		if !lineTo(yield, cx+outerSize, cy+outerSize) {
			return
		}
		if !lineTo(yield, cx-outerSize, cy+outerSize) {
			return
		}
		if !closePath(yield) {
			return
		}

		// Inner rectangle (counter-clockwise for hole)
		if !moveTo(yield, cx-innerSize, cy-innerSize) {
			return
		}
		if !lineTo(yield, cx-innerSize, cy+innerSize) {
			return
		}
		if !lineTo(yield, cx+innerSize, cy+innerSize) {
			return
		}
		if !lineTo(yield, cx+innerSize, cy-innerSize) {
			return
		}
		closePath(yield)
	}
}

// overlappingCircles builds two overlapping circles using cubic Bezier approximation.
func overlappingCircles(cx1, cy1 float64, cx2, cy2 float64, r float64) path.Path {
	const kappa = 0.5522847498307936

	return func(yield func(path.Command, []vec.Vec2) bool) {
		// First circle
		k := r * kappa
		if !moveTo(yield, cx1+r, cy1) {
			return
		}
		if !cubeTo(yield, cx1+r, cy1-k, cx1+k, cy1-r, cx1, cy1-r) {
			return
		}
		if !cubeTo(yield, cx1-k, cy1-r, cx1-r, cy1-k, cx1-r, cy1) {
			return
		}
		if !cubeTo(yield, cx1-r, cy1+k, cx1-k, cy1+r, cx1, cy1+r) {
			return
		}
		if !cubeTo(yield, cx1+k, cy1+r, cx1+r, cy1+k, cx1+r, cy1) {
			return
		}
		if !closePath(yield) {
			return
		}

		// Second circle
		if !moveTo(yield, cx2+r, cy2) {
			return
		}
		if !cubeTo(yield, cx2+r, cy2-k, cx2+k, cy2-r, cx2, cy2-r) {
			return
		}
		if !cubeTo(yield, cx2-k, cy2-r, cx2-r, cy2-k, cx2-r, cy2) {
			return
		}
		if !cubeTo(yield, cx2-r, cy2+k, cx2-k, cy2+r, cx2, cy2+r) {
			return
		}
		if !cubeTo(yield, cx2+k, cy2+r, cx2+r, cy2+k, cx2+r, cy2) {
			return
		}
		closePath(yield)
	}
}

// figureEight builds a self-crossing figure-eight shape.
func figureEight(cx, cy, width, height float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// Start at top-left, cross through center
		if !moveTo(yield, cx-width, cy-height) {
			return
		}
		// Go to bottom-right
		if !lineTo(yield, cx+width, cy+height) {
			return
		}
		// Go to top-right
		if !lineTo(yield, cx+width, cy-height) {
			return
		}
		// Go to bottom-left (crossing the first line)
		if !lineTo(yield, cx-width, cy+height) {
			return
		}
		closePath(yield)
	}
}

// highWindingRect builds a rectangle wound multiple times in the same direction.
func highWindingRect(cx, cy, size float64, windings int) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		for i := 0; i < windings; i++ {
			if !moveTo(yield, cx-size, cy-size) {
				return
			}
			if !lineTo(yield, cx+size, cy-size) {
				return
			}
			if !lineTo(yield, cx+size, cy+size) {
				return
			}
			if !lineTo(yield, cx-size, cy+size) {
				return
			}
			if !closePath(yield) {
				return
			}
		}
	}
}

// alternatingWinding builds a clockwise outer rectangle with a clockwise inner rectangle.
func alternatingWinding(cx, cy, outerSize, innerSize float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// Outer rectangle (counter-clockwise)
		if !moveTo(yield, cx-outerSize, cy-outerSize) {
			return
		}
		if !lineTo(yield, cx-outerSize, cy+outerSize) {
			return
		}
		if !lineTo(yield, cx+outerSize, cy+outerSize) {
			return
		}
		if !lineTo(yield, cx+outerSize, cy-outerSize) {
			return
		}
		if !closePath(yield) {
			return
		}

		// Inner rectangle (clockwise - same direction conceptually)
		if !moveTo(yield, cx-innerSize, cy-innerSize) {
			return
		}
		if !lineTo(yield, cx+innerSize, cy-innerSize) {
			return
		}
		if !lineTo(yield, cx+innerSize, cy+innerSize) {
			return
		}
		if !lineTo(yield, cx-innerSize, cy+innerSize) {
			return
		}
		closePath(yield)
	}
}

// diamond builds a diamond shape (45 degree rotated square).
func diamond(cx, cy, size float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !moveTo(yield, cx, cy-size) {
			return
		}
		if !lineTo(yield, cx+size, cy) {
			return
		}
		if !lineTo(yield, cx, cy+size) {
			return
		}
		if !lineTo(yield, cx-size, cy) {
			return
		}
		closePath(yield)
	}
}

// nearHorizontalQuad builds a quadrilateral with near-horizontal top/bottom edges.
func nearHorizontalQuad(x1, y1, x2, y2 float64) path.Path {
	height := 10.0
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !moveTo(yield, x1, y1) {
			return
		}
		if !lineTo(yield, x2, y2) {
			return
		}
		if !lineTo(yield, x2, y2+height) {
			return
		}
		if !lineTo(yield, x1, y1+height) {
			return
		}
		closePath(yield)
	}
}

// nearVerticalQuad builds a quadrilateral with near-vertical left/right edges.
func nearVerticalQuad(x1, y1, x2, y2 float64) path.Path {
	width := 10.0
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !moveTo(yield, x1, y1) {
			return
		}
		if !lineTo(yield, x1+width, y1) {
			return
		}
		if !lineTo(yield, x2+width, y2) {
			return
		}
		if !lineTo(yield, x2, y2) {
			return
		}
		closePath(yield)
	}
}
