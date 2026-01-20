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
	"fmt"
	"image"
	"image/color"
	"image/png"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/render/testcases"
)

func TestAgainstReference(t *testing.T) {
	for _, category := range slices.Sorted(maps.Keys(testcases.All)) {
		for _, tc := range testcases.All[category] {
			name := category + "_" + tc.Name
			t.Run(name, func(t *testing.T) {
				// load reference image
				refPath := filepath.Join("testdata", "reference", name+".png")
				ref, err := loadGray(refPath)
				if err != nil {
					t.Fatalf("loading reference: %v", err)
				}

				// allocate output buffer
				w, h := tc.Width, tc.Height
				actual := make([]byte, w*h)

				// render
				renderExample(tc, actual, w, h, w)

				// compare
				if err := compareImages(name, ref, actual, w, h); err != nil {
					t.Error(err)
				}
			})
		}
	}
}

// renderExample renders a test case into a grayscale buffer.
// The buffer is pre-initialized with zeros, in row-major order.
// Each byte represents coverage from 0 (transparent) to 255 (opaque).
func renderExample(tc testcases.TestCase, buf []byte, width, height, stride int) {
	clip := rect.Rect{
		LLx: 0,
		LLy: 0,
		URx: float64(width),
		URy: float64(height),
	}
	r := NewRasteriser(clip)

	// Apply CTM (zero-value means identity, which is already the default)
	if tc.CTM != (matrix.Matrix{}) {
		r.CTM = tc.CTM
	}

	// Emit callback: convert float32 coverage to bytes
	emit := func(y, xMin int, coverage []float32) {
		row := buf[y*stride:]
		for i, c := range coverage {
			row[xMin+i] = byte(max(0, min(255, int(c*256))))
		}
	}

	// Dispatch based on operation type
	switch op := tc.Op.(type) {
	case testcases.Fill:
		if op.Rule == testcases.EvenOdd {
			r.FillEvenOdd(tc.Path, emit)
		} else {
			r.FillNonZero(tc.Path, emit)
		}
	case testcases.Stroke:
		r.Width = op.Width
		r.Cap = op.Cap
		r.Join = op.Join
		r.MiterLimit = op.MiterLimit
		r.Dash = op.Dash
		r.DashPhase = op.DashPhase
		r.Stroke(tc.Path, emit)
	}
}

func loadGray(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	gray := make([]byte, w*h)

	for y := range h {
		for x := range w {
			c := color.GrayModel.Convert(img.At(x+bounds.Min.X, y+bounds.Min.Y)).(color.Gray)
			gray[y*w+x] = c.Y
		}
	}
	return gray, nil
}

func compareImages(name string, expected, actual []byte, w, h int) error {
	const tolerance = 72
	const maxDiffPercent = 10

	total := w * h
	diffCount := 0      // pixels with any difference
	outOfTolerance := 0 // pixels differing by more than tolerance

	for i := range total {
		e, a := int(expected[i]), int(actual[i])
		diff := e - a
		if diff < 0 {
			diff = -diff
		}
		if diff > 0 {
			diffCount++
			if diff > tolerance {
				outOfTolerance++
			}
		}
	}

	maxAllowed := total * maxDiffPercent / 100
	if diffCount > 0 {
		writeDiffImage(name, expected, actual, w, h)
	}
	if outOfTolerance > 0 {
		// Find max difference for debugging
		maxDiff := 0
		for i := range total {
			e, a := int(expected[i]), int(actual[i])
			diff := e - a
			if diff < 0 {
				diff = -diff
			}
			if diff > maxDiff {
				maxDiff = diff
			}
		}
		return fmt.Errorf("%d pixels differ by >%d (max diff: %d)",
			outOfTolerance, tolerance, maxDiff)
	}
	if diffCount > maxAllowed {
		return fmt.Errorf("%d pixels differ (max allowed: %d, i.e. %d%%)",
			diffCount, maxAllowed, maxDiffPercent)
	}
	return nil
}

func writeDiffImage(name string, expected, actual []byte, w, h int) {
	os.MkdirAll("debug", 0755)

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			i := y*w + x
			diff := int(expected[i]) - int(actual[i])
			var c color.RGBA
			if diff > 0 {
				// Under-producing (expected > actual): green
				c = color.RGBA{R: 0, G: uint8(diff), B: 0, A: 255}
			} else if diff < 0 {
				// Over-producing (expected < actual): red
				c = color.RGBA{R: uint8(-diff), G: 0, B: 0, A: 255}
			} else {
				// No difference: black
				c = color.RGBA{R: 0, G: 0, B: 0, A: 255}
			}
			img.Set(x, y, c)
		}
	}

	f, err := os.Create(filepath.Join("debug", name+".png"))
	if err != nil {
		return
	}
	defer f.Close()
	png.Encode(f, img)
}
