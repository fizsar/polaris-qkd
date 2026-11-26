// SPDX-License-Identifier: AGPL-3.0-only

package cascade

import (
	"testing"

	"github.com/fizsar/polaris-qkd/rng"
)

// noisyPair returns a random key and a copy with each bit flipped with prob q.
func noisyPair(n int, q float64, src rng.Source) (a, b []uint8) {
	a = rng.Bits(src, n)
	b = append([]uint8(nil), a...)
	for i := range b {
		if rng.Float64(src) < q {
			b[i] ^= 1
		}
	}
	return a, b
}

func TestCorrectsAcrossErrorRates(t *testing.T) {
	for _, q := range []float64{0, 0.005, 0.02, 0.05, 0.08, 0.11} {
		for seed := uint64(1); seed <= 5; seed++ {
			src := rng.NewSeeded(seed*1000 + uint64(q*1000))
			a, b := noisyPair(20_000, q, src)
			res, err := Reconcile(a, b, q, src)
			if err != nil {
				t.Fatal(err)
			}
			if res.ResidualErrors != 0 || string(res.Corrected) != string(a) {
				t.Errorf("q=%v seed=%d: %d residual errors left (of %d)", q, seed, res.ResidualErrors, res.InitialErrors)
			}
		}
	}
}

func TestLeakCloseToShannonBound(t *testing.T) {
	// Original Cascade sits around 1.1–1.25 times n·h(Q). The band is wide
	// enough for statistical noise at n = 100k, narrow enough to catch a
	// broken implementation (a naive scheme is far above, "no leak" below 1).
	for _, q := range []float64{0.01, 0.02, 0.05, 0.08} {
		src := rng.NewSeeded(77 + uint64(q*1000))
		a, b := noisyPair(100_000, q, src)
		res, _ := Reconcile(a, b, q, src)
		trueQ := float64(res.InitialErrors) / float64(len(a))
		eff := Efficiency(res.Leak, len(a), trueQ)
		t.Logf("q=%.3f errors=%d leak=%d efficiency=%.3f", q, res.InitialErrors, res.Leak, eff)
		if eff < 1.0 || eff > 1.30 {
			t.Errorf("q=%v: Cascade efficiency %.3f outside [1.0, 1.30]", q, eff)
		}
	}
}

func TestLeakAccounting(t *testing.T) {
	src := rng.NewSeeded(5)
	a, b := noisyPair(10_000, 0.03, src)
	res, _ := Reconcile(a, b, 0.03, src)
	if res.Leak != res.BlockParities+res.SearchParities {
		t.Fatalf("leak %d != block %d + search %d", res.Leak, res.BlockParities, res.SearchParities)
	}
}

func TestLengthMismatch(t *testing.T) {
	if _, err := Reconcile(make([]uint8, 3), make([]uint8, 4), 0.01, rng.NewSeeded(1)); err == nil {
		t.Fatal("expected an error for different lengths")
	}
}
