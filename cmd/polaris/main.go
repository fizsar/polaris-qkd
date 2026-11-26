// SPDX-License-Identifier: AGPL-3.0-only

// Command polaris runs the Polaris BB84 QKD simulator.
//
// THIS IS A SIMULATION: there is no quantum hardware.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/fizsar/polaris-qkd/protocol"
	"github.com/fizsar/polaris-qkd/rng"
)

const usage = `polaris — BB84 quantum key distribution SIMULATOR (no quantum hardware)

Usage:
  polaris simulate [flags]   run one simulation and print the result
  polaris sweep    [flags]   vary distance, noise or Eve and write a CSV (and an SVG)
  polaris version

Run "polaris simulate -h" or "polaris sweep -h" for the flags.
`

// version is overridden at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	var err error
	switch args[0] {
	case "simulate":
		err = simulate(args[1:], stdout, stderr)
	case "sweep":
		err = sweep(args[1:], stdout, stderr)
	case "version", "--version":
		fmt.Fprintln(stdout, "polaris", version, "(simulator)")
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n%s", args[0], usage)
		return 2
	}
	switch err {
	case nil:
		return 0
	case flag.ErrHelp:
		return 0
	default:
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
}

// common holds the flags shared by simulate and sweep.
type common struct {
	cfg  protocol.Config
	seed uint64
	lang string
	fs   *flag.FlagSet
}

func newCommon(name string, stderr io.Writer) *common {
	c := &common{cfg: protocol.DefaultConfig()}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	c.fs = fs
	d := c.cfg
	fs.IntVar(&c.cfg.Photons, "photons", d.Photons, "photons Alice sends")
	fs.Float64Var(&c.cfg.Channel.DistanceKm, "distance", d.Channel.DistanceKm, "fibre length in km")
	fs.Float64Var(&c.cfg.Channel.ErrorRate, "noise", d.Channel.ErrorRate, "intrinsic error rate (misalignment), 0..0.5")
	fs.Float64Var(&c.cfg.Channel.EveFraction, "eve", d.Channel.EveFraction, "fraction of photons Eve intercepts and resends, 0..1")
	fs.Uint64Var(&c.seed, "seed", 0, "seed for reproducible runs (omit to use crypto/rand)")
	fs.StringVar(&c.lang, "lang", "en", "language of the human-readable output: en or es")
	fs.Float64Var(&c.cfg.Channel.DetectorEfficiency, "detector-eff", d.Channel.DetectorEfficiency, "detector efficiency, (0,1]")
	fs.Float64Var(&c.cfg.Channel.AttenuationDBPerKm, "attenuation", d.Channel.AttenuationDBPerKm, "fibre attenuation in dB/km")
	fs.Float64Var(&c.cfg.Channel.DarkCountProb, "dark-counts", d.Channel.DarkCountProb, "dark count probability per detector per gate")
	fs.Float64Var(&c.cfg.SampleFraction, "sample-fraction", d.SampleFraction, "share of the sifted key spent estimating the QBER")
	fs.Float64Var(&c.cfg.QBERThreshold, "threshold", d.QBERThreshold, "abort above this QBER")
	fs.Float64Var(&c.cfg.SigmaMargin, "sigma", d.SigmaMargin, "z in Q_up = Q + z·σ used for the final key length")
	fs.IntVar(&c.cfg.SecurityMargin, "margin", d.SecurityMargin, "security margin in bits per amplification block")
	fs.IntVar(&c.cfg.VerifyBits, "verify-bits", d.VerifyBits, "length of the verification hash in bits")
	fs.IntVar(&c.cfg.BlockBits, "block-bits", d.BlockBits, "privacy-amplification block size in bits (0 = one block)")
	fs.Float64Var(&c.cfg.ClockHz, "clock-hz", d.ClockHz, "pulse rate in Hz, only to express the key rate per second")
	return c
}

func (c *common) parse(args []string) error {
	if err := c.fs.Parse(args); err != nil {
		return err
	}
	if c.lang != "en" && c.lang != "es" {
		return fmt.Errorf("--lang must be en or es")
	}
	return c.cfg.Validate()
}

func (c *common) seeded() bool {
	set := false
	c.fs.Visit(func(f *flag.Flag) {
		if f.Name == "seed" {
			set = true
		}
	})
	return set
}

// source returns the random source for run number i of point j.
func (c *common) source(j, i int) rng.Source {
	if !c.seeded() {
		return rng.NewCrypto()
	}
	return rng.NewSeeded(c.seed + uint64(j)*1_000_003 + uint64(i)*7919)
}

// tr picks the English or Spanish text.
func (c *common) tr(en, es string) string {
	if c.lang == "es" {
		return es
	}
	return en
}
