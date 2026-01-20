# Fill Rasteriser Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement FillNonZero and FillEvenOdd methods using signed-area coverage accumulation with hybrid 2D/active-edge-list buffer management.

**Architecture:** Path is traversed to build an edge list (with dummy curve flattening for now). Based on bounding box size, either 2D buffers (small paths) or active edge list (large paths) is used. Edges contribute cover/area values which are integrated row-by-row to produce final coverage.

**Tech Stack:** Go, seehuhn.de/go/geom (path, vec, matrix, rect), seehuhn.de/go/pdf/graphics

---

## Task 1: Add Constants and Edge Type

**Files:**
- Modify: `rasteriser.go`

**Step 1: Add the constants block after existing constants**

Add after line 38 (after `DefaultMiterLimit`):

```go
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
)

// edge represents a line segment in device coordinates.
type edge struct {
	x0, y0 float64 // start point
	x1, y1 float64 // end point
	dxdy   float64 // (x1-x0)/(y1-y0), precomputed for x-intercept calculation
}
```

**Step 2: Verify the code compiles**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add rasteriser.go
git commit -m "add edge type and numerical constants for rasteriser"
```

---

## Task 2: Add Internal Buffers to Rasteriser

**Files:**
- Modify: `rasteriser.go`

**Step 1: Update the internal buffers section**

Replace lines 78-82 (the internal buffers comment and fields) with:

```go
	// Internal buffers (reused across calls)
	cover  []float32  // coverage accumulation: cover change per pixel
	area   []float32  // coverage accumulation: area within pixel
	output []float32  // final coverage values for callback
	edges  []edge     // edge list for current path (device coordinates)
	active []int      // indices of active edges (for Approach B)
	xMin   []int      // per-scanline minimum x with edge contribution
	xMax   []int      // per-scanline maximum x with edge contribution
	stroke []vec.Vec2 // stroke outline vertices
```

**Step 2: Update the Reset method**

Replace the Reset method body (lines 128-133) with:

```go
func (r *Rasteriser) Reset() {
	r.cover = nil
	r.area = nil
	r.output = nil
	r.edges = nil
	r.active = nil
	r.xMin = nil
	r.xMax = nil
	r.stroke = nil
}
```

**Step 3: Verify the code compiles**

Run: `go build ./...`
Expected: No errors

**Step 4: Commit**

```bash
git add rasteriser.go
git commit -m "add edge list and bounds tracking buffers to Rasteriser"
```

---

## Task 3: Implement Path Traversal and Edge Collection

**Files:**
- Modify: `rasteriser.go`

**Step 1: Add "math" to the import block**

```go
import (
	"math"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/pdf/graphics"
)
```

**Step 2: Add the collectEdges method**

Add after the Reset method:

```go
// collectEdges walks the path, transforms to device space, and builds the edge list.
// Curves are approximated with a single line segment (TODO: implement proper flattening).
// Returns the bounding box of all edges in device coordinates (clamped to clip).
func (r *Rasteriser) collectEdges(p path.Path) (xMin, xMax, yMin, yMax int, ok bool) {
	r.edges = r.edges[:0]

	// Path state
	var currentX, currentY float64       // current point (user space)
	var subpathX, subpathY float64       // subpath start (user space)
	var devXMin, devXMax float64         // bounding box (device space)
	var devYMin, devYMax float64
	first := true

	// Helper to add an edge (user space coords, will be transformed)
	addEdge := func(x0, y0, x1, y1 float64) {
		// Transform to device space
		dx0 := r.CTM[0]*x0 + r.CTM[2]*y0 + r.CTM[4]
		dy0 := r.CTM[1]*x0 + r.CTM[3]*y0 + r.CTM[5]
		dx1 := r.CTM[0]*x1 + r.CTM[2]*y1 + r.CTM[4]
		dy1 := r.CTM[1]*x1 + r.CTM[3]*y1 + r.CTM[5]

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
		if first {
			devXMin = min(dx0, dx1)
			devXMax = max(dx0, dx1)
			devYMin = min(dy0, dy1)
			devYMax = max(dy0, dy1)
			first = false
		} else {
			devXMin = min(devXMin, min(dx0, dx1))
			devXMax = max(devXMax, max(dx0, dx1))
			devYMin = min(devYMin, min(dy0, dy1))
			devYMax = max(devYMax, max(dy0, dy1))
		}
	}

	// Walk the path
	for cmd, pts := range p {
		switch cmd {
		case path.CmdMoveTo:
			currentX, currentY = pts[0].X, pts[0].Y
			subpathX, subpathY = currentX, currentY

		case path.CmdLineTo:
			addEdge(currentX, currentY, pts[0].X, pts[0].Y)
			currentX, currentY = pts[0].X, pts[0].Y

		case path.CmdQuadTo:
			// TODO: implement proper curve flattening
			// For now, just draw a line to the endpoint
			endX, endY := pts[1].X, pts[1].Y
			addEdge(currentX, currentY, endX, endY)
			currentX, currentY = endX, endY

		case path.CmdCubeTo:
			// TODO: implement proper curve flattening
			// For now, just draw a line to the endpoint
			endX, endY := pts[2].X, pts[2].Y
			addEdge(currentX, currentY, endX, endY)
			currentX, currentY = endX, endY

		case path.CmdClose:
			if currentX != subpathX || currentY != subpathY {
				addEdge(currentX, currentY, subpathX, subpathY)
			}
			currentX, currentY = subpathX, subpathY
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

	xMin = max(int(math.Floor(devXMin)), clipXMin)
	xMax = min(int(math.Floor(devXMax))+1, clipXMax)
	yMin = max(int(math.Floor(devYMin)), clipYMin)
	yMax = min(int(math.Floor(devYMax))+1, clipYMax)

	if xMin >= xMax || yMin >= yMax {
		return 0, 0, 0, 0, false
	}

	return xMin, xMax, yMin, yMax, true
}
```

**Step 2: Verify the code compiles**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add rasteriser.go
git commit -m "implement path traversal and edge collection with dummy curve flattening"
```

---

## Task 4: Implement Edge Contribution Calculation

**Files:**
- Modify: `rasteriser.go`

**Step 1: Add the accumulateEdge method**

Add after collectEdges:

```go
// accumulateEdge adds a single edge's contribution to the cover and area buffers.
// The buffers are indexed by (x - bboxXMin), where bboxXMin/bboxXMax define the buffer range.
// The edge's y range should already be clamped to the current scanline.
func (r *Rasteriser) accumulateEdge(e *edge, y int, cover, area []float32, bboxXMin, bboxXMax int) {
	// Compute the portion of the edge within this scanline [y, y+1)
	yTop := float64(y)
	yBot := float64(y + 1)

	// Clamp to edge's actual y extent
	if e.y0 < e.y1 {
		yTop = max(yTop, e.y0)
		yBot = min(yBot, e.y1)
	} else {
		yTop = max(yTop, e.y1)
		yBot = min(yBot, e.y0)
	}

	dy := yBot - yTop
	if dy <= 0 {
		return
	}

	// Cover contribution: signed vertical extent
	var coverVal float32
	if e.y1 > e.y0 {
		coverVal = float32(dy) // downward edge: positive
	} else {
		coverVal = float32(-dy) // upward edge: negative
	}

	// Compute x at the midpoint of the edge segment within this scanline
	yMid := (yTop + yBot) / 2
	xMid := e.x0 + e.dxdy*(yMid-e.y0)

	// Determine which pixel column this falls into (use floor for correct negative handling)
	x := int(math.Floor(xMid))
	if x < bboxXMin {
		// Edge is to the left of bounding box
		// Still contributes to cover for all pixels to the right
		cover[0] += coverVal
		return
	}
	if x >= bboxXMax {
		// Edge is to the right of bounding box - no contribution
		return
	}

	// Area contribution: accounts for horizontal position within pixel
	xFrac := xMid - float64(x) // fractional x position within pixel [0, 1)
	areaVal := coverVal * float32(1.0-xFrac)

	idx := x - bboxXMin
	cover[idx] += coverVal
	area[idx] += areaVal
}
```

**Step 2: Verify the code compiles**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add rasteriser.go
git commit -m "implement edge contribution calculation for coverage accumulation"
```

---

## Task 5: Implement Integration and Fill Rules

**Files:**
- Modify: `rasteriser.go`

**Step 1: Add fill rule type and integration method**

Add after accumulateEdge:

```go
// fillRule identifies which fill rule to apply.
type fillRule int

const (
	fillNonZero fillRule = iota
	fillEvenOdd
)

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
	for i := 0; i < width; i++ {
		accum += cover[i]
		raw := accum + area[i]

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
```

**Step 2: Verify the code compiles**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add rasteriser.go
git commit -m "implement scanline integration with nonzero and even-odd fill rules"
```

---

## Task 6: Implement Approach A (2D Buffers)

**Files:**
- Modify: `rasteriser.go`

**Step 1: Add the fill2D method**

Add after integrateScanline:

```go
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
	for row := 0; row < height; row++ {
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
```

**Step 2: Verify the code compiles**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add rasteriser.go
git commit -m "implement 2D buffer approach for small path rasterisation"
```

---

## Task 7: Implement Approach B (Active Edge List)

**Files:**
- Modify: `rasteriser.go`

**Step 1: Add "slices" to the import block**

Update the import block (math was added in Task 3):

```go
import (
	"math"
	"slices"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/pdf/graphics"
)
```

**Step 2: Add the fillScanline method**

Add after fill2D:

```go
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
```

**Step 3: Verify the code compiles**

Run: `go build ./...`
Expected: No errors

**Step 4: Commit**

```bash
git add rasteriser.go
git commit -m "implement active edge list approach for large path rasterisation"
```

---

## Task 8: Implement FillNonZero and FillEvenOdd

**Files:**
- Modify: `rasteriser.go`

**Step 1: Replace FillNonZero stub**

Replace the FillNonZero method with:

```go
func (r *Rasteriser) FillNonZero(p path.Path, emit func(y, xMin int, coverage []float32)) {
	r.fill(p, fillNonZero, emit)
}
```

**Step 2: Replace FillEvenOdd stub**

Replace the FillEvenOdd method with:

```go
func (r *Rasteriser) FillEvenOdd(p path.Path, emit func(y, xMin int, coverage []float32)) {
	r.fill(p, fillEvenOdd, emit)
}
```

**Step 3: Add the internal fill method**

Add after the public methods:

```go
// fill is the internal implementation shared by FillNonZero and FillEvenOdd.
func (r *Rasteriser) fill(p path.Path, rule fillRule, emit func(y, xMin int, coverage []float32)) {
	// Collect edges from path (returns bounding box clamped to clip)
	xMin, xMax, yMin, yMax, ok := r.collectEdges(p)
	if !ok {
		return // empty or degenerate path
	}

	// Choose approach based on bounding box size
	width := xMax - xMin
	height := yMax - yMin

	if width*height < smallPathThreshold {
		r.fill2D(xMin, xMax, yMin, yMax, rule, emit)
	} else {
		r.fillScanline(xMin, xMax, yMin, yMax, rule, emit)
	}
}
```

**Step 4: Verify the code compiles**

Run: `go build ./...`
Expected: No errors

**Step 5: Run the tests**

Run: `go test -v -run TestAgainstReference`
Expected: Tests run (many will fail due to dummy curve flattening, but polygonal tests should pass)

**Step 6: Commit**

```bash
git add rasteriser.go
git commit -m "implement FillNonZero and FillEvenOdd with hybrid buffer approach"
```

---

## Task 9: Run Tests and Debug

**Files:**
- None (testing only)

**Step 1: Run all tests**

Run: `go test -v`

Note which tests pass and which fail. Polygonal fill tests (triangle, rectangle, star, etc.) should pass. Tests with curves will fail due to dummy flattening.

**Step 2: If tests fail unexpectedly, debug**

Check the debug/ directory for diff images if any tests fail that should pass.

**Step 3: Document results**

Note any issues discovered for follow-up.

---

## Summary

After completing all tasks, you will have:

1. **Edge type and constants** for numerical tolerances
2. **Path traversal** that builds an edge list with CTM transformation
3. **Dummy curve flattening** (TODO markers for proper implementation)
4. **Edge contribution calculation** implementing the signed-area algorithm
5. **Integration with fill rules** (nonzero winding and even-odd)
6. **Approach A (2D buffers)** for small paths
7. **Approach B (active edge list)** for large paths
8. **Working FillNonZero and FillEvenOdd** methods

Tests with polygonal paths should pass. Tests with curves will need proper flattening implementation (separate task).
