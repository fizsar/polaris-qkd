// SPDX-License-Identifier: AGPL-3.0-only
// ES / EN strings. Elements use data-i18n="key" (textContent) or
// data-i18n-html="key" (trusted, static markup written here).

export const dict = {
  es: {
    'doc.title': 'Polaris — simulador de QKD (BB84)',
    'nav.sim': 'simulador', 'nav.charts': 'gráficas', 'nav.about': 'qué es',
    'hero.sub': 'Simulador de distribución cuántica de claves (BB84)',
    'hero.warn.title': 'Esto es una simulación',
    'hero.warn.body': 'No hay hardware cuántico: los fotones son variables aleatorias en un modelo de software. No es QKD real y las claves que salen no sirven para uso real.',
    'hero.cta': 'Probar el simulador',

    'sim.h': 'Simulador',
    'sim.intro': 'Mueve los deslizadores y pulsa «Simular». El cálculo corre en tu navegador (Go compilado a WebAssembly), en un hilo aparte.',
    'ctl.distance': 'distancia', 'ctl.noise': 'ruido', 'ctl.eve': 'Eve intercepta', 'ctl.photons': 'fotones',
    'ctl.run': 'Simular', 'ctl.running': 'Simulando…',
    'ctl.fixed': 'Fijo: fibra 0,2 dB/km, detector 10 %, cuentas oscuras 1e-6, muestra de QBER 10 %, Cascade, verificación de 64 bits y amplificación por bloques.',
    'res.h': 'Resultado',
    'res.detected': 'fotones detectados', 'res.sifted': 'clave tras el sifting', 'res.qber': 'QBER',
    'res.upper': 'límite superior', 'res.sample': 'gastados en estimar el QBER', 'res.leak': 'revelados por Cascade',
    'res.verify': 'revelados por la verificación', 'res.final': 'clave final', 'res.rate': 'tasa',
    'res.match': 'claves de Alice y Bob', 'res.same': 'idénticas', 'res.diff': 'DISTINTAS',
    'res.of': 'de', 'res.bits': 'bits', 'res.perPulse': 'bit/pulso', 'res.perSec': 'bit/s a 1 GHz',
    'res.pctOfDetected': 'de los detectados',
    'res.ok': 'Clave segura generada', 'res.aborted': 'Protocolo abortado',
    'abort.qber_too_high': 'Alice y Bob discrepan en el {q} % de los bits que compararon. Por encima de ≈ 11 % esos errores podrían deberse a un espía (o a demasiado ruido), así que BB84 descarta la clave en vez de arriesgarse.',
    'abort.no_secret_key': 'El QBER ({q} %) es lo bastante alto como para que corregir los errores revele más bits de los que la clave puede permitirse. No queda nada seguro: la clave se tira.',
    'abort.sifted_too_short': 'Casi ningún fotón sobrevivió a la fibra: quedan solo {n} bits tras el sifting, muy pocos para hacer una clave. Prueba con menos distancia o más fotones.',
    'abort.verification_failed': 'Tras corregir errores las dos claves seguían siendo distintas (pasa con claves muy cortas) y la verificación lo detectó. Se descarta.',
    'abort.other': 'El protocolo se detuvo: {r}',

    'anim.h': 'Los primeros 100 fotones',
    'anim.intro': 'Alice envía fotones con una base (+ o ×) y un bit; Bob mide con una base al azar. Si las bases coinciden, el bit se conserva (sifting). Con Eve, el fotón cambia a magenta: ella mide y reenvía, y a veces se equivoca.',
    'anim.replay': '↻ repetir',
    'anim.showLost': 'mostrar también los fotones perdidos en la fibra',
    'anim.onlyDetected': 'Se muestran los primeros 100 fotones que llegan a Bob; a esta distancia solo llega un {p} % de los enviados.',
    'anim.alice': 'Alice', 'anim.bob': 'Bob', 'anim.eve': 'Eve',
    'leg.photon': 'fotón', 'leg.eve': 'Eve', 'leg.kept': 'bit conservado', 'leg.error': 'conservado con error', 'leg.disc': 'base distinta: descartado', 'leg.lost': 'perdido en la fibra',
    'anim.stats': 'conservados {k} · con error {e} · descartados {d} · perdidos {l}',
    'anim.base': 'base', 'anim.rect': 'rectilínea +', 'anim.diag': 'diagonal ×',

    'ch.h': 'Gráficas',
    'ch.intro': 'Se calculan aquí mismo, con tu ruido y (como mínimo) 10^6 fotones por punto, barriendo el parámetro de cada gráfica. Tasa: distancia de 0 a 150 km con tu Eve; QBER: Eve de 0 a 100 % con tu distancia.',
    'ch.rate.t': 'Tasa de clave según la distancia', 'ch.rate.y': 'tasa de clave (bit/s, escala log)', 'ch.rate.x': 'distancia (km)',
    'ch.qber.t': 'QBER según la fracción interceptada por Eve', 'ch.qber.y': 'QBER (%)', 'ch.qber.x': 'fotones interceptados por Eve (%)',
    'ch.theory': 'teoría', 'ch.sim': 'simulado', 'ch.limit': 'umbral 11 %', 'ch.loss': '∝ pérdida de fibra η',
    'ch.working': 'calculando…', 'ch.nokey': 'sin clave', 'ch.ratio.legend': 'los puntos sin clave no se dibujan',

    'about.h': 'Qué es esto',
    'about.qkd.h': 'Qué es la QKD',
    'about.qkd.p': 'La distribución cuántica de claves permite que dos partes acuerden una clave secreta usando fotones sueltos. Si alguien espía, perturba el estado de los fotones y deja una marca estadística: más errores. Alice y Bob miden esos errores y, si son demasiados, tiran la clave.',
    'about.bb84.h': 'Cómo funciona BB84',
    'about.bb84.1': '<b>Envío.</b> Alice elige un bit y una base (+ o ×) al azar para cada fotón. Bob mide con una base también aleatoria.',
    'about.bb84.2': '<b>Sifting.</b> Comparan en público solo las bases, no los bits, y se quedan con los casos en que coinciden (≈ 50 %).',
    'about.bb84.3': '<b>QBER.</b> Sacrifican una muestra al azar y comparan sus bits. Eve, si intercepta todo, provoca un 25 % de errores. Más de ≈ 11 %: se aborta.',
    'about.bb84.4': '<b>Cascade.</b> Corrigen los errores restantes intercambiando paridades, y cuentan cada bit revelado.',
    'about.bb84.5': '<b>Verificación y amplificación.</b> Un hash confirma que las claves son iguales; una matriz de Toeplitz comprime la clave para borrar lo que Eve pudo saber.',
    'about.lim.h': 'Límites del simulador',
    'about.lim.1': 'Es una simulación: no hay hardware cuántico ni dispositivos reales.',
    'about.lim.2': 'Fuente ideal de fotones únicos: no hay estados señuelo ni ataques multifotón.',
    'about.lim.3': 'Cota asintótica de clave con un margen simple de 3σ: sin análisis de clave finita ni parámetro de seguridad ε.',
    'about.lim.4': 'Eve solo hace intercepta-y-reenvía; la autenticación del canal clásico se da por supuesta.',
    'about.repo': 'Código y detalles en el repositorio',
    'foot.lic': 'AGPL-3.0-only · simulación, no QKD real',
  },
  en: {
    'doc.title': 'Polaris — QKD simulator (BB84)',
    'nav.sim': 'simulator', 'nav.charts': 'charts', 'nav.about': 'about',
    'hero.sub': 'Quantum key distribution simulator (BB84)',
    'hero.warn.title': 'This is a simulation',
    'hero.warn.body': 'There is no quantum hardware: photons are random variables in a software model. It is not real QKD and the keys it produces are not for real use.',
    'hero.cta': 'Try the simulator',

    'sim.h': 'Simulator',
    'sim.intro': 'Move the sliders and press “Simulate”. The computation runs in your browser (Go compiled to WebAssembly), in a separate thread.',
    'ctl.distance': 'distance', 'ctl.noise': 'noise', 'ctl.eve': 'Eve intercepts', 'ctl.photons': 'photons',
    'ctl.run': 'Simulate', 'ctl.running': 'Simulating…',
    'ctl.fixed': 'Fixed: fibre 0.2 dB/km, detector 10 %, dark counts 1e-6, 10 % QBER sample, Cascade, 64-bit verification and block-wise amplification.',
    'res.h': 'Result',
    'res.detected': 'photons detected', 'res.sifted': 'key after sifting', 'res.qber': 'QBER',
    'res.upper': 'upper bound', 'res.sample': 'spent on QBER estimation', 'res.leak': 'revealed by Cascade',
    'res.verify': 'revealed by verification', 'res.final': 'final key', 'res.rate': 'rate',
    'res.match': 'Alice and Bob keys', 'res.same': 'identical', 'res.diff': 'DIFFERENT',
    'res.of': 'of', 'res.bits': 'bits', 'res.perPulse': 'bit/pulse', 'res.perSec': 'bit/s at 1 GHz',
    'res.pctOfDetected': 'of detected',
    'res.ok': 'Secure key generated', 'res.aborted': 'Protocol aborted',
    'abort.qber_too_high': 'Alice and Bob disagree on {q} % of the bits they compared. Above ≈ 11 % those errors could come from an eavesdropper (or too much noise), so BB84 throws the key away instead of taking the risk.',
    'abort.no_secret_key': 'The QBER ({q} %) is high enough that fixing the errors reveals more bits than the key can afford. Nothing safe is left: the key is discarded.',
    'abort.sifted_too_short': 'Almost no photon survived the fibre: only {n} bits are left after sifting, too few to make a key. Try a shorter distance or more photons.',
    'abort.verification_failed': 'After error correction the two keys were still different (this happens with very short keys) and the verification caught it. It is discarded.',
    'abort.other': 'The protocol stopped: {r}',

    'anim.h': 'The first 100 photons',
    'anim.intro': 'Alice sends photons with a basis (+ or ×) and a bit; Bob measures in a random basis. When the bases match the bit is kept (sifting). With Eve the photon turns magenta: she measures and resends, and sometimes gets it wrong.',
    'anim.replay': '↻ replay',
    'anim.showLost': 'also show photons lost in the fibre',
    'anim.onlyDetected': 'Showing the first 100 photons that reach Bob; at this distance only {p} % of the photons sent get there.',
    'anim.alice': 'Alice', 'anim.bob': 'Bob', 'anim.eve': 'Eve',
    'leg.photon': 'photon', 'leg.eve': 'Eve', 'leg.kept': 'bit kept', 'leg.error': 'kept, with an error', 'leg.disc': 'different basis: discarded', 'leg.lost': 'lost in the fibre',
    'anim.stats': 'kept {k} · errors {e} · discarded {d} · lost {l}',
    'anim.base': 'basis', 'anim.rect': 'rectilinear +', 'anim.diag': 'diagonal ×',

    'ch.h': 'Charts',
    'ch.intro': 'Computed right here, with your noise and (at least) 10^6 photons per point, sweeping each chart’s parameter. Rate: distance from 0 to 150 km with your Eve; QBER: Eve from 0 to 100 % with your distance.',
    'ch.rate.t': 'Key rate vs distance', 'ch.rate.y': 'key rate (bit/s, log scale)', 'ch.rate.x': 'distance (km)',
    'ch.qber.t': 'QBER vs fraction intercepted by Eve', 'ch.qber.y': 'QBER (%)', 'ch.qber.x': 'photons intercepted by Eve (%)',
    'ch.theory': 'theory', 'ch.sim': 'simulated', 'ch.limit': '11 % threshold', 'ch.loss': '∝ fibre loss η',
    'ch.working': 'computing…', 'ch.nokey': 'no key', 'ch.ratio.legend': 'points with no key are not drawn',

    'about.h': 'What is this',
    'about.qkd.h': 'What QKD is',
    'about.qkd.p': 'Quantum key distribution lets two parties agree on a secret key using single photons. An eavesdropper disturbs the photons’ state and leaves a statistical mark: more errors. Alice and Bob measure those errors and, if there are too many, throw the key away.',
    'about.bb84.h': 'How BB84 works',
    'about.bb84.1': '<b>Send.</b> Alice picks a random bit and basis (+ or ×) for every photon. Bob measures in a random basis too.',
    'about.bb84.2': '<b>Sifting.</b> They compare only the bases in public, never the bits, and keep the cases where they match (≈ 50 %).',
    'about.bb84.3': '<b>QBER.</b> They sacrifice a random sample and compare its bits. If Eve intercepts everything she causes 25 % errors. Above ≈ 11 %: abort.',
    'about.bb84.4': '<b>Cascade.</b> They fix the remaining errors by exchanging parities, counting every revealed bit.',
    'about.bb84.5': '<b>Verification and amplification.</b> A hash confirms the keys are equal; a Toeplitz matrix compresses the key to erase whatever Eve may know.',
    'about.lim.h': 'Simulator limits',
    'about.lim.1': 'It is a simulation: no quantum hardware or real devices.',
    'about.lim.2': 'Ideal single-photon source: no decoy states and no multi-photon attacks.',
    'about.lim.3': 'Asymptotic key bound with a simple 3σ margin: no finite-key analysis or security parameter ε.',
    'about.lim.4': 'Eve only does intercept-resend; authentication of the classical channel is assumed.',
    'about.repo': 'Code and details in the repository',
    'foot.lic': 'AGPL-3.0-only · simulation, not real QKD',
  },
}

const KEY = 'polaris.lang'
let lang = 'en'

export function initLang() {
  let saved = null
  try { saved = localStorage.getItem(KEY) } catch { /* storage may be blocked */ }
  lang = saved === 'es' || saved === 'en' ? saved : (navigator.language || 'en').toLowerCase().startsWith('es') ? 'es' : 'en'
  return lang
}

export const getLang = () => lang

export function setLang(l) {
  lang = l
  try { localStorage.setItem(KEY, l) } catch { /* ignore */ }
  document.documentElement.lang = l
  document.title = t('doc.title')
  for (const el of document.querySelectorAll('[data-i18n]')) el.textContent = t(el.dataset.i18n)
  for (const el of document.querySelectorAll('[data-i18n-html]')) el.innerHTML = t(el.dataset.i18nHtml)
  for (const b of document.querySelectorAll('[data-lang]')) b.setAttribute('aria-pressed', String(b.dataset.lang === l))
}

export function t(key, vars) {
  let s = dict[lang][key] ?? dict.en[key] ?? key
  if (vars) for (const [k, v] of Object.entries(vars)) s = s.replaceAll(`{${k}}`, v)
  return s
}
