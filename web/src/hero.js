// SPDX-License-Identifier: AGPL-3.0-only
// Hero background: Alice and Bob as two bright stars, photons crossing
// between them as light pulses with their polarisation. White/cyan =
// photon, magenta = Eve. On narrow screens the path turns vertical (Alice on
// top, Bob below). prefers-reduced-motion: a still picture.

import { t } from './i18n.js'
import { TRAVEL, STAGGER, lossAt } from './engine.js'

const COL = {
  photon: '#eafcff', cyan: '90,200,250', eve: '#ff4fd8', eveRGB: '255,79,216',
  green: '#3dffa2', err: '#ff7a7a', dim: '#5a5f75', muted: '#8b90a5', line: 'rgba(143,140,255,.28)',
}
const baseGlyph = (b) => (b ? '×' : '+')
// polarisation angle of a BB84 state (canvas coordinates, y down)
const angle = (base, bit) => (base === 0 ? (bit ? Math.PI / 2 : 0) : bit ? -3 * Math.PI / 4 : -Math.PI / 4)

export function createHero(canvas, engine, reduced) {
  const ctx = canvas.getContext('2d')
  let W = 0, H = 0, dpr = 1, visible = true
  const font = (px, w = 500) => `${w} ${px}px "JetBrains Mono", ui-monospace, monospace`

  function geom() {
    // narrow screens: vertical path under the text, Alice on top and Bob below
    if (W < 900) { const x = W / 2; return { ax: x, ay: H - 330, bx: x, by: H - 70, vertical: true } }
    // wide screens: vertically centred, over the right two thirds (the title lives in the left third)
    const y = Math.round(H / 2)
    return { ax: W * 0.4 + Math.max(50, W * 0.035), ay: y, bx: W - Math.max(70, W * 0.06), by: y, vertical: false }
  }

  function resize() {
    dpr = Math.min(window.devicePixelRatio || 1, 2)
    W = canvas.clientWidth; H = canvas.clientHeight
    canvas.width = Math.round(W * dpr); canvas.height = Math.round(H * dpr)
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    draw(engine)
  }
  new ResizeObserver(resize).observe(canvas)
  new IntersectionObserver(([e]) => { visible = e.isIntersecting; if (visible) draw(engine) }).observe(canvas)

  function star(x, y, r, color, rgb, label, labelColor, g, below = true) {
    const gr = ctx.createRadialGradient(x, y, 0, x, y, r * 5.5)
    gr.addColorStop(0, `rgba(${rgb},.8)`); gr.addColorStop(0.3, `rgba(${rgb},.25)`); gr.addColorStop(1, `rgba(${rgb},0)`)
    ctx.fillStyle = gr; ctx.beginPath(); ctx.arc(x, y, r * 5.5, 0, 7); ctx.fill()
    ctx.strokeStyle = color; ctx.lineWidth = 1.6; ctx.globalAlpha = 0.9
    ctx.beginPath()
    ctx.moveTo(x - r * 2.8, y); ctx.lineTo(x + r * 2.8, y); ctx.moveTo(x, y - r * 2.8); ctx.lineTo(x, y + r * 2.8)
    ctx.stroke(); ctx.globalAlpha = 1
    ctx.fillStyle = color; ctx.beginPath(); ctx.arc(x, y, r * 0.5, 0, 7); ctx.fill()
    // name under the star
    ctx.fillStyle = labelColor || COL.photon; ctx.font = font(r > 12 ? 16 : 14, 600); ctx.textAlign = 'center'
    ctx.fillText(label, x, y + (below ? r * 2.8 + 18 : -r * 2.8 - 6))
  }

  function drawPhoton(g, p, i, prog, alpha, eveOn) {
    const ux = g.bx - g.ax, uy = g.by - g.ay, len = Math.hypot(ux, uy)
    const dx = ux / len, dy = uy / len
    const pp = Math.min(prog, 1)
    const x = g.ax + ux * pp, y = g.ay + uy * pp
    const afterEve = eveOn && p.e && prog >= 0.5
    const rgb = afterEve ? COL.eveRGB : COL.cyan
    const core = afterEve ? COL.eve : COL.photon
    const bs = afterEve ? p.eB : p.aB, bt = afterEve ? p.eb : p.ab
    ctx.globalAlpha = alpha
    // comet-like tail: wide soft band + a bright thin core line
    const tl = Math.min(110, len * pp)
    const tr = ctx.createLinearGradient(x - dx * tl, y - dy * tl, x, y)
    tr.addColorStop(0, `rgba(${rgb},0)`); tr.addColorStop(1, `rgba(${rgb},.9)`)
    ctx.lineCap = 'round'
    ctx.strokeStyle = tr; ctx.lineWidth = 7; ctx.globalAlpha = alpha * 0.28
    ctx.beginPath(); ctx.moveTo(x - dx * tl, y - dy * tl); ctx.lineTo(x, y); ctx.stroke()
    ctx.globalAlpha = alpha; ctx.lineWidth = 2.6
    ctx.beginPath(); ctx.moveTo(x - dx * tl, y - dy * tl); ctx.lineTo(x, y); ctx.stroke()
    ctx.lineCap = 'butt'
    const gl = ctx.createRadialGradient(x, y, 0, x, y, 24)
    gl.addColorStop(0, `rgba(${rgb},.95)`); gl.addColorStop(0.35, `rgba(${rgb},.35)`); gl.addColorStop(1, `rgba(${rgb},0)`)
    ctx.fillStyle = gl; ctx.beginPath(); ctx.arc(x, y, 24, 0, 7); ctx.fill()
    ctx.fillStyle = core; ctx.beginPath(); ctx.arc(x, y, 3.6, 0, 7); ctx.fill()
    // polarisation tick
    const a = angle(bs, bt)
    ctx.strokeStyle = core; ctx.lineWidth = 2
    ctx.beginPath(); ctx.moveTo(x - Math.cos(a) * 12, y - Math.sin(a) * 12); ctx.lineTo(x + Math.cos(a) * 12, y + Math.sin(a) * 12); ctx.stroke()
    ctx.fillStyle = afterEve ? COL.eve : COL.photon; ctx.font = font(13, 700); ctx.textAlign = 'center'
    // basis label: above the photon on a horizontal path, to its right on a vertical one
    if (dy) ctx.fillText(baseGlyph(bs), x + 24, y + 5)
    else ctx.fillText(baseGlyph(bs), x, y - 22)
    ctx.globalAlpha = 1
  }

  function draw(e) {
    if (!W || !visible) return
    ctx.clearRect(0, 0, W, H)
    const g = geom()
    const mx = (g.ax + g.bx) / 2, my = (g.ay + g.by) / 2
    ctx.strokeStyle = COL.line; ctx.lineWidth = 1; ctx.setLineDash([3, 6])
    ctx.beginPath(); ctx.moveTo(g.ax, g.ay); ctx.lineTo(g.bx, g.by); ctx.stroke(); ctx.setLineDash([])
    star(g.ax, g.ay, 16, COL.photon, COL.cyan, t('anim.alice'))
    star(g.bx, g.by, 16, COL.photon, COL.cyan, t('anim.bob'))
    if (e.eveOn) star(mx, my, 12, COL.eve, COL.eveRGB, t('anim.eve'), COL.eve)

    const N = e.photons.length
    if (!N) return
    if (reduced) { // still picture: a dozen frozen photons along the path
      const n = 12
      for (let j = 0; j < n; j++) {
        const i = (j * 7) % N, p = e.photons[i]
        drawPhoton(g, p, i, (j + 0.5) / n, 1, e.eveOn)
      }
      return
    }
    const [k0, k1] = e.kRange()
    for (let k = k0; k <= k1; k++) {
      const i = k % N, p = e.photons[i]
      const s = e.elapsed - k * STAGGER
      const prog = s / TRAVEL
      const lp = p.d ? 2 : lossAt(p, i)
      if (prog > Math.min(1.02, lp + 0.14)) continue
      let alpha = 1
      if (!p.d && prog > lp) alpha = Math.max(0, 1 - (prog - lp) / 0.14)
      drawPhoton(g, p, i, prog, alpha, e.eveOn)
      // Eve flash as the photon crosses her
      if (e.eveOn && p.e && (p.d || lp > 0.5)) {
        const age = s - TRAVEL * 0.5
        if (age >= 0 && age < 400) {
          const q = age / 400
          ctx.strokeStyle = COL.eve; ctx.globalAlpha = 1 - q; ctx.lineWidth = 1.6
          ctx.beginPath(); ctx.arc(mx, my, 12 + q * 26, 0, 7); ctx.stroke(); ctx.globalAlpha = 1
        }
      }
    }
    // what Alice is sending now / what Bob just measured
    ctx.font = font(12); ctx.fillStyle = COL.muted
    const side = g.vertical
    if (e.lastSent >= 0) {
      const p = e.photons[e.lastSent % N]
      ctx.textAlign = side ? 'left' : 'center'
      ctx.fillText(`${t('anim.base')} ${baseGlyph(p.aB)} · bit ${p.ab}`, side ? g.ax + 60 : g.ax, side ? g.ay + 4 : g.ay - 62)
    }
    const la = e.lastArrival
    if (la) {
      const { p } = la
      const col = p.s ? (p.x ? COL.err : COL.green) : COL.muted
      ctx.fillStyle = col; ctx.textAlign = side ? 'left' : 'center'
      ctx.fillText(`${t('anim.base')} ${baseGlyph(p.bB)} ${p.s ? '= ✓' : '≠ ✗'}`, side ? g.bx + 60 : g.bx, side ? g.by + 4 : g.by - 62)
      const age = e.elapsed - la.at
      if (age >= 0 && age < 480) {
        const q = age / 480
        ctx.strokeStyle = col; ctx.globalAlpha = 1 - q; ctx.lineWidth = 1.6
        ctx.beginPath(); ctx.arc(g.bx, g.by, 16 + q * 30, 0, 7); ctx.stroke(); ctx.globalAlpha = 1
      }
    }
  }

  return { draw, resize }
}
