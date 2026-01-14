package openai

import (
	"testing"
)

func TestChatChunkPool(t *testing.T) {
	// Test getting a chunk from pool
	chunk := getChatChunk()
	if chunk == nil {
		t.Fatal("getChatChunk returned nil")
	}

	if chunk.Object != "chat.completion.chunk" {
		t.Errorf("Expected Object='chat.completion.chunk', got %s", chunk.Object)
	}

	if len(chunk.Choices) == 0 {
		t.Error("Expected at least one choice")
	}

	// Test putting chunk back
	putChatChunk(chunk)

	// Get another chunk (might be the same one from pool)
	chunk2 := getChatChunk()
	if chunk2 == nil {
		t.Fatal("Second getChatChunk returned nil")
	}

	// Test that large chunks aren't returned to pool
	largeChunk := getChatChunk()
	largeChunk.Choices = make([]ChatCompletionChunkChoice, 15) // More than 10
	putChatChunk(largeChunk) // Should not panic, but won't be pooled
}

func TestCompletionChunkPool(t *testing.T) {
	// Test getting a chunk from pool
	chunk := getCompletionChunk()
	if chunk == nil {
		t.Fatal("getCompletionChunk returned nil")
	}

	if chunk.Object != "text_completion.chunk" {
		t.Errorf("Expected Object='text_completion.chunk', got %s", chunk.Object)
	}

	if len(chunk.Choices) == 0 {
		t.Error("Expected at least one choice")
	}

	// Test putting chunk back
	putCompletionChunk(chunk)

	// Get another chunk
	chunk2 := getCompletionChunk()
	if chunk2 == nil {
		t.Fatal("Second getCompletionChunk returned nil")
	}

	// Test that large chunks aren't returned to pool
	largeChunk := getCompletionChunk()
	largeChunk.Choices = make([]CompletionChunkChoice, 15)
	putCompletionChunk(largeChunk)
}

func TestSSEBufferPool(t *testing.T) {
	// Test getting a buffer from pool
	buf := getSSEBuffer()
	if buf == nil {
		t.Fatal("getSSEBuffer returned nil")
	}

	// Write some data
	buf.WriteString("test data")
	if buf.Len() != 9 {
		t.Errorf("Expected buffer length 9, got %d", buf.Len())
	}

	// Test putting buffer back
	putSSEBuffer(buf)

	// Get another buffer (should be reset)
	buf2 := getSSEBuffer()
	if buf2 == nil {
		t.Fatal("Second getSSEBuffer returned nil")
	}

	// If we got the same buffer, it should be reset
	if buf2 == buf && buf2.Len() != 0 {
		t.Error("Buffer was not reset when returned to pool")
	}

	// Test that huge buffers aren't returned to pool
	hugeBuf := getSSEBuffer()
	hugeBuf.Grow(20000) // More than 16384
	putSSEBuffer(hugeBuf) // Should not panic, but won't be pooled
}

func TestJSONBytesPool(t *testing.T) {
	// Test getting bytes from pool
	b := getJSONBytes()
	if b == nil {
		t.Fatal("getJSONBytes returned nil")
	}

	// Test putting bytes back
	putJSONBytes(b)

	// Get another slice
	b2 := getJSONBytes()
	if b2 == nil {
		t.Fatal("Second getJSONBytes returned nil")
	}

	// Test that large slices aren't returned to pool
	largeBytes := make([]byte, 20000) // More than 8192
	putJSONBytes(&largeBytes)          // Should not panic, but won't be pooled
}
