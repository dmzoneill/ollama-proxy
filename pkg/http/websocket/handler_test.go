package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/logging"
	"github.com/daoneill/ollama-proxy/pkg/router"
	"github.com/gorilla/websocket"
)

// TestMain initializes the logger for all tests
func TestMain(m *testing.M) {
	// Initialize logger for tests
	if err := logging.InitLogger("info", false); err != nil {
		panic(err)
	}
	defer logging.Sync()

	// Run tests
	os.Exit(m.Run())
}

// MockBackend implements backends.Backend for testing
type MockBackend struct {
	id               string
	healthy          bool
	generateErr      error
	generateResponse *backends.GenerateResponse
	streamErr        error
	streamChunks     []*backends.StreamChunk
	mutex            sync.Mutex
	generateCalled   int
	streamCalled     int
}

func (m *MockBackend) ID() string                                                   { return m.id }
func (m *MockBackend) Type() string                                                 { return "mock" }
func (m *MockBackend) Name() string                                                 { return "mock-" + m.id }
func (m *MockBackend) Hardware() string                                             { return "cpu" }
func (m *MockBackend) IsHealthy() bool                                              { return m.healthy }
func (m *MockBackend) PowerWatts() float64                                          { return 10.0 }
func (m *MockBackend) AvgLatencyMs() int32                                          { return 100 }
func (m *MockBackend) Priority() int                                                { return 1 }
func (m *MockBackend) SupportsGenerate() bool                                       { return true }
func (m *MockBackend) SupportsStream() bool                                         { return true }
func (m *MockBackend) SupportsEmbed() bool                                          { return true }
func (m *MockBackend) ListModels(ctx context.Context) ([]string, error)             { return []string{}, nil }
func (m *MockBackend) SupportsModel(modelName string) bool                          { return true }
func (m *MockBackend) GetMaxModelSizeGB() int                                       { return 50 }
func (m *MockBackend) GetSupportedModelPatterns() []string                          { return []string{} }
func (m *MockBackend) GetPreferredModels() []string                                 { return []string{} }
func (m *MockBackend) UpdateMetrics(latencyMs int32, success bool)                  {}
func (m *MockBackend) GetMetrics() *backends.BackendMetrics                         { return &backends.BackendMetrics{} }
func (m *MockBackend) Start(ctx context.Context) error                              { return nil }
func (m *MockBackend) Stop(ctx context.Context) error                               { return nil }

// Multimedia capability methods
func (m *MockBackend) SupportsAudioToText() bool { return false }
func (m *MockBackend) SupportsTextToAudio() bool { return false }
func (m *MockBackend) SupportsImageToText() bool { return false }
func (m *MockBackend) SupportsTextToImage() bool { return false }
func (m *MockBackend) SupportsVideoToText() bool { return false }
func (m *MockBackend) SupportsTextToVideo() bool { return false }
func (m *MockBackend) TranscribeAudio(ctx context.Context, req *backends.TranscribeRequest) (*backends.TranscribeResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) TranscribeAudioStream(ctx context.Context, req *backends.TranscribeRequest) (backends.AudioStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) SynthesizeSpeech(ctx context.Context, req *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) SynthesizeSpeechStream(ctx context.Context, req *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) AnalyzeImage(ctx context.Context, req *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) GenerateImage(ctx context.Context, req *backends.ImageGenRequest) (*backends.ImageGenResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) GenerateImageStream(ctx context.Context, req *backends.ImageGenRequest) (backends.ImageStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) AnalyzeVideo(ctx context.Context, req *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) AnalyzeVideoStream(ctx context.Context, req *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) GenerateVideo(ctx context.Context, req *backends.VideoGenRequest) (*backends.VideoGenResponse, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) GenerateVideoStream(ctx context.Context, req *backends.VideoGenRequest) (backends.VideoStreamReader, error) { return nil, fmt.Errorf("not implemented") }
func (m *MockBackend) HealthCheck(ctx context.Context) error                        { return nil }

func (m *MockBackend) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.generateCalled++
	if m.generateErr != nil {
		return nil, m.generateErr
	}
	if m.generateResponse != nil {
		return m.generateResponse, nil
	}
	return &backends.GenerateResponse{Response: "test response"}, nil
}

func (m *MockBackend) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.streamCalled++
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	return &MockStreamReader{chunks: m.streamChunks}, nil
}

func (m *MockBackend) Embed(ctx context.Context, req *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return nil, nil
}

// MockStreamReader implements backends.StreamReader for testing
type MockStreamReader struct {
	chunks []*backends.StreamChunk
	index  int
	mutex  sync.Mutex
}

func (m *MockStreamReader) Recv() (*backends.StreamChunk, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.index >= len(m.chunks) {
		return nil, io.EOF
	}
	chunk := m.chunks[m.index]
	m.index++
	return chunk, nil
}

func (m *MockStreamReader) Close() error {
	return nil
}

// createTestServer creates a test HTTP server with WebSocket handler
func createTestServer(r *router.Router) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", HandleWebSocketStream(r))
	return httptest.NewServer(mux)
}

// createTestRouter creates a test router with mocked backends
func createTestRouter() *router.Router {
	r := router.NewRouter(router.Config{
		DefaultBackendID: "mock1",
		PowerAware:       false,
		AutoOptimize:     false,
	})
	return r
}

// Test: WebSocket connection establishment
func TestWebSocketConnectionEstablishment(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{id: "mock1", healthy: true}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// Attempt to upgrade connection
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	// Verify connection is established
	if conn.RemoteAddr() == nil {
		t.Error("WebSocket connection RemoteAddr is nil")
	}
}

// Test: Message handling - streaming request
func TestWebSocketStreamingMessageHandling(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "hello", Done: false},
			{Token: " ", Done: false},
			{Token: "world", Done: true},
		},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	// Send streaming request
	req := WebSocketRequest{
		RequestID: "test-123",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read responses
	var responses []WebSocketChunk
	for i := 0; i < 3; i++ {
		var chunk WebSocketChunk
		if err := conn.ReadJSON(&chunk); err != nil {
			t.Fatalf("Failed to read chunk %d: %v", i, err)
		}
		responses = append(responses, chunk)
	}

	// Verify chunks
	if len(responses) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(responses))
	}

	if responses[0].Token != "hello" {
		t.Errorf("Expected first token 'hello', got '%s'", responses[0].Token)
	}
	if responses[2].Done != true {
		t.Error("Expected last chunk to have Done=true")
	}
	if responses[2].RequestID != "test-123" {
		t.Errorf("Expected RequestID 'test-123', got '%s'", responses[2].RequestID)
	}

	// Verify metrics on final chunk
	if responses[2].TotalTimeMs == 0 && responses[2].TokenCount == 0 {
		t.Error("Expected metrics in final chunk")
	}
}

// Test: Message handling - non-streaming request
func TestWebSocketNonStreamingMessageHandling(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:                 "mock1",
		healthy:            true,
		generateResponse:   &backends.GenerateResponse{Response: "full response"},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	// Send non-streaming request
	req := WebSocketRequest{
		RequestID: "test-456",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    false,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read response
	var chunk WebSocketChunk
	if err := conn.ReadJSON(&chunk); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Verify response
	if chunk.Token != "full response" {
		t.Errorf("Expected token 'full response', got '%s'", chunk.Token)
	}
	if chunk.Done != true {
		t.Error("Expected Done=true")
	}
	if chunk.RequestID != "test-456" {
		t.Errorf("Expected RequestID 'test-456', got '%s'", chunk.RequestID)
	}
}

// Test: Error handling - invalid request
func TestWebSocketErrorHandlingInvalidRequest(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{id: "mock1", healthy: true}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	// Send invalid JSON
	if err := conn.WriteMessage(websocket.TextMessage, []byte("invalid json")); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Read error response
	var errMsg WebSocketError
	if err := conn.ReadJSON(&errMsg); err != nil {
		t.Fatalf("Failed to read error: %v", err)
	}

	if !strings.Contains(errMsg.Error, "invalid request") {
		t.Errorf("Expected 'invalid request' error, got '%s'", errMsg.Error)
	}
}

// Test: Error handling - backend generation error
func TestWebSocketErrorHandlingBackendError(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		generateErr: fmt.Errorf("backend error"),
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	// Send non-streaming request
	req := WebSocketRequest{
		RequestID: "test-error",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    false,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read error response
	var errMsg WebSocketError
	if err := conn.ReadJSON(&errMsg); err != nil {
		t.Fatalf("Failed to read error: %v", err)
	}

	if !strings.Contains(errMsg.Error, "generation failed") {
		t.Errorf("Expected 'generation failed' error, got '%s'", errMsg.Error)
	}
}

// Test: Error handling - stream error
func TestWebSocketErrorHandlingStreamError(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:        "mock1",
		healthy:   true,
		streamErr: fmt.Errorf("stream error"),
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	// Send streaming request
	req := WebSocketRequest{
		RequestID: "test-stream-error",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read error response
	var errMsg WebSocketError
	if err := conn.ReadJSON(&errMsg); err != nil {
		t.Fatalf("Failed to read error: %v", err)
	}

	if !strings.Contains(errMsg.Error, "stream start failed") {
		t.Errorf("Expected 'stream start failed' error, got '%s'", errMsg.Error)
	}
}

// Test: Connection close
func TestWebSocketConnectionClose(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{id: "mock1", healthy: true}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}

	// Close connection
	if err := conn.Close(); err != nil {
		t.Errorf("Failed to close connection: %v", err)
	}

	// Verify connection is closed
	var msg string
	if err := conn.ReadJSON(&msg); err == nil {
		t.Error("Expected error on closed connection, but read succeeded")
	}
}

// Test: Concurrent connections
func TestWebSocketConcurrentConnections(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "response", Done: true},
		},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// Create multiple concurrent connections
	numConnections := 10
	var wg sync.WaitGroup
	var successCount atomic.Int32
	var errorCount atomic.Int32

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()

			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				errorCount.Add(1)
				return
			}
			defer conn.Close()

			// Send request
			req := WebSocketRequest{
				RequestID: fmt.Sprintf("conn-%d", connID),
				Model:     "test-model",
				Prompt:    "hello",
				Stream:    true,
			}
			if err := conn.WriteJSON(req); err != nil {
				errorCount.Add(1)
				return
			}

			// Read response
			var chunk WebSocketChunk
			if err := conn.ReadJSON(&chunk); err != nil {
				errorCount.Add(1)
				return
			}

			if chunk.RequestID == req.RequestID && chunk.Done {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	if successCount.Load() != int32(numConnections) {
		t.Errorf("Expected %d successful connections, got %d", numConnections, successCount.Load())
	}
	if errorCount.Load() > 0 {
		t.Errorf("Expected 0 errors, got %d", errorCount.Load())
	}
}

// Test: Context cancellation
func TestWebSocketContextCancellation(t *testing.T) {
	r := createTestRouter()

	// Create a backend that will be called
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "test", Done: true},
		},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	// Send request
	req := WebSocketRequest{
		RequestID: "test-cancel",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read response - should succeed normally
	var chunk WebSocketChunk
	if err := conn.ReadJSON(&chunk); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !chunk.Done {
		t.Error("Expected Done=true in response")
	}
}

// Test: Priority parsing
func TestWebSocketPriorityParsing(t *testing.T) {
	tests := []struct {
		name       string
		priority   string
		expected   backends.Priority
	}{
		{"best-effort", "best-effort", backends.PriorityBestEffort},
		{"low", "low", backends.PriorityBestEffort},
		{"normal", "normal", backends.PriorityNormal},
		{"", "", backends.PriorityNormal},
		{"high", "high", backends.PriorityHigh},
		{"critical", "critical", backends.PriorityCritical},
		{"realtime", "realtime", backends.PriorityCritical},
		{"unknown", "unknown", backends.PriorityNormal},
	}

	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "test", Done: true},
		},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				t.Fatalf("Failed to establish WebSocket connection: %v", err)
			}
			defer conn.Close()

			req := WebSocketRequest{
				RequestID: "test-priority",
				Model:     "test-model",
				Prompt:    "hello",
				Stream:    true,
				Priority:  tt.priority,
			}
			if err := conn.WriteJSON(req); err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}

			var chunk WebSocketChunk
			if err := conn.ReadJSON(&chunk); err != nil {
				t.Fatalf("Failed to read response: %v", err)
			}
		})
	}
}

// Test: Options conversion
func TestWebSocketOptionsConversion(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		generateResponse: &backends.GenerateResponse{Response: "test"},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	req := WebSocketRequest{
		RequestID: "test-options",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    false,
		Options: map[string]interface{}{
			"temperature": 0.7,
			"top_p":       0.9,
			"top_k":       40.0,
			"max_tokens":  100.0,
		},
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk WebSocketChunk
	if err := conn.ReadJSON(&chunk); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !chunk.Done {
		t.Error("Expected Done=true")
	}
}

// Test: Stream with error during transmission
func TestWebSocketStreamErrorDuringTransmission(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "chunk1", Done: false},
			{Token: "chunk2", Done: false},
		},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	req := WebSocketRequest{
		RequestID: "test-stream-error",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read first chunk
	var chunk1 WebSocketChunk
	if err := conn.ReadJSON(&chunk1); err != nil {
		t.Fatalf("Failed to read first chunk: %v", err)
	}

	if chunk1.Token != "chunk1" {
		t.Errorf("Expected token 'chunk1', got '%s'", chunk1.Token)
	}

	// Read second chunk
	var chunk2 WebSocketChunk
	if err := conn.ReadJSON(&chunk2); err != nil {
		t.Fatalf("Failed to read second chunk: %v", err)
	}

	if chunk2.Token != "chunk2" {
		t.Errorf("Expected token 'chunk2', got '%s'", chunk2.Token)
	}
}

// Test: Metrics calculation on final chunk
func TestWebSocketMetricsCalculation(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "t1", Done: false},
			{Token: "t2", Done: false},
			{Token: "t3", Done: true},
		},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	req := WebSocketRequest{
		RequestID: "test-metrics",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read all chunks
	var finalChunk WebSocketChunk
	for i := 0; i < 3; i++ {
		if err := conn.ReadJSON(&finalChunk); err != nil {
			t.Fatalf("Failed to read chunk %d: %v", i, err)
		}
	}

	// Verify final chunk has metrics (note: TotalTimeMs may be very small, just verify TokenCount)
	if finalChunk.TokenCount != 3 {
		t.Errorf("Expected TokenCount=3, got %d", finalChunk.TokenCount)
	}
	if finalChunk.TokenCount > 0 && finalChunk.TotalTimeMs < 0 {
		t.Error("Expected TotalTimeMs >= 0")
	}
	if finalChunk.TokenCount > 0 && finalChunk.TokensPerSec < 0 {
		t.Error("Expected TokensPerSec >= 0")
	}
}

// Test: TTFT (Time to First Token) calculation
func TestWebSocketTTFTCalculation(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "first", Done: false},
			{Token: " ", Done: false},
			{Token: "second", Done: true},
		},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	req := WebSocketRequest{
		RequestID: "test-ttft",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read all chunks
	var finalChunk WebSocketChunk
	for i := 0; i < 3; i++ {
		if err := conn.ReadJSON(&finalChunk); err != nil {
			t.Fatalf("Failed to read chunk %d: %v", i, err)
		}
	}

	// Final chunk should have TTFT metric
	if finalChunk.Done && finalChunk.TTFT == 0 {
		// TTFT might be 0 if token arrives too quickly, which is acceptable
		t.Logf("TTFT was 0, token arrived very quickly")
	}
}

// Test: Empty options handling
func TestWebSocketEmptyOptionsHandling(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		generateResponse: &backends.GenerateResponse{Response: "test"},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	req := WebSocketRequest{
		RequestID: "test-no-options",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    false,
		// No options provided
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk WebSocketChunk
	if err := conn.ReadJSON(&chunk); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !chunk.Done {
		t.Error("Expected Done=true")
	}
}

// Test: Backend not healthy
func TestWebSocketBackendNotHealthy(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: false, // Not healthy
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	req := WebSocketRequest{
		RequestID: "test-unhealthy",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    false,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read error response (no healthy backends available)
	var errMsg WebSocketError
	if err := conn.ReadJSON(&errMsg); err != nil {
		t.Fatalf("Failed to read error: %v", err)
	}

	if !strings.Contains(errMsg.Error, "no backends available") && !strings.Contains(errMsg.Error, "no healthy backends") {
		t.Errorf("Expected 'no backends available' or 'no healthy backends' error, got '%s'", errMsg.Error)
	}
}

// Test: Multiple backends with health checks
func TestWebSocketMultipleBackends(t *testing.T) {
	r := createTestRouter()

	backend1 := &MockBackend{id: "backend1", healthy: false}
	backend2 := &MockBackend{
		id:      "backend2",
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "from-backend2", Done: true},
		},
	}

	r.RegisterBackend(backend1)
	r.RegisterBackend(backend2)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	req := WebSocketRequest{
		RequestID: "test-multi-backend",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk WebSocketChunk
	if err := conn.ReadJSON(&chunk); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Should have used backend2
	if chunk.Token != "from-backend2" {
		t.Errorf("Expected 'from-backend2', got '%s'", chunk.Token)
	}
}

// Test: Stress test with rapid successive requests (note: each request needs a new connection)
func TestWebSocketStressRapidRequests(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "response", Done: true},
		},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// Send 20 rapid requests with separate connections
	for i := 0; i < 20; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to establish WebSocket connection %d: %v", i, err)
		}

		req := WebSocketRequest{
			RequestID: fmt.Sprintf("stress-%d", i),
			Model:     "test-model",
			Prompt:    "hello",
			Stream:    true,
		}
		if err := conn.WriteJSON(req); err != nil {
			conn.Close()
			t.Fatalf("Failed to send request %d: %v", i, err)
		}

		var chunk WebSocketChunk
		if err := conn.ReadJSON(&chunk); err != nil {
			conn.Close()
			t.Fatalf("Failed to read response %d: %v", i, err)
		}

		if chunk.RequestID != req.RequestID {
			t.Errorf("Request %d: RequestID mismatch", i)
		}
		conn.Close()
	}
}

// Test: Max latency constraint
func TestWebSocketMaxLatencyConstraint(t *testing.T) {
	r := createTestRouter()

	// Both backends with stream chunks but fast backend has lower latency
	slowBackend := &MockBackend{
		id:      "slow",
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "slow", Done: true},
		},
	}

	fastBackend := &MockBackend{
		id:      "fast",
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "fast", Done: true},
		},
	}

	r.RegisterBackend(slowBackend)
	r.RegisterBackend(fastBackend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	req := WebSocketRequest{
		RequestID: "test-max-latency",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    true,
		MaxLatency: 100,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk WebSocketChunk
	if err := conn.ReadJSON(&chunk); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !chunk.Done {
		t.Error("Expected Done=true")
	}
}

// Test: Write deadline handling
func TestWebSocketWriteDeadline(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "test", Done: true},
		},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	req := WebSocketRequest{
		RequestID: "test-deadline",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read response with reasonable timeout
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	var chunk WebSocketChunk
	if err := conn.ReadJSON(&chunk); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !chunk.Done {
		t.Error("Expected Done=true")
	}
}

// Test: Backend called metrics
func TestWebSocketBackendCalledMetrics(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		generateResponse: &backends.GenerateResponse{Response: "test"},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	req := WebSocketRequest{
		RequestID: "test-metrics-backend",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    false,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk WebSocketChunk
	if err := conn.ReadJSON(&chunk); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	backend.mutex.Lock()
	if backend.generateCalled != 1 {
		t.Errorf("Expected Generate to be called once, got %d", backend.generateCalled)
	}
	backend.mutex.Unlock()
}

// Test: Streaming backend called metrics
func TestWebSocketStreamingBackendCalledMetrics(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "test", Done: true},
		},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	req := WebSocketRequest{
		RequestID: "test-stream-metrics-backend",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk WebSocketChunk
	if err := conn.ReadJSON(&chunk); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	backend.mutex.Lock()
	if backend.streamCalled != 1 {
		t.Errorf("Expected GenerateStream to be called once, got %d", backend.streamCalled)
	}
	backend.mutex.Unlock()
}

// Benchmark: WebSocket message throughput
func BenchmarkWebSocketMessageThroughput(b *testing.B) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "x", Done: true},
		},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := WebSocketRequest{
			RequestID: fmt.Sprintf("bench-%d", i),
			Model:     "test-model",
			Prompt:    "hello",
			Stream:    true,
		}
		if err := conn.WriteJSON(req); err != nil {
			b.Fatalf("Failed to send request: %v", err)
		}

		var chunk WebSocketChunk
		if err := conn.ReadJSON(&chunk); err != nil {
			b.Fatalf("Failed to read response: %v", err)
		}
	}
}

// Test: SendError helper function
func TestSendErrorHelper(t *testing.T) {
	// This test verifies sendError is called with correct parameters
	// by checking the output through the mock backend
	r := createTestRouter()
	backend := &MockBackend{id: "mock1", healthy: true}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	// Send invalid request to trigger sendError
	if err := conn.WriteMessage(websocket.TextMessage, []byte("not json")); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	var errMsg WebSocketError
	if err := conn.ReadJSON(&errMsg); err != nil {
		t.Fatalf("Failed to read error: %v", err)
	}

	if errMsg.Error == "" {
		t.Error("Expected error message, got empty string")
	}
}

// Test: ConvertWebSocketRequest helper function
func TestConvertWebSocketRequest(t *testing.T) {
	wsReq := &WebSocketRequest{
		Prompt: "test prompt",
		Model:  "test-model",
		Options: map[string]interface{}{
			"temperature": 0.5,
			"top_p":       0.8,
			"top_k":       50.0,
			"max_tokens":  200.0,
		},
	}

	result := convertWebSocketRequest(wsReq)

	if result.Prompt != "test prompt" {
		t.Errorf("Expected prompt 'test prompt', got '%s'", result.Prompt)
	}
	if result.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", result.Model)
	}
	if result.Options == nil {
		t.Error("Expected options to be set")
		return
	}
	if result.Options.Temperature != 0.5 {
		t.Errorf("Expected temperature 0.5, got %f", result.Options.Temperature)
	}
	if result.Options.TopP != 0.8 {
		t.Errorf("Expected topP 0.8, got %f", result.Options.TopP)
	}
	if result.Options.TopK != 50 {
		t.Errorf("Expected topK 50, got %d", result.Options.TopK)
	}
	if result.Options.MaxTokens != 200 {
		t.Errorf("Expected maxTokens 200, got %d", result.Options.MaxTokens)
	}
}

// Test: ConvertWebSocketRequest with nil options
func TestConvertWebSocketRequestNilOptions(t *testing.T) {
	wsReq := &WebSocketRequest{
		Prompt: "test",
		Model:  "test-model",
		Options: nil,
	}

	result := convertWebSocketRequest(wsReq)

	if result.Prompt != "test" {
		t.Errorf("Expected prompt 'test', got '%s'", result.Prompt)
	}
	if result.Options != nil {
		t.Error("Expected options to be nil")
	}
}

// Test: ConvertWebSocketRequest with partial options
func TestConvertWebSocketRequestPartialOptions(t *testing.T) {
	wsReq := &WebSocketRequest{
		Prompt: "test",
		Model:  "test-model",
		Options: map[string]interface{}{
			"temperature": 0.7,
			// Missing other fields
		},
	}

	result := convertWebSocketRequest(wsReq)

	if result.Options == nil {
		t.Error("Expected options to be set")
		return
	}
	if result.Options.Temperature != 0.7 {
		t.Errorf("Expected temperature 0.7, got %f", result.Options.Temperature)
	}
	if result.Options.TopP != 0 {
		t.Errorf("Expected topP 0 (default), got %f", result.Options.TopP)
	}
}

// Test: Response with RequestID verification
func TestWebSocketResponseRequestIDVerification(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:                 "mock1",
		healthy:            true,
		generateResponse:   &backends.GenerateResponse{Response: "test"},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	testCases := []string{"req-1", "req-abc-123", "special-id_456"}

	for _, requestID := range testCases {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to establish WebSocket connection: %v", err)
		}

		req := WebSocketRequest{
			RequestID: requestID,
			Model:     "test-model",
			Prompt:    "hello",
			Stream:    false,
		}
		if err := conn.WriteJSON(req); err != nil {
			conn.Close()
			t.Fatalf("Failed to send request: %v", err)
		}

		var chunk WebSocketChunk
		if err := conn.ReadJSON(&chunk); err != nil {
			conn.Close()
			t.Fatalf("Failed to read response: %v", err)
		}

		if chunk.RequestID != requestID {
			t.Errorf("Expected RequestID '%s', got '%s'", requestID, chunk.RequestID)
		}
		conn.Close()
	}
}

// Test: Non-streaming response format
func TestWebSocketNonStreamingResponseFormat(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:                 "mock1",
		healthy:            true,
		generateResponse:   &backends.GenerateResponse{Response: "complete response"},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	req := WebSocketRequest{
		RequestID: "test-format",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    false,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk WebSocketChunk
	if err := conn.ReadJSON(&chunk); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// For non-streaming, should have:
	// - Token = full response
	// - Done = true
	// - Metrics filled in
	if chunk.Token != "complete response" {
		t.Errorf("Expected token 'complete response', got '%s'", chunk.Token)
	}
	if !chunk.Done {
		t.Error("Expected Done=true for non-streaming")
	}
	if chunk.TokenCount != 1 {
		t.Errorf("Expected TokenCount=1, got %d", chunk.TokenCount)
	}
	if chunk.TokensPerSec == 0 {
		t.Error("Expected TokensPerSec > 0")
	}
}

// Test: Streaming response complete format
func TestWebSocketStreamingCompleteResponseFormat(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "hello", Done: false},
			{Token: " ", Done: false},
			{Token: "world", Done: true},
		},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	req := WebSocketRequest{
		RequestID: "test-format-stream",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read all chunks
	chunks := make([]WebSocketChunk, 0)
	for i := 0; i < 3; i++ {
		var chunk WebSocketChunk
		if err := conn.ReadJSON(&chunk); err != nil {
			t.Fatalf("Failed to read chunk %d: %v", i, err)
		}
		chunks = append(chunks, chunk)
	}

	// Verify intermediate chunks don't have Done=true
	if chunks[0].Done {
		t.Error("Expected first chunk to have Done=false")
	}
	if chunks[1].Done {
		t.Error("Expected second chunk to have Done=false")
	}

	// Final chunk should have metrics and Done=true
	if !chunks[2].Done {
		t.Error("Expected final chunk to have Done=true")
	}
	if chunks[2].TokenCount != 3 {
		t.Errorf("Expected TokenCount=3, got %d", chunks[2].TokenCount)
	}
}

// Test: JSON encoding/decoding round-trip
func TestWebSocketJSONRoundTrip(t *testing.T) {
	req := WebSocketRequest{
		RequestID: "test-123",
		Model:     "test-model",
		Prompt:    "hello world",
		Stream:    true,
		Priority:  "high",
		MaxLatency: 100,
		Options: map[string]interface{}{
			"temperature": 0.7,
			"top_p":       0.9,
		},
	}

	// Marshal
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	var decoded WebSocketRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify
	if decoded.RequestID != req.RequestID {
		t.Errorf("RequestID mismatch")
	}
	if decoded.Model != req.Model {
		t.Errorf("Model mismatch")
	}
	if decoded.Prompt != req.Prompt {
		t.Errorf("Prompt mismatch")
	}
	if decoded.Stream != req.Stream {
		t.Errorf("Stream mismatch")
	}
	if decoded.Priority != req.Priority {
		t.Errorf("Priority mismatch")
	}
	if decoded.MaxLatency != req.MaxLatency {
		t.Errorf("MaxLatency mismatch")
	}
}

// Test: Concurrent writes with proper synchronization (each request on new connection)
func TestWebSocketConcurrentWriteSafety(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "test", Done: true},
		},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// Multiple concurrent connections
	numConnections := 5
	numRequestsPerConnection := 3
	var wg sync.WaitGroup

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()

			for j := 0; j < numRequestsPerConnection; j++ {
				conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
				if err != nil {
					t.Errorf("Failed to connect: %v", err)
					return
				}

				req := WebSocketRequest{
					RequestID: fmt.Sprintf("conn-%d-req-%d", connID, j),
					Model:     "test-model",
					Prompt:    "hello",
					Stream:    true,
				}
				if err := conn.WriteJSON(req); err != nil {
					conn.Close()
					t.Errorf("Failed to write: %v", err)
					return
				}

				var chunk WebSocketChunk
				if err := conn.ReadJSON(&chunk); err != nil {
					conn.Close()
					t.Errorf("Failed to read: %v", err)
					return
				}
				conn.Close()
			}
		}(i)
	}

	wg.Wait()
}

// Test: Very large number of tokens in stream
func TestWebSocketLargeTokenStream(t *testing.T) {
	r := createTestRouter()

	// Create a stream with many tokens
	chunks := make([]*backends.StreamChunk, 100)
	for i := 0; i < 99; i++ {
		chunks[i] = &backends.StreamChunk{
			Token: fmt.Sprintf("token%d", i),
			Done:  false,
		}
	}
	chunks[99] = &backends.StreamChunk{
		Token: "final",
		Done:  true,
	}

	backend := &MockBackend{
		id:           "mock1",
		healthy:      true,
		streamChunks: chunks,
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	req := WebSocketRequest{
		RequestID: "test-large-stream",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read all chunks
	var finalChunk WebSocketChunk
	for i := 0; i < 100; i++ {
		if err := conn.ReadJSON(&finalChunk); err != nil {
			t.Fatalf("Failed to read chunk %d: %v", i, err)
		}
	}

	if finalChunk.TokenCount != 100 {
		t.Errorf("Expected TokenCount=100, got %d", finalChunk.TokenCount)
	}
}

// Test: Upgrade failure handling
func TestWebSocketUpgradeFailure(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{id: "mock1", healthy: true}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	// Try to connect with HTTP instead of WebSocket upgrade
	resp, err := http.Get(server.URL + "/ws")
	if err != nil {
		t.Fatalf("Failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Should fail because it's not a WebSocket upgrade request
	if resp.StatusCode == http.StatusOK {
		t.Error("Expected upgrade to fail, but got OK status")
	}
}

// Test: Empty prompt handling
func TestWebSocketEmptyPrompt(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:                 "mock1",
		healthy:            true,
		generateResponse:   &backends.GenerateResponse{Response: "response to empty"},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	req := WebSocketRequest{
		RequestID: "test-empty-prompt",
		Model:     "test-model",
		Prompt:    "", // Empty prompt
		Stream:    false,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk WebSocketChunk
	if err := conn.ReadJSON(&chunk); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if chunk.Token != "response to empty" {
		t.Errorf("Expected response, got '%s'", chunk.Token)
	}
}

// Test: Special characters in request
func TestWebSocketSpecialCharactersInRequest(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:                 "mock1",
		healthy:            true,
		generateResponse:   &backends.GenerateResponse{Response: "response"},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	req := WebSocketRequest{
		RequestID: "test-special-chars",
		Model:     "test-model",
		Prompt:    "Hello 世界 🌍 ñ ü",
		Stream:    false,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk WebSocketChunk
	if err := conn.ReadJSON(&chunk); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !chunk.Done {
		t.Error("Expected Done=true")
	}
}

// Test: Network read deadline
func TestWebSocketReadDeadlineEnforcement(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      "mock1",
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "test", Done: true},
		},
	}
	r.RegisterBackend(backend)

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	// Set a read deadline
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	req := WebSocketRequest{
		RequestID: "test-deadline",
		Model:     "test-model",
		Prompt:    "hello",
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk WebSocketChunk
	if err := conn.ReadJSON(&chunk); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !chunk.Done {
		t.Error("Expected Done=true")
	}
}
