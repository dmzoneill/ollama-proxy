// Package websocket serves streaming chat over WebSocket.
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
	CheckOrigin: func(_ *http.Request) bool {
		// TODO: Implement proper CORS policy
		return true
	},
}

// Request represents an incoming WebSocket request
type Request struct {
	Options    map[string]interface{} `json:"options,omitempty"`
	RequestID  string                 `json:"request_id"`
	Model      string                 `json:"model"`
	Prompt     string                 `json:"prompt"`
	Priority   string                 `json:"priority,omitempty"`
	MaxLatency int32                  `json:"max_latency_ms,omitempty"`
	Stream     bool                   `json:"stream,omitempty"`
}

// Chunk represents a streaming response chunk
type Chunk struct {
	Error        *string `json:"error,omitempty"`
	RequestID    string  `json:"request_id"`
	Token        string  `json:"token,omitempty"`
	TTFT         int64   `json:"ttft_ms,omitempty"`
	TotalTimeMs  int64   `json:"total_time_ms,omitempty"`
	TokenCount   int     `json:"token_count,omitempty"`
	TokensPerSec float32 `json:"tokens_per_sec,omitempty"`
	Done         bool    `json:"done"`
}

// Error represents an error response
type Error struct {
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
		defer func() {
			if err := conn.Close(); err != nil {
				logging.Warn("close websocket", zap.Error(err))
			}
		}()

		if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			logging.Warn("set websocket read deadline", zap.Error(err))
			return
		}

		// Read initial request
		var streamReq Request
		if err := conn.ReadJSON(&streamReq); err != nil {
			sendError(conn, "invalid request", streamReq.RequestID)
			return
		}

		// Parse priority
		priority := backends.PriorityNormal
		switch streamReq.Priority {
		case "best-effort", "low":
			priority = backends.PriorityBestEffort
		case strHigh:
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
func handleStreamingRequest(conn *websocket.Conn, backend backends.Backend, req *backends.GenerateRequest, wsReq *Request) {
	ctx := context.Background()
	startTime := time.Now()

	reader, err := backend.GenerateStream(ctx, req)
	if err != nil {
		sendError(conn, fmt.Sprintf("stream start failed: %v", err), wsReq.RequestID)
		return
	}
	defer func() {
		if err := reader.Close(); err != nil {
			logging.Warn("close websocket stream", zap.Error(err))
		}
	}()

	var firstTokenTime *time.Time
	tokenCount := 0

	// Stream chunks directly with minimal transformation
	for {
		chunk, err := reader.Recv()
		if err != nil {
			if err != io.EOF {
				errorMsg := err.Error()
				wsChunk := Chunk{
					RequestID: wsReq.RequestID,
					Done:      true,
					Error:     &errorMsg,
				}
				if err := conn.WriteJSON(wsChunk); err != nil {
					logging.Warn("write websocket error chunk", zap.Error(err))
				}
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
		wsChunk := Chunk{
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
		if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
			logging.Warn("set websocket write deadline", zap.Error(err))
			break
		}
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
func handleNonStreamingRequest(conn *websocket.Conn, backend backends.Backend, req *backends.GenerateRequest, wsReq *Request) {
	ctx := context.Background()
	startTime := time.Now()

	response, err := backend.Generate(ctx, req)
	if err != nil {
		sendError(conn, fmt.Sprintf("generation failed: %v", err), wsReq.RequestID)
		return
	}

	elapsed := time.Since(startTime)

	// Send complete response
	wsChunk := Chunk{
		RequestID:    wsReq.RequestID,
		Token:        response.Response,
		Done:         true,
		TotalTimeMs:  elapsed.Milliseconds(),
		TokenCount:   1, // Non-streaming returns full response as one token
		TokensPerSec: 1.0 / float32(elapsed.Seconds()),
	}

	if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		logging.Warn("set websocket write deadline", zap.Error(err))
		return
	}
	if err := conn.WriteJSON(wsChunk); err != nil {
		logging.Logger.Error("WebSocket write failed", zap.Error(err))
	}
}

// convertWebSocketRequest converts WebSocket request to internal format
func convertWebSocketRequest(wsReq *Request) *backends.GenerateRequest {
	req := &backends.GenerateRequest{
		Prompt: wsReq.Prompt,
		Model:  wsReq.Model,
	}

	// Parse options if provided
	if wsReq.Options != nil {
		options := &backends.GenerationOptions{}

		if temp, ok := wsReq.Options[strTemperature].(float64); ok {
			options.Temperature = float32(temp)
		}
		if topP, ok := wsReq.Options[strTopP].(float64); ok {
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
	if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		logging.Warn("set websocket error deadline", zap.Error(err))
		return
	}
	if err := conn.WriteJSON(Error{
		Error:     message,
		RequestID: requestID,
	}); err != nil {
		logging.Warn("write websocket error", zap.Error(err))
	}
}
