package resolvers

import (
	"context"
	"testing"

	_ "github.com/photoview/photoview/api/test_utils/flags"
	"github.com/photoview/photoview/api/graphql/auth"
	"github.com/photoview/photoview/api/graphql/models"
	"github.com/photoview/photoview/api/test_utils"
	"github.com/stretchr/testify/assert"
)

// TestMediaResolver_Favorite_Unauthorized tests that Favorite resolver
// returns an error when no user is in context
func TestMediaResolver_Favorite_Unauthorized(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create test media
	album := models.Album{
		Title: "test_album",
		Path:  "/test_album",
	}
	assert.NoError(t, db.Create(&album).Error)

	media := models.Media{
		Title:   "test_photo",
		Path:    "/test_album/photo.jpg",
		Type:    models.MediaTypePhoto,
		AlbumID: album.ID,
	}
	assert.NoError(t, db.Create(&media).Error)

	// Create resolver without user context
	r := &mediaResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Context without user should return unauthorized error
	ctx := context.Background()
	favorite, err := r.Favorite(ctx, &media)

	assert.Error(t, err)
	assert.Equal(t, auth.ErrUnauthorized, err)
	assert.False(t, favorite)
}

// TestMediaResolver_Album tests that Album resolver
// returns the correct album for media
func TestMediaResolver_Album(t *testing.T) {
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

	// Create resolver
	r := &mediaResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Get album from media
	resultAlbum, err := r.Album(context.Background(), &media)
	assert.NoError(t, err)
	assert.NotNil(t, resultAlbum)
	assert.Equal(t, album.ID, resultAlbum.ID)
	assert.Equal(t, album.Title, resultAlbum.Title)
}

// TestMediaResolver_Exif tests that Exif resolver
// returns EXIF data for media
func TestMediaResolver_Exif(t *testing.T) {
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

	// Create EXIF data with correct field names
	camera := "TestCamera"
	maker := "TestMaker"
	exposure := 1.0 / 100.0
	aperture := 2.8
	iso := int64(100)
	focalLength := 50.0

	exif := models.MediaEXIF{
		Camera:      &camera,
		Maker:       &maker,
		Exposure:    &exposure,
		Aperture:    &aperture,
		Iso:         &iso,
		FocalLength: &focalLength,
	}
	assert.NoError(t, db.Model(&media).Association("Exif").Append(&exif))

	// Create resolver
	r := &mediaResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Get EXIF from media
	resultExif, err := r.Exif(context.Background(), &media)
	assert.NoError(t, err)
	assert.NotNil(t, resultExif)
	assert.Equal(t, "TestCamera", *resultExif.Camera)
	assert.Equal(t, "TestMaker", *resultExif.Maker)
}

// TestMediaResolver_Type tests that Type resolver
// returns properly formatted media type (Title case)
func TestMediaResolver_Type(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create resolver
	r := &mediaResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	ctx := context.Background()

	// Test photo type - should return "Photo" (Title case)
	photoMedia := models.Media{
		Type: models.MediaTypePhoto,
	}
	mediaType, err := r.Type(ctx, &photoMedia)
	assert.NoError(t, err)
	assert.Equal(t, models.MediaType("Photo"), mediaType)

	// Test video type - should return "Video" (Title case)
	videoMedia := models.Media{
		Type: models.MediaTypeVideo,
	}
	mediaType, err = r.Type(ctx, &videoMedia)
	assert.NoError(t, err)
	assert.Equal(t, models.MediaType("Video"), mediaType)
}

// TestMediaResolver_Shares tests that Shares resolver
// returns share tokens for media
func TestMediaResolver_Shares(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create test user
	user, err := models.RegisterUser(db, "testuser", nil, false)
	assert.NoError(t, err)

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

	// Create share tokens
	token1 := models.ShareToken{
		Value:   "TOKEN1",
		OwnerID: user.ID,
		MediaID: &media.ID,
	}
	token2 := models.ShareToken{
		Value:   "TOKEN2",
		OwnerID: user.ID,
		MediaID: &media.ID,
	}
	assert.NoError(t, db.Create(&token1).Error)
	assert.NoError(t, db.Create(&token2).Error)

	// Create resolver
	r := &mediaResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Get shares from media
	shares, err := r.Shares(context.Background(), &media)
	assert.NoError(t, err)
	assert.Len(t, shares, 2)
}

// TestMediaResolver_HighRes_PhotoOnly tests that HighRes resolver
// returns nil for non-photo media
func TestMediaResolver_HighRes_PhotoOnly(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create test album
	album := models.Album{
		Title: "test_album",
		Path:  "/test_album",
	}
	assert.NoError(t, db.Create(&album).Error)

	// Video media should return nil for HighRes
	videoMedia := models.Media{
		Title:   "video",
		Path:    "/test_album/video.mp4",
		Type:    models.MediaTypeVideo,
		AlbumID: album.ID,
	}
	assert.NoError(t, db.Create(&videoMedia).Error)

	// Create resolver
	r := &mediaResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// HighRes should return nil for video
	highRes, err := r.HighRes(context.Background(), &videoMedia)
	assert.NoError(t, err)
	assert.Nil(t, highRes)
}

// TestMediaResolver_VideoWeb_VideoOnly tests that VideoWeb resolver
// returns nil for non-video media
func TestMediaResolver_VideoWeb_VideoOnly(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create test album
	album := models.Album{
		Title: "test_album",
		Path:  "/test_album",
	}
	assert.NoError(t, db.Create(&album).Error)

	// Photo media should return nil for VideoWeb
	photoMedia := models.Media{
		Title:   "photo",
		Path:    "/test_album/photo.jpg",
		Type:    models.MediaTypePhoto,
		AlbumID: album.ID,
	}
	assert.NoError(t, db.Create(&photoMedia).Error)

	// Create resolver
	r := &mediaResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// VideoWeb should return nil for photo
	videoWeb, err := r.VideoWeb(context.Background(), &photoMedia)
	assert.NoError(t, err)
	assert.Nil(t, videoWeb)
}

// TestMediaResolver_FavoriteMedia_Unauthorized tests that FavoriteMedia mutation
// returns an error when no user is in context
func TestMediaResolver_FavoriteMedia_Unauthorized(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create test media
	album := models.Album{
		Title: "test_album",
		Path:  "/test_album",
	}
	assert.NoError(t, db.Create(&album).Error)

	media := models.Media{
		Title:   "test_photo",
		Path:    "/test_album/photo.jpg",
		Type:    models.MediaTypePhoto,
		AlbumID: album.ID,
	}
	assert.NoError(t, db.Create(&media).Error)

	// Create resolver without user context
	r := &mutationResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Context without user should return unauthorized error
	ctx := context.Background()
	result, err := r.FavoriteMedia(ctx, media.ID, true)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, auth.ErrUnauthorized, err)
}

// TestMediaResolver_MyMedia_Unauthorized tests that MyMedia query
// returns an error when no user is in context
func TestMediaResolver_MyMedia_Unauthorized(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create resolver without user context
	r := &queryResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Context without user should return error
	ctx := context.Background()
	result, err := r.MyMedia(ctx, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unauthorized")
}

// TestMediaResolver_Media_Unauthorized tests that Media query
// returns an error when no user is in context
func TestMediaResolver_Media_Unauthorized(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create resolver without user context
	r := &queryResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Context without user should return unauthorized error
	ctx := context.Background()
	result, err := r.Media(ctx, 1, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, auth.ErrUnauthorized, err)
}

// TestMediaResolver_MediaList_Unauthorized tests that MediaList query
// returns an error when no user is in context
func TestMediaResolver_MediaList_Unauthorized(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create resolver without user context
	r := &queryResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Context without user should return unauthorized error
	ctx := context.Background()
	result, err := r.MediaList(ctx, []int{1, 2, 3})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, auth.ErrUnauthorized, err)
}
