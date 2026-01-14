package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/logging"
	"github.com/daoneill/ollama-proxy/pkg/router"
	"go.uber.org/zap"
)

// HandleChatCompletion handles /v1/chat/completions endpoint
func HandleChatCompletion(r *router.Router) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Only accept POST
		if req.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "method_not_allowed")
			return
		}

		// Parse request body
		var chatReq ChatCompletionRequest
		if err := json.NewDecoder(req.Body).Decode(&chatReq); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body", "invalid_request_error")
			return
		}

		// Validate required fields
		if chatReq.Model == "" {
			writeError(w, http.StatusBadRequest, "Model is required", "invalid_request_error")
			return
		}
		if len(chatReq.Messages) == 0 {
			writeError(w, http.StatusBadRequest, "Messages are required", "invalid_request_error")
			return
		}

		// Parse routing headers
		annotations := ParseRoutingHeaders(req)

		// Convert to internal format
		internalReq := ConvertChatCompletionRequest(&chatReq)

		// Route request
		decision, err := r.RouteRequest(req.Context(), annotations)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, fmt.Sprintf("Routing failed: %v", err), "service_unavailable")
			return
		}

		// Check if backend supports the model
		if !decision.Backend.SupportsModel(chatReq.Model) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("Model %s not available", chatReq.Model), "model_not_found")
			return
		}

		// Handle streaming vs non-streaming
		if chatReq.Stream {
			handleChatCompletionStreaming(w, req.Context(), decision, internalReq, &chatReq)
		} else {
			handleChatCompletionNonStreaming(w, req.Context(), decision, internalReq, &chatReq)
		}
	}
}

func handleChatCompletionNonStreaming(w http.ResponseWriter, ctx context.Context, decision *router.RoutingDecision, internalReq *backends.GenerateRequest, chatReq *ChatCompletionRequest) {
	// Execute request
	resp, err := decision.Backend.Generate(ctx, internalReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Generation failed: %v", err), "internal_error")
		return
	}

	// Convert to OpenAI format
	openaiResp := ConvertToOpenAIChatResponse(chatReq, resp)

	// Write routing headers
	WriteRoutingHeaders(w, decision)

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(openaiResp)
}

func handleChatCompletionStreaming(w http.ResponseWriter, ctx context.Context, decision *router.RoutingDecision, internalReq *backends.GenerateRequest, chatReq *ChatCompletionRequest) {
	// Check if backend supports streaming
	if !decision.Backend.SupportsStream() {
		writeError(w, http.StatusBadRequest, "Backend does not support streaming", "invalid_request_error")
		return
	}

	// Execute streaming request
	reader, err := decision.Backend.GenerateStream(ctx, internalReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Streaming failed: %v", err), "internal_error")
		return
	}

	// Write routing headers before streaming
	WriteRoutingHeaders(w, decision)

	// Generate completion ID
	completionID := generateCompletionID("chatcmpl")

	// Stream response
	if err := StreamChatCompletion(w, reader, chatReq.Model, completionID); err != nil {
		// Can't send error after streaming has started
		// Just log it
		if logging.Logger != nil {
			logging.Logger.Error("Streaming error", zap.Error(err))
		}
	}
}

// HandleCompletion handles /v1/completions endpoint
func HandleCompletion(r *router.Router) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Only accept POST
		if req.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "method_not_allowed")
			return
		}

		// Parse request body
		var compReq CompletionRequest
		if err := json.NewDecoder(req.Body).Decode(&compReq); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body", "invalid_request_error")
			return
		}

		// Validate required fields
		if compReq.Model == "" {
			writeError(w, http.StatusBadRequest, "Model is required", "invalid_request_error")
			return
		}
		if compReq.Prompt == nil {
			writeError(w, http.StatusBadRequest, "Prompt is required", "invalid_request_error")
			return
		}

		// Parse routing headers
		annotations := ParseRoutingHeaders(req)

		// Convert to internal format
		internalReq := ConvertCompletionRequest(&compReq)

		// Route request
		decision, err := r.RouteRequest(req.Context(), annotations)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, fmt.Sprintf("Routing failed: %v", err), "service_unavailable")
			return
		}

		// Check if backend supports the model
		if !decision.Backend.SupportsModel(compReq.Model) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("Model %s not available", compReq.Model), "model_not_found")
			return
		}

		// Handle streaming vs non-streaming
		if compReq.Stream {
			handleCompletionStreaming(w, req.Context(), decision, internalReq, &compReq)
		} else {
			handleCompletionNonStreaming(w, req.Context(), decision, internalReq, &compReq)
		}
	}
}

func handleCompletionNonStreaming(w http.ResponseWriter, ctx context.Context, decision *router.RoutingDecision, internalReq *backends.GenerateRequest, compReq *CompletionRequest) {
	// Execute request
	resp, err := decision.Backend.Generate(ctx, internalReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Generation failed: %v", err), "internal_error")
		return
	}

	// Convert to OpenAI format
	openaiResp := ConvertToOpenAICompletionResponse(compReq, resp)

	// Write routing headers
	WriteRoutingHeaders(w, decision)

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(openaiResp)
}

func handleCompletionStreaming(w http.ResponseWriter, ctx context.Context, decision *router.RoutingDecision, internalReq *backends.GenerateRequest, compReq *CompletionRequest) {
	// Check if backend supports streaming
	if !decision.Backend.SupportsStream() {
		writeError(w, http.StatusBadRequest, "Backend does not support streaming", "invalid_request_error")
		return
	}

	// Execute streaming request
	reader, err := decision.Backend.GenerateStream(ctx, internalReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Streaming failed: %v", err), "internal_error")
		return
	}

	// Write routing headers before streaming
	WriteRoutingHeaders(w, decision)

	// Generate completion ID
	completionID := generateCompletionID("cmpl")

	// Stream response
	if err := StreamCompletion(w, reader, compReq.Model, completionID); err != nil {
		// Can't send error after streaming has started
		fmt.Printf("Streaming error: %v\n", err)
	}
}

// HandleEmbedding handles /v1/embeddings endpoint
func HandleEmbedding(r *router.Router) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Only accept POST
		if req.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "method_not_allowed")
			return
		}

		// Parse request body
		var embedReq EmbeddingRequest
		if err := json.NewDecoder(req.Body).Decode(&embedReq); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body", "invalid_request_error")
			return
		}

		// Validate required fields
		if embedReq.Model == "" {
			writeError(w, http.StatusBadRequest, "Model is required", "invalid_request_error")
			return
		}
		if embedReq.Input == nil {
			writeError(w, http.StatusBadRequest, "Input is required", "invalid_request_error")
			return
		}

		// Parse routing headers
		annotations := ParseRoutingHeaders(req)

		// Convert to internal format
		internalReq := ConvertEmbeddingRequest(&embedReq)

		// Route request
		decision, err := r.RouteRequest(req.Context(), annotations)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, fmt.Sprintf("Routing failed: %v", err), "service_unavailable")
			return
		}

		// Check if backend supports embeddings
		if !decision.Backend.SupportsEmbed() {
			writeError(w, http.StatusBadRequest, "Backend does not support embeddings", "invalid_request_error")
			return
		}

		// Check if backend supports the model
		if !decision.Backend.SupportsModel(embedReq.Model) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("Model %s not available", embedReq.Model), "model_not_found")
			return
		}

		// Execute request
		resp, err := decision.Backend.Embed(req.Context(), internalReq)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("Embedding failed: %v", err), "internal_error")
			return
		}

		// Convert to OpenAI format
		openaiResp := ConvertToOpenAIEmbeddingResponse(&embedReq, resp)

		// Write routing headers
		WriteRoutingHeaders(w, decision)

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(openaiResp)
	}
}

// HandleModels handles /v1/models endpoint
func HandleModels(r *router.Router) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Only accept GET
		if req.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "method_not_allowed")
			return
		}

		// Get all backends
		backends := r.ListBackends()

		// Collect all models from all backends
		modelsMap := make(map[string]bool)
		for _, backend := range backends {
			if !backend.IsHealthy() {
				continue
			}

			models, err := backend.ListModels(req.Context())
			if err != nil {
				// Skip backends that fail to list models
				continue
			}

			for _, model := range models {
				modelsMap[model] = true
			}

			// Also add preferred models
			for _, model := range backend.GetPreferredModels() {
				modelsMap[model] = true
			}
		}

		// Convert to model list
		var modelsList []Model
		timestamp := time.Now().Unix()

		for modelID := range modelsMap {
			modelsList = append(modelsList, Model{
				ID:      modelID,
				Object:  "model",
				Created: timestamp,
				OwnedBy: "ollama-proxy",
			})
		}

		// Build response
		response := ModelsResponse{
			Object: "list",
			Data:   modelsList,
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// writeError writes an OpenAI-compatible error response
func writeError(w http.ResponseWriter, statusCode int, message string, errorType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := ErrorResponse{
		Error: ErrorDetail{
			Message: message,
			Type:    errorType,
			Code:    errorType,
		},
	}

	json.NewEncoder(w).Encode(errorResp)
}
