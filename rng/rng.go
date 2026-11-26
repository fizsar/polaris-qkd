// SPDX-License-Identifier: AGPL-3.0-only

// Package rng provides the randomness used by the simulator.
//
// Two sources exist: a cryptographic one (crypto/rand) for normal use and a
// seeded one (ChaCha8) so simulations and tests are reproducible. A seeded
// source is NOT meant to produce real key material.
package rng

import (
	"crypto/rand"
	"encoding/binary"
	"math/bits"
	mrand "math/rand/v2"
)

// Source is a stream of uniformly random 64-bit words.
type Source interface {
	Uint64() uint64
}

type cryptoSource struct {
	buf [4096]byte
	pos int
}

// NewCrypto returns a Source backed by crypto/rand.
func NewCrypto() Source {
	return &cryptoSource{pos: len((&cryptoSource{}).buf)}
}

func (c *cryptoSource) Uint64() uint64 {
	if c.pos+8 > len(c.buf) {
		// crypto/rand.Read never returns an error on supported platforms
		// (it aborts the program instead), so the error is ignored.
		_, _ = rand.Read(c.buf[:])
		c.pos = 0
	}
	v := binary.LittleEndian.Uint64(c.buf[c.pos:])
	c.pos += 8
	return v
}

// NewSeeded returns a deterministic Source: the same seed always yields the
// same stream. Intended for reproducible simulations and tests only.
func NewSeeded(seed uint64) Source {
	var k [32]byte
	binary.LittleEndian.PutUint64(k[:8], seed)
	return mrand.NewChaCha8(k)
}

// Float64 returns a uniform value in [0, 1).
func Float64(s Source) float64 {
	return float64(s.Uint64()>>11) / (1 << 53)
}

// Bit returns a uniform bit (0 or 1).
func Bit(s Source) uint8 {
	return uint8(s.Uint64() >> 63)
}

// Intn returns a uniform integer in [0, n) without modulo bias. n must be > 0.
func Intn(s Source, n int) int {
	if n <= 0 {
		panic("rng: Intn with n <= 0")
	}
	un := uint64(n)
	hi, lo := bits.Mul64(s.Uint64(), un)
	if lo < un {
		thresh := -un % un
		for lo < thresh {
			hi, lo = bits.Mul64(s.Uint64(), un)
		}
	}
	return int(hi)
}

// Perm returns a uniformly random permutation of 0..n-1 (Fisher–Yates).
func Perm(s Source, n int) []int32 {
	p := make([]int32, n)
	for i := range p {
		p[i] = int32(i)
	}
	for i := n - 1; i > 0; i-- {
		j := Intn(s, i+1)
		p[i], p[j] = p[j], p[i]
	}
	return p
}

// Bits fills a new slice with n uniform bits, one per byte (values 0 or 1).
func Bits(s Source, n int) []uint8 {
	out := make([]uint8, n)
	var w uint64
	left := 0
	for i := range out {
		if left == 0 {
			w, left = s.Uint64(), 64
		}
		out[i] = uint8(w & 1)
		w >>= 1
		left--
	}
	return out
}
