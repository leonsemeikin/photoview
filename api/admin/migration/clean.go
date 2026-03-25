package migration

import (
	"github.com/photoview/photoview/api/database"
)

// CleanPath clears the photo directory
func CleanPath(dir string) error {
	db, err := database.SetupDatabase()
	if err != nil {
		return err
	}

	return database.ClearDatabase(db)
}

// CleanPath clears the photo directory
func CleanAlbums() error {
	db, err := database.SetupDatabase()
	if err != nil {
		return err
	}

	// Delete all albums
	return db.Exec("DELETE FROM albums").Error
}

// ClearUsers clears all users from the database
func ClearUsers() error {
	db, err := database.SetupDatabase()
	if err != nil {
		return err
	}

	// Delete all users
	return db.Exec("DELETE FROM users").Error
}
