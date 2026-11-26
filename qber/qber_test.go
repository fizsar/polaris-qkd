// SPDX-License-Identifier: AGPL-3.0-only

package qber

import (
	"math"
	"testing"

	"github.com/fizsar/polaris-qkd/rng"
)

func TestSampleRemovesAndEstimates(t *testing.T) {
	src := rng.NewSeeded(1)
	n, p := 100_000, 0.04
	a := rng.Bits(src, n)
	b := append([]uint8(nil), a...)
	for i := range b {
		if rng.Float64(src) < p {
			b[i] ^= 1
		}
	}
	est, ra, rb, err := Sample(a, b, 0.10, src)
	if err != nil {
		t.Fatal(err)
	}
	if est.SampleBits != 10_000 || len(ra) != n-10_000 || len(rb) != len(ra) {
		t.Fatalf("sample %d, remaining %d/%d", est.SampleBits, len(ra), len(rb))
	}
	if d := math.Abs(est.QBER - p); d > 4*math.Sqrt(p*(1-p)/float64(est.SampleBits)) {
		t.Errorf("QBER estimate %v, want %v", est.QBER, p)
	}
}

func TestUpperBound(t *testing.T) {
	// 0.02 + 3·sqrt(0.02·0.98/5000) = 0.02 + 3·0.0019799 = 0.025940
	if got := UpperBound(0.02, 5000, 3); math.Abs(got-0.02594) > 1e-5 {
		t.Errorf("UpperBound = %v, want 0.02594", got)
	}
	if got := UpperBound(0, 1000, 3); !(got > 0 && got < 0.1) {
		t.Errorf("zero errors must still get a positive margin, got %v", got)
	}
	if UpperBound(0.49, 10, 3) != 0.5 || UpperBound(0.1, 0, 3) != 0.5 {
		t.Error("bound must be capped at 0.5")
	}
	if UpperBound(0.05, 100, 0) != 0.05 {
		t.Error("z = 0 must return the estimate")
	}
}

func TestSampleErrors(t *testing.T) {
	src := rng.NewSeeded(2)
	if _, _, _, err := Sample(make([]uint8, 5), make([]uint8, 6), 0.1, src); err == nil {
		t.Error("length mismatch must fail")
	}
	if _, _, _, err := Sample(make([]uint8, 5), make([]uint8, 5), 1.5, src); err == nil {
		t.Error("bad fraction must fail")
	}
}
