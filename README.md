# Polaris — BB84 QKD simulator

[Español](README.es.md) · English

> **⚠️ This is a SIMULATION.** There is no quantum hardware here: photons are
> random variables in a software model. Polaris is **not** real quantum key
> distribution, its output keys are **not** for real use, and it must never be
> presented as QKD. It exists to study the protocol and its post-processing,
> and as a reusable core for later phases (see the [roadmap](#roadmap)).

Polaris simulates the complete BB84 protocol (Bennett and Brassard, 1984) with
single photons over a fibre link, including all the classical post-processing:

1. **Quantum phase** — Alice picks random bits and bases, Bob measures in random bases.
2. **Sifting** — keep the positions where the bases agree (≈ 50 %).
3. **QBER estimation** — a random sample of the sifted key is compared in public and discarded. Abort above the BB84 threshold (≈ 11 %, Shor–Preskill).
4. **Error correction** — Cascade (Brassard and Salvail, 1993), counting every revealed parity bit (`leak_EC`).
5. **Verification** — a 64-bit universal (Toeplitz) hash of the corrected keys.
6. **Privacy amplification** — random Toeplitz matrix, block-wise.

Written in Go with no dependencies, as a library first (so it can be reused and
compiled to WebAssembly later) and a CLI second.

## Quick start

```sh
go build -o bin/polaris ./cmd/polaris

# one run, human-readable (add --lang es for Spanish, --json for JSON)
bin/polaris simulate --photons 1000000 --distance 25 --noise 0.02 --eve 0 --seed 1

# sweep distance, noise or Eve's fraction → CSV (and optionally an SVG chart)
bin/polaris sweep --vary distance --from 0 --to 120 --steps 13 --photons 10000000 --seed 1 \
    --out distance.csv --svg distance.svg
```

Omit `--seed` to draw from `crypto/rand`; with `--seed` every run is reproducible.
`make plots` regenerates the figures below.

### Example output

```
*** SIMULATION — no quantum hardware. Not real QKD; the keys are not for real use. ***

Link: 1000000 photons, 25.0 km (η = 0.3162), detector 10.0 %, noise 2.0 %, Eve 0 %

  Photons sent                       1000000
  Photons detected                   31715  (dark 3, double 0)
  Key after sifting                  15804 bits  (49.8 % of detected)
  Spent on QBER estimation           1580 bits (discarded)
  QBER                               2.15 %  (34/1580);  upper bound 3.25 %
  Revealed by Cascade (leak_EC)      2244 bits  (efficiency 1.15)
  Revealed by verification           64 bits
  Final secret key                   8912 bits  (1 blocks)
  Key rate                           0.00891 bit/pulse  =  8.912e+06 bit/s @ 1 GHz
  Alice and Bob keys identical       yes
```

With Eve intercepting everything the QBER jumps to ≈ 25 % and the protocol aborts,
saying why. `--json` prints every field (photons sent and detected, sifted key,
QBER, bits revealed, final length, key rate, abort reason…).

### Main flags

| Flag | Default | Meaning |
|---|---|---|
| `--photons` | 1 000 000 | photons sent |
| `--distance` | 0 | fibre length, km (transmittance η = 10^(−0.2·L/10)) |
| `--noise` | 0.01 | intrinsic error rate (misalignment) |
| `--eve` | 0 | fraction of photons Eve intercepts and resends |
| `--detector-eff` | 0.1 | detector efficiency |
| `--dark-counts` | 1e-6 | dark-count probability per detector per gate |
| `--sample-fraction` | 0.10 | share of the sifted key spent on the QBER estimate |
| `--sigma` | 3 | z in `Q_up = Q + z·σ` (see below) |
| `--block-bits` | 65536 | privacy-amplification block size (0 = one block) |
| `--seed` | — | reproducible run |

## Figures

Both are produced by `polaris sweep` (fixed seed; data in [`docs/`](docs)).

![Secret key rate vs distance](docs/key-rate-vs-distance.svg)

The key rate follows the fibre loss η (dashed) until the key becomes so short
that the fixed costs — 64-bit verification hash and a 64-bit margin per block —
and the QBER margin start to bite, around 80 km with these parameters.

![QBER vs fraction intercepted by Eve](docs/qber-vs-eve.svg)

Intercept-resend gives QBER = 25 % × f on the sifted key (plus the intrinsic
noise). Note that in practice the protocol stops **before** the 11 % threshold:
with Cascade at efficiency ≈ 1.2 the leaked bits exceed `n·(1−h(Q))` already at
Q ≈ 9–10 %, and the run aborts with *no secret key left*.

## How the secret length is computed

```
Q̂     = errors / sample                       (QBER of the discarded sample)
Q_up  = Q̂ + 3·σ,  σ = sqrt(Q̃(1−Q̃)/n_s),  Q̃ = max(Q̂, 1/n_s)
ℓ     = Σ_blocks [ n_b·(1 − h(Q_up)) − leak_b − margin ]     with leak = leak_EC + verification bits
```

`h` is the binary entropy, `n_s` the sample size, `leak_b` the block's
proportional share of the revealed bits (rounded up), and the margin (64 bits)
is charged once per block. This is the **asymptotic bound for single photons**.
The `+ 3σ` only covers the sampling error of the QBER estimate with a normal
approximation; it is **not** a rigorous finite-key bound.

## Modelling choices

* Losses: η = 10^(−αL/10), α = 0.2 dB/km; a photon is detected with probability η × detector efficiency.
* Intrinsic error: the measured bit is flipped with the configured probability.
* Eve: intercept-resend on a fraction *f* of the photons, random basis, ideal detector and source placed next to Alice (she adds no loss).
* Dark counts and double clicks: Bob cannot tell which bit it was, so he assigns a **uniformly random bit**, as is done in practice. A photon plus a dark count in the same gate counts as a double click (slightly pessimistic).
* Cascade: the original four-pass version, first block ≈ 0.73/Q, doubling per pass, with back-tracking. Measured efficiency 1.12–1.19 for Q = 1–8 %.
* Privacy amplification is **block-wise** (65 536 bits by default). Toeplitz cost grows like n·ℓ, so with a perfect detector and 1 M photons (500 k sifted bits) one block takes ≈ 10 s and 2^16-bit blocks take ≈ 2 s in total, at the price of ≈ 0.1 % of the key. This is a common simplification in implementations. With the default 10 % detector either way takes ≈ 0.1 s.
* Randomness: `crypto/rand` normally; a seeded ChaCha8 generator for reproducible runs and tests (never use a seeded run as key material).

## Tests

`go test ./...` (a few seconds). They check the physics with statistical tolerances (≈ 4σ, fixed seeds):

* no Eve and no noise → QBER = 0 and identical keys; noise *p* → QBER ≈ *p*;
* Eve at 100 % → QBER ≈ 25 %; at fraction *f* → ≈ 0.25·*f*;
* sifting keeps ≈ 50 % of the detections; detection rate = η × efficiency;
* above ≈ 11 % the protocol aborts, with the reason;
* Cascade leaves identical keys and its leak stays within 1.0–1.3 × the Shannon bound;
* Toeplitz is linear, matches its definition, and gives the expected length;
* the key rate falls with distance in proportion to η;
* over 20 seeds the mean QBER at 2 % noise equals the 2 % plus the dark-count contribution.

## Limitations

* **Simulation only**; idealised single-photon model with no real device imperfections or side channels.
* **No decoy states**, so no multi-photon (PNS) attacks are modelled: the source is an ideal single-photon source.
* **No finite-key analysis**: the secret length is the asymptotic bound plus a simple 3σ margin. No composable security parameter ε.
* Eve only does intercept-resend (individual attacks), not collective or coherent attacks.
* The authentication of the classical channel is assumed, not simulated.

## Roadmap

1. **Phase 1 (this repo):** BB84 simulator and reusable core library.
2. **Phase 2:** key manager exposing the ETSI GS QKD 014 API.
3. **Phase 3:** hybrid tunnel combining QKD keys with ML-KEM.
4. **Phase 4:** network with trusted nodes.

## Layout

| Package | Role |
|---|---|
| `rng` | `crypto/rand` and seeded randomness |
| `channel` | fibre loss, detector, dark counts, intrinsic error, Eve |
| `bb84` | quantum exchange and sifting |
| `qber` | sampling, QBER estimate and upper bound |
| `cascade` | error correction |
| `privacy` | binary entropy, Toeplitz hashing, verification, block-wise amplification |
| `protocol` | the whole pipeline: `protocol.Run(cfg, src)` |
| `plot` | dependency-free SVG charts |
| `cmd/polaris` | CLI |

## References

* C. H. Bennett and G. Brassard, *Quantum cryptography: public key distribution and coin tossing*, Proc. IEEE Int. Conf. on Computers, Systems and Signal Processing, Bangalore, pp. 175–179 (1984).
* P. W. Shor and J. Preskill, *Simple proof of security of the BB84 quantum key distribution protocol*, Phys. Rev. Lett. 85, 441 (2000).
* G. Brassard and L. Salvail, *Secret-key reconciliation by public discussion*, EUROCRYPT ’93, LNCS 765, pp. 410–423 (1994).
* V. Scarani et al., *The security of practical quantum key distribution*, Rev. Mod. Phys. 81, 1301 (2009). <https://arxiv.org/abs/0802.4155>
* S. Pirandola et al., *Advances in quantum cryptography*, Adv. Opt. Photon. 12, 1012 (2020). <https://arxiv.org/abs/1906.01645>

## License

[AGPL-3.0-only](LICENSE).
