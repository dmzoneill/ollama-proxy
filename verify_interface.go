// Package main verifies backend interface satisfaction at compile time.
package main

import (
	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/backends/ollama"
)

// This file verifies at compile time that Backend implements Backend interface
func main() {
	// If this compiles, Backend satisfies Backend interface
	var _ backends.Backend = (*ollama.Backend)(nil)
	println("✓ Backend implements all Backend interface methods")
}
