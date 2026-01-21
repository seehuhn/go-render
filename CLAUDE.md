# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

go-raster is a 2D vector graphics rasterizer implementing the PDF/PostScript imaging model. It uses the **signed-area coverage accumulation** algorithm to convert vector paths to per-pixel coverage values.

## Build and Test Commands

```bash
go generate          # Export test cases to JSON and generate Cairo reference images
go test              # Run tests against 178 reference images
go test -v           # Verbose output
go test -run TestAgainstReference  # Run the main comparison test
```

The `go generate` pipeline requires Python 3 with Cairo installed.

## Architecture

### Two Processing Pipelines

**Fill Pipeline:**
```
Path (user space) → Flatten curves → Transform to device space → Rasterize → Coverage output
```

**Stroke Pipeline:**
```
Path (user space) → Flatten curves → Expand to outline (caps/joins/dashes) → Transform → Rasterize
```

### Core Algorithm

The rasterizer computes signed trapezoidal area contributions per pixel:
- **Cover value:** Signed vertical extent of edge crossing pixel
- **Area value:** Horizontal position of crossing within pixel
- Final coverage uses nonzero winding or even-odd fill rule

### Key Dependencies

- `seehuhn.de/go/geom` - Provides `path.Data` (path representation), `vec.Vec2`, `matrix.Matrix`
- `seehuhn.de/go/pdf` - Provides `graphics.LineCapStyle`, `graphics.LineJoinStyle` enums

### Entry Point

`RenderExample()` in `render.go` - takes a TestCase and writes grayscale coverage [0-255] to a buffer.

## Test Infrastructure

Tests compare rendered output against Cairo-generated reference images in `testdata/reference/`.

**Tolerance:** ±2 per pixel, max 10% of pixels may differ.

**On failure:** Diff images are written to `debug/` (red=expected, green=actual).

### Adding Test Cases

1. Add case to appropriate file in `testcases/` (fill.go, stroke.go, curve.go, dash.go, precision.go, complex.go, subpath.go)
2. Run `go generate` to regenerate references
3. Run `go test` to verify

## Key Numerical Constants

| Constant | Value | Purpose |
|----------|-------|---------|
| Default flatness | 0.25 | Curve flattening tolerance in device pixels |
| Default miter limit | 10.0 | PDF/PostScript standard |
| Zero-length threshold | 1e-10 | Skip degenerate stroke segments |
| Collinearity threshold | 1e-6 | Detect nearly collinear vectors |
| Cusp detection | cos(179.43°) | Path doubling back on itself |

## Documentation

- `docs/specification.md` - Complete algorithm specification
- `docs/implementation.md` - Design decisions and API
- `docs/methods.md` - State-of-the-art research references
- `admin/test-features.md` - Test coverage checklist
