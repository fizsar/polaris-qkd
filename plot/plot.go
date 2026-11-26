// SPDX-License-Identifier: AGPL-3.0-only

// Package plot renders small, dependency-free SVG line charts for the README.
//
// Design: thin 2 px lines, ≥ 7 px markers with a surface ring, a recessive
// grid, direct labels instead of a legend box (one data series per chart),
// and light/dark colours selected with prefers-color-scheme inside the SVG.
package plot

import (
	"fmt"
	"html"
	"math"
	"strings"
)

// Series is one polyline. Role "data" is drawn in the accent colour with
// markers; "ref" is a dashed neutral reference curve.
type Series struct {
	Label string
	Role  string // "data" or "ref"
	X, Y  []float64
}

// HLine is a horizontal reference line (e.g. a threshold).
type HLine struct {
	Y     float64
	Label string
}

// Chart describes one figure.
type Chart struct {
	Title, Subtitle string
	XLabel, YLabel  string
	LogY            bool
	XMin, XMax      float64 // XMax <= XMin → auto
	YMin, YMax      float64 // YMax <= YMin → auto
	YFormat         func(float64) string
	Series          []Series
	HLines          []HLine
	Footer          string
}

const (
	width, height = 760.0, 440.0
	ml, mr        = 76.0, 150.0
	mt, mb        = 78.0, 64.0
)

const style = `
:root{--bg:#ffffff;--fg:#191919;--mut:#666666;--grid:#e6e6e6;--axis:#b3b3b3;--accent:#257de0;--ref:#8a8a8a;--crit:#dc3c3d}
@media (prefers-color-scheme: dark){:root{--bg:#191919;--fg:#efefef;--mut:#9c9c9c;--grid:#2e2e2e;--axis:#555555;--accent:#338ced;--ref:#8f8f8f;--crit:#dc3c3d}}
text{font-family:system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;fill:var(--fg)}
.mut{fill:var(--mut)}.t{font-size:17px;font-weight:600}.s{font-size:12.5px}.tk{font-size:12px}.lb{font-size:12.5px;font-weight:600}
.grid{stroke:var(--grid);stroke-width:1}.axis{stroke:var(--axis);stroke-width:1}
.data{fill:none;stroke:var(--accent);stroke-width:2;stroke-linejoin:round;stroke-linecap:round}
.ref{fill:none;stroke:var(--ref);stroke-width:1.5;stroke-dasharray:6 4}
.crit{stroke:var(--crit);stroke-width:1.5;stroke-dasharray:2 4}
.dot{fill:var(--accent);stroke:var(--bg);stroke-width:2}
.accent{fill:var(--accent)}.reft{fill:var(--ref)}.critt{fill:var(--crit)}
`

// SVG renders the chart.
func (c Chart) SVG() []byte {
	// Collect plottable points to find the data ranges.
	xmin, xmax := math.Inf(1), math.Inf(-1)
	ymin, ymax := math.Inf(1), math.Inf(-1)
	for _, s := range c.Series {
		for i := range s.X {
			if c.LogY && s.Y[i] <= 0 {
				continue
			}
			xmin, xmax = math.Min(xmin, s.X[i]), math.Max(xmax, s.X[i])
			ymin, ymax = math.Min(ymin, s.Y[i]), math.Max(ymax, s.Y[i])
		}
	}
	for _, h := range c.HLines {
		ymin, ymax = math.Min(ymin, h.Y), math.Max(ymax, h.Y)
	}
	if math.IsInf(xmin, 0) {
		xmin, xmax, ymin, ymax = 0, 1, c.logFloor(), 1
	}
	if c.XMax > c.XMin {
		xmin, xmax = c.XMin, c.XMax
	}
	var xt, yt []float64
	xt = niceTicks(xmin, xmax, 7)
	xmin, xmax = math.Min(xmin, xt[0]), math.Max(xmax, xt[len(xt)-1])
	if c.LogY {
		lo, hi := math.Floor(math.Log10(ymin)), math.Ceil(math.Log10(ymax))
		if hi == lo {
			hi++
		}
		if c.YMax > c.YMin {
			lo, hi = math.Log10(c.YMin), math.Log10(c.YMax)
		}
		for e := lo; e <= hi+1e-9; e++ {
			yt = append(yt, math.Pow(10, e))
		}
		ymin, ymax = yt[0], yt[len(yt)-1]
	} else {
		if c.YMax > c.YMin {
			ymin, ymax = c.YMin, c.YMax
		} else {
			ymin = math.Min(0, ymin)
		}
		yt = niceTicks(ymin, ymax, 6)
		ymin, ymax = math.Min(ymin, yt[0]), math.Max(ymax, yt[len(yt)-1])
	}

	pw, ph := width-ml-mr, height-mt-mb
	px := func(x float64) float64 { return ml + (x-xmin)/(xmax-xmin)*pw }
	py := func(y float64) float64 {
		if c.LogY {
			return mt + ph - (math.Log10(y)-math.Log10(ymin))/(math.Log10(ymax)-math.Log10(ymin))*ph
		}
		return mt + ph - (y-ymin)/(ymax-ymin)*ph
	}
	yf := c.YFormat
	if yf == nil {
		yf = func(v float64) string { return fmt.Sprintf("%g", v) }
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %g %g" width="%g" height="%g" role="img" aria-label="%s">`,
		width, height, width, height, html.EscapeString(c.Title+". "+c.Subtitle))
	fmt.Fprintf(&b, "<style>%s</style>", style)
	fmt.Fprintf(&b, `<rect width="%g" height="%g" rx="8" fill="var(--bg)"/>`, width, height)
	fmt.Fprintf(&b, `<text class="t" x="%g" y="30">%s</text>`, ml, html.EscapeString(c.Title))
	fmt.Fprintf(&b, `<text class="s mut" x="%g" y="50">%s</text>`, ml, html.EscapeString(c.Subtitle))

	// Grid and ticks.
	for _, v := range yt {
		y := py(v)
		fmt.Fprintf(&b, `<line class="grid" x1="%g" x2="%g" y1="%.1f" y2="%.1f"/>`, ml, ml+pw, y, y)
		fmt.Fprintf(&b, `<text class="tk mut" x="%g" y="%.1f" text-anchor="end">%s</text>`, ml-8, y+4, html.EscapeString(yf(v)))
	}
	for _, v := range xt {
		x := px(v)
		fmt.Fprintf(&b, `<line class="axis" x1="%.1f" x2="%.1f" y1="%g" y2="%g"/>`, x, x, mt+ph, mt+ph+4)
		fmt.Fprintf(&b, `<text class="tk mut" x="%.1f" y="%g" text-anchor="middle">%g</text>`, x, mt+ph+19, v)
	}
	fmt.Fprintf(&b, `<line class="axis" x1="%g" x2="%g" y1="%g" y2="%g"/>`, ml, ml+pw, mt+ph, mt+ph)
	fmt.Fprintf(&b, `<text class="s mut" x="%g" y="%g" text-anchor="middle">%s</text>`, ml+pw/2, height-18, html.EscapeString(c.XLabel))
	fmt.Fprintf(&b, `<text class="s mut" transform="translate(18 %g) rotate(-90)" text-anchor="middle">%s</text>`, mt+ph/2, html.EscapeString(c.YLabel))

	// Threshold lines.
	for _, h := range c.HLines {
		y := py(h.Y)
		fmt.Fprintf(&b, `<line class="crit" x1="%g" x2="%g" y1="%.1f" y2="%.1f"/>`, ml, ml+pw, y, y)
		for i, ln := range strings.Split(h.Label, "\n") {
			fmt.Fprintf(&b, `<text class="lb critt" x="%g" y="%.1f">%s</text>`, ml+pw+8, y+4+float64(i)*15, html.EscapeString(ln))
		}
	}

	// End-of-line label heights of the data series, so that reference labels
	// that would collide with them are moved below.
	var dataEnds []float64
	for _, s := range c.Series {
		if s.Role != "data" {
			continue
		}
		for i := len(s.X) - 1; i >= 0; i-- {
			if !c.LogY || s.Y[i] > 0 {
				dataEnds = append(dataEnds, py(s.Y[i]))
				break
			}
		}
	}

	// Series: references first so data sits on top.
	for pass := 0; pass < 2; pass++ {
		for _, s := range c.Series {
			if (s.Role == "data") != (pass == 1) {
				continue
			}
			var pts []string
			var lastX, lastY float64
			for i := range s.X {
				if c.LogY && s.Y[i] <= 0 {
					continue
				}
				x, y := px(s.X[i]), py(s.Y[i])
				pts = append(pts, fmt.Sprintf("%.1f,%.1f", x, y))
				lastX, lastY = x, y
			}
			if len(pts) == 0 {
				continue
			}
			class, tclass := "ref", "reft"
			if s.Role == "data" {
				class, tclass = "data", "accent"
			}
			fmt.Fprintf(&b, `<polyline class="%s" points="%s"/>`, class, strings.Join(pts, " "))
			if s.Role == "data" {
				for _, p := range pts {
					var x, y float64
					fmt.Sscanf(p, "%f,%f", &x, &y)
					fmt.Fprintf(&b, `<circle class="dot" cx="%.1f" cy="%.1f" r="3.5"/>`, x, y)
				}
			}
			if s.Role != "data" {
				for _, de := range dataEnds {
					if math.Abs(lastY-de) < 36 {
						if de+36 < mt+ph-6 {
							lastY = de + 36
						} else {
							lastY = de - 36
						}
					}
				}
			}
			lines := strings.Split(s.Label, "\n")
			for i, ln := range lines {
				fmt.Fprintf(&b, `<text class="lb %s" x="%.1f" y="%.1f">%s</text>`, tclass, lastX+9, lastY+4+float64(i)*15, html.EscapeString(ln))
			}
		}
	}
	if c.Footer != "" {
		fmt.Fprintf(&b, `<text class="tk mut" x="%g" y="%g">%s</text>`, ml, height-4, html.EscapeString(c.Footer))
	}
	b.WriteString("</svg>\n")
	return []byte(b.String())
}

func (c Chart) logFloor() float64 {
	if c.LogY {
		return 1e-6
	}
	return 0
}

// niceTicks returns round tick values covering [lo, hi] with about n ticks.
func niceTicks(lo, hi float64, n int) []float64 {
	if hi <= lo {
		hi = lo + 1
	}
	raw := (hi - lo) / float64(n)
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	step := mag
	for _, m := range []float64{1, 2, 2.5, 5, 10} {
		step = m * mag
		if raw <= step {
			break
		}
	}
	var t []float64
	for v := math.Floor(lo/step) * step; v <= hi+step*1e-9; v += step {
		t = append(t, math.Round(v/step)*step)
	}
	return t
}

// SI formats a value with k/M/G suffixes, for log axes of rates.
func SI(v float64) string {
	switch {
	case v >= 1e9:
		return fmt.Sprintf("%g G", v/1e9)
	case v >= 1e6:
		return fmt.Sprintf("%g M", v/1e6)
	case v >= 1e3:
		return fmt.Sprintf("%g k", v/1e3)
	case v >= 1:
		return fmt.Sprintf("%g", v)
	}
	return fmt.Sprintf("%g", v)
}
