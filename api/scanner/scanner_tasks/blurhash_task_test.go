package scanner_tasks

import (
	"context"
	"io"
	"os"
	"path"
	"strconv"
	"testing"

	_ "github.com/photoview/photoview/api/test_utils/flags"
	"github.com/photoview/photoview/api/graphql/models"
	"github.com/photoview/photoview/api/scanner/media_encoding"
	"github.com/photoview/photoview/api/scanner/scanner_cache"
	"github.com/photoview/photoview/api/scanner/scanner_task"
	"github.com/photoview/photoview/api/test_utils"
	"github.com/photoview/photoview/api/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBlurhashTask_GeneratesBlurhash tests that blurhash is generated for a photo with thumbnail
func TestBlurhashTask_GeneratesBlurhash(t *testing.T) {
	test_utils.FilesystemTest(t)
	db := test_utils.DatabaseTest(t)

	// Create test album
	album := models.Album{
		Title: "test_album",
		Path:  "/test_album",
	}
	require.NoError(t, db.Create(&album).Error)

	// Create test media
	media := models.Media{
		Title:   "test_photo.jpg",
		Path:    "/test_album/test_photo.jpg",
		Type:    models.MediaTypePhoto,
		AlbumID: album.ID,
	}
	require.NoError(t, db.Create(&media).Error)

	// Setup: Copy a test JPEG to the cache
	// 1. Get test image path
	testImagePath := test_utils.PathFromAPIRoot("scanner", "test_media", "real_media", "jpeg.jpg")

	// 2. Verify test image exists
	if _, err := os.Stat(testImagePath); os.IsNotExist(err) {
		t.Skip("Test image not found, skipping blurhash test")
	}

	// 3. Create cache directory structure
	cacheDir := utils.MediaCachePath()
	albumCacheDir := path.Join(cacheDir, strconv.Itoa(album.ID))
	mediaCacheDir := path.Join(albumCacheDir, strconv.Itoa(media.ID))
	require.NoError(t, os.MkdirAll(mediaCacheDir, 0755), "Failed to create cache directory")

	// 4. Copy test image to cache
	destPath := path.Join(mediaCacheDir, "test_photo.jpg")
	srcFile, err := os.Open(testImagePath)
	require.NoError(t, err, "Failed to open test image")
	defer srcFile.Close()
	dstFile, err := os.Create(destPath)
	require.NoError(t, err, "Failed to create cache file")
	_, err = io.Copy(dstFile, srcFile)
	require.NoError(t, err, "Failed to copy test image")
	dstFile.Close()

	// Create a thumbnail MediaURL
	thumbnailURL := models.MediaURL{
		MediaID:  media.ID,
		Purpose:  models.PhotoThumbnail,
		Width:    200,
		Height:   200,
		MediaName: "test_photo.jpg",
	}
	require.NoError(t, db.Create(&thumbnailURL).Error)

	// Reload media with MediaURL preloaded
	var mediaWithURL models.Media
	require.NoError(t, db.Preload("MediaURL").First(&mediaWithURL, media.ID).Error)

	// Create task context
	ctx := scanner_task.NewTaskContext(context.Background(), db, &album, scanner_cache.MakeAlbumCache())

	// Create blurhash task
	task := BlurhashTask{}

	// Create EncodeMediaData
	mediaData := &media_encoding.EncodeMediaData{
		Media: &mediaWithURL,
	}

	// Execute the task
	err = task.AfterProcessMedia(ctx, mediaData, []*models.MediaURL{&thumbnailURL}, 0, 1)
	assert.NoError(t, err, "Blurhash task should complete without error")

	// Verify blurhash was generated
	var updatedMedia models.Media
	err = db.First(&updatedMedia, media.ID).Error
	require.NoError(t, err, "Failed to fetch updated media")

	assert.NotNil(t, updatedMedia.Blurhash, "Blurhash should be generated")
	assert.NotEmpty(t, *updatedMedia.Blurhash, "Blurhash should not be empty")

	// Blurhash format validation (typically 20-30 characters like "LHVH{aIAF0F0F0F0F0")
	assert.GreaterOrEqual(t, len(*updatedMedia.Blurhash), 15, "Blurhash should be at least 15 characters")

	t.Logf("Generated blurhash: %s", *updatedMedia.Blurhash)
}

// TestBlurhashTask_SkipsWhenBlurhashExists tests that blurhash generation is skipped
// when blurhash already exists and thumbnail wasn't updated
func TestBlurhashTask_SkipsWhenBlurhashExists(t *testing.T) {
	test_utils.FilesystemTest(t)
	db := test_utils.DatabaseTest(t)

	// Create test album
	album := models.Album{
		Title: "test_album",
		Path:  "/test_album",
	}
	require.NoError(t, db.Create(&album).Error)

	// Create test media with existing blurhash
	existingBlurhash := "LHVH{aIAF0F0F0F0F0"
	media := models.Media{
		Title:    "test_photo.jpg",
		Path:     "/test_album/test_photo.jpg",
		Type:     models.MediaTypePhoto,
		AlbumID:  album.ID,
		Blurhash: &existingBlurhash,
	}
	require.NoError(t, db.Create(&media).Error)

	// Create a thumbnail MediaURL
	thumbnailURL := models.MediaURL{
		MediaID:  media.ID,
		Purpose:  models.PhotoThumbnail,
		Width:    200,
		Height:   200,
		MediaName: "test_photo.jpg",
	}
	require.NoError(t, db.Create(&thumbnailURL).Error)

	// Create task context
	ctx := scanner_task.NewTaskContext(context.Background(), db, &album, scanner_cache.MakeAlbumCache())

	// Create blurhash task
	task := BlurhashTask{}

	// Create EncodeMediaData
	mediaData := &media_encoding.EncodeMediaData{
		Media: &media,
	}

	// Execute task with NO updated URLs (thumbnail wasn't updated)
	err := task.AfterProcessMedia(ctx, mediaData, []*models.MediaURL{}, 0, 1)
	assert.NoError(t, err, "Task should not error when skipping")

	// Verify blurhash was NOT changed
	var updatedMedia models.Media
	err = db.First(&updatedMedia, media.ID).Error
	require.NoError(t, err)

	assert.Equal(t, existingBlurhash, *updatedMedia.Blurhash, "Blurhash should remain unchanged when skipping")
}

// TestBlurhashTask_RegeneratesWhenThumbnailUpdated tests that blurhash is regenerated
// when thumbnail is updated, even if blurhash already exists
func TestBlurhashTask_RegeneratesWhenThumbnailUpdated(t *testing.T) {
	test_utils.FilesystemTest(t)
	db := test_utils.DatabaseTest(t)

	// Create test album
	album := models.Album{
		Title: "test_album",
		Path:  "/test_album",
	}
	require.NoError(t, db.Create(&album).Error)

	// Create test media with existing blurhash
	existingBlurhash := "LHVH{aIAF0F0F0F0F0"
	media := models.Media{
		Title:    "test_photo.jpg",
		Path:     "/test_album/test_photo.jpg",
		Type:     models.MediaTypePhoto,
		AlbumID:  album.ID,
		Blurhash: &existingBlurhash,
	}
	require.NoError(t, db.Create(&media).Error)

	// Setup: Copy test image to cache
	testImagePath := test_utils.PathFromAPIRoot("scanner", "test_media", "real_media", "jpeg.jpg")
	if _, err := os.Stat(testImagePath); os.IsNotExist(err) {
		t.Skip("Test image not found, skipping blurhash test")
	}

	cacheDir := utils.MediaCachePath()
	albumCacheDir := path.Join(cacheDir, strconv.Itoa(album.ID))
	mediaCacheDir := path.Join(albumCacheDir, strconv.Itoa(media.ID))
	require.NoError(t, os.MkdirAll(mediaCacheDir, 0755))

	destPath := path.Join(mediaCacheDir, "test_photo.jpg")
	srcFile, err := os.Open(testImagePath)
	require.NoError(t, err)
	defer srcFile.Close()
	dstFile, err := os.Create(destPath)
	require.NoError(t, err)
	_, err = io.Copy(dstFile, srcFile)
	require.NoError(t, err)
	dstFile.Close()

	// Create a thumbnail MediaURL
	thumbnailURL := models.MediaURL{
		MediaID:  media.ID,
		Purpose:  models.PhotoThumbnail,
		Width:    200,
		Height:   200,
		MediaName: "test_photo.jpg",
	}
	require.NoError(t, db.Create(&thumbnailURL).Error)

	// Reload media with MediaURL preloaded
	var mediaWithURL models.Media
	require.NoError(t, db.Preload("MediaURL").First(&mediaWithURL, media.ID).Error)

	// Create task context
	ctx := scanner_task.NewTaskContext(context.Background(), db, &album, scanner_cache.MakeAlbumCache())

	// Create blurhash task
	task := BlurhashTask{}

	// Create EncodeMediaData
	mediaData := &media_encoding.EncodeMediaData{
		Media: &mediaWithURL,
	}

	// Execute task WITH updated thumbnail URL (should regenerate)
	err = task.AfterProcessMedia(ctx, mediaData, []*models.MediaURL{&thumbnailURL}, 0, 1)
	assert.NoError(t, err)

	// Verify blurhash WAS changed (regenerated)
	var updatedMedia models.Media
	err = db.First(&updatedMedia, media.ID).Error
	require.NoError(t, err)

	assert.NotEqual(t, existingBlurhash, *updatedMedia.Blurhash, "Blurhash should be regenerated when thumbnail is updated")
	assert.NotEmpty(t, *updatedMedia.Blurhash, "Regenerated blurhash should not be empty")
}

// TestBlurhashTask_NoThumbnailReturnsError tests that task returns error when media has no thumbnail
func TestBlurhashTask_NoThumbnailReturnsError(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create test album
	album := models.Album{
		Title: "test_album",
		Path:  "/test_album",
	}
	require.NoError(t, db.Create(&album).Error)

	// Create test media WITHOUT thumbnail
	media := models.Media{
		Title:   "test_photo.jpg",
		Path:    "/test_album/test_photo.jpg",
		Type:    models.MediaTypePhoto,
		AlbumID: album.ID,
	}
	require.NoError(t, db.Create(&media).Error)

	// Create task context
	ctx := scanner_task.NewTaskContext(context.Background(), db, &album, scanner_cache.MakeAlbumCache())

	// Create blurhash task
	task := BlurhashTask{}

	// Create EncodeMediaData
	mediaData := &media_encoding.EncodeMediaData{
		Media: &media,
	}

	// Execute task - should fail because no thumbnail exists
	err := task.AfterProcessMedia(ctx, mediaData, []*models.MediaURL{}, 0, 1)
	assert.Error(t, err, "Should return error when media has no thumbnail")
	assert.Contains(t, err.Error(), "failed to get thumbnail", "Error should mention thumbnail")
}

// TestBlurhashTask_OnlyThumbnailInUpdatedURLs tests blurhash generation behavior
// when only thumbnail is in updatedURLs
func TestBlurhashTask_OnlyThumbnailInUpdatedURLs(t *testing.T) {
	test_utils.FilesystemTest(t)
	db := test_utils.DatabaseTest(t)

	// Create test album
	album := models.Album{
		Title: "test_album",
		Path:  "/test_album",
	}
	require.NoError(t, db.Create(&album).Error)

	// Create test media
	media := models.Media{
		Title:   "test_photo.jpg",
		Path:    "/test_album/test_photo.jpg",
		Type:    models.MediaTypePhoto,
		AlbumID: album.ID,
	}
	require.NoError(t, db.Create(&media).Error)

	// Setup: Copy test image to cache
	testImagePath := test_utils.PathFromAPIRoot("scanner", "test_media", "real_media", "jpeg.jpg")
	if _, err := os.Stat(testImagePath); os.IsNotExist(err) {
		t.Skip("Test image not found, skipping blurhash test")
	}

	cacheDir := utils.MediaCachePath()
	albumCacheDir := path.Join(cacheDir, strconv.Itoa(album.ID))
	mediaCacheDir := path.Join(albumCacheDir, strconv.Itoa(media.ID))
	require.NoError(t, os.MkdirAll(mediaCacheDir, 0755))

	destPath := path.Join(mediaCacheDir, "test_photo.jpg")
	srcFile, err := os.Open(testImagePath)
	require.NoError(t, err)
	defer srcFile.Close()
	dstFile, err := os.Create(destPath)
	require.NoError(t, err)
	_, err = io.Copy(dstFile, srcFile)
	require.NoError(t, err)
	dstFile.Close()

	// Create thumbnail and high-res MediaURLs
	thumbnailURL := models.MediaURL{
		MediaID:  media.ID,
		Purpose:  models.PhotoThumbnail,
		Width:    200,
		Height:   200,
		MediaName: "test_photo.jpg",
	}
	highResURL := models.MediaURL{
		MediaID:  media.ID,
		Purpose:  models.PhotoHighRes,
		Width:    800,
		Height:   600,
		MediaName: "test_photo.jpg",
	}
	require.NoError(t, db.Create(&thumbnailURL).Error)
	require.NoError(t, db.Create(&highResURL).Error)

	// Reload media with MediaURL preloaded
	var mediaWithURL models.Media
	require.NoError(t, db.Preload("MediaURL").First(&mediaWithURL, media.ID).Error)

	// Create task context
	ctx := scanner_task.NewTaskContext(context.Background(), db, &album, scanner_cache.MakeAlbumCache())

	// Create blurhash task
	task := BlurhashTask{}

	// Create EncodeMediaData
	mediaData := &media_encoding.EncodeMediaData{
		Media: &mediaWithURL,
	}

	// Execute with both URLs in updatedURLs (only thumbnail should trigger blurhash)
	updatedURLs := []*models.MediaURL{&thumbnailURL, &highResURL}
	err = task.AfterProcessMedia(ctx, mediaData, updatedURLs, 0, 1)
	assert.NoError(t, err)

	// Verify blurhash was generated
	var updatedMedia models.Media
	err = db.First(&updatedMedia, media.ID).Error
	require.NoError(t, err)

	assert.NotNil(t, updatedMedia.Blurhash, "Blurhash should be generated")
	assert.NotEmpty(t, *updatedMedia.Blurhash, "Blurhash should not be empty")
}
