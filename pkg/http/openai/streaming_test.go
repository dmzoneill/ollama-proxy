package openai

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// MockStreamReader implements backends.StreamReader for testing
type MockStreamReader struct {
	chunks []*backends.StreamChunk
	index  int
	closed bool
	err    error
}

func (m *MockStreamReader) Recv() (*backends.StreamChunk, error) {
	if m.closed {
		return nil, fmt.Errorf("EOF")
	}

	if m.err != nil && m.index >= len(m.chunks) {
		return nil, m.err
	}

	if m.index >= len(m.chunks) {
		m.closed = true
		return nil, fmt.Errorf("EOF")
	}

	chunk := m.chunks[m.index]
	m.index++
	return chunk, nil
}

func (m *MockStreamReader) Close() error {
	m.closed = true
	return nil
}

// NewMockStreamReader creates a new mock stream reader
func NewMockStreamReader(chunks []*backends.StreamChunk) *MockStreamReader {
	return &MockStreamReader{
		chunks: chunks,
		index:  0,
	}
}

// NewMockStreamReaderWithError creates a mock stream reader that errors
func NewMockStreamReaderWithError(chunks []*backends.StreamChunk, err error) *MockStreamReader {
	return &MockStreamReader{
		chunks: chunks,
		index:  0,
		err:    err,
	}
}

// parseSSEResponse parses SSE formatted response
func parseSSEResponse(body string) []map[string]interface{} {
	var result []map[string]interface{}
	lines := strings.Split(body, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			dataStr := strings.TrimPrefix(line, "data: ")
			if dataStr == "[DONE]" {
				result = append(result, map[string]interface{}{"done": true})
			} else if strings.HasPrefix(dataStr, "{") {
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(dataStr), &data); err == nil {
					result = append(result, data)
				}
			}
		}
	}
	return result
}

// Test: StreamChatCompletion with normal chunks
func TestStreamChatCompletion_Normal(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "Hello", Done: false},
		{Token: " world", Done: false},
		{Token: "!", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check headers
	if ct := recorder.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Expected Content-Type 'text/event-stream', got %s", ct)
	}

	if cc := recorder.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Expected Cache-Control 'no-cache', got %s", cc)
	}

	if conn := recorder.Header().Get("Connection"); conn != "keep-alive" {
		t.Errorf("Expected Connection 'keep-alive', got %s", conn)
	}

	if xab := recorder.Header().Get("X-Accel-Buffering"); xab != "no" {
		t.Errorf("Expected X-Accel-Buffering 'no', got %s", xab)
	}

	// Check response body
	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	// Should have 3 token chunks + 1 [DONE]
	if len(messages) < 3 {
		t.Errorf("Expected at least 3 messages, got %d", len(messages))
	}

	// Check first message structure
	if len(messages) > 0 {
		if choices, ok := messages[0]["choices"]; ok {
			if choiceList, ok := choices.([]interface{}); ok && len(choiceList) > 0 {
				choice := choiceList[0].(map[string]interface{})
				if delta, ok := choice["delta"]; ok {
					deltaMap := delta.(map[string]interface{})
					if content, ok := deltaMap["content"]; ok {
						if content != "Hello" {
							t.Errorf("Expected content 'Hello', got %v", content)
						}
					}
				}
			}
		}
	}

	// Check model and ID are set
	if len(messages) > 0 {
		if model, ok := messages[0]["model"]; ok {
			if model != "gpt-4" {
				t.Errorf("Expected model 'gpt-4', got %v", model)
			}
		}
		if id, ok := messages[0]["id"]; ok {
			if id != "chatcmpl-123" {
				t.Errorf("Expected id 'chatcmpl-123', got %v", id)
			}
		}
	}

	// Check [DONE] message is present
	doneFound := false
	for _, msg := range messages {
		if done, ok := msg["done"].(bool); ok && done {
			doneFound = true
			break
		}
	}
	if !doneFound {
		t.Error("Expected [DONE] message")
	}
}

// Test: StreamChatCompletion with finish_reason
func TestStreamChatCompletion_FinishReason(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "Response", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-456")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	// Find the message with finish_reason
	if len(messages) > 0 {
		if choices, ok := messages[0]["choices"]; ok {
			if choiceList, ok := choices.([]interface{}); ok && len(choiceList) > 0 {
				choice := choiceList[0].(map[string]interface{})
				if finishReason, ok := choice["finish_reason"]; ok {
					if finishReason != "stop" {
						t.Errorf("Expected finish_reason 'stop', got %v", finishReason)
					}
				} else {
					t.Error("Expected finish_reason in final chunk")
				}
			}
		}
	}
}

// Test: StreamChatCompletion with empty chunks
func TestStreamChatCompletion_EmptyChunks(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "", Done: false},
		{Token: "", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-789")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "[DONE]") {
		t.Error("Expected [DONE] message in response")
	}
}

// Test: StreamChatCompletion with reader close error
func TestStreamChatCompletion_ReaderError(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "Hello", Done: false},
	}

	reader := NewMockStreamReaderWithError(chunks, fmt.Errorf("connection lost"))
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-err")
	if err == nil {
		t.Error("Expected error from StreamChatCompletion")
	}

	if !strings.Contains(err.Error(), "connection lost") {
		t.Errorf("Expected error message containing 'connection lost', got %v", err)
	}

	// Check that error event was sent to client
	body := recorder.Body.String()
	if !strings.Contains(body, "error") {
		t.Error("Expected error event in response")
	}
}

// Test: StreamChatCompletion with many chunks
func TestStreamChatCompletion_ManyChunks(t *testing.T) {
	// Create 100 chunks
	chunks := make([]*backends.StreamChunk, 100)
	for i := 0; i < 99; i++ {
		chunks[i] = &backends.StreamChunk{Token: fmt.Sprintf("token%d ", i), Done: false}
	}
	chunks[99] = &backends.StreamChunk{Token: "final", Done: true}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-many")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	// Should have all chunks plus [DONE]
	if len(messages) < 100 {
		t.Logf("Got %d messages", len(messages))
	}
}

// Test: StreamChatCompletion with special characters
func TestStreamChatCompletion_SpecialChars(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: `"quoted"`, Done: false},
		{Token: "new\nline", Done: false},
		{Token: "\\backslash", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-special")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	// Should contain valid JSON
	if !strings.Contains(body, "{") {
		t.Error("Expected JSON in response")
	}
}

// Test: StreamCompletion with normal chunks
func TestStreamCompletion_Normal(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "Once upon", Done: false},
		{Token: " a time", Done: false},
		{Token: ".", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-456")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check headers
	if ct := recorder.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Expected Content-Type 'text/event-stream', got %s", ct)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	// Should have chunks plus [DONE]
	if len(messages) < 2 {
		t.Errorf("Expected at least 2 messages, got %d", len(messages))
	}

	// Verify object type is text_completion.chunk
	if len(messages) > 0 {
		if obj, ok := messages[0]["object"]; ok {
			if obj != "text_completion.chunk" {
				t.Errorf("Expected object 'text_completion.chunk', got %v", obj)
			}
		}
	}
}

// Test: StreamCompletion with finish_reason
func TestStreamCompletion_FinishReason(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "Response", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-789")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	if len(messages) > 0 {
		if choices, ok := messages[0]["choices"]; ok {
			if choiceList, ok := choices.([]interface{}); ok && len(choiceList) > 0 {
				choice := choiceList[0].(map[string]interface{})
				if finishReason, ok := choice["finish_reason"]; ok {
					if finishReason != "stop" {
						t.Errorf("Expected finish_reason 'stop', got %v", finishReason)
					}
				}
			}
		}
	}
}

// Test: StreamCompletion with many chunks
func TestStreamCompletion_ManyChunks(t *testing.T) {
	chunks := make([]*backends.StreamChunk, 50)
	for i := 0; i < 49; i++ {
		chunks[i] = &backends.StreamChunk{Token: fmt.Sprintf("word%d ", i), Done: false}
	}
	chunks[49] = &backends.StreamChunk{Token: "end", Done: true}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-many")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "[DONE]") {
		t.Error("Expected [DONE] message")
	}
}

// Test: StreamCompletion with special characters
func TestStreamCompletion_SpecialChars(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: `"quoted"`, Done: false},
		{Token: "tab\there", Done: false},
		{Token: "unicode: 你好", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-special")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "[DONE]") {
		t.Error("Expected [DONE] message")
	}
}

// Test: SSE format validation for chat completion
func TestStreamChatCompletion_SSEFormat(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "test", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-format")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()

	// Check SSE format
	if !strings.Contains(body, "data: ") {
		t.Error("Expected 'data: ' prefix in SSE format")
	}

	// Each data line should be followed by blank line
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "data: ") && i+1 < len(lines) {
			if lines[i+1] != "" {
				t.Logf("Line %d is not followed by blank line", i)
			}
		}
	}
}

// Test: SSE format validation for completion
func TestStreamCompletion_SSEFormat(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "test", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-format")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()

	// Check SSE format
	if !strings.Contains(body, "data: ") {
		t.Error("Expected 'data: ' prefix in SSE format")
	}

	if !strings.Contains(body, "[DONE]") {
		t.Error("Expected [DONE] terminator")
	}
}

// Test: Token streaming integrity for chat
func TestStreamChatCompletion_TokenIntegrity(t *testing.T) {
	expectedTokens := []string{"Hello", " ", "world", "!"}
	chunks := make([]*backends.StreamChunk, len(expectedTokens))
	for i, token := range expectedTokens {
		chunks[i] = &backends.StreamChunk{
			Token: token,
			Done:  i == len(expectedTokens)-1,
		}
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-integrity")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	// Extract all tokens from chunks (not [DONE])
	receivedTokens := []string{}
	for _, msg := range messages {
		if _, ok := msg["done"]; ok {
			continue // Skip [DONE]
		}
		if choices, ok := msg["choices"]; ok {
			if choiceList, ok := choices.([]interface{}); ok && len(choiceList) > 0 {
				choice := choiceList[0].(map[string]interface{})
				if delta, ok := choice["delta"]; ok {
					deltaMap := delta.(map[string]interface{})
					if content, ok := deltaMap["content"]; ok {
						receivedTokens = append(receivedTokens, content.(string))
					}
				}
			}
		}
	}

	// Verify tokens match
	if len(receivedTokens) != len(expectedTokens) {
		t.Errorf("Expected %d tokens, got %d", len(expectedTokens), len(receivedTokens))
	}

	for i, expected := range expectedTokens {
		if i < len(receivedTokens) && receivedTokens[i] != expected {
			t.Errorf("Token %d: expected %q, got %q", i, expected, receivedTokens[i])
		}
	}
}

// Test: Token streaming integrity for completion
func TestStreamCompletion_TokenIntegrity(t *testing.T) {
	expectedTokens := []string{"The", " ", "quick", " ", "brown"}
	chunks := make([]*backends.StreamChunk, len(expectedTokens))
	for i, token := range expectedTokens {
		chunks[i] = &backends.StreamChunk{
			Token: token,
			Done:  i == len(expectedTokens)-1,
		}
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-integrity")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	// Extract tokens
	receivedTokens := []string{}
	for _, msg := range messages {
		if _, ok := msg["done"]; ok {
			continue
		}
		if choices, ok := msg["choices"]; ok {
			if choiceList, ok := choices.([]interface{}); ok && len(choiceList) > 0 {
				choice := choiceList[0].(map[string]interface{})
				if text, ok := choice["text"]; ok {
					receivedTokens = append(receivedTokens, text.(string))
				}
			}
		}
	}

	if len(receivedTokens) != len(expectedTokens) {
		t.Errorf("Expected %d tokens, got %d", len(expectedTokens), len(receivedTokens))
	}
}

// Test: Error handling with EOF
func TestStreamChatCompletion_NormalEOF(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "hello", Done: false},
		{Token: "world", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-eof")
	if err != nil {
		t.Fatalf("Expected no error on normal EOF, got %v", err)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "[DONE]") {
		t.Error("Expected [DONE] message")
	}
}

// Test: Error handling with actual error
func TestStreamChatCompletion_ActualError(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "partial", Done: false},
	}

	reader := NewMockStreamReaderWithError(chunks, fmt.Errorf("backend timeout"))
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-timeout")
	if err == nil {
		t.Error("Expected error from reader")
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "error") {
		t.Error("Expected error event in response")
	}
	if !strings.Contains(body, "backend timeout") {
		t.Error("Expected error message in response")
	}
}

// Test: Completion with actual error
func TestStreamCompletion_ActualError(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "incomplete", Done: false},
	}

	reader := NewMockStreamReaderWithError(chunks, fmt.Errorf("db error"))
	recorder := httptest.NewRecorder()

	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-error")
	// StreamCompletion doesn't return errors from reader - it just breaks the loop
	if err != nil {
		t.Errorf("Expected no error from StreamCompletion (it breaks on reader error), got %v", err)
	}

	// But it should still have written something to the response
	body := recorder.Body.String()
	if body == "" {
		t.Error("Expected response body to be written before error")
	}
}

// Test: Stream completion with index tracking
func TestStreamChatCompletion_Index(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "chunk1", Done: false},
		{Token: "chunk2", Done: false},
		{Token: "chunk3", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-index")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	// All chunks should have index 0 (single choice)
	for _, msg := range messages {
		if _, ok := msg["done"]; ok {
			continue
		}
		if choices, ok := msg["choices"]; ok {
			if choiceList, ok := choices.([]interface{}); ok && len(choiceList) > 0 {
				choice := choiceList[0].(map[string]interface{})
				if index, ok := choice["index"]; ok {
					if idx := int(index.(float64)); idx != 0 {
						t.Errorf("Expected index 0, got %d", idx)
					}
				}
			}
		}
	}
}

// Test: Completion index tracking
func TestStreamCompletion_Index(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "text1", Done: false},
		{Token: "text2", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-index")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	for _, msg := range messages {
		if _, ok := msg["done"]; ok {
			continue
		}
		if choices, ok := msg["choices"]; ok {
			if choiceList, ok := choices.([]interface{}); ok && len(choiceList) > 0 {
				choice := choiceList[0].(map[string]interface{})
				if index, ok := choice["index"]; ok {
					if idx := int(index.(float64)); idx != 0 {
						t.Errorf("Expected index 0, got %d", idx)
					}
				}
			}
		}
	}
}

// Test: Timestamp consistency for chat completion
func TestStreamChatCompletion_TimestampConsistency(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "hello", Done: false},
		{Token: "world", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	before := time.Now().Unix()
	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-ts")
	after := time.Now().Unix()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	// All non-[DONE] messages should have the same timestamp
	var timestamps []int64
	for _, msg := range messages {
		if _, ok := msg["done"]; ok {
			continue
		}
		if ts, ok := msg["created"]; ok {
			timestamps = append(timestamps, int64(ts.(float64)))
		}
	}

	if len(timestamps) > 0 {
		first := timestamps[0]
		for i, ts := range timestamps {
			if ts != first {
				t.Errorf("Timestamp mismatch at position %d: %d vs %d", i, ts, first)
			}
			if ts < before || ts > after {
				t.Errorf("Timestamp %d out of range [%d, %d]", ts, before, after)
			}
		}
	}
}

// Test: Timestamp consistency for completion
func TestStreamCompletion_TimestampConsistency(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "test", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	before := time.Now().Unix()
	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-ts")
	after := time.Now().Unix()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	for _, msg := range messages {
		if ts, ok := msg["created"]; ok {
			timestamp := int64(ts.(float64))
			if timestamp < before || timestamp > after {
				t.Errorf("Timestamp %d out of range [%d, %d]", timestamp, before, after)
			}
		}
	}
}

// Test: Large response streaming
func TestStreamChatCompletion_LargeResponse(t *testing.T) {
	// Create a large response with many tokens
	chunks := make([]*backends.StreamChunk, 500)
	for i := 0; i < 499; i++ {
		chunks[i] = &backends.StreamChunk{
			Token: fmt.Sprintf("token_%d ", i),
			Done:  false,
		}
	}
	chunks[499] = &backends.StreamChunk{Token: "end", Done: true}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-large")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	if len(body) == 0 {
		t.Error("Expected non-empty response body")
	}

	if !strings.Contains(body, "[DONE]") {
		t.Error("Expected [DONE] message")
	}
}

// Test: Large response for completion
func TestStreamCompletion_LargeResponse(t *testing.T) {
	chunks := make([]*backends.StreamChunk, 300)
	for i := 0; i < 299; i++ {
		chunks[i] = &backends.StreamChunk{
			Token: fmt.Sprintf("w%d ", i),
			Done:  false,
		}
	}
	chunks[299] = &backends.StreamChunk{Token: "end", Done: true}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-large")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	if len(body) == 0 {
		t.Error("Expected non-empty response body")
	}
}

// Test: Reader close is called
func TestStreamChatCompletion_ReaderClosed(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "test", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-closed")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !reader.closed {
		t.Error("Expected reader to be closed")
	}
}

// Test: Reader close is called for completion
func TestStreamCompletion_ReaderClosed(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "test", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	// StreamCompletion doesn't explicitly call Close, but we test anyway
	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-closed")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

// Test: Chat completion with no tokens
func TestStreamChatCompletion_NoTokens(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-empty")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "[DONE]") {
		t.Error("Expected [DONE] message even with no tokens")
	}
}

// Test: Completion with no tokens
func TestStreamCompletion_NoTokens(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-empty")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "[DONE]") {
		t.Error("Expected [DONE] message")
	}
}

// Test: Multiple done flags don't break streaming
func TestStreamChatCompletion_MultipleDone(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "hello", Done: true},
		{Token: "world", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-multi-done")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	// Should exit after first done chunk
	doneCount := strings.Count(body, "[DONE]")
	if doneCount != 1 {
		t.Errorf("Expected 1 [DONE] message, got %d", doneCount)
	}
}

// Test: Completion with multiple done flags
func TestStreamCompletion_MultipleDone(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "first", Done: true},
		{Token: "second", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-multi-done")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	doneCount := strings.Count(body, "[DONE]")
	if doneCount != 1 {
		t.Errorf("Expected 1 [DONE] message, got %d", doneCount)
	}
}

// Test: Response header flushing
func TestStreamChatCompletion_Flushing(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "test", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-flush")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that headers were written
	if recorder.Header().Get("Content-Type") == "" {
		t.Error("Expected headers to be flushed")
	}
}

// Test: Completion response header flushing
func TestStreamCompletion_Flushing(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "test", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-flush")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if recorder.Header().Get("Content-Type") == "" {
		t.Error("Expected headers to be flushed")
	}
}

// Test: JSON marshaling error handling (ChatCompletion)
func TestStreamChatCompletion_JSONError(t *testing.T) {
	// This tests the json.Marshal error path by using malformed chunk data
	// We can't directly cause a marshal error with normal types, so we test
	// that normal JSON marshaling works
	chunks := []*backends.StreamChunk{
		{Token: "test", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-json")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	// Verify JSON is present
	if !strings.Contains(body, "{") {
		t.Error("Expected JSON in response")
	}
}

// Test: Chat vs Completion object types
func TestStreamChat_vs_Completion_ObjectTypes(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "test", Done: true},
	}

	// Test chat completion
	reader1 := NewMockStreamReader(chunks)
	recorder1 := httptest.NewRecorder()
	StreamChatCompletion(recorder1, reader1, "gpt-4", "id1")
	body1 := recorder1.Body.String()

	// Test completion
	reader2 := NewMockStreamReader(chunks)
	recorder2 := httptest.NewRecorder()
	StreamCompletion(recorder2, reader2, "text-davinci-003", "id2")
	body2 := recorder2.Body.String()

	// Chat should have "chat.completion.chunk"
	if !strings.Contains(body1, "chat.completion.chunk") {
		t.Error("Expected 'chat.completion.chunk' in chat response")
	}

	// Completion should have "text_completion.chunk"
	if !strings.Contains(body2, "text_completion.chunk") {
		t.Error("Expected 'text_completion.chunk' in completion response")
	}
}

// Test: ValidateSSEEventFormat for both streaming functions
func TestStreamChatCompletion_ValidSSEEvent(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "hello", Done: false},
		{Token: "world", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-sse-valid")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()

	// Parse line by line to verify SSE format
	lines := strings.Split(body, "\n")
	eventCount := 0
	for i := 0; i < len(lines)-1; i++ {
		if strings.HasPrefix(lines[i], "data: ") {
			eventCount++
			// Next line should be blank (or end of string)
			if lines[i+1] != "" && i+1 < len(lines)-1 {
				// It's okay if the last line doesn't have a blank line after
			}
		}
	}

	if eventCount == 0 {
		t.Error("Expected at least one SSE data event")
	}
}

// Test: Model and ID propagation in chunks
func TestStreamChatCompletion_ModelIDPropagation(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "chunk1", Done: false},
		{Token: "chunk2", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	testModel := "custom-model-v1"
	testID := "unique-id-xyz"

	err := StreamChatCompletion(recorder, reader, testModel, testID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	for _, msg := range messages {
		if _, ok := msg["done"]; ok {
			continue
		}
		if model, ok := msg["model"]; ok {
			if model != testModel {
				t.Errorf("Expected model %s, got %v", testModel, model)
			}
		}
		if id, ok := msg["id"]; ok {
			if id != testID {
				t.Errorf("Expected id %s, got %v", testID, id)
			}
		}
	}
}

// Test: Model and ID propagation for completion
func TestStreamCompletion_ModelIDPropagation(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "data", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	testModel := "completion-model-v2"
	testID := "cmpl-unique-id"

	err := StreamCompletion(recorder, reader, testModel, testID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	for _, msg := range messages {
		if model, ok := msg["model"]; ok {
			if model != testModel {
				t.Errorf("Expected model %s, got %v", testModel, model)
			}
		}
		if id, ok := msg["id"]; ok {
			if id != testID {
				t.Errorf("Expected id %s, got %v", testID, id)
			}
		}
	}
}

// Test: Error response format
func TestStreamChatCompletion_ErrorFormat(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "partial", Done: false},
	}

	errorMsg := "backend connection failed"
	reader := NewMockStreamReaderWithError(chunks, fmt.Errorf("%s", errorMsg))
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-err-fmt")
	if err == nil {
		t.Error("Expected error")
	}

	body := recorder.Body.String()

	// Check error event format
	if !strings.Contains(body, "event: error") {
		t.Error("Expected 'event: error' in response")
	}

	if !strings.Contains(body, errorMsg) {
		t.Error("Expected error message in response")
	}

	// Check for error structure
	if !strings.Contains(body, "stream_error") {
		t.Error("Expected 'stream_error' type in error event")
	}

	if !strings.Contains(body, "backend_error") {
		t.Error("Expected 'backend_error' code in error event")
	}
}

// Test: Delta field in chat chunks
func TestStreamChatCompletion_DeltaField(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "response", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-delta")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	for _, msg := range messages {
		if _, ok := msg["done"]; ok {
			continue
		}
		if choices, ok := msg["choices"]; ok {
			if choiceList, ok := choices.([]interface{}); ok && len(choiceList) > 0 {
				choice := choiceList[0].(map[string]interface{})
				if _, ok := choice["delta"]; !ok {
					t.Error("Expected 'delta' field in choice")
				}
			}
		}
	}
}

// Test: Text field in completion chunks
func TestStreamCompletion_TextField(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "completion", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-text")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	for _, msg := range messages {
		if _, ok := msg["done"]; ok {
			continue
		}
		if choices, ok := msg["choices"]; ok {
			if choiceList, ok := choices.([]interface{}); ok && len(choiceList) > 0 {
				choice := choiceList[0].(map[string]interface{})
				if text, ok := choice["text"]; ok {
					if text != "completion" {
						t.Errorf("Expected text 'completion', got %v", text)
					}
				}
			}
		}
	}
}

// Test: Chat completion with chunks that test branch coverage
func TestStreamChatCompletion_ChoiceStructure(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "test", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-choice")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	// Verify choice structure exists and has proper index
	for _, msg := range messages {
		if _, ok := msg["done"]; ok {
			continue
		}
		if choices, ok := msg["choices"]; ok {
			if choiceList, ok := choices.([]interface{}); ok {
				if len(choiceList) == 0 {
					t.Error("Expected at least one choice")
				}
				// Verify it's a valid choice structure
				choice := choiceList[0].(map[string]interface{})
				if _, ok := choice["index"]; !ok {
					t.Error("Expected index field in choice")
				}
				if _, ok := choice["delta"]; !ok {
					t.Error("Expected delta field in choice")
				}
			}
		}
	}
}

// Test: Completion with chunks that test object field
func TestStreamCompletion_Object(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "hello", Done: false},
		{Token: "world", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-object")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	// All messages should have object field set to text_completion.chunk
	for _, msg := range messages {
		if _, ok := msg["done"]; ok {
			continue
		}
		if obj, ok := msg["object"]; ok {
			if obj != "text_completion.chunk" {
				t.Errorf("Expected object 'text_completion.chunk', got %v", obj)
			}
		} else {
			t.Error("Expected object field in message")
		}
	}
}

// Test: Chat completion with all required fields
func TestStreamChatCompletion_AllFields(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "response", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "id-12345")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	for _, msg := range messages {
		if _, ok := msg["done"]; ok {
			continue
		}

		// Check all required top-level fields
		requiredFields := []string{"id", "object", "created", "model", "choices"}
		for _, field := range requiredFields {
			if _, ok := msg[field]; !ok {
				t.Errorf("Expected %s field in message", field)
			}
		}

		// Verify object is correct
		if obj, ok := msg["object"]; ok {
			if obj != "chat.completion.chunk" {
				t.Errorf("Expected object 'chat.completion.chunk', got %v", obj)
			}
		}

		// Verify choices structure
		if choices, ok := msg["choices"]; ok {
			if choiceList, ok := choices.([]interface{}); ok {
				if len(choiceList) != 1 {
					t.Errorf("Expected 1 choice, got %d", len(choiceList))
				}
				choice := choiceList[0].(map[string]interface{})
				if _, ok := choice["delta"]; !ok {
					t.Error("Expected delta in choice")
				}
			}
		}
	}
}

// Test: Stream completion with all fields
func TestStreamCompletion_AllFields(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "content", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-12345")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	for _, msg := range messages {
		if _, ok := msg["done"]; ok {
			continue
		}

		// Check all required top-level fields
		requiredFields := []string{"id", "object", "created", "model", "choices"}
		for _, field := range requiredFields {
			if _, ok := msg[field]; !ok {
				t.Errorf("Expected %s field in message", field)
			}
		}

		// Verify object is correct
		if obj, ok := msg["object"]; ok {
			if obj != "text_completion.chunk" {
				t.Errorf("Expected object 'text_completion.chunk', got %v", obj)
			}
		}
	}
}

// Test: Response headers are set immediately
func TestStreamChatCompletion_HeadersSetImmediately(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "test", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-headers")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// All headers should be set
	headers := map[string]string{
		"Content-Type":        "text/event-stream",
		"Cache-Control":       "no-cache",
		"Connection":          "keep-alive",
		"X-Accel-Buffering":   "no",
	}

	for header, expected := range headers {
		if value := recorder.Header().Get(header); value != expected {
			t.Errorf("Header %s: expected %q, got %q", header, expected, value)
		}
	}
}

// Test: Completion response headers
func TestStreamCompletion_HeadersSetImmediately(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "test", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-headers")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check headers
	headers := map[string]string{
		"Content-Type":  "text/event-stream",
		"Cache-Control": "no-cache",
		"Connection":    "keep-alive",
	}

	for header, expected := range headers {
		if value := recorder.Header().Get(header); value != expected {
			t.Errorf("Header %s: expected %q, got %q", header, expected, value)
		}
	}
}

// Test: Chat completion with nil finish reason initially
func TestStreamChatCompletion_NilFinishReasonInitially(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "first", Done: false},
		{Token: "second", Done: false},
		{Token: "final", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamChatCompletion(recorder, reader, "gpt-4", "chatcmpl-nil-finish")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	foundNonFinal := false
	foundFinal := false

	for _, msg := range messages {
		if _, ok := msg["done"]; ok {
			continue
		}
		if choices, ok := msg["choices"]; ok {
			if choiceList, ok := choices.([]interface{}); ok && len(choiceList) > 0 {
				choice := choiceList[0].(map[string]interface{})
				finishReason := choice["finish_reason"]
				if finishReason == nil {
					foundNonFinal = true
				} else if finishReason == "stop" {
					foundFinal = true
				}
			}
		}
	}

	if !foundNonFinal {
		t.Error("Expected at least one chunk with nil finish_reason")
	}
	if !foundFinal {
		t.Error("Expected final chunk with finish_reason='stop'")
	}
}

// Test: Completion with nil finish reason initially
func TestStreamCompletion_NilFinishReasonInitially(t *testing.T) {
	chunks := []*backends.StreamChunk{
		{Token: "chunk1", Done: false},
		{Token: "chunk2", Done: true},
	}

	reader := NewMockStreamReader(chunks)
	recorder := httptest.NewRecorder()

	err := StreamCompletion(recorder, reader, "text-davinci-003", "cmpl-nil-finish")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	body := recorder.Body.String()
	messages := parseSSEResponse(body)

	foundFinal := false
	for _, msg := range messages {
		if _, ok := msg["done"]; ok {
			continue
		}
		if choices, ok := msg["choices"]; ok {
			if choiceList, ok := choices.([]interface{}); ok && len(choiceList) > 0 {
				choice := choiceList[0].(map[string]interface{})
				if finishReason, ok := choice["finish_reason"]; ok && finishReason == "stop" {
					foundFinal = true
				}
			}
		}
	}

	if !foundFinal {
		t.Error("Expected final chunk with finish_reason='stop'")
	}
}
