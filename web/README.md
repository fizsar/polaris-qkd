# Polaris web

Static, self-hosted page that runs the simulator in the browser: the Go library
compiled to WebAssembly (TinyGo) inside a Web Worker. No CDN and no third
parties at runtime. **Simulation only, not real QKD.**

```sh
make wasm                 # from the repo root: builds web/public/polaris.wasm (+ wasm_exec.js)
cd web && npm install
npm run dev               # http://127.0.0.1:5174 with hot reload
npm test                  # fast checks of the .wasm under Node
npm run build             # → web/dist (static files)
npm run preview           # dist with the production CSP, http://127.0.0.1:5175
```

* `make wasm` uses TinyGo (≈ 1.0 MB, ≈ 385 KB gzip). `make wasm-go` builds the
  same thing with the standard toolchain (≈ 4.8 MB, ≈ 1.3 MB gzip) — each
  toolchain needs its own `wasm_exec.js`, so always rebuild both together.
* Fonts (JetBrains Mono, OFL) and the FS logo are copied into `public/`.

## Content-Security-Policy (Caddy)

WebAssembly needs `'wasm-unsafe-eval'`; nothing else is relaxed:

```
Content-Security-Policy: default-src 'self'; script-src 'self' 'wasm-unsafe-eval'; style-src 'self'; img-src 'self'; font-src 'self'; worker-src 'self'; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'
```

Serve `.wasm` as `application/wasm` (Caddy does by default) and enable
compression (`encode zstd gzip`). The build uses relative paths, so it works
under any sub-path.
