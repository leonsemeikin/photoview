package actions

import (
	"testing"

	_ "github.com/photoview/photoview/api/test_utils/flags"
	"github.com/photoview/photoview/api/graphql/models"
	"github.com/photoview/photoview/api/test_utils"
	"github.com/stretchr/testify/assert"
)

// TestGetTopLevelAlbumIDs_SingleUser tests that a single user with root albums
// gets those root albums as top-level albums
func TestGetTopLevelAlbumIDs_SingleUser(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create user
	user, err := models.RegisterUser(db, "testuser", nil, false)
	assert.NoError(t, err)

	// Create root albums for user
	rootAlbum1 := models.Album{
		Title: "Root1",
		Path:  "/photos/root1",
	}
	rootAlbum2 := models.Album{
		Title: "Root2",
		Path:  "/photos/root2",
	}

	assert.NoError(t, db.Create(&rootAlbum1).Error)
	assert.NoError(t, db.Create(&rootAlbum2).Error)

	// Assign albums to user
	assert.NoError(t, db.Model(&user).Association("Albums").Append(&rootAlbum1))
	assert.NoError(t, db.Model(&user).Association("Albums").Append(&rootAlbum2))

	// Fill user albums
	assert.NoError(t, user.FillAlbums(db))

	// Get top-level album IDs
	topLevelIDs, err := getTopLevelAlbumIDs(db, user)
	assert.NoError(t, err)

	// Both root albums should be returned (they have parent_album_id IS NULL)
	assert.Len(t, topLevelIDs, 2)
	assert.Contains(t, topLevelIDs, rootAlbum1.ID)
	assert.Contains(t, topLevelIDs, rootAlbum2.ID)
}

// TestGetTopLevelAlbumIDs_MultiUser tests that multiple users sharing
// a directory tree get correct top-level albums
func TestGetTopLevelAlbumIDs_MultiUser(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create albums structure:
	// /photos (root) - owned by admin
	//   /user1 (child1) - accessible by user1
	//   /user2 (child2) - accessible by user2

	// First create rootAlbum and get its ID
	rootAlbum := models.Album{
		Title: "root",
		Path:  "/photos",
	}
	assert.NoError(t, db.Create(&rootAlbum).Error)

	// Now create children with correct ParentAlbumID
	child1Album := models.Album{
		Title:         "user1",
		Path:          "/photos/user1",
		ParentAlbumID: &rootAlbum.ID,
	}
	child2Album := models.Album{
		Title:         "user2",
		Path:          "/photos/user2",
		ParentAlbumID: &rootAlbum.ID,
	}

	assert.NoError(t, db.Create(&child1Album).Error)
	assert.NoError(t, db.Create(&child2Album).Error)

	// Create admin user with access to root and children
	admin, err := models.RegisterUser(db, "admin", nil, false)
	assert.NoError(t, err)
	assert.NoError(t, db.Model(&admin).Association("Albums").Append(&rootAlbum))
	assert.NoError(t, db.Model(&admin).Association("Albums").Append(&child1Album))
	assert.NoError(t, db.Model(&admin).Association("Albums").Append(&child2Album))
	assert.NoError(t, admin.FillAlbums(db))

	// Admin should see rootAlbum as top-level (parent IS NULL)
	// child albums should NOT be top-level (their parent rootAlbum is owned by admin)
	topLevelIDs, err := getTopLevelAlbumIDs(db, admin)
	assert.NoError(t, err)
	assert.Len(t, topLevelIDs, 1)
	assert.Contains(t, topLevelIDs, rootAlbum.ID)

	// Create user1 with access only to child1
	user1, err := models.RegisterUser(db, "user1", nil, false)
	assert.NoError(t, err)
	assert.NoError(t, db.Model(&user1).Association("Albums").Append(&child1Album))
	assert.NoError(t, user1.FillAlbums(db))

	// User1 should see child1 as top-level (parent rootAlbum is NOT owned by user1)
	topLevelIDs, err = getTopLevelAlbumIDs(db, user1)
	assert.NoError(t, err)
	assert.Len(t, topLevelIDs, 1)
	assert.Contains(t, topLevelIDs, child1Album.ID)
}

// TestGetTopLevelAlbumIDs_SubAlbumScenario tests the scenario where
// a user's albums are all sub-albums of another user's root album
func TestGetTopLevelAlbumIDs_SubAlbumScenario(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create albums structure:
	// /photos (root) - owned by admin
	//   /family (child1) - shared
	//     /vacation (grandchild1) - owned by user1
	//   /work (child2) - owned by admin

	// Create albums in correct order to get IDs
	rootAlbum := models.Album{
		Title: "root",
		Path:  "/photos",
	}
	assert.NoError(t, db.Create(&rootAlbum).Error)

	child1Album := models.Album{
		Title:         "family",
		Path:          "/photos/family",
		ParentAlbumID: &rootAlbum.ID,
	}
	assert.NoError(t, db.Create(&child1Album).Error)

	grandchild1Album := models.Album{
		Title:         "vacation",
		Path:          "/photos/family/vacation",
		ParentAlbumID: &child1Album.ID,
	}
	assert.NoError(t, db.Create(&grandchild1Album).Error)

	child2Album := models.Album{
		Title:         "work",
		Path:          "/photos/work",
		ParentAlbumID: &rootAlbum.ID,
	}
	assert.NoError(t, db.Create(&child2Album).Error)

	// Create admin with access to root and child2
	admin, err := models.RegisterUser(db, "admin", nil, false)
	assert.NoError(t, err)
	assert.NoError(t, db.Model(&admin).Association("Albums").Append(&rootAlbum))
	assert.NoError(t, db.Model(&admin).Association("Albums").Append(&child2Album))
	assert.NoError(t, admin.FillAlbums(db))

	// Admin should see rootAlbum and child2Album as top-level
	// rootAlbum: parent IS NULL → top-level
	// child2Album: parent rootAlbum is owned by admin, BUT parent IS NULL check... wait
	// Actually: rootAlbum IS NULL → included, child2Album parent=1, 1 IN [1,4] → NOT included
	topLevelIDs, err := getTopLevelAlbumIDs(db, admin)
	assert.NoError(t, err)
	assert.Len(t, topLevelIDs, 1)
	assert.Contains(t, topLevelIDs, rootAlbum.ID)

	// Create user1 with access only to grandchild1
	user1, err := models.RegisterUser(db, "user1", nil, false)
	assert.NoError(t, err)
	assert.NoError(t, db.Model(&user1).Association("Albums").Append(&grandchild1Album))
	assert.NoError(t, user1.FillAlbums(db))

	// User1 should see grandchild1 as top-level (parent chain: child1→root, neither owned by user1)
	topLevelIDs, err = getTopLevelAlbumIDs(db, user1)
	assert.NoError(t, err)
	assert.Len(t, topLevelIDs, 1)
	assert.Contains(t, topLevelIDs, grandchild1Album.ID)

	// Create user2 with access to child1 and grandchild1
	user2, err := models.RegisterUser(db, "user2", nil, false)
	assert.NoError(t, err)
	assert.NoError(t, db.Model(&user2).Association("Albums").Append(&child1Album))
	assert.NoError(t, db.Model(&user2).Association("Albums").Append(&grandchild1Album))
	assert.NoError(t, user2.FillAlbums(db))

	// User2 should see child1 as top-level (parent rootAlbum NOT owned by user2)
	// grandchild1 should NOT be top-level (parent child1Album IS owned by user2)
	topLevelIDs, err = getTopLevelAlbumIDs(db, user2)
	assert.NoError(t, err)
	assert.Len(t, topLevelIDs, 1)
	assert.Contains(t, topLevelIDs, child1Album.ID)
}

// TestGetTopLevelAlbumIDs_NestedHierarchy tests a more complex
// nested hierarchy scenario
func TestGetTopLevelAlbumIDs_NestedHierarchy(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create deep hierarchy:
	// /a1 (root)
	//   /b1 (child of a1)
	//     /c1 (child of b1)
	//   /b2 (child of a1)

	// Create albums in correct order to get IDs
	a1 := models.Album{Title: "a1", Path: "/a1"}
	assert.NoError(t, db.Create(&a1).Error)

	b1 := models.Album{Title: "b1", Path: "/a1/b1", ParentAlbumID: &a1.ID}
	assert.NoError(t, db.Create(&b1).Error)

	c1 := models.Album{Title: "c1", Path: "/a1/b1/c1", ParentAlbumID: &b1.ID}
	assert.NoError(t, db.Create(&c1).Error)

	b2 := models.Album{Title: "b2", Path: "/a1/b2", ParentAlbumID: &a1.ID}
	assert.NoError(t, db.Create(&b2).Error)

	// User with access to c1 only
	user, err := models.RegisterUser(db, "user", nil, false)
	assert.NoError(t, err)
	assert.NoError(t, db.Model(&user).Association("Albums").Append(&c1))
	assert.NoError(t, user.FillAlbums(db))

	// c1 should be top-level (parent b1 not owned by user)
	topLevelIDs, err := getTopLevelAlbumIDs(db, user)
	assert.NoError(t, err)
	assert.Len(t, topLevelIDs, 1)
	assert.Contains(t, topLevelIDs, c1.ID)

	// User with access to b1 and c1
	user2, err := models.RegisterUser(db, "user2", nil, false)
	assert.NoError(t, err)
	assert.NoError(t, db.Model(&user2).Association("Albums").Append(&b1))
	assert.NoError(t, db.Model(&user2).Association("Albums").Append(&c1))
	assert.NoError(t, user2.FillAlbums(db))

	// b1 should be top-level (parent a1 not owned by user2)
	topLevelIDs, err = getTopLevelAlbumIDs(db, user2)
	assert.NoError(t, err)
	assert.Len(t, topLevelIDs, 1)
	assert.Contains(t, topLevelIDs, b1.ID)
}

// TestGetTopLevelAlbumIDs_EmptyAlbums tests that a user with no albums
// returns an empty slice
func TestGetTopLevelAlbumIDs_EmptyAlbums(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create user without albums
	user, err := models.RegisterUser(db, "testuser", nil, false)
	assert.NoError(t, err)
	assert.NoError(t, user.FillAlbums(db))

	// Should return empty slice
	topLevelIDs, err := getTopLevelAlbumIDs(db, user)
	assert.NoError(t, err)
	assert.Len(t, topLevelIDs, 0)
}

// TestGetTopLevelAlbumIDs_AllChildrenHaveParents tests the fallback logic
// when all albums have parents within user's list (second SQL query)
func TestGetTopLevelAlbumIDs_AllChildrenHaveParents(t *testing.T) {
	db := test_utils.DatabaseTest(t)

	// Create a 3-level hierarchy where user owns middle and bottom
	// /root (NOT owned by user)
	//   /mid (owned by user)
	//     /leaf (owned by user)

	rootAlbum := models.Album{Title: "root", Path: "/root"}
	assert.NoError(t, db.Create(&rootAlbum).Error)

	midAlbum := models.Album{
		Title:         "mid",
		Path:          "/root/mid",
		ParentAlbumID: &rootAlbum.ID,
	}
	assert.NoError(t, db.Create(&midAlbum).Error)

	leafAlbum := models.Album{
		Title:         "leaf",
		Path:          "/root/mid/leaf",
		ParentAlbumID: &midAlbum.ID,
	}
	assert.NoError(t, db.Create(&leafAlbum).Error)

	// User owns mid and leaf
	user, err := models.RegisterUser(db, "user", nil, false)
	assert.NoError(t, err)
	assert.NoError(t, db.Model(&user).Association("Albums").Append(&midAlbum))
	assert.NoError(t, db.Model(&user).Association("Albums").Append(&leafAlbum))
	assert.NoError(t, user.FillAlbums(db))

	// midAlbum should be top-level (parent rootAlbum NOT owned by user)
	// leafAlbum should NOT be top-level (parent midAlbum IS owned by user)
	topLevelIDs, err := getTopLevelAlbumIDs(db, user)
	assert.NoError(t, err)
	assert.Len(t, topLevelIDs, 1)
	assert.Contains(t, topLevelIDs, midAlbum.ID)
}
