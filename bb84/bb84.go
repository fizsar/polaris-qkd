// SPDX-License-Identifier: AGPL-3.0-only

// Package bb84 simulates the quantum phase of the BB84 protocol (Bennett and
// Brassard, 1984) over a modelled channel, and the sifting step.
package bb84

import (
	"github.com/fizsar/polaris-qkd/channel"
	"github.com/fizsar/polaris-qkd/rng"
)

// Stats counts what happened during the quantum phase.
type Stats struct {
	Sent         int `json:"sent"`
	Intercepted  int `json:"intercepted"`   // photons Eve measured and resent
	Detected     int `json:"detected"`      // Bob registered a click (any kind)
	PhotonClicks int `json:"photon_clicks"` // clean photon detections
	DarkClicks   int `json:"dark_clicks"`   // dark count, no photon
	DoubleClicks int `json:"double_clicks"` // photon plus dark count
}

// Raw holds Alice's and Bob's records for the detected photons only. All four
// slices have the same length (Stats.Detected); entry i is one detection.
type Raw struct {
	AliceBit, AliceBase []uint8
	BobBit, BobBase     []uint8
	Stats               Stats
}

// Exchange sends n photons through the channel and returns the detection
// records. Alice picks bit and basis uniformly at random per photon, Bob
// picks his measurement basis uniformly at random.
func Exchange(cfg channel.Config, n int, src rng.Source) Raw {
	pPhoton := cfg.PhotonDetectionProb()
	pDark := cfg.DarkClickProb()
	var r Raw
	r.Stats.Sent = n
	for i := 0; i < n; i++ {
		intercepted := cfg.EveFraction > 0 && rng.Float64(src) < cfg.EveFraction
		photon := rng.Float64(src) < pPhoton
		dark := pDark > 0 && rng.Float64(src) < pDark
		if intercepted {
			r.Stats.Intercepted++
		}
		if !photon && !dark {
			continue
		}
		// Alice's random choices are drawn only for photons that matter: they
		// are independent of the click, so the statistics are unchanged.
		w := src.Uint64()
		q := channel.Qubit{Bit: uint8(w & 1), Base: uint8(w >> 1 & 1)}
		bobBase := uint8(w >> 2 & 1)
		sent := q
		if intercepted {
			sent = channel.Intercept(q, src)
		}
		bit, click := cfg.Detect(sent, bobBase, photon, dark, src)
		r.AliceBit = append(r.AliceBit, q.Bit)
		r.AliceBase = append(r.AliceBase, q.Base)
		r.BobBit = append(r.BobBit, bit)
		r.BobBase = append(r.BobBase, bobBase)
		r.Stats.Detected++
		switch click {
		case channel.PhotonClick:
			r.Stats.PhotonClicks++
		case channel.DarkClick:
			r.Stats.DarkClicks++
		case channel.DoubleClick:
			r.Stats.DoubleClicks++
		}
	}
	return r
}

// Sift keeps the detections where Alice's and Bob's bases coincide (they
// compare bases publicly, never bits) and returns the two raw keys.
func Sift(r Raw) (alice, bob []uint8) {
	for i := range r.AliceBit {
		if r.AliceBase[i] == r.BobBase[i] {
			alice = append(alice, r.AliceBit[i])
			bob = append(bob, r.BobBit[i])
		}
	}
	return alice, bob
}
