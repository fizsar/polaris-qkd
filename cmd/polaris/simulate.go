// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/fizsar/polaris-qkd/protocol"
)

func simulate(args []string, stdout, stderr io.Writer) error {
	c := newCommon("simulate", stderr)
	asJSON := c.fs.Bool("json", false, "print the full result as JSON")
	if err := c.parse(args); err != nil {
		return err
	}
	res, err := protocol.Run(c.cfg, c.source(0, 0))
	if err != nil {
		return err
	}
	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}
	printHuman(c, res, stdout)
	return nil
}

func printHuman(c *common, r protocol.Result, w io.Writer) {
	t := c.tr
	p := func(label string, format string, a ...any) {
		fmt.Fprintf(w, "  %-34s %s\n", label, fmt.Sprintf(format, a...))
	}
	fmt.Fprintln(w, t(
		"*** SIMULATION — no quantum hardware. Not real QKD; the keys are not for real use. ***",
		"*** SIMULACIÓN — sin hardware cuántico. No es QKD real; las claves no sirven para uso real. ***"))
	ch := r.Config.Channel
	fmt.Fprintf(w, "\n%s: %d %s, %.1f km (η = %.4f), %s %.1f %%, %s %.1f %%, Eve %.0f %%\n\n",
		t("Link", "Enlace"), r.PhotonsSent, t("photons", "fotones"), ch.DistanceKm, ch.Transmittance(),
		t("detector", "detector"), 100*ch.DetectorEfficiency, t("noise", "ruido"), 100*ch.ErrorRate, 100*ch.EveFraction)

	p(t("Photons sent", "Fotones enviados"), "%d", r.PhotonsSent)
	p(t("Photons detected", "Fotones detectados"), "%d  (%s %d, %s %d)", r.PhotonsDetected,
		t("dark", "oscuras"), r.Quantum.DarkClicks, t("double", "dobles"), r.Quantum.DoubleClicks)
	if r.Quantum.Intercepted > 0 {
		p(t("Photons intercepted by Eve", "Fotones interceptados por Eve"), "%d", r.Quantum.Intercepted)
	}
	p(t("Key after sifting", "Clave tras el sifting"), "%d %s  (%.1f %% %s)", r.Sifted, t("bits", "bits"), 100*r.SiftRatio, t("of detected", "de los detectados"))
	if r.SampleBits > 0 {
		p(t("Spent on QBER estimation", "Gastados en estimar el QBER"), "%d %s", r.SampleBits, t("bits (discarded)", "bits (descartados)"))
		p("QBER", "%.2f %%  (%d/%d);  %s %.2f %%", 100*r.QBER, r.SampleErrors, r.SampleBits,
			t("upper bound", "cota superior"), 100*r.QBERUpper)
	}
	if r.LeakEC > 0 {
		p(t("Revealed by Cascade (leak_EC)", "Revelados por Cascade (leak_EC)"), "%d %s  (%s %.2f)", r.LeakEC, t("bits", "bits"), t("efficiency", "eficiencia"), r.CascadeEff)
		p(t("Revealed by verification", "Revelados por la verificación"), "%d %s", r.VerifyBits, t("bits", "bits"))
	}
	if r.Aborted {
		fmt.Fprintf(w, "\n  %s: %s\n", t("PROTOCOL ABORTED", "PROTOCOLO ABORTADO"), abortText(c, r))
		return
	}
	p(t("Final secret key", "Clave secreta final"), "%d %s  (%d %s)", r.FinalBits, t("bits", "bits"), r.AmplificationBlocks, t("blocks", "bloques"))
	p(t("Key rate", "Tasa de clave"), "%.5f %s  =  %.4g %s @ %.3g GHz", r.KeyRatePerPulse, t("bit/pulse", "bit/pulso"),
		r.KeyRatePerSecond, t("bit/s", "bit/s"), r.Config.ClockHz/1e9)
	match := t("yes", "sí")
	if !r.KeysMatch {
		match = t("NO", "NO")
	}
	p(t("Alice and Bob keys identical", "Claves de Alice y Bob idénticas"), "%s", match)
	fmt.Fprintf(w, "\n  %s\n", t(
		"Length = n·(1−h(Q_up)) − leak_EC − verification − margin: asymptotic single-photon bound (no finite-key, no decoy states).",
		"Longitud = n·(1−h(Q_sup)) − leak_EC − verificación − margen: cota asintótica de fotón único (sin clave finita ni estados señuelo)."))
}

// abortText translates the abort reason.
func abortText(c *common, r protocol.Result) string {
	switch r.AbortCode {
	case protocol.AbortQBERTooHigh:
		return fmt.Sprintf(c.tr(
			"estimated QBER %.2f %% exceeds the %.2f %% BB84 threshold (Shor–Preskill): eavesdropping or excessive noise",
			"el QBER estimado (%.2f %%) supera el umbral de BB84 del %.2f %% (Shor–Preskill): escucha o ruido excesivo"),
			100*r.QBER, 100*r.Config.QBERThreshold)
	case protocol.AbortSiftedTooShort:
		return fmt.Sprintf(c.tr(
			"sifted key has only %d bits (minimum %d): too few detections",
			"la clave tras el sifting tiene solo %d bits (mínimo %d): muy pocas detecciones"),
			r.Sifted, r.Config.MinSiftedBits)
	case protocol.AbortNoSecretKey:
		return c.tr(
			"no secret key left: the leaked bits and the margin use up the whole key",
			"no queda clave secreta: los bits revelados y el margen consumen toda la clave")
	}
	return r.AbortReason
}
