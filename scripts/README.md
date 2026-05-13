# Build Scripts

This directory contains utility scripts for building and deploying sing-box-easy.

## build.sh

Docker build script for creating x86_64 (amd64) images on macOS.

### Usage

```bash
# Basic build
./scripts/build.sh

# Build with custom tag
./scripts/build.sh -t v1.0.0

# Build without using cache
./scripts/build.sh --no-cache

# Build and push to registry
./scripts/build.sh -t v1.0.0 -p
```

### Features

- **Cross-platform support**: Works on both Intel and Apple Silicon Macs
- **Automatic buildx setup**: Creates and configures Docker buildx builder
- **Platform targeting**: Builds for linux/amd64 (x86_64) architecture
- **Flexible options**: Support for custom tags, push to registry, cache control
- **Error handling**: Validates Docker availability and build status

### Options

| Option | Description |
|--------|-------------|
| `-t, --tag TAG` | Image tag (default: latest) |
| `-n, --name NAME` | Image name (default: sing-box-easy) |
| `-p, --push` | Push image to registry after build |
| `-l, --load` | Load image to local docker (default: true) |
| `--no-load` | Don't load image to local docker |
| `--no-cache` | Build without using cache |
| `-h, --help` | Display help message |

### Examples

```bash
# Development build
./scripts/build.sh -t dev

# Production build with push
./scripts/build.sh -t v1.2.3 -p

# Clean build without cache
./scripts/build.sh --no-cache

# Build but don't load locally (for CI/CD)
./scripts/build.sh -t ci-build --no-load -p
```

### Requirements

- Docker Desktop for Mac
- Docker buildx (included in Docker Desktop)

### Notes

- The script automatically creates a buildx builder named `multiarch-builder` if it doesn't exist
- Built images are loaded to local Docker by default (can be disabled with `--no-load`)
- The script validates that Docker is running before attempting to build

---

## deploy-ssh.sh

Direct-SSH deployment (no Docker / Podman). Cross-compiles the Go binary
for Linux, builds the Vue frontend, ships both to the remote host over
`scp`, and (optionally) restarts the service in the background.

Designed for hosts that already have `sing-box` installed and only need
the `sing-box-easy` admin layered on top.

### Usage

```bash
# Default: deploy to root@192.168.9.253:/etc/sing-box and restart
bash scripts/deploy-ssh.sh

# Deploy to a different host
REMOTE_HOST=root@10.0.0.5 bash scripts/deploy-ssh.sh

# Upload only — don't restart the service
START_AFTER_DEPLOY=false bash scripts/deploy-ssh.sh

# Backend-only redeploy (skip vite build, reuse remote dist/)
SKIP_FRONTEND=true bash scripts/deploy-ssh.sh

# Use npm instead of bun for the frontend build
FRONTEND_TOOL=npm bash scripts/deploy-ssh.sh
```

### Configuration (env vars, all optional)

| Var | Default | Description |
|-----|---------|-------------|
| `REMOTE_HOST` | `root@192.168.9.253` | SSH target (user@host) |
| `REMOTE_DIR` | `/etc/sing-box` | Install dir on the remote |
| `REMOTE_PORT` | `8080` | HTTP port baked into the generated `app.yml` |
| `SING_BOX_CONFIG` | `${REMOTE_DIR}/config.json` | Path to sing-box's config (the file the admin manages) |
| `SING_BOX_BINARY` | `sing-box` | Path or name of the sing-box CLI on the remote |
| `SING_BOX_DB` | `${REMOTE_DIR}/sing-box-easy.db` | SQLite path |
| `GOOS_TARGET` | `linux` | Go cross-compile target OS |
| `GOARCH_TARGET` | `amd64` | Go cross-compile target arch |
| `BUILD_FRONTEND` | `true` | Run vite build before uploading |
| `SKIP_FRONTEND` | `false` | Convenience inverse of `BUILD_FRONTEND` |
| `SKIP_BACKEND` | `false` | Skip the Go build (frontend-only redeploy) |
| `FRONTEND_TOOL` | `bun` | `bun` or `npm` — falls back to npm if bun is missing |
| `BACKUP_CONFIG` | `true` | Copy `config.json` → `config.json.predeploy-<ts>` before deploy |
| `START_AFTER_DEPLOY` | `true` | Start the service in background and smoke-test `/init/status` |
| `INSTALL_APP_YML_IF_MISSING` | `true` | Drop a generated `app.yml` only if the remote doesn't have one |

### What it does

1. Probes SSH and remote architecture; warns if `GOARCH_TARGET` mismatches.
2. Cross-compiles `./main.go` with `CGO_ENABLED=0`, stripped, to `/tmp/sing-box-easy.<os>-<arch>`.
3. Builds the frontend with `bun` (or `npm`) — uses `vite build` directly to bypass the pre-existing `vue-tsc` strict-type error in `volt/Chips.vue`.
4. On the remote: backs up `config.json`, stops any running `sing-box-easy`, swaps the binary atomically, refreshes `dist/`, drops `app.yml` if missing, and (optionally) restarts in the background.
5. Smoke-tests `GET /api/1.12.12/init/status` and fails the deploy if the response isn't `{"code":0,...}`.

### Notes

- The script never overwrites an existing remote `app.yml` — your settings are preserved across redeploys.
- Backups accumulate as `config.json.predeploy-<timestamp>` — clean them up manually when you're confident.
- Requires key-based SSH (BatchMode); password prompts are not supported.
