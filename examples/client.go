//go:build ignore

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	pb "github.com/daoneill/ollama-proxy/api/gen/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to proxy
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewComputeServiceClient(conn)
	ctx := context.Background()

	// Test 1: Health Check
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("TEST 1: Health Check")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	healthResp, err := client.HealthCheck(ctx, &pb.HealthCheckRequest{})
	if err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		fmt.Printf("Overall Status: %s\n", healthResp.Status)
		fmt.Println("Backend Health:")
		for backend, status := range healthResp.BackendHealth {
			symbol := "✅"
			if status != "healthy" {
				symbol = "❌"
			}
			fmt.Printf("  %s %s: %s\n", symbol, backend, status)
		}
	}

	// Test 2: List Backends
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("TEST 2: List Backends")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	backendsResp, err := client.ListBackends(ctx, &pb.ListBackendsRequest{})
	if err != nil {
		log.Printf("List backends failed: %v", err)
	} else {
		for _, b := range backendsResp.Backends {
			fmt.Printf("\n%s (%s)\n", b.Name, b.Id)
			fmt.Printf("  Hardware: %s\n", b.Hardware)
			fmt.Printf("  Status: %s\n", b.Status.State)
			fmt.Printf("  Power: %.1fW\n", b.Metrics.PowerWatts)
			fmt.Printf("  Avg Latency: %dms\n", b.Metrics.AvgLatencyMs)
			fmt.Printf("  Models: %v\n", b.Capabilities.Models)
		}
	}

	// Test 3: Auto-routing (balanced)
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("TEST 3: Auto-routing (Balanced)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	runGeneration(client, ctx, &pb.GenerateRequest{
		Prompt: "What is 2+2? Answer briefly.",
		Model:  "qwen2.5:0.5b",
	}, "Auto-routing")

	// Test 4: Power-efficient (NPU)
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("TEST 4: Power-Efficient Routing (NPU)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	runGeneration(client, ctx, &pb.GenerateRequest{
		Prompt: "Hello, how are you?",
		Model:  "qwen2.5:0.5b",
		Annotations: &pb.JobAnnotations{
			PreferPowerEfficiency: true,
		},
	}, "Power-efficient")

	// Test 5: Latency-critical (NVIDIA)
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("TEST 5: Latency-Critical Routing (NVIDIA)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	runGeneration(client, ctx, &pb.GenerateRequest{
		Prompt: "Quick answer: name one planet",
		Model:  "qwen2.5:0.5b",
		Annotations: &pb.JobAnnotations{
			LatencyCritical: true,
		},
	}, "Latency-critical")

	// Test 6: Explicit target (Intel GPU)
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("TEST 6: Explicit Target (Intel GPU)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	runGeneration(client, ctx, &pb.GenerateRequest{
		Prompt: "What is the capital of France?",
		Model:  "qwen2.5:0.5b",
		Annotations: &pb.JobAnnotations{
			Target: "ollama-igpu",
		},
	}, "Explicit target")

	// Test 7: Power budget constraint
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("TEST 7: Max Power Budget (15W)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	runGeneration(client, ctx, &pb.GenerateRequest{
		Prompt: "Tell me about AI",
		Model:  "qwen2.5:0.5b",
		Annotations: &pb.JobAnnotations{
			MaxPowerWatts: 15, // Excludes NVIDIA (55W)
		},
	}, "Power budget")

	// Test 8: Streaming
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("TEST 8: Streaming Generation (NVIDIA)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	runStreaming(client, ctx, &pb.GenerateRequest{
		Prompt: "Count from 1 to 5",
		Model:  "qwen2.5:0.5b",
		Annotations: &pb.JobAnnotations{
			Target: "ollama-nvidia",
		},
	})

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ ALL TESTS COMPLETED")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func runGeneration(client pb.ComputeServiceClient, ctx context.Context, req *pb.GenerateRequest, label string) {
	fmt.Printf("Prompt: %q\n", req.Prompt)
	if req.Annotations != nil {
		fmt.Printf("Annotations: ")
		if req.Annotations.Target != "" {
			fmt.Printf("target=%s ", req.Annotations.Target)
		}
		if req.Annotations.LatencyCritical {
			fmt.Printf("latency_critical=true ")
		}
		if req.Annotations.PreferPowerEfficiency {
			fmt.Printf("prefer_power_efficiency=true ")
		}
		if req.Annotations.MaxPowerWatts > 0 {
			fmt.Printf("max_power_watts=%d ", req.Annotations.MaxPowerWatts)
		}
		fmt.Println()
	}

	start := time.Now()
	resp, err := client.Generate(ctx, req)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("❌ Generation failed: %v\n", err)
		return
	}

	fmt.Printf("\n✅ Success!\n")
	fmt.Printf("Backend: %s\n", resp.BackendUsed)
	fmt.Printf("Reason: %s\n", resp.Routing.Reason)
	fmt.Printf("Power: %.1fW\n", resp.Routing.EstimatedPowerWatts)
	fmt.Printf("Latency: %dms (estimated), %dms (actual)\n",
		resp.Routing.EstimatedLatencyMs, elapsed.Milliseconds())

	if resp.Stats != nil {
		fmt.Printf("Stats:\n")
		fmt.Printf("  Total time: %dms\n", resp.Stats.TotalTimeMs)
		fmt.Printf("  Tokens: %d\n", resp.Stats.TokensGenerated)
		fmt.Printf("  Speed: %.1f tok/s\n", resp.Stats.TokensPerSecond)
		fmt.Printf("  Energy: %.3f Wh\n", resp.Stats.EnergyWh)
	}

	fmt.Printf("\nResponse: %s\n", truncate(resp.Response, 150))

	if len(resp.Routing.Alternatives) > 0 {
		fmt.Printf("\nAlternatives available: %v\n", resp.Routing.Alternatives)
	}
}

func runStreaming(client pb.ComputeServiceClient, ctx context.Context, req *pb.GenerateRequest) {
	fmt.Printf("Prompt: %q\n", req.Prompt)

	stream, err := client.GenerateStream(ctx, req)
	if err != nil {
		log.Printf("❌ Streaming failed: %v\n", err)
		return
	}

	fmt.Printf("\nStreaming response: ")
	tokenCount := 0
	var backend string
	start := time.Now()

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("\n❌ Stream error: %v\n", err)
			break
		}

		if chunk.BackendUsed != "" {
			backend = chunk.BackendUsed
		}

		fmt.Print(chunk.Token)
		tokenCount++

		if chunk.Done {
			elapsed := time.Since(start)
			fmt.Printf("\n\n✅ Stream complete!\n")
			fmt.Printf("Backend: %s\n", backend)
			fmt.Printf("Tokens: %d\n", tokenCount)
			fmt.Printf("Time: %dms\n", elapsed.Milliseconds())
			if chunk.Stats != nil {
				fmt.Printf("Speed: %.1f tok/s\n", chunk.Stats.TokensPerSecond)
				fmt.Printf("Energy: %.3f Wh\n", chunk.Stats.EnergyWh)
			}
			break
		}
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
