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
	"math"
	"slices"

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

// Numerical tolerances for the rasteriser.
const (
	// horizontalEdgeThreshold is the minimum vertical extent for an edge
	// to contribute to coverage. Edges with |y1 - y0| below this threshold
	// are skipped as horizontal.
	horizontalEdgeThreshold = 1e-10

	// smallPathThreshold is the maximum bounding box area (in pixels) for
	// using 2D buffers (Approach A). Paths with larger bounding boxes use
	// the active edge list (Approach B).
	// TODO: tune this threshold based on profiling
	smallPathThreshold = 65536

	// zeroLengthThreshold is the minimum length for a stroke segment.
	// Segments shorter than this are skipped.
	zeroLengthThreshold = 1e-10

	// collinearityThreshold is used to detect nearly collinear segments
	// where no join is needed.
	collinearityThreshold = 1e-6

	// cuspCosineThreshold is the cosine threshold for detecting cusps
	// (path doubling back on itself). cos(179.43°) ≈ -0.9999
	cuspCosineThreshold = -0.9999
)

// edge represents a line segment in device coordinates.
type edge struct {
	x0, y0 float64 // start point
	x1, y1 float64 // end point
	dxdy   float64 // (x1-x0)/(y1-y0), precomputed for x-intercept calculation
}

// crossing represents a point where an edge crosses a pixel boundary.
type crossing struct {
	y float64
	x float64
}

// strokeSegment represents a line segment with precomputed geometry for stroking.
type strokeSegment struct {
	A, B vec.Vec2 // endpoints in user space
	T    vec.Vec2 // unit tangent (A→B direction)
	N    vec.Vec2 // unit normal (90° CCW from T)
}

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

	// smallPathThreshold is the maximum bounding box area (in pixels) for
	// using 2D buffers (Approach A). Paths with larger bounding boxes use
	// the active edge list (Approach B).
	smallPathThreshold int

	// Internal buffers (reused across calls)
	cover         []float32   // coverage accumulation: cover change per pixel
	area          []float32   // coverage accumulation: area within pixel
	output        []float32   // final coverage values for callback
	edges         []edge      // edge list for current path (device coordinates)
	active        []int       // indices of active edges (for Approach B)
	xMin          []int       // per-scanline minimum x with edge contribution
	xMax          []int       // per-scanline maximum x with edge contribution
	stroke        []vec.Vec2 // stroke outline vertices (all subpaths contiguous)
	strokeOffsets []int      // start index of each stroke polygon in stroke[]
	crossings     []crossing // reusable buffer for edge/pixel boundary crossings

	// Flattening buffers (for stroke path processing)
	flattenedSegs    []strokeSegment // all segments from all subpaths, contiguous
	flattenedOffsets []int           // start index of each subpath in flattenedSegs
	flattenedClosed  []bool          // whether each subpath is closed
	degeneratePoints []vec.Vec2      // degenerate subpaths (no orientation)

	// Edge collection state (used by collectEdges/addEdge)
	edgeBBoxFirst bool    // true if no edges added yet
	edgeDevXMin   float64 // bounding box in device space
	edgeDevXMax   float64
	edgeDevYMin   float64
	edgeDevYMax   float64

	// Dash pattern output buffers
	dashedSegs    []strokeSegment // all dashed segments, contiguous
	dashedOffsets []int           // start index of each dashed subpath
}

// NewRasteriser creates a new Rasteriser with the given clip rectangle
// and PDF default values for all other parameters.
func NewRasteriser(clip rect.Rect) *Rasteriser {
	return &Rasteriser{
		CTM:                matrix.Identity,
		Clip:               clip,
		Flatness:           DefaultFlatness,
		Width:              1.0,
		Cap:                graphics.LineCapButt,
		Join:               graphics.LineJoinMiter,
		MiterLimit:         DefaultMiterLimit,
		smallPathThreshold: smallPathThreshold,
	}
}

// transformLinear applies only the 2×2 linear part of CTM to a vector.
// Used for CTM-aware tolerance checking where translation is irrelevant.
func (r *Rasteriser) transformLinear(v vec.Vec2) vec.Vec2 {
	return vec.Vec2{
		X: r.CTM[0]*v.X + r.CTM[2]*v.Y,
		Y: r.CTM[1]*v.X + r.CTM[3]*v.Y,
	}
}

// flattenQuadratic flattens a quadratic Bézier and calls emit for each line segment.
// p0 is the start point (current point), p1 is control, p2 is endpoint.
// All points are in user space; CTM-aware tolerance checking is used.
func (r *Rasteriser) flattenQuadratic(p0, p1, p2 vec.Vec2, emit func(from, to vec.Vec2)) {
	// Compute error vector: e = (P0 - 2*P1 + P2) / 4
	e := p0.Sub(p1.Mul(2)).Add(p2).Mul(0.25)

	// Transform to device space
	eDev := r.transformLinear(e)

	// Compute segment count
	n := 1
	errDev := eDev.Length()
	if errDev > r.Flatness {
		n = int(math.Ceil(math.Sqrt(errDev / r.Flatness)))
	}

	// Evaluate curve at n+1 points and emit segments
	prev := p0
	for i := 1; i <= n; i++ {
		t := float64(i) / float64(n)
		// B(t) = (1-t)²P0 + 2(1-t)tP1 + t²P2
		omt := 1 - t
		pt := p0.Mul(omt * omt).Add(p1.Mul(2 * omt * t)).Add(p2.Mul(t * t))
		emit(prev, pt)
		prev = pt
	}
}

// flattenCubic flattens a cubic Bézier and calls emit for each line segment.
// p0 is start, p1/p2 are controls, p3 is endpoint. All in user space.
func (r *Rasteriser) flattenCubic(p0, p1, p2, p3 vec.Vec2, emit func(from, to vec.Vec2)) {
	// Compute deviation vectors
	d1 := p0.Sub(p1.Mul(2)).Add(p2) // P0 - 2*P1 + P2
	d2 := p1.Sub(p2.Mul(2)).Add(p3) // P1 - 2*P2 + P3

	// Transform to device space
	d1Dev := r.transformLinear(d1)
	d2Dev := r.transformLinear(d2)

	// Compute segment count using Wang's formula
	mDev := max(d1Dev.Length(), d2Dev.Length())
	n := 1
	if mDev > 0 {
		// n = ceil(sqrt(3 * mDev / (4 * ε)))
		nFloat := math.Sqrt(3 * mDev / (4 * r.Flatness))
		if nFloat > 1 {
			n = int(math.Ceil(nFloat))
		}
	}

	// Evaluate curve at n+1 points and emit segments
	prev := p0
	for i := 1; i <= n; i++ {
		t := float64(i) / float64(n)
		// B(t) = (1-t)³P0 + 3(1-t)²tP1 + 3(1-t)t²P2 + t³P3
		omt := 1 - t
		omt2 := omt * omt
		omt3 := omt2 * omt
		t2 := t * t
		t3 := t2 * t
		pt := p0.Mul(omt3).Add(p1.Mul(3 * omt2 * t)).Add(p2.Mul(3 * omt * t2)).Add(p3.Mul(t3))
		emit(prev, pt)
		prev = pt
	}
}

// FillNonZero rasterises the path using the nonzero winding rule.
// Coverage is delivered row-by-row via the emit callback.
// The coverage slice passed to emit is only valid for the duration
// of the callback.
func (r *Rasteriser) FillNonZero(p *path.Data, emit func(y, xMin int, coverage []float32)) {
	r.fill(p, fillNonZero, emit)
}

// FillEvenOdd rasterises the path using the even-odd fill rule.
// Coverage is delivered row-by-row via the emit callback.
// The coverage slice passed to emit is only valid for the duration
// of the callback.
func (r *Rasteriser) FillEvenOdd(p *path.Data, emit func(y, xMin int, coverage []float32)) {
	r.fill(p, fillEvenOdd, emit)
}

// fillRule identifies which fill rule to apply.
type fillRule int

const (
	fillNonZero fillRule = iota
	fillEvenOdd
)

// fill is the internal implementation shared by FillNonZero and FillEvenOdd.
func (r *Rasteriser) fill(p *path.Data, rule fillRule, emit func(y, xMin int, coverage []float32)) {
	// Collect edges from path (returns bounding box clamped to clip)
	xMin, xMax, yMin, yMax, ok := r.collectEdges(p)
	if !ok {
		return // empty or degenerate path
	}

	// Choose approach based on bounding box size
	width := xMax - xMin
	height := yMax - yMin

	if width*height < r.smallPathThreshold {
		r.fill2D(xMin, xMax, yMin, yMax, rule, emit)
	} else {
		r.fillScanline(xMin, xMax, yMin, yMax, rule, emit)
	}
}

// collectEdges walks the path, transforms to device space, and builds the edge list.
// Curves are approximated with a single line segment (TODO: implement proper flattening).
// Returns the bounding box of all edges in device coordinates (clamped to clip).
func (r *Rasteriser) collectEdges(p *path.Data) (xMin, xMax, yMin, yMax int, ok bool) {
	r.edges = r.edges[:0]
	r.edgeBBoxFirst = true

	// Path state
	var current vec.Vec2 // current point (user space)
	var subpath vec.Vec2 // subpath start (user space)

	// Walk the path using direct field access (no iterator allocation)
	coordIdx := 0
	for _, cmd := range p.Cmds {
		switch cmd {
		case path.CmdMoveTo:
			current = p.Coords[coordIdx]
			subpath = current
			coordIdx++

		case path.CmdLineTo:
			r.addEdge(current, p.Coords[coordIdx])
			current = p.Coords[coordIdx]
			coordIdx++

		case path.CmdQuadTo:
			r.flattenQuadratic(current, p.Coords[coordIdx], p.Coords[coordIdx+1], r.addEdge)
			current = p.Coords[coordIdx+1]
			coordIdx += 2

		case path.CmdCubeTo:
			r.flattenCubic(current, p.Coords[coordIdx], p.Coords[coordIdx+1], p.Coords[coordIdx+2], r.addEdge)
			current = p.Coords[coordIdx+2]
			coordIdx += 3

		case path.CmdClose:
			if current != subpath {
				r.addEdge(current, subpath)
			}
			current = subpath
		}
	}

	if len(r.edges) == 0 {
		return 0, 0, 0, 0, false
	}

	// Clamp to clip bounds and convert to integers
	clipXMin := int(r.Clip.LLx)
	clipXMax := int(r.Clip.URx)
	clipYMin := int(r.Clip.LLy)
	clipYMax := int(r.Clip.URy)

	xMin = max(int(math.Floor(r.edgeDevXMin)), clipXMin)
	xMax = min(int(math.Floor(r.edgeDevXMax))+1, clipXMax)
	yMin = max(int(math.Floor(r.edgeDevYMin)), clipYMin)
	yMax = min(int(math.Floor(r.edgeDevYMax))+1, clipYMax)

	if xMin >= xMax || yMin >= yMax {
		return 0, 0, 0, 0, false
	}

	return xMin, xMax, yMin, yMax, true
}

// addEdge adds an edge from user space coordinates, transforming to device space.
func (r *Rasteriser) addEdge(p0, p1 vec.Vec2) {
	// Transform to device space
	dx0 := r.CTM[0]*p0.X + r.CTM[2]*p0.Y + r.CTM[4]
	dy0 := r.CTM[1]*p0.X + r.CTM[3]*p0.Y + r.CTM[5]
	dx1 := r.CTM[0]*p1.X + r.CTM[2]*p1.Y + r.CTM[4]
	dy1 := r.CTM[1]*p1.X + r.CTM[3]*p1.Y + r.CTM[5]

	// Skip horizontal edges
	dy := dy1 - dy0
	if dy > -horizontalEdgeThreshold && dy < horizontalEdgeThreshold {
		return
	}

	// Compute dxdy
	dxdy := (dx1 - dx0) / dy

	r.edges = append(r.edges, edge{
		x0: dx0, y0: dy0,
		x1: dx1, y1: dy1,
		dxdy: dxdy,
	})

	// Update bounding box
	if r.edgeBBoxFirst {
		r.edgeDevXMin = min(dx0, dx1)
		r.edgeDevXMax = max(dx0, dx1)
		r.edgeDevYMin = min(dy0, dy1)
		r.edgeDevYMax = max(dy0, dy1)
		r.edgeBBoxFirst = false
	} else {
		r.edgeDevXMin = min(r.edgeDevXMin, min(dx0, dx1))
		r.edgeDevXMax = max(r.edgeDevXMax, max(dx0, dx1))
		r.edgeDevYMin = min(r.edgeDevYMin, min(dy0, dy1))
		r.edgeDevYMax = max(r.edgeDevYMax, max(dy0, dy1))
	}
}

// accumulateEdge adds a single edge's contribution to the cover and area buffers.
// The buffers are indexed by (x - bboxXMin), where bboxXMin/bboxXMax define the buffer range.
// For edges spanning multiple pixels horizontally, this function splits the edge at pixel
// boundaries and computes separate contributions for each pixel crossed.
func (r *Rasteriser) accumulateEdge(e *edge, y int, cover, area []float32, bboxXMin, bboxXMax int) {
	// Compute the portion of the edge within this scanline [y, y+1)
	yTop := float64(y)
	yBot := float64(y + 1)

	// Clamp to edge's actual y extent
	edgeYMin := min(e.y0, e.y1)
	edgeYMax := max(e.y0, e.y1)
	yTop = max(yTop, edgeYMin)
	yBot = min(yBot, edgeYMax)

	if yBot <= yTop {
		return
	}

	// Sign based on edge direction: +1 for downward (y1 > y0), -1 for upward
	sign := float32(1)
	if e.y1 < e.y0 {
		sign = -1
	}

	// Compute x at the y boundaries of the edge segment within this scanline
	xAtYTop := e.x0 + e.dxdy*(yTop-e.y0)
	xAtYBot := e.x0 + e.dxdy*(yBot-e.y0)

	// Determine pixel range the edge spans (ensure left <= right for iteration)
	xLeft, xRight := xAtYTop, xAtYBot
	if xLeft > xRight {
		xLeft, xRight = xRight, xLeft
	}

	pixLeft := int(math.Floor(xLeft))
	pixRight := int(math.Floor(xRight))

	// Handle edge entirely to the left of bbox
	if pixRight < bboxXMin {
		coverVal := sign * float32(yBot-yTop)
		cover[0] += coverVal
		area[0] += coverVal
		return
	}

	// Handle edge entirely to the right of bbox
	if pixLeft >= bboxXMax {
		return
	}

	// For vertical edges or edges within a single pixel column
	if pixLeft == pixRight {
		r.accumulateSinglePixel(e, yTop, yBot, sign, pixLeft, cover, area, bboxXMin, bboxXMax)
		return
	}

	// Edge spans multiple pixels - split at each pixel boundary
	// Collect all y values where the edge crosses integer x boundaries
	// Then process each segment between consecutive crossings
	dydx := 1 / e.dxdy

	// Build list of (y, x) crossing points, sorted by y (reuse buffer)
	r.crossings = r.crossings[:0]

	// Add start and end points
	r.crossings = append(r.crossings, crossing{yTop, xAtYTop})
	r.crossings = append(r.crossings, crossing{yBot, xAtYBot})

	// Add crossings at integer x boundaries
	for x := pixLeft + 1; x <= pixRight; x++ {
		yAtX := e.y0 + dydx*(float64(x)-e.x0)
		if yAtX > yTop && yAtX < yBot {
			r.crossings = append(r.crossings, crossing{yAtX, float64(x)})
		}
	}

	// Sort crossings by y
	slices.SortFunc(r.crossings, func(a, b crossing) int {
		if a.y < b.y {
			return -1
		}
		if a.y > b.y {
			return 1
		}
		return 0
	})

	// Process each segment between consecutive crossings
	for i := range len(r.crossings) - 1 {
		y0 := r.crossings[i].y
		y1 := r.crossings[i+1].y
		segDy := y1 - y0
		if segDy <= 0 {
			continue
		}

		// Compute contribution for this segment
		coverVal := sign * float32(segDy)

		// Find which pixel this segment is in (use midpoint x)
		yMid := (y0 + y1) / 2
		xMid := e.x0 + e.dxdy*(yMid-e.y0)
		pix := int(math.Floor(xMid))

		xFrac := xMid - float64(pix)
		areaVal := coverVal * float32(1-xFrac)

		// Add to buffers
		if pix < bboxXMin {
			cover[0] += coverVal
			area[0] += coverVal
		} else if pix < bboxXMax {
			idx := pix - bboxXMin
			cover[idx] += coverVal
			area[idx] += areaVal
		}
		// pix >= bboxXMax: no contribution
	}
}

// accumulateSinglePixel handles an edge segment that falls within a single pixel column.
func (r *Rasteriser) accumulateSinglePixel(e *edge, yTop, yBot float64, sign float32, pix int, cover, area []float32, bboxXMin, bboxXMax int) {
	coverVal := sign * float32(yBot-yTop)

	if pix < bboxXMin {
		cover[0] += coverVal
		area[0] += coverVal
		return
	}
	if pix >= bboxXMax {
		return
	}

	// Compute average x within this pixel
	yMid := (yTop + yBot) / 2
	xMid := e.x0 + e.dxdy*(yMid-e.y0)
	xFrac := xMid - float64(pix)
	areaVal := coverVal * float32(1-xFrac)

	idx := pix - bboxXMin
	cover[idx] += coverVal
	area[idx] += areaVal
}

// integrateScanline converts accumulated cover/area to final coverage values.
// xMin and xMax are the pixel range that was touched.
func (r *Rasteriser) integrateScanline(cover, area []float32, xMin, xMax int, rule fillRule) []float32 {
	width := xMax - xMin + 1
	if cap(r.output) < width {
		r.output = make([]float32, width)
	} else {
		r.output = r.output[:width]
	}

	var accum float32
	for i := range width {
		raw := accum + area[i]
		accum += cover[i]

		var cov float32
		switch rule {
		case fillNonZero:
			// clamp(abs(raw), 0, 1)
			if raw < 0 {
				cov = -raw
			} else {
				cov = raw
			}
			if cov > 1 {
				cov = 1
			}
		case fillEvenOdd:
			// 1 - abs(1 - mod(abs(raw), 2))
			if raw < 0 {
				raw = -raw
			}
			// mod(raw, 2) using floor
			mod := raw - 2*float32(int(raw/2))
			cov = 1 - abs32(1-mod)
		}
		r.output[i] = cov
	}

	return r.output
}

// abs32 returns the absolute value of a float32.
func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// fill2D rasterises using 2D buffers (Approach A).
// Used for small paths where width*height < smallPathThreshold.
// xMin, xMax, yMin, yMax define the path's bounding box (already clamped to clip).
func (r *Rasteriser) fill2D(xMin, xMax, yMin, yMax int, rule fillRule, emit func(y, xMin int, coverage []float32)) {
	width := xMax - xMin
	height := yMax - yMin

	// Ensure 2D buffers are large enough
	size := width * height
	if cap(r.cover) < size {
		r.cover = make([]float32, size)
		r.area = make([]float32, size)
	} else {
		r.cover = r.cover[:size]
		r.area = r.area[:size]
		// Zero the buffers
		for i := range r.cover {
			r.cover[i] = 0
			r.area[i] = 0
		}
	}

	// Ensure xMin/xMax tracking buffers are large enough
	if cap(r.xMin) < height {
		r.xMin = make([]int, height)
		r.xMax = make([]int, height)
	} else {
		r.xMin = r.xMin[:height]
		r.xMax = r.xMax[:height]
	}
	// Initialize bounds to "no edges"
	for i := range r.xMin {
		r.xMin[i] = width
		r.xMax[i] = -1
	}

	// Process all edges into 2D buffers
	for i := range r.edges {
		e := &r.edges[i]

		// Determine scanline range for this edge
		var edgeYMin, edgeYMax int
		if e.y0 < e.y1 {
			edgeYMin = int(math.Floor(e.y0))
			edgeYMax = int(math.Floor(e.y1)) + 1
		} else {
			edgeYMin = int(math.Floor(e.y1))
			edgeYMax = int(math.Floor(e.y0)) + 1
		}
		edgeYMin = max(edgeYMin, yMin)
		edgeYMax = min(edgeYMax, yMax)

		// Accumulate into each scanline
		for y := edgeYMin; y < edgeYMax; y++ {
			row := y - yMin
			rowOffset := row * width
			r.accumulateEdge(e, y, r.cover[rowOffset:rowOffset+width], r.area[rowOffset:rowOffset+width], xMin, xMax)

			// Update x bounds for this row
			// Compute x at midpoint of edge within this scanline
			yTop := max(float64(y), min(e.y0, e.y1))
			yBot := min(float64(y+1), max(e.y0, e.y1))
			yMid := (yTop + yBot) / 2
			xMidF := e.x0 + e.dxdy*(yMid-e.y0)
			x := int(math.Floor(xMidF))
			x = max(x, xMin)
			x = min(x, xMax-1)
			xIdx := x - xMin
			if xIdx < r.xMin[row] {
				r.xMin[row] = xIdx
			}
			if xIdx > r.xMax[row] {
				r.xMax[row] = xIdx
			}
		}
	}

	// Integrate and emit each row
	for row := range height {
		if r.xMax[row] < 0 {
			continue // no edges touched this row
		}

		y := yMin + row
		rowOffset := row * width

		// Integrate the full width (cover accumulates from left)
		coverage := r.integrateScanline(r.cover[rowOffset:rowOffset+width], r.area[rowOffset:rowOffset+width], 0, width-1, rule)

		// Emit only the non-zero portion
		// Find actual xMin/xMax with non-zero coverage
		outXMin := 0
		outXMax := width - 1
		for outXMin < width && coverage[outXMin] == 0 {
			outXMin++
		}
		for outXMax > outXMin && coverage[outXMax] == 0 {
			outXMax--
		}
		if outXMin <= outXMax {
			emit(y, xMin+outXMin, coverage[outXMin:outXMax+1])
		}
	}
}

// fillScanline rasterises using 1D buffers and an active edge list (Approach B).
// Used for large paths where width*height >= smallPathThreshold.
// xMin, xMax, yMin, yMax define the path's bounding box (already clamped to clip).
func (r *Rasteriser) fillScanline(xMin, xMax, yMin, yMax int, rule fillRule, emit func(y, xMin int, coverage []float32)) {
	width := xMax - xMin

	// Ensure 1D buffers are large enough
	if cap(r.cover) < width {
		r.cover = make([]float32, width)
		r.area = make([]float32, width)
	} else {
		r.cover = r.cover[:width]
		r.area = r.area[:width]
	}

	// Sort edges by y_min
	slices.SortFunc(r.edges, func(a, b edge) int {
		aYMin := min(a.y0, a.y1)
		bYMin := min(b.y0, b.y1)
		if aYMin < bYMin {
			return -1
		}
		if aYMin > bYMin {
			return 1
		}
		return 0
	})

	// Active edge list (indices into r.edges)
	r.active = r.active[:0]
	nextEdge := 0

	// Process scanlines
	for y := yMin; y < yMax; y++ {
		yf := float64(y)
		yfNext := float64(y + 1)

		// Add edges that start at this scanline
		for nextEdge < len(r.edges) {
			e := &r.edges[nextEdge]
			edgeYMin := min(e.y0, e.y1)
			if edgeYMin >= yfNext {
				break
			}
			r.active = append(r.active, nextEdge)
			nextEdge++
		}

		if len(r.active) == 0 {
			continue
		}

		// Clear buffers for this scanline
		for i := range r.cover {
			r.cover[i] = 0
			r.area[i] = 0
		}

		// Track x bounds for this scanline
		xMinBound := width
		xMaxBound := -1

		// Process active edges
		for i := 0; i < len(r.active); {
			e := &r.edges[r.active[i]]

			// Check if edge ends before this scanline
			edgeYMax := max(e.y0, e.y1)
			if edgeYMax <= yf {
				// Remove from active list (swap with last)
				r.active[i] = r.active[len(r.active)-1]
				r.active = r.active[:len(r.active)-1]
				continue
			}

			// Accumulate contribution
			r.accumulateEdge(e, y, r.cover, r.area, xMin, xMax)

			// Update x bounds
			yTop := max(yf, min(e.y0, e.y1))
			yBot := min(yfNext, max(e.y0, e.y1))
			if yBot > yTop {
				yMid := (yTop + yBot) / 2
				xMidF := e.x0 + e.dxdy*(yMid-e.y0)
				x := int(math.Floor(xMidF))
				x = max(x, xMin)
				x = min(x, xMax-1)
				xIdx := x - xMin
				if xIdx < xMinBound {
					xMinBound = xIdx
				}
				if xIdx > xMaxBound {
					xMaxBound = xIdx
				}
			}

			i++
		}

		if xMaxBound < 0 {
			continue // no edges contributed to this scanline
		}

		// Integrate and emit
		coverage := r.integrateScanline(r.cover, r.area, 0, width-1, rule)

		// Find actual non-zero range
		outXMin := 0
		outXMax := width - 1
		for outXMin < width && coverage[outXMin] == 0 {
			outXMin++
		}
		for outXMax > outXMin && coverage[outXMax] == 0 {
			outXMax--
		}
		if outXMin <= outXMax {
			emit(y, xMin+outXMin, coverage[outXMin:outXMax+1])
		}
	}
}

// Stroke rasterises the path as a stroked outline.
// Uses Width, Cap, Join, MiterLimit, Dash, and DashPhase from the Rasteriser.
// The stroke outline is filled using the nonzero winding rule.
// Coverage is delivered row-by-row via the emit callback.
// The coverage slice passed to emit is only valid for the duration
// of the callback.
func (r *Rasteriser) Stroke(p *path.Data, emit func(y, xMin int, coverage []float32)) {
	// Flatten path into subpaths (results stored in r.flattenedSegs, etc.)
	r.flattenPath(p)
	if len(r.flattenedOffsets) == 0 && len(r.degeneratePoints) == 0 {
		return
	}

	// Build stroke outlines for all subpaths into a single contiguous buffer.
	// strokeOffsets tracks where each polygon starts. This ensures overlapping
	// dash segments are composited correctly using the nonzero winding rule.
	r.stroke = r.stroke[:0]
	r.strokeOffsets = r.strokeOffsets[:0]

	// Handle degenerate subpaths (no orientation): only round cap produces circle
	if r.Cap == graphics.LineCapRound {
		for _, pt := range r.degeneratePoints {
			startOffset := len(r.stroke)
			r.addArc(pt, r.Width/2, vec.Vec2{X: 1, Y: 0}, 2*math.Pi, true)
			r.strokeOffsets = append(r.strokeOffsets, startOffset)
		}
	}

	// Apply dash pattern if specified
	if len(r.Dash) > 0 {
		r.strokeDashedSubpaths()
	} else {
		r.strokeAllSubpaths()
	}

	// Fill all stroke polygons together as a compound path
	r.fillStrokeOutlines(emit)
}

// strokeAllSubpaths strokes all flattened subpaths (non-dashed case).
func (r *Rasteriser) strokeAllSubpaths() {
	numSubpaths := len(r.flattenedOffsets)
	for i := 0; i < numSubpaths; i++ {
		segs := r.getSubpathSegments(i)
		closed := r.flattenedClosed[i]

		startOffset := len(r.stroke)
		r.strokeSubpath(segs, closed)
		if len(r.stroke)-startOffset >= 3 {
			r.strokeOffsets = append(r.strokeOffsets, startOffset)
		} else {
			// Degenerate polygon, discard by resetting to start
			r.stroke = r.stroke[:startOffset]
		}
	}
}

// getSubpathSegments returns the segments for subpath i as a slice into flattenedSegs.
func (r *Rasteriser) getSubpathSegments(i int) []strokeSegment {
	start := r.flattenedOffsets[i]
	var end int
	if i+1 < len(r.flattenedOffsets) {
		end = r.flattenedOffsets[i+1]
	} else {
		end = len(r.flattenedSegs)
	}
	return r.flattenedSegs[start:end]
}

// strokeDashedSubpaths applies dash pattern and strokes the resulting segments.
func (r *Rasteriser) strokeDashedSubpaths() {
	// Apply dash pattern - populates r.dashedSegs and r.dashedOffsets
	r.applyDashPattern()

	numDashes := len(r.dashedOffsets)
	for i := 0; i < numDashes; i++ {
		segs := r.getDashedSegments(i)

		// Handle dash-created zero-length segments (have orientation from underlying path)
		if len(segs) == 1 && segs[0].A == segs[0].B {
			seg := &segs[0]
			startOffset := len(r.stroke)
			switch r.Cap {
			case graphics.LineCapRound:
				r.addArc(seg.A, r.Width/2, vec.Vec2{X: 1, Y: 0}, 2*math.Pi, true)
				r.strokeOffsets = append(r.strokeOffsets, startOffset)
			case graphics.LineCapSquare:
				r.addSquare(seg.A, seg.T, r.Width/2)
				r.strokeOffsets = append(r.strokeOffsets, startOffset)
			}
			// Butt cap: no output
			continue
		}

		startOffset := len(r.stroke)
		r.strokeSubpath(segs, false) // dashed subpaths are never closed
		if len(r.stroke)-startOffset >= 3 {
			r.strokeOffsets = append(r.strokeOffsets, startOffset)
		} else {
			// Degenerate polygon, discard by resetting to start
			r.stroke = r.stroke[:startOffset]
		}
	}
}

// getDashedSegments returns the segments for dashed subpath i as a slice into dashedSegs.
func (r *Rasteriser) getDashedSegments(i int) []strokeSegment {
	start := r.dashedOffsets[i]
	var end int
	if i+1 < len(r.dashedOffsets) {
		end = r.dashedOffsets[i+1]
	} else {
		end = len(r.dashedSegs)
	}
	return r.dashedSegs[start:end]
}

// flattenPath walks the path, flattens curves, and populates the flattening
// buffers with precomputed segment geometry. Results are stored in:
//   - r.flattenedSegs: all segments from all subpaths, contiguous
//   - r.flattenedOffsets: start index of each subpath in flattenedSegs
//   - r.flattenedClosed: whether each subpath is closed
//   - r.degeneratePoints: degenerate subpaths (no orientation)
func (r *Rasteriser) flattenPath(p *path.Data) {
	// Clear buffers (preserving capacity)
	r.flattenedSegs = r.flattenedSegs[:0]
	r.flattenedOffsets = r.flattenedOffsets[:0]
	r.flattenedClosed = r.flattenedClosed[:0]
	r.degeneratePoints = r.degeneratePoints[:0]

	var currentPt vec.Vec2
	var subpathStartPt vec.Vec2
	subpathStartIdx := 0 // index into flattenedSegs where current subpath starts
	inSubpath := false
	sawDrawingCmd := false // tracks if we saw LineTo/QuadTo/CubeTo (for degenerate detection)

	// Walk the path using direct field access (no iterator allocation)
	coordIdx := 0
	for _, cmd := range p.Cmds {
		switch cmd {
		case path.CmdMoveTo:
			// Close previous subpath if needed
			if inSubpath && (len(r.flattenedSegs) > subpathStartIdx || sawDrawingCmd) {
				if len(r.flattenedSegs) == subpathStartIdx {
					// Degenerate subpath (no orientation) - collect for special handling
					r.degeneratePoints = append(r.degeneratePoints, subpathStartPt)
				} else {
					r.flattenedOffsets = append(r.flattenedOffsets, subpathStartIdx)
					r.flattenedClosed = append(r.flattenedClosed, false)
				}
			}
			currentPt = p.Coords[coordIdx]
			coordIdx++
			subpathStartPt = currentPt
			subpathStartIdx = len(r.flattenedSegs)
			inSubpath = true
			sawDrawingCmd = false

		case path.CmdLineTo:
			if !inSubpath {
				coordIdx++
				continue
			}
			sawDrawingCmd = true
			r.addFlattenedSegment(currentPt, p.Coords[coordIdx])
			currentPt = p.Coords[coordIdx]
			coordIdx++

		case path.CmdQuadTo:
			if !inSubpath {
				coordIdx += 2
				continue
			}
			sawDrawingCmd = true
			p0 := currentPt
			p1 := p.Coords[coordIdx]   // control point
			p2 := p.Coords[coordIdx+1] // endpoint
			r.flattenQuadratic(p0, p1, p2, r.addFlattenedSegment)
			currentPt = p2
			coordIdx += 2

		case path.CmdCubeTo:
			if !inSubpath {
				coordIdx += 3
				continue
			}
			sawDrawingCmd = true
			p0 := currentPt
			p1 := p.Coords[coordIdx]   // control point 1
			p2 := p.Coords[coordIdx+1] // control point 2
			p3 := p.Coords[coordIdx+2] // endpoint
			r.flattenCubic(p0, p1, p2, p3, r.addFlattenedSegment)
			currentPt = p3
			coordIdx += 3

		case path.CmdClose:
			if inSubpath {
				// Add closing segment if needed
				if currentPt != subpathStartPt {
					r.addFlattenedSegment(currentPt, subpathStartPt)
				}
				if len(r.flattenedSegs) == subpathStartIdx {
					// Degenerate closed subpath - collect for special handling
					r.degeneratePoints = append(r.degeneratePoints, subpathStartPt)
				} else {
					r.flattenedOffsets = append(r.flattenedOffsets, subpathStartIdx)
					r.flattenedClosed = append(r.flattenedClosed, true)
				}
				currentPt = subpathStartPt
				subpathStartIdx = len(r.flattenedSegs)
				inSubpath = false
				sawDrawingCmd = false
			}
		}
	}

	// Handle unclosed subpath at end
	if inSubpath && (len(r.flattenedSegs) > subpathStartIdx || sawDrawingCmd) {
		if len(r.flattenedSegs) == subpathStartIdx {
			// Degenerate subpath - collect for special handling
			r.degeneratePoints = append(r.degeneratePoints, subpathStartPt)
		} else {
			r.flattenedOffsets = append(r.flattenedOffsets, subpathStartIdx)
			r.flattenedClosed = append(r.flattenedClosed, false)
		}
	}
}

// addFlattenedSegment adds a line segment to the flattening buffer.
func (r *Rasteriser) addFlattenedSegment(a, b vec.Vec2) {
	d := b.Sub(a)
	length := d.Length()
	if length < zeroLengthThreshold {
		return // skip degenerate segment
	}
	t := d.Mul(1 / length)         // unit tangent
	n := vec.Vec2{X: -t.Y, Y: t.X} // unit normal (90° CCW)
	r.flattenedSegs = append(r.flattenedSegs, strokeSegment{A: a, B: b, T: t, N: n})
}

// strokeSubpath builds the stroke outline for a single subpath into r.stroke.
// The stroke outline is built as a closed polygon: forward pass on the +N side,
// then backward pass on the -N side. Join geometry is added on the outer side
// of each corner, which depends on the turn direction.
// Zero-length subpaths are handled by the caller before invoking this method.
func (r *Rasteriser) strokeSubpath(segs []strokeSegment, closed bool) {
	if len(segs) == 0 {
		return // empty, nothing to do
	}

	d := r.Width / 2 // half-width

	if closed {
		// Closed path: no caps, just joins
		// Build one continuous polygon: +N side forward, then -N side backward
		// The closing corner needs special handling to connect the two sides.

		first := &segs[0]
		last := &segs[len(segs)-1]

		// Forward pass: +N side (right side of path direction)
		// Start with the closing corner's +N point from segment 0's perspective
		r.stroke = append(r.stroke, first.A.Add(first.N.Mul(d)))
		for i := range len(segs) {
			seg := &segs[i]
			r.stroke = append(r.stroke, seg.B.Add(seg.N.Mul(d)))
			// Add join to next segment if outer side is +N
			if i < len(segs)-1 {
				next := &segs[i+1]
				r.addJoinIfOuter(seg.B, seg.T, next.T, d, true)
				// Add next segment's A point after join
				r.stroke = append(r.stroke, next.A.Add(next.N.Mul(d)))
			}
		}
		// Closing corner: join from last segment to first, then transition to -N side
		r.addJoinIfOuter(last.B, last.T, first.T, d, true)
		// Add first segment's A from +N side (same physical point, different offset)
		r.stroke = append(r.stroke, first.A.Add(first.N.Mul(d)))

		// Backward pass: -N side (left side of path direction)
		// Start with the closing corner's -N point from segment 0's perspective
		r.stroke = append(r.stroke, first.A.Sub(first.N.Mul(d)))
		// Add closing corner join on -N side
		r.addJoinIfOuter(first.A, last.T, first.T, d, false)
		// Continue with last segment's B from -N side
		r.stroke = append(r.stroke, last.B.Sub(last.N.Mul(d)))

		for i := len(segs) - 1; i >= 0; i-- {
			seg := &segs[i]
			r.stroke = append(r.stroke, seg.A.Sub(seg.N.Mul(d)))
			// Add join at this segment's A point (corner with previous segment)
			if i > 0 {
				prev := &segs[i-1]
				r.addJoinIfOuter(seg.A, prev.T, seg.T, d, false)
				r.stroke = append(r.stroke, prev.B.Sub(prev.N.Mul(d)))
			}
		}

	} else {
		// Open path: caps at ends, joins in between
		first := &segs[0]
		last := &segs[len(segs)-1]

		// Start cap (at first.A, direction = -T)
		r.addCap(first.A, first.T.Mul(-1), d)

		// Forward pass: +N side (right side of path direction)
		for i := range len(segs) {
			seg := &segs[i]
			r.stroke = append(r.stroke, seg.A.Add(seg.N.Mul(d)))
			r.stroke = append(r.stroke, seg.B.Add(seg.N.Mul(d)))
			if i < len(segs)-1 {
				r.addJoinIfOuter(seg.B, seg.T, segs[i+1].T, d, true)
			}
		}

		// End cap (at last.B, direction = T)
		r.addCap(last.B, last.T, d)

		// Backward pass: -N side (left side of path direction)
		for i := len(segs) - 1; i >= 0; i-- {
			seg := &segs[i]
			r.stroke = append(r.stroke, seg.B.Sub(seg.N.Mul(d)))
			r.stroke = append(r.stroke, seg.A.Sub(seg.N.Mul(d)))
			// Add join after this segment's A point (at the corner)
			if i > 0 {
				prev := &segs[i-1]
				r.addJoinIfOuter(seg.A, prev.T, seg.T, d, false)
			}
		}
	}
}

// addCap adds a line cap to the stroke outline at point P.
// T is the outward tangent direction (away from the line).
// d is half the stroke width.
func (r *Rasteriser) addCap(P, T vec.Vec2, d float64) {
	N := vec.Vec2{X: -T.Y, Y: T.X} // normal (90° CCW from T)

	switch r.Cap {
	case graphics.LineCapButt:
		// Butt cap: just connect left and right offset points (already done by caller)
		// No additional points needed

	case graphics.LineCapSquare:
		// Square cap: extend by d along tangent
		ext := P.Add(T.Mul(d))
		left := ext.Add(N.Mul(d))
		right := ext.Sub(N.Mul(d))
		r.stroke = append(r.stroke, left, right)

	case graphics.LineCapRound:
		// Round cap: semicircular arc curving outward (through T direction)
		// Arc starts at N direction and sweeps CW (negative angle) to reach -N,
		// passing through T (the outward direction)
		// includeStart=true because cap's start point is not yet in the polygon
		r.addArc(P, d, N, -math.Pi, true)
	}
}

// addJoinIfOuter adds a line join at point P only if we're on the outer side of the corner.
// isPositiveNormalSide indicates whether we're currently building the +N side (true) or -N side (false).
// Join geometry is only added when the outer side matches the current side.
func (r *Rasteriser) addJoinIfOuter(P, T1, T2 vec.Vec2, d float64, isPositiveNormalSide bool) {
	// Compute angle between tangents
	sinTheta := T1.X*T2.Y - T1.Y*T2.X // cross product Z component

	// Skip if nearly collinear
	if sinTheta > -collinearityThreshold && sinTheta < collinearityThreshold {
		return
	}

	// Determine which side is outer:
	// N = (-T.Y, T.X) points to the RIGHT of the walking direction in screen coords (Y down).
	// sinTheta > 0 means right turn (CW visually), outer side is LEFT (-N side)
	// sinTheta < 0 means left turn (CCW visually), outer side is RIGHT (+N side)
	outerIsLeft := sinTheta > 0

	// Only add join geometry if we're on the outer side
	// Forward pass (+N) is outer when outerIsLeft is false (left turn)
	// Backward pass (-N) is outer when outerIsLeft is true (right turn)
	if isPositiveNormalSide == outerIsLeft {
		return // inner side, skip join geometry
	}

	r.addJoin(P, T1, T2, d, isPositiveNormalSide)
}

// addJoin adds a line join at point P where tangent changes from T1 to T2.
// d is half the stroke width.
// isPositiveNormalSide indicates which side of the stroke we're building.
func (r *Rasteriser) addJoin(P, T1, T2 vec.Vec2, d float64, isPositiveNormalSide bool) {
	// Compute angle between tangents
	cosTheta := T1.Dot(T2)
	sinTheta := T1.X*T2.Y - T1.Y*T2.X // cross product Z component

	// Skip if nearly collinear
	if sinTheta > -collinearityThreshold && sinTheta < collinearityThreshold {
		return
	}

	// Check for cusp (path doubling back on itself)
	if cosTheta < cuspCosineThreshold {
		// Emit two caps instead of a join
		r.addCap(P, T1, d)
		r.addCap(P, T2.Mul(-1), d)
		return
	}

	// The join geometry extends in the direction of the current side we're building.
	// isPositiveNormalSide tells us which side: +N (true) or -N (false).

	switch r.Join {
	case graphics.LineJoinMiter:
		// Check miter limit: miterLength = 1 / sin(φ/2)
		// where φ is the visual angle at the corner (interior angle of the stroke).
		// If θ is the angle between tangents (cosTheta = T1·T2), then φ = 180° - θ.
		// sin(φ/2) = sin(90° − θ/2) = cos(θ/2) = sqrt((1 + cosθ) / 2)
		sinHalf := math.Sqrt((1 + cosTheta) / 2)
		// Use small tolerance for boundary cases (floating-point precision)
		const miterEpsilon = 1e-10
		if sinHalf > 0 && 1/sinHalf <= r.MiterLimit+miterEpsilon {
			// Miter join: compute miter point
			// The miter point is where the two offset lines intersect
			// Distance from P to miter point = d / sin(φ/2) = d / sinHalf
			N1 := vec.Vec2{X: -T1.Y, Y: T1.X}
			N2 := vec.Vec2{X: -T2.Y, Y: T2.X}

			// Bisector direction depends on which side we're building
			var bisector vec.Vec2
			if isPositiveNormalSide {
				bisector = N1.Add(N2) // +N side
			} else {
				bisector = N1.Add(N2).Mul(-1) // -N side
			}
			bisectorLen := bisector.Length()
			if bisectorLen > zeroLengthThreshold {
				bisector = bisector.Mul(1 / bisectorLen)
				// Distance to miter point = d / sinHalf
				miterDist := d / sinHalf
				miterPt := P.Add(bisector.Mul(miterDist))
				r.stroke = append(r.stroke, miterPt)
			}
			return
		}
		// Fall through to bevel if miter limit exceeded
		fallthrough

	case graphics.LineJoinBevel:
		// Bevel join: just let the two offset lines meet (no additional points)
		// The caller already adds the necessary points
		return

	case graphics.LineJoinRound:
		// Round join: arc curving outward on the current side
		// includeStart=false because join's start point is already in the polygon
		angle := math.Acos(max(-1, min(1, cosTheta)))
		if isPositiveNormalSide {
			// Forward pass: arc from +N of T1 to +N of T2
			N1 := vec.Vec2{X: -T1.Y, Y: T1.X} // +N direction of T1
			// For +N side: right turn needs CCW arc, left turn needs CW arc
			if sinTheta > 0 {
				r.addArc(P, d, N1, angle, false)
			} else {
				r.addArc(P, d, N1, -angle, false)
			}
		} else {
			// Backward pass: we just added offset using T2's normal, so arc must
			// start from -N of T2 and go to -N of T1 (reversed direction)
			N2 := vec.Vec2{X: T2.Y, Y: -T2.X} // -N direction of T2
			// Sweep direction is reversed from forward pass
			if sinTheta > 0 {
				r.addArc(P, d, N2, -angle, false)
			} else {
				r.addArc(P, d, N2, angle, false)
			}
		}
	}
}

// addArc adds arc vertices to the stroke outline.
// center is the arc center, radius is the arc radius.
// startDir is the unit vector from center to arc start.
// sweep is the sweep angle in radians (positive = CCW).
// includeStart indicates whether to include the start point (false if caller already added it).
func (r *Rasteriser) addArc(center vec.Vec2, radius float64, startDir vec.Vec2, sweep float64, includeStart bool) {
	// Compute number of segments based on flatness tolerance
	// Using device-space radius for segment count
	devRadius := r.transformLinear(vec.Vec2{X: radius, Y: 0}).Length()
	devRadius2 := r.transformLinear(vec.Vec2{X: 0, Y: radius}).Length()
	devRadius = max(devRadius, devRadius2)

	if devRadius < r.Flatness {
		// Arc too small to matter - just add end point (and start if needed)
		if includeStart {
			r.stroke = append(r.stroke, center.Add(startDir.Mul(radius)))
		}
		cos, sin := math.Cos(sweep), math.Sin(sweep)
		endDir := vec.Vec2{
			X: startDir.X*cos - startDir.Y*sin,
			Y: startDir.X*sin + startDir.Y*cos,
		}
		r.stroke = append(r.stroke, center.Add(endDir.Mul(radius)))
		return
	}

	// n = ceil(|sweep| / acos(1 - flatness/devRadius))
	absSweep := math.Abs(sweep)

	angleStep := math.Acos(1 - r.Flatness/devRadius)
	if angleStep <= 0 || math.IsNaN(angleStep) {
		angleStep = math.Pi / 4 // fallback
	}
	n := int(math.Ceil(absSweep / angleStep))
	n = max(n, 1)

	// Generate arc points
	dt := sweep / float64(n)
	startI := 0
	if !includeStart {
		startI = 1 // skip start point if caller already added it
	}
	for i := startI; i <= n; i++ {
		angle := float64(i) * dt
		// Rotate startDir by angle
		cos, sin := math.Cos(angle), math.Sin(angle)
		dir := vec.Vec2{
			X: startDir.X*cos - startDir.Y*sin,
			Y: startDir.X*sin + startDir.Y*cos,
		}
		pt := center.Add(dir.Mul(radius))
		r.stroke = append(r.stroke, pt)
	}
}

// addSquare adds a filled square to the stroke outline for a zero-length
// dash segment with square caps. The square is centered at the point with
// side length = 2*d (i.e., the line width), oriented by the tangent T.
func (r *Rasteriser) addSquare(center vec.Vec2, T vec.Vec2, d float64) {
	N := vec.Vec2{X: -T.Y, Y: T.X} // normal (90° CCW from T)
	// Four corners of the square
	r.stroke = append(r.stroke,
		center.Add(T.Mul(d)).Add(N.Mul(d)),
		center.Add(T.Mul(d)).Sub(N.Mul(d)),
		center.Sub(T.Mul(d)).Sub(N.Mul(d)),
		center.Sub(T.Mul(d)).Add(N.Mul(d)),
	)
}

// applyDashPattern applies the dash pattern to flattened subpaths.
// Results are stored in r.dashedSegs and r.dashedOffsets.
func (r *Rasteriser) applyDashPattern() {
	// Clear output buffers (preserving capacity)
	r.dashedSegs = r.dashedSegs[:0]
	r.dashedOffsets = r.dashedOffsets[:0]

	// Normalize dash pattern (odd length -> double it)
	dash := r.Dash
	if len(dash)%2 == 1 {
		dash = append(dash, dash...)
	}

	// Compute total pattern length
	patternLen := 0.0
	for _, d := range dash {
		patternLen += d
	}
	if patternLen <= 0 {
		return // no dashing
	}

	// Normalize phase to [0, patternLen)
	phase := r.DashPhase
	phase = math.Mod(phase, patternLen)
	if phase < 0 {
		phase += patternLen
	}

	numSubpaths := len(r.flattenedOffsets)
	for spIdx := 0; spIdx < numSubpaths; spIdx++ {
		segments := r.getSubpathSegments(spIdx)
		closed := r.flattenedClosed[spIdx]
		if len(segments) == 0 {
			continue
		}

		// Find starting dash index and remaining distance in that dash
		dashIdx := 0
		dist := phase
		for dist >= dash[dashIdx] && dash[dashIdx] > 0 {
			dist -= dash[dashIdx]
			dashIdx = (dashIdx + 1) % len(dash)
		}
		remaining := dash[dashIdx] - dist
		isOn := dashIdx%2 == 0 // even indices are "on"

		// Handle zero-length dash at the very start of the path.
		// This emits a point that will become a dot with round/square caps.
		if isOn && remaining == 0 && len(segments) > 0 {
			seg := segments[0]
			r.dashedOffsets = append(r.dashedOffsets, len(r.dashedSegs))
			r.dashedSegs = append(r.dashedSegs, strokeSegment{A: seg.A, B: seg.A, T: seg.T, N: seg.N})
			// Advance to next dash element
			dashIdx = (dashIdx + 1) % len(dash)
			remaining = dash[dashIdx]
			isOn = dashIdx%2 == 0
		}

		// Track if we started with "on" for closed path joining
		startedOn := isOn
		firstDashStart := -1 // index into dashedSegs where first dash starts
		firstDashEnd := -1   // index into dashedSegs where first dash ends

		// Walk segments and split at dash boundaries
		dashStartIdx := len(r.dashedSegs) // start of current dash in dashedSegs
		segIdx := 0
		segDist := 0.0 // distance along current segment

		for segIdx < len(segments) {
			seg := segments[segIdx]
			segLen := seg.B.Sub(seg.A).Length()
			segRemaining := segLen - segDist

			if remaining >= segRemaining {
				// Dash continues past this segment
				if isOn {
					// Add portion of segment from segDist to end
					if segDist > 0 {
						t := segDist / segLen
						startPt := seg.A.Add(seg.B.Sub(seg.A).Mul(t))
						r.dashedSegs = append(r.dashedSegs, strokeSegment{
							A: startPt, B: seg.B,
							T: seg.T, N: seg.N,
						})
					} else {
						r.dashedSegs = append(r.dashedSegs, seg)
					}
				}
				remaining -= segRemaining
				segIdx++
				segDist = 0
			} else {
				// Dash ends within this segment
				endDist := segDist + remaining
				t := endDist / segLen
				splitPt := seg.A.Add(seg.B.Sub(seg.A).Mul(t))

				if isOn {
					// Add portion from segDist to splitPt
					startT := segDist / segLen
					startPt := seg.A.Add(seg.B.Sub(seg.A).Mul(startT))
					d := splitPt.Sub(startPt)
					dLen := d.Length()
					if dLen > zeroLengthThreshold {
						tVec := d.Mul(1 / dLen)
						nVec := vec.Vec2{X: -tVec.Y, Y: tVec.X}
						r.dashedSegs = append(r.dashedSegs, strokeSegment{
							A: startPt, B: splitPt,
							T: tVec, N: nVec,
						})
					} else if len(r.dashedSegs) == dashStartIdx {
						// Zero-length dash: emit point with tangent from underlying segment
						// This allows square/round caps to be drawn at this point
						r.dashedSegs = append(r.dashedSegs, strokeSegment{
							A: startPt, B: startPt,
							T: seg.T, N: seg.N,
						})
					}

					// Save first dash indices for closed path joining
					if firstDashStart < 0 && len(r.dashedSegs) > dashStartIdx {
						firstDashStart = dashStartIdx
						firstDashEnd = len(r.dashedSegs)
					}

					// Emit current dash if non-empty
					if len(r.dashedSegs) > dashStartIdx {
						r.dashedOffsets = append(r.dashedOffsets, dashStartIdx)
						dashStartIdx = len(r.dashedSegs)
					}
				}

				// Move to next dash
				segDist = endDist
				dashIdx = (dashIdx + 1) % len(dash)
				remaining = dash[dashIdx]
				isOn = dashIdx%2 == 0
			}
		}

		// Emit final dash if any
		if len(r.dashedSegs) > dashStartIdx {
			// For closed paths, check if we should join first and last dash
			if closed && startedOn && isOn && firstDashStart >= 0 {
				// Merge: append first dash segments to current dash
				for i := firstDashStart; i < firstDashEnd; i++ {
					r.dashedSegs = append(r.dashedSegs, r.dashedSegs[i])
				}
				// Remove the first dash from offsets if we added it
				if len(r.dashedOffsets) > 0 && r.dashedOffsets[0] == firstDashStart {
					r.dashedOffsets = r.dashedOffsets[1:]
				}
			}
			r.dashedOffsets = append(r.dashedOffsets, dashStartIdx)
		}
	}
}

// fillStrokeOutlines fills all collected stroke polygons as a compound path.
// Using nonzero winding rule ensures overlapping regions are painted once.
func (r *Rasteriser) fillStrokeOutlines(emit func(y, xMin int, coverage []float32)) {
	if len(r.strokeOffsets) == 0 {
		return
	}

	// Collect edges directly from stroke polygons (no intermediate path allocation)
	xMin, xMax, yMin, yMax, ok := r.collectStrokeEdges()
	if !ok {
		return
	}

	// Choose approach based on bounding box size
	width := xMax - xMin
	height := yMax - yMin

	if width*height < r.smallPathThreshold {
		r.fill2D(xMin, xMax, yMin, yMax, fillNonZero, emit)
	} else {
		r.fillScanline(xMin, xMax, yMin, yMax, fillNonZero, emit)
	}
}

// collectStrokeEdges builds the edge list directly from stroke polygons.
// This avoids creating an intermediate path representation.
func (r *Rasteriser) collectStrokeEdges() (xMin, xMax, yMin, yMax int, ok bool) {
	r.edges = r.edges[:0]
	r.edgeBBoxFirst = true

	for i, start := range r.strokeOffsets {
		// Determine end of this polygon
		var end int
		if i+1 < len(r.strokeOffsets) {
			end = r.strokeOffsets[i+1]
		} else {
			end = len(r.stroke)
		}
		poly := r.stroke[start:end]
		if len(poly) < 2 {
			continue
		}

		// Add edges for each segment
		for j := 1; j < len(poly); j++ {
			r.addEdge(poly[j-1], poly[j])
		}
		// Close the polygon
		r.addEdge(poly[len(poly)-1], poly[0])
	}

	if len(r.edges) == 0 {
		return 0, 0, 0, 0, false
	}

	// Clamp to clip bounds and convert to integers
	clipXMin := int(r.Clip.LLx)
	clipXMax := int(r.Clip.URx)
	clipYMin := int(r.Clip.LLy)
	clipYMax := int(r.Clip.URy)

	xMin = max(int(math.Floor(r.edgeDevXMin)), clipXMin)
	xMax = min(int(math.Floor(r.edgeDevXMax))+1, clipXMax)
	yMin = max(int(math.Floor(r.edgeDevYMin)), clipYMin)
	yMax = min(int(math.Floor(r.edgeDevYMax))+1, clipYMax)

	if xMin >= xMax || yMin >= yMax {
		return 0, 0, 0, 0, false
	}

	return xMin, xMax, yMin, yMax, true
}

// Reset resets the Rasteriser to its initial state with the given clip rectangle,
// preserving internal buffer capacity for reuse. This is equivalent to creating
// a new Rasteriser but without allocations if buffers are already large enough.
func (r *Rasteriser) Reset(clip rect.Rect) {
	// Reset public state to defaults
	r.CTM = matrix.Identity
	r.Clip = clip
	r.Flatness = DefaultFlatness
	r.Width = 1.0
	r.Cap = graphics.LineCapButt
	r.Join = graphics.LineJoinMiter
	r.MiterLimit = DefaultMiterLimit
	r.Dash = nil
	r.DashPhase = 0

	// Preserve buffer capacity by slicing to zero length
	r.cover = r.cover[:0]
	r.area = r.area[:0]
	r.output = r.output[:0]
	r.edges = r.edges[:0]
	r.active = r.active[:0]
	r.xMin = r.xMin[:0]
	r.xMax = r.xMax[:0]
	r.stroke = r.stroke[:0]
	r.strokeOffsets = r.strokeOffsets[:0]
	r.crossings = r.crossings[:0]
	r.flattenedSegs = r.flattenedSegs[:0]
	r.flattenedOffsets = r.flattenedOffsets[:0]
	r.flattenedClosed = r.flattenedClosed[:0]
	r.degeneratePoints = r.degeneratePoints[:0]
	r.dashedSegs = r.dashedSegs[:0]
	r.dashedOffsets = r.dashedOffsets[:0]
}
