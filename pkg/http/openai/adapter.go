package openai

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// ConvertChatCompletionRequest converts OpenAI chat completion request to internal format
func ConvertChatCompletionRequest(req *ChatCompletionRequest) *backends.GenerateRequest {
	// Concatenate messages into a single prompt with role markers
	prompt := buildPromptFromMessages(req.Messages)

	// Build generation options
	options := &backends.GenerationOptions{}

	if req.Temperature != nil {
		options.Temperature = *req.Temperature
	}

	if req.TopP != nil {
		options.TopP = *req.TopP
	}

	if req.MaxTokens != nil {
		options.MaxTokens = *req.MaxTokens
	}

	if len(req.Stop) > 0 {
		options.Stop = req.Stop
	}

	return &backends.GenerateRequest{
		Prompt:  prompt,
		Model:   req.Model,
		Options: options,
	}
}

// ConvertCompletionRequest converts OpenAI completion request to internal format
func ConvertCompletionRequest(req *CompletionRequest) *backends.GenerateRequest {
	// Extract prompt (can be string or []string)
	prompt := extractPrompt(req.Prompt)

	// Build generation options
	options := &backends.GenerationOptions{}

	if req.Temperature != nil {
		options.Temperature = *req.Temperature
	}

	if req.TopP != nil {
		options.TopP = *req.TopP
	}

	if req.MaxTokens != nil {
		options.MaxTokens = *req.MaxTokens
	}

	if len(req.Stop) > 0 {
		options.Stop = req.Stop
	}

	return &backends.GenerateRequest{
		Prompt:  prompt,
		Model:   req.Model,
		Options: options,
	}
}

// ConvertEmbeddingRequest converts OpenAI embedding request to internal format
func ConvertEmbeddingRequest(req *EmbeddingRequest) *backends.EmbedRequest {
	// Extract text (can be string or []string, we use the first for now)
	text := extractPrompt(req.Input)

	return &backends.EmbedRequest{
		Text:  text,
		Model: req.Model,
	}
}

// ConvertToOpenAIChatResponse converts internal response to OpenAI chat completion format
func ConvertToOpenAIChatResponse(req *ChatCompletionRequest, resp *backends.GenerateResponse) *ChatCompletionResponse {
	completionID := generateCompletionID("chatcmpl")
	timestamp := time.Now().Unix()

	// Estimate tokens (rough approximation)
	promptTokens := estimateTokens(buildPromptFromMessages(req.Messages))
	completionTokens := int32(0)
	if resp.Stats != nil {
		completionTokens = resp.Stats.TokensGenerated
	}
	if completionTokens == 0 {
		completionTokens = estimateTokens(resp.Response)
	}

	return &ChatCompletionResponse{
		ID:      completionID,
		Object:  "chat.completion",
		Created: timestamp,
		Model:   req.Model,
		Choices: []ChatCompletionChoice{
			{
				Index: 0,
				Message: ChatCompletionMessage{
					Role:    "assistant",
					Content: resp.Response,
				},
				FinishReason: "stop",
			},
		},
		Usage: ChatCompletionUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}
}

// ConvertToOpenAICompletionResponse converts internal response to OpenAI completion format
func ConvertToOpenAICompletionResponse(req *CompletionRequest, resp *backends.GenerateResponse) *CompletionResponse {
	completionID := generateCompletionID("cmpl")
	timestamp := time.Now().Unix()

	// Estimate tokens
	promptTokens := estimateTokens(extractPrompt(req.Prompt))
	completionTokens := int32(0)
	if resp.Stats != nil {
		completionTokens = resp.Stats.TokensGenerated
	}
	if completionTokens == 0 {
		completionTokens = estimateTokens(resp.Response)
	}

	return &CompletionResponse{
		ID:      completionID,
		Object:  "text_completion",
		Created: timestamp,
		Model:   req.Model,
		Choices: []CompletionChoice{
			{
				Text:         resp.Response,
				Index:        0,
				FinishReason: "stop",
			},
		},
		Usage: CompletionUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}
}

// ConvertToOpenAIEmbeddingResponse converts internal response to OpenAI embedding format
func ConvertToOpenAIEmbeddingResponse(req *EmbeddingRequest, resp *backends.EmbedResponse) *EmbeddingResponse {
	// Estimate tokens
	promptTokens := estimateTokens(extractPrompt(req.Input))

	return &EmbeddingResponse{
		Object: "list",
		Data: []EmbeddingData{
			{
				Object:    "embedding",
				Index:     0,
				Embedding: resp.Embedding,
			},
		},
		Model: req.Model,
		Usage: EmbeddingUsage{
			PromptTokens: promptTokens,
			TotalTokens:  promptTokens,
		},
	}
}

// buildPromptFromMessages concatenates messages into a single prompt with role markers
func buildPromptFromMessages(messages []ChatCompletionMessage) string {
	var parts []string

	for _, msg := range messages {
		var prefix string
		switch strings.ToLower(msg.Role) {
		case "system":
			prefix = "System"
		case "user":
			prefix = "User"
		case "assistant":
			prefix = "Assistant"
		default:
			prefix = msg.Role
		}

		if msg.Content != "" {
			parts = append(parts, fmt.Sprintf("%s: %s", prefix, msg.Content))
		}
	}

	// Add assistant prompt at the end if the last message wasn't from assistant
	if len(messages) > 0 && strings.ToLower(messages[len(messages)-1].Role) != "assistant" {
		parts = append(parts, "Assistant:")
	}

	return strings.Join(parts, "\n")
}

// extractPrompt extracts string prompt from interface{} (can be string or []string)
func extractPrompt(prompt interface{}) string {
	switch v := prompt.(type) {
	case string:
		return v
	case []string:
		if len(v) > 0 {
			return v[0] // Use first prompt
		}
		return ""
	case []interface{}:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return s
			}
		}
		return ""
	default:
		return fmt.Sprintf("%v", prompt)
	}
}

// generateCompletionID generates a unique completion ID
func generateCompletionID(prefix string) string {
	// Generate random bytes
	b := make([]byte, 12)
	rand.Read(b)

	// Convert to hex
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}

// estimateTokens provides a rough token count estimate
// This is a simple approximation: ~4 characters per token on average
func estimateTokens(text string) int32 {
	if text == "" {
		return 0
	}
	// Rough approximation: divide character count by 4
	return int32((len(text) + 3) / 4)
}
