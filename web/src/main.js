// SPDX-License-Identifier: AGPL-3.0-only
import './style.css'
import { initLang, setLang, getLang, t } from './i18n.js'
import { startSky } from './stars.js'
import { createEngine } from './engine.js'
import { createHero } from './hero.js'
import { createSiftTable } from './sifttable.js'
import { renderChart, si } from './charts.js'

const $ = (id) => document.getElementById(id)
const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
const debounce = (fn, ms) => { let h; return (...a) => { clearTimeout(h); h = setTimeout(() => fn(...a), ms) } }
const esc = (s) => String(s).replace(/[&<>]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' }[c]))

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

// ── results: verdict, final key (count-up) and QBER gauge on top; the rest in "Details" ──
let lastResult = null
const pct = (v, d = 2) => `${(v * 100).toLocaleString(getLang(), { minimumFractionDigits: d, maximumFractionDigits: d })} %`
const compact = (v) => new Intl.NumberFormat(getLang(), { notation: 'compact', maximumFractionDigits: 2 }).format(v)

let countRaf = 0
function countUp(el, to) {
  cancelAnimationFrame(countRaf)
  const from = +(el.dataset.v || 0)
  el.dataset.v = String(to)
  if (reduced || from === to) { el.textContent = nf().format(to); return }
  const t0 = performance.now(), dur = 700
  const step = (now) => {
    const k = Math.min(1, (now - t0) / dur), e = 1 - (1 - k) ** 3
    el.textContent = nf().format(Math.round(from + (to - from) * e))
    if (k < 1) countRaf = requestAnimationFrame(step)
  }
  countRaf = requestAnimationFrame(step)
}

function abortText(r) {
  const text = t(`abort.${r.abort_code}`, { q: (r.qber * 100).toLocaleString(getLang(), { maximumFractionDigits: 1 }), n: r.sifted_bits, r: r.abort_reason })
  return text.startsWith('abort.') ? t('abort.other', { r: r.abort_reason }) : text
}

function renderResult(r, cfg = lastCfg) {
  lastResult = r
  const ok = !r.aborted
  const highQber = r.abort_code === 'qber_too_high' || r.abort_code === 'no_secret_key'
  // verdict: secure key (green) or key discarded (magenta)
  $('v-card').className = 'card verdict-card ' + (ok ? 'ok' : 'bad')
  $('v-title').textContent = ok ? t('v.ok')
    : highQber ? t('v.qberTitle')
    : r.abort_code === 'sifted_too_short' ? t('v.signalTitle') : t('v.diffTitle')
  const vt = $('v-text')
  vt.replaceChildren()
  if (ok) vt.append(`${t('res.ok')} · ${nf().format(r.final_key_bits)} ${t('res.bits')}`)
  else {
    vt.append(highQber ? t('v.qberNote') : abortText(r))
    if (highQber && cfg && cfg.eve > 0) {
      const s = document.createElement('span')
      s.className = 'eve-note'; s.textContent = ' ' + t('v.eveNote')
      vt.append(s)
    }
  }
  // final key
  countUp($('k-num'), r.final_key_bits)
  document.querySelector('.key-card').classList.toggle('dead', !ok)
  $('k-rate').textContent = ok ? `${r.key_rate_per_pulse.toPrecision(3)} ${t('res.perPulse')} · ${compact(r.key_rate_bits_per_s)} ${t('res.perSec')}` : ''
  // QBER gauge, scale 0–30 % with the 11 % line marked
  const g = $('gauge')
  if (r.sample_bits_spent) {
    $('q-val').textContent = pct(r.qber)
    g.style.setProperty('--w', '0%')
    $('g-fill').style.setProperty('--w', `${Math.min(r.qber, 0.3) / 0.3 * 100}%`)
    $('g-up').style.setProperty('--w', `${Math.min(r.qber_upper, 0.3) / 0.3 * 100}%`)
    g.classList.toggle('bad', !ok)
    $('q-sub').textContent = `${t('res.upper')} ${pct(r.qber_upper)} · ${r.sample_errors}/${nf().format(r.sample_bits_spent)}`
  } else {
    $('q-val').textContent = '—'
    $('g-fill').style.setProperty('--w', '0%'); $('g-up').style.setProperty('--w', '0%')
    $('q-sub').textContent = ''
  }
  // details
  const cells = [
    ['res.detected', `${nf().format(r.photons_detected)} <small>${t('res.of')} ${nf().format(r.photons_sent)}</small>`],
    ['res.sifted', `${nf().format(r.sifted_bits)} <small>${t('res.bits')} · ${pct(r.sift_ratio, 1)} ${t('res.pctOfDetected')}</small>`],
  ]
  if (r.sample_bits_spent) {
    cells.push(['res.qber', `${pct(r.qber)} <small>${t('res.upper')} ${pct(r.qber_upper)}</small>`, !ok && highQber ? 'warnv' : ''])
    cells.push(['res.sample', `${nf().format(r.sample_bits_spent)} <small>${t('res.bits')}</small>`])
  }
  if (r.leak_ec) {
    cells.push(['res.leak', `${nf().format(r.leak_ec)} <small>${t('res.bits')}</small>`])
    cells.push(['res.verify', `${r.verify_bits} <small>${t('res.bits')}</small>`])
  }
  cells.push(['res.final', `${nf().format(r.final_key_bits)} <small>${t('res.bits')}</small>`, ok ? 'key' : 'warnv'])
  if (ok) {
    cells.push(['res.rate', `${r.key_rate_per_pulse.toPrecision(3)} <small>${t('res.perPulse')}</small><br><small>${compact(r.key_rate_bits_per_s)} ${t('res.perSec')}</small>`])
    cells.push(['res.match', `<span class="${r.keys_match ? 't-green' : 't-eve'}">${t(r.keys_match ? 'res.same' : 'res.diff')}</span>`])
  }
  $('stats').innerHTML = cells.map(([k, val, cls]) => `<div class="stat ${cls || ''}"><dt>${t(k)}</dt><dd>${val}</dd></div>`).join('')
}

// ── hero + sifting table, both fed by one engine ──────────────────────────
let hero = null
const table = createSiftTable($('sift-wrap'), $('sift-body'), reduced)
const engine = createEngine(reduced, {
  onLoad: (e) => { table.reset(e.photons.length, e.eveOn); $('sift-stats').textContent = '' },
  onResolve: (i, p, first) => table.fill(i, p, first),
  onStats: (c) => { $('sift-stats').textContent = t('anim.stats', c) },
  onFrame: (e) => hero?.draw(e),
})
hero = createHero($('hero-canvas'), engine, reduced)

let lastCfg = null, lastSeed = 0
async function loadTrace() {
  if (!lastCfg) return
  const only = !$('show-lost').checked
  const trace = await simWorker.call({ op: 'trace', cfg: { ...lastCfg, seed: lastSeed, onlyDetected: only } })
  engine.load(trace, lastCfg.eve)
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
  const rate = chartData.rate.map((p) => [p.x, p.r.rateBps])
  const first = rate.find(([, y]) => y > 0)
  const lastX = rate.filter(([, y]) => y > 0).at(-1)?.[0]
  const ref = first ? rate.filter(([x]) => x <= lastX).map(([x]) => [x, first[1] * 10 ** (-0.2 * (x - first[0]) / 10)]) : []
  renderChart($('chart-rate'), {
    title: t('ch.rate.t'), xLabel: t('ch.rate.x'), yLabel: t('ch.rate.y'), logY: true, yFmt: si,
    xMin: 0, xMax: 150, cls: 'green',
    note: !rate.some(([, y]) => y > 0) ? (chartData.working.rate ? t('ch.working') : t('ch.nokey')) : '',
    series: [{ label: t('ch.sim'), role: 'data', pts: rate }, { label: t('ch.loss'), role: 'ref', pts: ref }],
  })
  const q = chartData.qber.map((p) => [p.x * 100, p.r.qber * 100])
  const th = chartData.qber.map((p) => [p.x * 100, 100 * (c.noise + (1 - 2 * c.noise) * 0.25 * p.x)])
  renderChart($('chart-qber'), {
    title: t('ch.qber.t'), xLabel: t('ch.qber.x'), yLabel: t('ch.qber.y'),
    xMin: 0, xMax: 100, yMin: 0, yMax: 30, cls: 'eve',
    note: !q.length && chartData.working.qber ? t('ch.working') : '',
    hline: { y: 11, label: t('ch.limit') },
    series: [{ label: t('ch.sim'), role: 'data', pts: q }, { label: t('ch.theory'), role: 'ref', pts: th }],
  })
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
// charts are heavy (≈ 37 simulations): wait until the sliders settle
const scheduleCharts = debounce((cfg) => runCharts(cfg), 1200)

// ── run ────────────────────────────────────────────────────────────────────
let runId = 0
// live = triggered by a slider: fixed seed, so only the parameter changes between
// runs; the button uses a fresh random seed.
async function simulate({ live = false } = {}) {
  const id = ++runId
  const cfg = read()
  const seed = live ? 29 : 0
  const btn = $('run')
  btn.disabled = true
  $('results').classList.add('busy')
  try {
    const res = await simWorker.call({ op: 'sim', cfg: { ...cfg, seed } })
    if (id !== runId) return
    renderResult(res, cfg)
    lastCfg = cfg; lastSeed = seed
    await loadTrace()
    scheduleCharts(cfg)
  } catch (err) {
    console.error(err)
    $('v-card').className = 'card verdict-card bad'
    $('v-title').textContent = 'error'
    $('v-text').textContent = err.message
  } finally {
    if (id === runId) { btn.disabled = false; $('results').classList.remove('busy') }
  }
}
const liveSimulate = debounce(() => simulate({ live: true }), 250)

// ── boot ───────────────────────────────────────────────────────────────────
function applyLang(l) {
  setLang(l)
  syncOutputs()
  if (lastResult) { renderResult(lastResult); renderNote() }
  engine.photons.forEach((p, i) => engine.slots[i] && table.fill(i, p, false))
  if (engine.photons.length) $('sift-stats').textContent = t('anim.stats', engine.counts())
  drawCharts()
  hero?.draw(engine)
}

initLang()
startSky($('sky'), reduced)
$('show-lost').addEventListener('change', () => loadTrace().catch(console.error))
$('controls').addEventListener('submit', (e) => { e.preventDefault(); simulate() })
for (const id of ['in-distance', 'in-noise', 'in-eve', 'in-photons']) {
  $(id).addEventListener('input', () => { syncOutputs(); liveSimulate() })
}
for (const b of document.querySelectorAll('[data-lang]')) b.addEventListener('click', () => applyLang(b.dataset.lang))
applyLang(getLang())
simulate({ live: true })
