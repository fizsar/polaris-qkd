# SPDX-License-Identifier: AGPL-3.0-only
.PHONY: build test vet plots wasm wasm-go

# WebAssembly build for the web app with TinyGo (≈ 1 MB instead of ≈ 4.8 MB).
# Each toolchain needs its own wasm_exec.js, copied next to the .wasm.
TINYGO ?= tinygo
wasm:
	$(TINYGO) build -o web/public/polaris.wasm -target wasm -no-debug ./cmd/polaris-wasm
	cp "$$($(TINYGO) env TINYGOROOT)/targets/wasm_exec.js" web/public/wasm_exec.js

# Same, with the standard Go toolchain (fallback).
wasm-go:
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -trimpath -o web/public/polaris.wasm ./cmd/polaris-wasm
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" web/public/wasm_exec.js

build:
	go build -o bin/polaris ./cmd/polaris

test:
	go test ./...

vet:
	go vet ./...

# Regenerates the README figures and their CSV data (reproducible: fixed seed).
plots: build
	./bin/polaris sweep --vary distance --from 0 --to 120 --steps 13 --photons 10000000 --seed 1 \
		--out docs/key-rate-vs-distance.csv --svg docs/key-rate-vs-distance.svg
	./bin/polaris sweep --vary eve --from 0 --to 1 --steps 21 --photons 1000000 --runs 3 --seed 1 \
		--out docs/qber-vs-eve.csv --svg docs/qber-vs-eve.svg
