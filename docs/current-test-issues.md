# Current Test Issues

## Summary
Initial test run in CI showed failures due to missing database environment variables. The tests are configured to require MySQL by default, but only provide MySQL configuration in CI, not for local runs.

## Issues Found

### 1. Missing Environment Variables
- Tests fail with: `Environment variable PHOTOVIEW_MYSQL_URL missing, exiting`
- Default database driver is MySQL (fallback in `database_drivers.go`)
- Tests need explicit database configuration or SQLite fallback

### 2. Test Configuration
- Tests use `-database -filesystem -p 1` flags as per CI
- Local testing requires either:
  - MySQL database setup, or
  - SQLite driver configuration

### 3. Test Files Status
- 29 Go test files exist
- 21 TypeScript test files exist
- Many tests require database integration

## Fixes Applied

### 1. Add testing.env for SQLite
Created `api/testing.env` with SQLite configuration for local testing:
```
PHOTOVIEW_DATABASE_DRIVER=sqlite
PHOTOVIEW_SQLITE_PATH=/tmp/photoview_test.db
```

### 2. Update Test Scripts
- Modified test execution to use SQLite by default for local testing
- CI still uses all three databases (SQLite, MySQL, PostgreSQL) as configured

## Verification
- Tests run successfully with SQLite locally
- CI tests run with all configured databases