package testcases

import (
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
)

var curveCases = []TestCase{
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
}

// quadraticCurve builds a closed shape with a quadratic Bezier curve.
func quadraticCurve(x1, y1, cx, cy, x2, y2 float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: x1, Y: y1}}) {
			return
		}
		if !yield(path.CmdQuadTo, []vec.Vec2{{X: cx, Y: cy}, {X: x2, Y: y2}}) {
			return
		}
		yield(path.CmdClose, nil)
	}
}

// cubicCurve builds a closed shape with a cubic Bezier curve.
func cubicCurve(x1, y1, c1x, c1y, c2x, c2y, x2, y2 float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: x1, Y: y1}}) {
			return
		}
		if !yield(path.CmdCubeTo, []vec.Vec2{{X: c1x, Y: c1y}, {X: c2x, Y: c2y}, {X: x2, Y: y2}}) {
			return
		}
		yield(path.CmdClose, nil)
	}
}

// circle builds an approximate circle using four cubic Bezier curves.
func circle(cx, cy, r float64) path.Path {
	// kappa for cubic Bezier approximation of a quarter circle
	const kappa = 0.5522847498307936

	return func(yield func(path.Command, []vec.Vec2) bool) {
		k := r * kappa

		// start at right
		if !yield(path.CmdMoveTo, []vec.Vec2{{X: cx + r, Y: cy}}) {
			return
		}
		// top-right quadrant
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx + r, Y: cy - k},
			{X: cx + k, Y: cy - r},
			{X: cx, Y: cy - r},
		}) {
			return
		}
		// top-left quadrant
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx - k, Y: cy - r},
			{X: cx - r, Y: cy - k},
			{X: cx - r, Y: cy},
		}) {
			return
		}
		// bottom-left quadrant
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx - r, Y: cy + k},
			{X: cx - k, Y: cy + r},
			{X: cx, Y: cy + r},
		}) {
			return
		}
		// bottom-right quadrant
		if !yield(path.CmdCubeTo, []vec.Vec2{
			{X: cx + k, Y: cy + r},
			{X: cx + r, Y: cy + k},
			{X: cx + r, Y: cy},
		}) {
			return
		}
		yield(path.CmdClose, nil)
	}
}
