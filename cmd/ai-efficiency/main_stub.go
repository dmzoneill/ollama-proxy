//go:build !linux

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "ai-efficiency is only supported on Linux (requires D-Bus)")
	os.Exit(1)
}
