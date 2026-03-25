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

// TestAlbumResolver_Media tests that Media resolver
// returns media for album
func TestAlbumResolver_Media(t *testing.T) {
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

	// Create MediaURL (required by Media query)
	mediaURL := models.MediaURL{
		MediaID: media.ID,
		Purpose:  models.PhotoThumbnail,
	}
	assert.NoError(t, db.Create(&mediaURL).Error)

	// Create resolver
	r := &albumResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Get media from album
	result, err := r.Media(context.Background(), &album, nil, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
}

// TestAlbumResolver_Thumbnail tests that Thumbnail resolver
// returns thumbnail media for album
func TestAlbumResolver_Thumbnail(t *testing.T) {
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
	r := &albumResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Get thumbnail from album
	thumbnail, err := r.Thumbnail(context.Background(), &album)
	assert.NoError(t, err)
	assert.NotNil(t, thumbnail)
	assert.Equal(t, media.ID, thumbnail.ID)
}

// TestAlbumResolver_SubAlbums tests that SubAlbums resolver
// returns sub-albums
func TestAlbumResolver_SubAlbums(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create parent album
	parentAlbum := models.Album{
		Title: "parent_album",
		Path:  "/parent_album",
	}
	assert.NoError(t, db.Create(&parentAlbum).Error)

	// Create child albums
	childAlbum1 := models.Album{
		Title:         "child_album1",
		Path:          "/parent_album/child_album1",
		ParentAlbumID: &parentAlbum.ID,
	}
	childAlbum2 := models.Album{
		Title:         "child_album2",
		Path:          "/parent_album/child_album2",
		ParentAlbumID: &parentAlbum.ID,
	}
	assert.NoError(t, db.Create(&childAlbum1).Error)
	assert.NoError(t, db.Create(&childAlbum2).Error)

	// Create resolver
	r := &albumResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Get sub-albums
	subAlbums, err := r.SubAlbums(context.Background(), &parentAlbum, nil, nil)
	assert.NoError(t, err)
	assert.Len(t, subAlbums, 2)
}

// TestAlbumResolver_Path_NoUser tests that Path resolver
// returns empty slice when no user is in context
func TestAlbumResolver_Path_NoUser(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create parent album
	parentAlbum := models.Album{
		Title: "parent_album",
		Path:  "/parent_album",
	}
	assert.NoError(t, db.Create(&parentAlbum).Error)

	// Create child album
	childAlbum := models.Album{
		Title:         "child_album",
		Path:          "/parent_album/child_album",
		ParentAlbumID: &parentAlbum.ID,
	}
	assert.NoError(t, db.Create(&childAlbum).Error)

	// Create resolver
	r := &albumResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Get path without user context
	path, err := r.Path(context.Background(), &childAlbum)
	assert.NoError(t, err)
	assert.NotNil(t, path)
	assert.Empty(t, path) // Empty slice when no user
}

// TestAlbumResolver_Shares tests that Shares resolver
// returns share tokens for album
func TestAlbumResolver_Shares(t *testing.T) {
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

	// Create share tokens
	token1 := models.ShareToken{
		Value:   "TOKEN1",
		OwnerID: user.ID,
		AlbumID: &album.ID,
	}
	token2 := models.ShareToken{
		Value:   "TOKEN2",
		OwnerID: user.ID,
		AlbumID: &album.ID,
	}
	assert.NoError(t, db.Create(&token1).Error)
	assert.NoError(t, db.Create(&token2).Error)

	// Create resolver
	r := &albumResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Get shares from album
	shares, err := r.Shares(context.Background(), &album)
	assert.NoError(t, err)
	assert.Len(t, shares, 2)
}

// TestAlbumQueryResolver_MyAlbums_Unauthorized tests that MyAlbums query
// returns an error when no user is in context
func TestAlbumQueryResolver_MyAlbums_Unauthorized(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create resolver without user context
	r := &queryResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Context without user should return error
	ctx := context.Background()
	result, err := r.MyAlbums(ctx, nil, nil, nil, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unauthorized")
}

// TestAlbumQueryResolver_Album_Unauthorized tests that Album query
// returns an error when no user is in context
func TestAlbumQueryResolver_Album_Unauthorized(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create resolver without user context
	r := &queryResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Context without user should return unauthorized error
	ctx := context.Background()
	result, err := r.Album(ctx, 1, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, auth.ErrUnauthorized, err)
}
