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

# Run in dev mode
./dev.sh  # equivalent to: go run .

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
- `app/pkg/database/` - SQLite database management, migrations, and JSON import
- `app/pkg/subscription/` - Subscription CRUD operations (database-backed)
- `app/pkg/initstate/` - Initialization state management (database-backed)
- `app/pkg/sublink/` - Node parsing and subscription fetching
- `app/pkg/installer/` - sing-box and dashboard installation

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
- **Routing**: Vue Router with `/init` wizard and `/dashboard` main app
- **Styling**: Tailwind CSS v4
- **API Client**: Axios wrapper in `src/services/api.ts`
- **Components**: Reusable UI components in `src/components/`
- **Views**:
  - `views/InitWizard.vue` - Multi-step initialization
  - `views/Dashboard.vue` - Main dashboard with nested routes
  - `views/dashboard/` - Feature-specific views (Config, DNS, Inbounds, Outbounds, etc.)

## Key Development Patterns

### Adding a New Protocol Parser

1. Create `app/pkg/sublink/protocol/newprotocol.go`
2. Implement struct with `Parse() (*node.SubNode, error)` method
3. Register in `ppMap`: `"newprotocol://": func() node.SubNodeParser { return new(NewProtocol) }`
4. Add test file following `trojan_test.go` pattern

### Adding a New API Endpoint

1. Add handler method in `app/routes/v1_12_12/handler.go` or specific handler file
2. Register route in `app/routes/v1_12_12/routes.go`
3. Use Hertz's `c.JSON()` for responses, `c.Bind()` for request parsing
4. Follow existing error handling pattern: return 400 for bad requests, 500 for server errors
5. Update frontend API client in `frontend/src/services/api.ts`
6. Add TypeScript types in `frontend/src/types/api.ts`

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
- **Database Driver**: github.com/mattn/go-sqlite3 (SQLite3 driver)
- **sing-box**: Must be installed and accessible (in PATH or via binary_path config)
- **Node**: v22.21.1 (specified in `frontend/.nvmrc`)
- **Frontend**: Vue 3.5+, TypeScript 5.9+, Vite 7+, Tailwind CSS 4+

## API Versioning

Current version: v1.12.12 (corresponds to sing-box 1.12.12)
- All routes prefixed with `/api/1.12.12/`
- Version-specific handlers in `app/routes/v1_12_12/`
- For new sing-box versions, create new versioned route group

## Testing Notes

- Protocol parsers have test files (e.g., `trojan_test.go`)
- Use `go test -v` for verbose output
- Test node link formats found in test files serve as documentation
- Frontend has no test setup currently - add Vue Test Utils if needed

## Important File Paths

- `main.go` - Entry point, loads config and starts server
- `app/svr.go` - Application initialization, database setup
- `app/pkg/database/` - Database initialization, migrations, and JSON import
- `app/routes/v1_12_12/routes.go` - API route definitions
- `app/pkg/config/types.go` - sing-box config struct definitions
- `frontend/src/router/index.ts` - Frontend routing configuration
- `doc/API_v1.13.0.md` - Complete API documentation (Chinese)
- `DATABASE_MIGRATION.md` - Database migration guide
