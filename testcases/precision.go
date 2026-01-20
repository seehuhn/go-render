package testcases

import (
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/pdf/graphics"
)

var precisionCases = []TestCase{
	// Section 6.1: Subpixel Positioning
	{
		Name:   "subpixel_offset_00",
		Path:   offsetRectangle(20, 20, 24, 24, 0.0),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "subpixel_offset_25",
		Path:   offsetRectangle(20, 20, 24, 24, 0.25),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "subpixel_offset_50",
		Path:   offsetRectangle(20, 20, 24, 24, 0.5),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "subpixel_offset_75",
		Path:   offsetRectangle(20, 20, 24, 24, 0.75),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "thin_line_y_integer",
		Path:   horizontalLineAt(5, 10.0, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      1.0,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "thin_line_y_half",
		Path:   horizontalLineAt(5, 10.5, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      1.0,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},

	// Section 6.2: Large Coordinates
	{
		Name:   "large_coord_centered",
		Path:   largeOffsetRectangle(1000, 1000, 20),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "small_shape_large_offset",
		Path:   smallShapeAtLargeOffset(10000, 10000, 2),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
	{
		Name:   "float64_precision",
		Path:   float64PrecisionShape(),
		Width:  64,
		Height: 64,
		Op:     Fill{Rule: NonZero},
	},
}

// offsetRectangle builds a rectangular path with a subpixel offset applied to all coordinates.
func offsetRectangle(x1, y1, w, h, offset float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		ox1 := x1 + offset
		oy1 := y1 + offset
		ox2 := x1 + w + offset
		oy2 := y1 + h + offset

		if !yield(path.CmdMoveTo, []vec.Vec2{{X: ox1, Y: oy1}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: ox2, Y: oy1}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: ox2, Y: oy2}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: ox1, Y: oy2}}) {
			return
		}
		yield(path.CmdClose, nil)
	}
}

// horizontalLineAt builds a horizontal line segment at a specific y position.
func horizontalLineAt(x1, y, x2 float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: x1, Y: y}}) {
			return
		}
		yield(path.CmdLineTo, []vec.Vec2{{X: x2, Y: y}})
	}
}

// largeOffsetRectangle builds a rectangle centered at large coordinates,
// but translated back to fit the canvas. This tests precision at large offsets.
func largeOffsetRectangle(cx, cy, size float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// Translate the large coordinates back to canvas center (32, 32)
		// The shape is computed at (cx, cy) then shifted to canvas
		translateX := 32 - cx
		translateY := 32 - cy

		x1 := cx - size/2 + translateX
		y1 := cy - size/2 + translateY
		x2 := cx + size/2 + translateX
		y2 := cy + size/2 + translateY

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

// smallShapeAtLargeOffset builds a tiny shape at a large offset,
// translated back to the canvas for rendering.
func smallShapeAtLargeOffset(cx, cy, size float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// The shape is logically at (cx, cy) but we translate it to canvas
		translateX := 32 - cx
		translateY := 32 - cy

		x1 := cx - size/2 + translateX
		y1 := cy - size/2 + translateY
		x2 := cx + size/2 + translateX
		y2 := cy + size/2 + translateY

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

// float64PrecisionShape builds a shape using coordinates that require
// full float64 precision to represent accurately.
func float64PrecisionShape() path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// Use coordinates with many significant digits
		// These values differ only in the low bits of float64
		base := 32.0
		delta1 := 0.123456789012345
		delta2 := 0.123456789012346

		x1 := base - 10 + delta1
		y1 := base - 10 + delta1
		x2 := base + 10 + delta2
		y2 := base + 10 + delta2

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
