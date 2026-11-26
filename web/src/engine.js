// SPDX-License-Identifier: AGPL-3.0-only
// Photon engine: one clock that replays a trace (records from the Go
// simulator) forever. Photon k is record k % N. The hero canvas draws what
// is in flight; the sifting table is filled as photons resolve at Bob.
// Nothing here changes the physics: it only replays what the WASM computed.

export const STAGGER = 140 // ms between emissions
export const TRAVEL = 2100 // ms from Alice to Bob

const hash = (i) => { let h = (i + 1) * 2654435761 >>> 0; h ^= h >>> 15; h = Math.imul(h, 2246822507) >>> 0; return h / 4294967296 }

export const outcome = (p) => (!p.d ? 'lost' : !p.s ? 'disc' : p.x ? 'err' : 'kept')

/** Fraction of the path at which a lost photon vanishes (past Eve if she touched it). */
export const lossAt = (p, i) => (p.e ? 0.58 + hash(i) * 0.37 : 0.18 + hash(i) * 0.77)
/** Time (ms) at which photon k is resolved (arrives or is lost). */
export const resolveAt = (p, i, k) => k * STAGGER + TRAVEL * (p.d ? 1 : lossAt(p, i))

export function createEngine(reduced, hooks) {
  const e = {
    photons: [], eveOn: false, elapsed: 0, slots: [], lastArrival: null, lastSent: -1,
    kRange() { // photons currently on screen
      return [Math.max(0, Math.floor((this.elapsed - TRAVEL * 1.15) / STAGGER)), Math.floor(this.elapsed / STAGGER)]
    },
    counts() {
      const c = { k: 0, e: 0, d: 0, l: 0 }
      for (const o of this.slots) if (o === 'kept') c.k++; else if (o === 'err') c.e++; else if (o === 'disc') c.d++; else if (o === 'lost') c.l++
      return c
    },
  }
  let emitted = -1, inflight = [], raf = 0, lastNow = 0

  function resolve(k) {
    const N = e.photons.length, i = k % N, p = e.photons[i]
    const o = outcome(p)
    const first = e.slots[i] === null
    e.slots[i] = o
    if (p.d) e.lastArrival = { p, i, at: e.elapsed }
    hooks.onResolve?.(i, p, first)
    if (first) hooks.onStats?.(e.counts())
  }

  function frame(now) {
    raf = requestAnimationFrame(frame)
    e.elapsed += Math.min(now - lastNow, 100); lastNow = now // no jumps after a hidden tab
    const N = e.photons.length
    if (!N) return
    const upTo = Math.floor(e.elapsed / STAGGER)
    while (emitted < upTo) { emitted++; inflight.push(emitted); e.lastSent = emitted % N }
    for (let j = inflight.length - 1; j >= 0; j--) {
      const k = inflight[j], i = k % N
      if (e.elapsed >= resolveAt(e.photons[i], i, k)) { resolve(k); inflight.splice(j, 1) }
    }
    hooks.onFrame?.(e)
  }

  function onVis() {
    cancelAnimationFrame(raf)
    if (!document.hidden && e.photons.length && !reduced) { lastNow = performance.now(); raf = requestAnimationFrame(frame) }
  }
  document.addEventListener('visibilitychange', onVis)

  e.load = (trace, eve) => {
    cancelAnimationFrame(raf)
    e.photons = trace.slice(0, 100); e.eveOn = eve > 0
    e.elapsed = 0; emitted = -1; inflight = []; e.lastArrival = null; e.lastSent = -1
    e.slots = new Array(e.photons.length).fill(null)
    hooks.onLoad?.(e)
    if (!e.photons.length) return
    if (reduced) { // still picture: everything is already filled in
      e.photons.forEach((p, i) => { e.elapsed = 1e9; resolve(i) })
      e.elapsed = 0
      hooks.onFrame?.(e)
    } else { lastNow = performance.now(); raf = requestAnimationFrame(frame) }
  }
  return e
}
