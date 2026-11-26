# SPDX-License-Identifier: AGPL-3.0-only
.PHONY: build test vet plots

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
