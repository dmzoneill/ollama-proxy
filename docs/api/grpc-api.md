# gRPC API

The Ollama Proxy provides a **high-performance gRPC API** for low-latency inference requests with streaming support.

---

## Overview

The gRPC API offers:
- **Lower latency** than HTTP REST (binary protocol)
- **Efficient streaming** with bidirectional communication
- **Type safety** via Protocol Buffers
- **Connection pooling** and multiplexing (HTTP/2)
- **Service discovery** via gRPC reflection

### When to Use gRPC

| Use Case | Recommended API |
|----------|-----------------|
| Microservice communication | ✅ gRPC |
| High-throughput batch processing | ✅ gRPC |
| Bidirectional streaming | ✅ gRPC |
| Browser-based applications | ❌ OpenAI REST API |
| Quick prototyping | ❌ OpenAI REST API |
| Legacy system integration | ❌ OpenAI REST API |

---

## Connection

### Default Endpoint

```
localhost:50051
```

### Connection Examples

**Python (grpcio):**
```python
import grpc
from ollama_proxy_pb2 import GenerateRequest
from ollama_proxy_pb2_grpc import OllamaProxyStub

# Create channel
channel = grpc.insecure_channel('localhost:50051')
stub = OllamaProxyStub(channel)

# Make request
request = GenerateRequest(
    prompt="Explain quantum computing",
    model="qwen2.5:0.5b"
)
response = stub.Generate(request)
print(response.response)
```

**Go:**
```go
import (
    "context"
    "google.golang.org/grpc"
    pb "github.com/daoneill/ollama-proxy/pkg/proto"
)

conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewOllamaProxyClient(conn)
response, err := client.Generate(context.Background(), &pb.GenerateRequest{
    Prompt: "Explain quantum computing",
    Model:  "qwen2.5:0.5b",
})
```

**grpcurl (CLI):**
```bash
grpcurl -plaintext localhost:50051 list
grpcurl -plaintext localhost:50051 list ollama_proxy.OllamaProxy
```

---

## Service Definition

### Protocol Buffer Schema

```protobuf
syntax = "proto3";

package ollama_proxy;

service OllamaProxy {
  // Generate completion
  rpc Generate(GenerateRequest) returns (GenerateResponse);

  // Generate with streaming
  rpc GenerateStream(GenerateRequest) returns (stream GenerateStreamResponse);

  // Chat completion
  rpc ChatCompletion(ChatCompletionRequest) returns (ChatCompletionResponse);

  // Chat completion with streaming
  rpc ChatCompletionStream(ChatCompletionRequest) returns (stream ChatCompletionStreamResponse);

  // Get embeddings
  rpc Embeddings(EmbeddingsRequest) returns (EmbeddingsResponse);

  // List models
  rpc ListModels(ListModelsRequest) returns (ListModelsResponse);

  // Health check
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}

message GenerateRequest {
  string prompt = 1;
  string model = 2;
  GenerateOptions options = 3;
  Annotations annotations = 4;
}

message GenerateResponse {
  string response = 1;
  int64 total_duration_ns = 2;
  int64 load_duration_ns = 3;
  int32 prompt_eval_count = 4;
  int64 prompt_eval_duration_ns = 5;
  int32 eval_count = 6;
  int64 eval_duration_ns = 7;
  string backend_id = 8;
}

message GenerateStreamResponse {
  string response = 1;
  bool done = 2;
  int32 eval_count = 3;
  string backend_id = 4;
}

message ChatCompletionRequest {
  string model = 1;
  repeated ChatMessage messages = 2;
  ChatOptions options = 3;
  Annotations annotations = 4;
}

message ChatMessage {
  string role = 1;
  string content = 2;
}

message ChatCompletionResponse {
  string id = 1;
  string model = 2;
  ChatMessage message = 3;
  int64 created = 4;
  Usage usage = 5;
  string backend_id = 6;
}

message ChatCompletionStreamResponse {
  string id = 1;
  string model = 2;
  string delta_content = 3;
  bool done = 4;
  string backend_id = 5;
}

message Annotations {
  string target = 1;
  bool latency_critical = 2;
  bool prefer_power_efficiency = 3;
  int32 max_latency_ms = 4;
  int32 max_power_watts = 5;
  string priority = 6;
  string request_id = 7;
  map<string, string> custom = 8;
}

message GenerateOptions {
  float temperature = 1;
  int32 top_k = 2;
  float top_p = 3;
  int32 num_predict = 4;
  repeated string stop = 5;
}

message EmbeddingsRequest {
  string model = 1;
  string input = 2;
  Annotations annotations = 3;
}

message EmbeddingsResponse {
  repeated float embedding = 1;
  string backend_id = 2;
}

message Usage {
  int32 prompt_tokens = 1;
  int32 completion_tokens = 2;
  int32 total_tokens = 3;
}

message HealthCheckRequest {}

message HealthCheckResponse {
  bool healthy = 1;
  int32 backend_count = 2;
  int32 healthy_backend_count = 3;
}

message ListModelsRequest {
  Annotations annotations = 1;
}

message ListModelsResponse {
  repeated Model models = 1;
}

message Model {
  string name = 1;
  int64 size_bytes = 2;
  string digest = 3;
  repeated string supported_backends = 4;
}
```

---

## API Methods

### Generate (Non-Streaming)

Generate a completion without streaming.

**Request:**
```protobuf
GenerateRequest {
  prompt: "Explain quantum computing"
  model: "qwen2.5:0.5b"
  options: {
    temperature: 0.7
    num_predict: 100
  }
  annotations: {
    priority: "high"
  }
}
```

**Response:**
```protobuf
GenerateResponse {
  response: "Quantum computing is a type of computing..."
  total_duration_ns: 850000000
  eval_count: 45
  backend_id: "ollama-igpu"
}
```

**Example (Python):**
```python
import grpc
from ollama_proxy_pb2 import GenerateRequest, GenerateOptions, Annotations
from ollama_proxy_pb2_grpc import OllamaProxyStub

channel = grpc.insecure_channel('localhost:50051')
stub = OllamaProxyStub(channel)

request = GenerateRequest(
    prompt="Explain quantum computing in simple terms",
    model="qwen2.5:0.5b",
    options=GenerateOptions(
        temperature=0.7,
        num_predict=100
    ),
    annotations=Annotations(
        priority="high",
        max_latency_ms=500
    )
)

response = stub.Generate(request)
print(f"Response: {response.response}")
print(f"Backend: {response.backend_id}")
print(f"Duration: {response.total_duration_ns / 1e9:.2f}s")
```

**Example (grpcurl):**
```bash
grpcurl -plaintext -d '{
  "prompt": "What is AI?",
  "model": "qwen2.5:0.5b",
  "options": {"temperature": 0.7},
  "annotations": {"priority": "high"}
}' localhost:50051 ollama_proxy.OllamaProxy/Generate
```

---

### GenerateStream (Streaming)

Generate a completion with token-by-token streaming.

**Request:**
```protobuf
GenerateRequest {
  prompt: "Count to 10"
  model: "qwen2.5:0.5b"
  options: {
    temperature: 0.0
  }
}
```

**Response Stream:**
```protobuf
GenerateStreamResponse { response: "1", done: false }
GenerateStreamResponse { response: ", ", done: false }
GenerateStreamResponse { response: "2", done: false }
...
GenerateStreamResponse { response: "10", done: true, eval_count: 20 }
```

**Example (Python):**
```python
request = GenerateRequest(
    prompt="Count to 10",
    model="qwen2.5:0.5b"
)

# Streaming response
for response in stub.GenerateStream(request):
    print(response.response, end='', flush=True)
    if response.done:
        print(f"\nTokens: {response.eval_count}")
        print(f"Backend: {response.backend_id}")
```

**Example (Go):**
```go
stream, err := client.GenerateStream(ctx, &pb.GenerateRequest{
    Prompt: "Count to 10",
    Model:  "qwen2.5:0.5b",
})

for {
    chunk, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }

    fmt.Print(chunk.Response)
    if chunk.Done {
        fmt.Printf("\nBackend: %s\n", chunk.BackendId)
    }
}
```

---

### ChatCompletion (Non-Streaming)

Chat completion with message history.

**Request:**
```protobuf
ChatCompletionRequest {
  model: "qwen2.5:0.5b"
  messages: [
    { role: "system", content: "You are a helpful assistant" }
    { role: "user", content: "What is the capital of France?" }
  ]
  options: {
    temperature: 0.7
  }
}
```

**Response:**
```protobuf
ChatCompletionResponse {
  id: "chat-abc123"
  model: "qwen2.5:0.5b"
  message: {
    role: "assistant"
    content: "The capital of France is Paris."
  }
  created: 1677652288
  usage: {
    prompt_tokens: 20
    completion_tokens: 8
    total_tokens: 28
  }
  backend_id: "ollama-igpu"
}
```

**Example (Python):**
```python
from ollama_proxy_pb2 import ChatCompletionRequest, ChatMessage, ChatOptions

request = ChatCompletionRequest(
    model="qwen2.5:0.5b",
    messages=[
        ChatMessage(role="system", content="You are a helpful assistant"),
        ChatMessage(role="user", content="What is the capital of France?")
    ],
    options=ChatOptions(temperature=0.7)
)

response = stub.ChatCompletion(request)
print(response.message.content)
print(f"Tokens: {response.usage.total_tokens}")
```

---

### ChatCompletionStream (Streaming)

Streaming chat completion.

**Request:**
```protobuf
ChatCompletionRequest {
  model: "qwen2.5:0.5b"
  messages: [
    { role: "user", content: "Tell me a short story" }
  ]
}
```

**Response Stream:**
```protobuf
ChatCompletionStreamResponse {
  id: "chat-abc123"
  model: "qwen2.5:0.5b"
  delta_content: "Once"
  done: false
}
ChatCompletionStreamResponse {
  delta_content: " upon"
  done: false
}
...
ChatCompletionStreamResponse {
  delta_content: "end."
  done: true
}
```

**Example (Python):**
```python
request = ChatCompletionRequest(
    model="qwen2.5:0.5b",
    messages=[
        ChatMessage(role="user", content="Tell me a short story")
    ]
)

for chunk in stub.ChatCompletionStream(request):
    print(chunk.delta_content, end='', flush=True)
    if chunk.done:
        print(f"\nBackend: {chunk.backend_id}")
```

---

### Embeddings

Generate text embeddings.

**Request:**
```protobuf
EmbeddingsRequest {
  model: "nomic-embed-text"
  input: "Hello, world!"
}
```

**Response:**
```protobuf
EmbeddingsResponse {
  embedding: [0.0123, -0.0234, 0.0345, ...]
  backend_id: "ollama-igpu"
}
```

**Example (Python):**
```python
from ollama_proxy_pb2 import EmbeddingsRequest

request = EmbeddingsRequest(
    model="nomic-embed-text",
    input="Hello, world!"
)

response = stub.Embeddings(request)
print(f"Embedding dimension: {len(response.embedding)}")
print(f"First 5 values: {response.embedding[:5]}")
```

---

### ListModels

List available models across all backends.

**Request:**
```protobuf
ListModelsRequest {}
```

**Response:**
```protobuf
ListModelsResponse {
  models: [
    {
      name: "qwen2.5:0.5b"
      size_bytes: 500000000
      digest: "sha256:abc123..."
      supported_backends: ["ollama-npu", "ollama-igpu", "ollama-nvidia"]
    },
    {
      name: "llama3.2:3b"
      size_bytes: 3000000000
      digest: "sha256:def456..."
      supported_backends: ["ollama-igpu", "ollama-nvidia"]
    }
  ]
}
```

**Example (Python):**
```python
from ollama_proxy_pb2 import ListModelsRequest

request = ListModelsRequest()
response = stub.ListModels(request)

for model in response.models:
    print(f"Model: {model.name}")
    print(f"  Size: {model.size_bytes / 1e9:.2f} GB")
    print(f"  Backends: {', '.join(model.supported_backends)}")
```

**Example (grpcurl):**
```bash
grpcurl -plaintext localhost:50051 ollama_proxy.OllamaProxy/ListModels
```

---

### HealthCheck

Check proxy and backend health.

**Request:**
```protobuf
HealthCheckRequest {}
```

**Response:**
```protobuf
HealthCheckResponse {
  healthy: true
  backend_count: 4
  healthy_backend_count: 4
}
```

**Example (Python):**
```python
from ollama_proxy_pb2 import HealthCheckRequest

request = HealthCheckRequest()
response = stub.HealthCheck(request)

print(f"Healthy: {response.healthy}")
print(f"Backends: {response.healthy_backend_count}/{response.backend_count}")
```

---

## Annotations (Routing Control)

Use annotations to control routing behavior:

### Annotations Fields

| Field | Type | Description |
|-------|------|-------------|
| `target` | string | Explicit backend ID |
| `latency_critical` | bool | Route to fastest backend |
| `prefer_power_efficiency` | bool | Route to lowest power backend |
| `max_latency_ms` | int32 | Maximum acceptable latency |
| `max_power_watts` | int32 | Maximum power budget |
| `priority` | string | Request priority (critical, high, normal, best-effort) |
| `request_id` | string | Request tracking ID |
| `custom` | map<string,string> | Custom metadata |

### Example with Annotations

```python
from ollama_proxy_pb2 import GenerateRequest, Annotations

request = GenerateRequest(
    prompt="Transcribe audio",
    model="qwen2.5:0.5b",
    annotations=Annotations(
        latency_critical=True,
        priority="critical",
        max_latency_ms=100,
        request_id="voice-001"
    )
)

response = stub.Generate(request)
print(f"Routed to: {response.backend_id}")
```

---

## Error Handling

### gRPC Status Codes

| Code | Name | Description |
|------|------|-------------|
| 0 | OK | Success |
| 3 | INVALID_ARGUMENT | Invalid request parameters |
| 5 | NOT_FOUND | Model not found |
| 8 | RESOURCE_EXHAUSTED | All backends busy |
| 13 | INTERNAL | Internal server error |
| 14 | UNAVAILABLE | Service unavailable |

### Error Details

Errors include detailed messages:

```python
import grpc

try:
    response = stub.Generate(request)
except grpc.RpcError as e:
    print(f"gRPC Error: {e.code()}")
    print(f"Details: {e.details()}")

    if e.code() == grpc.StatusCode.RESOURCE_EXHAUSTED:
        print("All backends busy, retry later")
    elif e.code() == grpc.StatusCode.NOT_FOUND:
        print("Model not found")
```

**Example Error:**
```
Code: UNAVAILABLE
Details: No healthy backends available matching criteria: max_latency_ms=50
```

---

## Performance Optimization

### Connection Pooling

Reuse gRPC channels:

```python
import grpc
from ollama_proxy_pb2_grpc import OllamaProxyStub

class ProxyClient:
    def __init__(self, address='localhost:50051'):
        # Create persistent channel
        self.channel = grpc.insecure_channel(
            address,
            options=[
                ('grpc.keepalive_time_ms', 10000),
                ('grpc.keepalive_timeout_ms', 5000),
                ('grpc.keepalive_permit_without_calls', True),
                ('grpc.http2.max_pings_without_data', 0),
            ]
        )
        self.stub = OllamaProxyStub(self.channel)

    def generate(self, prompt, model):
        request = GenerateRequest(prompt=prompt, model=model)
        return self.stub.Generate(request)

    def close(self):
        self.channel.close()

# Usage
client = ProxyClient()
response1 = client.generate("Hello", "qwen2.5:0.5b")
response2 = client.generate("World", "qwen2.5:0.5b")
client.close()
```

### Streaming vs Non-Streaming

**Use streaming when:**
- Interactive applications (chat)
- Long-running generations
- Progressive display needed

**Use non-streaming when:**
- Batch processing
- Full response needed before processing
- Simpler error handling

### Compression

Enable compression for large payloads:

```python
import grpc

channel = grpc.insecure_channel(
    'localhost:50051',
    options=[('grpc.default_compression_algorithm', grpc.Compression.Gzip)]
)
```

### Timeouts

Set appropriate timeouts:

```python
# Per-request timeout
response = stub.Generate(
    request,
    timeout=30.0  # 30 seconds
)

# Stream with timeout
for chunk in stub.GenerateStream(request, timeout=60.0):
    print(chunk.response, end='')
```

---

## Advanced Usage

### Bidirectional Streaming (Future)

For future voice/interactive applications:

```protobuf
service OllamaProxy {
  rpc InteractiveChat(stream ChatMessage) returns (stream ChatCompletionStreamResponse);
}
```

**Usage:**
```python
def generate_messages():
    yield ChatMessage(role="user", content="Hello")
    yield ChatMessage(role="user", content="How are you?")

responses = stub.InteractiveChat(generate_messages())
for response in responses:
    print(response.delta_content, end='')
```

### Metadata (Headers)

Pass custom metadata:

```python
import grpc

metadata = [
    ('x-request-id', 'req-123'),
    ('x-priority', 'high'),
]

response = stub.Generate(request, metadata=metadata)
```

### Interceptors

Add logging or authentication:

```python
import grpc

class LoggingInterceptor(grpc.UnaryUnaryClientInterceptor):
    def intercept_unary_unary(self, continuation, client_call_details, request):
        print(f"Calling: {client_call_details.method}")
        response = continuation(client_call_details, request)
        print(f"Completed: {client_call_details.method}")
        return response

channel = grpc.insecure_channel('localhost:50051')
channel = grpc.intercept_channel(channel, LoggingInterceptor())
stub = OllamaProxyStub(channel)
```

---

## Code Generation

### Generate Client Code

**Python:**
```bash
# Install tools
pip install grpcio-tools

# Generate from .proto file
python -m grpc_tools.protoc \
  -I. \
  --python_out=. \
  --grpc_python_out=. \
  ollama_proxy.proto
```

**Go:**
```bash
# Install protoc-gen-go
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate
protoc --go_out=. --go-grpc_out=. ollama_proxy.proto
```

**JavaScript/TypeScript:**
```bash
# Install tools
npm install -g grpc-tools

# Generate
grpc_tools_node_protoc \
  --js_out=import_style=commonjs:. \
  --grpc_out=grpc_js:. \
  ollama_proxy.proto
```

---

## Testing

### grpcurl (CLI Testing)

**List services:**
```bash
grpcurl -plaintext localhost:50051 list
```

**Describe service:**
```bash
grpcurl -plaintext localhost:50051 describe ollama_proxy.OllamaProxy
```

**Call method:**
```bash
grpcurl -plaintext -d '{
  "prompt": "Hello",
  "model": "qwen2.5:0.5b"
}' localhost:50051 ollama_proxy.OllamaProxy/Generate
```

**Streaming:**
```bash
grpcurl -plaintext -d '{
  "prompt": "Count to 5",
  "model": "qwen2.5:0.5b"
}' localhost:50051 ollama_proxy.OllamaProxy/GenerateStream
```

### Python Unit Tests

```python
import grpc
import unittest
from ollama_proxy_pb2 import GenerateRequest
from ollama_proxy_pb2_grpc import OllamaProxyStub

class TestGRPCAPI(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.channel = grpc.insecure_channel('localhost:50051')
        cls.stub = OllamaProxyStub(cls.channel)

    def test_generate(self):
        request = GenerateRequest(
            prompt="Hello",
            model="qwen2.5:0.5b"
        )
        response = self.stub.Generate(request)
        self.assertIsNotNone(response.response)
        self.assertGreater(len(response.response), 0)

    def test_streaming(self):
        request = GenerateRequest(
            prompt="Count to 3",
            model="qwen2.5:0.5b"
        )
        chunks = list(self.stub.GenerateStream(request))
        self.assertGreater(len(chunks), 0)
        self.assertTrue(chunks[-1].done)

    @classmethod
    def tearDownClass(cls):
        cls.channel.close()
```

---

## Best Practices

### 1. Reuse Channels

Don't create a new channel per request:

```python
# ❌ Bad: New channel each time
def generate(prompt):
    channel = grpc.insecure_channel('localhost:50051')
    stub = OllamaProxyStub(channel)
    response = stub.Generate(GenerateRequest(prompt=prompt, model="qwen2.5:0.5b"))
    channel.close()
    return response

# ✅ Good: Reuse channel
class Client:
    def __init__(self):
        self.channel = grpc.insecure_channel('localhost:50051')
        self.stub = OllamaProxyStub(self.channel)

    def generate(self, prompt):
        return self.stub.Generate(GenerateRequest(prompt=prompt, model="qwen2.5:0.5b"))
```

### 2. Handle Errors Gracefully

```python
import grpc
import time

def generate_with_retry(stub, request, max_retries=3):
    for attempt in range(max_retries):
        try:
            return stub.Generate(request)
        except grpc.RpcError as e:
            if e.code() == grpc.StatusCode.UNAVAILABLE:
                if attempt < max_retries - 1:
                    time.sleep(2 ** attempt)  # Exponential backoff
                    continue
            raise
    raise Exception("Max retries exceeded")
```

### 3. Use Streaming for Long Responses

```python
# ✅ Good: Stream for long responses
for chunk in stub.GenerateStream(request):
    process_chunk(chunk.response)

# ❌ Bad: Non-streaming for long responses (high latency)
response = stub.Generate(request)
process_response(response.response)
```

### 4. Set Appropriate Timeouts

```python
# Short timeout for health checks
health = stub.HealthCheck(HealthCheckRequest(), timeout=2.0)

# Longer timeout for generation
response = stub.Generate(request, timeout=60.0)

# Very long for batch processing
for chunk in stub.GenerateStream(request, timeout=300.0):
    pass
```

---

## Related Documentation

- [OpenAI API Compatibility](openai-compatibility.md) - REST API reference
- [WebSocket API](websocket-api.md) - Ultra-low latency streaming
- [Multi-Backend Routing](../features/routing.md) - Routing algorithm
- [Streaming Optimizations](../features/streaming-optimizations.md) - Performance details
