// SPDX-License-Identifier: AGPL-3.0-only

package privacy

import (
	"math"
	"testing"
	"time"

	"github.com/fizsar/polaris-qkd/rng"
)

func TestH2(t *testing.T) {
	cases := map[float64]float64{0: 0, 1: 0, 0.5: 1, 0.11: 0.4999}
	for p, want := range cases {
		if got := H2(p); math.Abs(got-want) > 1e-3 {
			t.Errorf("h(%v) = %v, want %v", p, got, want)
		}
	}
}

// naive computes T·x entry by entry from the definition T[i][j] = s[i+N−1−j].
func naive(t *Toeplitz, x []uint8) []uint8 {
	bit := func(k int) uint8 { return uint8(t.seed[k>>6] >> (uint(k) & 63) & 1) }
	out := make([]uint8, t.M)
	for i := 0; i < t.M; i++ {
		var s uint8
		for j := 0; j < t.N; j++ {
			s ^= bit(i+t.N-1-j) & x[j]
		}
		out[i] = s
	}
	return out
}

func TestToeplitzMatchesDefinition(t *testing.T) {
	src := rng.NewSeeded(1)
	for _, dim := range [][2]int{{1, 1}, {7, 3}, {64, 64}, {65, 40}, {130, 97}, {500, 311}} {
		T := NewToeplitz(dim[0], dim[1], src)
		x := rng.Bits(src, dim[0])
		got, want := T.Apply(x), naive(T, x)
		if string(got) != string(want) {
			t.Errorf("N=%d M=%d: fast product differs from the definition", dim[0], dim[1])
		}
	}
}

func TestToeplitzIsLinear(t *testing.T) {
	src := rng.NewSeeded(2)
	T := NewToeplitz(4000, 2500, src)
	x, y := rng.Bits(src, 4000), rng.Bits(src, 4000)
	xy := make([]uint8, 4000)
	for i := range xy {
		xy[i] = x[i] ^ y[i]
	}
	tx, ty, txy := T.Apply(x), T.Apply(y), T.Apply(xy)
	for i := range txy {
		if txy[i] != tx[i]^ty[i] {
			t.Fatalf("T(x⊕y) != T(x)⊕T(y) at output bit %d", i)
		}
	}
	zero := T.Apply(make([]uint8, 4000))
	for _, b := range zero {
		if b != 0 {
			t.Fatal("T·0 != 0")
		}
	}
}

func TestSecretLength(t *testing.T) {
	// h(0.05) = 0.286397 → n(1−h) = 7136.03 → 7136; minus 1200 and 64.
	if got := SecretLength(10_000, 0.05, 1200, 64); got != 5872 {
		t.Errorf("SecretLength = %d, want 5872", got)
	}
	if got := SecretLength(100, 0.2, 90, 64); got != 0 {
		t.Errorf("SecretLength must clamp at 0, got %d", got)
	}
}

func TestPlanOutputLength(t *testing.T) {
	src := rng.NewSeeded(3)
	n, q, leak, margin := 100_000, 0.03, 15_000, 64

	one := NewPlan(n, q, leak, margin, 0, src)
	if len(one.Blocks) != 1 || one.OutputBits() != SecretLength(n, q, leak, margin) {
		t.Fatalf("single block: got %d blocks, %d bits; want 1 block, %d bits",
			len(one.Blocks), one.OutputBits(), SecretLength(n, q, leak, margin))
	}

	blocks := NewPlan(n, q, leak, margin, 1<<14, src)
	if len(blocks.Blocks) != 7 { // 6 × 16384 + 1 × 1696
		t.Fatalf("got %d blocks, want 7", len(blocks.Blocks))
	}
	covered := 0
	for _, b := range blocks.Blocks {
		covered += b.N
	}
	if covered != n {
		t.Fatalf("blocks cover %d bits, want %d", covered, n)
	}
	// Blocks lose at most the margin per extra block plus rounding.
	loss := one.OutputBits() - blocks.OutputBits()
	if loss < 0 || loss > 6*margin+7*2 {
		t.Errorf("block-wise output %d vs single %d: loss %d outside [0, %d]",
			blocks.OutputBits(), one.OutputBits(), loss, 6*margin+14)
	}
	key := rng.Bits(src, n)
	if got := len(blocks.Apply(key)); got != blocks.OutputBits() {
		t.Errorf("Apply returned %d bits, plan says %d", got, blocks.OutputBits())
	}
}

func TestPlanSameKeySameOutput(t *testing.T) {
	src := rng.NewSeeded(4)
	key := rng.Bits(src, 5000)
	p := NewPlan(5000, 0.02, 800, 64, 2048, src)
	if string(p.Apply(key)) != string(p.Apply(append([]uint8(nil), key...))) {
		t.Fatal("equal keys must give equal amplified keys")
	}
	other := append([]uint8(nil), key...)
	other[100] ^= 1
	if string(p.Apply(key)) == string(p.Apply(other)) {
		t.Fatal("different keys gave the same amplified key (overwhelmingly unlikely)")
	}
}

func TestVerify(t *testing.T) {
	src := rng.NewSeeded(5)
	a := rng.Bits(src, 3000)
	b := append([]uint8(nil), a...)
	if !Verify(a, b, 64, src) {
		t.Fatal("equal keys must verify")
	}
	b[2999] ^= 1
	if Verify(a, b, 64, src) {
		t.Fatal("keys differing in one bit must not verify")
	}
}

// BenchmarkToeplitzWorstCase times one block with ~500k sifted bits, the size
// reached by 1M photons with a perfect detector and no loss.
func TestToeplitzTimingLargeBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("timing check skipped in -short mode")
	}
	src := rng.NewSeeded(6)
	n := 1 << 16
	T := NewToeplitz(n, n*3/4, src)
	x := rng.Bits(src, n)
	start := time.Now()
	T.Apply(x)
	t.Logf("2^16 bits → %d bits: %v", n*3/4, time.Since(start))
}
