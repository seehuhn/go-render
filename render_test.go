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
				RenderExample(tc, actual, w, h, w)

				// compare
				if err := compareImages(name, ref, actual, w, h); err != nil {
					t.Error(err)
				}
			})
		}
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
	const tolerance = 2
	const maxDiffPercent = 10

	total := w * h
	diffCount := 0
	hasDiff := false

	for i := range total {
		e, a := int(expected[i]), int(actual[i])
		diff := e - a
		if diff < 0 {
			diff = -diff
		}
		if diff > 0 {
			hasDiff = true
			if diff > tolerance {
				diffCount++
			}
		}
	}

	maxAllowed := total * maxDiffPercent / 100
	if diffCount > maxAllowed || hasDiff {
		writeDiffImage(name, expected, actual, w, h)
	}
	if diffCount > maxAllowed {
		return fmt.Errorf("%d pixels differ by >%d (max allowed: %d)",
			diffCount, tolerance, maxAllowed)
	}
	return nil
}

func writeDiffImage(name string, expected, actual []byte, w, h int) {
	os.MkdirAll("debug", 0755)

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			i := y*w + x
			img.Set(x, y, color.RGBA{
				R: expected[i], // expected in red
				G: actual[i],   // actual in green
				B: 0,
				A: 255,
			})
		}
	}

	f, err := os.Create(filepath.Join("debug", name+".png"))
	if err != nil {
		return
	}
	defer f.Close()
	png.Encode(f, img)
}
