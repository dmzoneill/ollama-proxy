package openai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// StreamChatCompletion streams a chat completion response in OpenAI SSE format
func StreamChatCompletion(w http.ResponseWriter, reader backends.StreamReader, model string, completionID string) error {
	defer reader.Close()

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
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

	// Writer goroutine with timeout protection
	go func() {
		defer close(done)
		for data := range writeChan {
			// Write with timeout protection (detect slow clients)
			written := make(chan bool, 1)
			go func() {
				fmt.Fprintf(w, "data: %s\n\n", string(data))
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
				written <- true
			}()

			select {
			case <-written:
				// Write successful
			case <-time.After(10 * time.Second):
				// Client too slow
				errChan <- fmt.Errorf("client write timeout - slow consumer")
				return
			}
		}
	}()

	// Reader loop
	for {
		chunk, err := reader.Recv()
		if err != nil {
			close(writeChan)
			<-done // Wait for writer to finish

			// Check if it's a normal EOF or an error
			if err.Error() != "EOF" {
				// Send error event to client
				errorEvent := map[string]interface{}{
					"error": map[string]interface{}{
						"message": err.Error(),
						"type":    "stream_error",
						"code":    "backend_error",
					},
				}
				errorJSON, _ := json.Marshal(errorEvent)
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", string(errorJSON))
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
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
			finishReason := "stop"
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

	// Send [DONE] message
	fmt.Fprintf(w, "data: [DONE]\n\n")

	// Final flush
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

// StreamCompletion streams a completion response in OpenAI SSE format
func StreamCompletion(w http.ResponseWriter, reader backends.StreamReader, model string, completionID string) error {
	defer reader.Close()

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

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
		openaiChunk.Object = "text_completion.chunk"
		openaiChunk.Created = timestamp
		openaiChunk.Model = model

		if chunk.Done {
			// Final chunk with finish_reason
			finishReason := "stop"
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

		// Write SSE formatted data
		fmt.Fprintf(w, "data: %s\n\n", string(data))

		// Flush the data immediately
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		// Return chunk to pool
		putCompletionChunk(openaiChunk)

		index++

		// Exit if this was the final chunk
		if chunk.Done {
			break
		}
	}

	// Send [DONE] message
	fmt.Fprintf(w, "data: [DONE]\n\n")

	// Final flush
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}
