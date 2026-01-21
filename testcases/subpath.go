// seehuhn.de/go/raster - a 2D rendering library
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
func twoTriangles(cx1, cy1, cx2, cy2 float64, size float64) *path.Data {
	// First triangle
	p := (&path.Data{}).
		MoveTo(pt(cx1, cy1-size)).
		LineTo(pt(cx1+size, cy1+size)).
		LineTo(pt(cx1-size, cy1+size)).
		Close()

	// Second triangle
	return p.
		MoveTo(pt(cx2, cy2-size)).
		LineTo(pt(cx2+size, cy2+size)).
		LineTo(pt(cx2-size, cy2+size)).
		Close()
}

// overlappingRectangles builds two overlapping rectangles.
func overlappingRectangles(x1a, y1a, x2a, y2a, x1b, y1b, x2b, y2b float64) *path.Data {
	// First rectangle
	p := (&path.Data{}).
		MoveTo(pt(x1a, y1a)).
		LineTo(pt(x2a, y1a)).
		LineTo(pt(x2a, y2a)).
		LineTo(pt(x1a, y2a)).
		Close()

	// Second rectangle
	return p.
		MoveTo(pt(x1b, y1b)).
		LineTo(pt(x2b, y1b)).
		LineTo(pt(x2b, y2b)).
		LineTo(pt(x1b, y2b)).
		Close()
}

// ringShape builds a ring (outer rectangle with inner rectangle cutout).
func ringShape(cx, cy, outerSize, innerSize float64) *path.Data {
	// Outer rectangle (clockwise)
	p := (&path.Data{}).
		MoveTo(pt(cx-outerSize, cy-outerSize)).
		LineTo(pt(cx+outerSize, cy-outerSize)).
		LineTo(pt(cx+outerSize, cy+outerSize)).
		LineTo(pt(cx-outerSize, cy+outerSize)).
		Close()

	// Inner rectangle (clockwise - same winding, will be hole with even-odd)
	return p.
		MoveTo(pt(cx-innerSize, cy-innerSize)).
		LineTo(pt(cx+innerSize, cy-innerSize)).
		LineTo(pt(cx+innerSize, cy+innerSize)).
		LineTo(pt(cx-innerSize, cy+innerSize)).
		Close()
}

// multipleRings builds multiple concentric donut shapes.
func multipleRings(cx, cy float64) *path.Data {
	// Three concentric rings with different offsets
	rings := []struct{ cx, cy, outer, inner float64 }{
		{cx - 30, cy - 30, 20, 10},
		{cx + 30, cy - 30, 20, 10},
		{cx, cy + 30, 20, 10},
	}

	p := &path.Data{}
	for _, ring := range rings {
		// Outer rectangle
		p = p.
			MoveTo(pt(ring.cx-ring.outer, ring.cy-ring.outer)).
			LineTo(pt(ring.cx+ring.outer, ring.cy-ring.outer)).
			LineTo(pt(ring.cx+ring.outer, ring.cy+ring.outer)).
			LineTo(pt(ring.cx-ring.outer, ring.cy+ring.outer)).
			Close()

		// Inner rectangle
		p = p.
			MoveTo(pt(ring.cx-ring.inner, ring.cy-ring.inner)).
			LineTo(pt(ring.cx+ring.inner, ring.cy-ring.inner)).
			LineTo(pt(ring.cx+ring.inner, ring.cy+ring.inner)).
			LineTo(pt(ring.cx-ring.inner, ring.cy+ring.inner)).
			Close()
	}

	return p
}

// manySmallShapes builds a grid of small triangles (stress test).
func manySmallShapes(rows, cols int) *path.Data {
	size := 5.0
	spacing := 14.0

	p := &path.Data{}
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			cx := 10.0 + float64(col)*spacing
			cy := 10.0 + float64(row)*spacing

			// Small triangle
			p = p.
				MoveTo(pt(cx, cy-size)).
				LineTo(pt(cx+size, cy+size)).
				LineTo(pt(cx-size, cy+size)).
				Close()
		}
	}

	return p
}
