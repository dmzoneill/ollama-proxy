package openai

import (
	"bytes"
	"sync"
)

// Pool for ChatCompletionChunk structs
var chatChunkPool = sync.Pool{
	New: func() interface{} {
		return &ChatCompletionChunk{
			Choices: make([]ChatCompletionChunkChoice, 1),
		}
	},
}

// getChatChunk gets a chunk from pool
func getChatChunk() *ChatCompletionChunk {
	chunk := chatChunkPool.Get().(*ChatCompletionChunk)
	// Reset fields to default state
	chunk.ID = ""
	chunk.Object = "chat.completion.chunk"
	chunk.Created = 0
	chunk.Model = ""
	if len(chunk.Choices) > 0 {
		chunk.Choices[0] = ChatCompletionChunkChoice{
			Index:        0,
			Delta:        ChatCompletionChunkDelta{},
			FinishReason: nil,
		}
	}
	return chunk
}

// putChatChunk returns chunk to pool
func putChatChunk(chunk *ChatCompletionChunk) {
	// Don't return to pool if it has grown too large
	if len(chunk.Choices) > 10 {
		return
	}
	chatChunkPool.Put(chunk)
}

// Pool for CompletionChunk structs
var completionChunkPool = sync.Pool{
	New: func() interface{} {
		return &CompletionChunk{
			Choices: make([]CompletionChunkChoice, 1),
		}
	},
}

// getCompletionChunk gets a chunk from pool
func getCompletionChunk() *CompletionChunk {
	chunk := completionChunkPool.Get().(*CompletionChunk)
	// Reset fields to default state
	chunk.ID = ""
	chunk.Object = "text_completion.chunk"
	chunk.Created = 0
	chunk.Model = ""
	if len(chunk.Choices) > 0 {
		chunk.Choices[0] = CompletionChunkChoice{
			Index:        0,
			Text:         "",
			FinishReason: nil,
		}
	}
	return chunk
}

// putCompletionChunk returns chunk to pool
func putCompletionChunk(chunk *CompletionChunk) {
	// Don't return to pool if it has grown too large
	if len(chunk.Choices) > 10 {
		return
	}
	completionChunkPool.Put(chunk)
}

// Pool for SSE frame buffers
var sseBufferPool = sync.Pool{
	New: func() interface{} {
		buf := bytes.NewBuffer(make([]byte, 0, 2048)) // Pre-allocate 2KB
		return buf
	},
}

// getSSEBuffer gets a buffer from pool
func getSSEBuffer() *bytes.Buffer {
	return sseBufferPool.Get().(*bytes.Buffer)
}

// putSSEBuffer returns buffer to pool
func putSSEBuffer(buf *bytes.Buffer) {
	// Reset capacity if grown too large (don't keep huge buffers)
	if buf.Cap() > 16384 {
		return
	}
	buf.Reset() // Reset length to 0
	sseBufferPool.Put(buf)
}

// Pool for JSON byte slices
var jsonBytesPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 1024) // Pre-allocate 1KB
		return &b
	},
}

// getJSONBytes gets a byte slice from pool
func getJSONBytes() *[]byte {
	return jsonBytesPool.Get().(*[]byte)
}

// putJSONBytes returns byte slice to pool
func putJSONBytes(b *[]byte) {
	// Don't return to pool if it has grown too large
	if cap(*b) > 8192 {
		return
	}
	*b = (*b)[:0] // Reset length
	jsonBytesPool.Put(b)
}
