# State-of-the-art CPU algorithms for 2D vector graphics rasterization

The dominant approach for high-quality PDF/PostScript rasterization is **signed-area coverage accumulation**, pioneered by Raph Levien's LibArt and now used by FreeType, AGG, Blend2D, and stb_truetype. This algorithm computes exact fractional pixel coverage without supersampling by calculating signed trapezoidal areas extending rightward from each path edge. For stroke rendering, **expansion to filled outlines** is the universal approach, though offset curve computation for Bézier curves presents significant mathematical challenges. Modern implementations achieve excellent quality with 256 gray levels while maintaining O(p log p) complexity where p equals edge pixels.

## Analytic coverage computation eliminates supersampling overhead

The mathematical foundation of modern rasterizers relies on computing exact polygon coverage per pixel rather than point sampling. Each directed edge contributes a signed trapezoid extending horizontally to the right edge of the coordinate system. The algorithm tracks two values per segment per pixel: **area** (the portion of the pixel covered by the trapezoid within that pixel) and **cover** (a running total representing edge height contributing to pixels further right).

The critical insight enabling efficiency is the cumulative sum technique. Rather than computing coverage by iterating from each edge to infinity, the algorithm accumulates contributions using the formula: `final_coverage[i] = area[i] + cumulative_sum(cover[0..i])`. This reduces complexity dramatically compared to naive approaches.

Fill rules are handled elegantly through the accumulated alpha value. For **even-odd fill**, the formula is `opacity = abs(alpha - 2.0 × round(0.5 × alpha))`, effectively creating a toggle behavior. For **nonzero winding**, it becomes `opacity = min(abs(alpha), 1.0)`, where edge direction determines whether contributions are positive or negative. Self-intersecting paths work naturally—signed areas cancel or accumulate according to the winding rule without special handling.

Two main approaches dominate implementations. The coverage accumulation method (FreeType, AGG, Blend2D) computes exact trapezoidal areas per pixel. The alternative scanline edge-flag algorithm, described by Kiia Kallio in "Scanline Edge-flag Algorithm for Antialiasing" (TPCG 2007), uses supersampling with an **N-rooks pattern** (8, 16, or 32 samples arranged one per row and column) combined with bit-parallel XOR operations, achieving performance comparable to non-antialiased filling with very good quality.

## All major rasterizers flatten Bézier curves to line segments

Every production CPU rasterizer converts curves to polylines before computing coverage. This includes FreeType, Cairo, Skia, AGG, Blend2D, and stb_truetype. Direct analytic rasterization of polynomial curves is theoretically possible but computationally expensive due to requiring root-finding of cubic or quartic polynomials per scanline, with numerical stability issues at multiple roots.

For **quadratic Bézier curves** (TrueType fonts), Raph Levien developed an optimal flattening formula: the number of segments needed is approximately `√(||P0 - 2P1 + P2|| / (8ε))` where ε is the tolerance (typically 0.25-1.0 pixels). This formula allows computing segments independently, enabling parallelization. For **cubic Bézier curves** (CFF/Type1 fonts, general vector graphics), recursive subdivision at t=0.5 until error falls below tolerance is traditional. Hain's method from 2005 achieves **37% faster flattening** generating only 67% of the segments compared to recursive subdivision. Levien's approach converts cubics to a sequence of quadratics first, then flattens those.

FreeType's smooth rasterizer in `ftgrays.c` explicitly states in its source comments that it "computes the *exact* coverage of the outline on each pixel cell by straight segments" and is "based on ideas that I initially found in Raph Levien's excellent LibArt graphics library." FreeType uses DDA (Digital Differential Analyzer) methods on supported platforms (SSE2, ARM64) for curve processing, falling back to binary subdivision otherwise.

## Raph Levien's font-rs achieved 7.6× FreeType performance through dense buffers

Levien's contributions span from the foundational LibArt signed-area algorithm through modern GPU rendering with Vello. His font-rs project (2016) demonstrated that dramatic speedups over FreeType were possible using the same underlying algorithm with different data structures. Where FreeType uses **sparse linked-list cells** optimized for memory efficiency with complex glyphs, font-rs uses a **dense accumulation buffer** where vectors are drawn immediately without sorting or storage in intermediate structures.

The dense approach avoids branch misprediction and exploits data parallelism. During drawing, the accumulation buffer stores the finite difference (in scanline order) of actual signed areas. Integration sweeps down columns computing winding numbers for final pixel values. Levien reported approximately **460ns to parse binary font data** for a typical "g" glyph at 15ns per path element.

His current Vello project pushes rendering to GPU using compute shaders with a sort-middle architecture. Key innovations include tile-based rendering (256×256 pixel bins subdivided to 16×16 tiles), atomic operations for inserting segments into linked lists per tile, and GPU-side curve flattening. The Ghostscript tiger at 2048×1536 renders in **2-7ms versus 100+ ms** for CPU renderers.

Levien's blog (raphlinus.github.io) contains essential reading on curve mathematics including "Flattening quadratic Béziers," "How long is that Bézier?" (arclength via Legendre-Gauss quadrature), "Fitting cubic Bézier curves," and "Parallel curves of cubic Béziers." His kurbo library provides Rust implementations of these algorithms with documentation of the underlying mathematics.

## Library implementations reveal different optimization priorities

**Cairo** has evolved through multiple approaches. Originally using Bentley-Ottmann algorithm for trapezoid decomposition (still used for X11 XRender backend), it now primarily uses a spans approach developed by Tor Andersson for the fitz library. Anti-aliasing uses supersampling with a **15×17 grid per pixel**. Low-level pixel manipulation delegates to Pixman library with SIMD optimizations.

**Skia's CPU path renderer** uses Analytic Anti-Aliasing (AAA), computing exact coverage values geometrically rather than supersampling. It maintains an active edge table sorted top-to-bottom, processing paths row-by-row with edge list updates. For simple shapes like circles and round-rects, specialized code computes distance to edge analytically. Documentation at skia.org/docs/dev/design/aaa/ describes the approach.

**Anti-Grain Geometry (AGG)** pioneered high-quality open-source rasterization with 256-level anti-aliasing, **24.8 fixed-point coordinates** (8 bits fractional precision), and exact coverage calculation. Despite creator Maxim Shemanarev's death, AGG remains widely used in Matplotlib, Mapnik (OpenStreetMap), PDFium, and FL Studio. Its cell-based scanline rasterizer with QuickSort (later bucket sort) directly influenced subsequent implementations.

**Blend2D** represents the current performance frontier. It uses the same fundamental algorithm as AGG/FreeType but adds **JIT compilation** via AsmJit for generating optimized pipelines at runtime, targets SSE2 through AVX-512 and ARM ASIMD, supports multi-threading, and employs an innovative bit-array indexing scheme where each bit represents N pixels, enabling efficient sparse rendering through bit-scanning. Blend2D achieves quality comparable to AGG while being significantly faster.

**FreeType** produces 256 gray levels through its smooth rasterizer and outputs spans (horizontal pixel segments with identical coverage) that enable direct compositing without intermediate buffers. Its scanline-bucket sorting provides **1.25× speedup** over QuickSort for glyph-sized shapes.

## Stroke expansion is universal but Bézier offsetting is mathematically challenging

Converting strokes to filled outlines is how virtually all PDF/PostScript/SVG renderers work. The algorithm generates parallel curves (offset curves) on both sides of the input path at half the stroke width, adds line caps at endpoints and line joins at segment connections, then fills the resulting outline.

The fundamental challenge is that **the offset of a Bézier curve is not a Bézier curve**—the offset of a cubic Bézier is a 10th-order algebraic curve because the unit normal vector involves square roots. This makes exact polynomial representation impossible.

The classic Tiller-Hanson method (1984) translates each edge of the control polygon by offset distance along the edge normal, then intersects consecutive translated edges to find new control points. This works well for quadratic Béziers but poorly for cubics where error doesn't decrease as fast with subdivision. Blend2D's approach splits curves at an angle threshold to bound approximation error, using the formula `ηmax(φ) = 2 sin⁴(φ/4) / cos(φ/2)` where φ is the angle between control polygon edges.

The 2024 paper "GPU-friendly Stroke Expansion" by Levien and Uguray converts cubic Béziers to **Euler spirals** (curvature linear in arc length) as an intermediate representation. Euler spiral parallel curves have simple closed-form representations, evolutes are simple curves, and cusp detection becomes a linear equation with at most one cusp per segment. This approach with invertible error metrics eliminates recursive subdivision.

PDF line styles require specific implementations: **butt caps** end square at the endpoint, **round caps** add semicircular arcs of radius equal to half line width, **square caps** extend beyond the endpoint by half line width. **Miter joins** extend outer edges until they meet (falling back to bevel when miter_length/line_width exceeds the miter limit), **round joins** add circular arcs, **bevel joins** connect outer endpoints directly. The default miter limit of 10 cuts off miters at angles less than approximately 11 degrees. Dash patterns require arc length parameterization along the path, with each dash segment stroked independently.

## Conclusion: coverage accumulation with curve flattening dominates

Building a PDF rasterizer should follow the well-established pattern: implement signed-area coverage accumulation for fills with curve flattening to line segments, stroke expansion with proper cap/join geometry, and span-based output for efficient compositing. The algorithm achieves exact coverage computation with O(p log p) complexity.

Key implementation choices include dense versus sparse cell storage (dense is faster but uses more memory for complex paths), sorting strategy (bucket sort for glyph-sized shapes, quicksort for larger geometry), and SIMD optimization for the integration sweep. Blend2D's approach of JIT-compiled pipelines with bit-array indexing represents the current performance frontier while maintaining AGG-level quality.

For strokes, the Euler spiral intermediate representation from Levien and Uguray's 2024 work provides the most principled approach to offset curve computation, though simpler Tiller-Hanson with subdivision remains practical for most use cases. The distinction between "weak" stroking (typical implementations) and "strong" stroking (including evolutes in high-curvature regions) matters only in edge cases where path curvature exceeds the reciprocal of half the stroke width.

The essential references are Sean Barrett's stb_truetype documentation explaining the signed trapezoid algorithm, FreeType's ftgrays.c source with extensive comments, Kallio's 2007 TPCG paper on the edge-flag alternative, Nehab's 2020 SIGGRAPH paper on correct stroke expansion, and Levien's blog posts on curve mathematics. These provide complete algorithmic foundations for implementing a production-quality rasterizer.
