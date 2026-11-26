// SPDX-License-Identifier: AGPL-3.0-only
// Alice and Bob as two stars; the first ~100 photons travel between them as
// light pulses. White/cyan = photon, magenta = Eve, green = kept bit.
// prefers-reduced-motion: a still picture with every outcome already filled.

import { t } from './i18n.js'

const LW = 1000, LH = 300, SS = 2 // logical size, supersampling
const AX = 80, BX = 920, CY = 112, EX = 500
const STAGGER = 105, TRAVEL = 1500
const COL = {
  photon: '#eafcff', cyan: '90,200,250', eve: '#ff4fd8', eveRGB: '255,79,216',
  green: '#3dffa2', err: '#ff7a7a', dim: '#5a5f75', muted: '#8b90a5', line: 'rgba(143,140,255,.22)',
}

const outcome = (p) => (!p.d ? 'lost' : !p.s ? 'disc' : p.x ? 'err' : 'kept')
const baseGlyph = (b) => (b ? '×' : '+')
// polarisation angle of a BB84 state (canvas coordinates, y down)
const angle = (base, bit) => (base === 0 ? (bit ? Math.PI / 2 : 0) : bit ? -3 * Math.PI / 4 : -Math.PI / 4)
const hash = (i) => { let h = (i + 1) * 2654435761 >>> 0; h ^= h >>> 15; h = Math.imul(h, 2246822507) >>> 0; return (h >>> 0) / 4294967296 }

export function createStage(canvas, reduced, onStats) {
  canvas.width = LW * SS; canvas.height = LH * SS
  const ctx = canvas.getContext('2d')
  ctx.scale(SS, SS)
  let photons = [], eveOn = false, start = 0, raf = 0, elapsed = 0, lastNow = 0
  const font = (px, w = 500) => `${w} ${px}px "JetBrains Mono", ui-monospace, monospace`

  const lossAt = (p, i) => {
    const u = hash(i)
    return p.e ? 0.58 + u * 0.37 : 0.18 + u * 0.77
  }
  // time (ms) at which photon i is resolved (arrives or is lost)
  const resolveAt = (p, i) => i * STAGGER + TRAVEL * (p.d ? 1 : lossAt(p, i))

  function star(x, y, r, color, rgb, label, labelColor) {
    const g = ctx.createRadialGradient(x, y, 0, x, y, r * 3.2)
    g.addColorStop(0, `rgba(${rgb},.5)`); g.addColorStop(1, `rgba(${rgb},0)`)
    ctx.fillStyle = g; ctx.beginPath(); ctx.arc(x, y, r * 3.2, 0, 7); ctx.fill()
    ctx.strokeStyle = color; ctx.lineWidth = 1.2; ctx.globalAlpha = 0.8
    ctx.beginPath(); ctx.moveTo(x - r * 2.2, y); ctx.lineTo(x + r * 2.2, y); ctx.moveTo(x, y - r * 2.2); ctx.lineTo(x, y + r * 2.2); ctx.stroke()
    ctx.globalAlpha = 1
    ctx.fillStyle = color; ctx.beginPath(); ctx.arc(x, y, r * 0.55, 0, 7); ctx.fill()
    ctx.fillStyle = labelColor || COL.muted; ctx.font = font(13, 600); ctx.textAlign = 'center'
    ctx.fillText(label, x, y + r * 2.9 + 8)
  }

  function cell(i, kind, now) {
    const w = 8, gap = 0.62, x0 = AX - 6
    const x = x0 + i * (w + gap), y = 228
    ctx.lineWidth = 1
    if (kind === 'kept') { ctx.fillStyle = COL.green; ctx.fillRect(x, y, w, 22) }
    else if (kind === 'err') { ctx.fillStyle = COL.err; ctx.fillRect(x, y, w, 22) }
    else if (kind === 'disc') { ctx.strokeStyle = COL.dim; ctx.strokeRect(x + .5, y + .5, w - 1, 21) }
    else if (kind === 'lost') { ctx.fillStyle = 'rgba(90,95,117,.35)'; ctx.fillRect(x, y + 8, w, 6) }
    else { ctx.strokeStyle = 'rgba(143,140,255,.12)'; ctx.strokeRect(x + .5, y + .5, w - 1, 21) }
  }

  function drawPhoton(p, i, tms) {
    const s = tms - i * STAGGER
    if (s < 0) return
    const prog = s / TRAVEL
    const lp = p.d ? 2 : lossAt(p, i)
    if (prog > Math.min(1.02, lp + 0.14)) return
    const x = AX + (BX - AX) * Math.min(prog, 1)
    const afterEve = eveOn && p.e && prog >= (EX - AX) / (BX - AX)
    let alpha = 1
    if (!p.d && prog > lp) alpha = Math.max(0, 1 - (prog - lp) / 0.14)
    const rgb = afterEve ? COL.eveRGB : COL.cyan
    const core = afterEve ? COL.eve : COL.photon
    const bs = afterEve ? p.eB : p.aB, bt = afterEve ? p.eb : p.ab
    ctx.globalAlpha = alpha
    // trail
    const tr = ctx.createLinearGradient(x - 46, CY, x, CY)
    tr.addColorStop(0, `rgba(${rgb},0)`); tr.addColorStop(1, `rgba(${rgb},.5)`)
    ctx.strokeStyle = tr; ctx.lineWidth = 2
    ctx.beginPath(); ctx.moveTo(Math.max(AX, x - 46), CY); ctx.lineTo(x, CY); ctx.stroke()
    // glow + core
    const g = ctx.createRadialGradient(x, CY, 0, x, CY, 13)
    g.addColorStop(0, `rgba(${rgb},.75)`); g.addColorStop(1, `rgba(${rgb},0)`)
    ctx.fillStyle = g; ctx.beginPath(); ctx.arc(x, CY, 13, 0, 7); ctx.fill()
    ctx.fillStyle = core; ctx.beginPath(); ctx.arc(x, CY, 2.6, 0, 7); ctx.fill()
    // polarisation tick + basis label
    const a = angle(bs, bt)
    ctx.strokeStyle = core; ctx.lineWidth = 1.5
    ctx.beginPath(); ctx.moveTo(x - Math.cos(a) * 8, CY - Math.sin(a) * 8); ctx.lineTo(x + Math.cos(a) * 8, CY + Math.sin(a) * 8); ctx.stroke()
    ctx.fillStyle = afterEve ? COL.eve : COL.muted; ctx.font = font(11, 600); ctx.textAlign = 'center'
    ctx.fillText(baseGlyph(bs), x, CY - 17)
    ctx.globalAlpha = 1
  }

  function draw(tms) {
    ctx.clearRect(0, 0, LW, LH)
    // channel
    ctx.strokeStyle = COL.line; ctx.lineWidth = 1; ctx.setLineDash([3, 5])
    ctx.beginPath(); ctx.moveTo(AX, CY); ctx.lineTo(BX, CY); ctx.stroke(); ctx.setLineDash([])
    star(AX, CY, 9, COL.photon, COL.cyan, t('anim.alice'))
    star(BX, CY, 9, COL.photon, COL.cyan, t('anim.bob'))
    if (eveOn) star(EX, CY, 6.5, COL.eve, COL.eveRGB, t('anim.eve'), COL.eve)

    // outcome strip + counters; "current" photon info near Alice and Bob
    const c = { k: 0, e: 0, d: 0, l: 0 }
    let lastArrived = -1, lastSent = -1
    photons.forEach((p, i) => {
      const done = reduced || tms >= resolveAt(p, i)
      if (tms >= i * STAGGER) lastSent = i
      if (done) {
        const o = outcome(p)
        cell(i, o)
        if (o === 'kept') c.k++; else if (o === 'err') c.e++; else if (o === 'disc') c.d++; else c.l++
        if (p.d) lastArrived = i
      } else cell(i, 'pending')
    })
    if (!reduced) photons.forEach((p, i) => drawPhoton(p, i, tms))

    ctx.textAlign = 'center'; ctx.font = font(11.5)
    if (lastSent >= 0 && !reduced && tms < photons.length * STAGGER) {
      const p = photons[lastSent]
      ctx.fillStyle = COL.muted
      ctx.fillText(`${t('anim.base')} ${baseGlyph(p.aB)} · bit ${p.ab}`, AX, CY - 38)
    }
    if (lastArrived >= 0) {
      const p = photons[lastArrived]
      const match = p.s
      ctx.fillStyle = match ? (p.x ? COL.err : COL.green) : COL.muted
      ctx.fillText(`${t('anim.base')} ${baseGlyph(p.bB)} ${match ? '= ✓' : '≠ ✗'}`, BX, CY - 38)
      // ripple at Bob for the last arrival
      const age = tms - resolveAt(p, lastArrived)
      if (!reduced && age >= 0 && age < 450) {
        const k = age / 450
        ctx.strokeStyle = match ? (p.x ? COL.err : COL.green) : COL.dim
        ctx.globalAlpha = 1 - k; ctx.lineWidth = 1.5
        ctx.beginPath(); ctx.arc(BX, CY, 14 + k * 26, 0, 7); ctx.stroke(); ctx.globalAlpha = 1
      }
    }
    // Eve flash
    if (eveOn && !reduced) {
      for (let i = 0; i < photons.length; i++) {
        const p = photons[i]
        if (!p.e) continue
        const age = tms - (i * STAGGER + TRAVEL * (EX - AX) / (BX - AX))
        if (age >= 0 && age < 380 && (p.d || lossAt(p, i) > 0.5)) {
          const k = age / 380
          ctx.strokeStyle = COL.eve; ctx.globalAlpha = 1 - k; ctx.lineWidth = 1.5
          ctx.beginPath(); ctx.arc(EX, CY, 10 + k * 22, 0, 7); ctx.stroke(); ctx.globalAlpha = 1
        }
      }
    }
    onStats && onStats(c)
  }

  function frame(now) {
    raf = requestAnimationFrame(frame)
    elapsed += Math.min(now - lastNow, 100); lastNow = now // no big jumps after a hidden tab
    draw(elapsed)
    const end = photons.length * STAGGER + TRAVEL + 600
    if (elapsed > end) { cancelAnimationFrame(raf); raf = 0; draw(end) }
  }

  return {
    /** Load a new trace (array of photon records) and play it. */
    show(trace, eve) {
      cancelAnimationFrame(raf)
      photons = trace.slice(0, 100); eveOn = eve > 0
      this.replay()
    },
    replay() {
      cancelAnimationFrame(raf)
      elapsed = 0; lastNow = performance.now()
      if (reduced) draw(1e9)
      else raf = requestAnimationFrame(frame)
    },
    redraw() { draw(raf ? elapsed : 1e9) },
  }
}
