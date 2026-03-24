package api

import (
	"context"
	"testing"

	"github.com/photoview/photoview/api/graphql/auth"
	"github.com/photoview/photoview/api/test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockResolver is a mock GraphQL resolver for testing
type mockResolver struct {
	called bool
}

func (m *mockResolver) resolve(ctx context.Context) (interface{}, error) {
	m.called = true
	return "success", nil
}

func TestIsAuthorized_WithUser(t *testing.T) {
	db := test_utils.CreateTestDatabase(t)
	defer test_utils.CleanupTestDatabase(t, db)

	// Create a test user
	user := test_utils.CreateTestUser(t, db, "testuser", false)

	// Create context with user
	ctx := auth.AddUserToContext(context.Background(), user)

	// Create mock resolver
	resolver := &mockResolver{}

	// Call IsAuthorized directive
	result, err := IsAuthorized(ctx, nil, resolver.resolve)

	// Should succeed
	assert.NoError(t, err, "IsAuthorized should not return error with valid user")
	assert.Equal(t, "success", result, "Should return resolver result")
	assert.True(t, resolver.called, "Resolver should have been called")
}

func TestIsAuthorized_WithoutUser(t *testing.T) {
	// Create context without user
	ctx := context.Background()

	// Create mock resolver
	resolver := &mockResolver{}

	// Call IsAuthorized directive
	result, err := IsAuthorized(ctx, nil, resolver.resolve)

	// Should fail
	assert.Error(t, err, "IsAuthorized should return error without user")
	assert.Equal(t, auth.ErrUnauthorized, err, "Should return unauthorized error")
	assert.Nil(t, result, "Result should be nil")
	assert.False(t, resolver.called, "Resolver should not have been called")
}

func TestIsAdmin_AdminUser(t *testing.T) {
	db := test_utils.CreateTestDatabase(t)
	defer test_utils.CleanupTestDatabase(t, db)

	// Create admin user
	adminUser := test_utils.CreateTestUser(t, db, "admin", true)

	// Create context with admin user
	ctx := auth.AddUserToContext(context.Background(), adminUser)

	// Create mock resolver
	resolver := &mockResolver{}

	// Call IsAdmin directive
	result, err := IsAdmin(ctx, nil, resolver.resolve)

	// Should succeed
	assert.NoError(t, err, "IsAdmin should not return error for admin user")
	assert.Equal(t, "success", result, "Should return resolver result")
	assert.True(t, resolver.called, "Resolver should have been called")
}

func TestIsAdmin_RegularUser(t *testing.T) {
	db := test_utils.CreateTestDatabase(t)
	defer test_utils.CleanupTestDatabase(t, db)

	// Create regular user (non-admin)
	regularUser := test_utils.CreateTestUser(t, db, "user", false)

	// Create context with regular user
	ctx := auth.AddUserToContext(context.Background(), regularUser)

	// Create mock resolver
	resolver := &mockResolver{}

	// Call IsAdmin directive
	result, err := IsAdmin(ctx, nil, resolver.resolve)

	// Should fail
	assert.Error(t, err, "IsAdmin should return error for regular user")
	assert.Contains(t, err.Error(), "user must be admin", "Error should mention admin requirement")
	assert.Nil(t, result, "Result should be nil")
	assert.False(t, resolver.called, "Resolver should not have been called")
}

func TestIsAdmin_NoUser(t *testing.T) {
	// Create context without user
	ctx := context.Background()

	// Create mock resolver
	resolver := &mockResolver{}

	// Call IsAdmin directive
	result, err := IsAdmin(ctx, nil, resolver.resolve)

	// Should fail
	assert.Error(t, err, "IsAdmin should return error without user")
	assert.Contains(t, err.Error(), "user must be admin", "Error should mention admin requirement")
	assert.Nil(t, result, "Result should be nil")
	assert.False(t, resolver.called, "Resolver should not have been called")
}

func TestIsAuthorized_ResolverError(t *testing.T) {
	db := test_utils.CreateTestDatabase(t)
	defer test_utils.CleanupTestDatabase(t, db)

	// Create a test user
	user := test_utils.CreateTestUser(t, db, "testuser", false)

	// Create context with user
	ctx := auth.AddUserToContext(context.Background(), user)

	// Create mock resolver that returns error
	errResolver := func(ctx context.Context) (interface{}, error) {
		return nil, assert.AnError
	}

	// Call IsAuthorized directive
	result, err := IsAuthorized(ctx, nil, errResolver)

	// Should pass the resolver error through
	assert.Error(t, err, "Should return resolver error")
	assert.Equal(t, assert.AnError, err, "Should return the exact resolver error")
	assert.Nil(t, result, "Result should be nil")
}

func TestIsAdmin_ResolverError(t *testing.T) {
	db := test_utils.CreateTestDatabase(t)
	defer test_utils.CleanupTestDatabase(t, db)

	// Create admin user
	adminUser := test_utils.CreateTestUser(t, db, "admin", true)

	// Create context with admin user
	ctx := auth.AddUserToContext(context.Background(), adminUser)

	// Create mock resolver that returns error
	errResolver := func(ctx context.Context) (interface{}, error) {
		return nil, assert.AnError
	}

	// Call IsAdmin directive
	result, err := IsAdmin(ctx, nil, errResolver)

	// Should pass the resolver error through
	assert.Error(t, err, "Should return resolver error")
	assert.Equal(t, assert.AnError, err, "Should return the exact resolver error")
	assert.Nil(t, result, "Result should be nil")
}

func TestIsAuthorized_NilObject(t *testing.T) {
	db := test_utils.CreateTestDatabase(t)
	defer test_utils.CleanupTestDatabase(t, db)

	// Create a test user
	user := test_utils.CreateTestUser(t, db, "testuser", false)

	// Create context with user
	ctx := auth.AddUserToContext(context.Background(), user)

	// Create mock resolver
	resolver := &mockResolver{}

	// Call IsAuthorized directive with nil object
	result, err := IsAuthorized(ctx, nil, resolver.resolve)

	// Should succeed
	assert.NoError(t, err, "IsAuthorized should work with nil object")
	assert.Equal(t, "success", result, "Should return resolver result")
}

func TestIsAdmin_NilObject(t *testing.T) {
	db := test_utils.CreateTestDatabase(t)
	defer test_utils.CleanupTestDatabase(t, db)

	// Create admin user
	adminUser := test_utils.CreateTestUser(t, db, "admin", true)

	// Create context with admin user
	ctx := auth.AddUserToContext(context.Background(), adminUser)

	// Create mock resolver
	resolver := &mockResolver{}

	// Call IsAdmin directive with nil object
	result, err := IsAdmin(ctx, nil, resolver.resolve)

	// Should succeed
	assert.NoError(t, err, "IsAdmin should work with nil object")
	assert.Equal(t, "success", result, "Should return resolver result")
}

// TestIsAuthorized_MultipleUsers tests that the directive works with multiple different users
func TestIsAuthorized_MultipleUsers(t *testing.T) {
	db := test_utils.CreateTestDatabase(t)
	defer test_utils.CleanupTestDatabase(t, db)

	// Create multiple users
	user1 := test_utils.CreateTestUser(t, db, "user1", false)
	user2 := test_utils.CreateTestUser(t, db, "user2", false)

	// Test with user1
	ctx1 := auth.AddUserToContext(context.Background(), user1)
	resolver1 := &mockResolver{}
	result1, err1 := IsAuthorized(ctx1, nil, resolver1.resolve)

	assert.NoError(t, err1, "IsAuthorized should work for user1")
	assert.Equal(t, "success", result1, "Should return resolver result for user1")
	assert.True(t, resolver1.called, "Resolver should have been called for user1")

	// Test with user2
	ctx2 := auth.AddUserToContext(context.Background(), user2)
	resolver2 := &mockResolver{}
	result2, err2 := IsAuthorized(ctx2, nil, resolver2.resolve)

	assert.NoError(t, err2, "IsAuthorized should work for user2")
	assert.Equal(t, "success", result2, "Should return resolver result for user2")
	assert.True(t, resolver2.called, "Resolver should have been called for user2")
}

// TestIsAdmin_MultipleAdmins tests that the directive works with multiple admin users
func TestIsAdmin_MultipleAdmins(t *testing.T) {
	db := test_utils.CreateTestDatabase(t)
	defer test_utils.CleanupTestDatabase(t, db)

	// Create multiple admin users
	admin1 := test_utils.CreateTestUser(t, db, "admin1", true)
	admin2 := test_utils.CreateTestUser(t, db, "admin2", true)

	// Test with admin1
	ctx1 := auth.AddUserToContext(context.Background(), admin1)
	resolver1 := &mockResolver{}
	result1, err1 := IsAdmin(ctx1, nil, resolver1.resolve)

	assert.NoError(t, err1, "IsAdmin should work for admin1")
	assert.Equal(t, "success", result1, "Should return resolver result for admin1")
	assert.True(t, resolver1.called, "Resolver should have been called for admin1")

	// Test with admin2
	ctx2 := auth.AddUserToContext(context.Background(), admin2)
	resolver2 := &mockResolver{}
	result2, err2 := IsAdmin(ctx2, nil, resolver2.resolve)

	assert.NoError(t, err2, "IsAdmin should work for admin2")
	assert.Equal(t, "success", result2, "Should return resolver result for admin2")
	assert.True(t, resolver2.called, "Resolver should have been called for admin2")
}
