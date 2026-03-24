package api

import (
	"context"
	"flag"
	"testing"

	"github.com/photoview/photoview/api/graphql/auth"
	"github.com/photoview/photoview/api/test_utils"
	"github.com/stretchr/testify/assert"
)

var _ = flag.Bool("database", false, "run database integration tests")
var _ = flag.Bool("filesystem", false, "run filesystem integration tests")

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