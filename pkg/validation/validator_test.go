package validation

import (
	"strings"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// TestValidateModelName is part of the public API.
func TestValidateModelName(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		wantErr bool
	}{
		{name: "valid simple", model: "llama3", wantErr: false},
		{name: "valid with version", model: strLlama37b, wantErr: false},
		{name: "valid with dot", model: "qwen2.5:0.5b", wantErr: false},
		{name: "valid with dash", model: "llama-3-8b", wantErr: false},
		{name: "valid with underscore", model: "model_name", wantErr: false},
		{name: "empty", model: "", wantErr: true},
		{name: "too long", model: strings.Repeat("a", MaxModelNameLength+1), wantErr: true},
		{name: "invalid chars space", model: "model name", wantErr: true},
		{name: "invalid chars slash", model: "model/name", wantErr: true},
		{name: "invalid chars special", model: "model@name", wantErr: true},
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

// TestValidatePrompt is part of the public API.
func TestValidatePrompt(t *testing.T) {
	tests := []struct {
		name    string
		prompt  string
		wantErr bool
	}{
		{name: "valid short", prompt: strHello, wantErr: false},
		{name: "valid long", prompt: strings.Repeat("a", 1000), wantErr: false},
		{name: "valid max", prompt: strings.Repeat("a", MaxPromptLength), wantErr: false},
		{name: "empty", prompt: "", wantErr: true},
		{name: "too long", prompt: strings.Repeat("a", MaxPromptLength+1), wantErr: true},
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

// TestValidateGenerationOptions is part of the public API.
func TestValidateGenerationOptions(t *testing.T) {
	tests := []struct {
		opts    *backends.GenerationOptions
		name    string
		wantErr bool
	}{
		{name: "nil options", opts: nil, wantErr: false},
		{name: "valid defaults", opts: &backends.GenerationOptions{
			Temperature: 0.7,
			TopP:        0.9,
			TopK:        40,
			MaxTokens:   1000,
		}, wantErr: false},
		{name: "temperature too low", opts: &backends.GenerationOptions{
			Temperature: -0.1,
		}, wantErr: true},
		{name: "temperature too high", opts: &backends.GenerationOptions{
			Temperature: 2.1,
		}, wantErr: true},
		{name: "topP too low", opts: &backends.GenerationOptions{
			TopP: -0.1,
		}, wantErr: true},
		{name: "topP too high", opts: &backends.GenerationOptions{
			TopP: 1.1,
		}, wantErr: true},
		{name: "topK negative", opts: &backends.GenerationOptions{
			TopK: -1,
		}, wantErr: true},
		{name: "topK too high", opts: &backends.GenerationOptions{
			TopK: MaxTopK + 1,
		}, wantErr: true},
		{name: "maxTokens negative", opts: &backends.GenerationOptions{
			MaxTokens: -1,
		}, wantErr: true},
		{name: "maxTokens too high", opts: &backends.GenerationOptions{
			MaxTokens: MaxMaxTokens + 1,
		}, wantErr: true},
		{name: "contextLength negative", opts: &backends.GenerationOptions{
			ContextLength: -1,
		}, wantErr: true},
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

// TestValidateBackendID is part of the public API.
func TestValidateBackendID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{name: "empty is valid", id: "", wantErr: false},
		{name: "auto is valid", id: "auto", wantErr: false},
		{name: "valid simple", id: "ollama-npu", wantErr: false},
		{name: "valid with underscore", id: "ollama_npu", wantErr: false},
		{name: "valid alphanumeric", id: "backend123", wantErr: false},
		{name: "invalid with space", id: "backend id", wantErr: true},
		{name: "invalid with dot", id: "backend.id", wantErr: true},
		{name: "invalid with special char", id: "backend@id", wantErr: true},
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

// TestValidateGenerateRequest is part of the public API.
func TestValidateGenerateRequest(t *testing.T) {
	tests := []struct {
		req     *backends.GenerateRequest
		name    string
		wantErr bool
	}{
		{name: "nil request", req: nil, wantErr: true},
		{name: "valid request", req: &backends.GenerateRequest{
			Prompt: strHello,
			Model:  strLlama37b,
			Options: &backends.GenerationOptions{
				Temperature: 0.7,
				TopP:        0.9,
			},
		}, wantErr: false},
		{name: "empty prompt", req: &backends.GenerateRequest{
			Prompt: "",
			Model:  strLlama37b,
		}, wantErr: true},
		{name: strInvalidModel, req: &backends.GenerateRequest{
			Prompt: strHello,
			Model:  "invalid model name",
		}, wantErr: true},
		{name: "invalid options", req: &backends.GenerateRequest{
			Prompt: strHello,
			Model:  strLlama37b,
			Options: &backends.GenerationOptions{
				Temperature: 3.0, // Invalid
			},
		}, wantErr: true},
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

// TestValidateEmbedRequest is part of the public API.
func TestValidateEmbedRequest(t *testing.T) {
	tests := []struct {
		req     *backends.EmbedRequest
		name    string
		wantErr bool
	}{
		{name: "nil request", req: nil, wantErr: true},
		{name: "valid request", req: &backends.EmbedRequest{
			Text:  "Hello world",
			Model: strNomicEmbed,
		}, wantErr: false},
		{name: "empty text", req: &backends.EmbedRequest{
			Text:  "",
			Model: strNomicEmbed,
		}, wantErr: true},
		{name: "text too long", req: &backends.EmbedRequest{
			Text:  strings.Repeat("a", MaxPromptLength+1),
			Model: strNomicEmbed,
		}, wantErr: true},
		{name: strInvalidModel, req: &backends.EmbedRequest{
			Text:  strHello,
			Model: strInvalidModel,
		}, wantErr: true},
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

// TestValidateAnnotations is part of the public API.
func TestValidateAnnotations(t *testing.T) {
	tests := []struct {
		ann     *backends.Annotations
		name    string
		wantErr bool
	}{
		{name: "nil annotations", ann: nil, wantErr: false},
		{name: "valid annotations", ann: &backends.Annotations{
			Target:                "ollama-npu",
			LatencyCritical:       true,
			PreferPowerEfficiency: false,
			MaxLatencyMs:          500,
			MaxPowerWatts:         15,
			Priority:              backends.PriorityHigh,
			MediaType:             backends.MediaTypeText,
		}, wantErr: false},
		{name: "invalid target", ann: &backends.Annotations{
			Target: "invalid target",
		}, wantErr: true},
		{name: "negative max latency", ann: &backends.Annotations{
			MaxLatencyMs: -1,
		}, wantErr: true},
		{name: "negative max power", ann: &backends.Annotations{
			MaxPowerWatts: -1,
		}, wantErr: true},
		{name: "negative deadline", ann: &backends.Annotations{
			DeadlineMs: -1,
		}, wantErr: true},
		{name: "invalid priority too low", ann: &backends.Annotations{
			Priority: backends.Priority(-1),
		}, wantErr: true},
		{name: "invalid priority too high", ann: &backends.Annotations{
			Priority: backends.Priority(4),
		}, wantErr: true},
		{name: "invalid media type", ann: &backends.Annotations{
			MediaType: backends.MediaType("invalid"),
		}, wantErr: true},
		{name: "empty media type is valid", ann: &backends.Annotations{
			MediaType: "",
		}, wantErr: false},
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
