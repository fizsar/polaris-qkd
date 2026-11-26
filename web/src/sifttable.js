// SPDX-License-Identifier: AGPL-3.0-only
// Sifting table: 100 placeholder rows, filled photon by photon as the engine
// resolves them. Alice (basis, bit) | Bob (basis, bit) | ✓/✗ | Eve rows in magenta.

import { t } from './i18n.js'

const glyph = (b) => (b ? '×' : '+')

export function createSiftTable(wrap, tbody, reduced) {
  let rows = [], eveOn = false, filled = 0, userScrolled = false
  wrap.addEventListener('wheel', () => { userScrolled = true }, { passive: true })
  wrap.addEventListener('touchmove', () => { userScrolled = true }, { passive: true })

  const td = (txt, cls) => { const c = document.createElement('td'); c.textContent = txt; if (cls) c.className = cls; return c }

  function reset(n, eve) {
    eveOn = eve; filled = 0; userScrolled = false
    wrap.classList.toggle('with-eve', eve)
    tbody.replaceChildren()
    rows = []
    for (let i = 0; i < n; i++) {
      const tr = document.createElement('tr')
      tr.className = 'pending'
      tr.append(td(String(i + 1), 'n'), td('·'), td('·'), td('·'), td('·'), td('·', 'm'), td('·', 'eve'))
      tbody.append(tr); rows.push(tr)
    }
    wrap.scrollTop = 0
  }

  function fill(i, p, first) {
    const tr = rows[i]
    if (!tr) return
    const c = tr.children
    const lost = !p.d
    c[1].textContent = glyph(p.aB); c[2].textContent = String(p.ab)
    c[3].textContent = lost ? '—' : glyph(p.bB); c[4].textContent = lost ? '—' : String(p.bb)
    c[5].textContent = lost ? t('tb.lost') : p.s ? '✓' : '✗'
    c[6].textContent = p.e ? glyph(p.eB) : ''
    tr.className = [lost ? 'lost' : p.s ? 'kept' : 'disc', p.e ? 'eve' : ''].filter(Boolean).join(' ')
    c[4].className = p.s && p.x ? 'bad' : ''
    c[5].className = 'm'
    if (!reduced) { // brief highlight on every (re)fill
      tr.classList.add('flash')
      setTimeout(() => tr.classList.remove('flash'), 600)
    }
    if (first) {
      filled++
      if (!userScrolled && !reduced) wrap.scrollTop = Math.max(0, tr.offsetTop - wrap.clientHeight + tr.offsetHeight * 2 + 30)
    }
  }

  return { reset, fill, retitle() { /* header text is data-i18n */ }, get eveOn() { return eveOn } }
}
