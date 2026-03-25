package database

import "flag"


var _ = flag.Bool("database", false, "run database integration tests")
var _ = flag.Bool("filesystem", false, "run filesystem integration tests")

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/photoview/photoview/api/graphql/models"
	"github.com/photoview/photoview/api/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestSetupDatabase_SQLite tests SQLite database connection
func TestSetupDatabase_SQLite(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}

	// Set environment for SQLite
	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), "file::memory:?cache=shared")

	db, err := SetupDatabase()
	require.NoError(t, err, "SetupDatabase should not return error")
	require.NotNil(t, db, "Database connection should not be nil")

	// Verify connection works
	sqlDB, err := db.DB()
	require.NoError(t, err, "Should get underlying sql.DB")
	defer sqlDB.Close()

	ctx, cancel := contextWithTimeout()
	defer cancel()

	err = sqlDB.PingContext(ctx)
	assert.NoError(t, err, "Should be able to ping database")
}

// TestSetupDatabase_MySQL tests MySQL database connection
func TestSetupDatabase_MySQL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MySQL test in short mode")
	}

	mysqlURL := os.Getenv("TEST_MYSQL_URL")
	if mysqlURL == "" {
		t.Skip("TEST_MYSQL_URL not set, skipping MySQL test")
	}

	os.Setenv(string(utils.EnvDatabaseDriver), "mysql")
	os.Setenv(string(utils.EnvMysqlURL), mysqlURL)

	db, err := SetupDatabase()
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
		return
	}
	require.NotNil(t, db, "Database connection should not be nil")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()

	ctx, cancel := contextWithTimeout()
	defer cancel()

	err = sqlDB.PingContext(ctx)
	assert.NoError(t, err, "Should be able to ping MySQL database")
}

// TestSetupDatabase_Postgres tests PostgreSQL database connection
func TestSetupDatabase_Postgres(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL test in short mode")
	}

	postgresURL := os.Getenv("TEST_POSTGRES_URL")
	if postgresURL == "" {
		t.Skip("TEST_POSTGRES_URL not set, skipping PostgreSQL test")
	}

	os.Setenv(string(utils.EnvDatabaseDriver), "postgres")
	os.Setenv(string(utils.EnvPostgresURL), postgresURL)

	db, err := SetupDatabase()
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
		return
	}
	require.NotNil(t, db, "Database connection should not be nil")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()

	ctx, cancel := contextWithTimeout()
	defer cancel()

	err = sqlDB.PingContext(ctx)
	assert.NoError(t, err, "Should be able to ping PostgreSQL database")
}

// TestSetupDatabase_RetryLogic tests that database connection retries properly
func TestSetupDatabase_RetryLogic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping retry test in short mode")
	}

	// Use invalid connection to test retry logic
	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), "/invalid/path/to/db.sqlite")

	start := time.Now()
	db, err := SetupDatabase()

	// Should eventually return db (even if connection fails) after retries
	// The implementation returns db, nil at the end regardless
	elapsed := time.Since(start)

	// Should have taken at least 4 * 5 seconds = 20 seconds for retries
	// But we skip the exact timing check in short mode
	assert.GreaterOrEqual(t, elapsed, 20*time.Second,
		"Should retry for at least 20 seconds (4 retries * 5 seconds)")

	// db might be non-nil even if connection failed
	_ = db
	_ = err
}

// TestMigrateDatabase_AutoMigrate tests automatic migration
func TestMigrateDatabase_AutoMigrate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping migration test in short mode")
	}

	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), "file::memory:?cache=shared")

	db, err := SetupDatabase()
	require.NoError(t, err)
	defer closeDB(t, db)

	err = MigrateDatabase(db)
	assert.NoError(t, err, "Migration should succeed")

	// Verify that some tables exist by checking if they can be queried
	// Check User table
	userTableExists := db.Migrator().HasTable(&models.User{})
	assert.True(t, userTableExists, "User table should exist")

	// Check Album table
	albumTableExists := db.Migrator().HasTable(&models.Album{})
	assert.True(t, albumTableExists, "Album table should exist")

	// Check Media table
	mediaTableExists := db.Migrator().HasTable(&models.Media{})
	assert.True(t, mediaTableExists, "Media table should exist")
}

// TestClearDatabase_AllModels tests that all tables are dropped
func TestClearDatabase_AllModels(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping clear database test in short mode")
	}

	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), "file::memory:?cache=shared")

	db, err := SetupDatabase()
	require.NoError(t, err)
	defer closeDB(t, db)

	// First migrate to create tables
	err = MigrateDatabase(db)
	require.NoError(t, err)

	// Verify tables exist
	assert.True(t, db.Migrator().HasTable(&models.User{}), "User table should exist before clear")
	assert.True(t, db.Migrator().HasTable(&models.Album{}), "Album table should exist before clear")

	// Clear database
	err = ClearDatabase(db)
	assert.NoError(t, err, "ClearDatabase should succeed")

	// Verify tables are dropped
	assert.False(t, db.Migrator().HasTable(&models.User{}), "User table should not exist after clear")
	assert.False(t, db.Migrator().HasTable(&models.Album{}), "Album table should not exist after clear")
}

// TestSetupDatabase_Concurrency tests concurrent database connections
func TestSetupDatabase_Concurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), "file::memory:?cache=shared")

	errors := make(chan error, 10)

	// Create 10 concurrent connections
	for i := 0; i < 10; i++ {
		go func() {
			db, err := SetupDatabase()
			if err != nil {
				errors <- err
				return
			}

			sqlDB, err := db.DB()
			if err != nil {
				errors <- err
				return
			}

			ctx, cancel := contextWithTimeout()
			defer cancel()

			err = sqlDB.PingContext(ctx)
			errors <- err
			sqlDB.Close()
		}()
	}

	// Check all connections succeeded
	for i := 0; i < 10; i++ {
		err := <-errors
		assert.NoError(t, err, "Concurrent connection should succeed")
	}
}

// TestSetupDatabase_ConnectionPool tests connection pool settings
func TestSetupDatabase_ConnectionPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping connection pool test in short mode")
	}

	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), "file::memory:?cache=shared")

	db, err := SetupDatabase()
	require.NoError(t, err)
	defer closeDB(t, db)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	// Check that MaxOpenConns is set
	maxOpenConns := sqlDB.Stats().MaxOpenConnections
	assert.Equal(t, 80, maxOpenConns, "MaxOpenConns should be set to 80")
}

// Helper functions

func contextWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func closeDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}
}
