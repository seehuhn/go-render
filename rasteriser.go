// seehuhn.de/go/raster - a 2D rendering library
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
	"cmp"
	"math"
	"slices"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/pdf/graphics"
)

// edge represents a line segment in device coordinates.
type edge struct {
	x0, y0 float64 // start point
	x1, y1 float64 // end point
	dxdy   float64 // (x1-x0)/(y1-y0), precomputed for x-intercept calculation
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
	cover         []float32  // coverage accumulation: cover change per pixel; reused as output
	area          []float32  // coverage accumulation: area within pixel
	edges         []edge     // edge list for current path (device coordinates)
	activeIdx     []int      // indices of active edges
	rowXMin       []int      // per-scanline minimum x with edge contribution
	rowXMax       []int      // per-scanline maximum x with edge contribution
	stroke        []vec.Vec2 // stroke outline vertices (all subpaths contiguous)
	strokeOffsets []int      // start index of each stroke polygon in stroke[]
	crossings     []float64  // y values where edge crosses pixel boundaries

	// Flattening buffers (for stroke path processing)
	segs             []strokeSegment // all segments from all subpaths, contiguous
	segsOffsets      []int           // start index of each subpath in segments
	subpathClosed    []bool          // whether each subpath is closed
	degeneratePoints []vec.Vec2      // degenerate subpaths (no orientation)

	// Edge collection state (used by collectEdges/addEdge)
	edgeBBoxFirst bool    // true if no edges added yet
	edgeDevXMin   float64 // bounding box in device space
	edgeDevXMax   float64
	edgeDevYMin   float64
	edgeDevYMax   float64

	// Dash pattern output buffers
	dashedSegs        []strokeSegment // all dashed segments, contiguous
	dashedSegsOffsets []int           // start index of each dashed subpath
}

// NewRasteriser creates a new Rasteriser with the given clip rectangle
// and PDF default values for all other parameters.
func NewRasteriser(clip rect.Rect) *Rasteriser {
	return &Rasteriser{
		CTM:        matrix.Identity,
		Clip:       clip,
		Flatness:   defaultFlatness,
		Width:      1.0,
		Cap:        graphics.LineCapButt,
		Join:       graphics.LineJoinMiter,
		MiterLimit: defaultMiterLimit,

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
	xMin, xMax, yMin, yMax, ok := r.collectPathEdges(p)
	if !ok {
		return // empty or degenerate path
	}

	// Choose approach based on bounding box size
	width := xMax - xMin
	height := yMax - yMin

	if width*height < r.smallPathThreshold {
		r.fillSmallPath(xMin, xMax, yMin, yMax, rule, emit)
	} else {
		r.fillLargePath(xMin, xMax, yMin, yMax, rule, emit)
	}
}

// collectPathEdges walks the path, transforms to device space, and builds the edge list.
// Returns the bounding box of all edges in device coordinates (clamped to clip).
func (r *Rasteriser) collectPathEdges(p *path.Data) (xMin, xMax, yMin, yMax int, ok bool) {
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

// Coverage accumulation model:
//
// For each pixel, we track two values:
//   cover: signed vertical extent of edges crossing this pixel column
//   area:  horizontal position weighting (how far right the crossing is)
//
// An edge crossing a pixel contributes:
//   cover = sign * dy   (where sign is +1 for downward, -1 for upward)
//   area  = cover * (1 - xFrac)   (where xFrac is the horizontal position within the pixel)
//
// Final coverage is computed by integrateScanline:
//   pixel_coverage = accumulated_cover + area[i]
//   accumulated_cover += cover[i]   (carry forward for next pixel)
//
// This computes the signed area of the path within each pixel, which gives
// anti-aliased coverage values when clamped to [0,1] (nonzero) or folded (even-odd).

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
		r.accumulateEdgeInColumn(e, yTop, yBot, sign, pixLeft, cover, area, bboxXMin, bboxXMax)
		return
	}

	// Edge spans multiple pixels - split at each pixel boundary
	// Collect all y values where the edge crosses integer x boundaries
	// Then process each segment between consecutive crossings
	dydx := 1 / e.dxdy

	// Build list of y values where edge crosses pixel boundaries (reuse buffer)
	r.crossings = r.crossings[:0]

	// Add start and end y values
	r.crossings = append(r.crossings, yTop, yBot)

	// Add y values at integer x boundaries
	for x := pixLeft + 1; x <= pixRight; x++ {
		yAtX := e.y0 + dydx*(float64(x)-e.x0)
		if yAtX > yTop && yAtX < yBot {
			r.crossings = append(r.crossings, yAtX)
		}
	}

	// Sort crossings by y
	slices.Sort(r.crossings)

	// Process each segment between consecutive crossings
	for i := range len(r.crossings) - 1 {
		y0 := r.crossings[i]
		y1 := r.crossings[i+1]
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

// accumulateEdgeInColumn handles an edge segment that falls within a single pixel column.
func (r *Rasteriser) accumulateEdgeInColumn(e *edge, yTop, yBot float64, sign float32, pix int, cover, area []float32, bboxXMin, bboxXMax int) {
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

// integrateScanlineNonZero converts accumulated cover/area to final coverage
// values using the nonzero winding rule. The cover slice is modified in place.
func integrateScanlineNonZero(cover, area []float32) {
	var accum float32
	for i := range cover {
		raw := accum + area[i]
		accum += cover[i]

		// clamp(abs(raw), 0, 1)
		cov := raw
		if raw < 0 {
			cov = -raw
		}
		if cov > 1 {
			cov = 1
		}
		cover[i] = cov
	}
}

// integrateScanlineEvenOdd converts accumulated cover/area to final coverage
// values using the even-odd fill rule. The cover slice is modified in place.
func integrateScanlineEvenOdd(cover, area []float32) {
	var accum float32
	for i := range cover {
		raw := accum + area[i]
		accum += cover[i]

		// 1 - abs(1 - mod(abs(raw), 2))
		if raw < 0 {
			raw = -raw
		}
		// mod(raw, 2) using floor
		mod := raw - 2*float32(int(raw/2))
		cov := 1 - abs32(1-mod)
		cover[i] = cov
	}
}

// abs32 returns the absolute value of a float32.
func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// trimZeros returns the non-zero portion of coverage and its starting offset.
// Returns nil, 0 if coverage is entirely zero.
func trimZeros(coverage []float32) (trimmed []float32, offset int) {
	n := len(coverage)
	lo := 0
	for lo < n && coverage[lo] == 0 {
		lo++
	}
	if lo == n {
		return nil, 0
	}
	hi := n - 1
	for hi > lo && coverage[hi] == 0 {
		hi--
	}
	return coverage[lo : hi+1], lo
}

// fillSmallPath rasterises using 2D buffers (Approach A).
// Used for small paths where width*height < smallPathThreshold.
// xMin, xMax, yMin, yMax define the path's bounding box (already clamped to clip).
func (r *Rasteriser) fillSmallPath(xMin, xMax, yMin, yMax int, rule fillRule, emit func(y, xMin int, coverage []float32)) {
	width := xMax - xMin
	height := yMax - yMin

	// Ensure 2D buffers are large enough and zero them
	size := width * height
	r.cover = slices.Grow(r.cover[:0], size)[:size]
	r.area = slices.Grow(r.area[:0], size)[:size]
	clear(r.cover)
	clear(r.area)

	// Ensure xMin/xMax tracking buffers are large enough
	r.rowXMin = slices.Grow(r.rowXMin[:0], height)[:height]
	r.rowXMax = slices.Grow(r.rowXMax[:0], height)[:height]
	// Initialize bounds to "no edges"
	for i := range r.rowXMin {
		r.rowXMin[i] = width
		r.rowXMax[i] = -1
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
			if xIdx < r.rowXMin[row] {
				r.rowXMin[row] = xIdx
			}
			if xIdx > r.rowXMax[row] {
				r.rowXMax[row] = xIdx
			}
		}
	}

	// Integrate and emit each row
	for row := range height {
		if r.rowXMax[row] < 0 {
			continue // no edges touched this row
		}

		y := yMin + row
		rowOffset := row * width

		// Integrate the full width (cover accumulates from left)
		coverage := r.cover[rowOffset : rowOffset+width]
		if rule == fillNonZero {
			integrateScanlineNonZero(coverage, r.area[rowOffset:rowOffset+width])
		} else {
			integrateScanlineEvenOdd(coverage, r.area[rowOffset:rowOffset+width])
		}

		// Emit only the non-zero portion
		if trimmed, offset := trimZeros(coverage); trimmed != nil {
			emit(y, xMin+offset, trimmed)
		}
	}
}

// fillLargePath rasterises using 1D buffers and an active edge list (Approach B).
// Used for large paths where width*height >= smallPathThreshold.
// xMin, xMax, yMin, yMax define the path's bounding box (already clamped to clip).
func (r *Rasteriser) fillLargePath(xMin, xMax, yMin, yMax int, rule fillRule, emit func(y, xMin int, coverage []float32)) {
	width := xMax - xMin

	// Ensure 1D buffers are large enough
	r.cover = slices.Grow(r.cover[:0], width)[:width]
	r.area = slices.Grow(r.area[:0], width)[:width]

	// Sort edges by y_min
	slices.SortFunc(r.edges, func(a, b edge) int {
		aYMin := min(a.y0, a.y1)
		bYMin := min(b.y0, b.y1)
		return cmp.Compare(aYMin, bYMin)
	})

	// Active edge list (indices into r.edges)
	r.activeIdx = r.activeIdx[:0]
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
			r.activeIdx = append(r.activeIdx, nextEdge)
			nextEdge++
		}

		if len(r.activeIdx) == 0 {
			continue
		}

		// Clear buffers for this scanline
		clear(r.cover)
		clear(r.area)

		// Track x bounds for this scanline
		xMinBound := width
		xMaxBound := -1

		// Process active edges
		for i := 0; i < len(r.activeIdx); {
			e := &r.edges[r.activeIdx[i]]

			// Check if edge ends before this scanline
			edgeYMax := max(e.y0, e.y1)
			if edgeYMax <= yf {
				// Remove from active list (swap with last)
				r.activeIdx[i] = r.activeIdx[len(r.activeIdx)-1]
				r.activeIdx = r.activeIdx[:len(r.activeIdx)-1]
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
		if rule == fillNonZero {
			integrateScanlineNonZero(r.cover, r.area)
		} else {
			integrateScanlineEvenOdd(r.cover, r.area)
		}

		// Emit only the non-zero portion
		if trimmed, offset := trimZeros(r.cover); trimmed != nil {
			emit(y, xMin+offset, trimmed)
		}
	}
}

// Reset resets the Rasteriser to its initial state with the given clip rectangle,
// preserving internal buffer capacity for reuse. This is equivalent to creating
// a new Rasteriser but without allocations if buffers are already large enough.
func (r *Rasteriser) Reset(clip rect.Rect) {
	// Reset public state to defaults
	r.CTM = matrix.Identity
	r.Clip = clip
	r.Flatness = defaultFlatness
	r.Width = 1.0
	r.Cap = graphics.LineCapButt
	r.Join = graphics.LineJoinMiter
	r.MiterLimit = defaultMiterLimit
	r.Dash = nil
	r.DashPhase = 0

	// Preserve buffer capacity by slicing to zero length
	r.cover = r.cover[:0]
	r.area = r.area[:0]
	r.edges = r.edges[:0]
	r.activeIdx = r.activeIdx[:0]
	r.rowXMin = r.rowXMin[:0]
	r.rowXMax = r.rowXMax[:0]
	r.stroke = r.stroke[:0]
	r.strokeOffsets = r.strokeOffsets[:0]
	r.crossings = r.crossings[:0]
	r.segs = r.segs[:0]
	r.segsOffsets = r.segsOffsets[:0]
	r.subpathClosed = r.subpathClosed[:0]
	r.degeneratePoints = r.degeneratePoints[:0]
	r.dashedSegs = r.dashedSegs[:0]
	r.dashedSegsOffsets = r.dashedSegsOffsets[:0]
}

// Default values for rasteriser parameters.
const (
	// defaultFlatness is the default curve flattening tolerance in device
	// pixels. Values of 0.25-1.0 are typical; 0.25 is below the threshold
	// of visual perception.
	defaultFlatness = 0.25

	// defaultMiterLimit is the default miter limit, matching PDF/PostScript.
	// This converts joins to bevels when the interior angle is less than
	// approximately 11.5 degrees.
	defaultMiterLimit = 10.0
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
