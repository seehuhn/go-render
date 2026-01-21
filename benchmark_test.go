package render

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"golang.org/x/image/vector"

	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/geom/vec"
)

// BenchmarkRasteriserO benchmarks our rasterizer drawing an "O" shape.
func BenchmarkRasteriserO(b *testing.B) {
	sizes := []int{20, 200, 2000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%dx%d", size, size), func(b *testing.B) {
			clip := rect.Rect{LLx: 0, LLy: 0, URx: float64(size), URy: float64(size)}
			r := NewRasteriser(clip)

			dst := image.NewAlpha(image.Rect(0, 0, size, size))

			center := float64(size) / 2
			outerR := float64(size) * 0.45
			innerR := float64(size) * 0.30

			// Create the "O" path: outer circle CCW, inner circle CW
			oPath := makeOPath(center, center, outerR, innerR)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				r.Reset(clip)
				r.FillEvenOdd(oPath, func(y, xMin int, coverage []float32) {
					row := dst.Pix[y*dst.Stride+xMin:]
					for i, c := range coverage {
						row[i] = uint8(c * 255)
					}
				})
			}
		})
	}
}

// BenchmarkVectorO benchmarks x/image/vector drawing an "O" shape.
func BenchmarkVectorO(b *testing.B) {
	sizes := []int{20, 200, 2000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%dx%d", size, size), func(b *testing.B) {
			r := vector.NewRasterizer(size, size)

			dst := image.NewAlpha(image.Rect(0, 0, size, size))
			src := image.NewUniform(color.Alpha{255})

			center := float32(size) / 2
			outerR := float32(size) * 0.45
			innerR := float32(size) * 0.30

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				r.Reset(size, size)

				// Outer circle (counter-clockwise)
				addCircleToVector(r, center, center, outerR, false)
				// Inner circle (clockwise)
				addCircleToVector(r, center, center, innerR, true)

				// Rasterize and composite
				r.Draw(dst, dst.Bounds(), src, image.Point{})
			}
		})
	}
}

// makeOPath creates an "O" shape path for our rasterizer.
// Outer circle is counter-clockwise, inner circle is clockwise.
func makeOPath(cx, cy, outerR, innerR float64) path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		// Outer circle (counter-clockwise)
		addCircleToPath(yield, cx, cy, outerR, false)
		// Inner circle (clockwise)
		addCircleToPath(yield, cx, cy, innerR, true)
	}
}

// addCircleToPath adds a circle to a path using cubic Bézier curves.
// Uses a stack-allocated buffer to avoid heap allocations.
func addCircleToPath(yield func(path.Command, []vec.Vec2) bool, cx, cy, r float64, clockwise bool) {
	// Magic number for circular arc approximation with cubic Bézier
	const k = 0.5522847498
	kr := k * r

	var buf [3]vec.Vec2 // stack-allocated, reused for each yield

	if clockwise {
		// Start at top, go clockwise
		buf[0] = vec.Vec2{X: cx, Y: cy - r}
		if !yield(path.CmdMoveTo, buf[:1]) {
			return
		}
		buf[0], buf[1], buf[2] = vec.Vec2{X: cx - kr, Y: cy - r}, vec.Vec2{X: cx - r, Y: cy - kr}, vec.Vec2{X: cx - r, Y: cy}
		if !yield(path.CmdCubeTo, buf[:3]) {
			return
		}
		buf[0], buf[1], buf[2] = vec.Vec2{X: cx - r, Y: cy + kr}, vec.Vec2{X: cx - kr, Y: cy + r}, vec.Vec2{X: cx, Y: cy + r}
		if !yield(path.CmdCubeTo, buf[:3]) {
			return
		}
		buf[0], buf[1], buf[2] = vec.Vec2{X: cx + kr, Y: cy + r}, vec.Vec2{X: cx + r, Y: cy + kr}, vec.Vec2{X: cx + r, Y: cy}
		if !yield(path.CmdCubeTo, buf[:3]) {
			return
		}
		buf[0], buf[1], buf[2] = vec.Vec2{X: cx + r, Y: cy - kr}, vec.Vec2{X: cx + kr, Y: cy - r}, vec.Vec2{X: cx, Y: cy - r}
		if !yield(path.CmdCubeTo, buf[:3]) {
			return
		}
	} else {
		// Start at top, go counter-clockwise
		buf[0] = vec.Vec2{X: cx, Y: cy - r}
		if !yield(path.CmdMoveTo, buf[:1]) {
			return
		}
		buf[0], buf[1], buf[2] = vec.Vec2{X: cx + kr, Y: cy - r}, vec.Vec2{X: cx + r, Y: cy - kr}, vec.Vec2{X: cx + r, Y: cy}
		if !yield(path.CmdCubeTo, buf[:3]) {
			return
		}
		buf[0], buf[1], buf[2] = vec.Vec2{X: cx + r, Y: cy + kr}, vec.Vec2{X: cx + kr, Y: cy + r}, vec.Vec2{X: cx, Y: cy + r}
		if !yield(path.CmdCubeTo, buf[:3]) {
			return
		}
		buf[0], buf[1], buf[2] = vec.Vec2{X: cx - kr, Y: cy + r}, vec.Vec2{X: cx - r, Y: cy + kr}, vec.Vec2{X: cx - r, Y: cy}
		if !yield(path.CmdCubeTo, buf[:3]) {
			return
		}
		buf[0], buf[1], buf[2] = vec.Vec2{X: cx - r, Y: cy - kr}, vec.Vec2{X: cx - kr, Y: cy - r}, vec.Vec2{X: cx, Y: cy - r}
		if !yield(path.CmdCubeTo, buf[:3]) {
			return
		}
	}
	yield(path.CmdClose, nil)
}

// addCircleToVector adds a circle to a vector.Rasterizer using cubic Bézier curves.
func addCircleToVector(r *vector.Rasterizer, cx, cy, radius float32, clockwise bool) {
	const k = float32(0.5522847498)
	kr := k * radius

	if clockwise {
		r.MoveTo(cx, cy-radius)
		r.CubeTo(cx-kr, cy-radius, cx-radius, cy-kr, cx-radius, cy)
		r.CubeTo(cx-radius, cy+kr, cx-kr, cy+radius, cx, cy+radius)
		r.CubeTo(cx+kr, cy+radius, cx+radius, cy+kr, cx+radius, cy)
		r.CubeTo(cx+radius, cy-kr, cx+kr, cy-radius, cx, cy-radius)
	} else {
		r.MoveTo(cx, cy-radius)
		r.CubeTo(cx+kr, cy-radius, cx+radius, cy-kr, cx+radius, cy)
		r.CubeTo(cx+radius, cy+kr, cx+kr, cy+radius, cx, cy+radius)
		r.CubeTo(cx-kr, cy+radius, cx-radius, cy+kr, cx-radius, cy)
		r.CubeTo(cx-radius, cy-kr, cx-kr, cy-radius, cx, cy-radius)
	}
	r.ClosePath()
}
