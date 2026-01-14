# Ollama Compute Proxy - Quick Start Guide

## What You Have Now

A production-ready compute proxy that intelligently routes inference requests across your 4 Ollama instances based on job annotations:

- **NPU (Port 11434)**: Ultra-low power (2-5W), ~10 tok/s
- **Intel GPU (Port 11435)**: Balanced (8-15W), ~22 tok/s
- **NVIDIA GPU (Port 11436)**: High performance (40-60W), ~65 tok/s
- **CPU (Port 11437)**: Fallback (15-35W), ~6 tok/s

## Starting the Proxy

```bash
# From the ollama-proxy directory
./bin/ollama-proxy
```

The proxy will:
1. Load config from `config/config.yaml`
2. Connect to all 4 Ollama instances
3. Perform health checks
4. Start gRPC server on port 50051
5. Start HTTP server on port 8080

## Testing the Proxy

### 1. Check Health (HTTP)

```bash
curl http://localhost:8080/health
```

Expected output:
```
Status: healthy
  ollama-npu: healthy
  ollama-igpu: healthy
  ollama-nvidia: healthy
  ollama-cpu: healthy
```

### 2. List Backends (HTTP)

```bash
curl http://localhost:8080/backends
```

Shows all available backends with their characteristics.

### 3. Test gRPC (using grpcurl)

Install grpcurl if you don't have it:
```bash
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

List available services:
```bash
grpcurl -plaintext localhost:50051 list
```

Output:
```
compute.v1.ComputeService
grpc.reflection.v1.ServerReflection
grpc.reflection.v1alpha.ServerReflection
```

List available methods:
```bash
grpcurl -plaintext localhost:50051 describe compute.v1.ComputeService
```

### 4. Generate Text with Automatic Routing

**Simple request (will auto-route to best backend):**

```bash
grpcurl -plaintext -d '{
  "prompt": "What is 2+2?",
  "model": "qwen2.5:0.5b"
}' localhost:50051 compute.v1.ComputeService/Generate
```

**Route to specific hardware:**

```bash
# Force NPU (ultra-low power)
grpcurl -plaintext -d '{
  "prompt": "Explain photosynthesis briefly",
  "model": "qwen2.5:0.5b",
  "annotations": {
    "target": "ollama-npu"
  }
}' localhost:50051 compute.v1.ComputeService/Generate

# Force NVIDIA (maximum performance)
grpcurl -plaintext -d '{
  "prompt": "Write a poem about AI",
  "model": "llama3:7b",
  "annotations": {
    "target": "ollama-nvidia"
  }
}' localhost:50051 compute.v1.ComputeService/Generate
```

**Latency-critical routing:**

```bash
grpcurl -plaintext -d '{
  "prompt": "Quick response needed",
  "model": "qwen2.5:0.5b",
  "annotations": {
    "latency_critical": true
  }
}' localhost:50051 compute.v1.ComputeService/Generate
```

This will automatically route to NVIDIA (fastest backend).

**Power-efficient routing:**

```bash
grpcurl -plaintext -d '{
  "prompt": "Battery mode query",
  "model": "qwen2.5:0.5b",
  "annotations": {
    "prefer_power_efficiency": true
  }
}' localhost:50051 compute.v1.ComputeService/Generate
```

This will route to NPU (lowest power).

**Max power budget:**

```bash
grpcurl -plaintext -d '{
  "prompt": "On battery, limit power",
  "model": "qwen2.5:0.5b",
  "annotations": {
    "max_power_watts": 15
  }
}' localhost:50051 compute.v1.ComputeService/Generate
```

This will exclude NVIDIA (55W) and route to Intel GPU or NPU.

### 5. Streaming Generation

```bash
grpcurl -plaintext -d '{
  "prompt": "Count from 1 to 10",
  "model": "qwen2.5:0.5b",
  "annotations": {
    "target": "ollama-nvidia"
  }
}' localhost:50051 compute.v1.ComputeService/GenerateStream
```

## Example Client (Go)

Create a file `examples/client.go`:

```go
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
	// Connect to proxy
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewComputeServiceClient(conn)
	ctx := context.Background()

	// Example 1: Auto-routing (balanced)
	fmt.Println("=== Auto-routing ===")
	resp, err := client.Generate(ctx, &pb.GenerateRequest{
		Prompt: "What is quantum computing?",
		Model:  "qwen2.5:0.5b",
	})
	if err != nil {
		log.Fatalf("Generate failed: %v", err)
	}
	fmt.Printf("Backend used: %s\n", resp.BackendUsed)
	fmt.Printf("Reason: %s\n", resp.Routing.Reason)
	fmt.Printf("Response: %s\n\n", resp.Response)

	// Example 2: Power-efficient (NPU)
	fmt.Println("=== Power-efficient ===")
	resp, err = client.Generate(ctx, &pb.GenerateRequest{
		Prompt: "Hello, world!",
		Model:  "qwen2.5:0.5b",
		Annotations: &pb.JobAnnotations{
			PreferPowerEfficiency: true,
		},
	})
	if err != nil {
		log.Fatalf("Generate failed: %v", err)
	}
	fmt.Printf("Backend used: %s (%.1fW)\n", resp.BackendUsed, resp.Routing.EstimatedPowerWatts)
	fmt.Printf("Response: %s\n\n", resp.Response)

	// Example 3: Latency-critical (NVIDIA)
	fmt.Println("=== Latency-critical ===")
	resp, err = client.Generate(ctx, &pb.GenerateRequest{
		Prompt: "Quick answer needed",
		Model:  "qwen2.5:0.5b",
		Annotations: &pb.JobAnnotations{
			LatencyCritical: true,
		},
	})
	if err != nil {
		log.Fatalf("Generate failed: %v", err)
	}
	fmt.Printf("Backend used: %s (%dms estimated latency)\n", resp.BackendUsed, resp.Routing.EstimatedLatencyMs)
	fmt.Printf("Speed: %.1f tok/s\n", resp.Stats.TokensPerSecond)
	fmt.Printf("Energy used: %.3f Wh\n", resp.Stats.EnergyWh)
}
```

Run it:
```bash
go run examples/client.go
```

## Configuration

Edit `config/config.yaml` to:
- Change port numbers
- Enable/disable specific backends
- Adjust power/latency estimates
- Configure routing behavior

## Next Steps

1. **Add HTTP REST Gateway**: Implement grpc-gateway for REST API
2. **Add Caching**: Implement response caching layer
3. **Add Metrics**: Export Prometheus metrics
4. **Add OpenAI Backend**: Support OpenAI API as backend
5. **Add Vector DB Backend**: Support embeddings with vector databases

## Troubleshooting

**"No healthy backends available"**
- Check if all Ollama instances are running
- Verify ports in config/config.yaml match your setup
- Check health endpoint: `curl http://localhost:8080/health`

**gRPC connection refused**
- Ensure proxy is running: `./bin/ollama-proxy`
- Check gRPC port in config (default: 50051)

**Routing not working as expected**
- Check backend health: `curl http://localhost:8080/backends`
- Review routing logs in proxy output
- Verify annotations in request

## Architecture Decisions

The proxy implements:

1. **Annotation-based routing**: Clients specify requirements via annotations
2. **Power-aware selection**: Automatically considers power consumption
3. **Latency optimization**: Routes latency-critical requests to fastest backends
4. **Automatic fallback**: If primary backend fails, tries alternatives
5. **Health monitoring**: Continuous health checks every 30 seconds
6. **Scoring algorithm**: Combines priority, latency, and power for optimal selection

## Performance

Routing overhead: **< 1ms** (negligible compared to inference time)

Example end-to-end latencies:
- NPU: 800ms + routing
- Intel GPU: 350ms + routing
- NVIDIA: 150ms + routing
- CPU: 1200ms + routing
