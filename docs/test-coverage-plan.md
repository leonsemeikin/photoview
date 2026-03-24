# План покрытия тестами проекта Photoview

> **Для агентов-исполнителей:** ОБЯЗАТЕЛЬНЫЙ НАВЫК: Используйте superpowers:subagent-driven-development (рекомендуется) или superpowers:executing-plans для реализации этого плана пошагово. Шаги используют синтаксис чекбокса (`- [ ]`) для отслеживания.

**Цель:** Создать комплексное покрытие тестами для Photoview (self-hosted photo gallery) — Go backend и React/TypeScript frontend

**Архитектура:** Go GraphQL API + React UI, SQLite/MariaDB/PostgreSQL, Scanner pipeline для медиа

**Технологии:** Go 1.26, gqlgen, GORM, React 18, TypeScript, Vite, Apollo Client

---

## ПРОГРЕСС

- [x] Этап 0: Подготовка (1/10) ✅ Шаг 0.1 выполнен
- [ ] Этап 1: Backend Stability (0/3 задачи)
- [ ] Этап 2: GraphQL (0/4 задачи) — обновлено
- [ ] Этап 3: UI (0/5 задач) — обновлено
- [ ] Этап 4: Performance (0/1 задача)

Overall: 1/69 шагов (1%)

---

## MOCKING STRATEGY

### Backend
- **sqlmock** для database layer (изолированные unit тесты)
- **httptest** для GraphQL endpoints (интеграционные тесты)
- **test fixtures** в `api/test_utils/fixtures.go` для повторного использования данных

### UI
- **MSW (Mock Service Worker)** для перехвата GraphQL запросов и ответов
- **MockApolloProvider** для component tests с мокированным клиентом
- **Jest mocks** для IntersectionObserver, Image loading API

---

## CHECKPOINTS FOR MERGE

После завершения этапа → PR + CI pass:

- [ ] **Checkpoint 1:** Этап 0 + Задачи 1-3 (Database + Concurrency + Security)
  - Покрытие database: ~80%
  - Покрытие scanner_queue: ~90%
  - Покрытие graphql/directive: 100%
- [ ] **Checkpoint 2:** Этап 2 (GraphQL Resolvers + Scanner User)
  - Покрытие graphql/resolvers: ~70%
  - Альбом actions, media resolvers, scanner_user протестированы
  - periodic_scanner и routes протестированы
- [ ] **Checkpoint 3:** Этап 3 (UI Components)
  - Покрытие UI: ~60%
  - Apollo Client, ProtectedMedia, PhotoGallery, AlbumPage протестированы
  - Hooks и pages протестированы
- [ ] **Checkpoint 4:** Этап 4 (Performance + Edge Cases)
  - Бенчмарки для критических путей с acceptance criteria
  - N+1 detection тесты
  - Full stack validation

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

### Исключения из тестирования

**ВАЖНО:** Функционал распознавания лиц (**face detection**) **НЕ тестируется и не используется**:

- На проде отключён через `PHOTOVIEW_DISABLE_FACE_RECOGNITION=true`
- Пользователь не использует эту функцию
- Тесты для `api/scanner/face_detection/` **НЕ пишутся**
- Тесты для `api/graphql/models/face_group.go` и `ImageFace` **НЕ пишутся**
- Модули, связанные с face recognition, пропускаются в плане тестирования

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

- [x] **Шаг 0.1: Проверить и исправить текущие тесты в GitHub** ✅ ВЫПОЛНЕНО

```bash
# 1. Запустить сборку в GitHub Actions для проверки текущих тестов
git push origin HEAD:fix-current-tests  # Создать ветку, запустить CI

# 2. Дождаться завершения тестов на GitHub
# 3. Проанализировать ошибки в логах тестов
# 4. Зафиксировать проблемы в файле docs/current-test-issues.md
# 5. Применить исправления в коде
# 6. Повторять push до тех пор, пока все тесты не пройдут

# Команды для локальной отладки (если нужно):
cd api
go test ./... -v -database -filesystem -p 1  # Запуск как в CI

cd ../ui
CI=true vitest --reporter=junit --reporter=verbose --run --coverage  # Запуск как в CI
```

**РЕЗУЛЬТАТ:** Все существующие тесты PASS в GitHub Actions ✅

**НАЙДЕННЫЕ ПРОБЛЕМЫ И ИСПРАВЛЕНИЯ:**

1. **Главная причина: Shell скрипты без прав на выполнение**
   - Проблема: `scripts/test_all.sh` и другие скрипты имели права `644` вместо `755`
   - Ошибка CI: `exec: "/app/scripts/test_all.sh": permission denied`
   - Исправление: `chmod +x scripts/*.sh`
   - Затронуто: 7 файлов (benchmark_api.sh, install_*.sh, set_compiler_env.sh, test_*.sh)

2. **Mock binaries без прав на выполнение**
   - Проблема: `api/scanner/media_encoding/executable_worker/test_data/mock_bin/*` имели права `644`
   - Ошибки тестов: `TestInitFfprobePath/Succeed`, `TestFfmpeg`, `TestFfmpegWithHWAcc`, `TestFfmpegWithCustomCodec`
   - Исправление: `chmod +x test_data/mock_bin/*`
   - Затронуто: ffmpeg, ffprobe, magick

3. **Тесты использовали UnitTestRun вместо IntegrationTestRun**
   - Проблема: `test_utils.UnitTestRun` не загружает `testing.env`, поэтому переменные БД не доступны
   - Затронуто 3 файла:
     - `api/graphql/auth/auth_test.go`
     - `api/graphql/endpoint/graphql_endpoint_test.go`
     - `api/scanner/periodic_scanner/periodic_scanner_test.go`
   - Исправление: Замена `UnitTestRun(m)` на `IntegrationTestRun(m)`

4. **Изменения в album_actions.go ломали тесты**
   - Проблема: Из `patch-album-fix` ветки были изменения в `getTopLevelAlbumIDs()`, которые ломали `TestAlbumsSingleRootExpand` и `TestNonRootAlbumPath`
   - Исправление: Откат `api/graphql/models/actions/album_actions.go` к версии master

5. **Создан testing.env для локального тестирования**
   - Добавлен `api/testing.env` с настройками SQLite для удобного локального запуска тестов
   - Файл в `.gitignore`, так как используется только для локальной разработки

**ПРИМЕЧАНИЕ:** Сообщение про "go generate ./..." в логах CI было ложным следствием - сгенерированный GraphQL код был синхронизирован.

**ВАЖНОЕ ИСПРАВЛЕНИЕ (2026-03-24):**

**Коммит `dfffaff96cb1458aca49c87c364736c3d2ef0816` из ветки `patch-album-fix` фиксит важный баг:**

- **Проблема:** У разных пользователей не отображались одни и те же альбомы в UI
- **Сценарий:** Пользователь A имеет root path `/photos`, Пользователь B добавляется с `/photos/userB`
- **До фикса:** Все альбомы Пользователя B имеют `parent_album_id` указывающий на альбомы Пользователя A, но у Пользователя B нет альбомов с `parent_album_id IS NULL`. Функция `MyAlbums()` с флагом `onlyRoot=true` возвращала пустой список.
- **Решение:** Добавлена функция `getTopLevelAlbumIDs()`, которая определяет "top-level" альбомы для конкретного пользователя — это либо root альбомы, либо прямые потомки альбомов, которые НЕ принадлежат пользователю.
- **Изменённые файлы:** `api/graphql/models/actions/album_actions.go`
- **Затронутые тесты:** `TestAlbumsSingleRootExpand`, `TestNonRootAlbumPath`, `TestNonRootAlbumPathMultipleUsers` — требуют корректного создания альбомов в БД перед связыванием с пользователями

**Коммит `d219ad15e138eb1f84c3b8145a21e66b20443a74` фиксит другой важный баг:**

- **Проблема:** Папки не сканировались при возникновении ошибки в другой папке
- **Сценарий:** При сканировании `/photos`, если папка `/photos/Моё Др 2023` давала ошибку разрешения, то ВСЕ остальные папки тоже не сканировались
- **Причина:** Функция `AddUserToQueue()` в `api/scanner/scanner_queue/queue.go` прерывала выполнение при любой ошибке от `FindAlbumsForUser()`, даже если это была ошибка доступа к одной папке
- **Решение:** Изменена логика обработки ошибок:
  - Non-fatal ошибки (permission denied, file not found) логируются но не прерывают сканирование
  - Только критические ошибки (database connection) прерывают выполнение
  - Добавлен `continue` в цикле обработки ошибок
- **Затронутые файлы:** `api/scanner/scanner_queue/queue.go`
- **Важность:** Без этого фикса даже одна недоступная папка блокировала сканирование всей библиотеки

- [ ] **Шаг 0.2: Создать docker-compose для тестирования**

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

- [ ] **Шаг 0.3: Создать директорию для тестовых данных**

```bash
mkdir -p test-data
# Добавить несколько тестовых изображений (можно symlink на существующие)
```

- [ ] **Шаг 0.4: Проверить что базовый контейнер собирается и стартует**

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

- [ ] **Шаг 0.5: Установить Go зависимости для тестирования**

```bash
cd api
go get github.com/DATA-DOG/go-sqlmock
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/mock
go mod tidy
```

- [ ] **Шаг 0.6: Установить Node.js зависимости для тестирования**

```bash
cd ui
npm install --save-dev @testing-library/react @testing-library/jest-dom @testing-library/user-event vitest @vitest/ui jsdom msw
```

- [ ] **Шаг 0.7: Проверить существующие тесты (как в GitHub Actions)**

```bash
# Запуск Go тестов с coverage (как в CI)
cd api
go test ./... -v -database -filesystem -p 1 \
  -cover -coverpkg=./... -coverprofile=coverage.txt -covermode=atomic

# Проверить coverage
go tool cover -func=coverage.txt | grep total

# Запуск UI тестов с coverage (как в CI)
cd ../ui
CI=true vitest --reporter=junit --reporter=verbose --run --coverage
```

Ожидается:
- Go тесты: PASS (29 тестовых файлов)
- UI тесты: PASS (21 тестовый файл)
- Покрытие фиксируется как baseline

**Существующие тесты (29 Go, 21 TS):**
- Go: `api/scanner/*_test.go`, `api/graphql/**/*_test.go`, `api/routes/*_test.go`, `api/database/**/*_test.go`
- UI: `ui/src/**/*.test.ts`, `ui/src/**/*.test.tsx`

- [ ] **Шаг 0.8: Проверить генерируемый код (как в CI)**

```bash
# Проверить что GraphQL сгенерирован корректно
cd api
go generate ./...

# Проверить что нет изменений
if [ "$(git status -s 2>/dev/null | head -1)" != "" ]; then
  echo 'FAIL: Generated code is out of sync'
  git status -s
  exit 1
fi

echo 'PASS: All generated code is in sync'
```

Ожидается: PASS — весь сгенерированный код в синхронизации

- [ ] **Шаг 0.9: Создать скрипт для валидации после каждой задачи**

```bash
# Создать файл scripts/validate-test-build.sh
```

Содержимое `scripts/validate-test-build.sh`:
```bash
#!/bin/bash
set -e

echo "=== 1. Checking generated code sync ==="
cd api
go generate ./...
if [ "$(git status -s 2>/dev/null | head -1)" != "" ]; then
    echo "FAILED: Generated code is out of sync"
    git status -s
    exit 1
fi
echo "PASS: Generated code is in sync"

echo "=== 2. Running Go tests (as in CI) ==="
cd api
go test ./... -v -database -filesystem -p 1 \
  -cover -coverpkg=./... -coverprofile=coverage.txt -covermode=atomic

echo "=== 3. Building test container ==="
cd ..
docker compose -f docker-compose.test.yml build --no-cache

echo "=== 4. Starting container ==="
docker compose -f docker-compose.test.yml up -d

echo "=== 5. Waiting for healthy status (timeout 60s) ==="
timeout 60s bash -c 'until docker compose -f docker-compose.test.yml ps | grep -q "healthy"; do sleep 2; done' || {
    echo "FAILED: Container did not become healthy"
    docker compose -f docker-compose.test.yml logs
    docker compose -f docker-compose.test.yml down
    exit 1
}

echo "=== 6. Checking health status ==="
docker compose -f docker-compose.test.yml ps

echo "=== 7. Stopping container ==="
docker compose -f docker-compose.test.yml down

echo "=== 8. Running UI tests (as in CI) ==="
cd ui
CI=true vitest --reporter=junit --reporter=verbose --run --coverage

echo "=== VALIDATION PASSED ==="
```

```bash
chmod +x scripts/validate-test-build.sh
```

- [ ] **Шаг 0.10: Commit подготовительных файлов**

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

### Задача 6a: Scanner User & Periodic Scanner Tests

**Файлы:**
- Создать: `api/scanner/scanner_user_test.go`
- Создать: `api/scanner/periodic_scanner/periodic_scanner_test.go`
- Создать: `api/routes/routes_test.go`

**Приоритет:** HIGH

**Почему это критично:** `FindAlbumsForUser()` содержит owner propagation логику, которая была источником багов. Periodic scanner может крашиться при ошибке БД.

- [ ] **Шаг 6a.1: Написать тест FindAlbumsForUser owner propagation**

```go
func TestFindAlbumsForUser_OwnerPropagation(t *testing.T)
func TestFindAlbumsForUser_NestedAlbums(t *testing.T)
func TestFindAlbumsForUser_PermissionDenied(t *testing.T)
```

Запуск: `cd api && go test ./scanner -run TestFindAlbumsForUser -v`
Ожидается: PASS, корректная propagation owners

```bash
git add api/scanner/scanner_user_test.go
git commit -m "test: add FindAlbumsForUser owner propagation tests"
```

- [ ] **Шаг 6a.2: Написать тест periodic scanner restart**

```go
func TestPeriodicScanner_RestartOnError(t *testing.T)
func TestPeriodicScanner_GracefulShutdown(t *testing.T)
```

Запуск: `cd api && go test ./scanner/periodic_scanner -v`
Ожидается: PASS, корректный restart и shutdown

```bash
git add api/scanner/periodic_scanner/periodic_scanner_test.go
git commit -m "test: add periodic scanner restart tests"
```

- [ ] **Шаг 6a.3: Написать тест routes 401 handling**

```go
func TestRoutes_AuthRequired_WithoutToken(t *testing.T)
func TestRoutes_CORS_Headers(t *testing.T)
```

Запуск: `cd api && go test ./routes -v`
Ожидается: PASS, 401 без токена

```bash
git add api/routes/routes_test.go
git commit -m "test: add routes auth and CORS tests"
```

- [ ] **Шаг 6a.4: Валидация задачи — проверить сборку и запуск контейнера**

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

---

### Задача 11: PhotoGallery Component Tests

**Файлы:**
- Создать: `ui/src/components/photoGallery/PhotoGallery.test.tsx`
- Создать: `ui/src/Pages/AlbumPage.test.tsx`

**Приоритет:** HIGH

**Почему это критично:** PhotoGallery — основной компонент для отображения медиа, AlbumPage — основной роут. Отсутствие тестов означает риск краша при edge cases (пустой альбом, загрузка ошибок).

- [ ] **Шаг 11.1: Написать тест PhotoGallery**

```typescript
test('renders empty state when no media', () => {})
test('renders media grid with items', () => {})
test('handles loading state', () => {})
test('handles error state', () => {})
test('calls onScrollEnd when scrolling', () => {})
```

Запуск: `cd ui && npm test PhotoGallery.test.tsx`
Ожидается: PASS

```bash
git add ui/src/components/photoGallery/PhotoGallery.test.tsx
git commit -m "test: add PhotoGallery component tests"
```

- [ ] **Шаг 11.2: Написать тест AlbumPage**

```typescript
test('renders album info', () => {})
test('redirects on 404', () => {})
test('shows loading skeleton', () => {})
test('handles share token', () => {})
```

Запуск: `cd ui && npm test AlbumPage.test.tsx`
Ожидается: PASS

```bash
git add ui/src/Pages/AlbumPage.test.tsx
git commit -m "test: add AlbumPage tests"
```

- [ ] **Шаг 11.3: Валидация задачи — проверить сборку и запуск контейнера**

```bash
./scripts/validate-test-build.sh
```

Ожидается: VALIDATION PASSED

---

### Задача 12: Pages Tests (Update)

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

### Задача 13: Performance Benchmarks

**Файлы:**
- Создать: `api/scanner/scanner_benchmark_test.go`
- Создать: `api/database/database_benchmark_test.go`

**Приоритет:** LOW

**ACCEPTANCE CRITERIA:**
| Benchmark | Target | Notes |
|-----------|--------|-------|
| `BenchmarkFindAlbumsForUser_100` | < 1ms/op | на dev machine |
| `BenchmarkFindAlbumsForUser_1000` | < 10ms/op | N+1 detection |
| `BenchmarkScannerQueue_Process_100` | < 5ms/op | per job |
| `BenchmarkDatabase_SQLite_Insert` | < 0.1ms/op | single insert |
| `BenchmarkDatabase_SQLite_Select` | < 0.5ms/op | indexed query |

- [ ] **Шаг 13.1: Написать бенчмарки для FindAlbumsForUser**

```go
func BenchmarkFindAlbumsForUser_10(b *testing.B)
func BenchmarkFindAlbumsForUser_100(b *testing.B)
func BenchmarkFindAlbumsForUser_1000(b *testing.B)
```

**Критерий:** О(N) или лучше, не O(N²). Если 1000 albums > 100× медленнее чем 10 albums — есть N+1 проблема.

Запуск: `cd api && go test -bench=BenchmarkFindAlbumsForUser -benchmem ./graphql/models/actions`
Ожидается: Линейная или sub-linear сложность

```bash
git add api/graphql/models/actions/album_actions_benchmark_test.go
git commit -m "test: add FindAlbumsForUser benchmarks"
```

- [ ] **Шаг 13.2: Написать бенчмарки для Scanner Queue**

```go
func BenchmarkScannerQueue_Process_10Jobs(b *testing.B)
func BenchmarkScannerQueue_Process_100Jobs(b *testing.B)
func BenchmarkScannerQueue_Process_1000Jobs(b *testing.B)
```

**Критерий:** Const latency per job независимо от queue size

Запуск: `cd api && go test -bench=BenchmarkScannerQueue -benchmem ./scanner/scanner_queue`
Ожидается: < 5ms/op

```bash
git add api/scanner/scanner_queue/queue_benchmark_test.go
git commit -m "test: add ScannerQueue benchmarks"
```

- [ ] **Шаг 13.3: Написать бенчмарки для Database операций**

```go
func BenchmarkDatabase_SQLite_Insert(b *testing.B)
func BenchmarkDatabase_SQLite_Select_Indexed(b *testing.B)
func BenchmarkDatabase_SQLite_Select_FullScan(b *testing.B)
```

**Критерий:** Indexed select > 10× быстрее чем full scan

Запуск: `cd api && go test -bench=BenchmarkDatabase -benchmem ./database`
Ожидается: Indexed queries значительно быстрее

```bash
git add api/database/database_benchmark_test.go
git commit -m "test: add database operation benchmarks"
```

- [ ] **Шаг 13.4: N+1 Detection тест**

```go
func TestAlbumResolvers_NoNPlusOneQueries(t *testing.T)
```

Использовать `sqltrace` или `go-sqlmock` для подсчёта SQL запросов.

**Критерий:** 1 запрос для album, +1 запрос для всех thumbnails (batched), NOT 1 запрос per thumbnail.

Запуск: `cd api && go test ./graphql/resolvers -run TestAlbumResolvers_NoNPlusOneQueries -v`
Ожидается: PASS (количество запросов не зависит от количества media)

```bash
git add api/graphql/resolvers/resolver_nplusone_test.go
git commit -m "test: add N+1 query detection test"
```

- [ ] **Шаг 13.5: Валидация задачи — проверить сборку и запуск контейнера**

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

**Backend:**
1. **`api/database/database.go`** — Ядро стабильности: подключение, миграции, retry logic
2. **`api/scanner/scanner_queue/queue.go`** — Concurrent worker pool, recent race condition fixes
3. **`api/graphql/directive.go`** — Security слой: @isAuthorized, @isAdmin
4. **`api/scanner/scanner_user.go`** — Owner propagation, источник прошлых багов
5. **`api/scanner/periodic_scanner/`** — Periodic scanning, может крашиться
6. **`api/routes/routes.go`** — Auth handling, CORS

**Frontend:**
7. **`ui/src/apolloClient.ts`** — GraphQL конфигурация: WebSocket, error handling
8. **`ui/src/components/photoGallery/ProtectedMedia.tsx`** — Auth media loading, lazy loading
9. **`ui/src/components/photoGallery/PhotoGallery.tsx`** — Основной компонент галереи
10. **`ui/src/Pages/AlbumPage.tsx`** — Основной роут для альбомов

---

## ОГРАНИЧЕНИЯ И ИСКЛЮЧЕНИЯ

**Функционал, который НЕ тестируется:**
- **Face Recognition** — отключён на проде (`PHOTOVIEW_DISABLE_FACE_RECOGNITION=true`), пользователем не используется
  - `api/scanner/face_detection/` — НЕ тестировать
  - `api/graphql/models/face_group.go`, `ImageFace` — НЕ тестировать
  - Все тесты с `face_*` в имени пропускаются

**Ограничения тестовой инфраструктуры:**
- Сканер и API тесно связаны — тесты API требуют настроенную БД и файловую систему
- Некоторые тесты требуют настоящие бинарные инструменты (ffmpeg, imagemagick) или их моки
- Mock binaries в `test_data/mock_bin/` должны быть исполняемыми

---

## СТАТИСТИКА

| Модуль | Текущее | Целевое | Тип |
|--------|---------|---------|-----|
| database/ | 0% | 80% | Integration |
| scanner_queue/ | 30% | 90% | Unit + Race |
| scanner/scanner_user.go | 0% | 85% | Integration |
| scanner/periodic_scanner/ | 0% | 75% | Unit |
| graphql/directive | 0% | 100% | Unit |
| graphql/resolvers | 20% | 70% | Integration |
| routes/ | 10% | 60% | Integration |
| apolloClient.ts | 0% | 80% | Unit |
| ProtectedMedia.tsx | 0% | 80% | Unit |
| PhotoGallery.tsx | 0% | 75% | Component |
| AlbumPage.tsx | 0% | 70% | Component |
| hooks/ | 0% | 70% | Unit |

**До:** ~15-20% покрытия
**После:** ~68-75% покрытия
**Критические модули:** 80%+
**Всего файлов тестов:** 29 Go → ~55 Go, 3 TS → ~35 TS

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
