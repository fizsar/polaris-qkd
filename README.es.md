# Polaris — simulador de QKD con BB84

Español · [English](README.md)

> **⚠️ Esto es una SIMULACIÓN.** No hay hardware cuántico: los fotones son
> variables aleatorias en un modelo de software. Polaris **no** es distribución
> cuántica de claves real, sus claves **no** sirven para uso real y nunca debe
> presentarse como QKD. Sirve para estudiar el protocolo y su postprocesado, y
> como núcleo reutilizable para las fases siguientes (ver la
> [hoja de ruta](#hoja-de-ruta)).

Polaris simula el protocolo BB84 completo (Bennett y Brassard, 1984) con
fotones individuales sobre una fibra, incluido todo el postprocesado clásico:

1. **Fase cuántica** — Alice elige bits y bases al azar; Bob mide en bases aleatorias.
2. **Sifting** — se conservan las posiciones donde coinciden las bases (≈ 50 %).
3. **Estimación del QBER** — una muestra aleatoria de la clave se compara en público y se descarta. Se aborta por encima del umbral de BB84 (≈ 11 %, Shor–Preskill).
4. **Corrección de errores** — Cascade (Brassard y Salvail, 1993), contando cada bit de paridad revelado (`leak_EC`).
5. **Verificación** — hash universal (Toeplitz) de 64 bits de las claves corregidas.
6. **Amplificación de privacidad** — matriz de Toeplitz aleatoria, por bloques.

Escrito en Go y sin dependencias; primero como librería (para reutilizarla y
compilarla después a WebAssembly) y después como CLI.

## Primeros pasos

```sh
go build -o bin/polaris ./cmd/polaris

# una simulación legible (--lang es para castellano, --json para JSON)
bin/polaris simulate --photons 1000000 --distance 25 --noise 0.02 --eve 0 --seed 1 --lang es

# barrer distancia, ruido o fracción de Eve → CSV (y opcionalmente un SVG)
bin/polaris sweep --vary distance --from 0 --to 120 --steps 13 --photons 10000000 --seed 1 \
    --out distancia.csv --svg distancia.svg
```

Sin `--seed` se usa `crypto/rand`; con `--seed` cada ejecución es reproducible.
`make plots` regenera las gráficas de abajo.

### Ejemplo de salida

```
*** SIMULACIÓN — sin hardware cuántico. No es QKD real; las claves no sirven para uso real. ***

Enlace: 1000000 fotones, 25.0 km (η = 0.3162), detector 10.0 %, ruido 2.0 %, Eve 0 %

  Fotones enviados                   1000000
  Fotones detectados                 31715  (oscuras 3, dobles 0)
  Clave tras el sifting              15804 bits  (49.8 % de los detectados)
  Gastados en estimar el QBER        1580 bits (descartados)
  QBER                               2.15 %  (34/1580);  cota superior 3.25 %
  Revelados por Cascade (leak_EC)    2244 bits  (eficiencia 1.15)
  Revelados por la verificación      64 bits
  Clave secreta final                8912 bits  (1 bloques)
  Tasa de clave                      0.00891 bit/pulso  =  8.912e+06 bit/s @ 1 GHz
  Claves de Alice y Bob idénticas    sí
```

Si Eve intercepta todo, el QBER sube a ≈ 25 % y el protocolo aborta explicando
el motivo. `--json` imprime todos los campos (fotones enviados y detectados,
clave tras el sifting, QBER, bits revelados, longitud final, tasa de clave,
motivo del aborto…).

### Opciones principales

| Opción | Por defecto | Significado |
|---|---|---|
| `--photons` | 1 000 000 | fotones enviados |
| `--distance` | 0 | longitud de fibra en km (transmitancia η = 10^(−0,2·L/10)) |
| `--noise` | 0,01 | tasa de error intrínseca (desalineamiento) |
| `--eve` | 0 | fracción de fotones que Eve intercepta y reenvía |
| `--detector-eff` | 0,1 | eficiencia del detector |
| `--dark-counts` | 1e-6 | probabilidad de cuenta oscura por detector y puerta |
| `--sample-fraction` | 0,10 | parte de la clave gastada en estimar el QBER |
| `--sigma` | 3 | z en `Q_sup = Q + z·σ` (ver abajo) |
| `--block-bits` | 65536 | tamaño de bloque de la amplificación (0 = un solo bloque) |
| `--seed` | — | ejecución reproducible |

## Gráficas

Las dos salen de `polaris sweep` (semilla fija; datos en [`docs/`](docs)).

![Tasa de clave según la distancia](docs/key-rate-vs-distance.svg)

La tasa sigue la pérdida de la fibra η (línea discontinua) hasta que la clave es
tan corta que los costes fijos —hash de verificación de 64 bits y margen de 64
bits por bloque— y el margen del QBER pesan, hacia los 80 km con estos
parámetros.

![QBER según la fracción interceptada por Eve](docs/qber-vs-eve.svg)

Intercepta-y-reenvía da QBER = 25 % × f en la clave filtrada (más el ruido
intrínseco). En la práctica el protocolo se detiene **antes** del umbral del
11 %: con Cascade a eficiencia ≈ 1,2, los bits revelados superan `n·(1−h(Q))`
ya hacia Q ≈ 9–10 % y la ejecución aborta con *no queda clave secreta*.

## Cómo se calcula la longitud de la clave

```
Q̂     = errores / muestra                     (QBER de la muestra descartada)
Q_sup = Q̂ + 3·σ,  σ = sqrt(Q̃(1−Q̃)/n_m),  Q̃ = max(Q̂, 1/n_m)
ℓ     = Σ_bloques [ n_b·(1 − h(Q_sup)) − fuga_b − margen ]   con fuga = leak_EC + bits de verificación
```

`h` es la entropía binaria, `n_m` el tamaño de la muestra, `fuga_b` la parte
proporcional de los bits revelados que toca al bloque (redondeada hacia arriba)
y el margen (64 bits) se descuenta una vez por bloque. Es la **cota asintótica
para fotones individuales**. El `+ 3σ` solo cubre el error de muestreo de la
estimación del QBER con una aproximación normal; **no** es una cota rigurosa de
clave finita.

## Decisiones de modelado

* Pérdidas: η = 10^(−αL/10), α = 0,2 dB/km; un fotón se detecta con probabilidad η × eficiencia del detector.
* Error intrínseco: el bit medido se invierte con la probabilidad configurada.
* Eve: intercepta-y-reenvía en una fracción *f* de los fotones, base aleatoria, detector y fuente ideales junto a Alice (no añade pérdidas).
* Cuentas oscuras y dobles clics: Bob no sabe qué bit era, así que asigna un **bit aleatorio uniforme**, como se hace en la práctica. Un fotón más una cuenta oscura en la misma puerta cuenta como doble clic (algo pesimista).
* Cascade: versión original de cuatro pasadas, primer bloque ≈ 0,73/Q, que se duplica en cada pasada, con retroceso. Eficiencia medida de 1,12 a 1,19 para Q = 1–8 %.
* La amplificación de privacidad es **por bloques** (65 536 bits por defecto). El coste de Toeplitz crece como n·ℓ: con detector perfecto y 1 M de fotones (500 k bits tras el sifting) un solo bloque tarda ≈ 10 s y los bloques de 2^16 bits ≈ 2 s en total, a cambio de ≈ 0,1 % de la clave. Es una simplificación habitual en las implementaciones. Con el detector por defecto (10 %) ambas opciones tardan ≈ 0,1 s.
* Aleatoriedad: `crypto/rand` en uso normal; un generador ChaCha8 con semilla para ejecuciones y tests reproducibles (una ejecución con semilla nunca sirve como material de clave).

## Tests

`go test ./...` (unos segundos). Validan la física con tolerancias estadísticas (≈ 4σ, semillas fijas):

* sin Eve y sin ruido → QBER = 0 y claves idénticas; con ruido *p* → QBER ≈ *p*;
* Eve al 100 % → QBER ≈ 25 %; con fracción *f* → ≈ 0,25·*f*;
* el sifting conserva ≈ 50 % de las detecciones; tasa de detección = η × eficiencia;
* por encima de ≈ 11 % el protocolo aborta, indicando el motivo;
* Cascade deja las claves idénticas y su fuga se mantiene entre 1,0 y 1,3 veces la cota de Shannon;
* Toeplitz es lineal, coincide con su definición y da la longitud esperada;
* la tasa de clave cae con la distancia en proporción a η;
* con 20 semillas, el QBER medio con un 2 % de ruido es el 2 % más la aportación de las cuentas oscuras.

## Limitaciones

* **Solo simulación**; modelo idealizado de fotón único, sin imperfecciones reales de los dispositivos ni canales laterales.
* **Sin estados señuelo**: no se modelan ataques multifotón (PNS); la fuente es un emisor ideal de fotones individuales.
* **Sin análisis de clave finita**: la longitud es la cota asintótica más un margen simple de 3σ. No hay parámetro de seguridad componible ε.
* Eve solo hace intercepta-y-reenvía (ataques individuales), no ataques colectivos ni coherentes.
* La autenticación del canal clásico se da por supuesta, no se simula.

## Hoja de ruta

1. **Fase 1 (este repo):** simulador BB84 y librería núcleo reutilizable.
2. **Fase 2:** gestor de claves con la API ETSI GS QKD 014.
3. **Fase 3:** túnel híbrido QKD + ML-KEM.
4. **Fase 4:** red con nodos de confianza.

## Estructura

| Paquete | Función |
|---|---|
| `rng` | aleatoriedad con `crypto/rand` y con semilla |
| `channel` | pérdida de fibra, detector, cuentas oscuras, error intrínseco, Eve |
| `bb84` | intercambio cuántico y sifting |
| `qber` | muestreo, estimación del QBER y cota superior |
| `cascade` | corrección de errores |
| `privacy` | entropía binaria, hash de Toeplitz, verificación, amplificación por bloques |
| `protocol` | todo el flujo: `protocol.Run(cfg, src)` |
| `plot` | gráficas SVG sin dependencias |
| `cmd/polaris` | CLI |

## Referencias

* C. H. Bennett y G. Brassard, *Quantum cryptography: public key distribution and coin tossing*, Proc. IEEE Int. Conf. on Computers, Systems and Signal Processing, Bangalore, pp. 175–179 (1984).
* P. W. Shor y J. Preskill, *Simple proof of security of the BB84 quantum key distribution protocol*, Phys. Rev. Lett. 85, 441 (2000).
* G. Brassard y L. Salvail, *Secret-key reconciliation by public discussion*, EUROCRYPT ’93, LNCS 765, pp. 410–423 (1994).
* V. Scarani et al., *The security of practical quantum key distribution*, Rev. Mod. Phys. 81, 1301 (2009). <https://arxiv.org/abs/0802.4155>
* S. Pirandola et al., *Advances in quantum cryptography*, Adv. Opt. Photon. 12, 1012 (2020). <https://arxiv.org/abs/1906.01645>

## Licencia

[AGPL-3.0-only](LICENSE).
