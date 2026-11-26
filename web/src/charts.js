// SPDX-License-Identifier: AGPL-3.0-only
// Tiny dependency-free SVG line charts (same look as the README figures).

const W = 560, H = 360
const M = { l: 62, r: 118, t: 48, b: 52 }

const esc = (s) => String(s).replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]))

export function niceTicks(lo, hi, n) {
  if (!(hi > lo)) hi = lo + 1
  const raw = (hi - lo) / n
  const mag = Math.pow(10, Math.floor(Math.log10(raw)))
  let step = mag
  for (const m of [1, 2, 2.5, 5, 10]) { step = m * mag; if (raw <= step) break }
  const out = []
  for (let v = Math.floor(lo / step) * step; v <= hi + step * 1e-9; v += step) out.push(Math.round(v / step) * step)
  return out
}

export const si = (v) => (v >= 1e9 ? `${v / 1e9}G` : v >= 1e6 ? `${v / 1e6}M` : v >= 1e3 ? `${v / 1e3}k` : `${v}`)

/**
 * spec: { title, xLabel, yLabel, logY, yMin, yMax, yFmt, color, note,
 *         series: [{ label, role: 'data'|'ref', pts: [[x,y],…] }],
 *         hline: { y, label } }
 */
export function renderChart(el, spec) {
  const log = !!spec.logY
  const ok = (y) => !log || y > 0
  let xmin = Infinity, xmax = -Infinity, ymin = Infinity, ymax = -Infinity
  for (const s of spec.series) for (const [x, y] of s.pts) {
    if (!ok(y)) continue
    xmin = Math.min(xmin, x); xmax = Math.max(xmax, x); ymin = Math.min(ymin, y); ymax = Math.max(ymax, y)
  }
  if (spec.hline) { ymin = Math.min(ymin, spec.hline.y); ymax = Math.max(ymax, spec.hline.y) }
  if (!isFinite(xmin)) { xmin = spec.xMin ?? 0; xmax = spec.xMax ?? 1; ymin = log ? 1 : 0; ymax = log ? 10 : 1 }
  if (spec.xMax !== undefined) { xmin = spec.xMin; xmax = spec.xMax }
  const xt = niceTicks(xmin, xmax, 6)
  xmin = Math.min(xmin, xt[0]); xmax = Math.max(xmax, xt[xt.length - 1])

  let yt
  if (log) {
    let lo = Math.floor(Math.log10(ymin)), hi = Math.ceil(Math.log10(ymax))
    if (hi === lo) hi++
    yt = []; for (let e = lo; e <= hi; e++) yt.push(Math.pow(10, e))
    ymin = yt[0]; ymax = yt[yt.length - 1]
  } else {
    if (spec.yMax !== undefined) { ymin = spec.yMin; ymax = spec.yMax } else ymin = Math.min(0, ymin)
    yt = niceTicks(ymin, ymax, 5)
    ymin = Math.min(ymin, yt[0]); ymax = Math.max(ymax, yt[yt.length - 1])
  }

  const pw = W - M.l - M.r, ph = H - M.t - M.b
  const px = (x) => M.l + ((x - xmin) / (xmax - xmin)) * pw
  const py = (y) => log
    ? M.t + ph - ((Math.log10(y) - Math.log10(ymin)) / (Math.log10(ymax) - Math.log10(ymin))) * ph
    : M.t + ph - ((y - ymin) / (ymax - ymin)) * ph
  const yf = spec.yFmt || ((v) => `${v}`)
  const cls = spec.cls || 'green'

  let o = `<svg viewBox="0 0 ${W} ${H}" role="img" aria-label="${esc(spec.title)}" xmlns="http://www.w3.org/2000/svg">`
  o += `<text x="${M.l}" y="24" class="c-t">${esc(spec.title)}</text>`
  for (const v of yt) {
    const y = py(v)
    o += `<line x1="${M.l}" x2="${M.l + pw}" y1="${y.toFixed(1)}" y2="${y.toFixed(1)}" class="c-grid"/>`
    o += `<text x="${M.l - 8}" y="${(y + 4).toFixed(1)}" text-anchor="end" class="c-tk">${esc(yf(v))}</text>`
  }
  for (const v of xt) {
    const x = px(v)
    o += `<line x1="${x.toFixed(1)}" x2="${x.toFixed(1)}" y1="${M.t + ph}" y2="${M.t + ph + 4}" class="c-axis"/>`
    o += `<text x="${x.toFixed(1)}" y="${M.t + ph + 18}" text-anchor="middle" class="c-tk">${v}</text>`
  }
  o += `<line x1="${M.l}" x2="${M.l + pw}" y1="${M.t + ph}" y2="${M.t + ph}" class="c-axis"/>`
  o += `<text x="${M.l + pw / 2}" y="${H - 10}" text-anchor="middle" class="c-ax">${esc(spec.xLabel)}</text>`
  o += `<text transform="translate(15 ${M.t + ph / 2}) rotate(-90)" text-anchor="middle" class="c-ax">${esc(spec.yLabel)}</text>`

  if (spec.hline) {
    const y = py(spec.hline.y)
    o += `<line x1="${M.l}" x2="${M.l + pw}" y1="${y.toFixed(1)}" y2="${y.toFixed(1)}" class="c-lim"/>`
    o += `<text x="${M.l + pw + 8}" y="${(y + 4).toFixed(1)}" class="c-lb c-limt">${esc(spec.hline.label)}</text>`
  }

  const dataEnds = []
  for (const s of spec.series) if (s.role === 'data') {
    const p = s.pts.filter(([, y]) => ok(y)).at(-1)
    if (p) dataEnds.push(py(p[1]))
  }
  for (const pass of [0, 1]) for (const s of spec.series) {
    if ((s.role === 'data') !== (pass === 1)) continue
    const pts = s.pts.filter(([, y]) => ok(y))
    if (!pts.length) continue
    const coords = pts.map(([x, y]) => [px(x), py(y)])
    o += `<polyline points="${coords.map((c) => c.map((n) => n.toFixed(1)).join(',')).join(' ')}" class="${s.role === 'data' ? 'c-data ' + cls : 'c-ref'}"/>`
    if (s.role === 'data') pts.forEach(([x, y], i) => {
      o += `<circle cx="${coords[i][0].toFixed(1)}" cy="${coords[i][1].toFixed(1)}" r="3.4" class="c-dot ${cls}"><title>${esc(`${x} → ${yf(+y.toPrecision(4))}`)}</title></circle>`
    })
    let ly = coords.at(-1)[1]
    if (s.role !== 'data') for (const de of dataEnds) if (Math.abs(ly - de) < 34) ly = de + 34 < M.t + ph - 6 ? de + 34 : de - 34
    esc(s.label).split('\n').forEach((ln, i) => {
      o += `<text x="${(coords.at(-1)[0] + 9).toFixed(1)}" y="${(ly + 4 + i * 14).toFixed(1)}" class="c-lb ${s.role === 'data' ? cls : 'c-mut'}">${ln}</text>`
    })
  }
  if (spec.note) o += `<text x="${M.l + pw / 2}" y="${M.t + ph / 2}" text-anchor="middle" class="c-note">${esc(spec.note)}</text>`
  o += '</svg>'
  el.innerHTML = o
}
