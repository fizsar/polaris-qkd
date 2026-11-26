// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func exec(t *testing.T, args ...string) (code int, out, errOut string) {
	t.Helper()
	var o, e bytes.Buffer
	code = run(args, &o, &e)
	return code, o.String(), e.String()
}

func TestSimulateJSON(t *testing.T) {
	code, out, errOut := exec(t, "simulate", "--photons", "200000", "--distance", "25", "--noise", "0.02", "--eve", "0", "--seed", "1", "--json")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	var r map[string]any
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"notice", "photons_sent", "photons_detected", "sifted_bits", "qber", "leak_ec", "final_key_bits", "key_rate_per_pulse", "keys_match", "sample_bits_spent"} {
		if _, ok := r[k]; !ok {
			t.Errorf("JSON is missing %q", k)
		}
	}
	if !strings.Contains(r["notice"].(string), "SIMULATION") {
		t.Error("notice must say SIMULATION")
	}
	if r["keys_match"] != true {
		t.Error("keys should match")
	}
}

func TestSimulateIsReproducible(t *testing.T) {
	_, a, _ := exec(t, "simulate", "--photons", "100000", "--seed", "7", "--json")
	_, b, _ := exec(t, "simulate", "--photons", "100000", "--seed", "7", "--json")
	if a != b {
		t.Fatal("same --seed must give identical output")
	}
}

func TestHumanOutputWarnsAndTranslates(t *testing.T) {
	_, en, _ := exec(t, "simulate", "--photons", "100000", "--seed", "1")
	_, es, _ := exec(t, "simulate", "--photons", "100000", "--seed", "1", "--lang", "es")
	if !strings.Contains(en, "SIMULATION") || !strings.Contains(es, "SIMULACIÓN") {
		t.Error("both languages must start with the simulation warning")
	}
	_, ab, _ := exec(t, "simulate", "--photons", "200000", "--eve", "1", "--seed", "1")
	if !strings.Contains(ab, "ABORTED") || !strings.Contains(ab, "threshold") {
		t.Errorf("abort not explained:\n%s", ab)
	}
}

func TestSweepCSVAndSVG(t *testing.T) {
	dir := t.TempDir()
	csvPath, svgPath := filepath.Join(dir, "s.csv"), filepath.Join(dir, "s.svg")
	code, _, errOut := exec(t, "sweep", "--vary", "eve", "--steps", "5", "--photons", "200000", "--seed", "1", "--out", csvPath, "--svg", svgPath)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	f, _ := os.Open(csvPath)
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil || len(rows) != 6 {
		t.Fatalf("want header + 5 rows, got %d (%v)", len(rows), err)
	}
	if rows[0][0] != "eve" || rows[1][0] != "0" || rows[5][0] != "1" {
		t.Errorf("unexpected CSV axis: %v ... %v", rows[1], rows[5])
	}
	svg, _ := os.ReadFile(svgPath)
	if !bytes.HasPrefix(svg, []byte("<svg")) {
		t.Error("SVG not written")
	}
}

func TestBadInput(t *testing.T) {
	for _, args := range [][]string{
		{"simulate", "--eve", "2"},
		{"simulate", "--lang", "fr"},
		{"sweep", "--vary", "colour"},
		{"sweep", "--steps", "1"},
		{"nope"},
		{},
	} {
		if code, _, _ := exec(t, args...); code == 0 {
			t.Errorf("%v should fail", args)
		}
	}
}
