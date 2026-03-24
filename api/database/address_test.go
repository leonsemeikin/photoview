package database

import (
	"flag"
	"os"
	"testing"

	"github.com/photoview/photoview/api/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var _ = flag.Bool("database", false, "run database integration tests")
var _ = flag.Bool("filesystem", false, "run filesystem integration tests")

// TestGetSqliteAddress_DefaultPath tests SQLite address generation with default path
func TestGetSqliteAddress_DefaultPath(t *testing.T) {