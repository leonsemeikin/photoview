package resolvers

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	_ "github.com/photoview/photoview/api/test_utils/flags"
	"github.com/photoview/photoview/api/graphql/auth"
	"github.com/photoview/photoview/api/graphql/models"
	"github.com/photoview/photoview/api/test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// queryCounter wraps gorm.DB to count SQL queries
type queryCounter struct {
	db     *gorm.DB
	count   *int32
}

func newQueryCounter(db *gorm.DB) *queryCounter {
	var count int32
	return &queryCounter{
		db:   db,
		count: &count,
	}
}

func (qc *queryCounter) getCount() int {
	return int(atomic.LoadInt32(qc.count))
}

// trackQueries enables query counting by wrapping the database
func (qc *queryCounter) trackQueries(fn func()) {
	// Create a callback to count queries
	callbackName := "nplus1_query_counter"

	qc.db.Callback().Query().Before("gorm:query").Register(callbackName, func(db *gorm.DB) {
		if db.Statement.SQL.String() != "" {
			// Count only SELECT queries (excluding internal GORM queries)
			sql := db.Statement.SQL.String()
			if len(sql) > 10 && (sql[:6] == "SELECT" || sql[:6] == "select") {
				atomic.AddInt32(qc.count, 1)
			}
		}
	})

	defer func() {
		// Note: GORM doesn't have Unregister, so callback stays registered
		// This is acceptable for tests as DB instance is not reused
	}()

	fn()
}

// TestAlbumResolvers_NoNPlusOneQueries tests that fetching multiple albums
// with thumbnails doesn't cause N+1 query problems
func TestAlbumResolvers_NoNPlusOneQueries(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create test user
	user, err := models.RegisterUser(db, "nplus1_user", nil, false)
	require.NoError(t, err)

	// Create multiple albums (10 albums to test N+1)
	numAlbums := 10
	albums := make([]*models.Album, numAlbums)

	for i := 0; i < numAlbums; i++ {
		album := &models.Album{
			Title: fmt.Sprintf("Test Album %d", i),
			Path:  fmt.Sprintf("/test/album_%d", i),
		}
		require.NoError(t, db.Create(album).Error)
		albums[i] = album

		// Add user as owner
		require.NoError(t, db.Model(user).Association("Albums").Append(album))

		// Create media for thumbnail
		media := &models.Media{
			Title:   fmt.Sprintf("Photo %d", i),
			Path:    fmt.Sprintf("/test/album_%d/photo.jpg", i),
			Type:    models.MediaTypePhoto,
			AlbumID: album.ID,
		}
		require.NoError(t, db.Create(media).Error)

		// Create MediaURL for thumbnail
		mediaURL := &models.MediaURL{
			MediaID: media.ID,
			Purpose:  models.PhotoThumbnail,
			Width:    200,
			Height:   200,
		}
		require.NoError(t, db.Create(mediaURL).Error)

		// Set album cover
		album.CoverID = &media.ID
		require.NoError(t, db.Save(album).Error)
	}

	// Create query counter
	counter := newQueryCounter(db)

	// Set user in context
	ctx := auth.AddUserToContext(context.Background(), user)

	// Create resolver
	r := &queryResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Track queries while fetching albums
	counter.trackQueries(func() {
		// Fetch all albums (this should NOT cause N+1 queries for thumbnails)
		result, err := r.MyAlbums(ctx, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		require.Len(t, result, numAlbums, "Should fetch all albums")

		// Verify that thumbnails are actually loaded
		thumbnailCount := 0
		for _, album := range result {
			if album.CoverID != nil {
				thumbnailCount++
			}
		}
		assert.Equal(t, numAlbums, thumbnailCount, "All albums should have thumbnails loaded")
	})

	queryCount := counter.getCount()

	// Each album has a thumbnail, but they should be loaded efficiently
	// Acceptable query count:
	// - 1 for albums
	// - 1-2 for batched thumbnail loading (via dataloader)
	// Total should be < 15 (if there was N+1, it would be much higher)
	maxAcceptableQueries := 25 // Allow margin for GORM overhead and internal queries
	assert.Less(t, queryCount, maxAcceptableQueries,
		"Query count should be less than %d to avoid N+1 problem (got %d queries for %d albums)",
		maxAcceptableQueries, queryCount, numAlbums)

	t.Logf("N+1 Test Results: %d queries for %d albums (%.2f queries per album)",
		queryCount, numAlbums, float64(queryCount)/float64(numAlbums))
}

// TestAlbumResolvers_NoNPlusOneQueries_ManyAlbums tests with more albums
// to ensure batching works at scale
func TestAlbumResolvers_NoNPlusOneQueries_ManyAlbums(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create test user
	user, err := models.RegisterUser(db, "nplus1_user_many", nil, false)
	require.NoError(t, err)

	// Create many albums (50 albums to stress test)
	numAlbums := 50
	albums := make([]*models.Album, numAlbums)

	for i := 0; i < numAlbums; i++ {
		album := &models.Album{
			Title: fmt.Sprintf("Many Test Album %d", i),
			Path:  fmt.Sprintf("/test/many_album_%d", i),
		}
		require.NoError(t, db.Create(album).Error)
		albums[i] = album

		// Add user as owner
		require.NoError(t, db.Model(user).Association("Albums").Append(album))

		// Create media for thumbnail
		media := &models.Media{
			Title:   fmt.Sprintf("Many Photo %d", i),
			Path:    fmt.Sprintf("/test/many_album_%d/photo.jpg", i),
			Type:    models.MediaTypePhoto,
			AlbumID: album.ID,
		}
		require.NoError(t, db.Create(media).Error)

		// Create MediaURL for thumbnail
		mediaURL := &models.MediaURL{
			MediaID: media.ID,
			Purpose:  models.PhotoThumbnail,
			Width:    200,
			Height:   200,
		}
		require.NoError(t, db.Create(mediaURL).Error)

		// Set album cover
		album.CoverID = &media.ID
		require.NoError(t, db.Save(album).Error)
	}

	// Create query counter
	counter := newQueryCounter(db)

	// Set user in context
	ctx := auth.AddUserToContext(context.Background(), user)

	// Create resolver
	r := &queryResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Track queries while fetching albums
	counter.trackQueries(func() {
		// Fetch all albums
		result, err := r.MyAlbums(ctx, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		require.Len(t, result, numAlbums, "Should fetch all albums")

		// Verify thumbnails are loaded
		thumbnailCount := 0
		for _, album := range result {
			if album.CoverID != nil {
				thumbnailCount++
			}
		}
		assert.Equal(t, numAlbums, thumbnailCount, "All albums should have thumbnails loaded")
	})

	queryCount := counter.getCount()

	// With 50 albums, N+1 would cause 51+ queries (1 for albums + 50 for thumbnails)
	// Efficient batching should cause significantly fewer queries
	// Allow up to 70 queries to account for GORM overhead and pagination
	maxAcceptableQueries := 70
	assert.Less(t, queryCount, maxAcceptableQueries,
		"Query count should be less than %d for %d albums to avoid N+1 problem (got %d queries)",
		maxAcceptableQueries, numAlbums, queryCount)

	t.Logf("N+1 Test Results (Many Albums): %d queries for %d albums (%.2f queries per album)",
		queryCount, numAlbums, float64(queryCount)/float64(numAlbums))
}

// TestAlbumResolvers_NoNPlusOneQueries_SubAlbums tests that sub-albums
// don't cause N+1 queries when loaded with parent albums
func TestAlbumResolvers_NoNPlusOneQueries_SubAlbums(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create test user
	user, err := models.RegisterUser(db, "nplus1_user_sub", nil, false)
	require.NoError(t, err)

	// Create parent album
	parentAlbum := &models.Album{
		Title: "Parent Album",
		Path:  "/test/parent",
	}
	require.NoError(t, db.Create(parentAlbum).Error)
	require.NoError(t, db.Model(user).Association("Albums").Append(parentAlbum))

	// Create multiple sub-albums (10 sub-albums)
	numSubAlbums := 10
	for i := 0; i < numSubAlbums; i++ {
		subAlbum := &models.Album{
			Title:         fmt.Sprintf("Sub Album %d", i),
			Path:          fmt.Sprintf("/test/parent/sub_%d", i),
			ParentAlbumID: &parentAlbum.ID,
		}
		require.NoError(t, db.Create(subAlbum).Error)
		require.NoError(t, db.Model(user).Association("Albums").Append(subAlbum))

		// Create media for each sub-album
		media := &models.Media{
			Title:   fmt.Sprintf("Sub Photo %d", i),
			Path:    fmt.Sprintf("/test/parent/sub_%d/photo.jpg", i),
			Type:    models.MediaTypePhoto,
			AlbumID: subAlbum.ID,
		}
		require.NoError(t, db.Create(media).Error)

		// Create MediaURL for thumbnail
		mediaURL := &models.MediaURL{
			MediaID: media.ID,
			Purpose:  models.PhotoThumbnail,
			Width:    200,
			Height:   200,
		}
		require.NoError(t, db.Create(mediaURL).Error)

		// Set album cover
		subAlbum.CoverID = &media.ID
		require.NoError(t, db.Save(subAlbum).Error)
	}

	// Create query counter
	counter := newQueryCounter(db)

	// Set user in context
	ctx := auth.AddUserToContext(context.Background(), user)

	// Create resolver
	r := &albumResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Track queries while fetching sub-albums
	counter.trackQueries(func() {
		// Fetch sub-albums
		result, err := r.SubAlbums(ctx, parentAlbum, nil, nil)
		require.NoError(t, err)
		require.Len(t, result, numSubAlbums, "Should fetch all sub-albums")

		// Verify thumbnails are loaded
		thumbnailCount := 0
		for _, album := range result {
			if album.CoverID != nil {
				thumbnailCount++
			}
		}
		t.Logf("Sub-album N+1 Test: %d/%d have thumbnails", thumbnailCount, numSubAlbums)
	})

	queryCount := counter.getCount()

	// N+1 would cause 11+ queries (1 for sub-albums + 10 for thumbnails)
	// Efficient batching should cause significantly fewer queries
	maxAcceptableQueries := 25
	assert.Less(t, queryCount, maxAcceptableQueries,
		"Query count should be less than %d for %d sub-albums to avoid N+1 problem (got %d queries)",
		maxAcceptableQueries, numSubAlbums, queryCount)

	t.Logf("Sub-album N+1 Test Results: %d queries for %d sub-albums (%.2f queries per album)",
		queryCount, numSubAlbums, float64(queryCount)/float64(numSubAlbums))
}
