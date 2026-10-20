package openai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/logging"
	"go.uber.org/zap"
)

// StreamChatCompletion streams a chat completion response in OpenAI SSE format
func StreamChatCompletion(w http.ResponseWriter, reader backends.StreamReader, model string, completionID string) error {
	defer closeStream(reader)

	// Set SSE headers
	w.Header().Set("Content-Type", strTextEventStream)
	w.Header().Set("Cache-Control", strNoCache)
	w.Header().Set("Connection", strKeepAlive)
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Flush headers immediately
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	timestamp := time.Now().Unix()
	index := 0

	// Channel for backpressure control
	writeChan := make(chan []byte, 10) // Buffer 10 chunks
	errChan := make(chan error, 1)
	done := make(chan struct{})

	go pumpSSE(w, writeChan, errChan, done)

	// Reader loop
	for {
		chunk, err := reader.Recv()
		if err != nil {
			close(writeChan)
			<-done // Wait for writer to finish

			// Check if it's a normal EOF or an error
			if err.Error() != "EOF" {
				if writeErr := writeStreamError(w, err); writeErr != nil {
					return writeErr
				}
				return err
			}
			break
		}

		// Get chunk from pool
		openaiChunk := getChatChunk()

		// Populate chunk fields
		openaiChunk.ID = completionID
		openaiChunk.Object = "chat.completion.chunk"
		openaiChunk.Created = timestamp
		openaiChunk.Model = model

		if chunk.Done {
			// Final chunk with finish_reason
			finishReason := strStop
			openaiChunk.Choices[0].Index = 0
			openaiChunk.Choices[0].Delta.Content = chunk.Token
			openaiChunk.Choices[0].FinishReason = &finishReason
		} else {
			// Regular chunk
			openaiChunk.Choices[0].Index = 0
			openaiChunk.Choices[0].Delta.Content = chunk.Token
			openaiChunk.Choices[0].FinishReason = nil
		}

		// Marshal to JSON
		data, err := json.Marshal(openaiChunk)
		if err != nil {
			putChatChunk(openaiChunk) // Return to pool
			close(writeChan)
			<-done
			return fmt.Errorf("failed to marshal chunk: %w", err)
		}

		// Return chunk to pool
		putChatChunk(openaiChunk)

		// Send to writer with backpressure (blocking)
		select {
		case writeChan <- data:
			// Sent successfully
		case err := <-errChan:
			// Writer encountered error (slow client)
			close(writeChan)
			<-done
			return err
		case <-time.After(5 * time.Second):
			// Backpressure timeout - client can't keep up
			close(writeChan)
			<-done
			return fmt.Errorf("backpressure timeout - client too slow")
		}

		index++

		// Exit if this was the final chunk
		if chunk.Done {
			break
		}
	}

	// Close write channel and wait for writer to finish
	close(writeChan)
	<-done

	// Check for writer errors
	select {
	case err := <-errChan:
		return err
	default:
		// No error
	}

	return writeSSE(w, "data: [DONE]\n\n")
}

// StreamCompletion streams a completion response in OpenAI SSE format
func StreamCompletion(w http.ResponseWriter, reader backends.StreamReader, model string, completionID string) error {
	defer closeStream(reader)

	// Set SSE headers
	w.Header().Set("Content-Type", strTextEventStream)
	w.Header().Set("Cache-Control", strNoCache)
	w.Header().Set("Connection", strKeepAlive)

	// Flush headers immediately
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	timestamp := time.Now().Unix()
	index := 0

	for {
		chunk, err := reader.Recv()
		if err != nil {
			// End of stream
			break
		}

		// Get chunk from pool
		openaiChunk := getCompletionChunk()

		// Populate chunk fields
		openaiChunk.ID = completionID
		openaiChunk.Object = strTextCompletionChunk
		openaiChunk.Created = timestamp
		openaiChunk.Model = model

		if chunk.Done {
			// Final chunk with finish_reason
			finishReason := strStop
			openaiChunk.Choices[0].Text = chunk.Token
			openaiChunk.Choices[0].Index = 0
			openaiChunk.Choices[0].FinishReason = &finishReason
		} else {
			// Regular chunk
			openaiChunk.Choices[0].Text = chunk.Token
			openaiChunk.Choices[0].Index = 0
			openaiChunk.Choices[0].FinishReason = nil
		}

		// Marshal to JSON
		data, err := json.Marshal(openaiChunk)
		if err != nil {
			putCompletionChunk(openaiChunk) // Return to pool
			return fmt.Errorf("failed to marshal chunk: %w", err)
		}

		if err := writeSSE(w, "data: %s\n\n", string(data)); err != nil {
			putCompletionChunk(openaiChunk)
			return err
		}

		// Return chunk to pool
		putCompletionChunk(openaiChunk)

		index++

		// Exit if this was the final chunk
		if chunk.Done {
			break
		}
	}

	return writeSSE(w, "data: [DONE]\n\n")
}

func pumpSSE(w http.ResponseWriter, writeChan <-chan []byte, errChan chan<- error, done chan struct{}) {
	defer close(done)
	for data := range writeChan {
		written := make(chan bool, 1)
		go func(payload []byte) {
			if err := writeSSE(w, "data: %s\n\n", string(payload)); err != nil {
				errChan <- err
				return
			}
			written <- true
		}(data)
		select {
		case <-written:
		case <-time.After(10 * time.Second):
			errChan <- fmt.Errorf("client write timeout - slow consumer")
			return
		}
	}
}

func writeStreamError(w http.ResponseWriter, streamErr error) error {
	errorEvent := map[string]interface{}{
		"error": map[string]interface{}{
			"message": streamErr.Error(),
			"type":    "stream_error",
			"code":    "backend_error",
		},
	}
	errorJSON, err := json.Marshal(errorEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal stream error: %w", err)
	}
	return writeSSE(w, "event: error\ndata: %s\n\n", string(errorJSON))
}

func closeStream(reader backends.StreamReader) {
	if reader == nil {
		return
	}
	if err := reader.Close(); err != nil {
		logging.Warn("close stream reader", zap.Error(err))
	}
}

func writeSSE(w http.ResponseWriter, format string, args ...any) error {
	if _, err := fmt.Fprintf(w, format, args...); err != nil {
		return err
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}
