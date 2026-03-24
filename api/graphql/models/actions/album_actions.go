package actions

import (
	"github.com/photoview/photoview/api/graphql/models"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func MyAlbums(db *gorm.DB, user *models.User, order *models.Ordering, paginate *models.Pagination,
	onlyRoot *bool, showEmpty *bool, onlyWithFavorites *bool) ([]*models.Album, error) {

	if err := user.FillAlbums(db); err != nil {
		return nil, err
	}

	if len(user.Albums) == 0 {
		return nil, nil
	}

	userAlbumIDs := make([]int, len(user.Albums))
	for i, album := range user.Albums {
		userAlbumIDs[i] = album.ID
	}

	query := db.Model(models.Album{}).Where("id IN (?)", userAlbumIDs)

	if onlyRoot != nil && *onlyRoot {

		singleRootAlbumID := getSingleRootAlbumID(user)

		if singleRootAlbumID != -1 && len(user.Albums) > 1 {
			query = query.Where("parent_album_id = ?", singleRootAlbumID)
		} else {
			// Handle case where user's albums are sub-albums of another user's root
			// Find top-level albums for this user (albums to show in the root view)
			topLevelIDs, err := getTopLevelAlbumIDs(db, user)
			if err != nil {
				return nil, err
			}
			// Filter to only show top-level albums
			query = query.Where("id IN (?)", topLevelIDs)
		}
	}

	query = favoritesQuery(showEmpty, db, onlyWithFavorites, user, query)

	query = models.FormatSQL(query, order, paginate)

	var albums []*models.Album
	if err := query.Find(&albums).Error; err != nil {
		return nil, err
	}

	return albums, nil
}

func getSingleRootAlbumID(user *models.User) int {
	var singleRootAlbumID int = -1
	for _, album := range user.Albums {
		if album.ParentAlbumID == nil {
			if singleRootAlbumID == -1 {
				singleRootAlbumID = album.ID
			} else {
				singleRootAlbumID = -1
				break
			}
		}
	}
	return singleRootAlbumID
}

// getTopLevelAlbumIDs returns IDs of albums that are either:
// - root albums (parent_album_id IS NULL), or
// - direct children of albums NOT owned by the user
// This handles cases where multiple users share a directory tree
func getTopLevelAlbumIDs(db *gorm.DB, user *models.User) ([]int, error) {
	userAlbumIDs := make([]int, len(user.Albums))
	for i, album := range user.Albums {
		userAlbumIDs[i] = album.ID
	}

	var topLevelIDs []int

	// Find albums whose parent is either NULL or NOT in user's album list
	err := db.Model(&models.Album{}).
		Where("id IN (?)", userAlbumIDs).
		Where("parent_album_id IS NULL OR parent_album_id NOT IN (?)", userAlbumIDs).
		Pluck("id", &topLevelIDs).Error

	if err != nil {
		return nil, err
	}

	// If no albums found with the above logic, it means all albums have parents within user's list
	// In this case, find albums that are at the "top" of the user's hierarchy
	if len(topLevelIDs) == 0 && len(userAlbumIDs) > 0 {
		// Find albums whose parent's parent is NOT in user's list (i.e., parent is owned by another user)
		err := db.Raw(`
			SELECT DISTINCT child.id
			FROM albums child
			WHERE child.id IN ?
			AND child.parent_album_id IN (
				SELECT parent.id FROM albums parent
				WHERE parent.id = child.parent_album_id
				AND (parent.parent_album_id IS NULL OR parent.parent_album_id NOT IN ?)
			)
		`, userAlbumIDs, userAlbumIDs).Scan(&topLevelIDs).Error

		if err != nil {
			return nil, err
		}
	}

	return topLevelIDs, nil
}

func favoritesQuery(showEmpty *bool, db *gorm.DB, onlyWithFavorites *bool, user *models.User, query *gorm.DB) *gorm.DB {
	if showEmpty == nil || !*showEmpty {
		subQuery := db.Model(&models.Media{}).Where("album_id = albums.id")

		if onlyWithFavorites != nil && *onlyWithFavorites {
			favoritesSubquery := db.
				Model(&models.UserMediaData{UserID: user.ID}).
				Where("user_media_data.media_id = media.id").
				Where("user_media_data.favorite = true")

			subQuery = subQuery.Where("EXISTS (?)", favoritesSubquery)
		}

		query = query.Where("EXISTS (?)", subQuery)
	}
	return query
}

func Album(db *gorm.DB, user *models.User, id int) (*models.Album, error) {
	var album models.Album
	if err := db.First(&album, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("album not found")
		}
		return nil, err
	}

	ownsAlbum, err := user.OwnsAlbum(db, &album)
	if err != nil {
		return nil, err
	}

	if !ownsAlbum {
		return nil, errors.New("forbidden")
	}

	return &album, nil
}

func AlbumPath(db *gorm.DB, user *models.User, album *models.Album) ([]*models.Album, error) {
	var albumPath []*models.Album

	err := db.Raw(`
		WITH recursive path_albums AS (
			SELECT * FROM albums anchor WHERE anchor.id = ?
			UNION
			SELECT parent.* FROM path_albums child JOIN albums parent ON parent.id = child.parent_album_id
		)
		SELECT * FROM path_albums WHERE id != ?
	`, album.ID, album.ID).Scan(&albumPath).Error

	// Make sure to only return albums this user owns
	for i := len(albumPath) - 1; i >= 0; i-- {
		album := albumPath[i]

		owns, err := user.OwnsAlbum(db, album)
		if err != nil {
			return nil, err
		}

		if !owns {
			albumPath = albumPath[i+1:]
			break
		}

	}

	if err != nil {
		return nil, err
	}

	return albumPath, nil
}

func SetAlbumCover(db *gorm.DB, user *models.User, mediaID int) (*models.Album, error) {
	var media models.Media

	if err := db.Find(&media, mediaID).Error; err != nil {
		return nil, err
	}

	var album models.Album

	if err := db.Find(&album, &media.AlbumID).Error; err != nil {
		return nil, err
	}

	ownsAlbum, err := user.OwnsAlbum(db, &album)
	if err != nil {
		return nil, err
	}

	if !ownsAlbum {
		return nil, errors.New("forbidden")
	}

	if err := db.Model(&album).Update("cover_id", mediaID).Error; err != nil {
		return nil, err
	}

	return &album, nil
}

func ResetAlbumCover(db *gorm.DB, user *models.User, albumID int) (*models.Album, error) {
	var album models.Album
	if err := db.Find(&album, albumID).Error; err != nil {
		return nil, err
	}

	ownsAlbum, err := user.OwnsAlbum(db, &album)
	if err != nil {
		return nil, err
	}

	if !ownsAlbum {
		return nil, errors.New("forbidden")
	}

	if err := db.Model(&album).Update("cover_id", nil).Error; err != nil {
		return nil, err
	}

	return &album, nil
}
