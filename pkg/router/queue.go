package router

import (
	"context"
	"sync"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/backends"
)

// BackendQueue tracks pending requests for a backend
type BackendQueue struct {
	mu             sync.RWMutex
	pending        int                    // Current pending requests
	priorityCounts [4]int                 // Count per priority level
	lastUpdate     time.Time
}

// QueueManager manages all backend queues
type QueueManager struct {
	mu     sync.RWMutex
	queues map[string]*BackendQueue // backend ID -> queue
}

// NewQueueManager creates a new queue manager
func NewQueueManager() *QueueManager {
	return &QueueManager{
		queues: make(map[string]*BackendQueue),
	}
}

// GetQueueDepth returns pending request count with priority weighting
// Higher priority requests contribute more to the weighted depth
func (qm *QueueManager) GetQueueDepth(backendID string, priority backends.Priority) int {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	queue, exists := qm.queues[backendID]
	if !exists {
		return 0
	}

	queue.mu.RLock()
	defer queue.mu.RUnlock()

	// Return weighted depth (higher priority = more weight)
	// This makes backends with high-priority requests appear busier
	weighted := 0
	for p := backends.Priority(0); p <= priority; p++ {
		weighted += queue.priorityCounts[p] * (int(p) + 1)
	}

	return weighted
}

// GetRawQueueDepth returns the actual number of pending requests
func (qm *QueueManager) GetRawQueueDepth(backendID string) int {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	queue, exists := qm.queues[backendID]
	if !exists {
		return 0
	}

	queue.mu.RLock()
	defer queue.mu.RUnlock()

	return queue.pending
}

// GetPriorityBreakdown returns count of requests at each priority level
func (qm *QueueManager) GetPriorityBreakdown(backendID string) [4]int {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	queue, exists := qm.queues[backendID]
	if !exists {
		return [4]int{}
	}

	queue.mu.RLock()
	defer queue.mu.RUnlock()

	return queue.priorityCounts
}

// MarkRequestStart increments queue depth
func (qm *QueueManager) MarkRequestStart(backendID string, priority backends.Priority) {
	qm.mu.Lock()
	queue, exists := qm.queues[backendID]
	if !exists {
		queue = &BackendQueue{
			lastUpdate: time.Now(),
		}
		qm.queues[backendID] = queue
	}
	qm.mu.Unlock()

	queue.mu.Lock()
	queue.pending++
	if int(priority) < len(queue.priorityCounts) {
		queue.priorityCounts[priority]++
	}
	queue.lastUpdate = time.Now()
	queue.mu.Unlock()
}

// MarkRequestEnd decrements queue depth
func (qm *QueueManager) MarkRequestEnd(backendID string, priority backends.Priority) {
	qm.mu.RLock()
	queue, exists := qm.queues[backendID]
	qm.mu.RUnlock()

	if !exists {
		return
	}

	queue.mu.Lock()
	queue.pending--
	if queue.pending < 0 {
		queue.pending = 0 // Safety check
	}

	if int(priority) < len(queue.priorityCounts) {
		queue.priorityCounts[priority]--
		if queue.priorityCounts[priority] < 0 {
			queue.priorityCounts[priority] = 0 // Safety check
		}
	}

	queue.lastUpdate = time.Now()
	queue.mu.Unlock()
}

// GetAllQueueStats returns queue statistics for all backends
func (qm *QueueManager) GetAllQueueStats() map[string]QueueStats {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	stats := make(map[string]QueueStats)
	for backendID, queue := range qm.queues {
		queue.mu.RLock()
		stats[backendID] = QueueStats{
			BackendID:      backendID,
			Pending:        queue.pending,
			PriorityCounts: queue.priorityCounts,
			LastUpdate:     queue.lastUpdate,
		}
		queue.mu.RUnlock()
	}

	return stats
}

// QueueStats represents queue statistics for a backend
type QueueStats struct {
	BackendID      string
	Pending        int
	PriorityCounts [4]int
	LastUpdate     time.Time
}

// QueueTrackingBackend wraps a backend to automatically track queue depth
type QueueTrackingBackend struct {
	backends.Backend
	queueMgr *QueueManager
	priority backends.Priority
}

// Generate wraps the underlying backend's Generate to track queue depth
func (qtb *QueueTrackingBackend) Generate(ctx context.Context, req *backends.GenerateRequest) (*backends.GenerateResponse, error) {
	defer qtb.queueMgr.MarkRequestEnd(qtb.Backend.ID(), qtb.priority)
	return qtb.Backend.Generate(ctx, req)
}

// GenerateStream wraps the underlying backend's GenerateStream to track queue depth
func (qtb *QueueTrackingBackend) GenerateStream(ctx context.Context, req *backends.GenerateRequest) (backends.StreamReader, error) {
	reader, err := qtb.Backend.GenerateStream(ctx, req)
	if err != nil {
		qtb.queueMgr.MarkRequestEnd(qtb.Backend.ID(), qtb.priority)
		return nil, err
	}

	// Wrap reader to mark end when stream closes
	return &trackingStreamReader{
		StreamReader: reader,
		onClose: func() {
			qtb.queueMgr.MarkRequestEnd(qtb.Backend.ID(), qtb.priority)
		},
	}, nil
}

// trackingStreamReader wraps a StreamReader to call onClose when closed
type trackingStreamReader struct {
	backends.StreamReader
	onClose func()
	closed  bool
	mu      sync.Mutex
}

// Close calls the underlying Close and the onClose callback
func (tsr *trackingStreamReader) Close() error {
	tsr.mu.Lock()
	if !tsr.closed {
		tsr.closed = true
		if tsr.onClose != nil {
			tsr.onClose()
		}
	}
	tsr.mu.Unlock()

	return tsr.StreamReader.Close()
}
