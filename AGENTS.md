# AGENTS.md

This file provides guidance to coding agents working in this repository.

## Build & Test

```bash
go build -o weoa-cli .                                      # Build binary
go test ./... -count=1                                      # Run all tests
go test ./internal/publish -v -run TestService_List_Success  # Run a single test
go test ./internal/storage -v -run TestMigrate              # Run migration tests
go vet ./...                                                # Static analysis
```

Tests do not require real WeChat credentials or external network access. Some API tests use `httptest.Server`, so restricted sandboxes must permit a local loopback listener. Storage tests use `t.TempDir()` and never touch the user's real database.

## Architecture

`weoa-cli` is a Go CLI tool that manages a WeChat Official Account (公众号) backend from the terminal. It uses the **internal backend API** of `mp.weixin.qq.com` (not the official developer API), relying on browser-automated login to obtain a session (cookies + token).

### Layer Model

```
cmd/           → Cobra commands (thin, wiring only)
internal/
  client/      → HTTP client with cookie jar + token injection; also holds settings API parsing
  publish/     → PublishService: calls appmsgpublish API, parses the double-JSON-encoded response
  auth/        → Playwright login flow, browser/terminal QR handling, and session persistence
  storage/     → SQLite (modernc.org/sqlite, pure Go, no CGO)
  config/      → Config dir paths: ~/.config/weoa-cli/
```

### Shared Data Model

`internal/publish/model.go` defines the `Article` struct — this is the **shared data model** used by both the `publish` and `storage` packages. The `storage` package imports `publish` for this type (storage depends on publish, not the reverse). The `Article` struct has JSON tags matching the WeChat API field names.

### Key Design Details

**Session lifecycle**: `auth login` launches Chromium via Playwright (auto-installs on first run), navigates to `mp.weixin.qq.com`, waits for the user to scan the QR code, extracts `token` from the redirect URL, and persists cookies + token + UA to `~/.config/weoa-cli/session.json`. `auth login --qr` decodes the browser page's QR image and renders it in the terminal, but still uses Playwright for the login flow. Commands that call the backend load the saved session and inject `token` and `lang=zh_CN` into every request.

**Token injection**: `client.Client.R()` is the central request builder — it pre-sets `token` and `lang=zh_CN` query params on every request. Use `GetWithParams` for additional params. The `Client` wraps a `resty` client with a cookie jar seeded from the session.

**Double JSON encoding**: The `GET /cgi-bin/appmsgpublish?sub=list&f=json` endpoint returns a response where `publish_page` is a **JSON-encoded string** (not a nested object), and each record's `publish_info` within that is also a JSON-encoded string. The parser in `internal/publish/model.go` (`parseResponse`) handles both decoding layers. Tests in `api_test.go` and `model_test.go` use `json.Marshal` to build the nested structure rather than hand-escaping.

**Settings API**: `internal/client/settings.go` calls `/cgi-bin/settingpage?t=setting/index&action=index&f=json` to fetch account profile info (nickname, fans count, categories, etc.). It uses the same `client.Client` (and therefore the same session/token injection) but has its own response types.

**Article identity**: An article is identified by the composite `(appmsgid, URL)` key. Do not use `appmsgid` alone for deduplication: multi-article WeChat publishes can contain distinct articles that share an `appmsgid` but have different content URLs. Keep database lookups, remote-list filtering, sync state, fakes, and tests aligned with `articleIdentity` in `cmd/publish/identity.go`. SQLite enforces `UNIQUE(appmsgid, url)` and `InsertArticle` uses `INSERT OR IGNORE` for exact duplicates.

**Incremental sync**: `publish sync` paginates through `appmsgpublish` and processes an entire page before stopping. In incremental mode, an exact `(appmsgid, URL)` pair that existed before the run requests pagination stop. The `seenThisRun` map prevents overlapping or repeated API data inserted during the current run from causing an early stop. Deleted records are checked for stop detection but are not inserted.

**Schema migration and backfill**: `internal/storage/migrate` uses `PRAGMA user_version`. Version 1 transactionally rebuilds the legacy `UNIQUE(appmsgid)` table as `UNIQUE(appmsgid, url)` while preserving rows and IDs. A migrated database receives `sync_state.needs_full_sync = 1`; the next successful `publish sync` performs a one-time full traversal to recover articles previously hidden by the legacy key, then clears the marker. Fresh databases start with the marker cleared. Future migrations must preserve both the article data and this retry-on-failure behavior.

**Config directory**: `~/.config/weoa-cli/` contains `session.json` (login session) and `cache.db` (SQLite article index). The `internal/config` package provides paths; tests use `t.TempDir()` to isolate. The config package exports `EnvPrefix = "WEOA"` (currently reserved for future use).

### Testing Patterns

**Interface-based testing**: `PublishService` depends on the `PublishClient` interface (only method: `GetWithParams`), allowing `httptest.Server`-based tests without real WeChat credentials. The test helper `mockClient` in `internal/publish/mock_test.go` implements this interface with a configurable `baseURL`.

**Command-level interfaces**: Commands that need storage access define their own **narrow interfaces** locally. For example, `cmd/publish/sync.go` defines:
```go
type syncStore interface {
    HasArticle(appMsgID int64, url string) (bool, error)
    InsertArticle(a publish.Article) (bool, error)
}
```
This allows `sync_test.go` to use a `fakeSyncStore` with configurable behavior (check errors, insert errors, pre-existing composite identities) without depending on the full `storage.Store` or a real SQLite database.

**Interface satisfaction checks**: Use compile-time assertions: `var _ PublishClient = (*mockClient)(nil)`

**Storage testing**: `internal/storage/sqlite_test.go` has a `testStore(t)` helper that creates a SQLite database in `t.TempDir()`. The `openTestDB` helper accepts a direct path and calls the package-level `migrate` function, bypassing `config.DBFile()`. Legacy migration tests create the old schema directly with `database/sql`, then call `migrate`; cover row preservation, same-`appmsgid`/different-URL insertion, and the persistent full-sync marker whenever changing the schema.

**API test helpers**: `internal/publish/api_test.go` provides `buildAPIResponse(records, totalCount)` and `makeRecord(msgID, appmsgID, title, contentURL)` — these build properly JSON-encoded API responses matching the real WeChat structure. Use `json.Marshal` to build test data, never hand-escape JSON strings.

### Adding a New Command

1. Create the command file under `cmd/<group>/` or `cmd/`
2. Wire it into `cmd/root.go` in the appropriate parent command
3. Add a new Service in `internal/` if it calls a new API endpoint
4. The Service should accept an interface for testability (like `PublishClient`)
5. If the command needs storage, define a narrow interface locally (like `syncStore`)
6. If the command has testable logic, use fakes implementing that interface in tests

### Dependencies

- `github.com/spf13/cobra` — CLI framework
- `github.com/go-resty/resty/v2` — HTTP client
- `github.com/mxschmitt/playwright-go` — browser automation (login only, auto-installs chromium on first run)
- `modernc.org/sqlite` — pure-Go SQLite (chosen to avoid CGO for cross-compilation)
- `github.com/makiuchi-d/gozxing` — QR image decoding for terminal login
- `github.com/mdp/qrterminal/v3` and `rsc.io/qr` — terminal QR rendering and QR data handling
