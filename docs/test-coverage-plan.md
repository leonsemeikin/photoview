# План покрытия тестами проекта Photoview

> **Для агентов-исполнителей:** ОБЯЗАТЕЛЬНЫЙ НАВЫК: Используйте superpowers:subagent-driven-development (рекомендуется) или superpowers:executing-plans для реализации этого плана пошагово. Шаги используют синтаксис чекбокса (`- [ ]`) для отслеживания.

**Цель:** Создать комплексное покрытие тестами для Photoview (self-hosted photo gallery) — Go backend и React/TypeScript frontend

**Архитектура:** Go GraphQL API + React UI, SQLite/MariaDB/PostgreSQL, Scanner pipeline для медиа

**Технологии:** Go 1.26, gqlgen, GORM, React 18, TypeScript, Vite, Apollo Client

---

## Контекст

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

## ЭТАП 1: КРИТИЧЕСКИЕ ТЕСТЫ ДЛЯ BACKEND STABILITY

### Задача 1: Database Layer Tests

**Файлы:**
- Создать: `api/database/database_test.go`
- Создать: `api/database/address_test.go`
- Модифицировать: `api/test_utils/env_test.go` (добавить helper'ы)

**Приоритет:** CRITICAL

- [ ] **Шаг 1: Создать helpers для тестов БД**

```go
// api/test_utils/fixtures.go
package test_utils

func CreateTestDatabase(t *testing.T) *gorm.DB
func CleanupTestDatabase(db *gorm.DB)
```

- [ ] **Шаг 2: Написать тест для SQLite подключения**

Запуск: `cd api && go test ./database -run TestSetupDatabase_SQLite -v`
Ожидается: PASS

- [ ] **Шаг 3: Написать тест для MySQL подключения**

Запуск: `cd api && go test ./database -run TestSetupDatabase_MySQL -v`
Ожидается: PASS (или SKIP если нет MySQL)

- [ ] **Шаг 4: Написать тест для PostgreSQL подключения**

Запуск: `cd api && go test ./database -run TestSetupDatabase_Postgres -v`
Ожидается: PASS (или SKIP если нет PostgreSQL)

- [ ] **Шаг 5: Написать тест для retry логики**

```go
func TestSetupDatabase_RetryLogic(t *testing.T)
```

Запуск: `cd api && go test ./database -run TestSetupDatabase_RetryLogic -v`
Ожидается: 5 попыток при ошибке

- [ ] **Шаг 6: Написать тест для WAL режима SQLite**

```go
func TestGetSqliteAddress_WALMode(t *testing.T)
```

Запуск: `cd api && go test ./database -run TestGetSqliteAddress_WALMode -v`
Ожидается: `_journal_mode=WAL` в URL

- [ ] **Шаг 7: Написать тесты для миграций**

```go
func TestMigrateDatabase_AutoMigrate(t *testing.T)
func TestClearDatabase_AllModels(t *testing.T)
```

Запуск: `cd api && go test ./database -run TestMigrate -v`
Ожидается: PASS

- [ ] **Шаг 8: Commit**

```bash
git add api/database/database_test.go api/test_utils/fixtures.go
git commit -m "test: add database layer integration tests"
```

---

### Задача 2: Scanner Queue Concurrency Tests

**Файлы:**
- Модифицировать: `api/scanner/scanner_queue/queue_test.go`
- Создать: `api/scanner/scanner_queue/queue_race_test.go`

**Приоритет:** CRITICAL

- [ ] **Шаг 1: Написать тест concurrent jobs**

```go
func TestScannerQueue_ConcurrentJobs(t *testing.T) {
    // Запустить 10 job одновременно
    // Проверить что все завершены
}
```

Запуск: `cd api && go test ./scanner/scanner_queue -race -run TestScannerQueue_ConcurrentJobs -v`
Ожидается: PASS, NO RACE CONDITIONS

- [ ] **Шаг 2: Написать тест для notify channel blocking**

```go
func TestScannerQueue_NotifyChannelBlocking(t *testing.T)
```

Запуск: `cd api && go test ./scanner/scanner_queue -run TestScannerQueue_NotifyChannelBlocking -v`
Ожидается: Buffer 100 предотвращает deadlock

- [ ] **Шаг 3: Написать тест graceful shutdown**

```go
func TestScannerQueue_CloseBackgroundWorker(t *testing.T)
```

Запуск: `cd api && go test ./scanner/scanner_queue -run TestScannerQueue_CloseBackgroundWorker -v`
Ожидается: Все jobs завершены перед shutdown

- [ ] **Шаг 4: Написать тест non-fatal errors**

```go
func TestAddUserToQueue_NonFatalErrors(t *testing.T)
```

Запуск: `cd api && go test ./scanner/scanner_queue -run TestAddUserToQueue_NonFatalErrors -v`
Ожидается: Permission errors не блокируют очередь

- [ ] **Шаг 5: Commit**

```bash
git add api/scanner/scanner_queue/queue_test.go api/scanner/scanner_queue/queue_race_test.go
git commit -m "test: add scanner queue concurrency tests"
```

---

### Задача 3: GraphQL Directives Tests

**Файлы:**
- Создать: `api/graphql/directive_test.go`

**Приоритет:** CRITICAL

- [ ] **Шаг 1: Написать тест @isAuthorized**

```go
func TestIsAuthorized_WithUser(t *testing.T)
func TestIsAuthorized_WithoutUser(t *testing.T)
```

Запуск: `cd api && go test ./graphql -run TestIsAuthorized -v`
Ожидается: ErrUnauthorized без user

- [ ] **Шаг 2: Написать тест @isAdmin**

```go
func TestIsAdmin_AdminUser(t *testing.T)
func TestIsAdmin_RegularUser(t *testing.T)
func TestIsAdmin_NoUser(t *testing.T)
```

Запуск: `cd api && go test ./graphql -run TestIsAdmin -v`
Ожидается: Error для non-admin

- [ ] **Шаг 3: Commit**

```bash
git add api/graphql/directive_test.go
git commit -m "test: add GraphQL directive security tests"
```

---

## ЭТАП 2: GRAPHQL RESOLVERS И БИЗНЕС-ЛОГИКА

### Задача 4: Album Actions Tests

**Файлы:**
- Создать: `api/graphql/models/actions/album_actions_detail_test.go`

**Приоритет:** HIGH

- [ ] **Шаг 1: Написать тест для getTopLevelAlbumIDs**

```go
func TestGetTopLevelAlbumIDs_SingleUser(t *testing.T)
func TestGetTopLevelAlbumIDs_MultiUser(t *testing.T)
func TestGetTopLevelAlbumIDs_SubAlbumScenario(t *testing.T)
```

Запуск: `cd api && go test ./graphql/models/actions -run TestGetTopLevelAlbumIDs -v`
Ожидается: Правильная фильтрация top-level albums

- [ ] **Шаг 2: Commit**

```bash
git add api/graphql/models/actions/album_actions_detail_test.go
git commit -m "test: add album ownership logic tests"
```

---

### Задача 5: Media Resolvers Tests

**Файлы:**
- Создать: `api/graphql/resolvers/media_resolver_test.go`

**Приоритет:** HIGH

- [ ] **Шаг 1: Написать тест Thumbnail с dataloader**

```go
func TestMediaResolver_Thumbnail_Dataloader(t *testing.T)
```

Запуск: `cd api && go test ./graphql/resolvers -run TestMediaResolver_Thumbnail -v`
Ожидается: Батчинг работает (1 SQL запрос вместо N)

- [ ] **Шаг 2: Написать тест favorite авторизации**

```go
func TestMediaResolver_Favorite_Unauthorized(t *testing.T)
```

Запуск: `cd api && go test ./graphql/resolvers -run TestMediaResolver_Favorite -v`
Ожидается: Ошибка без авторизации

- [ ] **Шаг 3: Commit**

```bash
git add api/graphql/resolvers/media_resolver_test.go
git commit -m "test: add media resolver tests"
```

---

### Задача 6: Scanner Tasks Tests

**Файлы:**
- Создать: `api/scanner/scanner_tasks/exif_task_test.go`
- Создать: `api/scanner/scanner_tasks/blurhash_task_test.go`
- Создать: `api/scanner/scanner_tasks/video_metadata_task_test.go`

**Приоритет:** MEDIUM

- [ ] **Шаг 1: EXIF task тесты**

```go
func TestSaveEXIF_NewMedia(t *testing.T)
func TestSaveEXIF_ParseError(t *testing.T)
```

- [ ] **Шаг 2: Blurhash task тесты**

```go
func TestGenerateBlurhashFromThumbnail_ValidImage(t *testing.T)
```

- [ ] **Шаг 3: Video metadata тесты**

```go
func TestVideoMetadataTask_ValidVideo(t *testing.T)
func TestVideoMetadataTask_FFprobeError(t *testing.T)
```

- [ ] **Шаг 4: Commit**

```bash
git add api/scanner/scanner_tasks/*_test.go
git commit -m "test: add scanner task unit tests"
```

---

## ЭТАП 3: UI COMPONENTS И USER FLOWS

### Задача 7: Apollo Client Tests

**Файлы:**
- Создать: `ui/src/apolloClient.test.ts`

**Приоритет:** HIGH

- [ ] **Шаг 1: Установить зависимости**

```bash
cd ui
npm install --save-dev @testing-library/react @testing-library/jest-dom vitest
```

- [ ] **Шаг 2: Написать тест HTTP link конфигурации**

```typescript
test('configures HTTP link correctly', () => {
  expect(APPLICATION_ENDPOINT).toBe('/api')
})
```

Запуск: `cd ui && npm test apolloClient.test.ts`
Ожидается: PASS

- [ ] **Шаг 3: Написать тест WebSocket split**

```typescript
test('splits subscriptions to WebSocket', () => {
  // Проверить что subscription идет через WS
})
```

- [ ] **Шаг 4: Написать тест error handler**

```typescript
test('error handler clears token on 401', () => {})
test('error handler shows GraphQL errors', () => {})
```

- [ ] **Шаг 5: Написать тест cache pagination**

```typescript
test('cache pagination merges correctly', () => {})
```

- [ ] **Шаг 6: Commit**

```bash
git add ui/src/apolloClient.test.ts ui/package.json ui/package-lock.json
git commit -m "test: add Apollo client configuration tests"
```

---

### Задача 8: Protected Media Tests

**Файлы:**
- Создать: `ui/src/components/photoGallery/ProtectedMedia.test.tsx`

**Приоритет:** HIGH

- [ ] **Шаг 1: Написать тест token appending**

```typescript
test('appends token to URL from share path', () => {
  // Проверить что ?token=X добавляется к URL
})
```

- [ ] **Шаг 2: Написать тест lazy loading**

```typescript
test('uses native lazy loading when supported', () => {})
test('falls back to IntersectionObserver', () => {})
```

- [ ] **Шаг 3: Написать тест blurhash**

```typescript
test('shows blurhash while loading', () => {})
test('hides blurhash after loaded', () => {})
```

- [ ] **Шаг 4: Commit**

```bash
git add ui/src/components/photoGallery/ProtectedMedia.test.tsx
git commit -m "test: add ProtectedMedia component tests"
```

---

### Задача 9: Custom Hooks Tests

**Файлы:**
- Создать: `ui/src/hooks/useURLParameters.test.ts`
- Создать: `ui/src/hooks/useOrderingParams.test.ts`
- Создать: `ui/src/hooks/useScrollPagination.test.ts`

**Приоритет:** MEDIUM

- [ ] **Шаг 1: Написать тесты для useURLParameters**

```typescript
test('reads parameters from URL', () => {})
test('updates URL on change', () => {})
```

- [ ] **Шаг 2: Написать тесты для useOrderingParams**

```typescript
test('toggles order direction', () => {})
```

- [ ] **Шаг 3: Написать тесты для useScrollPagination**

```typescript
test('triggers load on scroll', () => {})
test('cleans up event listener', () => {})
```

- [ ] **Шаг 4: Commit**

```bash
git add ui/src/hooks/*.test.ts
git commit -m "test: add custom hooks tests"
```

---

### Задача 10: Pages Tests

**Файлы:**
- Создать: `ui/src/Pages/AlbumsPage.test.tsx`
- Создать: `ui/src/Pages/TimelinePage.test.tsx`
- Создать: `ui/src/Pages/SettingsPage.test.tsx`

**Приоритет:** MEDIUM

- [ ] **Шаг 1: Написать базовые рендер тесты**

```typescript
test('renders page without crashing', () => {})
test('shows loading state', () => {})
test('shows error state', () => {})
```

- [ ] **Шаг 2: Commit**

```bash
git add ui/src/Pages/*.test.tsx
git commit -m "test: add page component tests"
```

---

## ЭТАП 4: EDGE CASES, PERFORMANCE, E2E

### Задача 11: Performance Benchmarks

**Файлы:**
- Создать: `api/scanner/scanner_benchmark_test.go`
- Создать: `api/database/database_benchmark_test.go`

**Приоритет:** LOW

- [ ] **Шаг 1: Написать бенчмарки**

```go
func BenchmarkFindAlbumsForUser_100Albums(b *testing.B)
func BenchmarkScannerQueue_Process_100Jobs(b *testing.B)
```

Запуск: `cd api && go test -bench=. ./scanner ./database -benchmem`
Ожидается: Базовая производительность

- [ ] **Шаг 2: Commit**

```bash
git add api/*/*_benchmark_test.go
git commit -m "test: add performance benchmarks"
```

---

## ПРОВЕРКА

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
go get golang.org/x/sync
```

### TypeScript
```bash
npm install --save-dev @testing-library/react @testing-library/jest-dom @testing-library/user-event msw vitest
```
