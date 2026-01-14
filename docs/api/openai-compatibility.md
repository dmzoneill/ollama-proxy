# OpenAI API Compatibility

The Ollama Proxy provides an **OpenAI-compatible REST API** that allows drop-in replacement of OpenAI API clients with local inference backends.

---

## Overview

The proxy implements the OpenAI API specification for:
- Chat Completions (streaming and non-streaming)
- Completions
- Embeddings
- Model listing

This allows existing applications using the OpenAI SDK to work with local Ollama backends with minimal changes.

### Compatibility Level

| Endpoint | Status | Notes |
|----------|--------|-------|
| `/v1/chat/completions` | ✅ Full | Streaming and non-streaming |
| `/v1/completions` | ✅ Full | Legacy completion API |
| `/v1/embeddings` | ✅ Full | Text embeddings |
| `/v1/models` | ✅ Full | List available models |
| `/v1/images/generations` | ❌ Not supported | Image generation not available |
| `/v1/audio/*` | ❌ Not supported | Audio endpoints not available |

---

## Authentication

The proxy currently runs without authentication for local usage.

**For production deployments**, consider:
- Reverse proxy with authentication (nginx, Caddy)
- API key validation (future feature)
- Network isolation (localhost only)

```bash
# Current: No auth required
curl http://localhost:8080/v1/chat/completions -d '{...}'

# Future: API key support
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-..." \
  -d '{...}'
```

---

## Chat Completions API

### Endpoint

```
POST /v1/chat/completions
```

### Request Format

```json
{
  "model": "qwen2.5:0.5b",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Explain quantum computing"}
  ],
  "temperature": 0.7,
  "max_tokens": 150,
  "stream": false
}
```

### Supported Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `model` | string | Required | Model identifier (e.g., "qwen2.5:0.5b") |
| `messages` | array | Required | Array of message objects |
| `temperature` | float | 0.8 | Randomness (0.0-2.0) |
| `max_tokens` | integer | null | Maximum tokens to generate |
| `stream` | boolean | false | Enable streaming |
| `top_p` | float | 0.9 | Nucleus sampling |
| `frequency_penalty` | float | 0.0 | Reduce repetition (-2.0 to 2.0) |
| `presence_penalty` | float | 0.0 | Encourage new topics (-2.0 to 2.0) |
| `stop` | array | null | Stop sequences |
| `n` | integer | 1 | Number of completions (only n=1 supported) |

### Non-Streaming Response

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen2.5:0.5b",
    "messages": [
      {"role": "user", "content": "What is the capital of France?"}
    ]
  }'
```

**Response:**
```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "qwen2.5:0.5b",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "The capital of France is Paris."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 14,
    "completion_tokens": 8,
    "total_tokens": 22
  }
}
```

### Streaming Response

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen2.5:0.5b",
    "messages": [
      {"role": "user", "content": "Count to 5"}
    ],
    "stream": true
  }'
```

**Response (Server-Sent Events):**
```
data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1677652288,"model":"qwen2.5:0.5b","choices":[{"index":0,"delta":{"role":"assistant","content":"1"},"finish_reason":null}]}

data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1677652288,"model":"qwen2.5:0.5b","choices":[{"index":0,"delta":{"content":", "},"finish_reason":null}]}

data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1677652288,"model":"qwen2.5:0.5b","choices":[{"index":0,"delta":{"content":"2"},"finish_reason":null}]}

...

data: [DONE]
```

---

## Completions API (Legacy)

### Endpoint

```
POST /v1/completions
```

### Request Format

```json
{
  "model": "qwen2.5:0.5b",
  "prompt": "Once upon a time",
  "max_tokens": 50,
  "temperature": 0.7,
  "stream": false
}
```

### Example

```bash
curl http://localhost:8080/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen2.5:0.5b",
    "prompt": "The meaning of life is",
    "max_tokens": 20
  }'
```

**Response:**
```json
{
  "id": "cmpl-abc123",
  "object": "text_completion",
  "created": 1677652288,
  "model": "qwen2.5:0.5b",
  "choices": [
    {
      "text": " a question that has puzzled philosophers for centuries.",
      "index": 0,
      "finish_reason": "length"
    }
  ],
  "usage": {
    "prompt_tokens": 5,
    "completion_tokens": 20,
    "total_tokens": 25
  }
}
```

---

## Embeddings API

### Endpoint

```
POST /v1/embeddings
```

### Request Format

```json
{
  "model": "nomic-embed-text",
  "input": "The quick brown fox jumps over the lazy dog"
}
```

### Example

```bash
curl http://localhost:8080/v1/embeddings \
  -H "Content-Type: application/json" \
  -d '{
    "model": "nomic-embed-text",
    "input": "Hello, world!"
  }'
```

**Response:**
```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "index": 0,
      "embedding": [0.0123, -0.0234, 0.0345, ...]
    }
  ],
  "model": "nomic-embed-text",
  "usage": {
    "prompt_tokens": 3,
    "total_tokens": 3
  }
}
```

### Batch Embeddings

```json
{
  "model": "nomic-embed-text",
  "input": [
    "First sentence",
    "Second sentence",
    "Third sentence"
  ]
}
```

**Response:**
```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "index": 0,
      "embedding": [...]
    },
    {
      "object": "embedding",
      "index": 1,
      "embedding": [...]
    },
    {
      "object": "embedding",
      "index": 2,
      "embedding": [...]
    }
  ],
  "model": "nomic-embed-text",
  "usage": {
    "prompt_tokens": 9,
    "total_tokens": 9
  }
}
```

---

## Models API

### List Models

```
GET /v1/models
```

**Example:**
```bash
curl http://localhost:8080/v1/models
```

**Response:**
```json
{
  "object": "list",
  "data": [
    {
      "id": "qwen2.5:0.5b",
      "object": "model",
      "created": 1677652288,
      "owned_by": "ollama",
      "permission": [],
      "root": "qwen2.5:0.5b",
      "parent": null
    },
    {
      "id": "llama3.2:3b",
      "object": "model",
      "created": 1677652288,
      "owned_by": "ollama",
      "root": "llama3.2:3b",
      "parent": null
    }
  ]
}
```

### Get Model Details

```
GET /v1/models/{model_id}
```

**Example:**
```bash
curl http://localhost:8080/v1/models/qwen2.5:0.5b
```

**Response:**
```json
{
  "id": "qwen2.5:0.5b",
  "object": "model",
  "created": 1677652288,
  "owned_by": "ollama",
  "permission": [],
  "root": "qwen2.5:0.5b",
  "parent": null
}
```

---

## Custom Headers (Routing Control)

The proxy extends the OpenAI API with custom routing headers:

### Request Headers

| Header | Type | Description |
|--------|------|-------------|
| `X-Target-Backend` | string | Explicit backend selection |
| `X-Latency-Critical` | boolean | Route to fastest backend |
| `X-Power-Efficient` | boolean | Route to lowest power backend |
| `X-Max-Latency-Ms` | integer | Maximum acceptable latency (ms) |
| `X-Max-Power-Watts` | integer | Maximum power budget (watts) |
| `X-Priority` | string | Request priority (critical, high, normal, best-effort) |
| `X-Request-ID` | string | Request tracking ID |
| `X-Media-Type` | string | Workload hint (realtime, batch, interactive) |

### Response Headers

| Header | Description |
|--------|-------------|
| `X-Backend-Used` | Backend that processed the request |
| `X-Estimated-Latency-Ms` | Estimated latency in milliseconds |
| `X-Estimated-Power-W` | Estimated power consumption in watts |
| `X-Routing-Reason` | Reason for backend selection |
| `X-Alternatives` | Alternative backends that could have been used |
| `X-Queue-Depth` | Number of pending requests on selected backend |

### Example with Custom Headers

```bash
curl -i http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-Priority: critical" \
  -H "X-Max-Latency-Ms: 200" \
  -H "X-Request-ID: voice-stream-001" \
  -d '{
    "model": "qwen2.5:0.5b",
    "messages": [{"role": "user", "content": "Transcribe audio"}],
    "stream": true
  }'
```

**Response Headers:**
```
HTTP/1.1 200 OK
Content-Type: text/event-stream
X-Backend-Used: ollama-nvidia
X-Estimated-Latency-Ms: 150
X-Estimated-Power-W: 55.0
X-Routing-Reason: critical-priority-low-latency
X-Queue-Depth: 2
```

---

## SDK Integration

### Python (OpenAI SDK)

```python
from openai import OpenAI

# Point to proxy instead of OpenAI
client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="not-needed"  # Proxy doesn't require auth currently
)

# Use as normal
response = client.chat.completions.create(
    model="qwen2.5:0.5b",
    messages=[
        {"role": "user", "content": "Hello!"}
    ]
)

print(response.choices[0].message.content)
```

#### Streaming with Python

```python
from openai import OpenAI

client = OpenAI(base_url="http://localhost:8080/v1", api_key="not-needed")

stream = client.chat.completions.create(
    model="qwen2.5:0.5b",
    messages=[{"role": "user", "content": "Count to 10"}],
    stream=True
)

for chunk in stream:
    if chunk.choices[0].delta.content is not None:
        print(chunk.choices[0].delta.content, end="", flush=True)

print()
```

#### Custom Headers with Python

```python
import httpx
from openai import OpenAI

# Use custom httpx client to add headers
http_client = httpx.Client(
    headers={
        "X-Priority": "critical",
        "X-Max-Latency-Ms": "200"
    }
)

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="not-needed",
    http_client=http_client
)

response = client.chat.completions.create(
    model="qwen2.5:0.5b",
    messages=[{"role": "user", "content": "Hello!"}]
)
```

### JavaScript/TypeScript (OpenAI SDK)

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:8080/v1',
  apiKey: 'not-needed',
});

const completion = await client.chat.completions.create({
  model: 'qwen2.5:0.5b',
  messages: [{ role: 'user', content: 'Hello!' }],
});

console.log(completion.choices[0].message.content);
```

#### Streaming with JavaScript

```javascript
const stream = await client.chat.completions.create({
  model: 'qwen2.5:0.5b',
  messages: [{ role: 'user', content: 'Count to 10' }],
  stream: true,
});

for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || '');
}
```

#### Custom Headers with JavaScript

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:8080/v1',
  apiKey: 'not-needed',
  defaultHeaders: {
    'X-Priority': 'critical',
    'X-Max-Latency-Ms': '200',
  },
});

const completion = await client.chat.completions.create({
  model: 'qwen2.5:0.5b',
  messages: [{ role: 'user', content: 'Voice input' }],
});
```

### Go

```go
package main

import (
    "context"
    "fmt"
    openai "github.com/sashabaranov/go-openai"
)

func main() {
    config := openai.DefaultConfig("not-needed")
    config.BaseURL = "http://localhost:8080/v1"
    client := openai.NewClientWithConfig(config)

    resp, err := client.CreateChatCompletion(
        context.Background(),
        openai.ChatCompletionRequest{
            Model: "qwen2.5:0.5b",
            Messages: []openai.ChatCompletionMessage{
                {
                    Role:    openai.ChatMessageRoleUser,
                    Content: "Hello!",
                },
            },
        },
    )

    if err != nil {
        panic(err)
    }

    fmt.Println(resp.Choices[0].Message.Content)
}
```

### cURL Examples

#### Non-Streaming Chat

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen2.5:0.5b",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "What is the speed of light?"}
    ],
    "temperature": 0.7,
    "max_tokens": 50
  }'
```

#### Streaming Chat

```bash
curl -N http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen2.5:0.5b",
    "messages": [{"role": "user", "content": "Count to 5"}],
    "stream": true
  }'
```

#### With Power Constraints

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-Max-Power-Watts: 15" \
  -d '{
    "model": "qwen2.5:0.5b",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

---

## Error Handling

### Error Response Format

Errors follow OpenAI API format:

```json
{
  "error": {
    "message": "No healthy backends available matching criteria",
    "type": "invalid_request_error",
    "param": null,
    "code": "no_available_backends"
  }
}
```

### Common Error Codes

| HTTP Status | Error Type | Description |
|-------------|------------|-------------|
| 400 | `invalid_request_error` | Invalid request parameters |
| 404 | `invalid_request_error` | Model not found |
| 429 | `rate_limit_error` | Too many requests |
| 500 | `api_error` | Internal server error |
| 503 | `service_unavailable` | No healthy backends |

### Example Error Responses

**No Available Backends:**
```json
{
  "error": {
    "message": "No healthy backends available matching criteria: max_latency_ms=50",
    "type": "service_unavailable",
    "code": "no_available_backends"
  }
}
```

**Model Not Supported:**
```json
{
  "error": {
    "message": "Model llama3.2:70b not supported by any backend. Model too large.",
    "type": "invalid_request_error",
    "param": "model",
    "code": "model_not_found"
  }
}
```

**Invalid Parameters:**
```json
{
  "error": {
    "message": "Temperature must be between 0 and 2",
    "type": "invalid_request_error",
    "param": "temperature",
    "code": "invalid_value"
  }
}
```

---

## Differences from OpenAI API

### Not Supported

| Feature | Status | Workaround |
|---------|--------|------------|
| Function calling | ❌ Not supported | Use prompt engineering |
| Logprobs | ❌ Not supported | N/A |
| Multiple choices (n>1) | ❌ Not supported | Make multiple requests |
| Seed parameter | ❌ Not supported | N/A |
| Response format (JSON mode) | ❌ Not supported | Post-process response |
| Vision (image inputs) | ❌ Not supported | Use vision-specific models separately |

### Extended Features

| Feature | Description |
|---------|-------------|
| **Multi-backend routing** | Automatic selection based on constraints |
| **Power-aware routing** | Route based on power consumption |
| **Priority queuing** | Critical requests get priority |
| **Thermal management** | Automatic backend switching on overheating |
| **Backend targeting** | Explicit backend selection via header |
| **Queue depth visibility** | Response headers show queue state |

---

## Best Practices

### 1. Use Streaming for Interactive Applications

Streaming provides lower perceived latency:

```python
# ✅ Good: Streaming for chat
stream = client.chat.completions.create(
    model="qwen2.5:0.5b",
    messages=messages,
    stream=True  # Enable streaming
)

for chunk in stream:
    print(chunk.choices[0].delta.content, end="", flush=True)
```

### 2. Set Appropriate Timeouts

Local inference can be slow on some backends:

```python
from openai import OpenAI
import httpx

# Increase timeout for slow backends (NPU)
http_client = httpx.Client(timeout=60.0)

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="not-needed",
    http_client=http_client
)
```

### 3. Handle Backend Unavailability

Check for 503 errors when all backends are busy:

```python
from openai import OpenAI, APIError

try:
    response = client.chat.completions.create(...)
except APIError as e:
    if e.status_code == 503:
        print("All backends busy, retry later")
        # Implement retry logic
    else:
        raise
```

### 4. Use Priority Headers for Critical Requests

Mark voice/realtime requests as critical:

```python
http_client = httpx.Client(
    headers={"X-Priority": "critical"}
)

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="not-needed",
    http_client=http_client
)
```

### 5. Monitor Backend Selection

Log which backend was used:

```python
import httpx
from openai import OpenAI

class LoggingTransport(httpx.HTTPTransport):
    def handle_request(self, request):
        response = super().handle_request(request)
        backend = response.headers.get("X-Backend-Used")
        if backend:
            print(f"Request routed to: {backend}")
        return response

http_client = httpx.Client(transport=LoggingTransport())
client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="not-needed",
    http_client=http_client
)
```

---

## Migration Guide

### From OpenAI API to Proxy

**1. Change Base URL:**
```python
# Before (OpenAI)
client = OpenAI(api_key=os.getenv("OPENAI_API_KEY"))

# After (Proxy)
client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="not-needed"
)
```

**2. Update Model Names:**
```python
# Before (OpenAI)
model = "gpt-4"

# After (Proxy)
model = "qwen2.5:0.5b"  # Use local model name
```

**3. Handle Extended Features:**
```python
# Optionally use routing headers
http_client = httpx.Client(
    headers={
        "X-Max-Power-Watts": "15",  # Power constraint
        "X-Priority": "high"         # Priority level
    }
)

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="not-needed",
    http_client=http_client
)
```

**4. Adjust Expectations:**
- Latency: Local inference typically slower than OpenAI (150-800ms vs 50-200ms)
- Quality: Depends on model size (0.5B vs 175B)
- Features: No function calling, vision, or JSON mode (yet)

---

## Related Documentation

- [gRPC API](grpc-api.md) - High-performance gRPC interface
- [WebSocket API](websocket-api.md) - Ultra-low latency streaming
- [Multi-Backend Routing](../features/routing.md) - Routing algorithm details
- [Priority Queuing](../features/priority-queuing.md) - Request prioritization
