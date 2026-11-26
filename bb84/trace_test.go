// SPDX-License-Identifier: AGPL-3.0-only

package bb84

import (
	"math"
	"testing"

	"github.com/fizsar/polaris-qkd/channel"
	"github.com/fizsar/polaris-qkd/rng"
)

func TestTraceMatchesTheory(t *testing.T) {
	cfg := ideal()
	cfg.EveFraction = 1
	ph := Trace(cfg, 200_000, rng.NewSeeded(1), false)
	var sifted, errs, inter int
	for _, p := range ph {
		if p.Intercepted {
			inter++
		}
		if p.Sifted {
			sifted++
			if p.Error {
				errs++
			}
		}
	}
	if inter != len(ph) {
		t.Fatalf("Eve at 100%% intercepted %d of %d", inter, len(ph))
	}
	q := float64(errs) / float64(sifted)
	if math.Abs(q-0.25) > 4*math.Sqrt(0.25*0.75/float64(sifted)) {
		t.Errorf("trace QBER %v, want 0.25", q)
	}
}

func TestTraceOnlyDetected(t *testing.T) {
	cfg := channel.Default()
	cfg.DistanceKm = 50
	ph := Trace(cfg, 100, rng.NewSeeded(3), true)
	if len(ph) != 100 {
		t.Fatalf("got %d records, want 100", len(ph))
	}
	for _, p := range ph {
		if !p.Detected {
			t.Fatal("onlyDetected returned an undetected photon")
		}
	}
}

func TestTraceLosses(t *testing.T) {
	cfg := channel.Default()
	cfg.DistanceKm = 50 // η·ε = 0.01
	ph := Trace(cfg, 100_000, rng.NewSeeded(2), false)
	d := 0
	for _, p := range ph {
		if p.Detected {
			d++
		}
	}
	if got := float64(d) / float64(len(ph)); math.Abs(got-0.01) > 4*math.Sqrt(0.01*0.99/float64(len(ph))) {
		t.Errorf("detected fraction %v, want ≈ 0.01", got)
	}
}
