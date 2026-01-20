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
		Path:   multipleRings(32, 32, 25),
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
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: cx1, Y: cy1 - size}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx1 + size, Y: cy1 + size}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx1 - size, Y: cy1 + size}}) {
			return
		}
		if !yield(path.CmdClose, nil) {
			return
		}

		// Second triangle
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: cx2, Y: cy2 - size}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx2 + size, Y: cy2 + size}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx2 - size, Y: cy2 + size}}) {
			return
		}
		yield(path.CmdClose, nil)
	}
}

// overlappingRectangles builds two overlapping rectangles.
func overlappingRectangles(x1a, y1a, x2a, y2a, x1b, y1b, x2b, y2b float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// First rectangle
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: x1a, Y: y1a}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x2a, Y: y1a}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x2a, Y: y2a}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x1a, Y: y2a}}) {
			return
		}
		if !yield(path.CmdClose, nil) {
			return
		}

		// Second rectangle
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: x1b, Y: y1b}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x2b, Y: y1b}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x2b, Y: y2b}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x1b, Y: y2b}}) {
			return
		}
		yield(path.CmdClose, nil)
	}
}

// ringShape builds a ring (outer rectangle with inner rectangle cutout).
func ringShape(cx, cy, outerSize, innerSize float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// Outer rectangle (clockwise)
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: cx - outerSize, Y: cy - outerSize}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + outerSize, Y: cy - outerSize}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + outerSize, Y: cy + outerSize}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx - outerSize, Y: cy + outerSize}}) {
			return
		}
		if !yield(path.CmdClose, nil) {
			return
		}

		// Inner rectangle (clockwise - same winding, will be hole with even-odd)
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: cx - innerSize, Y: cy - innerSize}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + innerSize, Y: cy - innerSize}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + innerSize, Y: cy + innerSize}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx - innerSize, Y: cy + innerSize}}) {
			return
		}
		yield(path.CmdClose, nil)
	}
}

// multipleRings builds multiple concentric donut shapes.
func multipleRings(cx, cy, maxRadius float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// Three concentric rings with different offsets
		rings := []struct{ cx, cy, outer, inner float64 }{
			{cx - 30, cy - 30, 20, 10},
			{cx + 30, cy - 30, 20, 10},
			{cx, cy + 30, 20, 10},
		}

		for _, ring := range rings {
			// Outer rectangle
			if !yield(path.CmdMoveTo, []vec.Vec2{{X: ring.cx - ring.outer, Y: ring.cy - ring.outer}}) {
				return
			}
			if !yield(path.CmdLineTo, []vec.Vec2{{X: ring.cx + ring.outer, Y: ring.cy - ring.outer}}) {
				return
			}
			if !yield(path.CmdLineTo, []vec.Vec2{{X: ring.cx + ring.outer, Y: ring.cy + ring.outer}}) {
				return
			}
			if !yield(path.CmdLineTo, []vec.Vec2{{X: ring.cx - ring.outer, Y: ring.cy + ring.outer}}) {
				return
			}
			if !yield(path.CmdClose, nil) {
				return
			}

			// Inner rectangle
			if !yield(path.CmdMoveTo, []vec.Vec2{{X: ring.cx - ring.inner, Y: ring.cy - ring.inner}}) {
				return
			}
			if !yield(path.CmdLineTo, []vec.Vec2{{X: ring.cx + ring.inner, Y: ring.cy - ring.inner}}) {
				return
			}
			if !yield(path.CmdLineTo, []vec.Vec2{{X: ring.cx + ring.inner, Y: ring.cy + ring.inner}}) {
				return
			}
			if !yield(path.CmdLineTo, []vec.Vec2{{X: ring.cx - ring.inner, Y: ring.cy + ring.inner}}) {
				return
			}
			if !yield(path.CmdClose, nil) {
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
				if !yield(path.CmdMoveTo, []vec.Vec2{{X: cx, Y: cy - size}}) {
					return
				}
				if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + size, Y: cy + size}}) {
					return
				}
				if !yield(path.CmdLineTo, []vec.Vec2{{X: cx - size, Y: cy + size}}) {
					return
				}
				if !yield(path.CmdClose, nil) {
					return
				}
			}
		}
	}
}
