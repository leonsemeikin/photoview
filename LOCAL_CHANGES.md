# Local Changes

This file documents custom modifications made to this fork of Photoview that differ from the upstream repository.

## Recent Fixes (2026-03)

### Scanner Queue Notification Race Condition
- Fixed race condition where `idle_chan` buffer size of 1 caused notifications to be lost when multiple jobs completed simultaneously
- Increased buffer to 100 and made `notify()` non-blocking to prevent deadlocks
- Added re-notification logic when jobs remain in queue after processing
- **Files modified**: `api/scanner/scanner_queue/queue.go`

### Non-Fatal Scanner Error Handling
- `AddUserToQueue()` was aborting if `FindAlbumsForUser()` returned ANY errors (e.g., permission denied on single directory)
- This prevented ALL albums from being queued for media scanning when one directory had permission issues
- Changed behavior to log non-fatal errors but continue queuing discovered albums
- Example: A permission error on `/photos/Моё Др 2023` was blocking scanning of all other albums
- **Files modified**: `api/scanner/scanner_user.go`

### Album Visibility for Users Without Root Albums
- Users whose albums are all sub-albums of another user's root album couldn't see albums in UI
- Fixed by adding `getTopLevelAlbumIDs()` function to properly identify top-level albums per user
- Affects scenarios where: User A scans `/photos` first, then User B is added with `/photos/userB` - User B's albums all have `parent_album_id` pointing to User A's album tree
- **Files modified**: `api/graphql/models/actions/album_actions.go`, `api/graphql/resolvers/album.go`

## Known Behavior

### Symlink Scanning
Photoview does NOT share scanning or media records between users, even when accessing the same files via symlinks:
- Each user gets separate media records in the database
- Scanning is duplicated - each file is scanned once per user
- Thumbnails are NOT shared - each user gets their own cached thumbnails
- EXIF, face detection, blurhash are all processed independently per user

**Example**: User A has root path `/photos` (10,000 files), User B has root path `/home/userb/photos` (symlink to `/photos`). Result: 20,000 media records, all files scanned twice.

## OpenWrt Deployment

This repository includes deployment configurations for OpenWrt (NanoPi R2S Plus):
- `docker-compose.yml`: Combined photoview + nginx setup
- `.env`: Production environment variables
- `nginx/`: SSL reverse proxy configuration for funspace.duckdns.org
- `.github/workflows/build-patched.yml`: Custom ARM64 build workflow

See `CLAUDE.md` for detailed OpenWrt deployment notes.
