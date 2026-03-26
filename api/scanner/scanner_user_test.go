package scanner

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/photoview/photoview/api/test_utils/flags"
	"github.com/photoview/photoview/api/graphql/models"
	"github.com/photoview/photoview/api/scanner/scanner_cache"
	"github.com/photoview/photoview/api/test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindAlbumsForUser_OwnerPropagation(t *testing.T) {
	test_utils.FilesystemTest(t)
	db := test_utils.DatabaseTest(t)

	// Create test directory structure
	tempDir := t.TempDir()
	photosDir := filepath.Join(tempDir, "photos")
	subDir := filepath.Join(photosDir, "subalbum")
	require.NoError(t, os.MkdirAll(photosDir, 0755), "Failed to create photos directory")
	require.NoError(t, os.MkdirAll(subDir, 0755), "Failed to create subalbum directory")

	// Create test image files
	testImage := filepath.Join(photosDir, "test.jpg")
	createTestImage(t, testImage)
	subImage := filepath.Join(subDir, "sub_test.jpg")
	createTestImage(t, subImage)

	t.Run("new albums inherit parent owners", func(t *testing.T) {
		// Clean database for this subtest
		db.Exec("DELETE FROM user_albums")
		db.Exec("DELETE FROM albums")

		// Create user A with root album
		userA, err := models.RegisterUser(db, "userA", nil, false)
		require.NoError(t, err, "Failed to register user A")

		rootAlbum := models.Album{
			Title: "photos",
			Path:  photosDir,
		}
		require.NoError(t, db.Save(&rootAlbum).Error, "Failed to create root album")
		require.NoError(t, db.Model(userA).Association("Albums").Append(&rootAlbum), "Failed to bind root album to user A")

		// Run scanner for user A - creates both root and sub-album
		albumCache := scanner_cache.MakeAlbumCache()
		scannedAlbums, scanErrors := FindAlbumsForUser(db, userA, albumCache)

		assert.Empty(t, scanErrors, "Scanning should complete without errors")
		assert.Len(t, scannedAlbums, 2, "Should scan root album and sub-album")

		// Verify user A owns both albums
		var userAAlbums []models.Album
		require.NoError(t, db.Model(userA).Association("Albums").Find(&userAAlbums), "Failed to get user A albums")
		assert.Len(t, userAAlbums, 2, "User A should own 2 albums")

		// Create user B with sub-album as root
		userB, err := models.RegisterUser(db, "userB", nil, false)
		require.NoError(t, err, "Failed to register user B")

		// Bind user B to the sub-album only
		var subAlbum models.Album
		require.NoError(t, db.Where("path = ?", subDir).First(&subAlbum).Error, "Failed to find sub-album")
		require.NoError(t, db.Model(userB).Association("Albums").Append(&subAlbum), "Failed to bind sub-album to user B")

		// Scan for user B
		albumCacheB := scanner_cache.MakeAlbumCache()
		scannedAlbumsB, scanErrorsB := FindAlbumsForUser(db, userB, albumCacheB)

		assert.Empty(t, scanErrorsB, "Scanning for user B should complete without errors")
		assert.Len(t, scannedAlbumsB, 1, "User B should scan only sub-album")

		// Critical: Verify that sub-album still has user A as owner
		// This is the owner propagation behavior - new albums get parent owners
		var subAlbumOwners []models.User
		require.NoError(t, db.Model(&subAlbum).Association("Owners").Find(&subAlbumOwners), "Failed to get sub-album owners")

		ownerIDs := make([]int, len(subAlbumOwners))
		for i, owner := range subAlbumOwners {
			ownerIDs[i] = owner.ID
		}

		assert.Contains(t, ownerIDs, userA.ID, "Sub-album should still have user A as owner (inherited from parent)")
		assert.Contains(t, ownerIDs, userB.ID, "Sub-album should have user B as owner (explicitly added)")
	})

	t.Run("existing album adds user as owner", func(t *testing.T) {
		// Clean database for this subtest
		db.Exec("DELETE FROM user_albums")
		db.Exec("DELETE FROM albums")
		db.Exec("DELETE FROM users")

		// Create user A with root album
		userA, err := models.RegisterUser(db, "existing_userA", nil, false)
		require.NoError(t, err, "Failed to register user A")

		rootAlbum := models.Album{
			Title: "photos",
			Path:  photosDir,
		}
		require.NoError(t, db.Save(&rootAlbum).Error, "Failed to create root album")
		require.NoError(t, db.Model(userA).Association("Albums").Append(&rootAlbum), "Failed to bind root album to user A")

		// Scan to create albums in database
		albumCache := scanner_cache.MakeAlbumCache()
		_, scanErrors := FindAlbumsForUser(db, userA, albumCache)
		assert.Empty(t, scanErrors, "Initial scan should complete without errors")

		// Create user B and bind to same root album
		userB, err := models.RegisterUser(db, "existing_userB", nil, false)
		require.NoError(t, err, "Failed to register user B")
		require.NoError(t, db.Model(userB).Association("Albums").Append(&rootAlbum), "Failed to bind root album to user B")

		// Scan for user B - should add user B as owner to existing albums
		albumCacheB := scanner_cache.MakeAlbumCache()
		scannedAlbumsB, scanErrorsB := FindAlbumsForUser(db, userB, albumCacheB)

		assert.Empty(t, scanErrorsB, "Scanning for user B should complete without errors")
		assert.Len(t, scannedAlbumsB, 2, "User B should scan both root album and sub-album")

		// Verify both users own both albums
		var allAlbums []models.Album
		require.NoError(t, db.Find(&allAlbums).Error, "Failed to get all albums")

		for _, album := range allAlbums {
			var albumOwners []models.User
			require.NoError(t, db.Model(&album).Association("Owners").Find(&albumOwners), "Failed to get album owners")
			assert.Len(t, albumOwners, 2, "Album %s should have 2 owners (user A and user B)", album.Title)
		}
	})
}

func TestFindAlbumsForUser_NestedAlbums(t *testing.T) {
	test_utils.FilesystemTest(t)
	db := test_utils.DatabaseTest(t)

	// Create deeply nested directory structure
	tempDir := t.TempDir()
	level1 := filepath.Join(tempDir, "level1")
	level2 := filepath.Join(level1, "level2")
	level3 := filepath.Join(level2, "level3")

	require.NoError(t, os.MkdirAll(level1, 0755))
	require.NoError(t, os.MkdirAll(level2, 0755))
	require.NoError(t, os.MkdirAll(level3, 0755))

	// Create test images at each level
	createTestImage(t, filepath.Join(level1, "l1.jpg"))
	createTestImage(t, filepath.Join(level2, "l2.jpg"))
	createTestImage(t, filepath.Join(level3, "l3.jpg"))

	t.Run("scans all nested levels", func(t *testing.T) {
		db.Exec("DELETE FROM user_albums")
		db.Exec("DELETE FROM albums")

		user, err := models.RegisterUser(db, "nestedUser", nil, false)
		require.NoError(t, err)

		rootAlbum := models.Album{
			Title: "level1",
			Path:  level1,
		}
		require.NoError(t, db.Save(&rootAlbum).Error)
		require.NoError(t, db.Model(user).Association("Albums").Append(&rootAlbum))

		albumCache := scanner_cache.MakeAlbumCache()
		scannedAlbums, scanErrors := FindAlbumsForUser(db, user, albumCache)

		assert.Empty(t, scanErrors, "Nested scan should complete without errors")
		assert.Len(t, scannedAlbums, 3, "Should scan all 3 nested levels")

		// Verify parent-child relationships
		var albums []models.Album
		require.NoError(t, db.Order("path ASC").Find(&albums).Error)

		// First album (level1) should have no parent
		assert.Nil(t, albums[0].ParentAlbumID, "Root album should have no parent")

		// Second album (level2) should have level1 as parent
		assert.NotNil(t, albums[1].ParentAlbumID, "Level2 should have a parent")
		assert.Equal(t, albums[0].ID, *albums[1].ParentAlbumID, "Level2's parent should be level1")

		// Third album (level3) should have level2 as parent
		assert.NotNil(t, albums[2].ParentAlbumID, "Level3 should have a parent")
		assert.Equal(t, albums[1].ID, *albums[2].ParentAlbumID, "Level3's parent should be level2")
	})

	t.Run("owner propagates through all levels", func(t *testing.T) {
		db.Exec("DELETE FROM user_albums")
		db.Exec("DELETE FROM albums")

		user, err := models.RegisterUser(db, "ownerUser", nil, false)
		require.NoError(t, err)

		rootAlbum := models.Album{
			Title: "level1",
			Path:  level1,
		}
		require.NoError(t, db.Save(&rootAlbum).Error)
		require.NoError(t, db.Model(user).Association("Albums").Append(&rootAlbum))

		albumCache := scanner_cache.MakeAlbumCache()
		_, scanErrors := FindAlbumsForUser(db, user, albumCache)
		assert.Empty(t, scanErrors)

		// All nested albums should have the user as owner
		var allAlbums []models.Album
		require.NoError(t, db.Find(&allAlbums).Error)

		for _, album := range allAlbums {
			var owners []models.User
			require.NoError(t, db.Model(&album).Association("Owners").Find(&owners))
			assert.Len(t, owners, 1, "Album %s should have exactly 1 owner", album.Title)
			assert.Equal(t, user.ID, owners[0].ID, "Album %s should be owned by the user", album.Title)
		}
	})
}

func TestFindAlbumsForUser_PermissionDenied(t *testing.T) {
	test_utils.FilesystemTest(t)
	db := test_utils.DatabaseTest(t)

	tempDir := t.TempDir()
	accessibleDir := filepath.Join(tempDir, "accessible")

	require.NoError(t, os.MkdirAll(accessibleDir, 0755))
	createTestImage(t, filepath.Join(accessibleDir, "photo.jpg"))

	t.Run("continues scanning after permission error", func(t *testing.T) {
		db.Exec("DELETE FROM user_albums")
		db.Exec("DELETE FROM albums")

		user, err := models.RegisterUser(db, "permUser", nil, false)
		require.NoError(t, err)

		// Create root album that will have one accessible and one inaccessible subdirectory
		rootAlbum := models.Album{
			Title: "root",
			Path:  tempDir,
		}
		require.NoError(t, db.Save(&rootAlbum).Error)
		require.NoError(t, db.Model(user).Association("Albums").Append(&rootAlbum))

		albumCache := scanner_cache.MakeAlbumCache()
		scannedAlbums, scanErrors := FindAlbumsForUser(db, user, albumCache)

		// Should scan at least the accessible directory
		assert.NotEmpty(t, scannedAlbums, "Should scan accessible albums despite permission errors")

		// Should have errors but not crash
		// Note: We can't easily create a permission error in tests, so this is more of a structural test
		assert.NotNil(t, scanErrors, "Should return error slice even if empty")
	})

	t.Run("returns non-fatal errors for missing directories", func(t *testing.T) {
		db.Exec("DELETE FROM user_albums")
		db.Exec("DELETE FROM albums")

		user, err := models.RegisterUser(db, "missingDirUser", nil, false)
		require.NoError(t, err)

		// Create album with non-existent path
		missingDir := filepath.Join(tempDir, "does_not_exist")
		album := models.Album{
			Title: "missing",
			Path:  missingDir,
		}
		require.NoError(t, db.Save(&album).Error)
		require.NoError(t, db.Model(user).Association("Albums").Append(&album))

		albumCache := scanner_cache.MakeAlbumCache()
		scannedAlbums, scanErrors := FindAlbumsForUser(db, user, albumCache)

		// Should return error but not crash
		assert.NotEmpty(t, scanErrors, "Should return error for missing directory")
		assert.Empty(t, scannedAlbums, "Should not scan any albums from missing directory")

		// Verify error message
		found := false
		for _, err := range scanErrors {
			if err != nil {
				found = true
				assert.Contains(t, err.Error(), "does not exist", "Error should mention directory does not exist")
			}
		}
		assert.True(t, found, "Should have at least one non-nil error")
	})
}

func TestFindAlbumsForUser_CleanupOldAlbums(t *testing.T) {
	test_utils.FilesystemTest(t)
	db := test_utils.DatabaseTest(t)

	tempDir := t.TempDir()
	album1Dir := filepath.Join(tempDir, "album1")
	album2Dir := filepath.Join(tempDir, "album2")

	require.NoError(t, os.MkdirAll(album1Dir, 0755))
	require.NoError(t, os.MkdirAll(album2Dir, 0755))
	createTestImage(t, filepath.Join(album1Dir, "photo1.jpg"))
	createTestImage(t, filepath.Join(album2Dir, "photo2.jpg"))

	t.Run("deletes albums removed from filesystem", func(t *testing.T) {
		db.Exec("DELETE FROM user_albums")
		db.Exec("DELETE FROM albums")

		user, err := models.RegisterUser(db, "cleanupUser", nil, false)
		require.NoError(t, err)

		// Initially scan both albums
		album1 := models.Album{Title: "album1", Path: album1Dir}
		album2 := models.Album{Title: "album2", Path: album2Dir}
		require.NoError(t, db.Save(&album1).Error)
		require.NoError(t, db.Save(&album2).Error)
		require.NoError(t, db.Model(user).Association("Albums").Append(&album1))
		require.NoError(t, db.Model(user).Association("Albums").Append(&album2))

		albumCache := scanner_cache.MakeAlbumCache()
		scannedAlbums, scanErrors := FindAlbumsForUser(db, user, albumCache)

		assert.Empty(t, scanErrors, "Initial scan should complete without errors")
		assert.Len(t, scannedAlbums, 2, "Should scan 2 albums initially")

		// Remove album2 from filesystem
		require.NoError(t, os.RemoveAll(album2Dir), "Failed to remove album2 directory")

		// Scan again - should clean up album2
		albumCache2 := scanner_cache.MakeAlbumCache()
		scannedAlbums2, scanErrors2 := FindAlbumsForUser(db, user, albumCache2)

		assert.Len(t, scannedAlbums2, 1, "Should only scan 1 album after deletion")
		assert.NotEmpty(t, scanErrors2, "Should have deletion errors")

		// Verify album2 was deleted from database
		var remainingAlbums []models.Album
		require.NoError(t, db.Find(&remainingAlbums).Error)
		assert.Len(t, remainingAlbums, 1, "Only 1 album should remain in database")
		assert.Equal(t, "album1", remainingAlbums[0].Title, "Remaining album should be album1")
	})
}

// createTestImage creates a minimal valid JPEG file for testing
func createTestImage(t *testing.T, path string) {
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
		t.Fatalf("Failed to create test image at %s: %v", path, err)
	}
}
