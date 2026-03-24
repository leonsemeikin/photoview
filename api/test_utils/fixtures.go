package test_utils

import (
	"os"
	"testing"

	"github.com/photoview/photoview/api/database"
	"github.com/photoview/photoview/api/graphql/models"
	"github.com/photoview/photoview/api/utils"
	"gorm.io/gorm"
)

// CreateTestDatabase creates a test database instance based on environment
// Uses environment variables to determine which database driver to use
// Returns a gorm.DB instance that must be cleaned up after use
func CreateTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()

	// Set test environment variables if not already set
	if os.Getenv(utils.EnvDatabaseDriver.GetName()) == "" {
		os.Setenv(utils.EnvDatabaseDriver.GetName(), "sqlite")
	}
	if os.Getenv(utils.EnvSqlitePath.GetName()) == "" {
		os.Setenv(utils.EnvSqlitePath.GetName(), "file::memory:?cache=shared")
	}

	// Use logger that doesn't output in tests
	db, err := database.SetupDatabase()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Auto migrate tables for testing
	if err := database.MigrateDatabase(db); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

// CleanupTestDatabase drops all tables and closes the database connection
func CleanupTestDatabase(t *testing.T, db *gorm.DB) {
	t.Helper()

	if err := database.ClearDatabase(db); err != nil {
		t.Errorf("Failed to cleanup test database: %v", err)
	}

	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}
}

// CreateTestUser creates a test user with the given username and admin status
func CreateTestUser(t *testing.T, db *gorm.DB, username string, admin bool) *models.User {
	t.Helper()

	password := "test-password"
	user := &models.User{
		Username: username,
		Admin:    admin,
		Password: &password,
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

// CreateTestAlbum creates a test album with the given title
func CreateTestAlbum(t *testing.T, db *gorm.DB, title string, path string) *models.Album {
	t.Helper()

	album := &models.Album{
		Title: title,
		Path:  path,
	}

	if err := db.Create(album).Error; err != nil {
		t.Fatalf("Failed to create test album: %v", err)
	}

	return album
}

// CreateUserAlbumRelation creates a relation between user and album
func CreateUserAlbumRelation(t *testing.T, db *gorm.DB, user *models.User, album *models.Album) {
	t.Helper()

	userAlbum := &models.UserAlbums{
		UserID:  user.ID,
		AlbumID: album.ID,
	}

	if err := db.Create(userAlbum).Error; err != nil {
		t.Fatalf("Failed to create user-album relation: %v", err)
	}
}
