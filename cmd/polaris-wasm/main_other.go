// SPDX-License-Identifier: AGPL-3.0-only

//go:build !(js && wasm)

// Command polaris-wasm only does something when built for js/wasm:
//
//	GOOS=js GOARCH=wasm go build -o web/public/polaris.wasm ./cmd/polaris-wasm
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "polaris-wasm must be built with GOOS=js GOARCH=wasm")
	os.Exit(2)
}
