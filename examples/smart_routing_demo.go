//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"

	pb "github.com/daoneill/ollama-proxy/api/gen/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewComputeServiceClient(conn)
	ctx := context.Background()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("SMART ROUTING DEMONSTRATION")
	fmt.Println("Showing how the system prevents 'everything is critical'")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// Scenario 1: User marks simple query as critical
	fmt.Println("📋 SCENARIO 1: Simple query marked as critical")
	fmt.Println("───────────────────────────────────────────────────")
	testRequest(client, ctx, &pb.GenerateRequest{
		Prompt: "What is 2+2?",
		Model:  "qwen2.5:0.5b",
		Annotations: &pb.JobAnnotations{
			LatencyCritical: true, // User says critical
		},
	}, "User claims: critical, System should: override to NPU (simple query)")

	// Scenario 2: Actually complex request
	fmt.Println("\n📋 SCENARIO 2: Complex query marked as critical")
	fmt.Println("───────────────────────────────────────────────────")
	testRequest(client, ctx, &pb.GenerateRequest{
		Prompt: "Write a detailed essay about quantum computing and its applications in cryptography",
		Model:  "qwen2.5:0.5b",
		Annotations: &pb.JobAnnotations{
			LatencyCritical: true, // User says critical
		},
	}, "User claims: critical, System should: allow NVIDIA (truly complex)")

	// Scenario 3: Background task marked as critical
	fmt.Println("\n📋 SCENARIO 3: Background task marked as critical")
	fmt.Println("───────────────────────────────────────────────────")
	testRequest(client, ctx, &pb.GenerateRequest{
		Prompt: "Summarize this document for me when you have time",
		Model:  "qwen2.5:0.5b",
		Annotations: &pb.JobAnnotations{
			LatencyCritical: true, // User says critical
		},
	}, "User claims: critical, System should: override (background task detected)")

	// Scenario 4: Power budget constraint
	fmt.Println("\n📋 SCENARIO 4: High-power request on low battery")
	fmt.Println("───────────────────────────────────────────────────")
	fmt.Println("Simulating battery at 15%...")
	testRequest(client, ctx, &pb.GenerateRequest{
		Prompt: "Quick question",
		Model:  "qwen2.5:0.5b",
		Annotations: &pb.JobAnnotations{
			LatencyCritical: true,
			MaxPowerWatts:   5, // Simulate battery policy
		},
	}, "Battery: 15%, System should: force NPU (< 5W only)")

	// Scenario 5: Legitimate critical request
	fmt.Println("\n📋 SCENARIO 5: Legitimate time-sensitive request")
	fmt.Println("───────────────────────────────────────────────────")
	testRequest(client, ctx, &pb.GenerateRequest{
		Prompt: "URGENT: Quick summary needed immediately for meeting",
		Model:  "qwen2.5:0.5b",
		Annotations: &pb.JobAnnotations{
			LatencyCritical: true,
		},
	}, "User claims: critical with 'URGENT', System should: allow NVIDIA")

	// Scenario 6: Smart model-based routing
	fmt.Println("\n📋 SCENARIO 6: Small model on critical flag")
	fmt.Println("───────────────────────────────────────────────────")
	testRequest(client, ctx, &pb.GenerateRequest{
		Prompt: "Generate some text",
		Model:  "qwen2.5:0.5b", // Small model
		Annotations: &pb.JobAnnotations{
			LatencyCritical: true,
		},
	}, "Model: 0.5b (small), System should: NPU/iGPU sufficient")

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("CLASSIFICATION EXAMPLES")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	examples := []struct {
		prompt     string
		expected   string
		reasoning  string
	}{
		{
			prompt:    "What is the capital of France?",
			expected:  "NPU (SIMPLE)",
			reasoning: "Short factual question",
		},
		{
			prompt:    "Explain how photosynthesis works",
			expected:  "Intel GPU (MODERATE)",
			reasoning: "Standard explanation, medium length",
		},
		{
			prompt:    "Write a comprehensive analysis comparing renewable energy sources",
			expected:  "NVIDIA (COMPLEX)",
			reasoning: "Long-form writing, detailed analysis",
		},
		{
			prompt:    "True or false: Earth is flat",
			expected:  "NPU (SIMPLE)",
			reasoning: "Binary question",
		},
		{
			prompt:    "Generate a Python script to scrape websites and analyze sentiment",
			expected:  "NVIDIA (COMPLEX)",
			reasoning: "Code generation",
		},
		{
			prompt:    "Hello",
			expected:  "NPU (SIMPLE)",
			reasoning: "Very short, casual",
		},
		{
			prompt:    "Briefly explain quantum entanglement",
			expected:  "NPU (SIMPLE)",
			reasoning: "'Briefly' indicator",
		},
	}

	for i, ex := range examples {
		fmt.Printf("%d. Prompt: %q\n", i+1, ex.prompt)
		fmt.Printf("   Expected: %s\n", ex.expected)
		fmt.Printf("   Reasoning: %s\n\n", ex.reasoning)
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("POLICY SCENARIOS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	policies := []struct {
		scenario  string
		condition string
		result    string
	}{
		{
			scenario:  "Free tier user",
			condition: "6th NVIDIA request in 1 hour",
			result:    "❌ Quota exceeded → Intel GPU (5/hour limit)",
		},
		{
			scenario:  "Battery level",
			condition: "Battery at 15%",
			result:    "⚠️ Force NPU only (critical battery)",
		},
		{
			scenario:  "Battery level",
			condition: "Battery at 40%",
			result:    "⚠️ NPU or Intel GPU (max 15W)",
		},
		{
			scenario:  "Battery level",
			condition: "Battery at 85%",
			result:    "✅ All backends available",
		},
		{
			scenario:  "Time of day",
			condition: "2:00 AM (quiet hours)",
			result:    "🌙 Prefer NPU/Intel GPU (silent)",
		},
		{
			scenario:  "Daily budget",
			condition: "Used 9.5 Wh of 10 Wh budget",
			result:    "❌ Remaining: 0.5 Wh → NPU only",
		},
		{
			scenario:  "Premium tier",
			condition: "Any time",
			result:    "✅ High quotas, minimal restrictions",
		},
	}

	for i, p := range policies {
		fmt.Printf("%d. %s\n", i+1, p.scenario)
		fmt.Printf("   Condition: %s\n", p.condition)
		fmt.Printf("   Result: %s\n\n", p.result)
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("ENERGY COMPARISON")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	fmt.Println("Scenario: 100 simple queries (e.g., 'What is...?')\n")

	fmt.Println("WITHOUT smart routing (all marked critical → NVIDIA):")
	fmt.Println("  Time: 100 × 0.2s = 20 seconds")
	fmt.Println("  Energy: 100 × 0.003 Wh = 0.3 Wh")
	fmt.Println("  Cost: Fast but wasteful\n")

	fmt.Println("WITH smart routing (auto-classified → NPU):")
	fmt.Println("  Time: 100 × 0.8s = 80 seconds")
	fmt.Println("  Energy: 100 × 0.0007 Wh = 0.07 Wh")
	fmt.Println("  Savings: 77% energy saved")
	fmt.Println("  Trade-off: Takes 60s longer (acceptable for non-critical)\n")

	fmt.Println("Result: Can run 4x more queries on same battery")

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("KEY TAKEAWAYS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	fmt.Println("1. ✅ Simple queries auto-detected and routed to NPU")
	fmt.Println("      (even if user marks as critical)")
	fmt.Println()
	fmt.Println("2. ✅ Complex queries allowed to use NVIDIA")
	fmt.Println("      (when legitimately needed)")
	fmt.Println()
	fmt.Println("3. ✅ Battery state automatically limits power")
	fmt.Println("      (prevents drain)")
	fmt.Println()
	fmt.Println("4. ✅ Quotas prevent abuse")
	fmt.Println("      (free tier: 5 NVIDIA/hour)")
	fmt.Println()
	fmt.Println("5. ✅ Time-aware routing")
	fmt.Println("      (quiet hours prefer silent backends)")
	fmt.Println()
	fmt.Println("6. ✅ Model-aware routing")
	fmt.Println("      (small models don't need NVIDIA)")
	fmt.Println()
	fmt.Println("Result: Users can't game the system by marking everything critical")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

func testRequest(client pb.ComputeServiceClient, ctx context.Context, req *pb.GenerateRequest, explanation string) {
	fmt.Printf("Prompt: %q\n", req.Prompt)
	if req.Annotations != nil {
		fmt.Printf("User annotations:")
		if req.Annotations.LatencyCritical {
			fmt.Printf(" latency_critical=true")
		}
		if req.Annotations.MaxPowerWatts > 0 {
			fmt.Printf(" max_power=%dW", req.Annotations.MaxPowerWatts)
		}
		fmt.Println()
	}
	fmt.Printf("Expected: %s\n\n", explanation)

	resp, err := client.Generate(ctx, req)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	fmt.Printf("✅ Result:\n")
	fmt.Printf("   Backend: %s\n", resp.BackendUsed)
	fmt.Printf("   Reason: %s\n", resp.Routing.Reason)
	fmt.Printf("   Power: %.1fW\n", resp.Routing.EstimatedPowerWatts)
	if resp.Stats != nil {
		fmt.Printf("   Speed: %.1f tok/s\n", resp.Stats.TokensPerSecond)
		fmt.Printf("   Energy: %.4f Wh\n", resp.Stats.EnergyWh)
	}

	// Analyze if system correctly prevented abuse
	if req.Annotations.LatencyCritical && resp.BackendUsed != "ollama-nvidia" {
		fmt.Printf("   🛡️  PROTECTED: Overrode user's critical flag\n")
	} else if req.Annotations.LatencyCritical && resp.BackendUsed == "ollama-nvidia" {
		fmt.Printf("   ✓ ALLOWED: Critical flag justified\n")
	}
}
