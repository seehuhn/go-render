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
- [ ] Concentric rectangles (nonzero vs even-odd) — nested shapes, winding matters
- [ ] Overlapping circles — multiple intersecting regions
- [ ] Figure-eight (self-crossing loop) — single subpath crossing itself
- [ ] High winding number — shape wound 3+ times in same direction
- [ ] Alternating winding — clockwise inside counter-clockwise

### 1.2 Edge Cases

- [ ] Horizontal edges only (axis-aligned rectangle)
- [ ] Vertical edges only (tall thin rectangle)
- [ ] 45° diagonal edges
- [ ] Near-horizontal edges (slope < 0.01)
- [ ] Near-vertical edges (slope > 100)
- [ ] Single-pixel coverage (tiny triangle)
- [ ] Subpixel shape (entirely within one pixel)

### 1.3 Boundary Conditions

- [ ] Shape touching canvas edge
- [ ] Shape partially clipped by canvas
- [ ] Shape fully outside canvas (should produce no output)
- [ ] Pixel-aligned rectangle (should have no antialiasing on edges)
- [ ] Half-pixel offset rectangle (maximum antialiasing)

---

## 2. Stroke Operations

### 2.1 Line Width

**Currently tested:**
- Width 8 (caps), width 6 (joins), width 4 (dashed)

**To add:**
- [ ] Very thin stroke (width 0.5) — subpixel line
- [ ] Thin stroke (width 1.0) — single pixel
- [ ] Thick stroke (width 20) — wide line
- [ ] Width equal to dash length — boundary case

### 2.2 Line Caps

**Currently tested:**
- Butt, round, square on horizontal line

**To add:**
- [ ] Caps on vertical line
- [ ] Caps on 45° diagonal line
- [ ] Caps on 30° diagonal line (asymmetric)
- [ ] Caps on very short segment (length < width)
- [ ] Caps where segment length equals width
- [ ] Zero-length subpath with round cap (should draw filled circle)
- [ ] Zero-length subpath with butt cap (should draw nothing)
- [ ] Zero-length subpath with square cap (should draw nothing)

### 2.3 Line Joins

**Currently tested:**
- Miter, round, bevel on ~70° angle

**To add:**
- [ ] Join at 90° (right angle)
- [ ] Join at 120° (obtuse)
- [ ] Join at 45° (acute)
- [ ] Join at 30° (sharp acute)
- [ ] Join at 15° (very sharp, miter limit test)
- [ ] Join at 170° (near-straight, almost no join visible)
- [ ] Join at 179° (near-cusp)
- [ ] Left turn vs right turn (both directions)
- [ ] Three-segment path (two consecutive joins)
- [ ] Closed triangle (three joins, no caps)
- [ ] Closed square (four 90° joins)

### 2.4 Miter Limit

**Currently tested:**
- Miter limit 10 (default)

**To add:**
- [ ] Miter limit 1.0 (always bevel)
- [ ] Miter limit 1.414 (bevel below 90°)
- [ ] Miter limit 2.0 (bevel below 60°)
- [ ] Angle just below miter cutoff (shows miter)
- [ ] Angle just above miter cutoff (shows bevel)
- [ ] Same angle with different miter limits (comparison)

### 2.5 Cusp Handling

- [ ] 180° turn (exact reversal) — should show two caps
- [ ] 179.5° turn — should show two caps (cusp threshold)
- [ ] 178° turn — should show normal join

---

## 3. Dash Patterns

### 3.1 Basic Patterns

**Currently tested:**
- [8, 4] pattern on horizontal line

**To add:**
- [ ] Single-element pattern [10] (becomes [10, 10])
- [ ] Three-element pattern [5, 3, 8] (becomes [5, 3, 8, 5, 3, 8])
- [ ] Long dash, short gap [20, 2]
- [ ] Short dash, long gap [2, 20]
- [ ] Equal dash and gap [10, 10]
- [ ] Many elements [2, 2, 6, 2, 2, 10]

### 3.2 Dash Phase

- [ ] Phase = 0 (default, starts with dash)
- [ ] Phase = half dash length (starts mid-dash)
- [ ] Phase = dash length (starts with gap)
- [ ] Phase = pattern length (same as phase 0)
- [ ] Negative phase (wraps around)
- [ ] Large negative phase

### 3.3 Zero-Length Dashes

- [ ] Pattern [0, 5] with round cap (dots)
- [ ] Pattern [0, 5] with butt cap (nothing visible from zero-length)
- [ ] Pattern [0, 5] with square cap (nothing visible)
- [ ] Pattern [0, 5, 10, 5] (alternating dots and dashes)

### 3.4 Dashes at Corners

- [ ] Corner falls within dash — join should appear
- [ ] Corner falls within gap — no join visible
- [ ] Dash ends exactly at corner
- [ ] Dash starts exactly at corner
- [ ] Very short dash at corner (dash < corner angle span)

### 3.5 Overlapping Dash Segments (Boundary Case)

This is a critical edge case: when line width is large relative to dash
length, and a corner occurs between two dash segments, the stroke outlines
of adjacent dashes may overlap.

- [ ] Thick dashed line with tight corner — width 10, dash [15, 5], 60° corner
- [ ] Overlapping dash caps at corner — verify no visual artifacts
- [ ] Multiple corners within one dash pattern cycle

### 3.6 Closed Path Dashing

- [ ] Closed square, dashed — pattern restarts or wraps
- [ ] First and last dash connect — when both "on", should join not cap
- [ ] First dash "on", last dash "off" — cap at start, gap at end
- [ ] Phase chosen so path starts and ends in same dash — proper join

---

## 4. Curves

### 4.1 Quadratic Bézier

**Currently tested:**
- Single quadratic curve (filled)

**To add:**
- [ ] Shallow quadratic (control point near chord)
- [ ] Deep quadratic (control point far from chord)
- [ ] Quadratic with control point below chord (curves other direction)
- [ ] S-shaped path from two quadratics
- [ ] Stroked quadratic curve

### 4.2 Cubic Bézier

**Currently tested:**
- Single cubic curve (filled)

**To add:**
- [ ] Shallow cubic
- [ ] Deep cubic
- [ ] S-curve (inflection point)
- [ ] Loop (self-intersecting cubic)
- [ ] Cusp in cubic (control points arranged to create cusp)
- [ ] Nearly-straight cubic (control points near chord)
- [ ] Stroked cubic curve
- [ ] Stroked S-curve

### 4.3 Circle/Ellipse

**Currently tested:**
- Circle approximated by four cubics (filled)

**To add:**
- [ ] Stroked circle
- [ ] Small circle (radius 5) — tests segment count at small scale
- [ ] Large circle (radius 100) — tests segment count at large scale
- [ ] Ellipse (stretched circle)
- [ ] Arc (partial circle)

### 4.4 Curve Flattening Edge Cases

- [ ] Curve requiring many segments (very detailed)
- [ ] Curve requiring minimal segments (nearly flat)
- [ ] Degenerate cubic (all control points coincident)
- [ ] Degenerate quadratic (control point on endpoint)

---

## 5. Complex Paths

### 5.1 Multiple Subpaths

- [ ] Two separate triangles (disjoint subpaths)
- [ ] Two overlapping rectangles (nonzero vs even-odd difference)
- [ ] Ring shape (outer rectangle, inner rectangle cutout)
- [ ] Multiple rings (donut shapes)
- [ ] Many small shapes (stress test)

### 5.2 Mixed Operations

- [ ] Path with lines and curves mixed
- [ ] Stroked path with lines and curves
- [ ] Complex glyph-like shape

### 5.3 Stroke Self-Intersection

- [ ] Spiral that overlaps itself
- [ ] Figure-eight stroke
- [ ] Thick stroke on tight curve (inner edge crosses)
- [ ] Zigzag with thick stroke (adjacent segments overlap)

---

## 6. Coordinate Precision

### 6.1 Subpixel Positioning

- [ ] Shape at integer coordinates
- [ ] Shape offset by 0.25 pixels
- [ ] Shape offset by 0.5 pixels
- [ ] Shape offset by 0.75 pixels
- [ ] Thin line at y=10.0 vs y=10.5 (different coverage patterns)

### 6.2 Large Coordinates

- [ ] Shape centered at (1000, 1000) — tests precision at offset
- [ ] Very small shape at large offset
- [ ] Coordinates requiring full float64 precision

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
