// SPDX-License-Identifier: AGPL-3.0-only

package bb84

import (
	"math"
	"testing"

	"github.com/fizsar/polaris-qkd/channel"
	"github.com/fizsar/polaris-qkd/rng"
)

// ideal is a lossless, noiseless link with a perfect detector.
func ideal() channel.Config {
	return channel.Config{AttenuationDBPerKm: 0.2, DetectorEfficiency: 1}
}

func siftedQBER(cfg channel.Config, n int, seed uint64) (qber float64, sifted int, detected int) {
	r := Exchange(cfg, n, rng.NewSeeded(seed))
	a, b := Sift(r)
	errs := 0
	for i := range a {
		if a[i] != b[i] {
			errs++
		}
	}
	return float64(errs) / float64(len(a)), len(a), r.Stats.Detected
}

// tol returns 4 standard deviations of a binomial proportion.
func tol(p float64, n int) float64 { return 4 * math.Sqrt(p*(1-p)/float64(n)) }

func TestNoEveNoNoiseQBERZero(t *testing.T) {
	q, n, _ := siftedQBER(ideal(), 100_000, 1)
	if q != 0 {
		t.Fatalf("QBER = %v with an ideal link, want exactly 0 (sifted %d)", q, n)
	}
}

func TestIntrinsicNoiseGivesQBER(t *testing.T) {
	for _, p := range []float64{0.01, 0.03, 0.08} {
		cfg := ideal()
		cfg.ErrorRate = p
		q, n, _ := siftedQBER(cfg, 400_000, 2)
		if d := math.Abs(q - p); d > tol(p, n) {
			t.Errorf("noise %v: QBER = %v (|Δ| = %v > %v)", p, q, d, tol(p, n))
		}
	}
}

func TestEveFullInterceptGivesQuarter(t *testing.T) {
	cfg := ideal()
	cfg.EveFraction = 1
	q, n, _ := siftedQBER(cfg, 400_000, 3)
	if d := math.Abs(q - 0.25); d > tol(0.25, n) {
		t.Fatalf("QBER = %v, want 0.25 ± %v", q, tol(0.25, n))
	}
}

func TestEvePartialInterceptGivesQuarterTimesF(t *testing.T) {
	for _, f := range []float64{0.2, 0.5, 0.8} {
		cfg := ideal()
		cfg.EveFraction = f
		q, n, _ := siftedQBER(cfg, 400_000, 4)
		want := 0.25 * f
		if d := math.Abs(q - want); d > tol(want, n) {
			t.Errorf("f=%v: QBER = %v, want %v ± %v", f, q, want, tol(want, n))
		}
	}
}

func TestSiftKeepsHalf(t *testing.T) {
	cfg := channel.Default()
	cfg.DistanceKm = 10
	_, sifted, detected := siftedQBER(cfg, 2_000_000, 5)
	ratio := float64(sifted) / float64(detected)
	if d := math.Abs(ratio - 0.5); d > tol(0.5, detected) {
		t.Fatalf("sift ratio = %v (%d of %d), want 0.5 ± %v", ratio, sifted, detected, tol(0.5, detected))
	}
}

func TestDetectionRateFollowsTransmittance(t *testing.T) {
	cfg := channel.Default()
	cfg.DarkCountProb = 0
	cfg.DistanceKm = 25
	n := 2_000_000
	r := Exchange(cfg, n, rng.NewSeeded(6))
	p := cfg.PhotonDetectionProb()
	got := float64(r.Stats.Detected) / float64(n)
	if d := math.Abs(got - p); d > tol(p, n) {
		t.Fatalf("detection rate = %v, want η·ε = %v ± %v", got, p, tol(p, n))
	}
}

func TestDarkCountsAreRandomBits(t *testing.T) {
	// No photon ever arrives (tiny detection probability is not allowed, so use
	// a very long fibre): every click is a dark count, whose bit is random, so
	// the sifted QBER of dark clicks is 50 %.
	cfg := ideal()
	cfg.DistanceKm = 5000
	cfg.DarkCountProb = 0.05
	q, n, _ := siftedQBER(cfg, 2_000_000, 7)
	if d := math.Abs(q - 0.5); d > tol(0.5, n) {
		t.Fatalf("dark-count QBER = %v over %d bits, want 0.5", q, n)
	}
}

func TestReproducibleWithSeed(t *testing.T) {
	cfg := channel.Default()
	cfg.EveFraction = 0.3
	a := Exchange(cfg, 50_000, rng.NewSeeded(9))
	b := Exchange(cfg, 50_000, rng.NewSeeded(9))
	if a.Stats != b.Stats || string(a.BobBit) != string(b.BobBit) {
		t.Fatal("same seed gave different runs")
	}
}
