# План покрытия тестами проекта Photoview

> **Для агентов-исполнителей:** ОБЯЗАТЕЛЬНЫЙ НАВЫК: Используйте superpowers:subagent-driven-development (рекомендуется) или superpowers:executing-plans для реализации этого плана пошагово. Шаги используют синтаксис чекбокса (`- [ ]`) для отслеживания.

**Цель:** Создать комплексное покрытие тестами для Photoview (self-hosted photo gallery) — Go backend и React/TypeScript frontend

**Архитектура:** Go GraphQL API + React UI, SQLite/MariaDB/PostgreSQL, Scanner pipeline для медиа

**Технологии:** Go 1.26, gqlgen, GORM, React 18, TypeScript, Vite, Apollo Client

---

## КОНТЕКСТ

### Почему это важно

Photoview — это production-система с 20,000+ фото, работающая на ограниченном железе (NanoPi). Отсутствие тестов в критических компонентах приводит к:

- **Data corruption** — ошибки в database слое
- **Race conditions** — конкурентная обработка в scanner queue
- **Unauthorized access** — отсутствие тестов для GraphQL directives
- **Broken UX** — Apollo client errors, lazy loading failures

### Текущее состояние

- **Go тесты:** 29 файлов (~15-20% покрытия)
- **TS тесты:** 3 файла (~5% покрытия UI)
- **Хуки:** PostToolUse запускает `go test ./... -short -count=1 -race` и бенчмарки

### Критические области БЕЗ тестов

1. `api/database/database.go` — подключение к БД, миграции
2. `api/scanner/scanner_queue/queue.go` — concurrent worker pool
3. `api/graphql/directive.go` — `@isAuthorized`, `@isAdmin`
4. `api/scanner/scanner_user.go` — owner propagation
5. `ui/src/apolloClient.ts` — GraphQL конфигурация
6. `ui/src/components/photoGallery/ProtectedMedia.tsx` — auth media loading

---

## ЭТАП 0: ПОДГОТОВКА

**Перед началом реализации необходимо подготовить инфраструктуру для тестирования и валидации.**

### Задача 0: Подготовка тестовой инфраструктуры

**Приоритет:** MUST DO FIRST

- [ ] **Шаг 0.1: Создать docker-compose для тестирования**

```bash
# Создать файл docker-compose.test.yml
```

Содержимое `docker-compose.test.yml`:
```yaml
version: '3.8'

services:
  photoview-test:
    build:
      context: .
      dockerfile: Dockerfile
      target: release
    image: photoview-test:latest
    container_name: photoview-test-container
    environment:
      - PHOTOVIEW_DATABASE_DRIVER=sqlite
      - PHOTOVIEW_DATABASE_PATH=/app/data/photoview-test.db
      - PHOTOVIEW_LISTEN_IP=0.0.0.0
      - PHOTOVIEW_LISTEN_PORT=80
    volumes:
      - ./test-data:/photos:ro
      - test-cache:/home/photoview/media-cache
      - test-db:/app/data
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:80/api"]
      interval: 5s
      timeout: 10s
      retries: 3
      start_period: 30s
    ports:
      - "4001:80"

volumes:
  test-cache:
  test-db:
```

- [ ] **Шаг 0.2: Создать директорию для тестовых данных**

```bash
mkdir -p test-data
# Добавить несколько тестовых изображений (можно symlink на существующие)
```

- [ ] **Шаг 0.3: Проверить что базовый контейнер собирается и стартует**

```bash
# Сборка
docker compose -f docker-compose.test.yml build

# Запуск в фоне
docker compose -f docker-compose.test.yml up -d

# Ожидание старта (макс 60 секунд)
timeout 60s bash -c 'until docker compose -f docker-compose.test.yml ps | grep -q "healthy"; do sleep 2; done' || echo "Timeout waiting for healthy status"

# Проверка статуса
docker compose -f docker-compose.test.yml ps

# Ожидается: Status: healthy (или Up + healthy)

# Остановка
docker compose -f docker-compose.test.yml down
```

Ожидается: Контейнер переходит в статус `healthy`

- [ ] **Шаг 0.4: Установить Go зависимости для тестирования**

```bash
cd api
go get github.com/DATA-DOG/go-sqlmock
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/mock
go mod tidy
```

- [ ] **Шаг 0.5: Установить Node.js зависимости для тестирования**

```bash
cd ui
npm install --save-dev @testing-library/react @testing-library/jest-dom @testing-library/user-event vitest @vitest/ui jsdom msw
```

- [ ] **Шаг 0.6: Проверить что базовые тесты запускаются**

```bash
# Go тесты
cd api
go test ./... -short -count=1 -v

# Node тесты
cd ui
npm test -- --run
```

Ожидается: Все существующие тесты PASS

- [ ] **Шаг 0.7: Создать скрипт для валидации после каждой задачи**

```bash
# Создать файл scripts/validate-test-build.sh
```

Содержимое `scripts/validate-test-build.sh`:
```bash
#!/bin/bash
set -e

echo "=== 1. Running Go tests ==="
cd api
go test ./... -short -count=1 -race

echo "=== 2. Building test container ==="
cd ..
docker compose -f docker-compose.test.yml build --no-cache

echo "=== 3. Starting container ==="
docker compose -f docker-compose.test.yml up -d

echo "=== 4. Waiting for healthy status (timeout 60s) ==="
timeout 60s bash -c 'until docker compose -f docker-compose.test.yml ps | grep -q "healthy"; do sleep 2; done' || {
    echo "FAILED: Container did not become healthy"
    docker compose -f docker-compose.test.yml logs
    docker compose -f docker-compose.test.yml down
    exit 1
}

echo "=== 5. Checking health status ==="
docker compose -f docker-compose.test.yml ps

echo "=== 6. Stopping container ==="
docker compose -f docker-compose.test.yml down

echo "=== VALIDATION PASSED ==="
```

```bash
chmod +x scripts/validate-test-build.sh
```

- [ ] **Шаг 0.8: Commit подготовительных файлов**

```bash
git add docker-compose.test.yml test-data/.gitkeep scripts/validate-test-build.sh api/go.sum ui/package.json ui/package-lock.json
git commit -m "test: prepare testing infrastructure"
```

---

## ЭТАП 1: КРИТИЧЕСКИЕ ТЕСТЫ ДЛЯ BACKEND STABILITY

### Задача 1: Database Layer Tests

**Файлы:**
- Создать: `api/database/database_test.go`
- Создать: `api/database/address_test.go`
- Создать: `api/test_utils/fixtures.go`

**Приоритет:** CRITICAL

- [ ] **Шаг 1.1: Создать helpers для тестов БД**

```go
// api/test_utils/fixtures.go
package test_utils

func CreateTestDatabase(t *testing.T) *gorm.DB
func CleanupTestDatabase(db *gorm.DB)
```

Запуск: `cd api && go test ./test_utils -v`
Ожидается: PASS, helpers компилируются

```bash
git add api/test_utils/fixtures.go
git commit -m "test: add database test helpers"
```

- [ ] **Шаг 1.2: Написать тест для SQLite подключения**

```go
func TestSetupDatabase_SQLite(t *testing.T)
```

Запуск: `cd api && go test ./database -run TestSetupDatabase_SQLite -v`
Ожидается: PASS

```bash
git add api/database/database_test.go
git commit -m "test: add SQLite connection test"
```

- [ ] **Шаг 1.3: Написать тест для MySQL подключения**

```go
func TestSetupDatabase_MySQL(t *testing.T)
```

Запуск: `cd api && go test ./database -run TestSetupDatabase_MySQL -v`
Ожидается: PASS (или SKIP если нет MySQL)

```bash
git add api/database/database_test.go
git commit -m "test: add MySQL connection test"
```

- [ ] **Шаг 1.4: Написать тест для PostgreSQL подключения**

```go
func TestSetupDatabase_Postgres(t *testing.T)
```

Запуск: `cd api && go test ./database -run TestSetupDatabase_Postgres -v`
Ожидается: PASS (или SKIP если нет PostgreSQL)

```bash
git add api/database/database_test.go
git commit -m "test: add PostgreSQL connection test"
```

- [ ] **Шаг 1.5: Написать тест для retry логики**

```go
func TestSetupDatabase_RetryLogic(t *testing.T)
```

Запуск: `cd api && go test ./database -run TestSetupDatabase_RetryLogic -v`
Ожидается: 5 попыток при ошибке

```bash
git add api/database/database_test.go
git commit -m "test: add database retry logic test"
```

- [ ] **Шаг 1.6: Написать тест для WAL режима SQLite**

```go
func TestGetSqliteAddress_WALMode(t *testing.T)
```

Запуск: `cd api && go test ./database -run TestGetSqliteAddress_WALMode -v`
Ожидается: `_journal_mode=WAL` в URL

```bash
git add api/database/address_test.go
git commit -m "test: add SQLite WAL mode test"
```

- [ ] **Шаг 1.7: Написать тесты для миграций**

```go
func TestMigrateDatabase_AutoMigrate(t *testing.T)
func TestClearDatabase_AllModels(t *testing.T)
```

Запуск: `cd api && go test ./database -run TestMigrate -v`
Ожидается: PASS

```bash
git add api/database/database_test.go
git commit -m "test: add database migration tests"
```

- [ ] **Шаг 1.8: Валидация задачи — проверить сборку и запуск контейнера**

```bash
./scripts/validate-test-build.sh
```

Ожидается: VALIDATION PASSED

---

### Задача 2: Scanner Queue Concurrency Tests

**Файлы:**
- Модифицировать: `api/scanner/scanner_queue/queue_test.go`
- Создать: `api/scanner/scanner_queue/queue_race_test.go`

**Приоритет:** CRITICAL

- [ ] **Шаг 2.1: Написать тест concurrent jobs**

```go
func TestScannerQueue_ConcurrentJobs(t *testing.T)
```

Запуск: `cd api && go test ./scanner/scanner_queue -race -run TestScannerQueue_ConcurrentJobs -v`
Ожидается: PASS, NO RACE CONDITIONS

```bash
git add api/scanner/scanner_queue/queue_test.go
git commit -m "test: add concurrent jobs test"
```

- [ ] **Шаг 2.2: Написать тест для notify channel blocking**

```go
func TestScannerQueue_NotifyChannelBlocking(t *testing.T)
```

Запуск: `cd api && go test ./scanner/scanner_queue -run TestScannerQueue_NotifyChannelBlocking -v`
Ожидается: Buffer 100 предотвращает deadlock

```bash
git add api/scanner/scanner_queue/queue_test.go
git commit -m "test: add notify channel blocking test"
```

- [ ] **Шаг 2.3: Написать тест graceful shutdown**

```go
func TestScannerQueue_CloseBackgroundWorker(t *testing.T)
```

Запуск: `cd api && go test ./scanner/scanner_queue -run TestScannerQueue_CloseBackgroundWorker -v`
Ожидается: Все jobs завершены перед shutdown

```bash
git add api/scanner/scanner_queue/queue_test.go
git commit -m "test: add graceful shutdown test"
```

- [ ] **Шаг 2.4: Написать тест non-fatal errors**

```go
func TestAddUserToQueue_NonFatalErrors(t *testing.T)
```

Запуск: `cd api && go test ./scanner/scanner_queue -run TestAddUserToQueue_NonFatalErrors -v`
Ожидается: Permission errors не блокируют очередь

```bash
git add api/scanner/scanner_queue/queue_test.go
git commit -m "test: add non-fatal errors test"
```

- [ ] **Шаг 2.5: Валидация задачи — проверить сборку и запуск контейнера**

```bash
./scripts/validate-test-build.sh
```

Ожидается: VALIDATION PASSED

---

### Задача 3: GraphQL Directives Tests

**Файлы:**
- Создать: `api/graphql/directive_test.go`

**Приоритет:** CRITICAL

- [ ] **Шаг 3.1: Написать тест @isAuthorized**

```go
func TestIsAuthorized_WithUser(t *testing.T)
func TestIsAuthorized_WithoutUser(t *testing.T)
```

Запуск: `cd api && go test ./graphql -run TestIsAuthorized -v`
Ожидается: ErrUnauthorized без user

```bash
git add api/graphql/directive_test.go
git commit -m "test: add @isAuthorized directive tests"
```

- [ ] **Шаг 3.2: Написать тест @isAdmin**

```go
func TestIsAdmin_AdminUser(t *testing.T)
func TestIsAdmin_RegularUser(t *testing.T)
func TestIsAdmin_NoUser(t *testing.T)
```

Запуск: `cd api && go test ./graphql -run TestIsAdmin -v`
Ожидается: Error для non-admin

```bash
git add api/graphql/directive_test.go
git commit -m "test: add @isAdmin directive tests"
```

- [ ] **Шаг 3.3: Валидация задачи — проверить сборку и запуск контейнера**

```bash
./scripts/validate-test-build.sh
```

Ожидается: VALIDATION PASSED

---

## ЭТАП 2: GRAPHQL RESOLVERS И БИЗНЕС-ЛОГИКА

### Задача 4: Album Actions Tests

**Файлы:**
- Создать: `api/graphql/models/actions/album_actions_detail_test.go`

**Приоритет:** HIGH

- [ ] **Шаг 4.1: Написать тест для getTopLevelAlbumIDs**

```go
func TestGetTopLevelAlbumIDs_SingleUser(t *testing.T)
func TestGetTopLevelAlbumIDs_MultiUser(t *testing.T)
func TestGetTopLevelAlbumIDs_SubAlbumScenario(t *testing.T)
```

Запуск: `cd api && go test ./graphql/models/actions -run TestGetTopLevelAlbumIDs -v`
Ожидается: Правильная фильтрация top-level albums

```bash
git add api/graphql/models/actions/album_actions_detail_test.go
git commit -m "test: add getTopLevelAlbumIDs tests"
```

- [ ] **Шаг 4.2: Валидация задачи — проверить сборку и запуск контейнера**

```bash
./scripts/validate-test-build.sh
```

Ожидается: VALIDATION PASSED

---

### Задача 5: Media Resolvers Tests

**Файлы:**
- Создать: `api/graphql/resolvers/media_resolver_test.go`

**Приоритет:** HIGH

- [ ] **Шаг 5.1: Написать тест Thumbnail с dataloader**

```go
func TestMediaResolver_Thumbnail_Dataloader(t *testing.T)
```

Запуск: `cd api && go test ./graphql/resolvers -run TestMediaResolver_Thumbnail -v`
Ожидается: Батчинг работает (1 SQL запрос вместо N)

```bash
git add api/graphql/resolvers/media_resolver_test.go
git commit -m "test: add Thumbnail dataloader test"
```

- [ ] **Шаг 5.2: Написать тест favorite авторизации**

```go
func TestMediaResolver_Favorite_Unauthorized(t *testing.T)
```

Запуск: `cd api && go test ./graphql/resolvers -run TestMediaResolver_Favorite -v`
Ожидается: Ошибка без авторизации

```bash
git add api/graphql/resolvers/media_resolver_test.go
git commit -m "test: add Favorite authorization test"
```

- [ ] **Шаг 5.3: Валидация задачи — проверить сборку и запуск контейнера**

```bash
./scripts/validate-test-build.sh
```

Ожидается: VALIDATION PASSED

---

### Задача 6: Scanner Tasks Tests

**Файлы:**
- Создать: `api/scanner/scanner_tasks/exif_task_test.go`
- Создать: `api/scanner/scanner_tasks/blurhash_task_test.go`
- Создать: `api/scanner/scanner_tasks/video_metadata_task_test.go`

**Приоритет:** MEDIUM

- [ ] **Шаг 6.1: Написать EXIF task тесты**

```go
func TestSaveEXIF_NewMedia(t *testing.T)
func TestSaveEXIF_ParseError(t *testing.T)
```

Запуск: `cd api && go test ./scanner/scanner_tasks -run TestSaveEXIF -v`
Ожидается: PASS

```bash
git add api/scanner/scanner_tasks/exif_task_test.go
git commit -m "test: add EXIF task tests"
```

- [ ] **Шаг 6.2: Написать Blurhash task тесты**

```go
func TestGenerateBlurhashFromThumbnail_ValidImage(t *testing.T)
```

Запуск: `cd api && go test ./scanner/scanner_tasks -run TestGenerateBlurhash -v`
Ожидается: PASS

```bash
git add api/scanner/scanner_tasks/blurhash_task_test.go
git commit -m "test: add Blurhash task tests"
```

- [ ] **Шаг 6.3: Написать Video metadata тесты**

```go
func TestVideoMetadataTask_ValidVideo(t *testing.T)
func TestVideoMetadataTask_FFprobeError(t *testing.T)
```

Запуск: `cd api && go test ./scanner/scanner_tasks -run TestVideoMetadataTask -v`
Ожидается: PASS

```bash
git add api/scanner/scanner_tasks/video_metadata_task_test.go
git commit -m "test: add Video metadata task tests"
```

- [ ] **Шаг 6.4: Валидация задачи — проверить сборку и запуск контейнера**

```bash
./scripts/validate-test-build.sh
```

Ожидается: VALIDATION PASSED

---

## ЭТАП 3: UI COMPONENTS И USER FLOWS

### Задача 7: Apollo Client Tests

**Файлы:**
- Создать: `ui/src/apolloClient.test.ts`

**Приоритет:** HIGH

- [ ] **Шаг 7.1: Написать тест HTTP link конфигурации**

```typescript
test('configures HTTP link correctly', () => {
  expect(APPLICATION_ENDPOINT).toBe('/api')
})
```

Запуск: `cd ui && npm test apolloClient.test.ts`
Ожидается: PASS

```bash
git add ui/src/apolloClient.test.ts
git commit -m "test: add Apollo HTTP link test"
```

- [ ] **Шаг 7.2: Написать тест WebSocket split**

```typescript
test('splits subscriptions to WebSocket', () => {})
```

Запуск: `cd ui && npm test apolloClient.test.ts`
Ожидается: PASS

```bash
git add ui/src/apolloClient.test.ts
git commit -m "test: add Apollo WebSocket split test"
```

- [ ] **Шаг 7.3: Написать тест error handler**

```typescript
test('error handler clears token on 401', () => {})
test('error handler shows GraphQL errors', () => {})
```

Запуск: `cd ui && npm test apolloClient.test.ts`
Ожидается: PASS

```bash
git add ui/src/apolloClient.test.ts
git commit -m "test: add Apollo error handler tests"
```

- [ ] **Шаг 7.4: Написать тест cache pagination**

```typescript
test('cache pagination merges correctly', () => {})
```

Запуск: `cd ui && npm test apolloClient.test.ts`
Ожидается: PASS

```bash
git add ui/src/apolloClient.test.ts
git commit -m "test: add Apollo cache pagination test"
```

- [ ] **Шаг 7.5: Валидация задачи — проверить сборку и запуск контейнера**

```bash
./scripts/validate-test-build.sh
```

Ожидается: VALIDATION PASSED

---

### Задача 8: Protected Media Tests

**Файлы:**
- Создать: `ui/src/components/photoGallery/ProtectedMedia.test.tsx`

**Приоритет:** HIGH

- [ ] **Шаг 8.1: Написать тест token appending**

```typescript
test('appends token to URL from share path', () => {})
```

Запуск: `cd ui && npm test ProtectedMedia.test.tsx`
Ожидается: PASS

```bash
git add ui/src/components/photoGallery/ProtectedMedia.test.tsx
git commit -m "test: add ProtectedMedia token appending test"
```

- [ ] **Шаг 8.2: Написать тест lazy loading**

```typescript
test('uses native lazy loading when supported', () => {})
test('falls back to IntersectionObserver', () => {})
```

Запуск: `cd ui && npm test ProtectedMedia.test.tsx`
Ожидается: PASS

```bash
git add ui/src/components/photoGallery/ProtectedMedia.test.tsx
git commit -m "test: add ProtectedMedia lazy loading tests"
```

- [ ] **Шаг 8.3: Написать тест blurhash**

```typescript
test('shows blurhash while loading', () => {})
test('hides blurhash after loaded', () => {})
```

Запуск: `cd ui && npm test ProtectedMedia.test.tsx`
Ожидается: PASS

```bash
git add ui/src/components/photoGallery/ProtectedMedia.test.tsx
git commit -m "test: add ProtectedMedia blurhash tests"
```

- [ ] **Шаг 8.4: Валидация задачи — проверить сборку и запуск контейнера**

```bash
./scripts/validate-test-build.sh
```

Ожидается: VALIDATION PASSED

---

### Задача 9: Custom Hooks Tests

**Файлы:**
- Создать: `ui/src/hooks/useURLParameters.test.ts`
- Создать: `ui/src/hooks/useOrderingParams.test.ts`
- Создать: `ui/src/hooks/useScrollPagination.test.ts`

**Приоритет:** MEDIUM

- [ ] **Шаг 9.1: Написать тесты для useURLParameters**

```typescript
test('reads parameters from URL', () => {})
test('updates URL on change', () => {})
```

Запуск: `cd ui && npm test useURLParameters.test.ts`
Ожидается: PASS

```bash
git add ui/src/hooks/useURLParameters.test.ts
git commit -m "test: add useURLParameters hook tests"
```

- [ ] **Шаг 9.2: Написать тесты для useOrderingParams**

```typescript
test('toggles order direction', () => {})
```

Запуск: `cd ui && npm test useOrderingParams.test.ts`
Ожидается: PASS

```bash
git add ui/src/hooks/useOrderingParams.test.ts
git commit -m "test: add useOrderingParams hook tests"
```

- [ ] **Шаг 9.3: Написать тесты для useScrollPagination**

```typescript
test('triggers load on scroll', () => {})
test('cleans up event listener', () => {})
```

Запуск: `cd ui && npm test useScrollPagination.test.ts`
Ожидается: PASS

```bash
git add ui/src/hooks/useScrollPagination.test.ts
git commit -m "test: add useScrollPagination hook tests"
```

- [ ] **Шаг 9.4: Валидация задачи — проверить сборку и запуск контейнера**

```bash
./scripts/validate-test-build.sh
```

Ожидается: VALIDATION PASSED

---

### Задача 10: Pages Tests

**Файлы:**
- Создать: `ui/src/Pages/AlbumsPage.test.tsx`
- Создать: `ui/src/Pages/TimelinePage.test.tsx`
- Создать: `ui/src/Pages/SettingsPage.test.tsx`

**Приоритет:** MEDIUM

- [ ] **Шаг 10.1: Написать базовые рендер тесты**

```typescript
test('renders page without crashing', () => {})
test('shows loading state', () => {})
test('shows error state', () => {})
```

Запуск: `cd ui && npm test -- --run`
Ожидается: PASS

```bash
git add ui/src/Pages/AlbumsPage.test.tsx
git commit -m "test: add AlbumsPage render tests"
```

```bash
git add ui/src/Pages/TimelinePage.test.tsx
git commit -m "test: add TimelinePage render tests"
```

```bash
git add ui/src/Pages/SettingsPage.test.tsx
git commit -m "test: add SettingsPage render tests"
```

- [ ] **Шаг 10.2: Валидация задачи — проверить сборку и запуск контейнера**

```bash
./scripts/validate-test-build.sh
```

Ожидается: VALIDATION PASSED

---

## ЭТАП 4: EDGE CASES, PERFORMANCE, E2E

### Задача 11: Performance Benchmarks

**Файлы:**
- Создать: `api/scanner/scanner_benchmark_test.go`
- Создать: `api/database/database_benchmark_test.go`

**Приоритет:** LOW

- [ ] **Шаг 11.1: Написать бенчмарки**

```go
func BenchmarkFindAlbumsForUser_100Albums(b *testing.B)
func BenchmarkScannerQueue_Process_100Jobs(b *testing.B)
```

Запуск: `cd api && go test -bench=. ./scanner ./database -benchmem`
Ожидается: Базовая производительность

```bash
git add api/scanner/scanner_benchmark_test.go
git commit -m "test: add scanner benchmarks"
```

```bash
git add api/database/database_benchmark_test.go
git commit -m "test: add database benchmarks"
```

- [ ] **Шаг 11.2: Валидация задачи — проверить сборку и запуск контейнера**

```bash
./scripts/validate-test-build.sh
```

Ожидается: VALIDATION PASSED

---

## ФИНАЛЬНАЯ ПРОВЕРКА

### Backend Verification

```bash
# Все тесты должны проходить
cd api
go test ./... -short -count=1 -race

# С coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Ожидается: Покрытие > 65%

### Frontend Verification

```bash
cd ui
npm test -- --coverage
```

Ожидается: Покрытие > 60%

### Race Condition Check

```bash
cd api
go test ./... -race -short
```

Ожидается: NO RACE CONDITIONS

### Full Container Validation

```bash
./scripts/validate-test-build.sh
```

Ожидается: VALIDATION PASSED

---

## КРИТИЧЕСКИЕ ФАЙЛЫ ДЛЯ РЕАЛИЗАЦИИ

1. **`api/database/database.go`** — Ядро стабильности: подключение, миграции, retry logic
2. **`api/scanner/scanner_queue/queue.go`** — Concurrent worker pool, recent race condition fixes
3. **`api/graphql/directive.go`** — Security слой: @isAuthorized, @isAdmin
4. **`ui/src/apolloClient.ts`** — GraphQL конфигурация: WebSocket, error handling
5. **`ui/src/components/photoGallery/ProtectedMedia.tsx`** — Auth media loading, lazy loading

---

## СТАТИСТИКА

| Модуль | Текущее | Целевое | Тип |
|--------|---------|---------|-----|
| database/ | 0% | 80% | Integration |
| scanner_queue/ | 30% | 90% | Unit + Race |
| graphql/directive | 0% | 100% | Unit |
| graphql/resolvers | 20% | 70% | Integration |
| apolloClient.ts | 0% | 80% | Unit |
| ProtectedMedia.tsx | 0% | 80% | Unit |

**До:** ~15-20% покрытия
**После:** ~65-75% покрытия
**Критические модули:** 80%+

---

## ТРЕБОВАНИЯ ИЗ PROMPT.MD

- **NO panic()** — все ошибки обрабатываются
- **ALL errors must be handled** — явная обработка
- **Use context.Context for I/O** — все I/O с context
- **Unit tests mandatory** — для всех функций
- **Integration tests if DB involved** — для работы с БД
- **Graceful shutdown for goroutines** — корректное завершение

---

## НЕОБХОДИМЫЕ ИНСТРУМЕНТЫ

### Go
```bash
go get github.com/DATA-DOG/go-sqlmock
go get github.com/stretchr/testify
```

### TypeScript
```bash
npm install --save-dev @testing-library/react @testing-library/jest-dom @testing-library/user-event msw vitest
```
