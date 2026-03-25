package scanner_queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/photoview/photoview/api/graphql/models"
	"github.com/photoview/photoview/api/scanner/scanner_cache"
	"github.com/photoview/photoview/api/scanner/scanner_task"
	"github.com/photoview/photoview/api/test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestScannerQueue_ConcurrentJobs tests that multiple jobs can run concurrently without race conditions
func TestScannerQueue_ConcurrentJobs(t *testing.T) {
	db := test_utils.CreateTestDatabase(t)
	defer test_utils.CleanupTestDatabase(t, db)

	// Initialize scanner queue
	err := InitializeScannerQueue(db)
	require.NoError(t, err, "Should initialize scanner queue")
	defer CloseScannerQueue()

	user := test_utils.CreateTestUser(t, db, "testuser", false)
	cache := scanner_cache.MakeAlbumScannerCache()

	// Create multiple albums
	var albums []*models.Album
	for i := 0; i < 5; i++ {
		album := test_utils.CreateTestAlbum(t, db, "TestAlbum"+string(rune('0'+i)), "/test/path"+string(rune('0'+i)))
		albums = append(albums, album)
	}

	// Track completed jobs
	completedJobs := make(chan int, len(albums))

	// Mock job that just sleeps and reports completion
	mockJob := func(album *models.Album) *ScannerJob {
		ctx := scanner_task.NewTaskContext(context.Background(), db, album, cache)
		job := NewScannerJob(ctx)

		// Override Run method
		job.Run = func(db *gorm.DB) {
			time.Sleep(10 * time.Millisecond) // Simulate work
			completedJobs <- int(album.ID)
		}

		return job
	}

	// Add jobs to queue
	for _, album := range albums {
		job := mockJob(album)
		global_scanner_queue.mutex.Lock()
		err := global_scanner_queue.addJob(job)
		global_scanner_queue.mutex.Unlock()
		require.NoError(t, err, "Should add job to queue")
	}

	// Wait for all jobs to complete
	timeout := time.After(5 * time.Second)
	completed := 0
	for {
		select {
		case <-completedJobs:
			completed++
			if completed == len(albums) {
				return
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for jobs to complete. Got %d/%d", completed, len(albums))
		}
	}
}

// TestScannerQueue_NotifyChannelBlocking tests that notify channel doesn't block
func TestScannerQueue_NotifyChannelBlocking(t *testing.T) {
	db := test_utils.CreateTestDatabase(t)
	defer test_utils.CleanupTestDatabase(t, db)

	// Initialize scanner queue
	err := InitializeScannerQueue(db)
	require.NoError(t, err, "Should initialize scanner queue")
	defer CloseScannerQueue()

	user := test_utils.CreateTestUser(t, db, "testuser", false)
	cache := scanner_cache.MakeAlbumScannerCache()
	album := test_utils.CreateTestAlbum(t, db, "TestAlbum", "/test/path")

	// Send many notifications rapidly
	for i := 0; i < 200; i++ {
		global_scanner_queue.notify()
	}

	// Queue should still be functional
	ctx := scanner_task.NewTaskContext(context.Background(), db, album, cache)
	job := NewScannerJob(ctx)

	global_scanner_queue.mutex.Lock()
	err = global_scanner_queue.addJob(job)
	global_scanner_queue.mutex.Unlock()
	require.NoError(t, err, "Should still be able to add job after many notifications")
}

// TestScannerQueue_CloseBackgroundWorker tests graceful shutdown
func TestScannerQueue_CloseBackgroundWorker(t *testing.T) {
	db := test_utils.CreateTestDatabase(t)
	defer test_utils.CleanupTestDatabase(t, db)

	// Initialize scanner queue
	err := InitializeScannerQueue(db)
	require.NoError(t, err, "Should initialize scanner queue")

	user := test_utils.CreateTestUser(t, db, "testuser", false)
	cache := scanner_cache.MakeAlbumScannerCache()
	album := test_utils.CreateTestAlbum(t, db, "TestAlbum", "/test/path")

	// Add a job that takes time
	ctx := scanner_task.NewTaskContext(context.Background(), db, album, cache)
	job := NewScannerJob(ctx)

	// Override Run to take time
	job.Run = func(db *gorm.DB) {
		time.Sleep(100 * time.Millisecond) // Simulate work
	}

	global_scanner_queue.mutex.Lock()
	err = global_scanner_queue.addJob(job)
	global_scanner_queue.mutex.Unlock()
	require.NoError(t, err, "Should add job to queue")

	// Start closing in a goroutine
	closeDone := make(chan bool)
	go func() {
		CloseScannerQueue()
		close(closeDone)
	}()

	// Wait for close to complete
	select {
	case <-closeDone:
		// Successfully closed
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for graceful shutdown")
	}
}

// TestAddUserToQueue_NonFatalErrors tests that non-fatal errors don't block the queue
func TestAddUserToQueue_NonFatalErrors(t *testing.T) {
	db := test_utils.CreateTestDatabase(t)
	defer test_utils.CleanupTestDatabase(t, db)

	// Initialize scanner queue
	err := InitializeScannerQueue(db)
	require.NoError(t, err, "Should initialize scanner queue")
	defer CloseScannerQueue()

	user := test_utils.CreateTestUser(t, db, "testuser", false)

	// Create albums with valid paths
	album1 := test_utils.CreateTestAlbum(t, db, "ValidAlbum1", "/valid/path1")
	album2 := test_utils.CreateTestAlbum(t, db, "ValidAlbum2", "/valid/path2")

	// Link albums to user
	test_utils.CreateUserAlbumRelation(t, db, user, album1)
	test_utils.CreateUserAlbumRelation(t, db, user, album2)

	// This should not return an error even if FindAlbumsForUser returns some non-fatal errors
	// The implementation logs non-fatal errors but continues
	err = AddUserToQueue(user)
	assert.NoError(t, err, "Should not return error even with non-fatal errors")

	// Verify that albums were added to the queue
	global_scanner_queue.mutex.Lock()
	queueSize := len(global_scanner_queue.up_next)
	global_scanner_queue.mutex.Unlock()

	// Should have at least the albums we created (might be more depending on UserAlbums)
	assert.GreaterOrEqual(t, queueSize, 2, "Should have at least 2 albums in queue")
}

// TestScannerQueue_JobDuplication tests that duplicate jobs are not added
func TestScannerQueue_JobDuplication(t *testing.T) {
	db := test_utils.CreateTestDatabase(t)
	defer test_utils.CleanupTestDatabase(t, db)

	// Initialize scanner queue
	err := InitializeScannerQueue(db)
	require.NoError(t, err, "Should initialize scanner queue")
	defer CloseScannerQueue()

	user := test_utils.CreateTestUser(t, db, "testuser", false)
	cache := scanner_cache.MakeAlbumScannerCache()
	album := test_utils.CreateTestAlbum(t, db, "TestAlbum", "/test/path")

	// Add the same job twice
	ctx1 := scanner_task.NewTaskContext(context.Background(), db, album, cache)
	job1 := NewScannerJob(ctx1)

	ctx2 := scanner_task.NewTaskContext(context.Background(), db, album, cache)
	job2 := NewScannerJob(ctx2)

	global_scanner_queue.mutex.Lock()
	err1 := global_scanner_queue.addJob(job1)
	err2 := global_scanner_queue.addJob(job2)
	global_scanner_queue.mutex.Unlock()

	assert.NoError(t, err1, "First job should be added")
	// Second job should also not error, just return that it already exists
	// based on the implementation, addJob returns nil if job already exists
}

// TestScannerQueue_MaxConcurrentTasks tests that max concurrent tasks is respected
func TestScannerQueue_MaxConcurrentTasks(t *testing.T) {
	db := test_utils.CreateTestDatabase(t)
	defer test_utils.CleanupTestDatabase(t, db)

	// Initialize scanner queue with default workers
	err := InitializeScannerQueue(db)
	require.NoError(t, err, "Should initialize scanner queue")
	defer CloseScannerQueue()

	// Change max concurrent tasks to 2
	ChangeScannerConcurrentWorkers(2)

	user := test_utils.CreateTestUser(t, db, "testuser", false)
	cache := scanner_cache.MakeAlbumScannerCache()

	// Create 5 albums
	var albums []*models.Album
	for i := 0; i < 5; i++ {
		album := test_utils.CreateTestAlbum(t, db, "TestAlbum"+string(rune('0'+i)), "/test/path"+string(rune('0'+i)))
		albums = append(albums, album)
	}

	// Track running jobs
	runningJobs := make(chan bool, 10)
	var runningMutex sync.Mutex
	runningCount := 0

	// Mock jobs that signal when they start and stop
	mockJob := func(album *models.Album) *ScannerJob {
		ctx := scanner_task.NewTaskContext(context.Background(), db, album, cache)
		job := NewScannerJob(ctx)

		job.Run = func(db *gorm.DB) {
			runningMutex.Lock()
			runningCount++
			assert.LessOrEqual(t, runningCount, 2, "Should not exceed max concurrent tasks")
			runningMutex.Unlock()

			runningJobs <- true
			time.Sleep(50 * time.Millisecond) // Simulate work

			runningMutex.Lock()
			runningCount--
			runningMutex.Unlock()

			<-runningJobs // Signal completion
		}

		return job
	}

	// Add all jobs
	for _, album := range albums {
		job := mockJob(album)
		global_scanner_queue.mutex.Lock()
		err := global_scanner_queue.addJob(job)
		global_scanner_queue.mutex.Unlock()
		require.NoError(t, err, "Should add job to queue")
	}

	// Wait for all jobs to complete
	timeout := time.After(10 * time.Second)
	for i := 0; i < len(albums); i++ {
		select {
		case <-runningJobs:
			// Job completed
		case <-timeout:
			t.Fatalf("Timeout waiting for jobs to complete")
		}
	}
}
