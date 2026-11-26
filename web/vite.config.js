// SPDX-License-Identifier: AGPL-3.0-only
import { defineConfig } from 'vite'

// Relative base so the built site works under any path (e.g. /polaris/).
export default defineConfig({
  base: './',
  server: { host: '127.0.0.1', port: 5174, strictPort: true },
  // `npm run preview` serves the production build with the CSP we will set in
  // Caddy. 'wasm-unsafe-eval' is required to compile WebAssembly; nothing else
  // is relaxed (no 'unsafe-inline', no third-party origins).
  preview: {
    host: '127.0.0.1', port: 5175, strictPort: true,
    headers: {
      'Content-Security-Policy': "default-src 'self'; script-src 'self' 'wasm-unsafe-eval'; style-src 'self'; img-src 'self'; font-src 'self'; worker-src 'self'; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'",
    },
  },
  build: { target: 'es2022', assetsInlineLimit: 0 },
})
