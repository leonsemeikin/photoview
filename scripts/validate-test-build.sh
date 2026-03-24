#!/bin/bash
set -e

# Allow running from any directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== Photoview Test Build Validation ==="
echo "Project root: $PROJECT_ROOT"
echo ""

echo "=== 1. Checking generated code sync ==="
cd api
go generate ./...
CHANGES=$(git status -s 2>/dev/null | grep -v "docs/test-coverage-plan.md" | grep -v "ui/junit-report.xml" | grep -v "scripts/validate-test-build.sh" | head -1)
if [ "$CHANGES" != "" ]; then
    echo "FAILED: Generated code is out of sync"
    git status -s
    exit 1
fi
echo "PASS: Generated code is in sync"
echo ""

cd ..

echo "=== 2. Running Go tests (as in CI) ==="
cd api
# Check if we're in CI (CI variable is set)
if [ -z "$CI" ]; then
    echo "SKIPPED: Not in CI environment. Run in Docker/GitHub Actions for full Go tests."
    echo "To run tests locally, use: docker compose -f docker-compose.test.yml build && docker compose -f docker-compose.test.yml up"
else
    go test ./... -v -database -filesystem -p 1 \
      -cover -coverpkg=./... -coverprofile=coverage.txt -covermode=atomic
fi
echo ""

cd ..

echo "=== 3. Building test container ==="
sudo docker compose -f docker-compose.test.yml build --no-cache
echo ""

echo "=== 4. Starting container ==="
sudo docker compose -f docker-compose.test.yml up -d
echo ""

echo "=== 5. Waiting for healthy status (timeout 60s) ==="
timeout 60s bash -c 'until sudo docker compose -f docker-compose.test.yml ps | grep -q "healthy"; do sleep 2; done' || {
    echo "FAILED: Container did not become healthy"
    sudo docker compose -f docker-compose.test.yml logs
    sudo docker compose -f docker-compose.test.yml down
    exit 1
}
echo ""

echo "=== 6. Checking health status ==="
sudo docker compose -f docker-compose.test.yml ps
echo ""

echo "=== 7. Stopping container ==="
sudo docker compose -f docker-compose.test.yml down
echo ""

echo "=== 8. Running UI tests (as in CI) ==="
cd ui
CI=true npm test -- --reporter=verbose --run --coverage
echo ""

echo "=== VALIDATION PASSED ==="
