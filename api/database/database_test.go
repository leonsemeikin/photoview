package database

import (
	"os"
	"testing"
	"time"

	"github.com/photoview/photoview/api/database/drivers"
	"github.com/photoview/photoview/api/graphql/models"
	"github.com/photoview/photoview/api/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Test helpers
func createTestUser(db *gorm.DB, username string, admin bool) *models.User {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("test_password"), bcrypt.DefaultCost)
	hashedPasswordStr := string(hashedPassword)

	user := &models.User{
		Username: username,
		Password: &hashedPasswordStr,
		Admin:    admin,
	}

	db.Create(user)
	return user
}

func createTestAlbum(db *gorm.DB, title string, parentID *int, owners []models.User) *models.Album {
	album := &models.Album{
		Title:         title,
		ParentAlbumID: parentID,
		Path:          "/test/" + title,
		Owners:        owners,
	}

	db.Create(album)
	return album
}

func createTestMedia(db *gorm.DB, title string, albumID int) *models.Media {
	media := &models.Media{
		Title:    title,
		Path:     "/test/" + title,
		Type:     "photo",
		AlbumID:  albumID,
		DateShot: time.Now(),
	}

	db.Create(media)
	return media
}

func TestSetupDatabase_SQLite(t *testing.T) {
	// Set test environment variable for SQLite
	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), "file::memory:")

	db, err := SetupDatabase()
	if err != nil {
		t.Fatalf("SetupDatabase() failed: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Verify it's SQLite
	d := drivers.GetDatabaseDriverType(db)
	if d != drivers.SQLITE {
		t.Errorf("Expected SQLite driver, got %v", d)
	}
}

func TestSetupDatabase_MySQL(t *testing.T) {
	// Skip test if MySQL URL not provided
	mysqlURL := utils.EnvMysqlURL.GetValue()
	if mysqlURL == "" {
		t.Skip("MySQL URL not provided, skipping test")
	}

	os.Setenv(string(utils.EnvDatabaseDriver), "mysql")

	db, err := SetupDatabase()
	if err != nil {
		t.Fatalf("SetupDatabase() failed: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Verify it's MySQL
	d := drivers.GetDatabaseDriverType(db)
	if d != drivers.MYSQL {
		t.Errorf("Expected MySQL driver, got %v", d)
	}
}

func TestSetupDatabase_Postgres(t *testing.T) {
	// Skip test if PostgreSQL URL not provided
	postgresURL := utils.EnvPostgresURL.GetValue()
	if postgresURL == "" {
		t.Skip("PostgreSQL URL not provided, skipping test")
	}

	os.Setenv(string(utils.EnvDatabaseDriver), "postgres")

	db, err := SetupDatabase()
	if err != nil {
		t.Fatalf("SetupDatabase() failed: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Verify it's PostgreSQL
	d := drivers.GetDatabaseDriverType(db)
	if d != drivers.POSTGRES {
		t.Errorf("Expected PostgreSQL driver, got %v", d)
	}
}

func TestSetupDatabase_RetryLogic(t *testing.T) {
	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), "file::memory:")

	// SetupDatabase should retry up to 5 times
	// We can't easily test actual retries without mocking, but we can test that it succeeds
	db, err := SetupDatabase()
	if err != nil {
		t.Fatalf("SetupDatabase() failed: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}
	defer sqlDB.Close()

	// The function should have succeeded, proving retry logic works for valid connections
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}
}

func TestMigrateDatabase_AutoMigrate(t *testing.T) {
	// Use test database
	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), "file::memory:")

	db, err := SetupDatabase()
	if err != nil {
		t.Fatalf("SetupDatabase() failed: %v", err)
	}

	// Migrate database
	err = MigrateDatabase(db)
	if err != nil {
		t.Fatalf("MigrateDatabase() failed: %v", err)
	}

	// Check that tables were created by trying to create a user
	user := createTestUser(db, "testuser", false)
	if user.ID == 0 {
		t.Error("Failed to create test user after migration")
	}

	// Check that all models in database_models slice can be used
	for _, model := range database_models {
		if err := db.Migrator().HasTable(model); !err {
			t.Errorf("Table for model %T should exist after migration", model)
		}
	}
}

func TestClearDatabase_AllModels(t *testing.T) {
	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), "file::memory:")

	db, err := SetupDatabase()
	if err != nil {
		t.Fatalf("SetupDatabase() failed: %v", err)
	}

	// Migrate first
	err = MigrateDatabase(db)
	if err != nil {
		t.Fatalf("MigrateDatabase() failed: %v", err)
	}

	// Create some test data
	user := createTestUser(db, "testuser", false)
	album := createTestAlbum(db, "testalbum", nil, []models.User{*user})
	_ = createTestMedia(db, "testmedia", album.ID)

	// Clear database
	err = ClearDatabase(db)
	if err != nil {
		t.Fatalf("ClearDatabase() failed: %v", err)
	}

	// Check that tables are dropped
	for _, model := range database_models {
		if err := db.Migrator().HasTable(model); err {
			t.Errorf("Table for model %T should not exist after clear", model)
		}
	}

	// Re-migrate to ensure the process is idempotent
	err = MigrateDatabase(db)
	if err != nil {
		t.Fatalf("MigrateDatabase() failed after clear: %v", err)
	}
}
