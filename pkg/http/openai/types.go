package openai

// OpenAI API compatible request/response types

// ChatCompletionRequest represents a request to /v1/chat/completions
type ChatCompletionRequest struct {
	Temperature      *float32                `json:"temperature,omitempty"`
	TopP             *float32                `json:"top_p,omitempty"`
	N                *int                    `json:"n,omitempty"`
	MaxTokens        *int32                  `json:"max_tokens,omitempty"`
	PresencePenalty  *float32                `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float32                `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]float32      `json:"logit_bias,omitempty"`
	Model            string                  `json:"model"`
	User             string                  `json:"user,omitempty"`
	Messages         []ChatCompletionMessage `json:"messages"`
	Stop             []string                `json:"stop,omitempty"`
	Stream           bool                    `json:"stream,omitempty"`
}

// ChatCompletionMessage represents a message in the chat
type ChatCompletionMessage struct {
	Role    string `json:"role"` // system, user, assistant, function
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// ChatCompletionResponse represents a response from /v1/chat/completions
type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Created int64                  `json:"created"`
	Usage   ChatCompletionUsage    `json:"usage"`
}

// ChatCompletionChoice represents a completion choice
type ChatCompletionChoice struct {
	Message      ChatCompletionMessage `json:"message"`
	FinishReason string                `json:"finish_reason"`
	Index        int                   `json:"index"`
}

// ChatCompletionUsage represents token usage statistics
type ChatCompletionUsage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	TotalTokens      int32 `json:"total_tokens"`
}

// ChatCompletionChunk represents a streaming chunk from /v1/chat/completions
type ChatCompletionChunk struct {
	ID      string                      `json:"id"`
	Object  string                      `json:"object"`
	Model   string                      `json:"model"`
	Choices []ChatCompletionChunkChoice `json:"choices"`
	Created int64                       `json:"created"`
}

// ChatCompletionChunkChoice represents a streaming choice
type ChatCompletionChunkChoice struct {
	FinishReason *string                  `json:"finish_reason"`
	Delta        ChatCompletionChunkDelta `json:"delta"`
	Index        int                      `json:"index"`
}

// ChatCompletionChunkDelta represents the incremental content in a chunk
type ChatCompletionChunkDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// CompletionRequest represents a request to /v1/completions (legacy)
type CompletionRequest struct {
	Prompt           interface{}        `json:"prompt"`
	LogProbs         *int               `json:"logprobs,omitempty"`
	PresencePenalty  *float32           `json:"presence_penalty,omitempty"`
	MaxTokens        *int32             `json:"max_tokens,omitempty"`
	Temperature      *float32           `json:"temperature,omitempty"`
	TopP             *float32           `json:"top_p,omitempty"`
	N                *int               `json:"n,omitempty"`
	LogitBias        map[string]float32 `json:"logit_bias,omitempty"`
	BestOf           *int               `json:"best_of,omitempty"`
	FrequencyPenalty *float32           `json:"frequency_penalty,omitempty"`
	Model            string             `json:"model"`
	Suffix           string             `json:"suffix,omitempty"`
	User             string             `json:"user,omitempty"`
	Stop             []string           `json:"stop,omitempty"`
	Stream           bool               `json:"stream,omitempty"`
	Echo             bool               `json:"echo,omitempty"`
}

// CompletionResponse represents a response from /v1/completions
type CompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Model   string             `json:"model"`
	Choices []CompletionChoice `json:"choices"`
	Created int64              `json:"created"`
	Usage   CompletionUsage    `json:"usage"`
}

// CompletionChoice represents a completion choice
type CompletionChoice struct {
	LogProbs     *int   `json:"logprobs"`
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason"`
	Index        int    `json:"index"`
}

// CompletionUsage represents token usage statistics
type CompletionUsage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	TotalTokens      int32 `json:"total_tokens"`
}

// CompletionChunk represents a streaming chunk from /v1/completions
type CompletionChunk struct {
	ID      string                  `json:"id"`
	Object  string                  `json:"object"`
	Model   string                  `json:"model"`
	Choices []CompletionChunkChoice `json:"choices"`
	Created int64                   `json:"created"`
}

// CompletionChunkChoice represents a streaming choice
type CompletionChunkChoice struct {
	FinishReason *string `json:"finish_reason"`
	Text         string  `json:"text"`
	Index        int     `json:"index"`
}

// EmbeddingRequest represents a request to /v1/embeddings
type EmbeddingRequest struct {
	Model          string      `json:"model"`
	Input          interface{} `json:"input"` // string or []string
	User           string      `json:"user,omitempty"`
	EncodingFormat string      `json:"encoding_format,omitempty"` // "float" or "base64"
}

// EmbeddingResponse represents a response from /v1/embeddings
type EmbeddingResponse struct {
	Object string          `json:"object"`
	Model  string          `json:"model"`
	Data   []EmbeddingData `json:"data"`
	Usage  EmbeddingUsage  `json:"usage"`
}

// EmbeddingData represents a single embedding
type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

// EmbeddingUsage represents token usage for embeddings
type EmbeddingUsage struct {
	PromptTokens int32 `json:"prompt_tokens"`
	TotalTokens  int32 `json:"total_tokens"`
}

// ModelsResponse represents a response from /v1/models
type ModelsResponse struct {
	Object string  `json:"object"` // "list"
	Data   []Model `json:"data"`
}

// Model represents a model in the models list
type Model struct {
	ID         string   `json:"id"`
	Object     string   `json:"object"`
	OwnedBy    string   `json:"owned_by"`
	Root       string   `json:"root,omitempty"`
	Parent     string   `json:"parent,omitempty"`
	Permission []string `json:"permission,omitempty"`
	Created    int64    `json:"created"`
}

// ErrorResponse represents an OpenAI API error
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents the error details
type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}
