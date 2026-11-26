// SPDX-License-Identifier: AGPL-3.0-only

//go:build js && wasm

// Command polaris-wasm exposes the simulator to JavaScript. It registers two
// global functions:
//
//	polarisSimulate(photons, distanceKm, noise, eve, seed) → JSON string (protocol.Result)
//	polarisTrace(photons, distanceKm, noise, eve, seed[, onlyDetected]) → JSON string (first photons, one record each)
//
// noise is 0..0.5, eve is 0..1, and a seed of 0 means crypto/rand (via the
// browser's crypto.getRandomValues). On bad input the JSON is {"error": "..."}.
// Arguments are plain numbers (not JSON) so the code behaves the same under
// the standard Go toolchain and TinyGo.
//
// THIS IS A SIMULATION: there is no quantum hardware.
package main

import (
	"encoding/json"
	"errors"
	"syscall/js"

	"github.com/fizsar/polaris-qkd/bb84"
	"github.com/fizsar/polaris-qkd/protocol"
	"github.com/fizsar/polaris-qkd/rng"
)

type request struct {
	photons  int
	distance float64
	noise    float64
	eve      float64
	seed     uint64

	onlyDetected bool // trace only: skip photons Bob never registers
}

func parse(args []js.Value) (request, error) {
	if len(args) < 5 || len(args) > 6 {
		return request{}, errors.New("expected (photons, distanceKm, noise, eve, seed[, onlyDetected])")
	}
	r := request{
		photons:  args[0].Int(),
		distance: args[1].Float(),
		noise:    args[2].Float(),
		eve:      args[3].Float(),
		seed:     uint64(args[4].Float()),
	}
	r.onlyDetected = len(args) == 6 && args[5].Float() != 0
	return r, nil
}

func (r request) config() protocol.Config {
	c := protocol.DefaultConfig()
	c.Photons = r.photons
	c.Channel.DistanceKm = r.distance
	c.Channel.ErrorRate = r.noise
	c.Channel.EveFraction = r.eve
	return c
}

func (r request) source() rng.Source {
	if r.seed == 0 {
		return rng.NewCrypto()
	}
	return rng.NewSeeded(r.seed)
}

func reply(v any, err error) any {
	if err != nil {
		b, _ := json.Marshal(map[string]string{"error": err.Error()})
		return string(b)
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func main() {
	js.Global().Set("polarisSimulate", js.FuncOf(func(_ js.Value, args []js.Value) any {
		r, err := parse(args)
		if err != nil {
			return reply(nil, err)
		}
		res, err := protocol.Run(r.config(), r.source())
		return reply(res, err)
	}))
	js.Global().Set("polarisTrace", js.FuncOf(func(_ js.Value, args []js.Value) any {
		r, err := parse(args)
		if err != nil {
			return reply(nil, err)
		}
		cfg := r.config()
		if err := cfg.Validate(); err != nil {
			return reply(nil, err)
		}
		return reply(bb84.Trace(cfg.Channel, min(r.photons, 1000), r.source(), r.onlyDetected), nil)
	}))
	select {} // keep the Go runtime alive for JS callbacks
}
