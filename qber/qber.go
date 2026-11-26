// SPDX-License-Identifier: AGPL-3.0-only

// Package qber estimates the quantum bit error rate from a random public
// sample of the sifted key.
package qber

import (
	"fmt"
	"math"

	"github.com/fizsar/polaris-qkd/rng"
)

// Threshold is the BB84 error rate above which no secret key can be
// distilled with one-way post-processing (Shor and Preskill, 2000): ≈ 11 %.
const Threshold = 0.11

// Estimate is the outcome of sampling.
type Estimate struct {
	SampleBits int     `json:"sample_bits"` // bits sacrificed (publicly compared, then discarded)
	Errors     int     `json:"sample_errors"`
	QBER       float64 `json:"qber"`
}

// Sample picks round(fraction·n) random positions (at least 1, at most n-1),
// compares them, and returns the estimate plus both keys without the sampled
// positions. The sampled bits are discarded: they were made public.
func Sample(alice, bob []uint8, fraction float64, src rng.Source) (Estimate, []uint8, []uint8, error) {
	n := len(alice)
	if n != len(bob) {
		return Estimate{}, nil, nil, fmt.Errorf("qber: key length mismatch (%d vs %d)", n, len(bob))
	}
	if n < 2 {
		return Estimate{}, nil, nil, fmt.Errorf("qber: need at least 2 bits, got %d", n)
	}
	if !(fraction > 0 && fraction < 1) {
		return Estimate{}, nil, nil, fmt.Errorf("qber: sample fraction must be in (0,1), got %v", fraction)
	}
	k := int(math.Round(fraction * float64(n)))
	k = max(1, min(k, n-1))

	// Partial Fisher–Yates: the first k entries of idx are the sample.
	idx := make([]int32, n)
	for i := range idx {
		idx[i] = int32(i)
	}
	picked := make([]bool, n)
	errs := 0
	for i := 0; i < k; i++ {
		j := i + rng.Intn(src, n-i)
		idx[i], idx[j] = idx[j], idx[i]
		p := idx[i]
		picked[p] = true
		if alice[p] != bob[p] {
			errs++
		}
	}
	restA := make([]uint8, 0, n-k)
	restB := make([]uint8, 0, n-k)
	for i := 0; i < n; i++ {
		if !picked[i] {
			restA = append(restA, alice[i])
			restB = append(restB, bob[i])
		}
	}
	return Estimate{SampleBits: k, Errors: errs, QBER: float64(errs) / float64(k)}, restA, restB, nil
}
