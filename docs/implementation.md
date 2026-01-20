# 2D Vector Graphics Rasteriser — Implementation Plan

This document records implementation decisions for the rasteriser specified in `specification.md`. The implementation is in Go.

---

## 1. Dependencies

The rasteriser uses the following existing packages:

```go
import (
    "seehuhn.de/go/geom/path"
    "seehuhn.de/go/geom/vec"
    "seehuhn.de/go/geom/matrix"
    "seehuhn.de/go/geom/rect"
    "seehuhn.de/go/pdf/graphics"  // for LineCapStyle, LineJoinStyle
)
```

### path.Path

Paths are provided as `path.Path`, which is `iter.Seq2[Command, []vec.Vec2]`. Commands are:

- `CmdMoveTo`: 1 point
- `CmdLineTo`: 1 point
- `CmdQuadTo`: 2 points (control point, endpoint)
- `CmdCubeTo`: 3 points (control point 1, control point 2, endpoint)
- `CmdClose`: 0 points

### vec.Vec2

2D vectors with float64 components:

```go
type Vec2 struct {
    X, Y float64
}
```

Methods include: `Add`, `Sub`, `Mul`, `Dot`, `Length`, `Normalize`, `Normal`, `Rot90`.

### matrix.Matrix

Transformation matrices as `[6]float64` with layout `[a b c d e f]`:

```
x' = a*x + c*y + e
y' = b*x + d*y + f
```

---

## 2. Rasteriser Struct

The rasteriser is a reusable struct. The caller creates one instance and reuses it for multiple paths. Internal buffers grow as needed but never shrink, achieving zero allocations in steady state.

```go
type Rasteriser struct {
    // Transformation (user space → device space)
    CTM matrix.Matrix

    // Clipping bounds in device coordinates
    Clip rect.Rect

    // Curve flattening tolerance in device pixels (typical: 0.25–1.0)
    Flatness float64

    // Stroke parameters (used only by Stroke method)
    Width      float64                // line width in user-space units (must be > 0)
    Cap        graphics.LineCapStyle  // LineCapButt, LineCapRound, LineCapSquare
    Join       graphics.LineJoinStyle // LineJoinMiter, LineJoinRound, LineJoinBevel
    MiterLimit float64                // must be ≥ 1.0
    Dash       []float64              // dash pattern in user-space units (nil for solid)
    DashPhase  float64                // offset into dash pattern

    // Internal buffers (reused across calls)
    cover  []float32 // coverage accumulation: cover change per pixel
    area   []float32 // coverage accumulation: area within pixel
    output []float32 // final coverage values for callback
    stroke []vec.Vec2 // stroke outline vertices
}
```

### Field Validity

The caller is responsible for setting all exported fields to valid values before calling `FillNonZero`, `FillEvenOdd`, or `Stroke`. The rasteriser may panic on invalid values.

| Field | Valid Range |
|-------|-------------|
| `CTM` | Any non-singular matrix |
| `Clip` | Non-empty rectangle with integer-aligned coordinates |
| `Flatness` | > 0 (typical: 0.25–1.0) |
| `Width` | > 0 |
| `Cap` | `LineCapButt`, `LineCapRound`, or `LineCapSquare` |
| `Join` | `LineJoinMiter`, `LineJoinRound`, or `LineJoinBevel` |
| `MiterLimit` | ≥ 1.0 |
| `Dash` | nil (solid) or slice of non-negative values, not all zero |
| `DashPhase` | any float64 |

---

## 3. Public API

### Fill Methods

```go
// FillNonZero rasterises the path using the nonzero winding rule.
// Coverage is delivered row-by-row via the callback.
func (r *Rasteriser) FillNonZero(p path.Path, emit func(y, xMin int, coverage []float32))

// FillEvenOdd rasterises the path using the even-odd fill rule.
// Coverage is delivered row-by-row via the callback.
func (r *Rasteriser) FillEvenOdd(p path.Path, emit func(y, xMin int, coverage []float32))
```

### Stroke Method

```go
// Stroke rasterises the path as a stroked outline.
// Uses Width, Cap, Join, MiterLimit, Dash, and DashPhase from the Rasteriser.
// The stroke outline is filled using the nonzero winding rule.
// Coverage is delivered row-by-row via the callback.
func (r *Rasteriser) Stroke(p path.Path, emit func(y, xMin int, coverage []float32))
```

### Callback Contract

The `emit` callback receives:

- `y`: scanline y-coordinate in device space (integer)
- `xMin`: x-coordinate of first coverage value (integer)
- `coverage`: slice of float32 coverage values in range [0.0, 1.0]

The callback is invoked once per scanline that has non-zero coverage, in increasing y order. The `coverage` slice is valid only for the duration of the callback; the caller must copy it if needed beyond that.

---

## 4. Internal Design

### Numeric Precision

- Coordinates: float64 (via vec.Vec2)
- Coverage accumulation: float32
- Coverage output: float32

### Curve Flattening

Curves are flattened directly to the edge processor without intermediate storage:

1. Receive path segment
2. If curve, compute segment count (Levien's formula for quadratics, Wang's formula for cubics)
3. Emit line segments directly to coverage accumulator

### Stroke Expansion

Stroke outlines are built in a reusable `[]vec.Vec2` buffer:

1. Flatten input path curves
2. Apply dash pattern (if any)
3. Generate stroke outline vertices into `r.stroke`
4. Rasterise the outline as a filled path

### Clipping

The `Clip` rectangle defines the output region. Internally, the rasteriser intersects `Clip` with the path's bounding box to skip empty scanlines.

---

## 5. Design Decisions

### Error Handling for Edge Cases

Empty paths, degenerate paths, and other unusual inputs are handled by silently doing nothing. No errors are returned; the rasteriser simply produces no coverage output.

### Future Optimisations

The code should be structured to accommodate future optimisations (e.g., SIMD) without requiring API changes. Hot paths should be isolated in separate functions that can be replaced with optimised versions later.

---

## 6. Implementation Parameters

All numerical tolerances are collected in a single location as unexported constants for easy tuning.

```go
// Numerical tolerances for the rasteriser.
// All distance tolerances are in device pixels unless otherwise noted.
const (
    // zeroLengthThreshold is the minimum length for a line segment to be
    // considered non-degenerate. Segments shorter than this are skipped
    // during stroke expansion. Value chosen to be well above float64
    // machine epsilon (~2.2e-16) but far below visual perception.
    zeroLengthThreshold = 1e-10

    // zeroLengthThresholdSq is zeroLengthThreshold squared, for efficient
    // comparison without sqrt.
    zeroLengthThresholdSq = 1e-20

    // collinearityThreshold is the |sin(θ)| threshold below which two
    // unit vectors are considered collinear. At this threshold, the angle
    // is approximately 0.00006 degrees (0.2 arc-seconds).
    collinearityThreshold = 1e-6

    // cuspCosineThreshold is the cos(θ) threshold below which a join is
    // treated as a cusp (path doubling back on itself). -0.9999 corresponds
    // to θ > 179.43°.
    cuspCosineThreshold = -0.9999

    // horizontalEdgeThreshold is the minimum vertical extent for an edge
    // to contribute to coverage. Edges with |y1 - y0| below this threshold
    // are skipped as horizontal.
    horizontalEdgeThreshold = 1e-10
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

    // minArcSegments is the minimum number of segments used to approximate
    // a full circle, regardless of flatness tolerance. This prevents
    // degenerate approximations at very large tolerances.
    minArcSegments = 4
)
```

### Tolerance Rationale

| Tolerance | Value | Rationale |
|-----------|-------|-----------|
| Zero-length segment | 1e-10 | Well above float64 epsilon, far below sub-pixel |
| Collinearity | 1e-6 | ~0.2 arc-seconds; handles floating-point rounding |
| Cusp detection | -0.9999 | θ > 179.43°; matches common practice (Skia, Cairo) |
| Horizontal edge | 1e-10 | Consistent with zero-length threshold |
| Default flatness | 0.25 | Below visual perception threshold |
| Default miter limit | 10.0 | PDF/PostScript standard |

---

## 7. Buffer Management

### Reset Method

The `Rasteriser` struct includes a `Reset` method to release internal buffers:

```go
// Reset releases all internal buffers, allowing memory to be reclaimed.
// The Rasteriser remains usable after Reset; buffers will be reallocated
// as needed on the next operation.
func (r *Rasteriser) Reset() {
    r.cover = nil
    r.area = nil
    r.output = nil
    r.stroke = nil
}
```

### Buffer Lifecycle

- Buffers grow as needed via `append`, never shrink during normal operation
- `Reset()` releases all buffers, returning the rasteriser to its initial state
- After `Reset()`, the next operation will allocate fresh buffers
- In steady state (no `Reset()` calls), the rasteriser achieves zero allocations

---

## 8. Testing Strategy

The rasteriser uses the existing testing infrastructure in the package:

### Golden File Testing

- Reference images are generated using Cairo (via `tools/generate_references.py`)
- Test cases are defined in the `testcases/` package
- The test runner compares rendered output against reference images pixel-by-pixel

### Comparison Parameters

- Tolerance: ±2 pixel value units (0-255 scale)
- Maximum allowed differences: 10% of total pixels
- On failure, diff images are written to `debug/` directory

### Test Categories

Existing test cases cover:

- **Fill**: Triangle, star (nonzero/even-odd), rectangle
- **Stroke**: Line caps (butt/round/square), joins (miter/round/bevel), dashed lines
- **Curves**: Quadratic Bézier, cubic Bézier, circle approximation

### Running Tests

```bash
# Generate reference images (requires Python 3 and Cairo)
go generate

# Run tests
go test

# Run with verbose output
go test -v
```

### Adding New Tests

1. Add test case definition to appropriate file in `testcases/`
2. Re-run `go generate` to create reference image
3. Run `go test` to verify
