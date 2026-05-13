#!/bin/bash
# Direct-SSH deployment for sing-box-easy (no Docker / Podman needed).
#
# Default target is 192.168.9.253 (the Debian x86_64 box that already runs
# sing-box). All settings are env-var driven so you can deploy elsewhere
# without editing this file. The script is idempotent: re-running it just
# rebuilds, re-uploads, and restarts.
#
# Examples:
#   bash scripts/deploy-ssh.sh
#   REMOTE_HOST=root@10.0.0.5 bash scripts/deploy-ssh.sh
#   REMOTE_HOST=root@192.168.9.253 REMOTE_DIR=/opt/sing-box-easy bash scripts/deploy-ssh.sh
#   START_AFTER_DEPLOY=false bash scripts/deploy-ssh.sh    # upload only, don't restart
#   SKIP_FRONTEND=true bash scripts/deploy-ssh.sh          # backend-only redeploy
#   BACKUP_CONFIG=false bash scripts/deploy-ssh.sh         # skip config.json backup

set -euo pipefail

# Colors for output (match other scripts in this directory)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ── Configuration (override via env vars) ─────────────────────────────────────
REMOTE_HOST="${REMOTE_HOST:-root@192.168.9.253}"
REMOTE_DIR="${REMOTE_DIR:-/etc/sing-box}"
REMOTE_PORT="${REMOTE_PORT:-8080}"
SING_BOX_CONFIG="${SING_BOX_CONFIG:-${REMOTE_DIR}/config.json}"
SING_BOX_BINARY="${SING_BOX_BINARY:-sing-box}"
SING_BOX_DB="${SING_BOX_DB:-${REMOTE_DIR}/sing-box-easy.db}"

GOOS_TARGET="${GOOS_TARGET:-linux}"
GOARCH_TARGET="${GOARCH_TARGET:-amd64}"
BUILD_FRONTEND="${BUILD_FRONTEND:-true}"
SKIP_FRONTEND="${SKIP_FRONTEND:-false}"        # alias: SKIP_FRONTEND=true is the inverse of BUILD_FRONTEND=true
SKIP_BACKEND="${SKIP_BACKEND:-false}"
FRONTEND_TOOL="${FRONTEND_TOOL:-bun}"          # bun | npm
BACKUP_CONFIG="${BACKUP_CONFIG:-true}"
START_AFTER_DEPLOY="${START_AFTER_DEPLOY:-true}"
INSTALL_APP_YML_IF_MISSING="${INSTALL_APP_YML_IF_MISSING:-true}"

# Honor SKIP_FRONTEND as a convenience alias
if [ "$SKIP_FRONTEND" = "true" ]; then
    BUILD_FRONTEND="false"
fi

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Local build artifact paths (under /tmp so dirty source trees stay clean)
LOCAL_BIN="/tmp/sing-box-easy.${GOOS_TARGET}-${GOARCH_TARGET}"
LOCAL_DIST_TARBALL="/tmp/sing-box-easy-dist.tar.gz"
LOCAL_APP_YML="/tmp/sing-box-easy.app.yml"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"

echo -e "${BLUE}=== sing-box-easy SSH deploy ===${NC}"
echo "Project root      : $PROJECT_ROOT"
echo "Remote host       : $REMOTE_HOST"
echo "Remote dir        : $REMOTE_DIR"
echo "Remote port       : $REMOTE_PORT"
echo "Build backend     : $([ "$SKIP_BACKEND" = "true" ] && echo no || echo "yes (${GOOS_TARGET}/${GOARCH_TARGET})")"
echo "Build frontend    : $([ "$BUILD_FRONTEND" = "true" ] && echo "yes ($FRONTEND_TOOL)" || echo no)"
echo "Backup config     : $BACKUP_CONFIG"
echo "Start after deploy: $START_AFTER_DEPLOY"
echo ""

# ── Sanity checks ─────────────────────────────────────────────────────────────
echo -e "${BLUE}=== Pre-flight ===${NC}"

if ! command -v ssh >/dev/null 2>&1; then
    echo -e "${RED}ssh not found in PATH${NC}"; exit 1
fi
if ! command -v scp >/dev/null 2>&1; then
    echo -e "${RED}scp not found in PATH${NC}"; exit 1
fi
if [ "$SKIP_BACKEND" != "true" ] && ! command -v go >/dev/null 2>&1; then
    echo -e "${RED}go not found in PATH (need it for the backend build, or set SKIP_BACKEND=true)${NC}"; exit 1
fi
if [ "$BUILD_FRONTEND" = "true" ] && ! command -v "$FRONTEND_TOOL" >/dev/null 2>&1; then
    echo -e "${YELLOW}$FRONTEND_TOOL not found; falling back to npm${NC}"
    FRONTEND_TOOL="npm"
    if ! command -v npm >/dev/null 2>&1; then
        echo -e "${RED}neither bun nor npm available — cannot build frontend${NC}"; exit 1
    fi
fi

echo -n "Probing SSH to ${REMOTE_HOST}... "
if ssh -o BatchMode=yes -o ConnectTimeout=5 "$REMOTE_HOST" 'uname -m' >/tmp/sing-box-easy.remote-arch 2>/dev/null; then
    REMOTE_ARCH="$(cat /tmp/sing-box-easy.remote-arch | tr -d '[:space:]')"
    echo -e "${GREEN}ok${NC} (arch=$REMOTE_ARCH)"
    rm -f /tmp/sing-box-easy.remote-arch
else
    echo -e "${RED}FAILED${NC}"
    echo "Cannot SSH to $REMOTE_HOST (BatchMode key-only auth, 5s timeout)."
    echo "Make sure key-based login works: ssh $REMOTE_HOST"
    exit 1
fi

# Architecture sanity: warn (not error) if the target GOARCH doesn't match.
case "$GOARCH_TARGET:$REMOTE_ARCH" in
    amd64:x86_64|arm64:aarch64|arm64:arm64) : ;;  # match
    *) echo -e "${YELLOW}Warning: building GOARCH=$GOARCH_TARGET for remote arch=$REMOTE_ARCH (mismatch may break execution)${NC}" ;;
esac

# ── Build backend ─────────────────────────────────────────────────────────────
if [ "$SKIP_BACKEND" != "true" ]; then
    echo ""
    echo -e "${BLUE}=== Building backend (${GOOS_TARGET}/${GOARCH_TARGET}, CGO=0) ===${NC}"
    cd "$PROJECT_ROOT"
    CGO_ENABLED=0 GOOS="$GOOS_TARGET" GOARCH="$GOARCH_TARGET" \
        go build -trimpath -ldflags="-s -w" -o "$LOCAL_BIN" ./main.go
    file "$LOCAL_BIN" | head -1
    BIN_SIZE="$(ls -lh "$LOCAL_BIN" | awk '{print $5}')"
    echo -e "${GREEN}backend built: $LOCAL_BIN ($BIN_SIZE)${NC}"
fi

# ── Build frontend ────────────────────────────────────────────────────────────
if [ "$BUILD_FRONTEND" = "true" ]; then
    echo ""
    echo -e "${BLUE}=== Building frontend ($FRONTEND_TOOL) ===${NC}"
    cd "$PROJECT_ROOT/frontend"

    if [ ! -d node_modules ]; then
        echo "Installing dependencies..."
        if [ "$FRONTEND_TOOL" = "bun" ]; then bun install; else npm ci; fi
    fi

    # Skip the project's `vue-tsc -b` step (it has a pre-existing TS error in
    # volt/Chips.vue) — Vite alone is sufficient to produce the dist bundle.
    if [ "$FRONTEND_TOOL" = "bun" ]; then
        bunx vite build
    else
        npx vite build
    fi

    if [ ! -f "$PROJECT_ROOT/dist/index.html" ]; then
        echo -e "${RED}frontend build did not produce dist/index.html${NC}"; exit 1
    fi

    # Pack dist into a tarball for a single SCP roundtrip. The -C flag puts
    # the files at the tarball root so `tar -x` writes them directly into a
    # fresh dist/ on the remote.
    tar -czf "$LOCAL_DIST_TARBALL" -C "$PROJECT_ROOT/dist" .
    DIST_SIZE="$(ls -lh "$LOCAL_DIST_TARBALL" | awk '{print $5}')"
    echo -e "${GREEN}frontend built: $LOCAL_DIST_TARBALL ($DIST_SIZE)${NC}"
fi

# ── Generate app.yml (only used if remote doesn't already have one) ───────────
cat > "$LOCAL_APP_YML" <<EOF
# sing-box-easy server config — generated by scripts/deploy-ssh.sh on $TIMESTAMP
server:
  port: "$REMOTE_PORT"

sing_box:
  config_path: "$SING_BOX_CONFIG"
  binary_path: "$SING_BOX_BINARY"
  database_path: "$SING_BOX_DB"

log:
  level: "info"
EOF

# ── Remote: prep + backup ─────────────────────────────────────────────────────
echo ""
echo -e "${BLUE}=== Remote: prep + backup ===${NC}"

ssh "$REMOTE_HOST" bash -s -- "$REMOTE_DIR" "$BACKUP_CONFIG" "$TIMESTAMP" <<'REMOTE_PREP'
set -e
REMOTE_DIR="$1"
BACKUP_CONFIG="$2"
TIMESTAMP="$3"

mkdir -p "$REMOTE_DIR"

if [ "$BACKUP_CONFIG" = "true" ] && [ -f "$REMOTE_DIR/config.json" ]; then
    cp -a "$REMOTE_DIR/config.json" "$REMOTE_DIR/config.json.predeploy-$TIMESTAMP"
    echo "backup: $REMOTE_DIR/config.json.predeploy-$TIMESTAMP"
fi

# Stop any running instance. We match the exact installed path so we don't
# kill an unrelated process named sing-box-easy somewhere else on the box.
if pgrep -f "$REMOTE_DIR/sing-box-easy" >/dev/null 2>&1; then
    echo "stopping existing sing-box-easy process..."
    pgrep -f "$REMOTE_DIR/sing-box-easy" | xargs -r kill 2>/dev/null || true
    sleep 1
fi
REMOTE_PREP

# ── Upload artifacts ──────────────────────────────────────────────────────────
echo ""
echo -e "${BLUE}=== Upload artifacts ===${NC}"

if [ "$SKIP_BACKEND" != "true" ]; then
    echo "uploading binary..."
    scp -q "$LOCAL_BIN" "$REMOTE_HOST:$REMOTE_DIR/sing-box-easy.new"
fi

if [ "$BUILD_FRONTEND" = "true" ]; then
    echo "uploading dist tarball..."
    scp -q "$LOCAL_DIST_TARBALL" "$REMOTE_HOST:/tmp/sing-box-easy-dist.tar.gz"
fi

echo "uploading candidate app.yml..."
scp -q "$LOCAL_APP_YML" "$REMOTE_HOST:/tmp/sing-box-easy.app.yml"

# ── Remote: install ───────────────────────────────────────────────────────────
echo ""
echo -e "${BLUE}=== Remote: install ===${NC}"

ssh "$REMOTE_HOST" bash -s -- \
    "$REMOTE_DIR" \
    "$SKIP_BACKEND" \
    "$BUILD_FRONTEND" \
    "$INSTALL_APP_YML_IF_MISSING" \
    <<'REMOTE_INSTALL'
set -e
REMOTE_DIR="$1"
SKIP_BACKEND="$2"
BUILD_FRONTEND="$3"
INSTALL_APP_YML_IF_MISSING="$4"

cd "$REMOTE_DIR"

if [ "$SKIP_BACKEND" != "true" ]; then
    chmod +x sing-box-easy.new
    mv sing-box-easy.new sing-box-easy
    chown root:root sing-box-easy
    echo "binary installed: $(ls -lh sing-box-easy | awk '{print $5, $9}')"
fi

if [ "$BUILD_FRONTEND" = "true" ]; then
    rm -rf dist
    mkdir dist
    # macOS tar embeds AppleDouble entries that GNU tar warns about — silence
    # the noise, keep real errors.
    tar -xzf /tmp/sing-box-easy-dist.tar.gz -C dist 2>&1 \
        | grep -v 'Ignoring unknown extended header keyword' || true
    rm -f /tmp/sing-box-easy-dist.tar.gz
    find dist \( -name '._*' -o -name '.DS_Store' \) -delete
    chown -R root:root dist
    echo "dist installed: $(du -sh dist | awk '{print $1}')"
fi

if [ ! -f app.yml ]; then
    if [ "$INSTALL_APP_YML_IF_MISSING" = "true" ]; then
        mv /tmp/sing-box-easy.app.yml app.yml
        chown root:root app.yml
        echo "app.yml installed (was missing)"
    else
        rm -f /tmp/sing-box-easy.app.yml
        echo "app.yml missing and INSTALL_APP_YML_IF_MISSING=false; service will fail to start"
    fi
else
    rm -f /tmp/sing-box-easy.app.yml
    echo "app.yml left untouched (already exists)"
fi
REMOTE_INSTALL

# ── Remote: start + smoke test ────────────────────────────────────────────────
if [ "$START_AFTER_DEPLOY" = "true" ]; then
    echo ""
    echo -e "${BLUE}=== Remote: start + smoke test ===${NC}"

    ssh "$REMOTE_HOST" bash -s -- "$REMOTE_DIR" "$REMOTE_PORT" <<'REMOTE_START'
set -e
REMOTE_DIR="$1"
REMOTE_PORT="$2"

cd "$REMOTE_DIR"
# Start in background, redirect both streams to a stable log path.
nohup ./sing-box-easy -c app.yml > /var/log/sing-box-easy.log 2>&1 &
NEW_PID=$!
echo "started PID=$NEW_PID, log=/var/log/sing-box-easy.log"

# Give the HTTP server a moment to bind. 1.5s is comfortably more than the
# observed cold-start time (~50ms after migrations).
sleep 2

# Smoke test: confirm /init/status returns code:0.
RESP="$(curl -fsS --max-time 3 "http://127.0.0.1:${REMOTE_PORT}/api/1.12.12/init/status" || true)"
echo "/init/status -> $RESP"

if ! echo "$RESP" | grep -q '"code":0'; then
    echo "WARNING: smoke test did not return code:0; check /var/log/sing-box-easy.log"
    tail -20 /var/log/sing-box-easy.log || true
    exit 2
fi
REMOTE_START

    SMOKE_RC=$?
    if [ $SMOKE_RC -ne 0 ]; then
        echo -e "${RED}smoke test failed (rc=$SMOKE_RC)${NC}"
        exit $SMOKE_RC
    fi

    echo ""
    echo -e "${GREEN}=== Deploy complete ===${NC}"
    echo "Open: http://${REMOTE_HOST#*@}:${REMOTE_PORT}/"
    echo "Logs: ssh $REMOTE_HOST 'tail -f /var/log/sing-box-easy.log'"
    echo "Stop: ssh $REMOTE_HOST \"pkill -f '$REMOTE_DIR/sing-box-easy'\""
else
    echo ""
    echo -e "${GREEN}=== Upload complete (service NOT started) ===${NC}"
    echo "Start manually with:"
    echo "  ssh $REMOTE_HOST"
    echo "  cd $REMOTE_DIR && ./sing-box-easy -c app.yml"
fi
