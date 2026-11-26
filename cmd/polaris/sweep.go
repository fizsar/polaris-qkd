// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"

	"github.com/fizsar/polaris-qkd/plot"
	"github.com/fizsar/polaris-qkd/protocol"
)

type point struct {
	x                                float64
	detected, sifted, leak, final    float64
	qber, qberUpper                  float64
	ratePulse, rateBps, abortedShare float64
}

func sweep(args []string, stdout, stderr io.Writer) error {
	c := newCommon("sweep", stderr)
	vary := c.fs.String("vary", "distance", "parameter to sweep: distance, noise or eve")
	from := c.fs.Float64("from", math.NaN(), "first value (default depends on --vary)")
	to := c.fs.Float64("to", math.NaN(), "last value (default depends on --vary)")
	steps := c.fs.Int("steps", 11, "number of points")
	runs := c.fs.Int("runs", 1, "runs averaged per point")
	out := c.fs.String("out", "", "CSV output file (default: stdout)")
	svg := c.fs.String("svg", "", "also write an SVG chart to this file")
	if err := c.parse(args); err != nil {
		return err
	}
	def, ok := map[string][2]float64{
		"distance": {0, 100}, "noise": {0, 0.15}, "eve": {0, 1},
	}[*vary]
	if !ok {
		return fmt.Errorf("--vary must be distance, noise or eve, got %q", *vary)
	}
	if math.IsNaN(*from) {
		*from = def[0]
	}
	if math.IsNaN(*to) {
		*to = def[1]
	}
	if *steps < 2 || *runs < 1 || *to <= *from {
		return fmt.Errorf("need --steps >= 2, --runs >= 1 and --to > --from")
	}

	pts := make([]point, *steps)
	for j := range pts {
		x := *from + (*to-*from)*float64(j)/float64(*steps-1)
		cfg := c.cfg
		switch *vary {
		case "distance":
			cfg.Channel.DistanceKm = x
		case "noise":
			cfg.Channel.ErrorRate = x
		case "eve":
			cfg.Channel.EveFraction = x
		}
		if err := cfg.Validate(); err != nil {
			return err
		}
		p := point{x: x}
		for i := 0; i < *runs; i++ {
			r, err := protocol.Run(cfg, c.source(j, i))
			if err != nil {
				return err
			}
			n := float64(*runs)
			p.detected += float64(r.PhotonsDetected) / n
			p.sifted += float64(r.Sifted) / n
			p.leak += float64(r.LeakEC) / n
			p.final += float64(r.FinalBits) / n
			p.qber += r.QBER / n
			p.qberUpper += r.QBERUpper / n
			p.ratePulse += r.KeyRatePerPulse / n
			p.rateBps += r.KeyRatePerSecond / n
			if r.Aborted {
				p.abortedShare += 1 / n
			}
		}
		pts[j] = p
	}

	w := stdout
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	cw := csv.NewWriter(w)
	g := func(v float64) string { return strconv.FormatFloat(v, 'g', 6, 64) }
	_ = cw.Write([]string{*vary, "photons_detected", "sifted_bits", "qber", "qber_upper", "leak_ec",
		"final_key_bits", "key_rate_per_pulse", "key_rate_bps", "aborted_fraction"})
	for _, p := range pts {
		_ = cw.Write([]string{g(p.x), g(p.detected), g(p.sifted), g(p.qber), g(p.qberUpper), g(p.leak),
			g(p.final), g(p.ratePulse), g(p.rateBps), g(p.abortedShare)})
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return err
	}
	if *svg != "" {
		return os.WriteFile(*svg, chart(c, *vary, pts).SVG(), 0o644)
	}
	return nil
}

// chart builds the figure that fits the swept parameter.
func chart(c *common, vary string, pts []point) plot.Chart {
	cfg := c.cfg
	sub := fmt.Sprintf("Simulation · %s photons/point · detector %.0f %% · noise %.1f %%",
		si(float64(cfg.Photons)), 100*cfg.Channel.DetectorEfficiency, 100*cfg.Channel.ErrorRate)
	foot := "SIMULATION — no quantum hardware. Asymptotic single-photon bound; no finite-key effects, no decoy states."
	var xs []float64
	for _, p := range pts {
		xs = append(xs, p.x)
	}
	col := func(f func(point) float64) []float64 {
		v := make([]float64, len(pts))
		for i, p := range pts {
			v[i] = f(p)
		}
		return v
	}

	switch vary {
	case "distance":
		rate := col(func(p point) float64 { return p.rateBps })
		// Reference: fibre loss alone, anchored at the first non-zero point.
		var ref []float64
		var y0, x0 float64
		for i, r := range rate {
			if r > 0 {
				y0, x0 = r, xs[i]
				break
			}
		}
		lastX := x0
		for i, r := range rate {
			if r > 0 {
				lastX = xs[i]
			}
		}
		for _, x := range xs {
			if x <= lastX { // same x range as the simulated points
				ref = append(ref, y0*math.Pow(10, -cfg.Channel.AttenuationDBPerKm*(x-x0)/10))
			} else {
				ref = append(ref, 0) // skipped by the log axis
			}
		}
		return plot.Chart{
			Title:    "Secret key rate vs distance",
			Subtitle: sub + fmt.Sprintf(" · %.0f GHz clock", cfg.ClockHz/1e9),
			XLabel:   "Fibre length (km)", YLabel: "Secret key rate (bit/s, log scale)",
			LogY: true, YFormat: plot.SI, Footer: foot,
			Series: []plot.Series{
				{Label: "Simulated\nkey rate", Role: "data", X: xs, Y: rate},
				{Label: "∝ fibre loss η", Role: "ref", X: xs, Y: ref},
			},
		}
	default: // eve, noise: QBER in %
		scale := 100.0
		qb := col(func(p point) float64 { return 100 * p.qber })
		var theory []float64
		label, xlabel, tl := "", "", ""
		xp := make([]float64, len(xs))
		for i, x := range xs {
			xp[i] = x * scale
			if vary == "eve" {
				p := cfg.Channel.ErrorRate
				theory = append(theory, 100*(p+(1-2*p)*0.25*x))
			} else {
				theory = append(theory, 100*x+100*0) // Q = p (plus negligible dark counts)
			}
		}
		if vary == "eve" {
			label, xlabel, tl = "QBER vs eavesdropped fraction", "Photons intercepted by Eve (%)", "theory p + (1−2p)·f/4"
		} else {
			label, xlabel, tl = "QBER vs intrinsic noise", "Intrinsic error rate (%)", "theory Q = p"
		}
		return plot.Chart{
			Title: label, Subtitle: sub,
			XLabel: xlabel, YLabel: "QBER (%)", Footer: foot,
			YMin: 0, YMax: 30,
			YFormat: func(v float64) string { return fmt.Sprintf("%g", v) },
			HLines:  []plot.HLine{{Y: 100 * cfg.QBERThreshold, Label: fmt.Sprintf("%.0f %% BB84\nthreshold", 100*cfg.QBERThreshold)}},
			Series: []plot.Series{
				{Label: "Simulated\nQBER", Role: "data", X: xp, Y: qb},
				{Label: tl, Role: "ref", X: xp, Y: theory},
			},
		}
	}
}

func si(v float64) string {
	switch {
	case v >= 1e6:
		return strconv.FormatFloat(v/1e6, 'g', 4, 64) + "M"
	case v >= 1e3:
		return strconv.FormatFloat(v/1e3, 'g', 4, 64) + "k"
	}
	return strconv.FormatFloat(v, 'g', 4, 64)
}
