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
