// SPDX-License-Identifier: AGPL-3.0-only
// Star field in three layers (far, middle, near) turning at different, very
// slow speeds around Polaris, with a soft twinkle. Some middle-layer stars
// are joined by thin lines routed like circuit traces (straight runs with
// 45° bends). prefers-reduced-motion: a single still frame.

const TAU = Math.PI * 2
const OMEGA = 0.0022 // rad/s for the middle layer (a full turn in ~48 min)
const SPEED = [0.45, 1, 2.4] // far, middle, near

export function startSky(canvas, reduced) {
  const ctx = canvas.getContext('2d')
  let W = 0, H = 0, dpr = 1
  let P = { x: 0, y: 0 }
  let layers = [[], [], []], edges = [], nodes = new Set()
  let raf = 0, last = 0

  // seeded so the sky is the same on every visit
  let s = 0x9e3779b9
  const rnd = () => ((s = (Math.imul(s ^ (s >>> 15), 2246822507) + 0x6d2b79f5) | 0), ((s ^ (s >>> 13)) >>> 0) / 4294967296)

  function build() {
    s = 0x9e3779b9
    const mobile = W < 760
    P = { x: W * (mobile ? 0.8 : 0.78), y: Math.max(H * (mobile ? 0.11 : 0.2), 96) }
    const rMax = Math.hypot(Math.max(P.x, W - P.x), Math.max(P.y, H - P.y)) + 20
    const area = W * H
    const counts = [
      Math.round(Math.min(480, Math.max(130, area / 2600))),
      Math.round(Math.min(190, Math.max(60, area / 7000))),
      Math.round(Math.min(60, Math.max(18, area / 20000))),
    ]
    layers = counts.map((n, li) => {
      const arr = []
      for (let i = 0; i < n; i++) {
        const r = rMax * Math.sqrt(rnd()), th = rnd() * TAU
        const big = rnd() < (li === 2 ? 0.35 : 0.12)
        const base = [0.4, 0.55, 0.9][li]
        arr.push({
          dx: r * Math.cos(th), dy: r * Math.sin(th),
          size: base + rnd() * [0.45, 0.7, 1.0][li] + (big ? 0.55 : 0),
          a: [0.28, 0.4, 0.55][li] + rnd() * 0.4,
          ph: rnd() * TAU, sp: 0.25 + rnd() * 0.7, cyan: rnd() < 0.3,
        })
      }
      return arr
    })

    // Constellations on the middle layer: greedy nearest-neighbour chains.
    const stars = layers[1]
    edges = []; nodes = new Set()
    const used = new Set()
    const dist = (a, b) => Math.hypot(a.dx - b.dx, a.dy - b.dy)
    const chain = (startIdx, len) => {
      let cur = startIdx
      used.add(cur); nodes.add(cur)
      for (let k = 0; k < len; k++) {
        let best = -1, bd = 1e9
        for (let j = 0; j < stars.length; j++) {
          if (used.has(j)) continue
          const d = dist(stars[cur], stars[j])
          if (d > 55 && d < 240 && d < bd) { bd = d; best = j }
        }
        if (best < 0) break
        edges.push(trace(stars[cur], stars[best]))
        used.add(best); nodes.add(best)
        cur = best
      }
    }
    const polarisStar = { dx: 0, dy: 0 }
    let first = -1, bd = 1e9
    stars.forEach((st, j) => { const d = dist(polarisStar, st); if (d > 80 && d < bd) { bd = d; first = j } })
    if (first >= 0) { edges.push(trace(polarisStar, stars[first])); chain(first, 5) }
    const groups = Math.round(stars.length / 9)
    for (let g = 0, tries = 0; g < groups && tries < 300; tries++) {
      const i = Math.floor(rnd() * stars.length)
      if (used.has(i)) continue
      chain(i, 2 + Math.floor(rnd() * 4)); g++
    }
  }

  // A route from a to b: a straight run along the longer axis, then a 45° bend.
  function trace(a, b) {
    const dx = b.dx - a.dx, dy = b.dy - a.dy
    const ax = Math.abs(dx), ay = Math.abs(dy)
    const pts = [[a.dx, a.dy]]
    if (ax >= ay) pts.push([a.dx + Math.sign(dx) * (ax - ay), a.dy])
    else pts.push([a.dx, a.dy + Math.sign(dy) * (ay - ax)])
    pts.push([b.dx, b.dy])
    return pts
  }

  function resize() {
    dpr = Math.min(window.devicePixelRatio || 1, 2)
    W = window.innerWidth; H = window.innerHeight
    canvas.width = Math.round(W * dpr); canvas.height = Math.round(H * dpr)
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    build()
    draw(performance.now() / 1000)
  }

  function draw(t) {
    ctx.clearRect(0, 0, W, H)
    const rot = (li) => {
      const a = reduced ? 0 : t * OMEGA * SPEED[li]
      return [Math.cos(a), Math.sin(a)]
    }

    // circuit traces (middle layer)
    {
      const [c, sn] = rot(1)
      ctx.lineWidth = 1; ctx.lineJoin = 'round'
      ctx.strokeStyle = 'rgba(143,140,255,.26)'
      for (const e of edges) {
        ctx.beginPath()
        e.forEach(([dx, dy], i) => {
          const x = P.x + dx * c - dy * sn, y = P.y + dx * sn + dy * c
          i ? ctx.lineTo(x, y) : ctx.moveTo(x, y)
        })
        ctx.stroke()
      }
    }
    // stars, far → near
    for (let li = 0; li < 3; li++) {
      const [c, sn] = rot(li)
      const arr = layers[li]
      for (let i = 0; i < arr.length; i++) {
        const st = arr[i]
        const x = P.x + st.dx * c - st.dy * sn, y = P.y + st.dx * sn + st.dy * c
        if (x < -10 || x > W + 10 || y < -10 || y > H + 10) continue
        const tw = reduced ? 1 : 0.72 + 0.28 * Math.sin(t * st.sp + st.ph) // soft twinkle
        ctx.fillStyle = st.cyan ? `rgba(160,225,255,${st.a * tw})` : `rgba(235,240,255,${st.a * tw})`
        ctx.beginPath(); ctx.arc(x, y, st.size, 0, TAU); ctx.fill()
        if (li === 2 && st.size > 1.5) { // near layer: a faint halo
          ctx.fillStyle = `rgba(160,225,255,${0.08 * tw})`
          ctx.beginPath(); ctx.arc(x, y, st.size * 3.2, 0, TAU); ctx.fill()
        }
        if (li === 1 && nodes.has(i)) {
          ctx.strokeStyle = 'rgba(143,140,255,.4)'
          ctx.beginPath(); ctx.arc(x, y, st.size + 2.6, 0, TAU); ctx.stroke()
        }
      }
    }
    // Polaris: the brightest star
    const pulse = reduced ? 1 : 0.9 + 0.1 * Math.sin(t * 1.05)
    const R = 78
    const g = ctx.createRadialGradient(P.x, P.y, 0, P.x, P.y, R)
    g.addColorStop(0, `rgba(234,252,255,${0.7 * pulse})`)
    g.addColorStop(0.18, `rgba(150,225,255,${0.3 * pulse})`)
    g.addColorStop(0.5, `rgba(90,200,250,${0.08 * pulse})`)
    g.addColorStop(1, 'rgba(90,200,250,0)')
    ctx.fillStyle = g
    ctx.beginPath(); ctx.arc(P.x, P.y, R, 0, TAU); ctx.fill()
    const L = 58 * pulse
    for (const [dx, dy, w] of [[1, 0, 1.2], [0, 1, 1.2], [0.7071, 0.7071, 0.7], [0.7071, -0.7071, 0.7]]) {
      const len = w > 1 ? L : L * 0.55
      const lg = ctx.createLinearGradient(P.x - dx * len, P.y - dy * len, P.x + dx * len, P.y + dy * len)
      lg.addColorStop(0, 'rgba(234,252,255,0)'); lg.addColorStop(0.5, `rgba(234,252,255,${0.85 * pulse})`); lg.addColorStop(1, 'rgba(234,252,255,0)')
      ctx.strokeStyle = lg; ctx.lineWidth = w
      ctx.beginPath(); ctx.moveTo(P.x - dx * len, P.y - dy * len); ctx.lineTo(P.x + dx * len, P.y + dy * len); ctx.stroke()
    }
    ctx.fillStyle = '#fff'
    ctx.beginPath(); ctx.arc(P.x, P.y, 3.8, 0, TAU); ctx.fill()
    ctx.fillStyle = 'rgba(139,144,165,.8)'
    ctx.font = '11px "JetBrains Mono", ui-monospace, monospace'
    ctx.textAlign = 'left'
    ctx.fillText('α UMi · Polaris', P.x + 22, P.y - 20)
  }

  function frame(now) {
    raf = requestAnimationFrame(frame)
    if (now - last < 33) return // ~30 fps is plenty for a slow sky
    last = now
    draw(now / 1000)
  }

  window.addEventListener('resize', resize)
  resize()
  if (!reduced) {
    raf = requestAnimationFrame(frame)
    document.addEventListener('visibilitychange', () => {
      cancelAnimationFrame(raf)
      if (!document.hidden) raf = requestAnimationFrame(frame)
    })
  }
  return () => { cancelAnimationFrame(raf); window.removeEventListener('resize', resize) }
}
