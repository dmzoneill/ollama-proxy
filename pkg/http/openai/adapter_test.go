package openai

import (
	"strings"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

func TestConvertChatCompletionRequest(t *testing.T) {
	temp := float32(0.7)
	topP := float32(0.9)
	maxTokens := int32(100)

	req := &ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []ChatCompletionMessage{
			{Role: "system", Content: "You are a helpful assistant"},
			{Role: "user", Content: "Hello!"},
		},
		Temperature: &temp,
		TopP:        &topP,
		MaxTokens:   &maxTokens,
		Stop:        []string{"\n\n"},
	}

	genReq := ConvertChatCompletionRequest(req)

	if genReq.Model != "gpt-3.5-turbo" {
		t.Errorf("Expected model gpt-3.5-turbo, got %s", genReq.Model)
	}

	if genReq.Options.Temperature != 0.7 {
		t.Errorf("Expected temperature 0.7, got %.1f", genReq.Options.Temperature)
	}

	if genReq.Options.TopP != 0.9 {
		t.Errorf("Expected topP 0.9, got %.1f", genReq.Options.TopP)
	}

	if genReq.Options.MaxTokens != 100 {
		t.Errorf("Expected maxTokens 100, got %d", genReq.Options.MaxTokens)
	}

	if len(genReq.Options.Stop) != 1 || genReq.Options.Stop[0] != "\n\n" {
		t.Errorf("Expected stop [\"\n\n\"], got %v", genReq.Options.Stop)
	}
}

func TestConvertChatCompletionRequest_NoOptions(t *testing.T) {
	req := &ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "Test"},
		},
	}

	genReq := ConvertChatCompletionRequest(req)

	if genReq.Options.Temperature != 0 {
		t.Errorf("Expected default temperature 0, got %.1f", genReq.Options.Temperature)
	}
}

func TestConvertCompletionRequest_StringPrompt(t *testing.T) {
	temp := float32(0.5)
	req := &CompletionRequest{
		Model:       "text-davinci-003",
		Prompt:      "Once upon a time",
		Temperature: &temp,
	}

	genReq := ConvertCompletionRequest(req)

	if genReq.Model != "text-davinci-003" {
		t.Errorf("Expected model text-davinci-003, got %s", genReq.Model)
	}

	if genReq.Prompt != "Once upon a time" {
		t.Errorf("Expected prompt 'Once upon a time', got '%s'", genReq.Prompt)
	}
}

func TestConvertCompletionRequest_ArrayPrompt(t *testing.T) {
	prompts := []string{"First prompt", "Second prompt"}
	req := &CompletionRequest{
		Model:  "gpt-3.5-turbo",
		Prompt: prompts,
	}

	genReq := ConvertCompletionRequest(req)

	if genReq.Prompt != "First prompt" {
		t.Errorf("Expected prompt 'First prompt', got '%s'", genReq.Prompt)
	}
}

func TestConvertCompletionRequest_AllOptions(t *testing.T) {
	temp := float32(0.8)
	topP := float32(0.95)
	maxTokens := int32(200)

	req := &CompletionRequest{
		Model:       "gpt-3.5-turbo",
		Prompt:      "Test",
		Temperature: &temp,
		TopP:        &topP,
		MaxTokens:   &maxTokens,
		Stop:        []string{"END", "STOP"},
	}

	genReq := ConvertCompletionRequest(req)

	if genReq.Options.Temperature != 0.8 {
		t.Errorf("Expected temperature 0.8, got %.1f", genReq.Options.Temperature)
	}

	if len(genReq.Options.Stop) != 2 {
		t.Errorf("Expected 2 stop sequences, got %d", len(genReq.Options.Stop))
	}
}

func TestConvertEmbeddingRequest_String(t *testing.T) {
	req := &EmbeddingRequest{
		Model: "text-embedding-ada-002",
		Input: "This is a test",
	}

	embedReq := ConvertEmbeddingRequest(req)

	if embedReq.Model != "text-embedding-ada-002" {
		t.Errorf("Expected model text-embedding-ada-002, got %s", embedReq.Model)
	}

	if embedReq.Text != "This is a test" {
		t.Errorf("Expected text 'This is a test', got '%s'", embedReq.Text)
	}
}

func TestConvertEmbeddingRequest_Array(t *testing.T) {
	inputs := []string{"First text", "Second text"}
	req := &EmbeddingRequest{
		Model: "text-embedding-ada-002",
		Input: inputs,
	}

	embedReq := ConvertEmbeddingRequest(req)

	if embedReq.Text != "First text" {
		t.Errorf("Expected text 'First text', got '%s'", embedReq.Text)
	}
}

func TestConvertToOpenAIEmbeddingResponse(t *testing.T) {
	req := &EmbeddingRequest{
		Model: "text-embedding-ada-002",
		Input: "Test input",
	}

	embedResp := &backends.EmbedResponse{
		Embedding: []float32{0.1, 0.2, 0.3, 0.4},
	}

	openaiResp := ConvertToOpenAIEmbeddingResponse(req, embedResp)

	if openaiResp.Object != "list" {
		t.Errorf("Expected object list, got %s", openaiResp.Object)
	}

	if openaiResp.Model != "text-embedding-ada-002" {
		t.Errorf("Expected model text-embedding-ada-002, got %s", openaiResp.Model)
	}

	if len(openaiResp.Data) != 1 {
		t.Fatalf("Expected 1 embedding, got %d", len(openaiResp.Data))
	}

	if len(openaiResp.Data[0].Embedding) != 4 {
		t.Errorf("Expected embedding length 4, got %d", len(openaiResp.Data[0].Embedding))
	}
}

func TestBuildPromptFromMessages(t *testing.T) {
	messages := []ChatCompletionMessage{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "Hello"},
	}

	result := buildPromptFromMessages(messages)

	if result == "" {
		t.Error("Expected non-empty prompt")
	}

	// Should contain system message
	if len(result) < 10 {
		t.Error("Expected substantial prompt")
	}
}

func TestBuildPromptFromMessages_EndsWithAssistant(t *testing.T) {
	messages := []ChatCompletionMessage{
		{Role: "user", Content: "Hello"},
	}

	prompt := buildPromptFromMessages(messages)

	// Should add "Assistant:" at the end
	expectedSuffix := "Assistant:"
	if len(prompt) < len(expectedSuffix) || prompt[len(prompt)-len(expectedSuffix):] != expectedSuffix {
		t.Errorf("Expected prompt to end with '%s', got '%s'", expectedSuffix, prompt)
	}
}

func TestBuildPromptFromMessages_EmptyContent(t *testing.T) {
	messages := []ChatCompletionMessage{
		{Role: "user", Content: ""},
		{Role: "assistant", Content: "Response"},
	}

	result := buildPromptFromMessages(messages)

	// Should skip empty messages
	if len(result) == 0 {
		t.Error("Expected non-empty result")
	}
}

func TestBuildPromptFromMessages_CustomRole(t *testing.T) {
	messages := []ChatCompletionMessage{
		{Role: "system", Content: "You are helpful"},
		{Role: "tool", Content: "Tool output data"},
		{Role: "user", Content: "What is this?"},
	}

	result := buildPromptFromMessages(messages)

	// Should use the custom role name as prefix
	if !strings.Contains(result, "tool: Tool output data") {
		t.Errorf("Expected custom role 'tool' to be used as prefix, got: %s", result)
	}

	// Should also have standard roles
	if !strings.Contains(result, "System: You are helpful") {
		t.Error("Expected system message")
	}

	// Should end with Assistant: prompt
	if !strings.HasSuffix(result, "Assistant:") {
		t.Error("Expected to end with 'Assistant:'")
	}
}

func TestExtractPrompt_String(t *testing.T) {
	result := extractPrompt("Simple string")

	if result != "Simple string" {
		t.Errorf("Expected 'Simple string', got '%s'", result)
	}
}

func TestExtractPrompt_StringArray(t *testing.T) {
	prompts := []string{"First", "Second", "Third"}
	result := extractPrompt(prompts)

	if result != "First" {
		t.Errorf("Expected 'First', got '%s'", result)
	}
}

func TestExtractPrompt_EmptyStringArray(t *testing.T) {
	prompts := []string{}
	result := extractPrompt(prompts)

	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

func TestExtractPrompt_InterfaceArray(t *testing.T) {
	prompts := []interface{}{"Interface string", 123}
	result := extractPrompt(prompts)

	if result != "Interface string" {
		t.Errorf("Expected 'Interface string', got '%s'", result)
	}
}

func TestExtractPrompt_EmptyInterfaceArray(t *testing.T) {
	prompts := []interface{}{}
	result := extractPrompt(prompts)

	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

func TestExtractPrompt_InterfaceArrayNonString(t *testing.T) {
	prompts := []interface{}{123, 456}
	result := extractPrompt(prompts)

	if result != "" {
		t.Errorf("Expected empty string for non-string interface array, got '%s'", result)
	}
}

func TestExtractPrompt_OtherType(t *testing.T) {
	result := extractPrompt(12345)

	if result != "12345" {
		t.Errorf("Expected '12345', got '%s'", result)
	}
}

func TestGenerateCompletionID(t *testing.T) {
	id1 := generateCompletionID("chatcmpl")
	id2 := generateCompletionID("chatcmpl")

	if id1 == id2 {
		t.Error("Expected different IDs for multiple calls")
	}

	// Check prefix
	prefix := "chatcmpl-"
	if len(id1) < len(prefix) || id1[:len(prefix)] != prefix {
		t.Errorf("Expected ID to start with '%s', got %s", prefix, id1)
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text     string
		expected int32
	}{
		{"", 0},
		{"a", 1},
		{"test", 1},
		{"hello", 2},
		{"hello world", 3},
		{"This is a longer text with more characters", 11},
	}

	for _, tt := range tests {
		result := estimateTokens(tt.text)
		if result != tt.expected {
			t.Errorf("estimateTokens(%q) = %d, want %d", tt.text, result, tt.expected)
		}
	}
}

func TestConvertToOpenAIChatResponse_TokenEstimation(t *testing.T) {
	// Test when TokensGenerated is 0, should estimate tokens
	req := &ChatCompletionRequest{
		Model: "test-model",
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "test"},
		},
	}

	resp := &backends.GenerateResponse{
		Response: "Hello, this is a test response with multiple words",
		Stats: &backends.GenerationStats{
			TokensGenerated: 0, // Zero tokens, should trigger estimation
			TotalTimeMs:     100,
		},
	}

	result := ConvertToOpenAIChatResponse(req, resp)

	if result.Usage.CompletionTokens == 0 {
		t.Error("Expected CompletionTokens to be estimated when TokensGenerated is 0")
	}

	// Should have estimated based on response text
	expectedTokens := estimateTokens(resp.Response)
	if result.Usage.CompletionTokens != expectedTokens {
		t.Errorf("CompletionTokens = %d, want %d", result.Usage.CompletionTokens, expectedTokens)
	}
}

func TestConvertToOpenAICompletionResponse_TokenEstimation(t *testing.T) {
	// Test when TokensGenerated is 0, should estimate tokens
	req := &CompletionRequest{
		Model:  "test-model",
		Prompt: "test prompt",
	}

	resp := &backends.GenerateResponse{
		Response: "This is another test response for completion",
		Stats: &backends.GenerationStats{
			TokensGenerated: 0, // Zero tokens, should trigger estimation
			TotalTimeMs:     150,
		},
	}

	result := ConvertToOpenAICompletionResponse(req, resp)

	if result.Usage.CompletionTokens == 0 {
		t.Error("Expected CompletionTokens to be estimated when TokensGenerated is 0")
	}

	// Should have estimated based on response text
	expectedTokens := estimateTokens(resp.Response)
	if result.Usage.CompletionTokens != expectedTokens {
		t.Errorf("CompletionTokens = %d, want %d", result.Usage.CompletionTokens, expectedTokens)
	}
}
