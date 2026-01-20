# 2D Vector Graphics Rasteriser Specification

This document specifies a rasteriser for the PDF/PostScript imaging model. It produces anti-aliased coverage values without supersampling, suitable for rendering both page graphics and font glyphs.

---

## 1. Scope and Goals

**Input:**
- Paths composed of straight line segments, quadratic Bézier curves, and cubic Bézier curves
- Coordinates are floating-point in user space (sub-pixel precision after transformation)
- Fill rule: nonzero winding number or even-odd
- For strokes: line width, cap style, join style, miter limit, dash pattern

**Output:**
- Per-pixel coverage values in the range [0.0, 1.0]
- Coverage represents the fraction of each pixel's area covered by the path interior
- Output is delivered row-by-row to a compositor

**Priorities:**
1. Correctness
2. Speed

---

## 2. Coordinate Systems and Renderer State

### 2.1 Coordinate Systems

The renderer operates with two coordinate systems:

**User space:** The coordinate system in which paths are defined. This is the coordinate system used by the PDF/PostScript content stream. Stroke parameters (line width, dash lengths) are specified in user-space units.

**Device space:** The coordinate system of the output pixel grid. One unit equals one pixel. The rasteriser's coverage accumulation algorithm (§3) operates in device space.

### 2.2 Current Transformation Matrix (CTM)

The CTM transforms points from user space to device space:
```
[x_device]   [a  b  tx] [x_user]
[y_device] = [c  d  ty] [y_user]
[   1    ]   [0  0   1] [   1   ]
```

Or in expanded form:
```
x_device = a × x_user + b × y_user + tx
y_device = c × x_user + d × y_user + ty
```

The 2×2 linear part `M = [a b; c d]` controls scaling, rotation, and shear. The translation `(tx, ty)` controls position.

**Conformal CTM:** A CTM is conformal if it preserves angles, i.e., it consists only of uniform scaling, rotation, and translation. Mathematically: `a = d`, `b = -c`, and `a² + b² > 0`. Under a conformal CTM, circles remain circles.

**Non-conformal CTM:** Includes non-uniform scaling or shear. Circles become ellipses.

### 2.3 Renderer State

The renderer maintains the following state:

**Transformation:**
- `ctm` — the current transformation matrix (user space → device space)

**Rasterisation:**
- `flatness_tolerance` — maximum deviation in device pixels

**Stroke parameters** (used only for stroke operations):
- `line_width` — stroke width in user-space units
- `line_cap` — cap style: butt, round, or square
- `line_join` — join style: miter, round, or bevel
- `miter_limit` — dimensionless ratio
- `dash_pattern` — array of dash/gap lengths in user-space units (empty array = solid)
- `dash_phase` — offset into dash pattern in user-space units

**Fill parameters:**
- `fill_rule` — nonzero winding or even-odd

### 2.4 Processing Pipelines

**Fill pipeline:**
1. Receive path in user space
2. Flatten curves (CTM-aware tolerance, see §5)
3. Transform flattened path to device space
4. Rasterise line segments (§3)

**Stroke pipeline:**
1. Receive path in user space
2. Flatten curves (CTM-aware tolerance, see §5)
3. Expand to stroke outline in user space (§6)
4. Transform stroke outline to device space
5. Rasterise line segments (§3)

Both pipelines end with the same final steps: transform to device space, then rasterise. The core rasteriser (§3) always operates on line segments in device coordinates.

---

## 3. Core Algorithm: Signed-Area Coverage Accumulation

### 3.1 Principle

Each directed edge of the path contributes a signed trapezoidal area extending horizontally to the right. The algorithm computes two values per pixel touched by an edge:

- **cover**: the change in winding number contribution as we cross this pixel vertically
- **area**: the signed area within this pixel only

These values are accumulated in row buffers, then integrated left-to-right to produce final coverage.

### 3.2 Edge Contribution

Consider a directed line segment from point P0 = (x0, y0) to P1 = (x1, y1) crossing through a pixel whose left edge is at integer coordinate X and whose top edge is at integer coordinate Y. The pixel spans [X, X+1) horizontally and [Y, Y+1) vertically.

**Horizontal edges:** If y0 = y1, the edge has no vertical extent and contributes nothing. Skip it entirely.

For non-horizontal edges, compute the portion of the edge within this pixel:
- y_top = max(Y, min(y0, y1))
- y_bot = min(Y+1, max(y0, y1))
- dy = y_bot - y_top (always ≥ 0)

The **cover** contribution is the signed vertical extent:
```
cover = (y1 > y0) ? +dy : -dy
```
More precisely, cover = (y_bot - y_top) × sign(y1 - y0).

The **area** contribution accounts for the horizontal position of the edge within the pixel. If the edge crosses the pixel at x_mid (the x-coordinate at the vertical midpoint of the edge segment within this pixel):
```
area = cover × (1.0 - (x_mid - X))
```
where (x_mid - X) is the fractional x-position within the pixel (0.0 = left edge, 1.0 = right edge).

Note: For steep edges, x_mid may fall outside the pixel's horizontal bounds [X, X+1). This is correct — do not clamp x_mid. The resulting area values outside [0, cover] are compensated during integration across adjacent pixels.

For a non-vertical edge, compute x_mid by linear interpolation. Since horizontal edges were already excluded, (y1 - y0) ≠ 0:
```
y_mid = (y_top + y_bot) / 2
x_mid = x0 + (x1 - x0) × (y_mid - y0) / (y1 - y0)
```

For a vertical edge at x = x0:
```
area = cover × (1.0 - frac(x0))
```
where frac(x0) = x0 - floor(x0).

### 3.3 Integration

After all edges have been processed for a scanline Y, compute final coverage by integrating left-to-right:

```
accumulated_cover = 0.0
for x from x_min to x_max:
    accumulated_cover += cover[x]
    raw_coverage = accumulated_cover + area[x]
    final_coverage[x] = apply_fill_rule(raw_coverage)
```

### 3.4 Fill Rules

**Nonzero winding:**
```
apply_fill_rule(c) = clamp(abs(c), 0.0, 1.0)
```

**Even-odd:**
```
apply_fill_rule(c) = 1.0 - abs(1.0 - mod(abs(c), 2.0))
```
where mod(x, 2.0) returns x - 2.0 × floor(x / 2.0). This maps c to a sawtooth wave oscillating between 0 and 1: as |c| increases from 0 to 1, coverage rises from 0 to 1; as |c| continues from 1 to 2, coverage falls back from 1 to 0; then the pattern repeats.

### 3.5 Edge Direction

Edge direction matters for correct winding:
- Downward edges (y1 > y0): positive contribution to cover
- Upward edges (y1 < y0): negative contribution to cover
- Horizontal edges (y1 = y0): no contribution (skip entirely, as noted in §3.2)

---

## 4. Buffer Management

Two approaches exist for managing the coverage accumulation buffers. Implementations may use either or both, selected by heuristic.

### 4.1 Approach A: 2D Buffers (Small Paths)

For small paths (e.g., glyphs, icons), use 2D buffers sized to the path's bounding box:

- `cover[y][x]` — float32, indexed by pixel coordinates
- `area[y][x]` — float32, indexed by pixel coordinates

**Processing:**
1. Compute the path's bounding box in device coordinates, clamp to clip bounds
2. Allocate (or reuse) 2D buffers for the bounding box
3. Process edges in path order — each edge writes to all scanlines it touches
4. After all edges are processed, integrate and emit each row

**Memory:** O(width × height). For a 256×256 region: ~512 KB (2 buffers × 64K pixels × 4 bytes).

**Trade-off:** Simple implementation, no sorting required. Suitable when `width × height` is small (e.g., < 256K pixels).

### 4.2 Approach B: 1D Buffers + Active Edge List (Large Paths)

For large paths (e.g., page-spanning fills), use 1D buffers with scanline-by-scanline processing:

- `cover[x]` — float32, one row tall
- `area[x]` — float32, one row tall

**Processing:**
1. Compute the path's bounding box in device coordinates, clamp to clip bounds
2. Build an edge list from the path, storing (x0, y0, x1, y1) for each edge
3. Sort edges by y_min (the smaller y-coordinate of each edge)
4. Process scanlines from y_min to y_max:
   - Add edges that start at this scanline to the active edge list
   - For each active edge, compute its contribution to this scanline's `cover` and `area`
   - Remove edges that end at this scanline from the active edge list
   - Integrate and emit the row
   - Clear the touched portion of `cover` and `area`

**Memory:** O(width + edges). Independent of path height.

**Trade-off:** More complex implementation, requires sorting. Suitable for large regions with relatively few edges.

### 4.3 Selection Heuristic

A practical heuristic for choosing between approaches:

```
if (bbox_width × bbox_height < threshold)   // e.g., 262144 (512×512)
    use Approach A (2D buffers)
else
    use Approach B (active edge list)
```

Glyphs and small shapes get the simpler 2D approach. Large page-spanning fills get the memory-efficient active edge list.

### 4.4 Output

After integrating each scanline:
- Pass the row's coverage data to the compositor
- The compositor receives: Y coordinate, x_min, x_max, and the coverage array for that row

---

## 5. Curve Flattening

All curves are converted to sequences of line segments before rasterisation. Flattening occurs in user space, with CTM-aware tolerance checking to ensure accuracy in device space.

### 5.1 CTM-Aware Tolerance

The flatness tolerance ε is specified in device pixels (typical values: 0.25 to 1.0). Since flattening occurs in user space but accuracy is measured in device space, the CTM must be used to evaluate whether a curve segment is flat enough.

**Principle:** Compute the deviation vector in user space, transform it by the CTM's linear part (the 2×2 matrix M, ignoring translation), and compare the resulting device-space magnitude against ε.

For a user-space deviation vector `d_user`:
```
d_device = M × d_user
flat_enough = ||d_device|| ≤ ε
```

This approach generates the optimal number of segments for any CTM, including non-conformal transformations with shear or non-uniform scaling.

### 5.2 Quadratic Bézier Curves

A quadratic Bézier with control points P0, P1, P2 is flattened using Levien's formula, extended for CTM-awareness.

The maximum error vector (in user space) for a quadratic Bézier approximated by its chord is:
```
e_user = (P0 - 2×P1 + P2) / 4
```

Transform to device space and check against tolerance:
```
e_device = M × e_user
```

If `||e_device|| ≤ ε`, draw a single line from P0 to P2.

Otherwise, compute the number of segments needed. When a quadratic Bézier is subdivided into n equal parameter intervals, the maximum error of each sub-curve is the original error divided by n². Therefore:
```
n = ceil(sqrt(||e_device|| / ε))
```

For a conformal CTM with scale factor s, this simplifies to:
```
n = ceil(sqrt(s × ||e_user|| / ε))
```

For a non-conformal CTM, compute ||e_device|| = ||M × e_user|| directly. Alternatively, use the maximum singular value σ_max of M as a conservative upper bound (this may generate slightly more segments than necessary):
```
n = ceil(sqrt(σ_max × ||e_user|| / ε))
```

The maximum singular value can be computed as:
```
σ_max = sqrt((a² + b² + c² + d²) / 2 + sqrt(((a² + b²) - (c² + d²))² / 4 + (a×c + b×d)²))
```

Or approximated conservatively as:
```
σ_max ≤ sqrt(max(a² + b², c² + d²)) × sqrt(2)
```

Subdivide the parameter range [0, 1] into n equal intervals and evaluate the curve at t = i/n for i = 0, 1, ..., n:
```
B(t) = (1-t)²×P0 + 2×(1-t)×t×P1 + t²×P2
```

### 5.3 Cubic Bézier Curves

A cubic Bézier with control points P0, P1, P2, P3 is flattened using Wang's formula, extended for CTM-awareness. This computes the required number of segments directly from the control points, avoiding recursive subdivision.

**Deviation bound:** The maximum deviation of a cubic Bézier from its chord is bounded by the "bend" at each end of the curve:
```
d1_user = P0 - 2×P1 + P2
d2_user = P1 - 2×P2 + P3
```

These vectors measure how far the control polygon deviates from collinearity at each end.

**CTM-aware segment count:** Transform the deviation vectors to device space and compute the segment count:
```
d1_device = M × d1_user
d2_device = M × d2_user
M_dev = max(||d1_device||, ||d2_device||)

if M_dev ≤ ε:
    n = 1  // Single line segment suffices
else:
    n = ceil(sqrt(3 × M_dev / (4 × ε)))
```

Where M is the CTM's 2×2 linear part and ε is the flatness tolerance in device pixels.

For a conformal CTM with scale factor s, this simplifies to:
```
M_user = max(||d1_user||, ||d2_user||)
n = ceil(sqrt(3 × s × M_user / (4 × ε)))
```

For a non-conformal CTM, either compute ||d1_device|| and ||d2_device|| directly, or use the maximum singular value σ_max of M as a conservative upper bound:
```
M_user = max(||d1_user||, ||d2_user||)
n = ceil(sqrt(3 × σ_max × M_user / (4 × ε)))
```

**Curve evaluation:** Subdivide the parameter range [0, 1] into n equal intervals and evaluate the curve at t = i/n for i = 0, 1, ..., n:
```
B(t) = (1-t)³×P0 + 3×(1-t)²×t×P1 + 3×(1-t)×t²×P2 + t³×P3
```

**Degenerate curves:** If all four control points are coincident (or nearly so), the curve degenerates to a point. In this case, emit a single line segment from P0 to P3 (which will have zero or near-zero length and contribute nothing to the rasterised output).

---

## 6. Stroke Expansion

Strokes are converted to filled outlines in user space, then transformed to device space and rasterised using the same fill algorithm. All stroke parameters (line width, dash lengths, etc.) are interpreted in user-space units.

### 6.1 Overview

1. Flatten all curves to line segments (CTM-aware tolerance, see §5)
2. Apply dash pattern to split path into dash segments (arc length measured in user space)
3. Offset each line segment by ±half_line_width perpendicular to the segment
4. Add line joins between consecutive segments
5. Add line caps at path endpoints
6. The result is a closed polygon representing the stroke outline in user space
7. Transform the outline to device space (applied by the pipeline, §2.4)

### 6.2 Line Segment Offsetting

For a line segment from A to B:

**Zero-length segments:** If A = B (or ||B - A|| < δ for a small numerical tolerance δ), the segment has no defined direction. Skip such segments during stroke expansion — they contribute no geometry to the stroke outline. Note that zero-length *subpaths* (an entire subpath with no length) are handled separately in §6.9.

For non-zero-length segments:

1. Compute the unit tangent: T = normalise(B - A)
2. Compute the unit normal: N = (-T.y, T.x) (90° counterclockwise rotation)
3. Offset amount: d = line_width / 2

The offset segment on the left (counterclockwise) side:
- A_left = A + d × N
- B_left = B + d × N

The offset segment on the right (clockwise) side:
- A_right = A - d × N
- B_right = B - d × N

### 6.3 Line Caps

Applied at the start and end of each open subpath (and at dash endpoints).

Let P be the endpoint, T the outward-pointing unit tangent (pointing away from the path interior), and N the unit normal. Let d = line_width / 2.

**Butt cap:**
No extension. The stroke ends with a line perpendicular to the path:
- Left point: P + d × N
- Right point: P - d × N
- Connect with a straight line.

**Square cap:**
Extend by half the line width beyond the endpoint:
- P_extended = P + d × T
- Left point: P_extended + d × N
- Right point: P_extended - d × N
- Connect with straight lines from the stroke edges to these points.

**Round cap:**
Add a semicircular arc centred at P with radius d:
- Approximate the semicircle with line segments (see §6.6)
- The arc spans from (P + d × N) to (P - d × N), going around in the direction of T.

### 6.4 Line Joins

Applied where two consecutive line segments meet at a point P. Let T1 be the incoming unit tangent, T2 the outgoing unit tangent, and d = line_width / 2.

Compute the turn angle θ between segments:
```
cos_θ = dot(T1, T2)
sin_θ = cross(T1, T2)  // 2D cross product: T1.x × T2.y - T1.y × T2.x
```

The sign of sin_θ determines the turn direction:
- sin_θ > 0: left turn (counterclockwise) — outer side is on the left
- sin_θ < 0: right turn (clockwise) — outer side is on the right

If |sin_θ| < δ (nearly collinear, where δ is a small numerical tolerance such as 1e-6), no join is needed.

**Miter join:**

On the outer side of the turn, extend both offset edges until they meet.

The miter length is the distance from the inner edge of the stroke (at the join) to the tip of the miter. Per the PDF specification:
```
miter_length / line_width = 1 / sin(θ/2)
```

Using the half-angle identity sin(θ/2) = sqrt((1 - cos_θ) / 2):
```
sin_half = sqrt((1 - cos_θ) / 2)
miter_length = line_width / sin_half
```

If miter_length / line_width > miter_limit (equivalently, 1 / sin_half > miter_limit), fall back to bevel join.

Otherwise, compute the miter point. Using cos(θ/2) = sqrt((1 + cos_θ) / 2):
```
cos_half = sqrt((1 + cos_θ) / 2)
miter_direction = normalise(T1 + T2)  // points toward the outer side
miter_point = P + miter_direction × (d / cos_half)
```

On the inner side, the two offset edges converge and intersect before reaching P. Compute the inner intersection point:
```
inner_point = P - miter_direction × (d / cos_half)
```
Use this single point to connect the inner offset edges, avoiding self-intersection in the stroke outline.

**Bevel join:**

On the outer side, connect the two offset endpoints with a straight line (no extension).

On the inner side, compute the intersection of the two inner offset lines as described above.

**Round join:**

On the outer side, add a circular arc centred at P connecting the two offset endpoints.

On the inner side, compute the intersection of the two inner offset lines as described above.

Approximate the outer arc with line segments (see §6.6).

### 6.5 Miter Limit

The miter limit is a dimensionless ratio, which must be ≥ 1.0.

The miter limit M corresponds to a minimum angle θ_min:
```
M = 1 / sin(θ_min / 2)
```

For M = 10, θ_min ≈ 11.5°.

When the turn angle θ < θ_min, the miter join is replaced with a bevel join.

### 6.6 CTM-Aware Arc Approximation (for Round Joins and Caps)

Round caps and joins are circles in user space, which become ellipses in device space under non-conformal CTMs. The arc approximation must account for this to achieve the desired device-space accuracy.

**Principle:** Generate arc vertices in user space, but determine the number of vertices by considering how the arc will appear in device space after transformation.

Let d = line_width / 2, M be the CTM's 2×2 linear part, and ε be the device-space flatness tolerance.

**For a conformal CTM** (uniform scale s, rotation only):

The circle remains a circle in device space with radius s × d. Use:
```
n = ceil(π / acos(1 - ε / (s × d)))
```

**For a non-conformal CTM:**

The user-space circle of radius d becomes an ellipse in device space. The approximation error varies around the ellipse — it is largest where the ellipse has the highest curvature (near the ends of the major axis).

Compute the semi-major axis length of the transformed ellipse:
```
σ_max = maximum singular value of M (see §5.2)
device_radius_max = σ_max × d
```

Use this maximum radius to determine the segment count (conservative but simple):
```
n = ceil(π / acos(1 - ε / device_radius_max))
```

For a more accurate (but complex) approach, compute the number of segments adaptively based on the local curvature of the ellipse at each point.

**Generating arc vertices:**

For a full circle, generate n+1 vertices (n segments) in user space. The loop bounds are inclusive:
```
for i in 0 to n (inclusive):
    angle = 2π × i / n
    vertex[i] = centre + (d × cos(angle), d × sin(angle))
```

For a semicircle (cap) spanning from angle α to α + π:
```
segments = ceil(n / 2)
for i in 0 to segments (inclusive):
    angle = α + π × i / segments
    vertex[i] = centre + (d × cos(angle), d × sin(angle))
```

For an arc spanning angle θ (for round joins):
```
segments = ceil(n × θ / (2π))
for i in 0 to segments (inclusive):
    angle = start_angle + θ × i / segments
    vertex[i] = centre + (d × cos(angle), d × sin(angle))
```

These user-space vertices will be transformed to device space along with the rest of the stroke outline.

### 6.7 Dash Patterns

A dash pattern is an array of non-negative lengths [d0, d1, d2, ...] in user-space units, specifying alternating "on" (drawn) and "off" (gap) segments. The elements must not all be zero. The dash phase is an offset into the pattern at which stroking begins, also in user-space units.

**Odd-length patterns:** If the dash pattern array has an odd number of elements, it is doubled before use. For example, a pattern [3] becomes [3, 3], and a pattern [2, 1, 4] becomes [2, 1, 4, 2, 1, 4]. This ensures the pattern always has paired on/off lengths.

**Negative dash phase:** If the dash phase is negative, add twice the total pattern length repeatedly until it becomes non-negative:
```
L = sum of all dash lengths
while phase < 0:
    phase += 2 × L
```

**Processing:**

1. Compute the total pattern length: L = sum of all dash lengths
2. Normalise the phase to [0, L) as described above
3. Walk along the flattened path, measuring arc length in user space
4. At each transition from on→off or off→on, split the path
5. Each "on" segment becomes a separate open subpath to be stroked with caps at both ends

**Subpath independence:** When a path consists of multiple subpaths, each subpath is treated independently. The dash pattern restarts and the dash phase is reapplied at the beginning of each subpath.

**Closed dashed subpaths:** In a closed subpath that is dashed, if the first dash segment starts with "on" and the last dash segment ends within "on", then the first and last dash segments are joined (no caps at the closure point; instead, a line join connects them).

**Corners and dashes:**
- If a corner (where two segments meet at an angle) falls within an "on" dash segment, it is painted using the current line join style.
- If a corner falls within an "off" gap, it is not painted.
- If a dash segment ends exactly at a corner, the end cap is painted before the corner join.

**Arc length along flattened path:**

For a line segment from A to B in user space:
```
length = ||B - A||
```

Sum the lengths of all line segments to get cumulative arc length. Since the path has been flattened before dashing, this measures arc length along the polygonal approximation, which closely approximates the true curve arc length (error is O(ε²) where ε is the flatness tolerance).

### 6.8 Cusp Handling

When the path doubles back sharply (the incoming and outgoing tangents point in nearly opposite directions), a cusp occurs.

Detection:
```
cos_θ = dot(T1, T2)
if cos_θ < -0.9999:  // nearly 180° turn
    cusp detected
```

At a cusp, instead of a normal join:
1. Add a line cap on the incoming segment (as if it ended at P)
2. Add a line cap on the outgoing segment (as if it started at P)

This prevents self-intersection in the stroke outline.

### 6.9 Zero-Length Subpaths

A zero-length subpath occurs when a subpath consists of a single point with no line segments (e.g., a moveto immediately followed by a closepath, or a moveto followed by a lineto to the same point).

**Round caps:** Draw a filled circle centred at the point with radius equal to half the line width. This is equivalent to two semicircular caps drawn back-to-back.

**Butt and square caps:** Draw nothing. Since there is no line segment, the tangent direction is undefined, and thus the orientation of the caps is indeterminate.

**Zero-length dashes:** A zero-length "on" segment in a dash pattern (e.g., from a dash array like [0, 3]) is treated like a zero-length subpath: round caps produce a filled circle; butt and square caps produce nothing.

### 6.10 Stroke Self-Intersection

When stroking thick lines along paths with tight curves, the stroke outline may self-intersect. This occurs when the offset curves on opposite sides of the path cross each other.

The stroke outline should be rasterised as-is, without attempting to detect or resolve self-intersections. Stroke outlines are always filled using the nonzero winding rule, regardless of the current fill_rule setting (which applies only to fill operations, not strokes). Under the nonzero rule, self-intersecting regions are rendered correctly: overlapping areas receive winding number ±2 or higher, which clamps to full coverage. This produces the expected visual result where the entire stroke interior is filled uniformly.

Attempting to compute the union of self-intersecting stroke regions adds significant complexity and is not required for correct rendering.

### 6.11 Assembling the Stroke Outline

For an open subpath with segments S0, S1, ..., Sn-1:

1. Start with the cap at the beginning of S0
2. Walk forward along the left (outer) side:
   - Left edge of S0
   - Join at S0/S1 junction (left side)
   - Left edge of S1
   - ... continue to end of Sn-1
3. Add cap at end of Sn-1
4. Walk backward along the right (inner) side:
   - Right edge of Sn-1
   - Join at Sn-2/Sn-1 junction (right side)
   - ... continue back to start of S0
5. Close the path

For a closed subpath (where the last point equals the first point):
- No caps
- Add a join connecting the last segment to the first segment

The resulting outline is in user space. The pipeline (§2.4) transforms it to device space before rasterisation.

---

## 7. Clipping

Clipping is out of scope for this specification. The rasteriser produces coverage values for paths; combining these with clip masks is the responsibility of the compositor.

---

## 8. Summary of Parameters

| Parameter | Notes |
|-----------|-------|
| Flatness tolerance | Device pixels (typical values: 0.25–1.0) |
| Miter limit | Dimensionless ratio; must be ≥ 1.0 |
| Line cap | Butt, round, or square |
| Line join | Miter, round, or bevel |
| Line width | User-space units; must be > 0 |
| Dash pattern | User-space units; empty array = no dashing |
| Dash phase | User-space units; offset into dash pattern |
| Fill rule | Nonzero winding or even-odd |

---

## 9. References

- Sean Barrett, "How the stb_truetype Anti-Aliased Software Rasterizer v2 Works" — explains signed-area coverage accumulation
- FreeType `ftgrays.c` source code — production implementation with extensive comments
- Raph Levien, "Flattening quadratic Béziers" — optimal segment count formula for quadratics
- Raph Levien, kurbo library (`flatten.rs`) — Wang's formula implementation for cubics
- Wang, Xiaolin, "Parabolic approximation and best-fit of Bézier curves and curve degree reduction" — original derivation of segment count bounds
- Cairo `cairo-path-stroke.c` — stroke expansion implementation
- PDF Reference Manual — specification of line styles, fill rules, and flatness
- Blend2D `precise_offset_curves.pdf` — advanced offset curve techniques (for future reference)
