# Rasteriser Test Features

This document lists renderer features that can be tested by comparing output
to cairo-generated reference images.

---

## 1. Fill Operations

### 1.1 Fill Rules

**Currently tested:**
- triangle_nonzero, triangle_evenodd
- star_nonzero, star_evenodd (self-intersecting)

**To add:**
- [x] Concentric rectangles (nonzero vs even-odd) — nested shapes, winding matters
- [x] Overlapping circles — multiple intersecting regions
- [x] Figure-eight (self-crossing loop) — single subpath crossing itself
- [x] High winding number — shape wound 3+ times in same direction
- [x] Alternating winding — clockwise inside counter-clockwise

### 1.2 Edge Cases

- [x] Horizontal edges only (axis-aligned rectangle)
- [x] Vertical edges only (tall thin rectangle)
- [x] 45° diagonal edges
- [x] Near-horizontal edges (slope < 0.01)
- [x] Near-vertical edges (slope > 100)
- [x] Single-pixel coverage (tiny triangle)
- [x] Subpixel shape (entirely within one pixel)

### 1.3 Boundary Conditions

- [x] Shape touching canvas edge
- [x] Shape partially clipped by canvas
- [x] Shape fully outside canvas (should produce no output)
- [x] Pixel-aligned rectangle (should have no antialiasing on edges)
- [x] Half-pixel offset rectangle (maximum antialiasing)

---

## 2. Stroke Operations

### 2.1 Line Width

**Currently tested:**
- Width 8 (caps), width 6 (joins), width 4 (dashed)

**To add:**
- [x] Very thin stroke (width 0.5) — subpixel line
- [x] Thin stroke (width 1.0) — single pixel
- [x] Thick stroke (width 20) — wide line
- [x] Width equal to dash length — boundary case

### 2.2 Line Caps

**Currently tested:**
- Butt, round, square on horizontal line

**To add:**
- [x] Caps on vertical line
- [x] Caps on 45° diagonal line
- [x] Caps on 30° diagonal line (asymmetric)
- [x] Caps on very short segment (length < width)
- [x] Caps where segment length equals width
- [x] Zero-length subpath with round cap (should draw filled circle)
- [x] Zero-length subpath with butt cap (should draw nothing)
- [x] Zero-length subpath with square cap (should draw nothing)

### 2.3 Line Joins

**Currently tested:**
- Miter, round, bevel on ~70° angle

**To add:**
- [x] Join at 90° (right angle)
- [x] Join at 120° (obtuse)
- [x] Join at 45° (acute)
- [x] Join at 30° (sharp acute)
- [x] Join at 15° (very sharp, miter limit test)
- [x] Join at 170° (near-straight, almost no join visible)
- [x] Join at 179° (near-cusp)
- [x] Left turn vs right turn (both directions)
- [x] Three-segment path (two consecutive joins)
- [x] Closed triangle (three joins, no caps)
- [x] Closed square (four 90° joins)

### 2.4 Miter Limit

**Currently tested:**
- Miter limit 10 (default)

**To add:**
- [x] Miter limit 1.0 (always bevel)
- [x] Miter limit 1.414 (bevel below 90°)
- [x] Miter limit 2.0 (bevel below 60°)
- [x] Angle just below miter cutoff (shows miter)
- [x] Angle just above miter cutoff (shows bevel)
- [x] Same angle with different miter limits (comparison)

### 2.5 Cusp Handling

- [x] 180° turn (exact reversal) — should show two caps
- [x] 179.5° turn — should show two caps (cusp threshold)
- [x] 178° turn — should show normal join

---

## 3. Dash Patterns

### 3.1 Basic Patterns

**Currently tested:**
- [8, 4] pattern on horizontal line

**To add:**
- [x] Single-element pattern [10] (becomes [10, 10])
- [x] Three-element pattern [5, 3, 8] (becomes [5, 3, 8, 5, 3, 8])
- [x] Long dash, short gap [20, 2]
- [x] Short dash, long gap [2, 20]
- [x] Equal dash and gap [10, 10]
- [x] Many elements [2, 2, 6, 2, 2, 10]

### 3.2 Dash Phase

- [x] Phase = 0 (default, starts with dash)
- [x] Phase = half dash length (starts mid-dash)
- [x] Phase = dash length (starts with gap)
- [x] Phase = pattern length (same as phase 0)
- [x] Negative phase (wraps around)
- [x] Large negative phase

### 3.3 Zero-Length Dashes

- [x] Pattern [0, 5] with round cap (dots)
- [x] Pattern [0, 5] with butt cap (nothing visible from zero-length)
- [x] Pattern [0, 5] with square cap (nothing visible)
- [x] Pattern [0, 5, 10, 5] (alternating dots and dashes)

### 3.4 Dashes at Corners

- [x] Corner falls within dash — join should appear
- [x] Corner falls within gap — no join visible
- [x] Dash ends exactly at corner
- [x] Dash starts exactly at corner
- [x] Very short dash at corner (dash < corner angle span)

### 3.5 Overlapping Dash Segments (Boundary Case)

This is a critical edge case: when line width is large relative to dash
length, and a corner occurs between two dash segments, the stroke outlines
of adjacent dashes may overlap.

- [x] Thick dashed line with tight corner — width 10, dash [15, 5], 60° corner
- [x] Overlapping dash caps at corner — verify no visual artifacts
- [x] Multiple corners within one dash pattern cycle

### 3.6 Closed Path Dashing

- [x] Closed square, dashed — pattern restarts or wraps
- [x] First and last dash connect — when both "on", should join not cap
- [x] First dash "on", last dash "off" — cap at start, gap at end
- [x] Phase chosen so path starts and ends in same dash — proper join

---

## 4. Curves

### 4.1 Quadratic Bézier

**Currently tested:**
- Single quadratic curve (filled)

**To add:**
- [x] Shallow quadratic (control point near chord)
- [x] Deep quadratic (control point far from chord)
- [x] Quadratic with control point below chord (curves other direction)
- [x] S-shaped path from two quadratics
- [x] Stroked quadratic curve

### 4.2 Cubic Bézier

**Currently tested:**
- Single cubic curve (filled)

**To add:**
- [x] Shallow cubic
- [x] Deep cubic
- [x] S-curve (inflection point)
- [x] Loop (self-intersecting cubic)
- [x] Cusp in cubic (control points arranged to create cusp)
- [x] Nearly-straight cubic (control points near chord)
- [x] Stroked cubic curve
- [x] Stroked S-curve

### 4.3 Circle/Ellipse

**Currently tested:**
- Circle approximated by four cubics (filled)

**To add:**
- [x] Stroked circle
- [x] Small circle (radius 5) — tests segment count at small scale
- [x] Large circle (radius 100) — tests segment count at large scale
- [x] Ellipse (stretched circle)
- [x] Arc (partial circle)

### 4.4 Curve Flattening Edge Cases

- [x] Curve requiring many segments (very detailed)
- [x] Curve requiring minimal segments (nearly flat)
- [x] Degenerate cubic (all control points coincident)
- [x] Degenerate quadratic (control point on endpoint)

---

## 5. Complex Paths

### 5.1 Multiple Subpaths

- [x] Two separate triangles (disjoint subpaths)
- [x] Two overlapping rectangles (nonzero vs even-odd difference)
- [x] Ring shape (outer rectangle, inner rectangle cutout)
- [x] Multiple rings (donut shapes)
- [x] Many small shapes (stress test)

### 5.2 Mixed Operations

- [x] Path with lines and curves mixed
- [x] Stroked path with lines and curves
- [x] Complex glyph-like shape

### 5.3 Stroke Self-Intersection

- [x] Spiral that overlaps itself
- [x] Figure-eight stroke
- [x] Thick stroke on tight curve (inner edge crosses)
- [x] Zigzag with thick stroke (adjacent segments overlap)

---

## 6. Coordinate Precision

### 6.1 Subpixel Positioning

- [x] Shape at integer coordinates
- [x] Shape offset by 0.25 pixels
- [x] Shape offset by 0.5 pixels
- [x] Shape offset by 0.75 pixels
- [x] Thin line at y=10.0 vs y=10.5 (different coverage patterns)

### 6.2 Large Coordinates

- [x] Shape centered at (1000, 1000) — tests precision at offset
- [x] Very small shape at large offset
- [x] Coordinates requiring full float64 precision

---

## 7. Transformation (CTM) Effects

If CTM testing is supported:

### 7.1 Uniform Scaling

- [ ] 2x scale
- [ ] 0.5x scale
- [ ] Large scale (10x)

### 7.2 Rotation

- [ ] 45° rotation
- [ ] 90° rotation
- [ ] Small rotation (5°)

### 7.3 Non-Uniform Scaling

- [ ] 2x horizontal, 1x vertical
- [ ] 1x horizontal, 2x vertical
- [ ] Circle under non-uniform scale (becomes ellipse)

### 7.4 Shear

- [ ] Horizontal shear
- [ ] Vertical shear
- [ ] Combined shear and rotation

### 7.5 Strokes Under Transform

- [ ] Round cap under non-uniform scale (should remain circular in user space)
- [ ] Round join under rotation
- [ ] Dash pattern under scale (lengths should scale)

---

## 8. Visual Quality Tests

### 8.1 Antialiasing Consistency

- [ ] Horizontal line vs vertical line (same coverage quality)
- [ ] Diagonal line at multiple angles (smooth gradient)
- [ ] Circle edge quality (smooth curve, no faceting visible)

### 8.2 Corner Quality

- [ ] Sharp miter point (no gaps, correct coverage)
- [ ] Round join smoothness
- [ ] Bevel join alignment

---

## Priority Order for Implementation

### High Priority (Core Functionality)

1. Line width variations (2.1)
2. Caps on different orientations (2.2)
3. Joins at various angles (2.3)
4. Miter limit tests (2.4)
5. Dash phase variations (3.2)
6. Zero-length dashes (3.3)
7. Stroked curves (4.1, 4.2, 4.3)

### Medium Priority (Edge Cases)

8. Dashes at corners (3.4)
9. Overlapping dash segments (3.5)
10. Closed path dashing (3.6)
11. Cusp handling (2.5)
12. Multiple subpaths (5.1)
13. Fill rule edge cases (1.1)

### Lower Priority (Stress Tests)

14. Subpixel positioning (6.1)
15. Large coordinates (6.2)
16. Complex paths (5.2, 5.3)
17. Transformation tests (7.x) — if supported
18. Visual quality tests (8.x)

---

## Test Image Conventions

- Canvas size: 64×64 for basic tests, 128×128 for complex tests
- Background: transparent or white
- Stroke color: black
- Fill color: black
- Naming: `{category}_{feature}_{variant}.png`
  - Example: `stroke_join_miter_90deg.png`
  - Example: `dash_phase_half.png`
