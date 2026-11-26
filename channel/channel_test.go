// SPDX-License-Identifier: AGPL-3.0-only

package channel

import (
	"math"
	"testing"
)

func TestTransmittance(t *testing.T) {
	c := Default()
	c.DistanceKm = 50
	if got := c.Transmittance(); math.Abs(got-0.1) > 1e-12 { // 10 dB
		t.Errorf("η(50 km) = %v, want 0.1", got)
	}
	c.DistanceKm = 0
	if c.Transmittance() != 1 {
		t.Error("η(0 km) must be 1")
	}
}

func TestValidate(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatal(err)
	}
	bad := []func(*Config){
		func(c *Config) { c.DistanceKm = -1 },
		func(c *Config) { c.DetectorEfficiency = 0 },
		func(c *Config) { c.ErrorRate = 0.6 },
		func(c *Config) { c.EveFraction = -0.1 },
		func(c *Config) { c.DarkCountProb = 0.7 },
	}
	for i, f := range bad {
		c := Default()
		f(&c)
		if c.Validate() == nil {
			t.Errorf("case %d should be rejected", i)
		}
	}
}
