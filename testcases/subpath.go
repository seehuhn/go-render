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
)

var subpathCases = []TestCase{
	// Section 5.1 Multiple Subpaths
	{
		Name:   "two_triangles",
		Path:   twoTriangles(16, 32, 48, 32, 12),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "overlapping_rect_nonzero",
		Path:   overlappingRectangles(10, 10, 40, 40, 24, 24, 54, 54),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "overlapping_rect_evenodd",
		Path:   overlappingRectangles(10, 10, 40, 40, 24, 24, 54, 54),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: EvenOdd},
	},
	{
		Name:   "ring_shape",
		Path:   ringShape(32, 32, 25, 12),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: EvenOdd},
	},
	{
		Name:   "multiple_rings",
		Path:   multipleRings(32, 32),
		Width:  128,
		Height: 128,
		Op:     Fill{Rule: EvenOdd},
	},
	{
		Name:   "many_small_shapes",
		Path:   manySmallShapes(8, 8),
		Width:  128,
		Height: 128,
		Op:     Fill{Rule: NonZero},
	},
}

// twoTriangles builds two separate, disjoint triangles.
func twoTriangles(cx1, cy1, cx2, cy2 float64, size float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// First triangle
		if !moveTo(yield, cx1, cy1-size) {
			return
		}
		if !lineTo(yield, cx1+size, cy1+size) {
			return
		}
		if !lineTo(yield, cx1-size, cy1+size) {
			return
		}
		if !closePath(yield) {
			return
		}

		// Second triangle
		if !moveTo(yield, cx2, cy2-size) {
			return
		}
		if !lineTo(yield, cx2+size, cy2+size) {
			return
		}
		if !lineTo(yield, cx2-size, cy2+size) {
			return
		}
		closePath(yield)
	}
}

// overlappingRectangles builds two overlapping rectangles.
func overlappingRectangles(x1a, y1a, x2a, y2a, x1b, y1b, x2b, y2b float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// First rectangle
		if !moveTo(yield, x1a, y1a) {
			return
		}
		if !lineTo(yield, x2a, y1a) {
			return
		}
		if !lineTo(yield, x2a, y2a) {
			return
		}
		if !lineTo(yield, x1a, y2a) {
			return
		}
		if !closePath(yield) {
			return
		}

		// Second rectangle
		if !moveTo(yield, x1b, y1b) {
			return
		}
		if !lineTo(yield, x2b, y1b) {
			return
		}
		if !lineTo(yield, x2b, y2b) {
			return
		}
		if !lineTo(yield, x1b, y2b) {
			return
		}
		closePath(yield)
	}
}

// ringShape builds a ring (outer rectangle with inner rectangle cutout).
func ringShape(cx, cy, outerSize, innerSize float64) path.Path {
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

		// Inner rectangle (clockwise - same winding, will be hole with even-odd)
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

// multipleRings builds multiple concentric donut shapes.
func multipleRings(cx, cy float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// Three concentric rings with different offsets
		rings := []struct{ cx, cy, outer, inner float64 }{
			{cx - 30, cy - 30, 20, 10},
			{cx + 30, cy - 30, 20, 10},
			{cx, cy + 30, 20, 10},
		}

		for _, ring := range rings {
			// Outer rectangle
			if !moveTo(yield, ring.cx-ring.outer, ring.cy-ring.outer) {
				return
			}
			if !lineTo(yield, ring.cx+ring.outer, ring.cy-ring.outer) {
				return
			}
			if !lineTo(yield, ring.cx+ring.outer, ring.cy+ring.outer) {
				return
			}
			if !lineTo(yield, ring.cx-ring.outer, ring.cy+ring.outer) {
				return
			}
			if !closePath(yield) {
				return
			}

			// Inner rectangle
			if !moveTo(yield, ring.cx-ring.inner, ring.cy-ring.inner) {
				return
			}
			if !lineTo(yield, ring.cx+ring.inner, ring.cy-ring.inner) {
				return
			}
			if !lineTo(yield, ring.cx+ring.inner, ring.cy+ring.inner) {
				return
			}
			if !lineTo(yield, ring.cx-ring.inner, ring.cy+ring.inner) {
				return
			}
			if !closePath(yield) {
				return
			}
		}
	}
}

// manySmallShapes builds a grid of small triangles (stress test).
func manySmallShapes(rows, cols int) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		size := 5.0
		spacing := 14.0

		for row := 0; row < rows; row++ {
			for col := 0; col < cols; col++ {
				cx := 10.0 + float64(col)*spacing
				cy := 10.0 + float64(row)*spacing

				// Small triangle
				if !moveTo(yield, cx, cy-size) {
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
}
