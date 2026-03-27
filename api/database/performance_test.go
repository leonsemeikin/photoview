package database

import (
	"fmt"
	"os"
	"testing"

	_ "github.com/photoview/photoview/api/test_utils/flags"
	"github.com/photoview/photoview/api/graphql/models"
	"github.com/photoview/photoview/api/utils"
)

func BenchmarkDatabase_SQLite_Insert(b *testing.B) {
	// Setup test database with unique file for each benchmark run
	dbPath := fmt.Sprintf("/tmp/photoview_bench_insert_%d.db", os.Getpid())
	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), dbPath)
	defer os.Remove(dbPath)

	db, err := SetupDatabase()
	if err != nil {
		b.Fatalf("failed to setup test database: %v", err)
	}
	defer closeDB(db)

	if err := MigrateDatabase(db); err != nil {
		b.Fatalf("failed to migrate database: %v", err)
	}

	// Create test album once
	album := models.Album{
		Title:    "Benchmark Insert Album",
		Path:     "/test/path_insert",
		PathHash: "test_hash_insert",
	}
	if err := db.Create(&album).Error; err != nil {
		b.Fatalf("failed to create album: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		media := models.Media{
			Title:    fmt.Sprintf("Media Insert %d", i),
			Path:     fmt.Sprintf("/test/path_insert/media_%d.jpg", i),
			PathHash: fmt.Sprintf("hash_insert_%d", i),
			Type:     "photo",
			AlbumID:  album.ID,
		}

		if err := db.Create(&media).Error; err != nil {
			b.Errorf("failed to insert media: %v", err)
		}
	}
}

func BenchmarkDatabase_SQLite_Select_Indexed(b *testing.B) {
	// Setup test database with unique file
	dbPath := fmt.Sprintf("/tmp/photoview_bench_indexed_%d.db", os.Getpid())
	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), dbPath)
	defer os.Remove(dbPath)

	db, err := SetupDatabase()
	if err != nil {
		b.Fatalf("failed to setup test database: %v", err)
	}
	defer closeDB(db)

	if err := MigrateDatabase(db); err != nil {
		b.Fatalf("failed to migrate database: %v", err)
	}

	// Create test album once
	album := models.Album{
		Title:    "Benchmark Indexed Album",
		Path:     "/test/path_indexed",
		PathHash: "test_hash_indexed",
	}
	if err := db.Create(&album).Error; err != nil {
		b.Fatalf("failed to create album: %v", err)
	}

	// Insert test data
	for i := 0; i < 1000; i++ {
		media := models.Media{
			Title:    fmt.Sprintf("Media Indexed %d", i),
			Path:     fmt.Sprintf("/test/path_indexed/media_%d.jpg", i),
			PathHash: fmt.Sprintf("hash_indexed_%d", i),
			Type:     "photo",
			AlbumID:  album.ID,
		}
		if err := db.Create(&media).Error; err != nil {
			b.Fatalf("failed to insert media %d: %v", i, err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var media models.Media
		// Query using indexed field (PathHash)
		if err := db.Where("path_hash = ?", "hash_indexed_42").First(&media).Error; err != nil {
			b.Errorf("failed to query media: %v", err)
		}
	}
}

func BenchmarkDatabase_SQLite_Select_FullScan(b *testing.B) {
	// Setup test database with unique file
	dbPath := fmt.Sprintf("/tmp/photoview_bench_fullscan_%d.db", os.Getpid())
	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), dbPath)
	defer os.Remove(dbPath)

	db, err := SetupDatabase()
	if err != nil {
		b.Fatalf("failed to setup test database: %v", err)
	}
	defer closeDB(db)

	if err := MigrateDatabase(db); err != nil {
		b.Fatalf("failed to migrate database: %v", err)
	}

	// Create test album once
	album := models.Album{
		Title:    "Benchmark FullScan Album",
		Path:     "/test/path_fullscan",
		PathHash: "test_hash_fullscan",
	}
	if err := db.Create(&album).Error; err != nil {
		b.Fatalf("failed to create album: %v", err)
	}

	// Insert test data
	for i := 0; i < 1000; i++ {
		media := models.Media{
			Title:    fmt.Sprintf("Media FullScan %d", i),
			Path:     fmt.Sprintf("/test/path_fullscan/media_%d.jpg", i),
			PathHash: fmt.Sprintf("hash_fullscan_%d", i),
			Type:     "photo",
			AlbumID:  album.ID,
		}
		if err := db.Create(&media).Error; err != nil {
			b.Fatalf("failed to insert media %d: %v", i, err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var media models.Media
		// Query using non-indexed field (Title) - will cause full scan
		if err := db.Where("title = ?", "Media FullScan 42").First(&media).Error; err != nil {
			b.Errorf("failed to query media: %v", err)
		}
	}
}

func BenchmarkDatabase_SQLite_Transaction_Commit(b *testing.B) {
	// Setup test database with unique file
	dbPath := fmt.Sprintf("/tmp/photoview_bench_commit_%d.db", os.Getpid())
	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), dbPath)
	defer os.Remove(dbPath)

	db, err := SetupDatabase()
	if err != nil {
		b.Fatalf("failed to setup test database: %v", err)
	}
	defer closeDB(db)

	if err := MigrateDatabase(db); err != nil {
		b.Fatalf("failed to migrate database: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tx := db.Begin()
		if tx == nil {
			b.Fatal("failed to begin transaction")
		}

		user := models.User{
			Username: fmt.Sprintf("bench_user_commit_%d", i),
			Admin:    false,
		}
		if err := tx.Create(&user).Error; err != nil {
			tx.Rollback()
			b.Errorf("failed to create user: %v", err)
			continue
		}

		if err := tx.Commit().Error; err != nil {
			b.Errorf("failed to commit transaction: %v", err)
		}
	}
}

func BenchmarkDatabase_SQLite_Transaction_Rollback(b *testing.B) {
	// Setup test database with unique file
	dbPath := fmt.Sprintf("/tmp/photoview_bench_rollback_%d.db", os.Getpid())
	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), dbPath)
	defer os.Remove(dbPath)

	db, err := SetupDatabase()
	if err != nil {
		b.Fatalf("failed to setup test database: %v", err)
	}
	defer closeDB(db)

	if err := MigrateDatabase(db); err != nil {
		b.Fatalf("failed to migrate database: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tx := db.Begin()
		if tx == nil {
			b.Fatal("failed to begin transaction")
		}

		user := models.User{
			Username: fmt.Sprintf("bench_user_rollback_%d", i),
			Admin:    false,
		}
		if err := tx.Create(&user).Error; err != nil {
			tx.Rollback()
			b.Errorf("failed to create user: %v", err)
			continue
		}

		if err := tx.Rollback().Error; err != nil {
			b.Errorf("failed to rollback transaction: %v", err)
		}
	}
}

func BenchmarkDatabase_SQLite_Connection_Pool(b *testing.B) {
	// Setup test database with unique file
	dbPath := fmt.Sprintf("/tmp/photoview_bench_pool_%d.db", os.Getpid())
	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), dbPath)
	defer os.Remove(dbPath)

	db, err := SetupDatabase()
	if err != nil {
		b.Fatalf("failed to setup test database: %v", err)
	}
	defer closeDB(db)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var result int
		if err := db.Raw("SELECT 1").Scan(&result).Error; err != nil {
			b.Errorf("failed to execute query: %v", err)
		}
	}
}

func BenchmarkDatabase_SQLite_WAL_Read(b *testing.B) {
	// Setup test database with unique file
	dbPath := fmt.Sprintf("/tmp/photoview_bench_wal_%d.db", os.Getpid())
	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), dbPath)
	defer os.Remove(dbPath)

	db, err := SetupDatabase()
	if err != nil {
		b.Fatalf("failed to setup test database: %v", err)
	}
	defer closeDB(db)

	if err := MigrateDatabase(db); err != nil {
		b.Fatalf("failed to migrate database: %v", err)
	}

	// Enable WAL mode
	if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		b.Fatalf("failed to enable WAL mode: %v", err)
	}

	// Insert test data
	album := models.Album{
		Title:    "Benchmark WAL Album",
		Path:     "/test/path_wal",
		PathHash: "test_hash_wal",
	}
	if err := db.Create(&album).Error; err != nil {
		b.Fatalf("failed to create album: %v", err)
	}

	for i := 0; i < 1000; i++ {
		media := models.Media{
			Title:    fmt.Sprintf("Media WAL %d", i),
			Path:     fmt.Sprintf("/test/path_wal/media_%d.jpg", i),
			PathHash: fmt.Sprintf("hash_wal_%d", i),
			Type:     "photo",
			AlbumID:  album.ID,
		}
		if err := db.Create(&media).Error; err != nil {
			b.Fatalf("failed to insert media %d: %v", i, err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var count int64
		if err := db.Model(&models.Media{}).Count(&count).Error; err != nil {
			b.Errorf("failed to count media: %v", err)
		}
	}
}

// Helper function to close database connection
func closeDB(db interface{}) {
	// No-op, database will be closed when file is deleted
}
