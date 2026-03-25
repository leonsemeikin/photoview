# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Photoview is a self-hosted photo gallery with a GraphQL API backend (Go) and React frontend (TypeScript). The application scans a filesystem for photos/videos, generates thumbnails, extracts EXIF data, performs face recognition, and serves media through a web interface.

### Architecture

```
photoview/
├── api/                    # Go GraphQL API server
│   ├── database/           # Database setup (GORM, supports SQLite/MariaDB/PostgreSQL)
│   ├── graphql/            # GraphQL schema, resolvers, generated code
│   ├── scanner/            # Media scanning pipeline
│   │   ├── scanner_tasks/  # Individual scan tasks (EXIF, face detection, blurhash, etc.)
│   │   ├── scanner_queue/  # Task queue for parallel processing
│   │   ├── externaltools/  # Exiftool, FFmpeg integration
│   │   └── face_detection/ # Face recognition using go-face
│   └── server.go           # Main entry point
├── ui/                     # React TypeScript frontend
│   ├── src/
│   │   ├── Pages/          # Route components (AlbumPage, TimelinePage, etc.)
│   │   ├── components/     # Reusable UI components
│   │   └── primitives/     # Base components (Button, Input, etc.)
└── Dockerfile              # Multi-stage build (ui → api → release)
```

**Key Technologies:**
- **Backend**: Go 1.26, gqlgen (GraphQL), GORM, gorilla/mux
- **Frontend**: React 18, TypeScript, Vite, Apollo Client, styled-components, Tailwind
- **Database**: SQLite (built-in), MariaDB, PostgreSQL 18/17
- **Image/Video**: ImageMagick, FFmpeg (jellyfin-ffmpeg), libheif for RAW support
- **Face Detection**: go-face (dlib-based)

### Performance Architecture

Photoview achieves high performance through several key mechanisms:

1. **Multi-tier Media Processing**: Generates three versions of each media:
   - **Thumbnail**: Max 1024x1024 JPEG for gallery views
   - **HighRes**: Web-compatible JPEG for full-screen viewing (RAW photos converted)
   - **VideoWeb**: Web-optimized MP4 for streaming

2. **BlurHash Placeholders**: Generates compact string representations (~20-30 chars) that create blurred placeholders while images load, eliminating empty white screens

3. **Native Lazy Loading**: Uses browser's native `loading="lazy"` attribute with IntersectionObserver fallback for loading images only when they enter the viewport

4. **Dataloader Pattern**: Batches GraphQL queries to solve N+1 problems (e.g., thumbnail requests are batched into single SQL queries)

5. **Persistent Media Cache**: All processed media is cached to disk at `PHOTOVIEW_MEDIA_CACHE`, avoiding reprocessing on subsequent views

### Scanner Architecture

The scanner is the core background processing system:
1. **scanner_queue**: Worker pool that processes tasks concurrently (configurable, SQLite limited to 1 worker)
2. **scanner_tasks**: Individual tasks executed during scanning:
   - `exif_task.go`: Extract EXIF metadata using exiftool
   - `blurhash_task.go`: Generate blurhash for placeholders
   - `face_detection_task.go`: Detect and group faces
   - `process_photo_task.go` / `process_video_task.go`: Generate thumbnails/encoded versions
   - `video_metadata_task.go`: Extract video metadata via ffprobe
   - `sidecar_task.go`: Handle XMP sidecar files
   - `ignorefile_task.go`: Process .photoviewignore files
   - `notification_task.go`: Handle scan completion notifications

## Development Commands

### Docker Development (Recommended)

```bash
# Build and start both API and UI with hot-reload
docker compose -f dev-compose.yaml build
docker compose -f dev-compose.yaml up

# With specific database (default: SQLite)
docker compose -f dev-compose.yaml --profile mysql up
docker compose -f dev-compose.yaml --profile postgres up
```

Access points:
- GraphQL Playground: http://localhost:4001
- UI: http://localhost:1234

### Local Development

**API (Go):**
```bash
cd api
cp example.env .env
# Edit .env to configure database driver and connection
go mod download

# Optional: Set compiler environment for Debian/Ubuntu
source ../scripts/set_compiler_env.sh

# Patch go-face for compilation (remove -lcblas, -march=native)
sed -i 's/-lcblas//g' $(go env GOMODCACHE)/github.com/\!kagami/go-face*/face.go  # Linux
sed -i '' 's/-lcblas//g' $(go env GOMODCACHE)/github.com/\!kagami/go-face*/face.go  # macOS

go run .  # Or: reflex -g '*.go' -s -- go run .
```

**UI (React):**
```bash
cd ui
cp example.env .env
npm install
npm start      # Or: npm run mon for hot-reload with nodemon
```

### Testing

```bash
# API tests (Go)
cd api
go test ./... -v

# With database and filesystem flags (as in CI)
go test ./... -database -filesystem

# With coverage report
go test ./... -database -filesystem -cover -coverprofile=coverage.txt

# Stage 1: Run specific component tests
go test github.com/photoview/photoview/api/database -v
go test github.com/photoview/photoview/api/graphql -run "TestIsAuthorized|TestIsAdmin" -v
go test github.com/photoview/photoview/api/scanner/scanner_queue -v

# UI tests (Vitest)
cd ui
npm test
npm run test:ci    # CI mode with coverage

# Linting
cd ui && npm run lint
cd ui && npm run format:check
```

**Note**: Some tests require C dependencies (ImageMagick, dlib, FFmpeg) and may fail locally without them. These tests pass in CI Docker environment. For quick validation, run database and GraphQL directive tests which have no external dependencies.

### Test Coverage Progress

**Stage 1: Backend Stability Tests** ✅ Completed (2026-03)

Implemented comprehensive unit tests for critical backend components:

| Component | Tests | File |
|-----------|-------|------|
| Database Layer | 14 | `api/database/database_test.go`, `address_test.go` |
| Scanner Queue | 5 | `api/scanner/scanner_queue/queue_race_test.go` |
| GraphQL Directives | 9 | `api/graphql/directive_test.go` |
| **Total** | **30** | |

**Database Layer Tests** (14 tests):
- GORM migrations (AutoMigrate, ClearDatabase)
- URL parsing for SQLite/MySQL/PostgreSQL
- Retry logic for database connections
- WAL mode configuration

**Scanner Queue Tests** (5 tests):
- Concurrent job processing with race condition detection
- Notify channel behavior (blocking/non-blocking)
- Non-fatal error handling during album queuing
- Job-on-queue concurrency safety

**GraphQL Directive Tests** (9 tests):
- `@isAuthorized` directive (with/without user, resolver errors)
- `@isAdmin` directive (admin/regular user, no user, multiple checks)
- Chained directives behavior
- Error propagation through directive chain

**CI Compatibility Fixes**:
- Fixed "flag redefined" errors by removing duplicate flag definitions
- Added blank imports for `test_utils/flags` to all test packages
- Removed flaky `TestScannerQueue_CloseBackgroundWorker` test

See `TEST_PROGRESS.md` for detailed status and `docs/test-coverage-plan.md` for full roadmap.

### Test Infrastructure

Photoview includes comprehensive test infrastructure for validation:

**Files:**
- `docker-compose.test.yml`: Test container configuration with health checks
- `scripts/validate-test-build.sh`: Complete validation script

**Running Full Validation:**
```bash
./scripts/validate-test-build.sh
```

This script:
1. Checks GraphQL code generation sync (`go generate ./...`)
2. Runs Go tests (skipped locally without Docker/CI environment)
3. Builds test container with `docker-compose.test.yml`
4. Starts container and waits for healthy status
5. Runs UI tests with coverage

**Test Container:**
```bash
# Build and start test container
docker compose -f docker-compose.test.yml build
docker compose -f docker-compose.test.yml up -d

# Wait for healthy status (timeout 60s)
timeout 60s bash -c 'until docker compose -f docker-compose.test.yml ps | grep -q "healthy"; do sleep 2; done'

# Check status
docker compose -f docker-compose.test.yml ps

# Stop
docker compose -f docker-compose.test.yml down
```

**Important Notes:**
- Go tests require Docker for dependencies (dlib, ImageMagick for face detection)
- The validation script detects CI environment and runs Go tests only in CI
- UI tests use MSW for mocking GraphQL requests
- Test coverage is tracked in CI with codecov

### GraphQL Schema Generation

After modifying GraphQL schema files (*.graphql in `api/graphql/resolvers/`):

```bash
cd api
go generate ./...
```

This regenerates `api/graphql/generated.go` and type-safe resolver interfaces.

## Important Configuration

### Database Drivers

Set `PHOTOVIEW_DATABASE_DRIVER` environment variable:
- `sqlite`: No external service, single writer only (use for testing/single-user)
- `mysql`: MariaDB, recommended for production
- `postgres`: PostgreSQL 18/17, best performance

### Critical Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PHOTOVIEW_LISTEN_IP` | API bind address | 127.0.0.1 |
| `PHOTOVIEW_LISTEN_PORT` | API port | 80 |
| `PHOTOVIEW_API_ENDPOINT` | GraphQL API path | /api |
| `PHOTOVIEW_SERVE_UI` | Serve built UI from API | 1 |
| `PHOTOVIEW_UI_PATH` | Path to built UI assets | /app/ui |
| `PHOTOVIEW_MEDIA_CACHE` | Thumbnail cache directory | /home/photoview/media-cache |
| `PHOTOVIEW_FACE_RECOGNITION_MODELS_PATH` | Face detection models | /app/data/models |

### Hardware Acceleration

Set `PHOTOVIEW_VIDEO_HARDWARE_ACCELERATION` for video transcoding:
- `qsv`: Intel Quick Sync (requires `/dev/dri` mounted)
- `vaapi`: VAAPI (requires `/dev/dri` mounted)
- `nvenc`: NVIDIA NVENC (requires nvidia-docker runtime)

## File Permissions in Production

The container runs as UID/GID `999:999` (user `photoview`). Media volumes must be readable by this user or by `others`:

```bash
# Option 1: Make files world-readable
chmod -R 755 /path/to/photos

# Option 2: Create group with GID 999
groupadd -g 999 photoview-group
chown -R :photoview-group /path/to/photos
chmod -R 750 /path/to/photos
```

## Key Files to Understand

- `api/server.go`: Main server initialization, database setup, GraphQL handler registration
- `api/graphql/models/`: GraphQL type definitions
- `api/graphql/resolvers/*.graphql`: Query/mutation definitions by domain
- `api/graphql/resolvers/*.go`: Corresponding resolver implementations
- `api/graphql/directive.go`: Implementation of `@isAdmin` and `@isAuthorized` directives
- `api/scanner/scanner_queue/queue.go`: Task queue implementation with worker pool
- `api/scanner/periodic_scanner/`: Background periodic scanning logic
- `ui/src/apolloClient.ts`: Apollo Client setup with WebSocket subscriptions
- `ui/src/components/photoGallery/ProtectedMedia.tsx`: Image loading with blurhash and lazy loading
- `Dockerfile`: Multi-stage build (UI → API → release with compression)

## GraphQL Directives

The API uses two custom directives for access control (defined in `api/graphql/resolvers/root.graphql`):

- `@isAuthorized`: Requires any authenticated user (token-based auth via cookie)
- `@isAdmin`: Requires user with `admin: true` flag

These directives are implemented in `api/graphql/directive.go` and wrap resolver execution.

## Known Limitations

- **SQLite**: Only 1 concurrent writer; scanning will show "database is locked" errors (harmless, retries work)
- **Face Detection**: Requires significant CPU; can be disabled per user in settings
- **Large Libraries**: Initial scan of 10k+ photos can take hours; thumbnails are cached persistently

## Important Architecture Patterns

### Album-User Relationship

Albums and users have a **many-to-many** relationship via the `user_albums` junction table. This allows multiple users to share access to the same albums (e.g., a family photo directory structure).

**Key implication**: When multiple users share a directory tree, albums are created once with `parent_album_id` references. Users whose root album is a sub-album of another user's root album will have NO albums with `parent_album_id IS NULL`.

The `MyAlbums()` function in `api/graphql/models/actions/album_actions.go` handles this via `getTopLevelAlbumIDs()`, which finds albums that are "top-level" for a specific user (either root albums or direct children of albums NOT owned by the user).

### Scanner Owner Propagation

When `FindAlbumsForUser()` runs in `api/scanner/scanner_user.go`:
1. New albums get their parent's owners appended (`tx.Model(&album).Association("Owners").Append(parentOwners)`)
2. Existing albums get the current user added as an owner if not already present
3. This creates a shared ownership model where users can have overlapping album access

### Symlink Scanning Behavior

**Important**: Photoview does NOT share scanning or media records between users, even when accessing the same files via symlinks.

When multiple users have root paths that point to the same physical files via symlinks:
1. **Each user gets separate media records** in the database
2. **Scanning is duplicated** - each file is scanned once per user
3. **Thumbnails are NOT shared** - each user gets their own cached thumbnails
4. **EXIF, face detection, blurhash** are all processed independently per user

**Example scenario**:
- User A has root path `/photos` containing 10,000 files
- User B has root path `/home/userb/photos` which is a symlink to `/photos`
- Result: The database will have 20,000 media records (10,000 per user), and scanning will process all files twice

**Why this happens**: The scanner uses file paths as the primary key, not inodes or filesystem hashes. When User B's scan encounters `/home/userb/photos/photo.jpg`, it creates a new media record even if User A already has `/photos/photo.jpg` scanned.

**Implication**: Using symlinks to share a photo library between users effectively doubles (or more) the storage requirements for thumbnails and duplicates all scanning work.

### Cross-Platform Docker Builds

Building multi-platform Docker images locally has limitations:
- **Cross-compilation with buildx** fails at stages that run architecture-specific binaries (e.g., `api` stage binaries run during `release` stage)
- **GitHub Actions** uses QEMU emulation and has more resources (~7GB RAM vs ~1GB on low-end devices)
- For ARM64 builds on resource-constrained devices, use GitHub Actions or native ARM64 build environments

The workflow in `.github/workflows/build.yml` demonstrates proper multi-platform builds using `docker/setup-buildx-action` and `docker/setup-qemu-action`.

### Recent Fixes

**Scanner Queue Notification Race Condition** (2026-03)
- Fixed race condition where `idle_chan` buffer size of 1 caused notifications to be lost when multiple jobs completed simultaneously
- Increased buffer to 100 and made `notify()` non-blocking to prevent deadlocks
- Added re-notification logic when jobs remain in queue after processing

**Non-Fatal Scanner Error Handling** (2026-03)
- `AddUserToQueue()` was aborting if `FindAlbumsForUser()` returned ANY errors (e.g., permission denied on single directory)
- This prevented ALL albums from being queued for media scanning when one directory had permission issues
- Changed behavior to log non-fatal errors but continue queuing discovered albums
- Example: A permission error on `/photos/Моё Др 2023` was blocking scanning of all other albums

**Album Visibility for Users Without Root Albums** (2026-03)
- Users whose albums are all sub-albums of another user's root album couldn't see albums in UI
- Fixed by adding `getTopLevelAlbumIDs()` function to properly identify top-level albums per user
- Affects scenarios where: User A scans `/photos` first, then User B is added with `/photos/userB` - User B's albums all have `parent_album_id` pointing to User A's album tree

## Security Model

**Read-only Media Access**: The API does NOT provide mutations to delete media or albums from the filesystem. Even if an attacker gains access, they cannot delete photos.

**User Capabilities** (regular user):
- View own media, EXIF metadata (including GPS), face groups
- Mark media as favorites
- Change album covers
- Edit face group labels
- Create/delete share tokens (public links with optional password/expiry)
- Change language preferences

**Admin Capabilities** (in addition to user):
- Create/update/delete users
- Change user admin status
- Manage user root paths
- Trigger scans
- Configure scanner settings

**Media Protection**: All media URLs require authentication tokens (cookie-based). Public access is only through share tokens.

## OpenWrt Deployment Notes

This repository contains deployment configurations for OpenWrt (NanoPi R2S Plus):

### Deployment Files

- `docker-compose.yml`: Combined photoview + nginx setup with SQLite backend
- `.env`: Environment variables for production deployment
- `nginx/nginx.conf`: Main nginx config with performance optimizations
- `nginx/conf.d/photoview.conf`: SSL reverse proxy for funspace.duckdns.org

### OpenWrt-Specific Considerations

1. **No /etc/localtime**: Timezone volume mounts are removed (OpenWrt uses different structure)
2. **UID 999**: Container runs as photoview user; media files must be readable (chmod 755)
3. **nftables**: Use raw_prerouting chain to block ports before Docker DNAT (standard filter rules don't work):
   ```bash
   nft add rule inet fw4 raw_prerouting iifname eth0 tcp dport 8000 drop
   ```
4. **SSL Certificates**: Mounted from `/etc/acme/funspace.duckdns.org_ecc/` (acme.sh managed)

### Custom Build Workflow

The `.github/workflows/build-patched.yml` workflow builds a custom ARM64 image with local patches:
- Triggered on push to any branch or manual dispatch
- Produces `photoview-patched:latest` image
- Artifact downloadable as `docker-image-arm64.zip`
- Load and deploy on OpenWrt:
   ```bash
   docker load < /opt/photoview/docker-image-arm64.zip
   docker tag photoview-patched:latest photoview-patched:latest
   ```

### Monitoring Scanning Progress

```bash
# Check media count in database
sqlite3 /opt/photoview/database/photoview.db "SELECT COUNT(*) FROM media;"

# Check video count
sqlite3 /opt/photoview/database/photoview.db "SELECT COUNT(*) FROM media WHERE type = 'video';"

# Count total files on disk
find /home/syncthing/sync -type f \( -iname "*.jpg" -o -iname "*.jpeg" -o -iname "*.png" -o -iname "*.mp4" -o -iname "*.mov" \) | wc -l
```
