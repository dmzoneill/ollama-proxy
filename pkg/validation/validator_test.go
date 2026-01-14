package validation

import (
	"strings"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

func TestValidateModelName(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		wantErr bool
	}{
		{"valid simple", "llama3", false},
		{"valid with version", "llama3:7b", false},
		{"valid with dot", "qwen2.5:0.5b", false},
		{"valid with dash", "llama-3-8b", false},
		{"valid with underscore", "model_name", false},
		{"empty", "", true},
		{"too long", strings.Repeat("a", MaxModelNameLength+1), true},
		{"invalid chars space", "model name", true},
		{"invalid chars slash", "model/name", true},
		{"invalid chars special", "model@name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModelName(tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateModelName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePrompt(t *testing.T) {
	tests := []struct {
		name    string
		prompt  string
		wantErr bool
	}{
		{"valid short", "Hello", false},
		{"valid long", strings.Repeat("a", 1000), false},
		{"valid max", strings.Repeat("a", MaxPromptLength), false},
		{"empty", "", true},
		{"too long", strings.Repeat("a", MaxPromptLength+1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePrompt(tt.prompt)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePrompt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateGenerationOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    *backends.GenerationOptions
		wantErr bool
	}{
		{"nil options", nil, false},
		{"valid defaults", &backends.GenerationOptions{
			Temperature: 0.7,
			TopP:        0.9,
			TopK:        40,
			MaxTokens:   1000,
		}, false},
		{"temperature too low", &backends.GenerationOptions{
			Temperature: -0.1,
		}, true},
		{"temperature too high", &backends.GenerationOptions{
			Temperature: 2.1,
		}, true},
		{"topP too low", &backends.GenerationOptions{
			TopP: -0.1,
		}, true},
		{"topP too high", &backends.GenerationOptions{
			TopP: 1.1,
		}, true},
		{"topK negative", &backends.GenerationOptions{
			TopK: -1,
		}, true},
		{"topK too high", &backends.GenerationOptions{
			TopK: MaxTopK + 1,
		}, true},
		{"maxTokens negative", &backends.GenerationOptions{
			MaxTokens: -1,
		}, true},
		{"maxTokens too high", &backends.GenerationOptions{
			MaxTokens: MaxMaxTokens + 1,
		}, true},
		{"contextLength negative", &backends.GenerationOptions{
			ContextLength: -1,
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGenerationOptions(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGenerationOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateBackendID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"empty is valid", "", false},
		{"auto is valid", "auto", false},
		{"valid simple", "ollama-npu", false},
		{"valid with underscore", "ollama_npu", false},
		{"valid alphanumeric", "backend123", false},
		{"invalid with space", "backend id", true},
		{"invalid with dot", "backend.id", true},
		{"invalid with special char", "backend@id", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBackendID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBackendID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateGenerateRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *backends.GenerateRequest
		wantErr bool
	}{
		{"nil request", nil, true},
		{"valid request", &backends.GenerateRequest{
			Prompt: "Hello",
			Model:  "llama3:7b",
			Options: &backends.GenerationOptions{
				Temperature: 0.7,
				TopP:        0.9,
			},
		}, false},
		{"empty prompt", &backends.GenerateRequest{
			Prompt: "",
			Model:  "llama3:7b",
		}, true},
		{"invalid model", &backends.GenerateRequest{
			Prompt: "Hello",
			Model:  "invalid model name",
		}, true},
		{"invalid options", &backends.GenerateRequest{
			Prompt: "Hello",
			Model:  "llama3:7b",
			Options: &backends.GenerationOptions{
				Temperature: 3.0, // Invalid
			},
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGenerateRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGenerateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateEmbedRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *backends.EmbedRequest
		wantErr bool
	}{
		{"nil request", nil, true},
		{"valid request", &backends.EmbedRequest{
			Text:  "Hello world",
			Model: "nomic-embed",
		}, false},
		{"empty text", &backends.EmbedRequest{
			Text:  "",
			Model: "nomic-embed",
		}, true},
		{"text too long", &backends.EmbedRequest{
			Text:  strings.Repeat("a", MaxPromptLength+1),
			Model: "nomic-embed",
		}, true},
		{"invalid model", &backends.EmbedRequest{
			Text:  "Hello",
			Model: "invalid model",
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmbedRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmbedRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAnnotations(t *testing.T) {
	tests := []struct {
		name    string
		ann     *backends.Annotations
		wantErr bool
	}{
		{"nil annotations", nil, false},
		{"valid annotations", &backends.Annotations{
			Target:                "ollama-npu",
			LatencyCritical:       true,
			PreferPowerEfficiency: false,
			MaxLatencyMs:          500,
			MaxPowerWatts:         15,
			Priority:              backends.PriorityHigh,
			MediaType:             backends.MediaTypeText,
		}, false},
		{"invalid target", &backends.Annotations{
			Target: "invalid target",
		}, true},
		{"negative max latency", &backends.Annotations{
			MaxLatencyMs: -1,
		}, true},
		{"negative max power", &backends.Annotations{
			MaxPowerWatts: -1,
		}, true},
		{"negative deadline", &backends.Annotations{
			DeadlineMs: -1,
		}, true},
		{"invalid priority too low", &backends.Annotations{
			Priority: backends.Priority(-1),
		}, true},
		{"invalid priority too high", &backends.Annotations{
			Priority: backends.Priority(4),
		}, true},
		{"invalid media type", &backends.Annotations{
			MediaType: backends.MediaType("invalid"),
		}, true},
		{"empty media type is valid", &backends.Annotations{
			MediaType: "",
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAnnotations(tt.ann)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAnnotations() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
