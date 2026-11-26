// SPDX-License-Identifier: AGPL-3.0-only

package protocol

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/fizsar/polaris-qkd/rng"
)

func cfg(photons int, km, noise, eve float64) Config {
	c := DefaultConfig()
	c.Photons = photons
	c.Channel.DistanceKm = km
	c.Channel.ErrorRate = noise
	c.Channel.EveFraction = eve
	c.Channel.DarkCountProb = 0
	return c
}

func run(t *testing.T, c Config, seed uint64) Result {
	t.Helper()
	r, err := Run(c, rng.NewSeeded(seed))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestCleanLinkGivesIdenticalKeys(t *testing.T) {
	r := run(t, cfg(500_000, 10, 0, 0), 1)
	if r.Aborted || !r.Verified || !r.KeysMatch || r.FinalBits == 0 {
		t.Fatalf("clean run failed: %+v", r)
	}
	if r.QBER != 0 || r.TrueQBER != 0 {
		t.Errorf("QBER = %v / true %v, want 0", r.QBER, r.TrueQBER)
	}
	if len(r.AliceKey) != r.FinalBits || string(r.AliceKey) != string(r.BobKey) {
		t.Error("final keys differ or have the wrong length")
	}
}

func TestNoiseQBERAndKeyMatch(t *testing.T) {
	for _, p := range []float64{0.02, 0.05} {
		r := run(t, cfg(1_000_000, 10, p, 0), 2)
		n := r.SampleBits
		if d := math.Abs(r.QBER - p); d > 4*math.Sqrt(p*(1-p)/float64(n)) {
			t.Errorf("noise %v: estimated QBER %v", p, r.QBER)
		}
		if r.Aborted || !r.KeysMatch {
			t.Errorf("noise %v: aborted=%v match=%v reason=%q", p, r.Aborted, r.KeysMatch, r.AbortReason)
		}
	}
}

func TestEveTotalAborts(t *testing.T) {
	r := run(t, cfg(500_000, 5, 0, 1), 3)
	if !r.Aborted || r.AbortCode != AbortQBERTooHigh {
		t.Fatalf("Eve at 100%% should abort with %q, got %+v", AbortQBERTooHigh, r.AbortCode)
	}
	if math.Abs(r.QBER-0.25) > 0.02 {
		t.Errorf("QBER = %v, want ≈ 0.25", r.QBER)
	}
	if r.FinalBits != 0 || r.AliceKey != nil {
		t.Error("an aborted run must not return key material")
	}
}

func TestAbortThresholdAround11Percent(t *testing.T) {
	below := run(t, cfg(1_000_000, 5, 0.07, 0), 4)
	if below.Aborted && below.AbortCode == AbortQBERTooHigh {
		t.Errorf("7%% noise must not trip the 11%% threshold: %s", below.AbortReason)
	}
	above := run(t, cfg(1_000_000, 5, 0.14, 0), 4)
	if !above.Aborted || above.AbortCode != AbortQBERTooHigh {
		t.Errorf("14%% noise must abort for QBER, got %+v", above.AbortCode)
	}
	if above.AbortReason == "" {
		t.Error("abort must give a reason")
	}
}

func TestPartialEve(t *testing.T) {
	// f = 0.3 → QBER ≈ 7.5 %: still below threshold, key survives (smaller).
	r := run(t, cfg(1_000_000, 5, 0, 0.3), 5)
	if math.Abs(r.QBER-0.075) > 0.015 {
		t.Errorf("QBER = %v, want ≈ 0.075", r.QBER)
	}
	if r.Aborted {
		t.Errorf("unexpected abort: %s", r.AbortReason)
	}
	// f = 0.6 → 15 %: abort.
	if r := run(t, cfg(1_000_000, 5, 0, 0.6), 5); !r.Aborted {
		t.Error("Eve at 60% (QBER 15%) must abort")
	}
}

func TestSiftingRatio(t *testing.T) {
	r := run(t, cfg(1_000_000, 10, 0.01, 0), 6)
	if math.Abs(r.SiftRatio-0.5) > 4*math.Sqrt(0.25/float64(r.PhotonsDetected)) {
		t.Errorf("sift ratio %v, want 0.5", r.SiftRatio)
	}
}

func TestKeyRateTracksTransmittance(t *testing.T) {
	// With no dark counts the sifted length is proportional to η, so the
	// final rate between two distances must follow the ratio of η (fixed
	// costs — 64-bit margins — are well below 1 % at these sizes).
	a := run(t, cfg(4_000_000, 5, 0.01, 0), 7)
	b := run(t, cfg(4_000_000, 20, 0.01, 0), 7)
	got := b.KeyRatePerPulse / a.KeyRatePerPulse
	want := b.Config.Channel.Transmittance() / a.Config.Channel.Transmittance()
	if math.Abs(got-want)/want > 0.05 {
		t.Errorf("rate ratio %v, want η ratio %v (±5%%)", got, want)
	}
	if !(b.KeyRatePerPulse < a.KeyRatePerPulse) {
		t.Error("rate must fall with distance")
	}
}

func TestBlockwiseLosesLittle(t *testing.T) {
	one := cfg(1_000_000, 0, 0.02, 0)
	one.Channel.DetectorEfficiency = 0.3
	one.BlockBits = 0
	blk := one
	blk.BlockBits = 1 << 14
	a, b := run(t, one, 8), run(t, blk, 8)
	if a.FinalBits == 0 || b.FinalBits > a.FinalBits {
		t.Fatalf("single %d, blocks %d", a.FinalBits, b.FinalBits)
	}
	if loss := float64(a.FinalBits-b.FinalBits) / float64(a.FinalBits); loss > 0.02 {
		t.Errorf("block-wise amplification loses %.2f%% of the key, expected < 2%%", 100*loss)
	}
	if !b.KeysMatch {
		t.Error("block-wise keys differ")
	}
}

func TestTooFewDetections(t *testing.T) {
	r := run(t, cfg(1000, 100, 0.01, 0), 9)
	if !r.Aborted || r.AbortCode != AbortSiftedTooShort {
		t.Fatalf("got %+v", r.AbortCode)
	}
}

func TestReproducible(t *testing.T) {
	c := cfg(300_000, 20, 0.02, 0.1)
	a, b := run(t, c, 11), run(t, c, 11)
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) || string(a.AliceKey) != string(b.AliceKey) {
		t.Fatal("same seed must reproduce the whole run")
	}
}

func TestInvalidConfig(t *testing.T) {
	c := cfg(1000, 0, 0, 0)
	c.Channel.EveFraction = 1.5
	if _, err := Run(c, rng.NewSeeded(1)); err == nil {
		t.Fatal("expected a validation error")
	}
}

func TestNoticePresent(t *testing.T) {
	if r := run(t, cfg(100_000, 5, 0.01, 0), 1); r.Notice == "" {
		t.Fatal("every result must carry the simulation notice")
	}
}
