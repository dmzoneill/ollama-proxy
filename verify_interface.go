// +build ignore

package main

import (
	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/backends/ollama"
)

// This file verifies at compile time that OllamaBackend implements Backend interface
func main() {
	// If this compiles, OllamaBackend satisfies Backend interface
	var _ backends.Backend = (*ollama.OllamaBackend)(nil)
	println("✓ OllamaBackend implements all Backend interface methods")
}
