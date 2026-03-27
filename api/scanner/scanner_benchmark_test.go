package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/photoview/photoview/api/test_utils/flags"
	"github.com/photoview/photoview/api/graphql/models"
	"github.com/photoview/photoview/api/scanner/scanner_cache"
	"github.com/photoview/photoview/api/test_utils"
	"github.com/photoview/photoview/api/test_utils/flags"
	"github.com/photoview/photoview/api/utils"
)

var test_dbm test_utils.TestDBManager = test_utils.TestDBManager{}

// benchmarkSetup prepares the database and filesystem for benchmarking
func benchmarkSetup(b *testing.B) *test_utils.TestDBManager {
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

	// Setup filesystem cache
	utils.ConfigureTestCache(b.TempDir())

	return dbm
}

// BenchmarkFindAlbumsForUser_10 benchmarks scanning 10 albums
func BenchmarkFindAlbumsForUser_10(b *testing.B) {
	dbm := benchmarkSetup(b)
	defer dbm.Close()

	// Create test directory structure with 10 albums
	tempDir := b.TempDir()
	_ = createAlbumDirectories(b, tempDir, 10)

	// Setup user and root album
	user, err := models.RegisterUser(dbm.DB, "bench_user_10", nil, false)
	if err != nil {
		b.Fatalf("Failed to register user: %v", err)
	}

	rootAlbum := models.Album{
		Title: "root",
		Path:  tempDir,
	}
	if err := dbm.DB.Save(&rootAlbum).Error; err != nil {
		b.Fatalf("Failed to create root album: %v", err)
	}
	if err := dbm.DB.Model(user).Association("Albums").Append(&rootAlbum); err != nil {
		b.Fatalf("Failed to bind root album to user: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Clean up database before each iteration
		dbm.DB.Exec("DELETE FROM user_albums WHERE album_id > ?", rootAlbum.ID)
		dbm.DB.Exec("DELETE FROM albums WHERE id > ?", rootAlbum.ID)
		b.StartTimer()

		albumCache := scanner_cache.MakeAlbumCache()
		_, scanErrors := FindAlbumsForUser(dbm.DB, user, albumCache)

		if len(scanErrors) > 0 {
			b.Fatalf("Scanning failed with errors: %v", scanErrors)
		}
	}
}

// BenchmarkFindAlbumsForUser_100 benchmarks scanning 100 albums
func BenchmarkFindAlbumsForUser_100(b *testing.B) {
	dbm := benchmarkSetup(b)
	defer dbm.Close()

	// Create test directory structure with 100 albums
	tempDir := b.TempDir()
	_ = createAlbumDirectories(b, tempDir, 100)

	// Setup user and root album
	user, err := models.RegisterUser(dbm.DB, "bench_user_100", nil, false)
	if err != nil {
		b.Fatalf("Failed to register user: %v", err)
	}

	rootAlbum := models.Album{
		Title: "root",
		Path:  tempDir,
	}
	if err := dbm.DB.Save(&rootAlbum).Error; err != nil {
		b.Fatalf("Failed to create root album: %v", err)
	}
	if err := dbm.DB.Model(user).Association("Albums").Append(&rootAlbum); err != nil {
		b.Fatalf("Failed to bind root album to user: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Clean up database before each iteration
		dbm.DB.Exec("DELETE FROM user_albums WHERE album_id > ?", rootAlbum.ID)
		dbm.DB.Exec("DELETE FROM albums WHERE id > ?", rootAlbum.ID)
		b.StartTimer()

		albumCache := scanner_cache.MakeAlbumCache()
		_, scanErrors := FindAlbumsForUser(dbm.DB, user, albumCache)

		if len(scanErrors) > 0 {
			b.Fatalf("Scanning failed with errors: %v", scanErrors)
		}
	}
}

// BenchmarkFindAlbumsForUser_1000 benchmarks scanning 1000 albums
func BenchmarkFindAlbumsForUser_1000(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	dbm := benchmarkSetup(b)
	defer dbm.Close()

	// Create test directory structure with 1000 albums
	tempDir := b.TempDir()
	_ = createAlbumDirectories(b, tempDir, 1000)

	// Setup user and root album
	user, err := models.RegisterUser(dbm.DB, "bench_user_1000", nil, false)
	if err != nil {
		b.Fatalf("Failed to register user: %v", err)
	}

	rootAlbum := models.Album{
		Title: "root",
		Path:  tempDir,
	}
	if err := dbm.DB.Save(&rootAlbum).Error; err != nil {
		b.Fatalf("Failed to create root album: %v", err)
	}
	if err := dbm.DB.Model(user).Association("Albums").Append(&rootAlbum); err != nil {
		b.Fatalf("Failed to bind root album to user: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Clean up database before each iteration
		dbm.DB.Exec("DELETE FROM user_albums WHERE album_id > ?", rootAlbum.ID)
		dbm.DB.Exec("DELETE FROM albums WHERE id > ?", rootAlbum.ID)
		b.StartTimer()

		albumCache := scanner_cache.MakeAlbumCache()
		_, scanErrors := FindAlbumsForUser(dbm.DB, user, albumCache)

		if len(scanErrors) > 0 {
			b.Fatalf("Scanning failed with errors: %v", scanErrors)
		}
	}
}

// BenchmarkFindAlbumsForUser_Nested_10 benchmarks scanning 10 nested albums (2 levels deep)
func BenchmarkFindAlbumsForUser_Nested_10(b *testing.B) {
	dbm := benchmarkSetup(b)
	defer dbm.Close()

	// Create nested directory structure: 5 parent albums, each with 2 sub-albums = 10 total
	tempDir := b.TempDir()
	_ = createNestedAlbumDirectories(b, tempDir, 5, 2)

	// Setup user and root album
	user, err := models.RegisterUser(dbm.DB, "bench_user_nested_10", nil, false)
	if err != nil {
		b.Fatalf("Failed to register user: %v", err)
	}

	rootAlbum := models.Album{
		Title: "root",
		Path:  tempDir,
	}
	if err := dbm.DB.Save(&rootAlbum).Error; err != nil {
		b.Fatalf("Failed to create root album: %v", err)
	}
	if err := dbm.DB.Model(user).Association("Albums").Append(&rootAlbum); err != nil {
		b.Fatalf("Failed to bind root album to user: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Clean up database before each iteration
		dbm.DB.Exec("DELETE FROM user_albums WHERE album_id > ?", rootAlbum.ID)
		dbm.DB.Exec("DELETE FROM albums WHERE id > ?", rootAlbum.ID)
		b.StartTimer()

		albumCache := scanner_cache.MakeAlbumCache()
		_, scanErrors := FindAlbumsForUser(dbm.DB, user, albumCache)

		if len(scanErrors) > 0 {
			b.Fatalf("Scanning failed with errors: %v", scanErrors)
		}
	}
}

// BenchmarkFindAlbumsForUser_Nested_100 benchmarks scanning 100 nested albums (5x20 structure)
func BenchmarkFindAlbumsForUser_Nested_100(b *testing.B) {
	dbm := benchmarkSetup(b)
	defer dbm.Close()

	// Create nested directory structure: 5 parent albums, each with 20 sub-albums = 100 total
	tempDir := b.TempDir()
	_ = createNestedAlbumDirectories(b, tempDir, 5, 20)

	// Setup user and root album
	user, err := models.RegisterUser(dbm.DB, "bench_user_nested_100", nil, false)
	if err != nil {
		b.Fatalf("Failed to register user: %v", err)
	}

	rootAlbum := models.Album{
		Title: "root",
		Path:  tempDir,
	}
	if err := dbm.DB.Save(&rootAlbum).Error; err != nil {
		b.Fatalf("Failed to create root album: %v", err)
	}
	if err := dbm.DB.Model(user).Association("Albums").Append(&rootAlbum); err != nil {
		b.Fatalf("Failed to bind root album to user: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Clean up database before each iteration
		dbm.DB.Exec("DELETE FROM user_albums WHERE album_id > ?", rootAlbum.ID)
		dbm.DB.Exec("DELETE FROM albums WHERE id > ?", rootAlbum.ID)
		b.StartTimer()

		albumCache := scanner_cache.MakeAlbumCache()
		_, scanErrors := FindAlbumsForUser(dbm.DB, user, albumCache)

		if len(scanErrors) > 0 {
			b.Fatalf("Scanning failed with errors: %v", scanErrors)
		}
	}
}

// createAlbumDirectories creates a flat structure of n albums under tempDir
// Each album contains a test image file
func createAlbumDirectories(b *testing.B, tempDir string, n int) []string {
	albums := make([]string, n)

	for i := 0; i < n; i++ {
		albumPath := filepath.Join(tempDir, fmt.Sprintf("album%04d", i))
		if err := os.MkdirAll(albumPath, 0755); err != nil {
			b.Fatalf("Failed to create album directory %s: %v", albumPath, err)
		}

		// Create a test image in each album
		imagePath := filepath.Join(albumPath, "photo.jpg")
		createBenchmarkImage(b, imagePath)

		albums[i] = albumPath
	}

	return albums
}

// createNestedAlbumDirectories creates a nested structure:
// - parentCount top-level albums
// - Each parent has childrenCount sub-albums
// Total albums = parentCount * childrenCount
func createNestedAlbumDirectories(b *testing.B, tempDir string, parentCount, childrenCount int) []string {
	albums := make([]string, 0, parentCount*childrenCount)

	for i := 0; i < parentCount; i++ {
		parentPath := filepath.Join(tempDir, fmt.Sprintf("parent%04d", i))
		if err := os.MkdirAll(parentPath, 0755); err != nil {
			b.Fatalf("Failed to create parent directory %s: %v", parentPath, err)
		}

		// Create a test image in the parent album
		createBenchmarkImage(b, filepath.Join(parentPath, "photo.jpg"))

		albums = append(albums, parentPath)

		// Create sub-albums
		for j := 0; j < childrenCount; j++ {
			childPath := filepath.Join(parentPath, fmt.Sprintf("child%04d", j))
			if err := os.MkdirAll(childPath, 0755); err != nil {
				b.Fatalf("Failed to create child directory %s: %v", childPath, err)
			}

			// Create a test image in the child album
			createBenchmarkImage(b, filepath.Join(childPath, "photo.jpg"))

			albums = append(albums, childPath)
		}
	}

	return albums
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
