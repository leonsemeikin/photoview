package scanner_queue

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	_ "github.com/photoview/photoview/api/test_utils/flags"
	"github.com/photoview/photoview/api/graphql/models"
	"github.com/photoview/photoview/api/scanner/scanner_cache"
	"github.com/photoview/photoview/api/scanner/scanner_task"
	"github.com/photoview/photoview/api/test_utils"
	"github.com/photoview/photoview/api/test_utils/flags"
	"github.com/photoview/photoview/api/utils"
)

// benchmarkQueueSetup prepares a scanner queue with test albums
func benchmarkQueueSetup(b *testing.B, albumCount int) (*test_utils.TestDBManager, []models.Album) {
	if !flags.Database {
		b.Skip("Database integration tests disabled")
	}
	if !flags.Filesystem {
		b.Skip("Filesystem integration tests disabled")
	}

	// Setup database
	dbm := &test_utils.TestDBManager{}
	if err := dbm.SetupAndReset(); err != nil {
		b.Fatalf("failed to setup test database: %v", err)
	}
	utils.ConfigureTestCache(b.TempDir())

	// Create test user
	user, err := models.RegisterUser(dbm.DB, "bench_user", nil, false)
	if err != nil {
		b.Fatalf("Failed to register user: %v", err)
	}

	// Create test directory structure
	tempDir := b.TempDir()
	albums := make([]models.Album, albumCount)

	for i := 0; i < albumCount; i++ {
		albumPath := filepath.Join(tempDir, fmt.Sprintf("album%04d", i))
		if err := os.MkdirAll(albumPath, 0755); err != nil {
			b.Fatalf("Failed to create album directory %s: %v", albumPath, err)
		}

		// Create a test image in each album
		imagePath := filepath.Join(albumPath, "photo.jpg")
		createBenchmarkImage(b, imagePath)

		albums[i] = models.Album{
			Title: fmt.Sprintf("album%04d", i),
			Path:  albumPath,
		}

		if err := dbm.DB.Save(&albums[i]).Error; err != nil {
			b.Fatalf("Failed to create album %d: %v", i, err)
		}

		if err := dbm.DB.Model(user).Association("Albums").Append(&albums[i]); err != nil {
			b.Fatalf("Failed to bind album %d to user: %v", i, err)
		}
	}

	return dbm, albums
}

// BenchmarkScannerQueue_AddJob_10 benchmarks adding 10 jobs to the queue
func BenchmarkScannerQueue_AddJob_10(b *testing.B) {
	dbm, albums := benchmarkQueueSetup(b, 10)
	defer dbm.Close()

	queue := &ScannerQueue{
		idle_chan:   make(chan bool, 100),
		in_progress: make([]ScannerJob, 0),
		up_next:     make([]ScannerJob, 0),
		db:          dbm.DB,
		settings:    ScannerQueueSettings{max_concurrent_tasks: 2},
		close_chan:  nil,
		running:     true,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		queue.up_next = queue.up_next[:0]
		b.StartTimer()

		// Add jobs to queue (this is what we're benchmarking)
		albumCache := scanner_cache.MakeAlbumCache()
		for _, album := range albums {
			ctx := scanner_task.NewTaskContext(context.Background(), dbm.DB, &album, albumCache)
			job := NewScannerJob(ctx)
			queue.addJob(&job)
		}
	}
}

// BenchmarkScannerQueue_AddJob_100 benchmarks adding 100 jobs to the queue
func BenchmarkScannerQueue_AddJob_100(b *testing.B) {
	dbm, albums := benchmarkQueueSetup(b, 100)
	defer dbm.Close()

	queue := &ScannerQueue{
		idle_chan:   make(chan bool, 100),
		in_progress: make([]ScannerJob, 0),
		up_next:     make([]ScannerJob, 0),
		db:          dbm.DB,
		settings:    ScannerQueueSettings{max_concurrent_tasks: 2},
		close_chan:  nil,
		running:     true,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		queue.up_next = queue.up_next[:0]
		b.StartTimer()

		albumCache := scanner_cache.MakeAlbumCache()
		for _, album := range albums {
			ctx := scanner_task.NewTaskContext(context.Background(), dbm.DB, &album, albumCache)
			job := NewScannerJob(ctx)
			queue.addJob(&job)
		}
	}
}

// BenchmarkScannerQueue_AddJob_1000 benchmarks adding 1000 jobs to the queue
func BenchmarkScannerQueue_AddJob_1000(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping large benchmark in short mode")
	}

	dbm, albums := benchmarkQueueSetup(b, 1000)
	defer dbm.Close()

	queue := &ScannerQueue{
		idle_chan:   make(chan bool, 100),
		in_progress: make([]ScannerJob, 0),
		up_next:     make([]ScannerJob, 0),
		db:          dbm.DB,
		settings:    ScannerQueueSettings{max_concurrent_tasks: 2},
		close_chan:  nil,
		running:     true,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		queue.up_next = queue.up_next[:0]
		b.StartTimer()

		albumCache := scanner_cache.MakeAlbumCache()
		for _, album := range albums {
			ctx := scanner_task.NewTaskContext(context.Background(), dbm.DB, &album, albumCache)
			job := NewScannerJob(ctx)
			queue.addJob(&job)
		}
	}
}

// BenchmarkScannerQueue_JobOnQueue_10 benchmarks checking if jobs are on queue (10 items)
func BenchmarkScannerQueue_JobOnQueue_10(b *testing.B) {
	dbm, albums := benchmarkQueueSetup(b, 10)
	defer dbm.Close()

	queue := &ScannerQueue{
		idle_chan:   make(chan bool, 100),
		in_progress: make([]ScannerJob, 0),
		up_next:     make([]ScannerJob, 0),
		db:          dbm.DB,
		settings:    ScannerQueueSettings{max_concurrent_tasks: 2},
		close_chan:  nil,
		running:     true,
	}

	// Add jobs to queue
	albumCache := scanner_cache.MakeAlbumCache()
	for _, album := range albums {
		ctx := scanner_task.NewTaskContext(context.Background(), dbm.DB, &album, albumCache)
		job := NewScannerJob(ctx)
		queue.addJob(&job)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, album := range albums {
			ctx := scanner_task.NewTaskContext(context.Background(), dbm.DB, &album, albumCache)
			job := NewScannerJob(ctx)
			_, _ = queue.jobOnQueue(&job)
		}
	}
}

// BenchmarkScannerQueue_JobOnQueue_100 benchmarks checking if jobs are on queue (100 items)
func BenchmarkScannerQueue_JobOnQueue_100(b *testing.B) {
	dbm, albums := benchmarkQueueSetup(b, 100)
	defer dbm.Close()

	queue := &ScannerQueue{
		idle_chan:   make(chan bool, 100),
		in_progress: make([]ScannerJob, 0),
		up_next:     make([]ScannerJob, 0),
		db:          dbm.DB,
		settings:    ScannerQueueSettings{max_concurrent_tasks: 2},
		close_chan:  nil,
		running:     true,
	}

	// Add jobs to queue
	albumCache := scanner_cache.MakeAlbumCache()
	for _, album := range albums {
		ctx := scanner_task.NewTaskContext(context.Background(), dbm.DB, &album, albumCache)
		job := NewScannerJob(ctx)
		queue.addJob(&job)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, album := range albums {
			ctx := scanner_task.NewTaskContext(context.Background(), dbm.DB, &album, albumCache)
			job := NewScannerJob(ctx)
			_, _ = queue.jobOnQueue(&job)
		}
	}
}

// BenchmarkScannerQueue_Notify benchmarks the notify function
func BenchmarkScannerQueue_Notify(b *testing.B) {
	queue := &ScannerQueue{
		idle_chan:   make(chan bool, 100),
		in_progress: make([]ScannerJob, 0),
		up_next:     make([]ScannerJob, 0),
		db:          nil,
		settings:    ScannerQueueSettings{max_concurrent_tasks: 2},
		close_chan:  nil,
		running:     true,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		queue.notify()
	}
}

// BenchmarkScannerQueue_ConcurrentAdds benchmarks concurrent job additions
func BenchmarkScannerQueue_ConcurrentAdds(b *testing.B) {
	dbm, albums := benchmarkQueueSetup(b, 100)
	defer dbm.Close()

	queue := &ScannerQueue{
		idle_chan:   make(chan bool, 100),
		in_progress: make([]ScannerJob, 0),
		up_next:     make([]ScannerJob, 0),
		db:          dbm.DB,
		settings:    ScannerQueueSettings{max_concurrent_tasks: 4},
		close_chan:  nil,
		running:     true,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		queue.up_next = queue.up_next[:0]
		var ops uint64
		b.StartTimer()

		// Concurrent additions (4 goroutines adding 25 jobs each = 100 total)
		var wg sync.WaitGroup
		for g := 0; g < 4; g++ {
			wg.Add(1)
			go func(startIdx int) {
				defer wg.Done()
				albumCache := scanner_cache.MakeAlbumCache()
				for j := 0; j < 25; j++ {
					albumIdx := startIdx + j
					ctx := scanner_task.NewTaskContext(context.Background(), dbm.DB, &albums[albumIdx], albumCache)
					job := NewScannerJob(ctx)
					queue.addJob(&job)
					atomic.AddUint64(&ops, 1)
				}
			}(g * 25)
		}
		wg.Wait()
	}
}

// createBenchmarkImage creates a minimal valid JPEG file for benchmarking
func createBenchmarkImage(b *testing.B, path string) {
	// Minimal JPEG header (1x1 pixel black image)
	minimalJPEG := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46,
		0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01,
		0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43,
		0x00, 0x03, 0x02, 0x02, 0x03, 0x02, 0x02, 0x03,
		0x03, 0x03, 0x03, 0x04, 0x03, 0x03, 0x04, 0x05,
		0x08, 0x05, 0x05, 0x04, 0x04, 0x05, 0x0A, 0x07,
		0x07, 0x06, 0x08, 0x0C, 0x0C, 0x0C, 0x0B, 0x0A,
		0x0B, 0x0B, 0x0D, 0x0E, 0x12, 0x10, 0x0D, 0x0E,
		0x11, 0x0E, 0x0B, 0x0B, 0x10, 0x16, 0x10, 0x11,
		0x13, 0x14, 0x15, 0x15, 0x15, 0x0C, 0x0F, 0x17,
		0x18, 0x16, 0x14, 0x18, 0x12, 0x14, 0x15, 0x14,
		0xFF, 0xC0, 0x00, 0x0B, 0x08, 0x00, 0x01, 0x00,
		0x01, 0x01, 0x01, 0x11, 0x00, 0xFF, 0xC4, 0x00,
		0x14, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x0A, 0xFF, 0xC4, 0x00, 0x14,
		0x10, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0xFF, 0xDA, 0x00, 0x08,
		0x01, 0x01, 0x00, 0x00, 0x3F, 0x00, 0x37, 0xFF,
		0xD9,
	}

	err := os.WriteFile(path, minimalJPEG, 0644)
	if err != nil {
		b.Fatalf("Failed to create test image at %s: %v", path, err)
	}
}
