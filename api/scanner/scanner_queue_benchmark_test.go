package scanner

import (
	"context"
	"testing"
	"time"

	"github.com/photoview/photoview/api/database"
	"github.com/photoview/photoview/api/graphql/models"
	"github.com/stretchr/testify/assert"
)

func BenchmarkScannerQueue_Process_10Jobs(b *testing.B) {
	benchmarkScannerQueueProcess(b, 10)
}

func BenchmarkScannerQueue_Process_100Jobs(b *testing.B) {
	benchmarkScannerQueueProcess(b, 100)
}

func BenchmarkScannerQueue_Process_1000Jobs(b *testing.B) {
	benchmarkScannerQueueProcess(b, 1000)
}

func benchmarkScannerQueueProcess(b *testing.B, numJobs int) {
	if !*databaseFlag {
		b.Skip("database tests disabled")
	}

	db, err := database.TestDatabase()
	if err != nil {
		b.Fatalf("failed to get test database: %v", err)
	}

	user := models.User{
		Username: "benchmark_user",
		Admin:    false,
	}
	if err := db.Create(&user).Error; err != nil {
		b.Fatalf("failed to create user: %v", err)
	}

	rootAlbum := models.Album{
		Title:   "Root Album",
		Path:    "/benchmark/root",
		PathHash: "benchmark_root_hash",
		OwnerID: user.ID,
	}
	if err := db.Create(&rootAlbum).Error; err != nil {
		b.Fatalf("failed to create root album: %v", err)
	}

	albums := make([]*models.Album, numJobs)
	for i := 0; i < numJobs; i++ {
		album := &models.Album{
			Title:         "Benchmark Album",
			Path:          "/benchmark/root/album_" + string(rune(i)),
			PathHash:      "benchmark_album_hash_" + string(rune(i)),
			ParentAlbumID: &rootAlbum.ID,
			OwnerID:       user.ID,
		}
		albums[i] = album
		if err := db.Create(album).Error; err != nil {
			b.Fatalf("failed to create album %d: %v", i, err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		queue, err := NewScannerQueue(db, context.Background())
		if err != nil {
			b.Fatalf("failed to create scanner queue: %v", err)
		}
		defer queue.Close()

		for _, album := range albums {
			job := NewScanAlbumJob(album.ID, user.ID)
			if err := queue.AddJob(job); err != nil {
				b.Fatalf("failed to add job: %v", err)
			}
		}

		done := make(chan struct{})
		go func() {
			for {
				job, err := queue.GetNextJob()
				if err != nil {
					break
				}
				if job == nil {
					break
				}
				queue.DoneJob(job)
			}
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(30 * time.Second):
			b.Fatalf("benchmark timed out")
		}
	}

	b.StopTimer()

	for _, album := range albums {
		db.Unscoped().Delete(album)
	}
	db.Unscoped().Delete(&rootAlbum)
	db.Unscoped().Delete(&user)
}

func BenchmarkScannerQueue_ConcurrentJobs(b *testing.B) {
	if !*databaseFlag {
		b.Skip("database tests disabled")
	}

	db, err := database.TestDatabase()
	if err != nil {
		b.Fatalf("failed to get test database: %v", err)
	}

	user := models.User{
		Username: "concurrent_user",
		Admin:    false,
	}
	if err := db.Create(&user).Error; err != nil {
		b.Fatalf("failed to create user: %v", err)
	}

	album := models.Album{
		Title:   "Concurrent Album",
		Path:    "/concurrent",
		PathHash: "concurrent_hash",
		OwnerID: user.ID,
	}
	if err := db.Create(&album).Error; err != nil {
		b.Fatalf("failed to create album: %v", err)
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			queue, err := NewScannerQueue(db, context.Background())
			if err != nil {
				b.Fatal(err)
			}
			defer queue.Close()

			job := NewScanAlbumJob(album.ID, user.ID)
			if err := queue.AddJob(job); err != nil {
				b.Fatal(err)
			}

			retrievedJob, err := queue.GetNextJob()
			if err != nil {
				b.Fatal(err)
			}
			if retrievedJob == nil {
				b.Fatal("expected job but got nil")
			}
			assert.Equal(b, job.ID, retrievedJob.ID)
			assert.Equal(b, job.AlbumID, retrievedJob.AlbumID)
			assert.Equal(b, job.UserID, retrievedJob.UserID)

			if err := queue.DoneJob(retrievedJob); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.StopTimer()

	db.Unscoped().Delete(&album)
	db.Unscoped().Delete(&user)
}