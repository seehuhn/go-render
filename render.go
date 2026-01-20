// Package render implements a 2D rasterizer for the PDF/PostScript imaging model.
package render

//go:generate go run ./testcases/export
//go:generate python3 tools/generate_references.py

import "seehuhn.de/go/render/testcases"

// RenderExample renders a test case into a grayscale buffer.
// The buffer is pre-initialized with zeros, in row-major order.
// Each byte represents coverage from 0 (transparent) to 255 (opaque).
func RenderExample(tc testcases.TestCase, buf []byte, width, height, stride int) {
	// TODO: implement
}
