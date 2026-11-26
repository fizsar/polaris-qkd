// SPDX-License-Identifier: AGPL-3.0-only

package rng

import (
	"math"
	"testing"
)

func TestSeededIsDeterministic(t *testing.T) {
	a, b, c := NewSeeded(1), NewSeeded(1), NewSeeded(2)
	same, diff := true, false
	for i := 0; i < 100; i++ {
		x, y, z := a.Uint64(), b.Uint64(), c.Uint64()
		same = same && x == y
		diff = diff || x != z
	}
	if !same || !diff {
		t.Fatalf("seeded determinism broken (same=%v, differs from other seed=%v)", same, diff)
	}
}

func TestCryptoSourceWorks(t *testing.T) {
	s := NewCrypto()
	seen := map[uint64]bool{}
	for i := 0; i < 2000; i++ { // crosses the 4 KiB buffer refill
		seen[s.Uint64()] = true
	}
	if len(seen) < 1999 {
		t.Fatalf("only %d distinct values out of 2000", len(seen))
	}
}

func TestBitIsBalanced(t *testing.T) {
	s := NewSeeded(3)
	n, ones := 200_000, 0
	for i := 0; i < n; i++ {
		ones += int(Bit(s))
	}
	if d := math.Abs(float64(ones)/float64(n) - 0.5); d > 4*math.Sqrt(0.25/float64(n)) {
		t.Fatalf("bit frequency off by %v", d)
	}
}

func TestIntnRangeAndPerm(t *testing.T) {
	s := NewSeeded(4)
	for i := 0; i < 10_000; i++ {
		if v := Intn(s, 7); v < 0 || v >= 7 {
			t.Fatalf("Intn out of range: %d", v)
		}
	}
	p := Perm(s, 1000)
	seen := make([]bool, 1000)
	for _, v := range p {
		if seen[v] {
			t.Fatal("Perm repeats a value")
		}
		seen[v] = true
	}
}
