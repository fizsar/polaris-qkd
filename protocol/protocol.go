// SPDX-License-Identifier: AGPL-3.0-only

// Package protocol runs the whole simulated BB84 pipeline: quantum exchange,
// sifting, QBER estimation, Cascade, verification and privacy amplification.
// It is the entry point meant to be reused by later phases.
//
// THIS IS A SIMULATION. There is no quantum hardware: photons are modelled
// as random variables. The output is not key material and must never be
// presented as real quantum key distribution.
package protocol

import (
	"fmt"

	"github.com/fizsar/polaris-qkd/bb84"
	"github.com/fizsar/polaris-qkd/cascade"
	"github.com/fizsar/polaris-qkd/channel"
	"github.com/fizsar/polaris-qkd/privacy"
	"github.com/fizsar/polaris-qkd/qber"
	"github.com/fizsar/polaris-qkd/rng"
)

// Notice is attached to every Result.
const Notice = "SIMULATION: no quantum hardware is involved. Photons are modelled as random variables; this is not real QKD and the keys are not for real use."

// Abort codes.
const (
	AbortSiftedTooShort    = "sifted_too_short"
	AbortQBERTooHigh       = "qber_too_high"
	AbortVerificationFails = "verification_failed"
	AbortNoSecretKey       = "no_secret_key"
)

// Config holds every parameter of a run.
type Config struct {
	Photons int            `json:"photons"`
	Channel channel.Config `json:"channel"`

	SampleFraction float64 `json:"sample_fraction"` // share of the sifted key spent on QBER estimation
	QBERThreshold  float64 `json:"qber_threshold"`  // abort above this value
	VerifyBits     int     `json:"verify_bits"`     // length of the verification hash
	SecurityMargin int     `json:"security_margin"` // bits subtracted per amplification block
	BlockBits      int     `json:"block_bits"`      // privacy-amplification block size (0 = one block)
	MinSiftedBits  int     `json:"min_sifted_bits"` // abort below this sifted length
	ClockHz        float64 `json:"clock_hz"`        // pulse rate, only to express a rate per second
}

// DefaultConfig returns the defaults used by the CLI.
func DefaultConfig() Config {
	return Config{
		Photons:        1_000_000,
		Channel:        channel.Default(),
		SampleFraction: 0.10,
		QBERThreshold:  qber.Threshold,
		VerifyBits:     64,
		SecurityMargin: 64,
		BlockBits:      1 << 16,
		MinSiftedBits:  200,
		ClockHz:        1e9,
	}
}

// Validate checks the configuration.
func (c Config) Validate() error {
	if c.Photons <= 0 {
		return fmt.Errorf("protocol: photons must be > 0, got %d", c.Photons)
	}
	if err := c.Channel.Validate(); err != nil {
		return err
	}
	switch {
	case !(c.SampleFraction > 0 && c.SampleFraction < 1):
		return fmt.Errorf("protocol: sample fraction must be in (0,1), got %v", c.SampleFraction)
	case !(c.QBERThreshold > 0 && c.QBERThreshold < 0.5):
		return fmt.Errorf("protocol: QBER threshold must be in (0,0.5), got %v", c.QBERThreshold)
	case c.VerifyBits < 1 || c.SecurityMargin < 0 || c.BlockBits < 0 || c.MinSiftedBits < 2 || c.ClockHz < 0:
		return fmt.Errorf("protocol: invalid verify bits, margin, block size, minimum length or clock")
	}
	return nil
}

// Result reports a run. Lengths are in bits.
type Result struct {
	Notice string `json:"notice"`
	Config Config `json:"config"`

	PhotonsSent     int        `json:"photons_sent"`
	PhotonsDetected int        `json:"photons_detected"`
	Quantum         bb84.Stats `json:"quantum"`

	Sifted    int     `json:"sifted_bits"` // key length after sifting
	SiftRatio float64 `json:"sift_ratio"`  // sifted / detected, ≈ 0.5

	SampleBits   int     `json:"sample_bits_spent"` // sacrificed for QBER estimation
	SampleErrors int     `json:"sample_errors"`
	QBER         float64 `json:"qber"`
	TrueQBER     float64 `json:"true_qber_sim_only"` // ground truth, only known in a simulation

	ReconciledBits int     `json:"reconciled_bits"` // key length entering Cascade
	LeakEC         int     `json:"leak_ec"`         // parity bits revealed by Cascade
	CascadeEff     float64 `json:"cascade_efficiency"`
	ResidualErrors int     `json:"residual_errors_after_cascade_sim_only"`
	VerifyBits     int     `json:"verify_bits"` // revealed by the verification hash
	Verified       bool    `json:"verified"`

	AmplificationBlocks int     `json:"amplification_blocks"`
	FinalBits           int     `json:"final_key_bits"`
	KeyRatePerPulse     float64 `json:"key_rate_per_pulse"`
	KeyRatePerSecond    float64 `json:"key_rate_bits_per_s"`
	KeysMatch           bool    `json:"keys_match"` // Alice's and Bob's final keys are equal

	Aborted     bool   `json:"aborted"`
	AbortCode   string `json:"abort_code,omitempty"`
	AbortReason string `json:"abort_reason,omitempty"`

	AliceKey []uint8 `json:"-"`
	BobKey   []uint8 `json:"-"`
}

func (r *Result) abort(code, format string, a ...any) {
	r.Aborted, r.AbortCode, r.AbortReason = true, code, fmt.Sprintf(format, a...)
	r.FinalBits, r.KeyRatePerPulse, r.KeyRatePerSecond = 0, 0, 0
	r.AliceKey, r.BobKey = nil, nil
}

// Run executes the protocol. An aborted run (too much error, no secret key,
// failed verification) is a valid outcome and is reported in the Result, not
// as an error; errors are only for invalid configuration.
func Run(cfg Config, src rng.Source) (Result, error) {
	if err := cfg.Validate(); err != nil {
		return Result{}, err
	}
	res := Result{Notice: Notice, Config: cfg, PhotonsSent: cfg.Photons}

	// 1. Quantum phase and sifting.
	raw := bb84.Exchange(cfg.Channel, cfg.Photons, src)
	res.Quantum = raw.Stats
	res.PhotonsDetected = raw.Stats.Detected
	alice, bob := bb84.Sift(raw)
	res.Sifted = len(alice)
	if raw.Stats.Detected > 0 {
		res.SiftRatio = float64(res.Sifted) / float64(raw.Stats.Detected)
	}
	if res.Sifted > 0 {
		res.TrueQBER = float64(hamming(alice, bob)) / float64(res.Sifted)
	}
	if res.Sifted < cfg.MinSiftedBits {
		res.abort(AbortSiftedTooShort, "sifted key has %d bits, below the minimum of %d", res.Sifted, cfg.MinSiftedBits)
		return res, nil
	}

	// 2. QBER estimation on a random sample that is then discarded.
	est, alice, bob, err := qber.Sample(alice, bob, cfg.SampleFraction, src)
	if err != nil {
		return res, err
	}
	res.SampleBits, res.SampleErrors, res.QBER = est.SampleBits, est.Errors, est.QBER
	res.ReconciledBits = len(alice)
	if est.QBER > cfg.QBERThreshold {
		res.abort(AbortQBERTooHigh,
			"estimated QBER %.2f %% exceeds the %.2f %% BB84 threshold (Shor–Preskill): eavesdropping or excessive noise",
			100*est.QBER, 100*cfg.QBERThreshold)
		return res, nil
	}

	// 3. Error correction with Cascade.
	rec, err := cascade.Reconcile(alice, bob, est.QBER, src)
	if err != nil {
		return res, err
	}
	res.LeakEC = rec.Leak
	res.ResidualErrors = rec.ResidualErrors
	// Efficiency is measured against the real error rate of the key entering
	// Cascade (known only in a simulation), so it reflects Cascade itself and
	// not the noise of the QBER sample.
	res.CascadeEff = cascade.Efficiency(rec.Leak, len(alice), float64(rec.InitialErrors)/float64(len(alice)))

	// 4. Verification with a universal hash (its tag is public: it is a leak).
	res.VerifyBits = cfg.VerifyBits
	res.Verified = privacy.Verify(alice, rec.Corrected, cfg.VerifyBits, src)
	if !res.Verified {
		res.abort(AbortVerificationFails, "the corrected keys differ (%d residual errors): hash verification failed", rec.ResidualErrors)
		return res, nil
	}

	// 5. Privacy amplification (asymptotic single-photon bound).
	plan := privacy.NewPlan(len(alice), est.QBER, rec.Leak+cfg.VerifyBits, cfg.SecurityMargin, cfg.BlockBits, src)
	res.AmplificationBlocks = len(plan.Blocks)
	if plan.OutputBits() == 0 {
		res.abort(AbortNoSecretKey,
			"no secret key left: n·(1−h(Q)) = %d does not cover the leak of %d bits plus the margin",
			int(float64(len(alice))*(1-privacy.H2(est.QBER))), rec.Leak+cfg.VerifyBits)
		return res, nil
	}
	res.AliceKey = plan.Apply(alice)
	res.BobKey = plan.Apply(rec.Corrected)
	res.FinalBits = len(res.AliceKey)
	res.KeysMatch = string(res.AliceKey) == string(res.BobKey)
	res.KeyRatePerPulse = float64(res.FinalBits) / float64(cfg.Photons)
	res.KeyRatePerSecond = res.KeyRatePerPulse * cfg.ClockHz
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
