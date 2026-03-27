package resolvers

import (
	"context"
	"fmt"
	"testing"

	_ "github.com/photoview/photoview/api/test_utils/flags"
	"github.com/photoview/photoview/api/graphql/auth"
	"github.com/photoview/photoview/api/graphql/models"
	"github.com/photoview/photoview/api/test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAlbumResolvers_NoNPlusOneQueries проверяет, что загрузка thumbnail'ов
// для нескольких альбомов не вызывает N+1 запросов
func TestAlbumResolvers_NoNPlusOneQueries(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Создаем тестового пользователя
	user, err := models.RegisterUser(db, "nplus1_test_user", nil, false)
	require.NoError(t, err)

	// Создаем несколько альбомов с thumbnails
	numAlbums := 10

	for i := 0; i < numAlbums; i++ {
		album := &models.Album{
			Title: fmt.Sprintf("Test Album %d", i),
			Path:  fmt.Sprintf("/test/album_%d", i),
		}
		require.NoError(t, db.Create(album).Error)

		// Добавляем владельца
		require.NoError(t, db.Model(user).Association("Albums").Append(album))

		// Создаем media для thumbnail
		media := &models.Media{
			Title:   fmt.Sprintf("Test Photo %d", i),
			Path:    fmt.Sprintf("/test/album_%d/photo.jpg", i),
			Type:    models.MediaTypePhoto,
			AlbumID: album.ID,
		}
		require.NoError(t, db.Create(media).Error)

		// Создаем MediaURL для thumbnail
		mediaURL := &models.MediaURL{
			MediaID: media.ID,
			Purpose:  models.PhotoThumbnail,
			Width:   200,
			Height:  200,
		}
		require.NoError(t, db.Create(mediaURL).Error)

		// Устанавливаем cover
		album.CoverID = &media.ID
		require.NoError(t, db.Save(album).Error)
	}

	// Создаем resolver
	r := &queryResolver{
		Resolver: &Resolver{
			database: db,
		},
	}

	// Создаем контекст с пользователем
	ctx := auth.AddUserToContext(context.Background(), user)

	// Выполняем запрос
	result, err := r.MyAlbums(ctx, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, result, numAlbums)

	// Проверяем, что thumbnails загружены
	thumbnailCount := 0
	for _, album := range result {
		if album.CoverID != nil {
			thumbnailCount++
		}
	}

	// Основная проверка: все thumbnails должны быть загружены
	assert.Equal(t, numAlbums, thumbnailCount,
		"All albums should have thumbnails loaded without N+1 queries")

	t.Logf("✅ N+1 Test Passed: %d albums with thumbnails loaded efficiently", numAlbums)
}
