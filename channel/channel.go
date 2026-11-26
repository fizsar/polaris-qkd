// SPDX-License-Identifier: AGPL-3.0-only

// Package channel models the quantum channel and Bob's detector for a
// single-photon BB84 link. It is an idealised physical model, not hardware.
//
// Model summary:
//   - Fibre attenuation: transmittance η = 10^(−α·L/10).
//   - A photon is detected with probability η·detectorEfficiency.
//   - Dark counts: each of Bob's two detectors fires with probability
//     DarkCountProb per gate, independently of the photon.
//   - Intrinsic error (optical misalignment): the measured bit is flipped with
//     probability ErrorRate.
//   - Eve (optional) runs an intercept-resend attack on a fraction of the
//     photons. She is placed next to Alice with an ideal detector and source,
//     so she adds no loss, and measures in a uniformly random basis.
//   - Dark count or double click: Bob cannot tell which bit it was, so he
//     assigns a uniformly random bit (the usual practice). A dark count with
//     no photon and a photon plus a dark count in the same gate are both
//     handled this way (the second case is slightly pessimistic).
package channel

import (
	"fmt"
	"math"

	"github.com/fizsar/polaris-qkd/rng"
)

// Config holds the physical parameters of the link.
type Config struct {
	DistanceKm         float64 `json:"distance_km"`
	AttenuationDBPerKm float64 `json:"attenuation_db_per_km"`
	DetectorEfficiency float64 `json:"detector_efficiency"`
	DarkCountProb      float64 `json:"dark_count_prob"` // per detector, per gate
	ErrorRate          float64 `json:"error_rate"`      // intrinsic misalignment error
	EveFraction        float64 `json:"eve_fraction"`    // fraction of photons intercepted
}

// Default returns typical telecom-fibre values (0.2 dB/km, 10 % detector).
func Default() Config {
	return Config{
		DistanceKm:         0,
		AttenuationDBPerKm: 0.2,
		DetectorEfficiency: 0.1,
		DarkCountProb:      1e-6,
		ErrorRate:          0.01,
		EveFraction:        0,
	}
}

// Validate checks that every parameter is in range.
func (c Config) Validate() error {
	switch {
	case c.DistanceKm < 0 || math.IsNaN(c.DistanceKm):
		return fmt.Errorf("channel: distance must be >= 0, got %v", c.DistanceKm)
	case c.AttenuationDBPerKm < 0 || math.IsNaN(c.AttenuationDBPerKm):
		return fmt.Errorf("channel: attenuation must be >= 0, got %v", c.AttenuationDBPerKm)
	case !(c.DetectorEfficiency > 0 && c.DetectorEfficiency <= 1):
		return fmt.Errorf("channel: detector efficiency must be in (0,1], got %v", c.DetectorEfficiency)
	case !(c.DarkCountProb >= 0 && c.DarkCountProb < 0.5):
		return fmt.Errorf("channel: dark count probability must be in [0,0.5), got %v", c.DarkCountProb)
	case !(c.ErrorRate >= 0 && c.ErrorRate <= 0.5):
		return fmt.Errorf("channel: error rate must be in [0,0.5], got %v", c.ErrorRate)
	case !(c.EveFraction >= 0 && c.EveFraction <= 1):
		return fmt.Errorf("channel: Eve fraction must be in [0,1], got %v", c.EveFraction)
	}
	return nil
}

// Transmittance is the fibre transmittance η = 10^(−αL/10).
func (c Config) Transmittance() float64 {
	return math.Pow(10, -c.AttenuationDBPerKm*c.DistanceKm/10)
}

// PhotonDetectionProb is the probability that a sent photon is registered:
// η · detector efficiency.
func (c Config) PhotonDetectionProb() float64 {
	return c.Transmittance() * c.DetectorEfficiency
}

// Qubit is a single-photon BB84 state: Bit in {0,1}, Base 0 = rectilinear,
// 1 = diagonal.
type Qubit struct {
	Bit, Base uint8
}

// Measure returns the outcome of measuring q in the given basis. If the bases
// match the result is the encoded bit; otherwise it is uniformly random.
func Measure(q Qubit, base uint8, src rng.Source) uint8 {
	if q.Base == base {
		return q.Bit
	}
	return rng.Bit(src)
}

// Intercept performs an intercept-resend attack: Eve measures in a random
// basis and resends a photon prepared in what she measured.
func Intercept(q Qubit, src rng.Source) Qubit {
	b := rng.Bit(src)
	return Qubit{Bit: Measure(q, b, src), Base: b}
}

// Click classifies a detection event at Bob.
type Click uint8

const (
	NoClick     Click = iota // nothing registered
	PhotonClick              // clean single-photon detection
	DarkClick                // dark count with no photon
	DoubleClick              // photon and dark count in the same gate
)

// DarkClickProb is the probability that at least one of Bob's two detectors
// has a dark count in a gate.
func (c Config) DarkClickProb() float64 {
	d := c.DarkCountProb
	return 1 - (1-d)*(1-d)
}

// Detect simulates Bob's reception of photon q (already past Eve, if any)
// measured in bobBase. photon and dark say whether a photon arrived and
// whether a dark count fired; the caller draws them so that it can skip
// photons that produce no click. It returns the bit Bob records.
func (c Config) Detect(q Qubit, bobBase uint8, photon, dark bool, src rng.Source) (uint8, Click) {
	switch {
	case photon && !dark:
		bit := Measure(q, bobBase, src)
		if c.ErrorRate > 0 && rng.Float64(src) < c.ErrorRate {
			bit ^= 1
		}
		return bit, PhotonClick
	case photon && dark:
		return rng.Bit(src), DoubleClick
	case dark:
		return rng.Bit(src), DarkClick
	}
	return 0, NoClick
}
