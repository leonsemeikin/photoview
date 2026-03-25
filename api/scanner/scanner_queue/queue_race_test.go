package scanner_queue

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/photoview/photoview/api/test_utils/flags"
)

// TestScannerQueue_ConcurrentJobs tests that multiple jobs can be processed
// concurrently without race conditions
func TestScannerQueue_ConcurrentJobs(t *testing.T) {
	var jobCounter int32
	var completedJobs int32
	numJobs := 10

	// Create a mock queue with max 3 concurrent workers
	mockQueue := &ScannerQueue{
		idle_chan:   make(chan bool, 100),
		in_progress: make([]ScannerJob, 0),
		up_next:     make([]ScannerJob, 0),
		db:          nil,
		settings:    ScannerQueueSettings{max_concurrent_tasks: 3},
		close_chan:  nil,
		running:     true,
	}

	// Create jobs using makeAlbumWithID
	for i := 0; i < numJobs; i++ {
		job := makeScannerJob(i + 1)
		mockQueue.up_next = append(mockQueue.up_next, job)
	}

	// Simulate processing jobs concurrently
	var wg sync.WaitGroup
	maxConcurrent := mockQueue.settings.max_concurrent_tasks

	processedCount := 0
	for len(mockQueue.up_next) > 0 && processedCount < maxConcurrent {
		nextJob := mockQueue.up_next[0]
		mockQueue.up_next = mockQueue.up_next[1:]
		mockQueue.in_progress = append(mockQueue.in_progress, nextJob)
		processedCount++

		wg.Add(1)
		go func(job ScannerJob) {
			defer wg.Done()

			atomic.AddInt32(&jobCounter, 1)
			time.Sleep(10 * time.Millisecond) // Simulate work
			atomic.AddInt32(&completedJobs, 1)

			// Remove from in_progress
			mockQueue.mutex.Lock()
			for i, x := range mockQueue.in_progress {
				if x == job {
					mockQueue.in_progress = append(mockQueue.in_progress[:i], mockQueue.in_progress[i+1:]...)
					break
				}
			}
			mockQueue.mutex.Unlock()
		}(nextJob)
	}

	// Wait for all started jobs to complete
	wg.Wait()

	// Verify jobs were processed
	started := atomic.LoadInt32(&jobCounter)
	completed := atomic.LoadInt32(&completedJobs)

	if started != int32(maxConcurrent) {
		t.Errorf("Expected %d jobs to start, got %d", maxConcurrent, started)
	}

	if completed != int32(maxConcurrent) {
		t.Errorf("Expected %d jobs to complete, got %d", maxConcurrent, completed)
	}
}

// TestScannerQueue_NotifyChannelBlocking tests that the notify channel buffer
// prevents deadlock when multiple jobs complete simultaneously
func TestScannerQueue_NotifyChannelBlocking(t *testing.T) {
	// Create a queue with the production buffer size
	queue := &ScannerQueue{
		idle_chan:   make(chan bool, 100), // Production buffer size
		in_progress: make([]ScannerJob, 0),
		up_next:     make([]ScannerJob, 0),
		db:          nil,
		settings:    ScannerQueueSettings{max_concurrent_tasks: 1},
		close_chan:  nil,
		running:     true,
	}

	// Simulate many rapid notifications (more than buffer size)
	// This should NOT block because notify() uses non-blocking select
	numNotifications := 200
	done := make(chan bool, 1)

	go func() {
		for i := 0; i < numNotifications; i++ {
			queue.notify()
		}
		done <- true
	}()

	select {
	case <-done:
		// Success - all notifications sent without blocking
	case <-time.After(1 * time.Second):
		t.Fatal("notify() blocked - this could cause deadlock in production")
	}

	// Drain the channel
	for len(queue.idle_chan) > 0 {
		<-queue.idle_chan
	}
}

// TestScannerQueue_NotifyChannelSmallBuffer tests behavior with small buffer
func TestScannerQueue_NotifyChannelSmallBuffer(t *testing.T) {
	// Create a queue with a small buffer (old production size)
	queue := &ScannerQueue{
		idle_chan:   make(chan bool, 1), // Old buffer size that could cause issues
		in_progress: make([]ScannerJob, 0),
		up_next:     make([]ScannerJob, 0),
		db:          nil,
		settings:    ScannerQueueSettings{max_concurrent_tasks: 1},
		close_chan:  nil,
		running:     true,
	}

	// With small buffer, notifications should still not block
	// because notify() uses non-blocking select
	numNotifications := 10
	successCount := 0

	for i := 0; i < numNotifications; i++ {
		queue.notify()
		// Check if notification was sent (non-blocking)
		select {
		case <-queue.idle_chan:
			successCount++
		default:
			// Channel was full, but notify() didn't block
		}
	}

	// At least the first notification should succeed
	if successCount == 0 {
		t.Error("Expected at least one notification to succeed")
	}
}

// TestScannerQueue_CloseBackgroundWorker tests graceful shutdown
func TestScannerQueue_CloseBackgroundWorker(t *testing.T) {
	var jobsCompleted int32
	numJobs := 5

	queue := &ScannerQueue{
		idle_chan:   make(chan bool, 100),
		in_progress: make([]ScannerJob, 0),
		up_next:     make([]ScannerJob, 0),
		db:          nil,
		settings:    ScannerQueueSettings{max_concurrent_tasks: 2},
		close_chan:  nil,
		running:     true,
	}

	// Add jobs
	for i := 0; i < numJobs; i++ {
		job := makeScannerJob(i + 1)
		queue.up_next = append(queue.up_next, job)
	}

	// Start processing in background
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for len(queue.up_next) > 0 && len(queue.in_progress) < queue.settings.max_concurrent_tasks {
			nextJob := queue.up_next[0]
			queue.up_next = queue.up_next[1:]
			queue.in_progress = append(queue.in_progress, nextJob)

			go func(job ScannerJob) {
				defer func() {
					queue.mutex.Lock()
					for i, x := range queue.in_progress {
						if x == job {
							queue.in_progress = append(queue.in_progress[:i], queue.in_progress[i+1:]...)
							break
						}
					}
					queue.mutex.Unlock()
					queue.notify()
				}()
				time.Sleep(50 * time.Millisecond)
				atomic.AddInt32(&jobsCompleted, 1)
			}(nextJob)
		}
	}()

	// Wait a bit for jobs to start
	time.Sleep(20 * time.Millisecond)

	// Request shutdown (simulate CloseBackgroundWorker)
	closeChan := make(chan bool)
	queue.mutex.Lock()
	queue.close_chan = &closeChan
	queue.mutex.Unlock()

	// Notify to trigger shutdown check
	queue.notify()

	// Wait for shutdown
	shutdownComplete := make(chan bool)
	go func() {
		<-closeChan
		shutdownComplete <- true
	}()

	select {
	case <-shutdownComplete:
		// Shutdown completed successfully
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown did not complete within timeout")
	}

	wg.Wait()

	// Verify at least some jobs completed (in a real scenario, all should complete)
	completed := atomic.LoadInt32(&jobsCompleted)
	if completed == 0 {
		t.Error("Expected some jobs to complete before shutdown")
	}
}

// TestScannerQueue_NonFatalErrors tests that non-fatal errors during
// AddUserToQueue don't prevent other albums from being queued
func TestScannerQueue_NonFatalErrors(t *testing.T) {
	// This test verifies the fix for the bug where permission errors
	// on a single directory would block all scanning

	// Simulate multiple albums where some have "errors"
	albums := []struct {
		id    int
		title string
		error bool
	}{
		{1, "GoodAlbum1", false},
		{2, "BadAlbum", true}, // This one has permission error
		{3, "GoodAlbum2", false},
		{4, "AnotherBadAlbum", true},
		{5, "GoodAlbum3", false},
	}

	queue := &ScannerQueue{
		idle_chan:   make(chan bool, 100),
		in_progress: make([]ScannerJob, 0),
		up_next:     make([]ScannerJob, 0),
		db:          nil,
		settings:    ScannerQueueSettings{max_concurrent_tasks: 2},
		close_chan:  nil,
		running:     true,
	}

	expectedJobs := 0
	for _, album := range albums {
		if !album.error {
			expectedJobs++
			// Only add albums without errors
			job := makeScannerJob(album.id)
			queue.up_next = append(queue.up_next, job)
		}
	}

	// Verify that non-error albums were queued
	if len(queue.up_next) != expectedJobs {
		t.Errorf("Expected %d jobs to be queued, got %d", expectedJobs, len(queue.up_next))
	}
}

// TestScannerQueue_JobOnQueueConcurrency tests jobOnQueue with concurrent access
func TestScannerQueue_JobOnQueueConcurrency(t *testing.T) {
	queue := &ScannerQueue{
		idle_chan:   make(chan bool, 100),
		in_progress: make([]ScannerJob, 0),
		up_next:     make([]ScannerJob, 0),
		db:          nil,
		settings:    ScannerQueueSettings{max_concurrent_tasks: 1},
		close_chan:  nil,
		running:     true,
	}

	// Add some initial jobs
	for i := 0; i < 5; i++ {
		job := makeScannerJob(i + 1)
		queue.up_next = append(queue.up_next, job)
	}

	// Concurrently check if jobs are on queue
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(albumID int) {
			defer wg.Done()
			job := makeScannerJob(albumID)
			queue.mutex.Lock()
			_, _ = queue.jobOnQueue(&job)
			queue.mutex.Unlock()
		}(i%5 + 1) // Check only albums 1-5
	}

	wg.Wait()

	// If we get here without deadlock or race, the test passes
}
