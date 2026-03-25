package database

import (
	"os"
	"testing"

	"github.com/photoview/photoview/api/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestGetSqliteAddress_DefaultPath tests SQLite address generation with default path
func TestGetSqliteAddress_DefaultPath(t *testing.T) {
	address, err := GetSqliteAddress("")
	require.NoError(t, err, "Should not return error for default path")
	require.NotNil(t, address, "Address should not be nil")

	// Check that default path is "photoview.db"
	assert.Equal(t, "photoview.db", address.Path, "Should use default filename")
	assert.Contains(t, address.String(), "photoview.db", "Should contain default filename")
}

// TestGetSqliteAddress_CustomPath tests SQLite address generation with custom path
func TestGetSqliteAddress_CustomPath(t *testing.T) {
	customPath := "/custom/path/database.sqlite"
	address, err := GetSqliteAddress(customPath)
	require.NoError(t, err, "Should not return error for custom path")
	require.NotNil(t, address, "Address should not be nil")

	assert.Equal(t, customPath, address.Path, "Should parse custom path correctly")
	assert.Contains(t, address.String(), customPath, "Should contain custom path")
}

// TestGetSqliteAddress_WALMode tests that WAL mode is enabled
func TestGetSqliteAddress_WALMode(t *testing.T) {
	address, err := GetSqliteAddress("test.db")
	require.NoError(t, err, "Should not return error")
	require.NotNil(t, address, "Address should not be nil")

	query := address.Query()
	assert.Equal(t, "shared", query.Get("cache"), "Cache should be shared")
	assert.Equal(t, "rwc", query.Get("mode"), "Mode should be rwc")
	assert.Equal(t, "WAL", query.Get("_journal_mode"), "Journal mode should be WAL")
	assert.Equal(t, "NORMAL", query.Get("_locking_mode"), "Locking mode should be NORMAL")
	assert.Equal(t, "ON", query.Get("_foreign_keys"), "Foreign keys should be ON")
}

// TestGetSqliteAddress_InvalidPath tests error handling for invalid path
func TestGetSqliteAddress_InvalidPath(t *testing.T) {
	// Using a path that cannot be parsed as URL
	invalidPath := "://invalid/path"
	_, err := GetSqliteAddress(invalidPath)
	assert.Error(t, err, "Should return error for invalid path")
	assert.Contains(t, err.Error(), "could not parse sqlite url", "Error message should mention parsing")
}

// TestGetMysqlAddress_EmptyString tests MySQL address with empty string
func TestGetMysqlAddress_EmptyString(t *testing.T) {
	_, err := GetMysqlAddress("")
	assert.Error(t, err, "Should return error for empty address")
	assert.Contains(t, err.Error(), "missing", "Error should mention missing environment variable")
}

// TestGetMysqlAddress_ValidAddress tests MySQL address parsing
func TestGetMysqlAddress_ValidAddress(t *testing.T) {
	validDSN := "user:password@tcp(localhost:3306)/database?charset=utf8mb4"
	address, err := GetMysqlAddress(validDSN)
	require.NoError(t, err, "Should not return error for valid DSN")

	// The returned address should be the DSN with MultiStatements and ParseTime enabled
	assert.Contains(t, address, "multiStatements=true", "Should enable multi statements")
	assert.Contains(t, address, "parseTime=true", "Should enable parse time")
}

// TestGetMysqlAddress_InvalidAddress tests MySQL address with invalid format
func TestGetMysqlAddress_InvalidAddress(t *testing.T) {
	invalidDSN := "invalid:mysql://dsn"
	_, err := GetMysqlAddress(invalidDSN)
	assert.Error(t, err, "Should return error for invalid DSN")
	assert.Contains(t, err.Error(), "could not parse mysql url", "Error should mention parsing")
}

// TestGetPostgresAddress_EmptyString tests PostgreSQL address with empty string
func TestGetPostgresAddress_EmptyString(t *testing.T) {
	_, err := GetPostgresAddress("")
	assert.Error(t, err, "Should return error for empty address")
	assert.Contains(t, err.Error(), "missing", "Error should mention missing environment variable")
}

// TestGetPostgresAddress_ValidAddress tests PostgreSQL address parsing
func TestGetPostgresAddress_ValidAddress(t *testing.T) {
	validURL := "postgres://user:password@localhost:5432/database"
	address, err := GetPostgresAddress(validURL)
	require.NoError(t, err, "Should not return error for valid URL")
	require.NotNil(t, address, "Address should not be nil")

	assert.Equal(t, "postgres", address.Scheme, "Scheme should be postgres")
	assert.Equal(t, "localhost", address.Hostname(), "Host should be localhost")
	assert.Equal(t, "5432", address.Port(), "Port should be 5432")
	assert.Equal(t, "/database", address.Path, "Path should be /database")
}

// TestGetPostgresAddress_InvalidAddress tests PostgreSQL address with invalid format
func TestGetPostgresAddress_InvalidAddress(t *testing.T) {
	invalidURL := "://invalid:url"
	_, err := GetPostgresAddress(invalidURL)
	assert.Error(t, err, "Should return error for invalid URL")
	assert.Contains(t, err.Error(), "could not parse postgres url", "Error should mention parsing")
}

// TestConfigureDatabase_SQLite tests ConfigureDatabase with SQLite
func TestConfigureDatabase_SQLite(t *testing.T) {
	// Set environment variables using os.Setenv
	os.Setenv(string(utils.EnvDatabaseDriver), "sqlite")
	os.Setenv(string(utils.EnvSqlitePath), "file::memory:?cache=shared")

	db, err := ConfigureDatabase(&gorm.Config{})
	require.NoError(t, err, "Should not return error for SQLite")
	require.NotNil(t, db, "Database should not be nil")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.Close()
}

// TestGetSqliteAddress_QueryParameters tests that query parameters are correctly set
func TestGetSqliteAddress_QueryParameters(t *testing.T) {
	address, err := GetSqliteAddress("test.db")
	require.NoError(t, err)

	// Parse the query parameters
	query := address.Query()

	// Check all required parameters
	assert.Equal(t, "shared", query.Get("cache"), "Cache should be shared")
	assert.Equal(t, "rwc", query.Get("mode"), "Mode should be rwc")
	assert.Equal(t, "WAL", query.Get("_journal_mode"), "Journal mode should be WAL")
	assert.Equal(t, "NORMAL", query.Get("_locking_mode"), "Locking mode should be NORMAL")
	assert.Equal(t, "ON", query.Get("_foreign_keys"), "Foreign keys should be ON")
}

// TestGetSqliteAddress_ComplexPath tests SQLite address with complex path
func TestGetSqliteAddress_ComplexPath(t *testing.T) {
	complexPath := "/var/lib/photoview/database.sqlite"
	address, err := GetSqliteAddress(complexPath)
	require.NoError(t, err)
	require.NotNil(t, address)

	assert.Equal(t, complexPath, address.Path, "Should preserve complex path")
	assert.Contains(t, address.String(), complexPath, "Should contain complex path in string representation")
}
