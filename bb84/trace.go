// SPDX-License-Identifier: AGPL-3.0-only

package bb84

import (
	"github.com/fizsar/polaris-qkd/channel"
	"github.com/fizsar/polaris-qkd/rng"
)

// Photon is the full story of one sent photon, lost ones included. It exists
// to animate the protocol; the statistics are the same as in Exchange.
type Photon struct {
	AliceBit    uint8 `json:"ab"`
	AliceBase   uint8 `json:"aB"`
	Intercepted bool  `json:"e"`
	EveBase     uint8 `json:"eB"` // basis Eve measured and resent in (if Intercepted)
	EveBit      uint8 `json:"eb"` // bit Eve measured (if Intercepted)
	BobBase     uint8 `json:"bB"`
	BobBit      uint8 `json:"bb"` // bit Bob recorded (if Detected)
	Detected    bool  `json:"d"`  // Bob registered a click
	Sifted      bool  `json:"s"`  // detected and bases coincide
	Error       bool  `json:"x"`  // sifted and Bob's bit differs from Alice's
}

// maxTraceAttempts bounds the search for detected photons in Trace.
const maxTraceAttempts = 5_000_000

// Trace sends photons one by one and records everything about each, until n
// are recorded. With onlyDetected it skips the photons Bob never registers
// (still bounded by maxTraceAttempts), which makes the first n records the
// first n detections rather than the first n photons sent.
func Trace(cfg channel.Config, n int, src rng.Source, onlyDetected bool) []Photon {
	pPhoton := cfg.PhotonDetectionProb()
	pDark := cfg.DarkClickProb()
	out := make([]Photon, 0, n)
	for i := 0; len(out) < n && (i < n || (onlyDetected && i < maxTraceAttempts)); i++ {
		w := src.Uint64()
		q := channel.Qubit{Bit: uint8(w & 1), Base: uint8(w >> 1 & 1)}
		p := Photon{AliceBit: q.Bit, AliceBase: q.Base, BobBase: uint8(w >> 2 & 1)}
		sent := q
		if cfg.EveFraction > 0 && rng.Float64(src) < cfg.EveFraction {
			sent = channel.Intercept(q, src)
			p.Intercepted, p.EveBase, p.EveBit = true, sent.Base, sent.Bit
		}
		photon := rng.Float64(src) < pPhoton
		dark := pDark > 0 && rng.Float64(src) < pDark
		bit, click := cfg.Detect(sent, p.BobBase, photon, dark, src)
		if click != channel.NoClick {
			p.Detected, p.BobBit = true, bit
			p.Sifted = p.AliceBase == p.BobBase
			p.Error = p.Sifted && bit != q.Bit
		}
		if onlyDetected && !p.Detected {
			continue
		}
		out = append(out, p)
	}
	return out
}
