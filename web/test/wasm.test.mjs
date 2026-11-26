// SPDX-License-Identifier: AGPL-3.0-only
// Fast checks of the WebAssembly build, run under Node with the same
// wasm_exec.js the browser uses. WASM and WASM_EXEC env vars select another
// build (e.g. the TinyGo one).
import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { pathToFileURL, fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const pub = join(dirname(fileURLToPath(import.meta.url)), '..', 'public')
const wasmPath = process.env.WASM || join(pub, 'polaris.wasm')
const execPath = process.env.WASM_EXEC || join(pub, 'wasm_exec.js')

await import(pathToFileURL(execPath).href)
const go = new globalThis.Go()
const { instance } = await WebAssembly.instantiate(readFileSync(wasmPath), go.importObject)
go.run(instance) // registers the globals, then blocks (not awaited)

const args = (c) => [c.photons, c.distance, c.noise, c.eve, c.seed || 0]
const sim = (c) => JSON.parse(globalThis.polarisSimulate(...args(c)))
const base = { photons: 200000, distance: 10, noise: 0.02, eve: 0, seed: 1 }

test('clean link: identical keys and the simulation notice', () => {
  const r = sim(base)
  assert.equal(r.aborted, false)
  assert.equal(r.keys_match, true)
  assert.ok(r.final_key_bits > 0)
  assert.match(r.notice, /SIMULATION/)
  assert.ok(Math.abs(r.qber - 0.02) < 0.01)
})

test('Eve at 100% aborts with a high QBER', () => {
  const r = sim({ ...base, eve: 1 })
  assert.equal(r.aborted, true)
  assert.equal(r.abort_code, 'qber_too_high')
  assert.ok(Math.abs(r.qber - 0.26) < 0.03)
})

test('same seed, same run; seed 0 uses crypto/rand', () => {
  assert.equal(JSON.stringify(sim(base)), JSON.stringify(sim(base)))
  const a = sim({ ...base, seed: 0 }), b = sim({ ...base, seed: 0 })
  assert.notEqual(a.final_key_bits + ':' + a.leak_ec, b.final_key_bits + ':' + b.leak_ec)
})

test('trace returns per-photon records', () => {
  const t = JSON.parse(globalThis.polarisTrace(...args({ ...base, photons: 100, eve: 0.5, distance: 0 })))
  assert.equal(t.length, 100)
  for (const k of ['ab', 'aB', 'e', 'eB', 'bB', 'd', 's', 'x']) assert.ok(k in t[0], k)
  assert.ok(t.some((p) => p.e) && t.some((p) => !p.e))
})

test('invalid input gives an error object, not a crash', () => {
  assert.ok(sim({ ...base, eve: 2 }).error)
  assert.ok(JSON.parse(globalThis.polarisSimulate(1, 2)).error) // wrong arity
})

test('speed: 1M photons at 0 km and 100 km', () => {
  for (const distance of [0, 100]) {
    const t0 = performance.now()
    const r = sim({ photons: 1000000, distance, noise: 0.02, eve: 0, seed: 3 })
    const ms = Math.round(performance.now() - t0)
    console.log(`  1M photons @ ${distance} km: ${ms} ms (final ${r.final_key_bits} bits)`)
    assert.ok(ms < 20000)
  }
})
