// SPDX-License-Identifier: AGPL-3.0-only
import './style.css'
import { initLang, setLang, getLang, t } from './i18n.js'
import { startSky } from './stars.js'
import { createStage } from './anim.js'
import { renderChart, si } from './charts.js'

const $ = (id) => document.getElementById(id)
const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches


// ── workers ────────────────────────────────────────────────────────────────
function makeWorker() {
  const w = new Worker('./worker.js')
  let seq = 0
  const pending = new Map()
  w.onmessage = (e) => {
    const m = e.data, h = pending.get(m.id)
    if (!h) return
    if (m.type === 'point') h.onPoint?.(m)
    else { pending.delete(m.id); m.type === 'error' ? h.reject(new Error(m.message)) : h.resolve(m.result) }
  }
  w.onerror = (e) => { for (const h of pending.values()) h.reject(new Error(e.message || 'worker error')); pending.clear() }
  return {
    call(msg, onPoint) {
      const id = ++seq
      return new Promise((resolve, reject) => { pending.set(id, { resolve, reject, onPoint }); w.postMessage({ id, ...msg }) })
    },
    kill() { w.terminate(); pending.clear() },
  }
}
const simWorker = makeWorker()
let sweepWorker = null

// ── controls ───────────────────────────────────────────────────────────────
const nf = () => new Intl.NumberFormat(getLang())
const photonsFrom = (v) => Math.round(parseFloat((10 ** v).toPrecision(2)))
const read = () => ({
  distance: +$('in-distance').value,
  noise: +$('in-noise').value / 100,
  eve: +$('in-eve').value / 100,
  photons: photonsFrom(+$('in-photons').value),
})
function syncOutputs() {
  const c = read()
  $('o-distance').textContent = `${c.distance} km`
  $('o-noise').textContent = `${(c.noise * 100).toLocaleString(getLang(), { minimumFractionDigits: 1, maximumFractionDigits: 1 })} %`
  $('o-eve').textContent = `${Math.round(c.eve * 100)} %`
  $('o-photons').textContent = nf().format(c.photons)
}
for (const id of ['in-distance', 'in-noise', 'in-eve', 'in-photons']) $(id).addEventListener('input', syncOutputs)

// ── results ────────────────────────────────────────────────────────────────
let lastResult = null
const pct = (v, d = 2) => `${(v * 100).toLocaleString(getLang(), { minimumFractionDigits: d, maximumFractionDigits: d })} %`
const compact = (v) => new Intl.NumberFormat(getLang(), { notation: 'compact', maximumFractionDigits: 2 }).format(v)

function renderResult(r) {
  lastResult = r
  const v = $('verdict')
  v.hidden = false
  v.className = 'verdict ' + (r.aborted ? 'bad' : 'ok')
  if (r.aborted) {
    const key = `abort.${r.abort_code}`
    const text = t(key, { q: (r.qber * 100).toLocaleString(getLang(), { maximumFractionDigits: 1 }), n: r.sifted_bits, r: r.abort_reason })
    v.innerHTML = `<b>✗ ${t('res.aborted')}</b>${esc(text.startsWith('abort.') ? t('abort.other', { r: r.abort_reason }) : text)}`
  } else {
    v.innerHTML = `<b>✓ ${t('res.ok')}</b>${esc(`${nf().format(r.final_key_bits)} ${t('res.bits')} · ${compact(r.key_rate_bits_per_s)} ${t('res.perSec')}`)}`
  }
  const cells = [
    ['res.detected', `${nf().format(r.photons_detected)} <small>${t('res.of')} ${nf().format(r.photons_sent)}</small>`],
    ['res.sifted', `${nf().format(r.sifted_bits)} <small>${t('res.bits')} · ${pct(r.sift_ratio, 1)} ${t('res.pctOfDetected')}</small>`],
  ]
  if (r.sample_bits_spent) {
    cells.push(['res.qber', `${pct(r.qber)} <small>${t('res.upper')} ${pct(r.qber_upper)}</small>`, r.aborted && r.abort_code === 'qber_too_high' ? 'warnv' : ''])
    cells.push(['res.sample', `${nf().format(r.sample_bits_spent)} <small>${t('res.bits')}</small>`])
  }
  if (r.leak_ec) {
    cells.push(['res.leak', `${nf().format(r.leak_ec)} <small>${t('res.bits')}</small>`])
    cells.push(['res.verify', `${r.verify_bits} <small>${t('res.bits')}</small>`])
  }
  cells.push(['res.final', `${nf().format(r.final_key_bits)} <small>${t('res.bits')}</small>`, r.aborted ? 'warnv' : 'key'])
  if (!r.aborted) {
    cells.push(['res.rate', `${r.key_rate_per_pulse.toPrecision(3)} <small>${t('res.perPulse')}</small><br><small>${compact(r.key_rate_bits_per_s)} ${t('res.perSec')}</small>`])
    cells.push(['res.match', `<span class="${r.keys_match ? 't-green' : 't-eve'}">${t(r.keys_match ? 'res.same' : 'res.diff')}</span>`])
  }
  $('stats').innerHTML = cells.map(([k, val, cls]) => `<div class="stat ${cls || ''}"><dt>${t(k)}</dt><dd>${val}</dd></div>`).join('')
}
const esc = (s) => String(s).replace(/[&<>]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' }[c]))

// ── animation ──────────────────────────────────────────────────────────────
let stage
function animStats(c) {
  $('anim-stats').textContent = t('anim.stats', { k: c.k, e: c.e, d: c.d, l: c.l })
}

let lastCfg = null
async function loadTrace() {
  if (!lastCfg) return
  const only = !$('show-lost').checked
  const trace = await simWorker.call({ op: 'trace', cfg: { ...lastCfg, seed: 0, onlyDetected: only } })
  stage.show(trace, lastCfg.eve)
  renderNote()
}
function renderNote() {
  const note = $('anim-note')
  const r = lastResult
  const p = r && r.photons_sent ? r.photons_detected / r.photons_sent : 1
  note.hidden = $('show-lost').checked || p > 0.9
  note.textContent = t('anim.onlyDetected', { p: (p * 100).toLocaleString(getLang(), { maximumFractionDigits: 1 }) })
}

// ── charts ─────────────────────────────────────────────────────────────────
const chartData = { rate: [], qber: [], cfg: null, working: { rate: false, qber: false } }

function drawCharts() {
  const c = chartData.cfg
  if (!c) return
  const lang = getLang()
  const yfmtRate = (v) => si(v)
  // rate vs distance (+ fibre-loss reference anchored at the first non-zero point)
  const rate = chartData.rate.map((p) => [p.x, p.r.rateBps])
  const first = rate.find(([, y]) => y > 0)
  const lastX = rate.filter(([, y]) => y > 0).at(-1)?.[0]
  const ref = first ? rate.filter(([x]) => x <= lastX).map(([x]) => [x, first[1] * 10 ** (-0.2 * (x - first[0]) / 10)]) : []
  renderChart($('chart-rate'), {
    title: t('ch.rate.t'), xLabel: t('ch.rate.x'), yLabel: t('ch.rate.y'), logY: true, yFmt: yfmtRate,
    xMin: 0, xMax: 150, cls: 'green',
    note: !rate.some(([, y]) => y > 0) ? (chartData.working.rate ? t('ch.working') : t('ch.nokey')) : '',
    series: [{ label: t('ch.sim'), role: 'data', pts: rate }, { label: t('ch.loss'), role: 'ref', pts: ref }],
  })
  // QBER vs Eve (+ theory p + (1−2p)f/4)
  const q = chartData.qber.map((p) => [p.x * 100, p.r.qber * 100])
  const th = chartData.qber.map((p) => [p.x * 100, 100 * (c.noise + (1 - 2 * c.noise) * 0.25 * p.x)])
  renderChart($('chart-qber'), {
    title: t('ch.qber.t'), xLabel: t('ch.qber.x'), yLabel: t('ch.qber.y'),
    xMin: 0, xMax: 100, yMin: 0, yMax: 30, cls: 'eve',
    note: !q.length && chartData.working.qber ? t('ch.working') : '',
    hline: { y: 11, label: t('ch.limit') },
    series: [{ label: t('ch.sim'), role: 'data', pts: q }, { label: t('ch.theory'), role: 'ref', pts: th }],
  })
  void lang
}

function runCharts(cfg) {
  sweepWorker?.kill()
  sweepWorker = makeWorker()
  const w = sweepWorker
  chartData.cfg = cfg
  chartData.rate = []; chartData.qber = []
  chartData.working = { rate: true, qber: true }
  drawCharts()
  const dist = Array.from({ length: 16 }, (_, i) => i * 10)
  const eve = Array.from({ length: 21 }, (_, i) => i / 20)
  const base = { photons: Math.max(cfg.photons, 1_000_000), noise: cfg.noise, seed: 1 }
  w.call({ op: 'sweep', vary: 'distance', xs: dist, base: { ...base, eve: cfg.eve, distance: 0 } }, (m) => {
    chartData.rate[m.i] = { x: m.x, r: m.r }; drawCharts()
  }).then(() => w.call({ op: 'sweep', vary: 'eve', xs: eve, base: { ...base, distance: cfg.distance, eve: 0 } }, (m) => {
    chartData.working.rate = false
    chartData.qber[m.i] = { x: m.x, r: m.r }; drawCharts()
  })).then(() => { chartData.working = { rate: false, qber: false }; drawCharts() })
    .catch((e) => { if (sweepWorker === w) console.error(e) })
}

// ── run ────────────────────────────────────────────────────────────────────
let runId = 0
async function simulate() {
  const id = ++runId
  const cfg = read()
  const btn = $('run')
  btn.disabled = true; btn.textContent = t('ctl.running')
  document.querySelector('.results').classList.add('busy')
  try {
    const res = await simWorker.call({ op: 'sim', cfg: { ...cfg, seed: 0 } })
    if (id !== runId) return
    renderResult(res)
    lastCfg = cfg
    await loadTrace()
    runCharts(cfg)
  } catch (err) {
    console.error(err)
    $('verdict').hidden = false
    $('verdict').className = 'verdict bad'
    $('verdict').innerHTML = `<b>✗ error</b>${esc(err.message)}`
  } finally {
    if (id === runId) {
      btn.disabled = false; btn.textContent = t('ctl.run')
      document.querySelector('.results').classList.remove('busy')
    }
  }
}

// ── boot ───────────────────────────────────────────────────────────────────
function applyLang(l) {
  setLang(l)
  syncOutputs()
  $('run').textContent = $('run').disabled ? t('ctl.running') : t('ctl.run')
  if (lastResult) { renderResult(lastResult); renderNote() }
  drawCharts()
  stage?.redraw()
}

initLang()
startSky($('sky'), reduced)
stage = createStage($('stage'), reduced, animStats)
$('replay').addEventListener('click', () => stage.replay())
$('show-lost').addEventListener('change', () => loadTrace().catch(console.error))
$('controls').addEventListener('submit', (e) => { e.preventDefault(); simulate() })
for (const b of document.querySelectorAll('[data-lang]')) b.addEventListener('click', () => applyLang(b.dataset.lang))
applyLang(getLang())
simulate()
