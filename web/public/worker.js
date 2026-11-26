// SPDX-License-Identifier: AGPL-3.0-only
// Web Worker: loads the Go WebAssembly build and runs the simulator off the
// main thread. Classic worker (wasm_exec.js uses importScripts).
/* global Go */
importScripts('wasm_exec.js')

let ready
function load() {
  const go = new Go()
  const url = 'polaris.wasm'
  const inst = WebAssembly.instantiateStreaming
    ? WebAssembly.instantiateStreaming(fetch(url), go.importObject).catch(() =>
        fetch(url).then((r) => r.arrayBuffer()).then((b) => WebAssembly.instantiate(b, go.importObject)))
    : fetch(url).then((r) => r.arrayBuffer()).then((b) => WebAssembly.instantiate(b, go.importObject))
  return inst.then((r) => {
    go.run(r.instance) // registers polarisSimulate / polarisTrace, then blocks
  })
}
ready = load()

const args = (c) => [c.photons, c.distance, c.noise, c.eve, c.seed || 0]
const sim = (c) => JSON.parse(self.polarisSimulate(...args(c)))

// Compact summary of a run: what the charts need.
const brief = (r) => ({
  qber: r.qber, qberUpper: r.qber_upper, aborted: r.aborted, code: r.abort_code || '',
  final: r.final_key_bits, ratePulse: r.key_rate_per_pulse, rateBps: r.key_rate_bits_per_s,
})

self.onmessage = async (e) => {
  const { id, op, cfg } = e.data
  try {
    await ready
    if (op === 'sim') {
      const r = sim(cfg)
      if (r.error) throw new Error(r.error)
      self.postMessage({ id, type: 'result', result: r })
    } else if (op === 'trace') {
      const t = JSON.parse(self.polarisTrace(...args(cfg), cfg.onlyDetected ? 1 : 0))
      if (t.error) throw new Error(t.error)
      self.postMessage({ id, type: 'result', result: t })
    } else if (op === 'sweep') {
      // cfg: { base, vary: 'distance'|'eve', xs: [...] }; one message per point.
      for (let i = 0; i < e.data.xs.length; i++) {
        const c = { ...e.data.base, [e.data.vary]: e.data.xs[i] }
        const r = sim(c)
        if (r.error) throw new Error(r.error)
        self.postMessage({ id, type: 'point', i, x: e.data.xs[i], r: brief(r) })
      }
      self.postMessage({ id, type: 'done' })
    }
  } catch (err) {
    self.postMessage({ id, type: 'error', message: String(err && err.message || err) })
  }
}
