# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

sing-box-easy is a RESTful API service for managing sing-box configurations and controlling the sing-box proxy service. It provides:
- Full sing-box configuration management (Inbound, Outbound, DNS, Route, etc.)
- Service lifecycle control (start, stop, restart)
- Subscription management with auto-update
- Multi-protocol node parsing (shadowsocks, vmess, trojan)
- Vue 3 + TypeScript frontend with Tailwind CSS

## Build and Run Commands

### Backend (Go)
```bash
# Build
go build -o sing-box-easy ./main.go

# Run with default config (app.yml)
./sing-box-easy

# Run with custom config
./sing-box-easy -c /path/to/config.yml

# Run in dev mode (uses bin/app.yml — port 5100, DB in ./bin/, DEBUG=true)
./dev.sh  # equivalent to: DEBUG=true go run . -c bin/app.yml

# Run tests
go test ./...

# Run specific test
go test ./app/pkg/sublink/protocol -v -run TestTrojanParser
```

### Frontend (Vue 3)
```bash
cd frontend

# Install dependencies
npm install

# Dev server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

## Configuration

- **App config**: `app.yml` (copy from `app.yml.example`)
- **sing-box config**: Managed via API, default path `/etc/sing-box/config.json`
- **Database**: SQLite database at `/etc/sing-box/sing-box-easy.db` (stores subscriptions and init state)
- **Subscriptions**: Now stored in SQLite database (legacy JSON path kept for migration)
- Port can be overridden via `HTTP_PORT` environment variable

## Architecture

### Backend Structure

**Core Packages**:
- `app/pkg/appconfig/` - Application configuration loading (app.yml)
- `app/pkg/config/` - sing-box config management with validation and rollback
- `app/pkg/service/` - sing-box service lifecycle control (start/stop/restart)
- `app/pkg/process/` - Process discovery and signaling helpers (pgrep/SIGTERM/SIGHUP)
- `app/pkg/database/` - SQLite database management, migrations, and JSON import
- `app/pkg/subscription/` - Subscription CRUD + cron AutoUpdater (database-backed)
- `app/pkg/initstate/` - Initialization state management (database-backed)
- `app/pkg/sublink/` - Node parsing and subscription fetching
- `app/pkg/installer/` - sing-box and dashboard installation (task-based)
- `app/pkg/logger/` - Zap logger setup + Hertz logger adapter

**Protocol Parsers** (`app/pkg/sublink/protocol/`):
- Factory pattern for creating protocol-specific parsers
- Currently supports: Shadowsocks (ss://), VMess (vmess://), Trojan (trojan://)
- Each parser implements `node.SubNodeParser` interface
- New protocols: Add parser file + register in `ppMap` in `protocol.go`

**HTTP Layer**:
- Framework: CloudWeGo Hertz (high-performance HTTP framework)
- Routes: `app/routes/v1_12_12/` - All API handlers for v1.12.12
- API prefix: `/api/1.12.12/`
- Route registration: `routes.go` in v1_12_12 package

### Configuration Safety Mechanism

All config modifications follow a safe workflow (implemented in `config.Manager`):
1. Write to temporary file `config_new.json`
2. Validate using `sing-box check` command
3. On success:
   - Backup current config to `config.old.json`
   - Move validated config to `config.json`
4. On failure: Keep original config, remove temp file
5. Rollback available via `/api/1.12.12/config/rollback`

### Service Control Architecture

The `service.Controller` manages sing-box lifecycle:
- Uses `pgrep` to check process status
- Validates config before any start/restart operation
- Graceful shutdown via SIGTERM, force stop via SIGKILL
- Reload support via SIGHUP (falls back to restart if unsupported)

### Database Architecture

The application uses SQLite with XORM ORM for persistent storage:
- **ORM**: XORM (xorm.io/xorm) for struct-based models
- **Database initialization**: Automatic on startup via `database.Init()`
- **Schema migrations**: Automatic sync via `engine.Sync2()` (no manual SQL)
- **Models**: `database.InitState`, `database.Subscription` structs
- **Automatic migration**: JSON files automatically imported on first run
- **Tables**: `init_state`, `subscriptions`
- **Managers**: XORM-backed managers with interface compatibility

### Subscription Flow

1. User adds subscription with URL and settings
2. Subscription stored in SQLite database
3. `SubLink.fetchNodes()` downloads and base64-decodes content
4. Each line parsed through protocol factory
5. Parsed nodes can be added to outbounds as batch
6. Auto-update can be configured per subscription
7. All subscription data persisted in database with ACID guarantees

### Frontend Architecture

- **Framework**: Vue 3 + TypeScript + Vite
- **Routing**: Vue Router with `/init` wizard and `/dashboard` main app. A global `beforeEach` guard hits `/init/status` and redirects to `/init` until initialization is marked complete.
- **State**: Pinia stores in `src/stores/` (currently `dns`, `outbounds`, `route`)
- **Styling**: Tailwind CSS v4 + DaisyUI utility classes; PrimeVue + HeadlessUI for components; Heroicons for icons
- **API Client**: Axios wrapper in `src/services/api.ts` (`baseURL: /api/1.12.12`), with one service module per domain (`config.ts`, `dns.ts`, `outbound.ts`, …)
- **Editor**: Monaco (via `monaco-editor-vue3`) — kept in its own chunk by `vite.config.ts`
- **Dev proxy**: `vite.config.ts` proxies `/api/*` to `http://localhost:5100` — match this with `server.port` in `bin/app.yml` when changing dev ports
- **Components**: Reusable UI components in `src/components/`
- **Views**:
  - `views/InitWizard.vue` - Multi-step initialization (steps in `views/init-steps/`)
  - `views/Dashboard.vue` - Main dashboard with nested routes
  - `views/dashboard/` - Feature-specific views (Config, DNS, Inbounds, Outbounds, Route, Experimental, Subscriptions, Log, Overview)

## Key Development Patterns

### Adding a New Protocol Parser

1. Create `app/pkg/sublink/protocol/newprotocol.go`
2. Implement struct with `Parse() (*node.SubNode, error)` method
3. Register in `ppMap`: `"newprotocol://": func() node.SubNodeParser { return new(NewProtocol) }`
4. Add test file following `trojan_test.go` pattern

### Adding a New API Endpoint

1. Add handler method in the appropriate `app/routes/v1_12_12/<domain>_handler.go`
2. Register the route in `app/routes/v1_12_12/routes.go`
3. Parse the request with `c.Bind(&body)`; on parse error call `respErr(ctx, c, CodeBadRequest, msg)`
4. Use the response helpers from `handler.go` — `respOK(ctx, c, data)` for success and `respErr(ctx, c, code, msg)` for errors. **All responses leave with HTTP 200**; failure semantics live in the business `Code` enum inside the `BasicResponse[T] { code, data, msg }` envelope (see `Code` constants in `handler.go`).
5. Do **not** call raw `c.JSON()` for new endpoints — it bypasses the envelope and the sing-box-aware JSON marshaller.
6. Update the matching frontend service in `frontend/src/services/<domain>.ts` (the shared axios client is `frontend/src/services/api.ts`).
7. Add or extend TypeScript types in `frontend/src/types/`.

### Config Modification Pattern

Always use `config.Manager.UpdateConfig()` with update function:
```go
err := configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
    // Modify cfg fields here
    cfg.Outbounds = append(cfg.Outbounds, newOutbound)
    return nil
})
```
This ensures validation and backup happen automatically.

## Dependencies

- **Go**: 1.25.3+
- **HTTP Framework**: CloudWeGo Hertz v0.10.3
- **HTTP Client**: imroc/req/v3 for subscription fetching
- **ORM**: xorm.io/xorm v1.3.11 (XORM ORM framework)
- **Database Driver**: modernc.org/sqlite v1.40.1 (pure-Go SQLite, no CGO required)
- **Scheduler**: github.com/robfig/cron/v3 v3.0.1 (subscription auto-updater)
- **Logging**: go.uber.org/zap v1.27.0
- **sing-box**: Must be installed and accessible (in PATH or via binary_path config)
- **Node**: v22.21.1 (specified in `frontend/.nvmrc`)
- **Frontend**: Vue 3.5+, TypeScript 5.9+, Vite 7+, Tailwind CSS 4+

## API Versioning

Current version: v1.12.12 (corresponds to sing-box 1.12.12)
- All routes prefixed with `/api/1.12.12/`
- Version-specific handlers in `app/routes/v1_12_12/`
- For new sing-box versions, create new versioned route group

API surface groups (registered in `routes.go`):
- `/config` — get/update/validate, backup, rollback
- `/nodes/parse` — parse subscription link batch
- `/outbounds`, `/outbounds/batch`, `/outbounds/:tag/members`, `/outbounds/groups`
- `/dns`, `/dns/servers`, `/dns/hosts`, `/dns/rules`
- `/inbounds`
- `/route/rules`, `/route/rule-sets`, `/route/final`
- `/log`
- `/experimental/{clash-api,cache-file,v2ray-api}`
- `/service/{status,start,stop,restart}`
- `/subscriptions` + `/subscriptions/:id/update`
- `/scheduler/{status,start,stop,trigger,jobs}` — cron auto-updater control
- `/install`, `/install/task/:task_id`, `/install/status`, `/update`
- `/dashboard/{download,upload}`, `/dashboard/task/:task_id`, `/dashboard/status`
- `/init/{status,complete,reset}`
- `/templates/rule-sets`

## Testing Notes

- Protocol parsers have test files (e.g., `trojan_test.go`); only Shadowsocks/VMess/Trojan parsing is covered today.
- Use `go test -v` for verbose output.
- Test node link formats inside the parser tests serve as documentation for accepted URI shapes.
- Frontend has no test setup currently — add Vitest + Vue Test Utils if introducing tests.

## Response Envelope Reference

Defined in `app/routes/v1_12_12/handler.go`. All new v1.12.12 endpoints must use these helpers; HTTP status is always 200 and clients branch on `code`.

```go
type Code uint8
const (
    CodeSuccess         Code = iota // 0
    CodeBadRequest                  // 1
    CodeNotFound                    // 2
    CodeInternalError               // 3
    CodeValidationError             // 4
    CodeConflict                    // 5
    CodeUnauthorized                // 6
    CodeForbidden                   // 7
    CodeServiceError                // 8
    CodeConfigError                 // 9
    CodeOperationFailed             // 10
)

type BasicResponse[T any] struct {
    Code Code   `json:"code"`
    Data T      `json:"data"`
    Msg  string `json:"msg"`
}

// helpers
respOK(ctx, c, data)                 // code=0, msg="success"
respErr(ctx, c, CodeBadRequest, msg) // data=nil
```

Note: `app/routes/handler.go` (the older non-versioned `ListNodes`) still uses raw `c.JSON()` with HTTP 400/500. Treat that as legacy — new code goes under `v1_12_12/` with the envelope.

## Important File Paths

- `main.go` - Entry point, loads config and starts server
- `app/svr.go` - Application initialization, database setup
- `app/pkg/database/` - Database initialization, migrations, and JSON import
- `app/routes/v1_12_12/routes.go` - API route definitions
- `app/pkg/config/types.go` - sing-box config struct definitions
- `frontend/src/router/index.ts` - Frontend routing configuration
- `frontend/src/services/api.ts` - Shared axios client (`baseURL: /api/1.12.12`)
- `frontend/vite.config.ts` - Dev server proxies `/api` → `http://localhost:5100`
- `bin/app.yml` - Local dev config used by `./dev.sh` (port `5100`, db in `./bin/`)
- `doc/API_v1.13.0.md` - Complete API documentation (Chinese)
- `doc/DATABASE_MIGRATION.md` - Database migration guide
