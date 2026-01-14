package websocket

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/logging"
	"github.com/daoneill/ollama-proxy/pkg/router"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper CORS policy
		return true
	},
}

// WebSocketRequest represents an incoming WebSocket request
type WebSocketRequest struct {
	RequestID   string                 `json:"request_id"`
	Model       string                 `json:"model"`
	Prompt      string                 `json:"prompt"`
	Options     map[string]interface{} `json:"options,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Priority    string                 `json:"priority,omitempty"` // "best-effort", "normal", "high", "critical"
	MaxLatency  int32                  `json:"max_latency_ms,omitempty"`
}

// WebSocketChunk represents a streaming response chunk
type WebSocketChunk struct {
	RequestID string  `json:"request_id"`
	Token     string  `json:"token,omitempty"`
	Done      bool    `json:"done"`
	Error     *string `json:"error,omitempty"`

	// Performance metrics
	TTFT        int64   `json:"ttft_ms,omitempty"`         // Time to first token
	TotalTimeMs int64   `json:"total_time_ms,omitempty"`
	TokenCount  int     `json:"token_count,omitempty"`
	TokensPerSec float32 `json:"tokens_per_sec,omitempty"`
}

// WebSocketError represents an error response
type WebSocketError struct {
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

// HandleWebSocketStream provides low-latency WebSocket streaming
func HandleWebSocketStream(r *router.Router) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Upgrade to WebSocket
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			logging.Logger.Error("WebSocket upgrade failed", zap.Error(err))
			return
		}
		defer conn.Close()

		// Set read deadline
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// Read initial request
		var streamReq WebSocketRequest
		if err := conn.ReadJSON(&streamReq); err != nil {
			sendError(conn, "invalid request", streamReq.RequestID)
			return
		}

		// Parse priority
		priority := backends.PriorityNormal
		switch streamReq.Priority {
		case "best-effort", "low":
			priority = backends.PriorityBestEffort
		case "high":
			priority = backends.PriorityHigh
		case "critical", "realtime":
			priority = backends.PriorityCritical
		}

		// Build annotations for routing
		annotations := &backends.Annotations{
			LatencyCritical: priority >= backends.PriorityHigh,
			Priority:        priority,
			RequestID:       streamReq.RequestID,
			MediaType:       backends.MediaTypeRealtime,
			MaxLatencyMs:    streamReq.MaxLatency,
		}

		// Route request
		decision, err := r.RouteRequest(req.Context(), annotations)
		if err != nil {
			sendError(conn, fmt.Sprintf("routing failed: %v", err), streamReq.RequestID)
			return
		}

		// Log routing decision
		logging.Logger.Info("WebSocket request routed",
			zap.String("request_id", streamReq.RequestID),
			zap.String("backend", decision.Backend.ID()),
			zap.Int("priority", int(priority)),
		)

		// Convert WebSocket request to internal format
		internalReq := convertWebSocketRequest(&streamReq)

		// Start streaming
		if streamReq.Stream {
			handleStreamingRequest(conn, decision.Backend, internalReq, &streamReq)
		} else {
			handleNonStreamingRequest(conn, decision.Backend, internalReq, &streamReq)
		}
	}
}

// handleStreamingRequest processes a streaming WebSocket request
func handleStreamingRequest(conn *websocket.Conn, backend backends.Backend, req *backends.GenerateRequest, wsReq *WebSocketRequest) {
	ctx := context.Background()
	startTime := time.Now()

	reader, err := backend.GenerateStream(ctx, req)
	if err != nil {
		sendError(conn, fmt.Sprintf("stream start failed: %v", err), wsReq.RequestID)
		return
	}
	defer reader.Close()

	var firstTokenTime *time.Time
	tokenCount := 0

	// Stream chunks directly with minimal transformation
	for {
		chunk, err := reader.Recv()
		if err != nil {
			if err != io.EOF {
				errorMsg := err.Error()
				wsChunk := WebSocketChunk{
					RequestID: wsReq.RequestID,
					Done:      true,
					Error:     &errorMsg,
				}
				conn.WriteJSON(wsChunk)
			}
			break
		}

		// Track TTFT
		now := time.Now()
		if firstTokenTime == nil && chunk.Token != "" {
			firstTokenTime = &now
		}
		tokenCount++

		// Passthrough mode: Send chunk with minimal transformation
		wsChunk := WebSocketChunk{
			RequestID: wsReq.RequestID,
			Token:     chunk.Token,
			Done:      chunk.Done,
		}

		// Add metrics on final chunk
		if chunk.Done {
			elapsed := time.Since(startTime)
			wsChunk.TotalTimeMs = elapsed.Milliseconds()
			wsChunk.TokenCount = tokenCount

			if firstTokenTime != nil {
				wsChunk.TTFT = firstTokenTime.Sub(startTime).Milliseconds()
			}

			if elapsed.Seconds() > 0 {
				wsChunk.TokensPerSec = float32(tokenCount) / float32(elapsed.Seconds())
			}
		}

		// Write with timeout
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := conn.WriteJSON(wsChunk); err != nil {
			logging.Logger.Error("WebSocket write failed", zap.Error(err))
			break
		}

		if chunk.Done {
			break
		}
	}
}

// handleNonStreamingRequest processes a non-streaming WebSocket request
func handleNonStreamingRequest(conn *websocket.Conn, backend backends.Backend, req *backends.GenerateRequest, wsReq *WebSocketRequest) {
	ctx := context.Background()
	startTime := time.Now()

	response, err := backend.Generate(ctx, req)
	if err != nil {
		sendError(conn, fmt.Sprintf("generation failed: %v", err), wsReq.RequestID)
		return
	}

	elapsed := time.Since(startTime)

	// Send complete response
	wsChunk := WebSocketChunk{
		RequestID:    wsReq.RequestID,
		Token:        response.Response,
		Done:         true,
		TotalTimeMs:  elapsed.Milliseconds(),
		TokenCount:   1, // Non-streaming returns full response as one token
		TokensPerSec: 1.0 / float32(elapsed.Seconds()),
	}

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := conn.WriteJSON(wsChunk); err != nil {
		logging.Logger.Error("WebSocket write failed", zap.Error(err))
	}
}

// convertWebSocketRequest converts WebSocket request to internal format
func convertWebSocketRequest(wsReq *WebSocketRequest) *backends.GenerateRequest {
	req := &backends.GenerateRequest{
		Prompt: wsReq.Prompt,
		Model:  wsReq.Model,
	}

	// Parse options if provided
	if wsReq.Options != nil {
		options := &backends.GenerationOptions{}

		if temp, ok := wsReq.Options["temperature"].(float64); ok {
			options.Temperature = float32(temp)
		}
		if topP, ok := wsReq.Options["top_p"].(float64); ok {
			options.TopP = float32(topP)
		}
		if topK, ok := wsReq.Options["top_k"].(float64); ok {
			options.TopK = int32(topK)
		}
		if maxTokens, ok := wsReq.Options["max_tokens"].(float64); ok {
			options.MaxTokens = int32(maxTokens)
		}

		req.Options = options
	}

	return req
}

// sendError sends an error message via WebSocket
func sendError(conn *websocket.Conn, message string, requestID string) {
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	conn.WriteJSON(WebSocketError{
		Error:     message,
		RequestID: requestID,
	})
}
