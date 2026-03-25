package api

import (
	"context"
	"errors"
	"flag"
	"testing"

	"github.com/photoview/photoview/api/graphql/auth"
	"github.com/photoview/photoview/api/graphql/models"
	"golang.org/x/crypto/bcrypt"
)

var _ = flag.Bool("database", false, "run database integration tests")
var _ = flag.Bool("filesystem", false, "run filesystem integration tests")

// mockResolver is a simple mock for graphql.Resolver
type mockResolver struct {
	called bool
	result interface{}
	err    error
}

func (m *mockResolver) Resolve(ctx context.Context) (interface{}, error) {
	m.called = true
	return m.result, m.err
}

func TestIsAuthorized_WithUser(t *testing.T) {
	ctx := context.Background()

	// Create a test user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("test_password"), bcrypt.DefaultCost)
	hashedPasswordStr := string(hashedPassword)
	user := &models.User{
		Username: "testuser",
		Password: &hashedPasswordStr,
		Admin:    false,
	}

	// Add user to context
	ctx = auth.AddUserToContext(ctx, user)

	// Create mock resolver
	resolver := &mockResolver{
		result: "success",
		err:    nil,
	}

	// Call IsAuthorized directive
	result, err := IsAuthorized(ctx, nil, resolver.Resolve)

	if err != nil {
		t.Fatalf("IsAuthorized() returned unexpected error: %v", err)
	}

	if !resolver.called {
		t.Error("Expected next resolver to be called")
	}

	if result != "success" {
		t.Errorf("Expected result 'success', got: %v", result)
	}
}

func TestIsAuthorized_WithoutUser(t *testing.T) {
	ctx := context.Background()

	// No user in context

	// Create mock resolver
	resolver := &mockResolver{
		result: "success",
		err:    nil,
	}

	// Call IsAuthorized directive
	result, err := IsAuthorized(ctx, nil, resolver.Resolve)

	if err != auth.ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized, got: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result, got: %v", result)
	}

	if resolver.called {
		t.Error("Expected next resolver NOT to be called")
	}
}

func TestIsAdmin_AdminUser(t *testing.T) {
	ctx := context.Background()

	// Create an admin user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("test_password"), bcrypt.DefaultCost)
	hashedPasswordStr := string(hashedPassword)
	user := &models.User{
		Username: "adminuser",
		Password: &hashedPasswordStr,
		Admin:    true,
	}

	// Add user to context
	ctx = auth.AddUserToContext(ctx, user)

	// Create mock resolver
	resolver := &mockResolver{
		result: "admin success",
		err:    nil,
	}

	// Call IsAdmin directive
	result, err := IsAdmin(ctx, nil, resolver.Resolve)

	if err != nil {
		t.Fatalf("IsAdmin() returned unexpected error: %v", err)
	}

	if !resolver.called {
		t.Error("Expected next resolver to be called")
	}

	if result != "admin success" {
		t.Errorf("Expected result 'admin success', got: %v", result)
	}
}

func TestIsAdmin_RegularUser(t *testing.T) {
	ctx := context.Background()

	// Create a regular (non-admin) user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("test_password"), bcrypt.DefaultCost)
	hashedPasswordStr := string(hashedPassword)
	user := &models.User{
		Username: "regularuser",
		Password: &hashedPasswordStr,
		Admin:    false,
	}

	// Add user to context
	ctx = auth.AddUserToContext(ctx, user)

	// Create mock resolver
	resolver := &mockResolver{
		result: "should not reach",
		err:    nil,
	}

	// Call IsAdmin directive
	result, err := IsAdmin(ctx, nil, resolver.Resolve)

	if err == nil {
		t.Error("Expected error for non-admin user, got nil")
	}

	if err != nil && err.Error() != "user must be admin" {
		t.Errorf("Expected 'user must be admin' error, got: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result, got: %v", result)
	}

	if resolver.called {
		t.Error("Expected next resolver NOT to be called for non-admin user")
	}
}

func TestIsAdmin_NoUser(t *testing.T) {
	ctx := context.Background()

	// No user in context

	// Create mock resolver
	resolver := &mockResolver{
		result: "should not reach",
		err:    nil,
	}

	// Call IsAdmin directive
	result, err := IsAdmin(ctx, nil, resolver.Resolve)

	if err == nil {
		t.Error("Expected error for no user, got nil")
	}

	if err != nil && err.Error() != "user must be admin" {
		t.Errorf("Expected 'user must be admin' error, got: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result, got: %v", result)
	}

	if resolver.called {
		t.Error("Expected next resolver NOT to be called when no user")
	}
}

func TestIsAuthorized_ChainedWithIsAdmin(t *testing.T) {
	ctx := context.Background()

	// Create an admin user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("test_password"), bcrypt.DefaultCost)
	hashedPasswordStr := string(hashedPassword)
	user := &models.User{
		Username: "adminuser",
		Password: &hashedPasswordStr,
		Admin:    true,
	}

	ctx = auth.AddUserToContext(ctx, user)

	// Create mock resolver
	resolver := &mockResolver{
		result: "chained success",
		err:    nil,
	}

	// First apply IsAuthorized
	result, err := IsAuthorized(ctx, nil, resolver.Resolve)
	if err != nil {
		t.Fatalf("IsAuthorized() failed: %v", err)
	}

	// Then apply IsAdmin (simulating chained directives)
	result, err = IsAdmin(ctx, nil, func(ctx context.Context) (interface{}, error) {
		return result, err
	})

	if err != nil {
		t.Fatalf("IsAdmin() failed: %v", err)
	}

	if result != "chained success" {
		t.Errorf("Expected 'chained success', got: %v", result)
	}
}

func TestIsAuthorized_ResolverError(t *testing.T) {
	ctx := context.Background()

	// Create a test user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("test_password"), bcrypt.DefaultCost)
	hashedPasswordStr := string(hashedPassword)
	user := &models.User{
		Username: "testuser",
		Password: &hashedPasswordStr,
		Admin:    false,
	}

	ctx = auth.AddUserToContext(ctx, user)

	// Create mock resolver that returns an error
	expectedErr := errors.New("resolver error")
	resolver := &mockResolver{
		result: nil,
		err:    expectedErr,
	}

	// Call IsAuthorized directive
	result, err := IsAuthorized(ctx, nil, resolver.Resolve)

	if !resolver.called {
		t.Error("Expected next resolver to be called")
	}

	if err != expectedErr {
		t.Errorf("Expected resolver error, got: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result, got: %v", result)
	}
}

func TestIsAdmin_ResolverError(t *testing.T) {
	ctx := context.Background()

	// Create an admin user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("test_password"), bcrypt.DefaultCost)
	hashedPasswordStr := string(hashedPassword)
	user := &models.User{
		Username: "adminuser",
		Password: &hashedPasswordStr,
		Admin:    true,
	}

	ctx = auth.AddUserToContext(ctx, user)

	// Create mock resolver that returns an error
	expectedErr := errors.New("admin resolver error")
	resolver := &mockResolver{
		result: nil,
		err:    expectedErr,
	}

	// Call IsAdmin directive
	result, err := IsAdmin(ctx, nil, resolver.Resolve)

	if !resolver.called {
		t.Error("Expected next resolver to be called")
	}

	if err != expectedErr {
		t.Errorf("Expected resolver error, got: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result, got: %v", result)
	}
}

func TestIsAdmin_MultipleAdminChecks(t *testing.T) {
	ctx := context.Background()

	// Create an admin user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("test_password"), bcrypt.DefaultCost)
	hashedPasswordStr := string(hashedPassword)
	user := &models.User{
		Username: "adminuser",
		Password: &hashedPasswordStr,
		Admin:    true,
	}

	ctx = auth.AddUserToContext(ctx, user)

	// Test multiple consecutive admin checks
	for i := 0; i < 5; i++ {
		resolver := &mockResolver{
			result: i,
			err:    nil,
		}

		result, err := IsAdmin(ctx, nil, resolver.Resolve)

		if err != nil {
			t.Errorf("Iteration %d: IsAdmin() failed: %v", i, err)
		}

		if result != i {
			t.Errorf("Iteration %d: Expected %d, got: %v", i, i, result)
		}
	}
}
