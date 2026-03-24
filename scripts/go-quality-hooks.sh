#!/bin/bash

# Go Quality Hooks for Claude Code
# Runs after each tool use to ensure code quality
# Exit code is always 0 to avoid blocking development

set -euo pipefail

# Change to api directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT/api"

echo "🔍 Running Go quality checks..."

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Only run if we're in Go project
if [[ ! -f "go.mod" ]]; then
    exit 0
fi

# 1. Auto-format Go code
echo -e "${GREEN}✓ Auto-formatting Go code...${NC}"
find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | xargs gofmt -w 2>/dev/null || true

# 2. Check for syntax errors (skip packages with C deps)
echo -e "${GREEN}✓ Checking syntax...${NC}"
go fmt ./database/... ./test_utils/... > /dev/null 2>&1 || true

# 3. Check GraphQL generation
echo -e "${GREEN}✓ Checking GraphQL generation...${NC}"
if [[ -f "graphql/generated.go" ]]; then
    echo -e "${GREEN}✓ GraphQL files present${NC}"
else
    echo -e "${YELLOW}⚠️  Run 'cd api && go generate ./...'${NC}"
fi

echo -e "${GREEN}✅ Go quality checks complete${NC}"
exit 0
