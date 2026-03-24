package database

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/photoview/photoview/api/graphql/models"
	"github.com/photoview/photoview/api/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var _ = flag.Bool("database", false, "run database integration tests")
var _ = flag.Bool("filesystem", false, "run filesystem integration tests")

// TestSetupDatabase_SQLite tests SQLite database connection
func TestSetupDatabase_SQLite(t *testing.T) {