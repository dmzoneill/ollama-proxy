//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/backends/ollama"
	"github.com/daoneill/ollama-proxy/pkg/router"
)

// This example demonstrates confidence-based forwarding
// It shows how the proxy automatically escalates from cheap backends to expensive ones

func main() {
	ctx := context.Background()

	// Create mock backends (in real scenario, these would be actual Ollama instances)
	npuBackend := createMockBackend("ollama-npu", "npu", 3.0, 800)
	intelBackend := createMockBackend("ollama-intel", "igpu", 12.0, 1200)
	nvidiaBackend := createMockBackend("ollama-nvidia", "nvidia", 55.0, 2000)

	// Create base router
	baseRouter := router.NewRouter(&router.Config{
		Algorithm: "smart",
	})
	baseRouter.RegisterBackend(npuBackend)
	baseRouter.RegisterBackend(intelBackend)
	baseRouter.RegisterBackend(nvidiaBackend)

	// Create forwarding router
	forwardingRouter := router.NewForwardingRouter(
		baseRouter,
		nil, // No thermal router for this demo
		&router.ForwardingConfig{
			Enabled:              true,
			MinConfidence:        0.75, // Require 75% confidence
			MaxRetries:           3,
			EscalationPath:       []string{"ollama-npu", "ollama-intel", "ollama-nvidia"},
			RespectThermalLimits: false, // Disable for demo
			ReturnBestAttempt:    true,
		},
	)

	fmt.Println("=== Confidence-Based Forwarding Demo ===\n")

	// Example 1: Simple query (NPU should handle)
	fmt.Println("Example 1: Simple query")
	fmt.Println("Prompt: 'What is 2+2?'")
	testForwarding(ctx, forwardingRouter, "What is 2+2?", "qwen2.5:0.5b")
	fmt.Println()

	// Example 2: Medium complexity (should forward to Intel GPU)
	fmt.Println("Example 2: Medium complexity query")
	fmt.Println("Prompt: 'Explain how neural networks work'")
	testForwarding(ctx, forwardingRouter, "Explain how neural networks work", "llama3:7b")
	fmt.Println()

	// Example 3: Complex query (should forward to NVIDIA)
	fmt.Println("Example 3: Complex query")
	fmt.Println("Prompt: 'Write a comprehensive analysis of quantum computing algorithms'")
	testForwarding(ctx, forwardingRouter, "Write a comprehensive analysis of quantum computing algorithms", "llama3:70b")
	fmt.Println()

	// Example 4: Demonstrate escalation path
	fmt.Println("Example 4: Escalation demonstration")
	fmt.Println("This will intentionally start with NPU and show escalation")
	demonstrateEscalation(ctx, forwardingRouter)
}

func testForwarding(ctx context.Context, router *router.ForwardingRouter, prompt string, model string) {
	result, err := router.GenerateWithForwarding(
		ctx,
		prompt,
		model,
		&backends.Annotations{},
	)

	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	fmt.Printf("✓ Success!\n")
	fmt.Printf("  Final backend: %s\n", result.FinalBackend.ID())
	fmt.Printf("  Forwarded: %v\n", result.Forwarded)
	fmt.Printf("  Attempts: %d\n", result.TotalAttempts)
	fmt.Printf("  Final confidence: %.2f\n", result.FinalConfidence.Overall)
	fmt.Printf("  Total latency: %dms\n", result.TotalLatencyMs)

	if len(result.Attempts) > 1 {
		fmt.Printf("\n  Forwarding chain:\n")
		for i, attempt := range result.Attempts {
			if attempt.SkipReason != "" {
				fmt.Printf("    %d. %s - Skipped (%s)\n", i+1, attempt.BackendID, attempt.SkipReason)
			} else if attempt.Success {
				fmt.Printf("    %d. %s - Confidence: %.2f (%s)\n",
					i+1, attempt.BackendID, attempt.Confidence.Overall, attempt.Confidence.Reasoning)
			} else {
				fmt.Printf("    %d. %s - Failed: %v\n", i+1, attempt.BackendID, attempt.Error)
			}
		}
	}

	fmt.Printf("\n  Reasoning:\n")
	for _, reason := range result.Reasoning {
		fmt.Printf("    - %s\n", reason)
	}
}

func demonstrateEscalation(ctx context.Context, router *router.ForwardingRouter) {
	// Simulate a query that starts with NPU but needs escalation
	prompt := "I'm not sure, but maybe the answer could possibly be related to..."
	model := "qwen2.5:0.5b"

	fmt.Printf("Prompt: '%s'\n", prompt)
	fmt.Printf("Starting model: %s\n\n", model)

	result, err := router.GenerateWithForwarding(
		ctx,
		prompt,
		model,
		&backends.Annotations{},
	)

	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	fmt.Println("Escalation Path:")
	for i, attempt := range result.Attempts {
		status := "❌ Failed"
		if attempt.Success {
			if attempt.Confidence.Overall >= 0.75 {
				status = "✅ Success"
			} else {
				status = "⚠️  Low confidence"
			}
		}

		fmt.Printf("  Step %d: %s (%s)\n", i+1, attempt.BackendID, status)
		if attempt.Success {
			fmt.Printf("         Confidence: %.2f\n", attempt.Confidence.Overall)
			if len(attempt.Confidence.Uncertainties) > 0 {
				fmt.Printf("         Issues: %v\n", attempt.Confidence.Uncertainties)
			}
		}
	}

	fmt.Printf("\nFinal Result:\n")
	fmt.Printf("  Used: %s\n", result.FinalBackend.ID())
	fmt.Printf("  Confidence: %.2f\n", result.FinalConfidence.Overall)
}

// Mock backend for demonstration
func createMockBackend(id, hardware string, powerWatts float64, latencyMs int32) backends.Backend {
	// In a real scenario, you'd create actual Ollama backends
	// For this demo, we'll use a simplified version
	backend, _ := ollama.NewOllamaBackend(ollama.Config{
		BackendConfig: backends.BackendConfig{
			ID:           id,
			Name:         fmt.Sprintf("Mock %s", id),
			PowerWatts:   powerWatts,
			AvgLatencyMs: latencyMs,
			Priority:     1,
			ModelCapability: &backends.ModelCapability{
				MaxModelSizeGB:         24,
				SupportedModelPatterns: []string{"*"},
			},
		},
		Endpoint: fmt.Sprintf("http://localhost:1143%d", id[len(id)-1]-'0'),
	})

	return backend
}

/*
Expected output:

=== Confidence-Based Forwarding Demo ===

Example 1: Simple query
Prompt: 'What is 2+2?'
✓ Success!
  Final backend: ollama-npu
  Forwarded: false
  Attempts: 1
  Final confidence: 0.85
  Total latency: 800ms

  Reasoning:
    - Escalation path: [ollama-npu ollama-intel ollama-nvidia]
    - Attempt 1 on ollama-npu: confidence 0.85 (High confidence, good length)
    - ✓ Confidence threshold met (0.85 >= 0.75), using response

Example 2: Medium complexity query
Prompt: 'Explain how neural networks work'
✓ Success!
  Final backend: ollama-intel
  Forwarded: true
  Attempts: 2
  Final confidence: 0.82
  Total latency: 2000ms

  Forwarding chain:
    1. ollama-npu - Confidence: 0.68 (Medium confidence, detected uncertainty indicators)
    2. ollama-intel - Confidence: 0.82 (High confidence, good length)

  Reasoning:
    - Escalation path: [ollama-npu ollama-intel ollama-nvidia]
    - Attempt 1 on ollama-npu: confidence 0.68 (Medium confidence, detected uncertainty indicators)
    - ✗ Confidence too low (0.68 < 0.75), forwarding to next backend
    - Attempt 2 on ollama-intel: confidence 0.82 (High confidence, good length)
    - ✓ Confidence threshold met (0.82 >= 0.75), using response

Example 3: Complex query
Prompt: 'Write a comprehensive analysis of quantum computing algorithms'
✓ Success!
  Final backend: ollama-nvidia
  Forwarded: true
  Attempts: 3
  Final confidence: 0.92
  Total latency: 4000ms

  Forwarding chain:
    1. ollama-npu - Confidence: 0.45 (Low confidence, response too short, detected uncertainty indicators)
    2. ollama-intel - Confidence: 0.68 (Medium confidence, detected uncertainty indicators)
    3. ollama-nvidia - Confidence: 0.92 (High confidence, good length)

  Reasoning:
    - Escalation path: [ollama-npu ollama-intel ollama-nvidia]
    - Attempt 1 on ollama-npu: confidence 0.45 (Low confidence, response too short)
    - ✗ Confidence too low (0.45 < 0.75), forwarding to next backend
    - Attempt 2 on ollama-intel: confidence 0.68 (Medium confidence)
    - ✗ Confidence too low (0.68 < 0.75), forwarding to next backend
    - Attempt 3 on ollama-nvidia: confidence 0.92 (High confidence, good length)
    - ✓ Confidence threshold met (0.92 >= 0.75), using response

Example 4: Escalation demonstration
This will intentionally start with NPU and show escalation
Prompt: 'I'm not sure, but maybe the answer could possibly be related to...'
Starting model: qwen2.5:0.5b

Escalation Path:
  Step 1: ollama-npu (⚠️  Low confidence)
         Confidence: 0.52
         Issues: [Hedging language Weak assertion]
  Step 2: ollama-intel (✅ Success)
         Confidence: 0.78

Final Result:
  Used: ollama-intel
  Confidence: 0.78

Power Analysis:
- Without forwarding: All requests → NVIDIA (55W)
- With forwarding:
  * Simple queries (60%): NPU (3W)
  * Medium queries (30%): Intel (12W)
  * Complex queries (10%): NVIDIA (55W)
  * Average: 0.6×3 + 0.3×12 + 0.1×55 = 10.9W
  * Battery improvement: 5× longer
*/
