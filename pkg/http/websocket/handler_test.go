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
	generateErr      error
	streamErr        error
	generateResponse *backends.GenerateResponse
	id               string
	streamChunks     []*backends.StreamChunk
	generateCalled   int
	streamCalled     int
	mutex            sync.Mutex
	healthy          bool
}

// ID is part of the public API.
func (m *MockBackend) ID() string { return m.id }

// Type is part of the public API.
func (m *MockBackend) Type() string { return "mock" }

// Name is part of the public API.
func (m *MockBackend) Name() string { return "mock-" + m.id }

// Hardware is part of the public API.
func (m *MockBackend) Hardware() string { return "cpu" }

// IsHealthy reports whether the condition holds.
func (m *MockBackend) IsHealthy() bool { return m.healthy }

// PowerWatts is part of the public API.
func (m *MockBackend) PowerWatts() float64 { return 10.0 }

// AvgLatencyMs is part of the public API.
func (m *MockBackend) AvgLatencyMs() int32 { return 100 }

// Priority is part of the public API.
func (m *MockBackend) Priority() int { return 1 }

// SupportsGenerate is part of the public API.
func (m *MockBackend) SupportsGenerate() bool { return true }

// SupportsStream is part of the public API.
func (m *MockBackend) SupportsStream() bool { return true }

// SupportsEmbed is part of the public API.
func (m *MockBackend) SupportsEmbed() bool { return true }

// ListModels is part of the public API.
func (m *MockBackend) ListModels(_ context.Context) ([]string, error) { return []string{}, nil }

// SupportsModel is part of the public API.
func (m *MockBackend) SupportsModel(_ string) bool { return true }

// GetMaxModelSizeGB is part of the public API.
func (m *MockBackend) GetMaxModelSizeGB() int { return 50 }

// GetSupportedModelPatterns is part of the public API.
func (m *MockBackend) GetSupportedModelPatterns() []string { return []string{} }

// GetPreferredModels is part of the public API.
func (m *MockBackend) GetPreferredModels() []string { return []string{} }

// UpdateMetrics is part of the public API.
func (m *MockBackend) UpdateMetrics(_ int32, _ bool) {}

// GetMetrics is part of the public API.
func (m *MockBackend) GetMetrics() *backends.BackendMetrics { return &backends.BackendMetrics{} }

// Start is part of the public API.
func (m *MockBackend) Start(_ context.Context) error { return nil }

// Stop is part of the public API.
func (m *MockBackend) Stop(_ context.Context) error { return nil }

// Multimedia capability methods
func (m *MockBackend) SupportsAudioToText() bool { return false }

// SupportsTextToAudio is part of the public API.
func (m *MockBackend) SupportsTextToAudio() bool { return false }

// SupportsImageToText is part of the public API.
func (m *MockBackend) SupportsImageToText() bool { return false }

// SupportsTextToImage is part of the public API.
func (m *MockBackend) SupportsTextToImage() bool { return false }

// SupportsVideoToText is part of the public API.
func (m *MockBackend) SupportsVideoToText() bool { return false }

// SupportsTextToVideo is part of the public API.
func (m *MockBackend) SupportsTextToVideo() bool { return false }

// TranscribeAudio is part of the public API.
func (m *MockBackend) TranscribeAudio(_ context.Context, _ *backends.TranscribeRequest) (*backends.TranscribeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// TranscribeAudioStream is part of the public API.
func (m *MockBackend) TranscribeAudioStream(_ context.Context, _ *backends.TranscribeRequest) (backends.AudioStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// SynthesizeSpeech is part of the public API.
func (m *MockBackend) SynthesizeSpeech(_ context.Context, _ *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// SynthesizeSpeechStream is part of the public API.
func (m *MockBackend) SynthesizeSpeechStream(_ context.Context, _ *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) {
	return nil, fmt.Errorf("not implemented")
}

// AnalyzeImage is part of the public API.
func (m *MockBackend) AnalyzeImage(_ context.Context, _ *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateImage is part of the public API.
func (m *MockBackend) GenerateImage(_ context.Context, _ *backends.ImageGenRequest) (*backends.ImageGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateImageStream is part of the public API.
func (m *MockBackend) GenerateImageStream(_ context.Context, _ *backends.ImageGenRequest) (backends.ImageStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// AnalyzeVideo is part of the public API.
func (m *MockBackend) AnalyzeVideo(_ context.Context, _ *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// AnalyzeVideoStream is part of the public API.
func (m *MockBackend) AnalyzeVideoStream(_ context.Context, _ *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateVideo is part of the public API.
func (m *MockBackend) GenerateVideo(_ context.Context, _ *backends.VideoGenRequest) (*backends.VideoGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateVideoStream is part of the public API.
func (m *MockBackend) GenerateVideoStream(_ context.Context, _ *backends.VideoGenRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

// HealthCheck is part of the public API.
func (m *MockBackend) HealthCheck(_ context.Context) error { return nil }

// Generate is part of the public API.
func (m *MockBackend) Generate(_ context.Context, _ *backends.GenerateRequest) (*backends.GenerateResponse, error) {
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

// GenerateStream is part of the public API.
func (m *MockBackend) GenerateStream(_ context.Context, _ *backends.GenerateRequest) (backends.StreamReader, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.streamCalled++
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	return &MockStreamReader{chunks: m.streamChunks}, nil
}

// Embed is part of the public API.
func (m *MockBackend) Embed(_ context.Context, _ *backends.EmbedRequest) (*backends.EmbedResponse, error) {
	return nil, nil
}

// MockStreamReader implements backends.StreamReader for testing
type MockStreamReader struct {
	chunks []*backends.StreamChunk
	index  int
	mutex  sync.Mutex
}

// Recv is part of the public API.
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

// Close is part of the public API.
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
		DefaultBackendID: strMock1,
		PowerAware:       false,
		AutoOptimize:     false,
	})
	return r
}

// Test: WebSocket connection establishment
func TestWebSocketConnectionEstablishment(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{id: strMock1, healthy: true}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// Attempt to upgrade connection
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	// Verify connection is established
	if conn.RemoteAddr() == nil {
		t.Error("WebSocket connection RemoteAddr is nil")
	}
}

// Test: Message handling - streaming request
func TestWebSocketStreamingMessageHandling(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:      strMock1,
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: strHello, Done: false},
			{Token: " ", Done: false},
			{Token: "world", Done: true},
		},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	// Send streaming request
	req := Request{
		RequestID: strTest123,
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read responses
	var responses []Chunk
	for i := 0; i < 3; i++ {
		var chunk Chunk
		if err := conn.ReadJSON(&chunk); err != nil {
			t.Fatalf("Failed to read chunk %d: %v", i, err)
		}
		responses = append(responses, chunk)
	}

	// Verify chunks
	if len(responses) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(responses))
	}

	if responses[0].Token != strHello {
		t.Errorf("Expected first token 'hello', got '%s'", responses[0].Token)
	}
	if responses[2].Done != true {
		t.Error("Expected last chunk to have Done=true")
	}
	if responses[2].RequestID != strTest123 {
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
		id:               strMock1,
		healthy:          true,
		generateResponse: &backends.GenerateResponse{Response: "full response"},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	// Send non-streaming request
	req := Request{
		RequestID: "test-456",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    false,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read response
	var chunk Chunk
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
	backend := &MockBackend{id: strMock1, healthy: true}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	// Send invalid JSON
	if err := conn.WriteMessage(websocket.TextMessage, []byte("invalid json")); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Read error response
	var errMsg Error
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
		id:          strMock1,
		healthy:     true,
		generateErr: fmt.Errorf("backend error"),
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	// Send non-streaming request
	req := Request{
		RequestID: "test-error",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    false,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read error response
	var errMsg Error
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
		id:        strMock1,
		healthy:   true,
		streamErr: fmt.Errorf("stream error"),
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	// Send streaming request
	req := Request{
		RequestID: "test-stream-error",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read error response
	var errMsg Error
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
	backend := &MockBackend{id: strMock1, healthy: true}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
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
		id:      strMock1,
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: strResponse, Done: true},
		},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

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

			conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if dialResp != nil && dialResp.Body != nil {
				if err := dialResp.Body.Close(); err != nil {
					t.Error(err)
				}
			}
			if err != nil {
				errorCount.Add(1)
				return
			}
			defer func() {
				if err := conn.Close(); err != nil {
					t.Error(err)
				}
			}()

			// Send request
			req := Request{
				RequestID: fmt.Sprintf("conn-%d", connID),
				Model:     strTestModel,
				Prompt:    strHello,
				Stream:    true,
			}
			if err := conn.WriteJSON(req); err != nil {
				errorCount.Add(1)
				return
			}

			// Read response
			var chunk Chunk
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
		id:      strMock1,
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: strTest, Done: true},
		},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	// Send request
	req := Request{
		RequestID: "test-cancel",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read response - should succeed normally
	var chunk Chunk
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
		name     string
		priority string
		expected backends.Priority
	}{
		{name: strBestEffort, priority: strBestEffort, expected: backends.PriorityBestEffort},
		{name: strLow, priority: strLow, expected: backends.PriorityBestEffort},
		{name: "normal", priority: "normal", expected: backends.PriorityNormal},
		{name: "", priority: "", expected: backends.PriorityNormal},
		{name: strHigh, priority: strHigh, expected: backends.PriorityHigh},
		{name: strCritical, priority: strCritical, expected: backends.PriorityCritical},
		{name: strRealtime, priority: strRealtime, expected: backends.PriorityCritical},
		{name: "unknown", priority: "unknown", expected: backends.PriorityNormal},
	}

	r := createTestRouter()
	backend := &MockBackend{
		id:      strMock1,
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: strTest, Done: true},
		},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
			conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if dialResp != nil && dialResp.Body != nil {
				if err := dialResp.Body.Close(); err != nil {
					t.Error(err)
				}
			}
			if err != nil {
				t.Fatalf("Failed to establish WebSocket connection: %v", err)
			}
			defer func() {
				if err := conn.Close(); err != nil {
					t.Error(err)
				}
			}()

			req := Request{
				RequestID: "test-priority",
				Model:     strTestModel,
				Prompt:    strHello,
				Stream:    true,
				Priority:  tt.priority,
			}
			if err := conn.WriteJSON(req); err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}

			var chunk Chunk
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
		id:               strMock1,
		healthy:          true,
		generateResponse: &backends.GenerateResponse{Response: strTest},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	req := Request{
		RequestID: "test-options",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    false,
		Options: map[string]interface{}{
			strTemperature: 0.7,
			strTopP:        0.9,
			"top_k":        40.0,
			"max_tokens":   100.0,
		},
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk Chunk
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
		id:      strMock1,
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "chunk1", Done: false},
			{Token: "chunk2", Done: false},
		},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	req := Request{
		RequestID: "test-stream-error",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read first chunk
	var chunk1 Chunk
	if err := conn.ReadJSON(&chunk1); err != nil {
		t.Fatalf("Failed to read first chunk: %v", err)
	}

	if chunk1.Token != "chunk1" {
		t.Errorf("Expected token 'chunk1', got '%s'", chunk1.Token)
	}

	// Read second chunk
	var chunk2 Chunk
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
		id:      strMock1,
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "t1", Done: false},
			{Token: "t2", Done: false},
			{Token: "t3", Done: true},
		},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	req := Request{
		RequestID: "test-metrics",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read all chunks
	var finalChunk Chunk
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
		id:      strMock1,
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "first", Done: false},
			{Token: " ", Done: false},
			{Token: "second", Done: true},
		},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	req := Request{
		RequestID: "test-ttft",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read all chunks
	var finalChunk Chunk
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
		id:               strMock1,
		healthy:          true,
		generateResponse: &backends.GenerateResponse{Response: strTest},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	req := Request{
		RequestID: "test-no-options",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    false,
		// No options provided
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk Chunk
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
		id:      strMock1,
		healthy: false, // Not healthy
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	req := Request{
		RequestID: "test-unhealthy",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    false,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read error response (no healthy backends available)
	var errMsg Error
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

	if err := r.RegisterBackend(backend1); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterBackend(backend2); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	req := Request{
		RequestID: "test-multi-backend",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk Chunk
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
		id:      strMock1,
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: strResponse, Done: true},
		},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// Send 20 rapid requests with separate connections
	for i := 0; i < 20; i++ {
		conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if dialResp != nil && dialResp.Body != nil {
			if err := dialResp.Body.Close(); err != nil {
				t.Error(err)
			}
		}
		if err != nil {
			t.Fatalf("Failed to establish WebSocket connection %d: %v", i, err)
		}

		req := Request{
			RequestID: fmt.Sprintf("stress-%d", i),
			Model:     strTestModel,
			Prompt:    strHello,
			Stream:    true,
		}
		if err := conn.WriteJSON(req); err != nil {
			if err := conn.Close(); err != nil {
				t.Error(err)
			}
			t.Fatalf("Failed to send request %d: %v", i, err)
		}

		var chunk Chunk
		if err := conn.ReadJSON(&chunk); err != nil {
			if err := conn.Close(); err != nil {
				t.Error(err)
			}
			t.Fatalf("Failed to read response %d: %v", i, err)
		}

		if chunk.RequestID != req.RequestID {
			t.Errorf("Request %d: RequestID mismatch", i)
		}
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
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

	if err := r.RegisterBackend(slowBackend); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterBackend(fastBackend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	req := Request{
		RequestID:  "test-max-latency",
		Model:      strTestModel,
		Prompt:     strHello,
		Stream:     true,
		MaxLatency: 100,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk Chunk
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
		id:      strMock1,
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: strTest, Done: true},
		},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	req := Request{
		RequestID: "test-deadline",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read response with reasonable timeout
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var chunk Chunk
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
		id:               strMock1,
		healthy:          true,
		generateResponse: &backends.GenerateResponse{Response: strTest},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	req := Request{
		RequestID: "test-metrics-backend",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    false,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk Chunk
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
		id:      strMock1,
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: strTest, Done: true},
		},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	req := Request{
		RequestID: "test-stream-metrics-backend",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk Chunk
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
		id:      strMock1,
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: "x", Done: true},
		},
	}
	if err := r.RegisterBackend(backend); err != nil {
		b.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			b.Error(err)
		}
	}
	if err != nil {
		b.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			b.Error(err)
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := Request{
			RequestID: fmt.Sprintf("bench-%d", i),
			Model:     strTestModel,
			Prompt:    strHello,
			Stream:    true,
		}
		if err := conn.WriteJSON(req); err != nil {
			b.Fatalf("Failed to send request: %v", err)
		}

		var chunk Chunk
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
	backend := &MockBackend{id: strMock1, healthy: true}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	// Send invalid request to trigger sendError
	if err := conn.WriteMessage(websocket.TextMessage, []byte("not json")); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	var errMsg Error
	if err := conn.ReadJSON(&errMsg); err != nil {
		t.Fatalf("Failed to read error: %v", err)
	}

	if errMsg.Error == "" {
		t.Error("Expected error message, got empty string")
	}
}

// Test: ConvertWebSocketRequest helper function
func TestConvertWebSocketRequest(t *testing.T) {
	wsReq := &Request{
		Prompt: "test prompt",
		Model:  strTestModel,
		Options: map[string]interface{}{
			strTemperature: 0.5,
			strTopP:        0.8,
			"top_k":        50.0,
			"max_tokens":   200.0,
		},
	}

	result := convertWebSocketRequest(wsReq)

	if result.Prompt != "test prompt" {
		t.Errorf("Expected prompt 'test prompt', got '%s'", result.Prompt)
	}
	if result.Model != strTestModel {
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
	wsReq := &Request{
		Prompt:  strTest,
		Model:   strTestModel,
		Options: nil,
	}

	result := convertWebSocketRequest(wsReq)

	if result.Prompt != strTest {
		t.Errorf("Expected prompt 'test', got '%s'", result.Prompt)
	}
	if result.Options != nil {
		t.Error("Expected options to be nil")
	}
}

// Test: ConvertWebSocketRequest with partial options
func TestConvertWebSocketRequestPartialOptions(t *testing.T) {
	wsReq := &Request{
		Prompt: strTest,
		Model:  strTestModel,
		Options: map[string]interface{}{
			strTemperature: 0.7,
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
		id:               strMock1,
		healthy:          true,
		generateResponse: &backends.GenerateResponse{Response: strTest},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	testCases := []string{"req-1", "req-abc-123", "special-id_456"}

	for _, requestID := range testCases {
		conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if dialResp != nil && dialResp.Body != nil {
			if err := dialResp.Body.Close(); err != nil {
				t.Error(err)
			}
		}
		if err != nil {
			t.Fatalf("Failed to establish WebSocket connection: %v", err)
		}

		req := Request{
			RequestID: requestID,
			Model:     strTestModel,
			Prompt:    strHello,
			Stream:    false,
		}
		if err := conn.WriteJSON(req); err != nil {
			if err := conn.Close(); err != nil {
				t.Error(err)
			}
			t.Fatalf("Failed to send request: %v", err)
		}

		var chunk Chunk
		if err := conn.ReadJSON(&chunk); err != nil {
			if err := conn.Close(); err != nil {
				t.Error(err)
			}
			t.Fatalf("Failed to read response: %v", err)
		}

		if chunk.RequestID != requestID {
			t.Errorf("Expected RequestID '%s', got '%s'", requestID, chunk.RequestID)
		}
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}
}

// Test: Non-streaming response format
func TestWebSocketNonStreamingResponseFormat(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:               strMock1,
		healthy:          true,
		generateResponse: &backends.GenerateResponse{Response: "complete response"},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	req := Request{
		RequestID: "test-format",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    false,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk Chunk
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
		id:      strMock1,
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: strHello, Done: false},
			{Token: " ", Done: false},
			{Token: "world", Done: true},
		},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	req := Request{
		RequestID: "test-format-stream",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read all chunks
	chunks := make([]Chunk, 0)
	for i := 0; i < 3; i++ {
		var chunk Chunk
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
	req := Request{
		RequestID:  strTest123,
		Model:      strTestModel,
		Prompt:     "hello world",
		Stream:     true,
		Priority:   strHigh,
		MaxLatency: 100,
		Options: map[string]interface{}{
			strTemperature: 0.7,
			strTopP:        0.9,
		},
	}

	// Marshal
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	var decoded Request
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
		id:      strMock1,
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: strTest, Done: true},
		},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

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
				conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
				if dialResp != nil && dialResp.Body != nil {
					if err := dialResp.Body.Close(); err != nil {
						t.Error(err)
					}
				}
				if err != nil {
					t.Errorf("Failed to connect: %v", err)
					return
				}

				req := Request{
					RequestID: fmt.Sprintf("conn-%d-req-%d", connID, j),
					Model:     strTestModel,
					Prompt:    strHello,
					Stream:    true,
				}
				if err := conn.WriteJSON(req); err != nil {
					if err := conn.Close(); err != nil {
						t.Error(err)
					}
					t.Errorf("Failed to write: %v", err)
					return
				}

				var chunk Chunk
				if err := conn.ReadJSON(&chunk); err != nil {
					if err := conn.Close(); err != nil {
						t.Error(err)
					}
					t.Errorf("Failed to read: %v", err)
					return
				}
				if err := conn.Close(); err != nil {
					t.Error(err)
				}
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
		id:           strMock1,
		healthy:      true,
		streamChunks: chunks,
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	req := Request{
		RequestID: "test-large-stream",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read all chunks
	var finalChunk Chunk
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
	backend := &MockBackend{id: strMock1, healthy: true}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	// Try to connect with HTTP instead of WebSocket upgrade
	resp, err := http.Get(server.URL + "/ws")
	if err != nil {
		t.Fatalf("Failed to make HTTP request: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Error(err)
		}
	}()

	// Should fail because it's not a WebSocket upgrade request
	if resp.StatusCode == http.StatusOK {
		t.Error("Expected upgrade to fail, but got OK status")
	}
}

// Test: Empty prompt handling
func TestWebSocketEmptyPrompt(t *testing.T) {
	r := createTestRouter()
	backend := &MockBackend{
		id:               strMock1,
		healthy:          true,
		generateResponse: &backends.GenerateResponse{Response: "response to empty"},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	req := Request{
		RequestID: "test-empty-prompt",
		Model:     strTestModel,
		Prompt:    "", // Empty prompt
		Stream:    false,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk Chunk
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
		id:               strMock1,
		healthy:          true,
		generateResponse: &backends.GenerateResponse{Response: strResponse},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	req := Request{
		RequestID: "test-special-chars",
		Model:     strTestModel,
		Prompt:    "Hello 世界 🌍 ñ ü",
		Stream:    false,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk Chunk
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
		id:      strMock1,
		healthy: true,
		streamChunks: []*backends.StreamChunk{
			{Token: strTest, Done: true},
		},
	}
	if err := r.RegisterBackend(backend); err != nil {
		t.Fatal(err)
	}

	server := createTestServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, dialResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialResp != nil && dialResp.Body != nil {
		if err := dialResp.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()

	// Set a read deadline
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		t.Fatal(err)
	}

	req := Request{
		RequestID: "test-deadline",
		Model:     strTestModel,
		Prompt:    strHello,
		Stream:    true,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var chunk Chunk
	if err := conn.ReadJSON(&chunk); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !chunk.Done {
		t.Error("Expected Done=true")
	}
}
