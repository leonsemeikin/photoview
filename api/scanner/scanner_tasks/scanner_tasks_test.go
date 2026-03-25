package scanner_tasks

import (
	"context"
	"testing"

	_ "github.com/photoview/photoview/api/test_utils/flags"
	"github.com/photoview/photoview/api/graphql/models"
	"github.com/photoview/photoview/api/scanner/scanner_cache"
	"github.com/photoview/photoview/api/scanner/scanner_task"
	"github.com/photoview/photoview/api/test_utils"
	"github.com/stretchr/testify/assert"
)

// TestExifTask_NotNewMedia tests that EXIF task
// does nothing when media is not new
func TestExifTask_NotNewMedia(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create test album
	album := models.Album{
		Title: "test_album",
		Path:  "/test_album",
	}
	assert.NoError(t, db.Create(&album).Error)

	// Create test media
	media := models.Media{
		Title:   "test_photo",
		Path:    "/test_album/photo.jpg",
		Type:    models.MediaTypePhoto,
		AlbumID: album.ID,
	}
	assert.NoError(t, db.Create(&media).Error)

	// Create task context
	ctx := scanner_task.NewTaskContext(context.Background(), db, &album, scanner_cache.MakeAlbumCache())

	// Create EXIF task
	task := ExifTask{}

	// Call with newMedia=false should do nothing
	err := task.AfterMediaFound(ctx, &media, false)
	assert.NoError(t, err)

	// EXIF should not be created
	var exifCount int64
	db.Model(&models.MediaEXIF{}).Count(&exifCount)
	assert.Equal(t, int64(0), exifCount)
}

// TestExifTask_NewMedia_NoFile tests that EXIF task
// handles error when file doesn't exist gracefully (logs warning but doesn't fail)
func TestExifTask_NewMedia_NoFile(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create test album
	album := models.Album{
		Title: "test_album",
		Path:  "/test_album",
	}
	assert.NoError(t, db.Create(&album).Error)

	// Create test media with non-existent path
	media := models.Media{
		Title:   "test_photo",
		Path:    "/nonexistent/photo.jpg", // File doesn't exist
		Type:    models.MediaTypePhoto,
		AlbumID: album.ID,
	}
	assert.NoError(t, db.Create(&media).Error)

	// Create task context
	ctx := scanner_task.NewTaskContext(context.Background(), db, &album, scanner_cache.MakeAlbumCache())

	// Create EXIF task
	task := ExifTask{}

	// Should log warning but not return error
	err := task.AfterMediaFound(ctx, &media, true)
	assert.NoError(t, err) // Task logs warning but doesn't fail
}

// TestVideoMetadataTask_NotNewMedia tests that VideoMetadata task
// does nothing when media is not new
func TestVideoMetadataTask_NotNewMedia(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create test album
	album := models.Album{
		Title: "test_album",
		Path:  "/test_album",
	}
	assert.NoError(t, db.Create(&album).Error)

	// Create test video media
	media := models.Media{
		Title:   "test_video",
		Path:    "/test_album/video.mp4",
		Type:    models.MediaTypeVideo,
		AlbumID: album.ID,
	}
	assert.NoError(t, db.Create(&media).Error)

	// Create task context
	ctx := scanner_task.NewTaskContext(context.Background(), db, &album, scanner_cache.MakeAlbumCache())

	// Create VideoMetadata task
	task := VideoMetadataTask{}

	// Call with newMedia=false should do nothing
	err := task.AfterMediaFound(ctx, &media, false)
	assert.NoError(t, err)

	// VideoMetadata should not be created
	var videoMetadataCount int64
	db.Model(&models.VideoMetadata{}).Count(&videoMetadataCount)
	assert.Equal(t, int64(0), videoMetadataCount)
}

// TestVideoMetadataTask_NotVideo tests that VideoMetadata task
// does nothing when media type is not video
func TestVideoMetadataTask_NotVideo(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create test album
	album := models.Album{
		Title: "test_album",
		Path:  "/test_album",
	}
	assert.NoError(t, db.Create(&album).Error)

	// Create test photo media (not video)
	media := models.Media{
		Title:   "test_photo",
		Path:    "/test_album/photo.jpg",
		Type:    models.MediaTypePhoto,
		AlbumID: album.ID,
	}
	assert.NoError(t, db.Create(&media).Error)

	// Create task context
	ctx := scanner_task.NewTaskContext(context.Background(), db, &album, scanner_cache.MakeAlbumCache())

	// Create VideoMetadata task
	task := VideoMetadataTask{}

	// Call with photo media should do nothing
	err := task.AfterMediaFound(ctx, &media, true)
	assert.NoError(t, err)

	// VideoMetadata should not be created
	var videoMetadataCount int64
	db.Model(&models.VideoMetadata{}).Count(&videoMetadataCount)
	assert.Equal(t, int64(0), videoMetadataCount)
}

// TestVideoMetadataTask_NewMedia_NoFile tests that VideoMetadata task
// handles error when file doesn't exist gracefully
func TestVideoMetadataTask_NewMedia_NoFile(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create test album
	album := models.Album{
		Title: "test_album",
		Path:  "/test_album",
	}
	assert.NoError(t, db.Create(&album).Error)

	// Create test video media with non-existent path
	media := models.Media{
		Title:   "test_video",
		Path:    "/nonexistent/video.mp4", // File doesn't exist
		Type:    models.MediaTypeVideo,
		AlbumID: album.ID,
	}
	assert.NoError(t, db.Create(&media).Error)

	// Create task context
	ctx := scanner_task.NewTaskContext(context.Background(), db, &album, scanner_cache.MakeAlbumCache())

	// Create VideoMetadata task
	task := VideoMetadataTask{}

	// Should log error but not return error (task logs but doesn't fail)
	err := task.AfterMediaFound(ctx, &media, true)
	assert.NoError(t, err) // Task logs warning but doesn't fail
}
