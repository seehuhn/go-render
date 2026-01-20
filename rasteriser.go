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

package render

import (
	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/pdf/graphics"
)

// Default values for rasteriser parameters.
const (
	// DefaultFlatness is the default curve flattening tolerance in device
	// pixels. Values of 0.25-1.0 are typical; 0.25 is below the threshold
	// of visual perception.
	DefaultFlatness = 0.25

	// DefaultMiterLimit is the default miter limit, matching PDF/PostScript.
	// This converts joins to bevels when the interior angle is less than
	// approximately 11.5 degrees.
	DefaultMiterLimit = 10.0
)

// Rasteriser converts vector paths to pixel coverage values.
// The caller creates one instance and reuses it for multiple paths.
// Internal buffers grow as needed but never shrink, achieving zero
// allocations in steady state.
type Rasteriser struct {
	// CTM is the current transformation matrix (user space to device space).
	// Must be a non-singular matrix.
	CTM matrix.Matrix

	// Clip defines the output region in device coordinates.
	// Must be a non-empty rectangle with integer-aligned coordinates.
	Clip rect.Rect

	// Flatness is the curve flattening tolerance in device pixels.
	// Must be > 0. Typical values are 0.25-1.0.
	Flatness float64

	// Width is the stroke line width in user-space units.
	// Must be > 0 for stroke operations.
	Width float64

	// Cap is the line cap style for stroke endpoints.
	Cap graphics.LineCapStyle

	// Join is the line join style for stroke corners.
	Join graphics.LineJoinStyle

	// MiterLimit is the miter limit for miter joins.
	// Must be >= 1.0.
	MiterLimit float64

	// Dash is the dash pattern in user-space units.
	// Nil means solid line (no dashing).
	Dash []float64

	// DashPhase is the offset into the dash pattern.
	DashPhase float64

	// Internal buffers (reused across calls)
	cover  []float32
	area   []float32
	output []float32
	stroke []vec.Vec2
}

// NewRasteriser creates a new Rasteriser with the given clip rectangle
// and PDF default values for all other parameters.
func NewRasteriser(clip rect.Rect) *Rasteriser {
	return &Rasteriser{
		CTM:        matrix.Identity,
		Clip:       clip,
		Flatness:   DefaultFlatness,
		Width:      1.0,
		Cap:        graphics.LineCapButt,
		Join:       graphics.LineJoinMiter,
		MiterLimit: DefaultMiterLimit,
	}
}

// FillNonZero rasterises the path using the nonzero winding rule.
// Coverage is delivered row-by-row via the emit callback.
// The coverage slice passed to emit is only valid for the duration
// of the callback.
func (r *Rasteriser) FillNonZero(p path.Path, emit func(y, xMin int, coverage []float32)) {
	// TODO: implement
}

// FillEvenOdd rasterises the path using the even-odd fill rule.
// Coverage is delivered row-by-row via the emit callback.
// The coverage slice passed to emit is only valid for the duration
// of the callback.
func (r *Rasteriser) FillEvenOdd(p path.Path, emit func(y, xMin int, coverage []float32)) {
	// TODO: implement
}

// Stroke rasterises the path as a stroked outline.
// Uses Width, Cap, Join, MiterLimit, Dash, and DashPhase from the Rasteriser.
// The stroke outline is filled using the nonzero winding rule.
// Coverage is delivered row-by-row via the emit callback.
// The coverage slice passed to emit is only valid for the duration
// of the callback.
func (r *Rasteriser) Stroke(p path.Path, emit func(y, xMin int, coverage []float32)) {
	// TODO: implement
}

// Reset releases all internal buffers, allowing memory to be reclaimed.
// The Rasteriser remains usable after Reset; buffers will be reallocated
// as needed on the next operation.
func (r *Rasteriser) Reset() {
	r.cover = nil
	r.area = nil
	r.output = nil
	r.stroke = nil
}
