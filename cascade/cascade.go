// SPDX-License-Identifier: AGPL-3.0-only

// Package cascade implements the Cascade error-correction protocol
// (Brassard and Salvail, 1993) in its original form: four passes, first block
// size k1 ≈ 0.73/Q, doubling in every pass, random permutation between
// passes, and back-tracking into earlier passes after each correction.
//
// Alice's key is the reference and Bob corrects his. In a real run the parities
// travel over the public classical channel; here both keys are local and we
// count every parity that would be disclosed (leak_EC).
package cascade

import (
	"fmt"

	"github.com/fizsar/polaris-qkd/privacy"
	"github.com/fizsar/polaris-qkd/rng"
)

// Passes is the number of Cascade passes.
const Passes = 4

// Result describes a reconciliation run.
type Result struct {
	Corrected []uint8 `json:"-"` // Bob's key after correction

	Leak           int `json:"leak_ec"`        // parity bits disclosed
	BlockParities  int `json:"block_parities"` // part of Leak: block parities
	SearchParities int `json:"search_parities"`

	// Simulation-only ground truth (a real Bob cannot know these).
	InitialErrors  int `json:"initial_errors"`
	ResidualErrors int `json:"residual_errors"`
}

// Efficiency is leak / (n·h(Q)): 1 is the Shannon limit (Slepian–Wolf).
// Cascade typically reaches about 1.1–1.25.
func Efficiency(leak, n int, q float64) float64 {
	h := privacy.H2(q)
	if n == 0 || h == 0 {
		return 0
	}
	return float64(leak) / (float64(n) * h)
}

type pass struct {
	k    int     // block size
	perm []int32 // position order for this pass
	inv  []int32 // inv[pos] = index of pos inside perm
	diff []uint8 // per block: parity(Alice) xor parity(Bob), kept up to date
}

func (p *pass) block(pos int32) int { return int(p.inv[pos]) / p.k }

// Reconcile corrects bob against alice. qber is the (estimated) error rate
// used to choose the first block size.
func Reconcile(alice, bob []uint8, qber float64, src rng.Source) (Result, error) {
	n := len(alice)
	if n != len(bob) {
		return Result{}, fmt.Errorf("cascade: key length mismatch (%d vs %d)", n, len(bob))
	}
	b := append([]uint8(nil), bob...)
	res := Result{InitialErrors: hamming(alice, b)}
	if n == 0 {
		res.Corrected = b
		return res, nil
	}

	q := max(qber, 1e-4)
	k1 := int(0.73/q) + 1
	passes := make([]*pass, 0, Passes)

	parity := func(key []uint8, p *pass, lo, hi int) uint8 {
		var x uint8
		for _, pos := range p.perm[lo:hi] {
			x ^= key[pos]
		}
		return x
	}

	for pi := 0; pi < Passes; pi++ {
		p := &pass{k: min(n, k1<<uint(pi))}
		if pi == 0 {
			p.perm = make([]int32, n)
			for i := range p.perm {
				p.perm[i] = int32(i)
			}
		} else {
			p.perm = rng.Perm(src, n)
		}
		p.inv = make([]int32, n)
		for i, pos := range p.perm {
			p.inv[pos] = int32(i)
		}
		nb := (n + p.k - 1) / p.k
		p.diff = make([]uint8, nb)
		passes = append(passes, p)

		type blk struct{ pass, idx int }
		var stack []blk
		for i := 0; i < nb; i++ {
			lo, hi := i*p.k, min((i+1)*p.k, n)
			res.Leak++ // Alice discloses this block's parity
			res.BlockParities++
			p.diff[i] = parity(alice, p, lo, hi) ^ parity(b, p, lo, hi)
			if p.diff[i] == 1 {
				stack = append(stack, blk{pi, i})
			}
		}

		for len(stack) > 0 {
			t := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			q := passes[t.pass]
			if q.diff[t.idx] == 0 {
				continue // fixed meanwhile by another correction
			}
			// Binary search for one error in an odd-error block: each step
			// discloses Alice's parity of the first half.
			lo, hi := t.idx*q.k, min((t.idx+1)*q.k, n)
			for hi-lo > 1 {
				mid := (lo + hi) / 2
				res.Leak++
				res.SearchParities++
				if parity(alice, q, lo, mid) != parity(b, q, lo, mid) {
					hi = mid
				} else {
					lo = mid
				}
			}
			pos := q.perm[lo]
			b[pos] ^= 1
			// The flip changes the parity of its block in every pass so far.
			for ri, r := range passes {
				bi := r.block(pos)
				r.diff[bi] ^= 1
				if r.diff[bi] == 1 {
					stack = append(stack, blk{ri, bi})
				}
			}
		}
	}
	res.Corrected = b
	res.ResidualErrors = hamming(alice, b)
	return res, nil
}

func hamming(a, b []uint8) int {
	d := 0
	for i := range a {
		if a[i] != b[i] {
			d++
		}
	}
	return d
}
