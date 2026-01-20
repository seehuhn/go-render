package testcases

import (
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/pdf/graphics"
)

var strokeCases = []TestCase{
	{
		Name:   "line_butt",
		Path:   horizontalLine(10, 32, 54),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "line_round",
		Path:   horizontalLine(10, 32, 54),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapRound,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "line_square",
		Path:   horizontalLine(10, 32, 54),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      8,
			Cap:        graphics.LineCapSquare,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "corner_miter",
		Path:   corner(10, 50, 32, 14, 54, 50),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
		},
	},
	{
		Name:   "corner_round",
		Path:   corner(10, 50, 32, 14, 54, 50),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinRound,
			MiterLimit: 10,
		},
	},
	{
		Name:   "corner_bevel",
		Path:   corner(10, 50, 32, 14, 54, 50),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      6,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinBevel,
			MiterLimit: 10,
		},
	},
	{
		Name:   "dashed",
		Path:   horizontalLine(5, 32, 59),
		Width:  64,
		Height: 64,
		Op: Stroke{
			Width:      4,
			Cap:        graphics.LineCapButt,
			Join:       graphics.LineJoinMiter,
			MiterLimit: 10,
			Dash:       []float64{8, 4},
			DashPhase:  0,
		},
	},
}

// horizontalLine builds a horizontal line segment.
func horizontalLine(x1, y, x2 float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: x1, Y: y}}) {
			return
		}
		yield(path.CmdLineTo, []vec.Vec2{{X: x2, Y: y}})
	}
}

// corner builds a path with two line segments meeting at a corner.
func corner(x1, y1, x2, y2, x3, y3 float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: x1, Y: y1}}) {
			return
		}
		if !yield(path.CmdLineTo, []vec.Vec2{{X: x2, Y: y2}}) {
			return
		}
		yield(path.CmdLineTo, []vec.Vec2{{X: x3, Y: y3}})
	}
}
