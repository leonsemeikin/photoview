#!/bin/bash
set -e

echo "=== Photoview Test Build Validation ==="
echo ""

echo "=== 1. Checking generated code sync ==="
cd api
go generate ./...
if [ "$(git status -s 2>/dev/null | grep -v "docs/test-coverage-plan.md" | head -1)" != "" ]; then
    echo "FAILED: Generated code is out of sync"
    git status -s
    exit 1
fi
echo "PASS: Generated code is in sync"
echo ""

echo "=== 2. Running Go tests (as in CI) ==="
cd api
go test ./... -v -database -filesystem -p 1 \
  -cover -coverpkg=./... -coverprofile=coverage.txt -covermode=atomic
echo ""

echo "=== 3. Building test container ==="
cd ..
docker compose -f docker-compose.test.yml build --no-cache
echo ""

echo "=== 4. Starting container ==="
docker compose -f docker-compose.test.yml up -d
echo ""

echo "=== 5. Waiting for healthy status (timeout 60s) ==="
timeout 60s bash -c 'until docker compose -f docker-compose.test.yml ps | grep -q "healthy"; do sleep 2; done' || {
    echo "FAILED: Container did not become healthy"
    docker compose -f docker-compose.test.yml logs
    docker compose -f docker-compose.test.yml down
    exit 1
}
echo ""

echo "=== 6. Checking health status ==="
docker compose -f docker-compose.test.yml ps
echo ""

echo "=== 7. Stopping container ==="
docker compose -f docker-compose.test.yml down
echo ""

echo "=== 8. Running UI tests (as in CI) ==="
cd ui
CI=true vitest --reporter=junit --reporter=verbose --run --coverage
echo ""

echo "=== VALIDATION PASSED ==="
