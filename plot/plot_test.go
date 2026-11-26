// SPDX-License-Identifier: AGPL-3.0-only

package plot

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func wellFormed(t *testing.T, svg []byte) {
	t.Helper()
	d := xml.NewDecoder(bytes.NewReader(svg))
	for {
		if _, err := d.Token(); err == io.EOF {
			return
		} else if err != nil {
			t.Fatalf("SVG is not well-formed XML: %v", err)
		}
	}
}

func TestSVGWellFormedLinearAndLog(t *testing.T) {
	x := []float64{0, 10, 20, 30}
	lin := Chart{Title: "T <&>", XLabel: "x", YLabel: "y", Series: []Series{
		{Label: "a\nb", Role: "data", X: x, Y: []float64{1, 2, 4, 8}},
		{Label: "ref", Role: "ref", X: x, Y: []float64{1, 2, 3, 4}},
	}, HLines: []HLine{{Y: 5, Label: "limit"}}}
	wellFormed(t, lin.SVG())

	logc := Chart{Title: "log", LogY: true, YFormat: SI, Series: []Series{
		{Label: "d", Role: "data", X: x, Y: []float64{1e6, 1e5, 0, 1e3}}, // zero must be skipped
	}}
	svg := logc.SVG()
	wellFormed(t, svg)
	if strings.Contains(string(svg), "NaN") || strings.Contains(string(svg), "Inf") {
		t.Error("SVG contains NaN/Inf coordinates")
	}
}

func TestEmptyChartDoesNotPanic(t *testing.T) {
	wellFormed(t, Chart{Title: "empty"}.SVG())
	wellFormed(t, Chart{Title: "empty log", LogY: true}.SVG())
}

func TestNiceTicks(t *testing.T) {
	got := niceTicks(0, 120, 7)
	if got[0] != 0 || got[len(got)-1] < 120 || len(got) > 12 {
		t.Errorf("ticks %v", got)
	}
}
