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
	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/pdf/graphics"
)

// TestCase defines a single rendering test.
type TestCase struct {
	Name   string        // lowercase a-z and _ only
	Path   path.Path     // the geometry to render
	Width  int           // canvas width in pixels
	Height int           // canvas height in pixels
	Op     Operation     // fill or stroke
	CTM    matrix.Matrix // transformation matrix (zero-value means no transform)
}

// Operation is the rendering operation to apply to the path.
type Operation interface {
	isOperation()
}

// FillRule specifies the rule for determining interior points.
type FillRule int

const (
	NonZero FillRule = iota
	EvenOdd
)

// Fill specifies a fill operation.
type Fill struct {
	Rule FillRule
}

func (Fill) isOperation() {}

// Stroke specifies a stroke operation.
type Stroke struct {
	Width      float64                // line width (>0)
	Cap        graphics.LineCapStyle  // LineCapButt, LineCapRound, LineCapSquare
	Join       graphics.LineJoinStyle // LineJoinMiter, LineJoinRound, LineJoinBevel
	MiterLimit float64                // miter limit
	Dash       []float64              // dash pattern (nil for solid)
	DashPhase  float64                // dash phase offset
}

func (Stroke) isOperation() {}

// Path building helpers with zero allocations.
// These use a package-level buffer that is reused for each yield.
// Safe because path iteration is single-threaded within a call.

var pathBuf [3]vec.Vec2

func moveTo(yield func(path.Command, []vec.Vec2) bool, x, y float64) bool {
	pathBuf[0] = vec.Vec2{X: x, Y: y}
	return yield(path.CmdMoveTo, pathBuf[:1])
}

func lineTo(yield func(path.Command, []vec.Vec2) bool, x, y float64) bool {
	pathBuf[0] = vec.Vec2{X: x, Y: y}
	return yield(path.CmdLineTo, pathBuf[:1])
}

func quadTo(yield func(path.Command, []vec.Vec2) bool, x1, y1, x2, y2 float64) bool {
	pathBuf[0] = vec.Vec2{X: x1, Y: y1}
	pathBuf[1] = vec.Vec2{X: x2, Y: y2}
	return yield(path.CmdQuadTo, pathBuf[:2])
}

func cubeTo(yield func(path.Command, []vec.Vec2) bool, x1, y1, x2, y2, x3, y3 float64) bool {
	pathBuf[0] = vec.Vec2{X: x1, Y: y1}
	pathBuf[1] = vec.Vec2{X: x2, Y: y2}
	pathBuf[2] = vec.Vec2{X: x3, Y: y3}
	return yield(path.CmdCubeTo, pathBuf[:3])
}

func closePath(yield func(path.Command, []vec.Vec2) bool) bool {
	return yield(path.CmdClose, nil)
}
