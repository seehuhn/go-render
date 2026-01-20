package testcases

import (
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/pdf/graphics"
)

// TestCase defines a single rendering test.
type TestCase struct {
	Name   string    // lowercase a-z and _ only
	Path   path.Path // the geometry to render
	Width  int       // canvas width in pixels
	Height int       // canvas height in pixels
	Op     Operation // fill or stroke
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
