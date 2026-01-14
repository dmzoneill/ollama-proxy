package router

import (
	"context"
	"fmt"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

func TestQueueManager_GetRawQueueDepth(t *testing.T) {
	qm := NewQueueManager()

	// Test non-existent backend
	depth := qm.GetRawQueueDepth("non-existent")
	if depth != 0 {
		t.Errorf("Expected depth 0 for non-existent backend, got %d", depth)
	}
}

func TestQueueManager_GetPriorityBreakdown(t *testing.T) {
	qm := NewQueueManager()

	// Test non-existent backend
	breakdown := qm.GetPriorityBreakdown("non-existent")
	expected := [4]int{}
	if breakdown != expected {
		t.Errorf("Expected empty breakdown for non-existent backend, got %v", breakdown)
	}
}

func TestQueueManager_MarkRequestEnd(t *testing.T) {
	qm := NewQueueManager()

	// Test marking end for non-existent backend (should not panic)
	qm.MarkRequestEnd("non-existent", backends.PriorityNormal)

	// Test marking end for another backend
	qm.MarkRequestEnd("test-backend", backends.PriorityHigh)
}

func TestQueueManager_GetAllQueueStats(t *testing.T) {
	qm := NewQueueManager()

	stats := qm.GetAllQueueStats()

	// Initially should have no queues
	if len(stats) != 0 {
		t.Errorf("Expected no queue stats initially, got %d", len(stats))
	}
}

func TestQueueTrackingBackend_Generate(t *testing.T) {
	qm := NewQueueManager()

	backend := &MockBackend{
		id:      "test-backend",
		healthy: true,
	}

	qtb := &QueueTrackingBackend{
		Backend:  backend,
		queueMgr: qm,
		priority: backends.PriorityNormal,
	}

	// Call Generate (it will call MarkRequestEnd via defer)
	resp, err := qtb.Generate(nil, &backends.GenerateRequest{})

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if resp == nil {
		t.Error("Generate returned nil response")
	}
}

func TestQueueTrackingBackend_GenerateStream(t *testing.T) {
	qm := NewQueueManager()

	backend := &MockBackend{
		id:      "test-backend",
		healthy: true,
	}

	qtb := &QueueTrackingBackend{
		Backend:  backend,
		queueMgr: qm,
		priority: backends.PriorityNormal,
	}

	// Call GenerateStream (MockBackend returns nil reader, nil error)
	_, err := qtb.GenerateStream(nil, &backends.GenerateRequest{})

	// MockBackend returns nil, nil so we expect this to succeed
	// but won't get a usable reader
	if err != nil {
		t.Logf("GenerateStream returned error (expected with MockBackend): %v", err)
	}
}

func TestQueueTrackingBackend_GenerateStream_Success(t *testing.T) {
	qm := NewQueueManager()

	// Create a mock stream reader
	mockReader := &MockStreamReader{}

	backend := &mockBackendWithStreamReader{
		MockBackend: MockBackend{
			id:      "test-backend",
			healthy: true,
		},
		streamReader: mockReader,
		streamError:  nil,
	}

	qtb := &QueueTrackingBackend{
		Backend:  backend,
		queueMgr: qm,
		priority: backends.PriorityNormal,
	}

	// Start a request to track
	qm.MarkRequestStart("test-backend", backends.PriorityNormal)

	// Call GenerateStream - should return tracking reader
	reader, err := qtb.GenerateStream(nil, &backends.GenerateRequest{})
	if err != nil {
		t.Fatalf("GenerateStream failed: %v", err)
	}

	if reader == nil {
		t.Fatal("GenerateStream returned nil reader")
	}

	// Verify it's a tracking reader
	trackingReader, ok := reader.(*trackingStreamReader)
	if !ok {
		t.Fatal("Reader is not a trackingStreamReader")
	}

	// Verify queue depth increased
	depth := qm.GetRawQueueDepth("test-backend")
	if depth != 1 {
		t.Errorf("Expected queue depth 1, got %d", depth)
	}

	// Close the reader - this should call onClose and mark request end
	trackingReader.Close()

	// Verify queue depth decreased
	depth = qm.GetRawQueueDepth("test-backend")
	if depth != 0 {
		t.Errorf("Expected queue depth 0 after close, got %d", depth)
	}
}

func TestQueueTrackingBackend_GenerateStream_Error(t *testing.T) {
	qm := NewQueueManager()

	backend := &mockBackendWithStreamReader{
		MockBackend: MockBackend{
			id:      "test-backend",
			healthy: true,
		},
		streamReader: nil,
		streamError:  fmt.Errorf("stream error"),
	}

	qtb := &QueueTrackingBackend{
		Backend:  backend,
		queueMgr: qm,
		priority: backends.PriorityNormal,
	}

	// Start a request to track
	qm.MarkRequestStart("test-backend", backends.PriorityNormal)

	// Verify initial queue depth
	depth := qm.GetRawQueueDepth("test-backend")
	if depth != 1 {
		t.Errorf("Expected queue depth 1, got %d", depth)
	}

	// Call GenerateStream - should return error and mark request end
	reader, err := qtb.GenerateStream(nil, &backends.GenerateRequest{})

	if err == nil {
		t.Fatal("Expected error from GenerateStream")
	}

	if reader != nil {
		t.Error("Expected nil reader on error")
	}

	// Verify queue depth decreased (marked end automatically)
	depth = qm.GetRawQueueDepth("test-backend")
	if depth != 0 {
		t.Errorf("Expected queue depth 0 after error, got %d", depth)
	}
}

// MockStreamReader is a simple mock for testing
type MockStreamReader struct {
	closeCalled bool
}

func (m *MockStreamReader) Recv() (*backends.StreamChunk, error) {
	return nil, nil
}

func (m *MockStreamReader) Close() error {
	m.closeCalled = true
	return nil
}

// mockBackendWithStreamReader is a mock backend that can return specific stream readers/errors
type mockBackendWithStreamReader struct {
	MockBackend
	streamReader backends.StreamReader
	streamError  error
}

func (m *mockBackendWithStreamReader) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
	return m.streamReader, m.streamError
}

func TestTrackingStreamReader_Close(t *testing.T) {
	called := false
	onClose := func() {
		called = true
	}

	// Create a mock StreamReader
	mockReader := &MockStreamReader{}

	tsr := &trackingStreamReader{
		StreamReader: mockReader,
		onClose:      onClose,
		closed:       false,
	}

	// Close should call onClose
	tsr.Close()

	if !called {
		t.Error("onClose callback was not called")
	}

	if !tsr.closed {
		t.Error("closed flag was not set")
	}

	if !mockReader.closeCalled {
		t.Error("underlying reader Close was not called")
	}

	// Calling Close again should not call onClose again
	called = false
	mockReader.closeCalled = false
	tsr.Close()

	if called {
		t.Error("onClose should not be called on second Close")
	}

	// Note: underlying reader Close is ALWAYS called (not protected by closed flag)
	if !mockReader.closeCalled {
		t.Error("underlying reader Close should still be called on second Close")
	}
}

func TestQueueManager_MarkRequestStart(t *testing.T) {
	qm := NewQueueManager()

	// Start a request
	qm.MarkRequestStart("backend-1", backends.PriorityNormal)

	// Check queue depth
	depth := qm.GetRawQueueDepth("backend-1")
	if depth != 1 {
		t.Errorf("Expected depth 1, got %d", depth)
	}

	// Start another request with different priority
	qm.MarkRequestStart("backend-1", backends.PriorityHigh)

	depth = qm.GetRawQueueDepth("backend-1")
	if depth != 2 {
		t.Errorf("Expected depth 2, got %d", depth)
	}

	// Check priority breakdown
	breakdown := qm.GetPriorityBreakdown("backend-1")
	if breakdown[backends.PriorityNormal] != 1 {
		t.Errorf("Expected 1 normal priority request, got %d", breakdown[backends.PriorityNormal])
	}
	if breakdown[backends.PriorityHigh] != 1 {
		t.Errorf("Expected 1 high priority request, got %d", breakdown[backends.PriorityHigh])
	}
}

func TestQueueManager_MarkRequestEnd_WithStart(t *testing.T) {
	qm := NewQueueManager()

	// Start then end a request
	qm.MarkRequestStart("backend-1", backends.PriorityNormal)
	qm.MarkRequestEnd("backend-1", backends.PriorityNormal)

	depth := qm.GetRawQueueDepth("backend-1")
	if depth != 0 {
		t.Errorf("Expected depth 0 after end, got %d", depth)
	}
}

func TestQueueManager_MarkRequestEnd_NegativeSafety(t *testing.T) {
	qm := NewQueueManager()

	// End without start (should handle gracefully with safety check)
	qm.MarkRequestEnd("backend-1", backends.PriorityNormal)
	qm.MarkRequestEnd("backend-1", backends.PriorityNormal)

	// Should not go negative
	depth := qm.GetRawQueueDepth("backend-1")
	if depth < 0 {
		t.Errorf("Depth went negative: %d", depth)
	}
}

func TestQueueManager_GetQueueDepth_Weighted(t *testing.T) {
	qm := NewQueueManager()

	// Add requests at different priorities
	qm.MarkRequestStart("backend-1", backends.PriorityBestEffort)
	qm.MarkRequestStart("backend-1", backends.PriorityNormal)
	qm.MarkRequestStart("backend-1", backends.PriorityHigh)
	qm.MarkRequestStart("backend-1", backends.PriorityCritical)

	// Get weighted depth for different priorities
	depthLow := qm.GetQueueDepth("backend-1", backends.PriorityBestEffort)
	depthNormal := qm.GetQueueDepth("backend-1", backends.PriorityNormal)
	depthHigh := qm.GetQueueDepth("backend-1", backends.PriorityHigh)
	depthCritical := qm.GetQueueDepth("backend-1", backends.PriorityCritical)

	// Weighted depth should increase with priority level
	if depthLow >= depthNormal {
		t.Errorf("Expected weighted depth to increase: low=%d, normal=%d", depthLow, depthNormal)
	}
	if depthNormal >= depthHigh {
		t.Errorf("Expected weighted depth to increase: normal=%d, high=%d", depthNormal, depthHigh)
	}
	if depthHigh >= depthCritical {
		t.Errorf("Expected weighted depth to increase: high=%d, critical=%d", depthHigh, depthCritical)
	}

	t.Logf("Weighted depths: low=%d, normal=%d, high=%d, critical=%d",
		depthLow, depthNormal, depthHigh, depthCritical)
}

func TestQueueManager_GetQueueDepth_NonExistent(t *testing.T) {
	qm := NewQueueManager()

	depth := qm.GetQueueDepth("non-existent", backends.PriorityNormal)
	if depth != 0 {
		t.Errorf("Expected depth 0 for non-existent backend, got %d", depth)
	}
}

func TestQueueManager_GetAllQueueStats_Multiple(t *testing.T) {
	qm := NewQueueManager()

	// Add requests to multiple backends
	qm.MarkRequestStart("backend-1", backends.PriorityNormal)
	qm.MarkRequestStart("backend-1", backends.PriorityHigh)
	qm.MarkRequestStart("backend-2", backends.PriorityBestEffort)

	stats := qm.GetAllQueueStats()

	if len(stats) != 2 {
		t.Errorf("Expected 2 backends in stats, got %d", len(stats))
	}

	if stats["backend-1"].Pending != 2 {
		t.Errorf("Expected backend-1 to have 2 pending, got %d", stats["backend-1"].Pending)
	}

	if stats["backend-2"].Pending != 1 {
		t.Errorf("Expected backend-2 to have 1 pending, got %d", stats["backend-2"].Pending)
	}

	// Check priority counts
	if stats["backend-1"].PriorityCounts[backends.PriorityNormal] != 1 {
		t.Error("Expected 1 normal priority request for backend-1")
	}
	if stats["backend-1"].PriorityCounts[backends.PriorityHigh] != 1 {
		t.Error("Expected 1 high priority request for backend-1")
	}
}

// TestQueueSafetyChecks tests that negative value safety checks work
func TestQueueSafetyChecks(t *testing.T) {
	qm := NewQueueManager()

	backendID := "test-backend"
	priority := backends.PriorityNormal

	// Add a request
	qm.MarkRequestStart(backendID, priority)

	// Complete it twice to trigger safety check
	qm.MarkRequestEnd(backendID, priority)
	qm.MarkRequestEnd(backendID, priority) // Should trigger safety check

	stats := qm.GetAllQueueStats()
	if stat, ok := stats[backendID]; ok {
		// After completing twice, pending should be 0 (safety check prevents negative)
		if stat.Pending < 0 {
			t.Errorf("Expected pending >= 0, got %d", stat.Pending)
		}

		// Priority count should also be 0 (safety check)
		if stat.PriorityCounts[priority] < 0 {
			t.Errorf("Expected priority count >= 0, got %d", stat.PriorityCounts[priority])
		}
	}
}
