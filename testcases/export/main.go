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

// Command export writes test case definitions to JSON for the Python reference generator.
// Run from the go-render module root directory.
package main

import (
	"encoding/json"
	"maps"
	"os"
	"slices"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/render/testcases"
)

func main() {
	var out struct {
		TestCases []jsonTestCase `json:"testcases"`
	}

	for _, category := range slices.Sorted(maps.Keys(testcases.All)) {
		for _, tc := range testcases.All[category] {
			out.TestCases = append(out.TestCases, toJSON(category, tc))
		}
	}

	f, err := os.Create("testdata/testcases.json")
	if err != nil {
		panic(err)
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		f.Close()
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
}

type jsonTestCase struct {
	Name       string        `json:"name"`
	Width      int           `json:"width"`
	Height     int           `json:"height"`
	Path       []jsonSegment `json:"path"`
	Op         string        `json:"op"`
	FillRule   string        `json:"fill_rule,omitempty"`
	LineWidth  float64       `json:"line_width,omitempty"`
	LineCap    string        `json:"line_cap,omitempty"`
	LineJoin   string        `json:"line_join,omitempty"`
	MiterLimit float64       `json:"miter_limit,omitempty"`
	Dash       []float64     `json:"dash,omitempty"`
	DashPhase  float64       `json:"dash_phase,omitempty"`
	CTM        []float64     `json:"ctm,omitempty"`
}

type jsonSegment struct {
	Cmd string      `json:"cmd"`
	Pts [][]float64 `json:"pts"`
}

func toJSON(category string, tc testcases.TestCase) jsonTestCase {
	jtc := jsonTestCase{
		Name:   category + "_" + tc.Name,
		Width:  tc.Width,
		Height: tc.Height,
		Path:   pathToJSON(tc.Path),
	}

	switch op := tc.Op.(type) {
	case testcases.Fill:
		jtc.Op = "fill"
		if op.Rule == testcases.EvenOdd {
			jtc.FillRule = "evenodd"
		} else {
			jtc.FillRule = "nonzero"
		}
	case testcases.Stroke:
		jtc.Op = "stroke"
		jtc.LineWidth = op.Width
		jtc.LineCap = op.Cap.String()
		jtc.LineJoin = op.Join.String()
		jtc.MiterLimit = op.MiterLimit
		jtc.Dash = op.Dash
		jtc.DashPhase = op.DashPhase
	}

	// Only serialize non-trivial CTMs (zero-value and identity both omitted)
	if tc.CTM != (matrix.Matrix{}) && tc.CTM != matrix.Identity {
		jtc.CTM = tc.CTM[:]
	}

	return jtc
}

func pathToJSON(p *path.Data) []jsonSegment {
	var segs []jsonSegment
	for cmd, pts := range p.Iter() {
		seg := jsonSegment{Pts: make([][]float64, len(pts))}
		switch cmd {
		case path.CmdMoveTo:
			seg.Cmd = "M"
		case path.CmdLineTo:
			seg.Cmd = "L"
		case path.CmdQuadTo:
			seg.Cmd = "Q"
		case path.CmdCubeTo:
			seg.Cmd = "C"
		case path.CmdClose:
			seg.Cmd = "Z"
		}
		for i, pt := range pts {
			seg.Pts[i] = []float64{pt.X, pt.Y}
		}
		segs = append(segs, seg)
	}
	return segs
}
