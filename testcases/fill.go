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
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: x1, Y: y1}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x2, Y: y2}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x3, Y: y3}}) {
			return
		}
		yield(path.CmdClose, nil)
	}
}

// fivePointStar builds a five-pointed star (self-intersecting).
func fivePointStar(cx, cy, r float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// five points, connecting every second point
		pts := make([]vec.Vec2, 5)
		for i := range 5 {
			angle := float64(i)*2*math.Pi/5 - math.Pi/2
			pts[i] = vec.Vec2{
				X: cx + r*math.Cos(angle),
				Y: cy + r*math.Sin(angle),
			}
		}

		// draw star: 0 -> 2 -> 4 -> 1 -> 3 -> 0
		order := []int{0, 2, 4, 1, 3}
		if !yield(path.CmdMoveTo, []vec.Vec2{pts[order[0]]}) {
			return
		}
		for _, i := range order[1:] {
			if !yield(path.CmdLineTo, []vec.Vec2{pts[i]}) {
				return
			}
		}
		yield(path.CmdClose, nil)
	}
}

// rectangle builds a rectangular path.
func rectangle(x1, y1, x2, y2 float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: x1, Y: y1}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x2, Y: y1}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x2, Y: y2}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x1, Y: y2}}) {
			return
		}
		yield(path.CmdClose, nil)
	}
}

// concentricRectangles builds two nested rectangles (outer clockwise, inner counter-clockwise).
func concentricRectangles(cx, cy, outerSize, innerSize float64) path.Path {
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

		// Inner rectangle (counter-clockwise for hole)
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: cx - innerSize, Y: cy - innerSize}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx - innerSize, Y: cy + innerSize}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + innerSize, Y: cy + innerSize}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + innerSize, Y: cy - innerSize}}) {
			return
		}
		yield(path.CmdClose, nil)
	}
}

// overlappingCircles builds two overlapping circles using cubic Bezier approximation.
func overlappingCircles(cx1, cy1 float64, cx2, cy2 float64, r float64) path.Path {
	const kappa = 0.5522847498307936

	return func(yield func(path.Command, []vec.Vec2) bool) {
		// First circle
		k := r * kappa
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: cx1 + r, Y: cy1}}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx1 + r, Y: cy1 - k},
			{X: cx1 + k, Y: cy1 - r},
			{X: cx1, Y: cy1 - r},
		}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx1 - k, Y: cy1 - r},
			{X: cx1 - r, Y: cy1 - k},
			{X: cx1 - r, Y: cy1},
		}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx1 - r, Y: cy1 + k},
			{X: cx1 - k, Y: cy1 + r},
			{X: cx1, Y: cy1 + r},
		}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx1 + k, Y: cy1 + r},
			{X: cx1 + r, Y: cy1 + k},
			{X: cx1 + r, Y: cy1},
		}) {
			return
		}
		if !yield(path.CmdClose, nil) {
			return
		}

		// Second circle
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: cx2 + r, Y: cy2}}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx2 + r, Y: cy2 - k},
			{X: cx2 + k, Y: cy2 - r},
			{X: cx2, Y: cy2 - r},
		}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx2 - k, Y: cy2 - r},
			{X: cx2 - r, Y: cy2 - k},
			{X: cx2 - r, Y: cy2},
		}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx2 - r, Y: cy2 + k},
			{X: cx2 - k, Y: cy2 + r},
			{X: cx2, Y: cy2 + r},
		}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx2 + k, Y: cy2 + r},
			{X: cx2 + r, Y: cy2 + k},
			{X: cx2 + r, Y: cy2},
		}) {
			return
		}
		yield(path.CmdClose, nil)
	}
}

// figureEight builds a self-crossing figure-eight shape.
func figureEight(cx, cy, width, height float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// Start at top-left, cross through center
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: cx - width, Y: cy - height}}) {
			return
		}
		// Go to bottom-right
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + width, Y: cy + height}}) {
			return
		}
		// Go to top-right
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + width, Y: cy - height}}) {
			return
		}
		// Go to bottom-left (crossing the first line)
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx - width, Y: cy + height}}) {
			return
		}
		yield(path.CmdClose, nil)
	}
}

// highWindingRect builds a rectangle wound multiple times in the same direction.
func highWindingRect(cx, cy, size float64, windings int) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		for i := 0; i < windings; i++ {
			if !yield(path.CmdMoveTo, []vec.Vec2{{X: cx - size, Y: cy - size}}) {
				return
			}
			if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + size, Y: cy - size}}) {
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

// alternatingWinding builds a clockwise outer rectangle with a clockwise inner rectangle.
func alternatingWinding(cx, cy, outerSize, innerSize float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// Outer rectangle (counter-clockwise)
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: cx - outerSize, Y: cy - outerSize}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx - outerSize, Y: cy + outerSize}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + outerSize, Y: cy + outerSize}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + outerSize, Y: cy - outerSize}}) {
			return
		}
		if !yield(path.CmdClose, nil) {
			return
		}

		// Inner rectangle (clockwise - same direction conceptually)
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

// diamond builds a diamond shape (45 degree rotated square).
func diamond(cx, cy, size float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: cx, Y: cy - size}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx + size, Y: cy}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx, Y: cy + size}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: cx - size, Y: cy}}) {
			return
		}
		yield(path.CmdClose, nil)
	}
}

// nearHorizontalQuad builds a quadrilateral with near-horizontal top/bottom edges.
func nearHorizontalQuad(x1, y1, x2, y2 float64) path.Path {
	height := 10.0
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: x1, Y: y1}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x2, Y: y2}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x2, Y: y2 + height}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x1, Y: y1 + height}}) {
			return
		}
		yield(path.CmdClose, nil)
	}
}

// nearVerticalQuad builds a quadrilateral with near-vertical left/right edges.
func nearVerticalQuad(x1, y1, x2, y2 float64) path.Path {
	width := 10.0
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: x1, Y: y1}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x1 + width, Y: y1}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x2 + width, Y: y2}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x2, Y: y2}}) {
			return
		}
		yield(path.CmdClose, nil)
	}
}
