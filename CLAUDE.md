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
# Patch go-face for compilation (remove -lcblas, -march=native)
sed -i 's/-lcblas//g' $(go env GOMODCACHE)/github.com/\!kagami/go-face*/face.go
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

# UI tests (Vitest)
cd ui
npm test
npm run test:ci    # CI mode with coverage

# Linting
cd ui && npm run lint
cd ui && npm run format:check
```

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
- `api/scanner/scanner_queue/queue.go`: Task queue implementation with worker pool
- `api/scanner/periodic_scanner/`: Background periodic scanning logic
- `ui/src/apolloClient.ts`: Apollo Client setup with WebSocket subscriptions
- `Dockerfile`: Multi-stage build (UI → API → release with compression)

## Known Limitations

- **SQLite**: Only 1 concurrent writer; scanning will show "database is locked" errors (harmless, retries work)
- **Face Detection**: Requires significant CPU; can be disabled per user in settings
- **Large Libraries**: Initial scan of 10k+ photos can take hours; thumbnails are cached persistently

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

### Monitoring Scanning Progress

```bash
# Check media count in database
sqlite3 /opt/photoview/database/photoview.db "SELECT COUNT(*) FROM media;"

# Check video count
sqlite3 /opt/photoview/database/photoview.db "SELECT COUNT(*) FROM media WHERE type = 'video';"

# Count total files on disk
find /home/syncthing/sync -type f \( -iname "*.jpg" -o -iname "*.jpeg" -o -iname "*.png" -o -iname "*.mp4" -o -iname "*.mov" \) | wc -l
```
