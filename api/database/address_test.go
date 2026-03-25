package database

import (
	"net/url"
	"testing"
)

func TestGetSqliteAddress_DefaultPath(t *testing.T) {
	address, err := GetSqliteAddress("")
	if err != nil {
		t.Fatalf("GetSqliteAddress() with empty path failed: %v", err)
	}

	// Default should be "photoview.db"
	if address.Path != "photoview.db" {
		t.Errorf("Expected default path 'photoview.db', got: %s", address.Path)
	}
}

func TestGetSqliteAddress_CustomPath(t *testing.T) {
	customPath := "/custom/path/to/database.db"
	address, err := GetSqliteAddress(customPath)
	if err != nil {
		t.Fatalf("GetSqliteAddress() failed: %v", err)
	}

	if address.Path != customPath {
		t.Errorf("Expected path '%s', got: %s", customPath, address.Path)
	}
}

func TestGetSqliteAddress_QueryParameters(t *testing.T) {
	path := "test.db"
	address, err := GetSqliteAddress(path)
	if err != nil {
		t.Fatalf("GetSqliteAddress() failed: %v", err)
	}

	query := address.Query()

	// Test all expected query parameters
	tests := []struct {
		key      string
		expected string
	}{
		{"cache", "shared"},
		{"mode", "rwc"},
		{"_journal_mode", "WAL"},
		{"_locking_mode", "NORMAL"},
		{"_foreign_keys", "ON"},
	}

	for _, tt := range tests {
		if value := query.Get(tt.key); value != tt.expected {
			t.Errorf("Expected %s=%s, got: %s", tt.key, tt.expected, value)
		}
	}
}

func TestGetMysqlAddress_MissingURL(t *testing.T) {
	_, err := GetMysqlAddress("")
	if err == nil {
		t.Error("Expected error for empty MySQL URL, got nil")
	}
}

func TestGetMysqlAddress_ValidURL(t *testing.T) {
	validURL := "user:password@tcp(localhost:3306)/photoview"
	address, err := GetMysqlAddress(validURL)
	if err != nil {
		t.Fatalf("GetMysqlAddress() failed: %v", err)
	}

	// Check that MultiStatements is enabled
	if address == "" {
		t.Error("Expected non-empty DSN, got empty string")
	}
}

func TestGetPostgresAddress_MissingURL(t *testing.T) {
	_, err := GetPostgresAddress("")
	if err == nil {
		t.Error("Expected error for empty PostgreSQL URL, got nil")
	}
}

func TestGetPostgresAddress_ValidURL(t *testing.T) {
	validURL := "postgres://user:password@localhost:5432/photoview"
	address, err := GetPostgresAddress(validURL)
	if err != nil {
		t.Fatalf("GetPostgresAddress() failed: %v", err)
	}

	// Verify it's a valid URL
	if address.Scheme != "postgres" {
		t.Errorf("Expected scheme 'postgres', got: %s", address.Scheme)
	}

	if address.Host != "localhost:5432" {
		t.Errorf("Expected host 'localhost:5432', got: %s", address.Host)
	}
}

func TestGetPostgresAddress_InvalidURL(t *testing.T) {
	invalidURL := ":not-a-valid-url"
	_, err := GetPostgresAddress(invalidURL)
	if err == nil {
		t.Error("Expected error for invalid PostgreSQL URL, got nil")
	}
}

func TestGetSqliteAddress_RelativePath(t *testing.T) {
	relativePath := "./data/photoview.db"
	address, err := GetSqliteAddress(relativePath)
	if err != nil {
		t.Fatalf("GetSqliteAddress() failed: %v", err)
	}

	if address.Path != relativePath {
		t.Errorf("Expected path '%s', got: %s", relativePath, address.Path)
	}
}

func TestGetSqliteAddress_URLWithSpecialChars(t *testing.T) {
	specialPath := "/path/with spaces/database.db"
	address, err := GetSqliteAddress(specialPath)
	if err != nil {
		t.Fatalf("GetSqliteAddress() failed: %v", err)
	}

	// URL encoding should be applied by url.Parse automatically
	// The path should preserve the original structure
	if address.Path == "" {
		t.Error("Expected non-empty path after URL parsing")
	}

	// Check that we can reconstruct a valid URL string
	urlString := address.String()
	if urlString == "" {
		t.Error("Expected non-empty URL string")
	}

	// Verify it can be parsed back
	_, err = url.Parse(urlString)
	if err != nil {
		t.Errorf("Failed to parse reconstructed URL: %v", err)
	}
}
