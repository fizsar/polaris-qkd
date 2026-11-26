// SPDX-License-Identifier: AGPL-3.0-only

// Package privacy implements the last two post-processing steps: universal-
// hash verification of the corrected keys and privacy amplification with a
// random Toeplitz matrix.
//
// Secret length. The asymptotic bound for single-photon BB84 with one-way
// reconciliation (Shor–Preskill; see Scarani et al., 2009) is
//
//	ℓ = n·(1 − h(Q)) − leak_EC − margin
//
// with h the binary entropy. It ignores finite-key effects and decoy states,
// which are future work: do not read it as a composable security proof.
package privacy

import (
	"math"
	"math/bits"

	"github.com/fizsar/polaris-qkd/rng"
)

// H2 is the binary entropy function h(p) = −p·log2 p − (1−p)·log2(1−p).
func H2(p float64) float64 {
	if p <= 0 || p >= 1 {
		return 0
	}
	return -p*math.Log2(p) - (1-p)*math.Log2(1-p)
}

// SecretLength returns floor(n·(1−h(q))) − leakEC − margin, never below 0.
func SecretLength(n int, q float64, leakEC, margin int) int {
	l := int(math.Floor(float64(n)*(1-H2(q)))) - leakEC - margin
	return max(l, 0)
}

// Toeplitz is a random M×N binary matrix with constant diagonals:
// T[i][j] = s[i + N−1 − j], where s is a random vector of N+M−1 bits.
// The family is 2-universal, so it serves both as a verification hash and as
// the privacy-amplification function.
type Toeplitz struct {
	N, M int
	seed []uint64 // s, bit-packed (little endian), padded with spare words
}

// NewToeplitz draws a random matrix. Alice would announce the seed publicly.
func NewToeplitz(n, m int, src rng.Source) *Toeplitz {
	words := (n+m-1+63)/64 + 2 // spare words so the sliding window never overruns
	s := make([]uint64, words)
	for i := range s {
		s[i] = src.Uint64()
	}
	return &Toeplitz{N: n, M: m, seed: s}
}

// SeedBits is the number of public random bits that define the matrix.
func (t *Toeplitz) SeedBits() int { return t.N + t.M - 1 }

// Apply computes T·x over GF(2). x has N bits (one per byte, 0 or 1); the
// result has M bits. Rows are evaluated 64 columns at a time with popcount
// parity, so the cost is about M·N/64 word operations.
func (t *Toeplitz) Apply(x []uint8) []uint8 {
	if len(x) != t.N {
		panic("privacy: Toeplitz.Apply input length != N")
	}
	// Row i is Σ_j s[i+N−1−j]·x[j]. Packing x reversed (bit j' = x[N−1−j'])
	// turns it into a plain AND of x' with the window of s starting at bit i.
	nw := (t.N + 63) / 64
	xr := make([]uint64, nw)
	for j := 0; j < t.N; j++ {
		if x[t.N-1-j] != 0 {
			xr[j>>6] |= 1 << (uint(j) & 63)
		}
	}
	out := make([]uint8, t.M)
	for i := 0; i < t.M; i++ {
		wi, r := i>>6, uint(i&63)
		var acc uint64
		if r == 0 {
			for k := 0; k < nw; k++ {
				acc ^= t.seed[wi+k] & xr[k]
			}
		} else {
			for k := 0; k < nw; k++ {
				acc ^= (t.seed[wi+k]>>r | t.seed[wi+k+1]<<(64-r)) & xr[k]
			}
		}
		out[i] = uint8(bits.OnesCount64(acc) & 1)
	}
	return out
}

// Verify checks, with a bits-long universal hash, that two corrected keys are
// equal. The tag of one side is sent to the other, so `bits` bits are revealed
// and must be subtracted from the secret length. Two different keys collide
// with probability 2^−bits.
func Verify(a, b []uint8, tagBits int, src rng.Source) bool {
	if len(a) != len(b) {
		return false
	}
	h := NewToeplitz(len(a), tagBits, src)
	ta, tb := h.Apply(a), h.Apply(b)
	for i := range ta {
		if ta[i] != tb[i] {
			return false
		}
	}
	return true
}

// Block is one independent privacy-amplification block.
type Block struct {
	Start, N, M int
	T           *Toeplitz // nil when M == 0
}

// Plan splits a key into blocks and fixes each block's output length and
// Toeplitz matrix. Both parties apply the same plan to their key.
type Plan struct {
	Blocks []Block
}

// OutputBits is the total final key length.
func (p Plan) OutputBits() int {
	t := 0
	for _, b := range p.Blocks {
		t += b.M
	}
	return t
}

// NewPlan builds the amplification plan for an n-bit corrected key.
//
// With blockBits <= 0 (or >= n) the whole key is one block. Otherwise it is
// cut into blocks of blockBits bits and each block gets its own length
//
//	ℓ_b = n_b·(1 − h(q)) − leak_b − margin
//
// where leak_b is the share of leakEC proportional to n_b, rounded up, and
// the margin is charged once per block. Block-wise amplification is a common
// simplification: it bounds the time and the seed size (Toeplitz cost grows
// with n·ℓ) at the price of a few extra bits lost per block.
func NewPlan(n int, q float64, leakEC, margin, blockBits int, src rng.Source) Plan {
	if blockBits <= 0 || blockBits >= n {
		blockBits = max(n, 1)
	}
	var p Plan
	for start := 0; start < n; start += blockBits {
		nb := min(blockBits, n-start)
		leakB := (leakEC*nb + n - 1) / n
		m := SecretLength(nb, q, leakB, margin)
		blk := Block{Start: start, N: nb, M: m}
		if m > 0 {
			blk.T = NewToeplitz(nb, m, src)
		}
		p.Blocks = append(p.Blocks, blk)
	}
	return p
}

// Apply amplifies a key (length must equal the n the plan was built for).
func (p Plan) Apply(key []uint8) []uint8 {
	out := make([]uint8, 0, p.OutputBits())
	for _, b := range p.Blocks {
		if b.T == nil {
			continue
		}
		out = append(out, b.T.Apply(key[b.Start:b.Start+b.N])...)
	}
	return out
}
