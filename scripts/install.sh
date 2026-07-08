#!/usr/bin/env bash
#
# install.sh — install & launch sing-box-easy on Linux.
#
# Scope (for now): Debian-family distributions on x86_64 only.
#
# What it does:
#   1. Verifies the OS (Debian-family) and architecture (x86_64).
#   2. Resolves the release to install (latest by default) and downloads the
#      matching GitHub Release asset, verifying its sha256 checksum.
#   3. Extracts the package (binary + bundled frontend) into the current dir.
#   4. Determines the sing-box config.json path: uses /etc/sing-box/config.json
#      if present, otherwise prompts for the full path.
#   5. Generates app.yml (preserving an existing one, so upgrades keep your
#      settings) and launches sing-box-easy as a systemd service (falling back
#      to a background nohup process if systemd is unavailable).
#
# Usage:
#   ./install.sh                 # install the latest release
#   ./install.sh v1.2.3          # install a specific release tag
#   VERSION=v1.2.3 ./install.sh  # same, via env var
#
# Optional environment overrides:
#   PORT             HTTP port for sing-box-easy (default: 8080)
#   SINGBOX_CONFIG   Path to sing-box config.json (skips the interactive prompt)
#   INSTALL_DIR      Where to extract/run (default: current directory)
#   ADMIN_USER       Default admin username (default: admin)
#   ADMIN_PASS       Default admin password (default: admin)
#
set -euo pipefail

# ── Constants ─────────────────────────────────────────────────────────────────
REPO="SealinGp/sing-box-easy"
ASSET="sing-box-easy-linux-amd64.tar.gz"
SERVICE_NAME="sing-box-easy"
DEFAULT_SINGBOX_CONFIG="/etc/sing-box/config.json"

# ── Configuration (override via args / env) ────────────────────────────────────
VERSION="${1:-${VERSION:-}}"                 # empty => latest release
INSTALL_DIR="${INSTALL_DIR:-$(pwd)}"
PORT="${PORT:-8080}"
SINGBOX_CONFIG="${SINGBOX_CONFIG:-}"         # empty => detect/prompt
ADMIN_USER="${ADMIN_USER:-}"
ADMIN_PASS="${ADMIN_PASS:-}"
IS_FIRST_INSTALL=false

# ── Colors ──────────────────────────────────────────────────────────────────--
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
info()  { echo -e "${BLUE}==>${NC} $*"; }
ok()    { echo -e "${GREEN}✓${NC} $*"; }
warn()  { echo -e "${YELLOW}!${NC} $*"; }
die()   { echo -e "${RED}error:${NC} $*" >&2; exit 1; }

# Privileged-command prefix: empty when root, otherwise `sudo`.
SUDO=""
if [ "$(id -u)" -ne 0 ]; then
    if command -v sudo >/dev/null 2>&1; then
        SUDO="sudo"
    fi
fi

# ── 1. Detect OS + architecture ────────────────────────────────────────────────
info "Detecting system..."

[ "$(uname -s)" = "Linux" ] || die "this installer supports Linux only (found $(uname -s))"

ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64) : ;;
    *) die "unsupported architecture: $ARCH (only x86_64 is supported for now)" ;;
esac

[ -r /etc/os-release ] || die "/etc/os-release not found; cannot determine the distribution"

# Read distro identity in subshells so /etc/os-release cannot clobber this
# script's own variables — notably VERSION, which os-release also defines
# (e.g. VERSION="12 (bookworm)").
OS_ID="$( . /etc/os-release 2>/dev/null; printf '%s' "${ID:-}" )"
OS_ID_LIKE="$( . /etc/os-release 2>/dev/null; printf '%s' "${ID_LIKE:-}" )"
OS_PRETTY="$( . /etc/os-release 2>/dev/null; printf '%s' "${PRETTY_NAME:-}" )"

# Accept Debian and Debian-family distros (e.g. Ubuntu). Anything else is
# unsupported for now.
is_debian_family=false
case "$OS_ID" in
    debian|ubuntu) is_debian_family=true ;;
esac
case " $OS_ID_LIKE " in
    *" debian "*) is_debian_family=true ;;
esac
[ "$is_debian_family" = "true" ] || \
    die "unsupported distribution: ${OS_PRETTY:-${OS_ID:-unknown}} (only Debian is supported for now)"

ok "Debian-family Linux on x86_64: ${OS_PRETTY:-${OS_ID:-debian}}"

# ── 2. Resolve version + download ──────────────────────────────────────────────
command -v curl >/dev/null 2>&1 || die "curl is required but not installed"
command -v tar  >/dev/null 2>&1 || die "tar is required but not installed"

if [ -z "$VERSION" ]; then
    info "Resolving latest release from GitHub..."
    # Prefer the web "latest" redirect: it does not count against the low
    # unauthenticated API rate limit, which often returns HTTP 403 from shared
    # egress IPs (e.g. a proxy host). The final URL looks like
    # https://github.com/<repo>/releases/tag/<tag>.
    LATEST_URL="$(curl -fsSL -o /dev/null -w '%{url_effective}' \
        "https://github.com/${REPO}/releases/latest" 2>/dev/null || true)"
    case "$LATEST_URL" in
        */releases/tag/*)
            VERSION="${LATEST_URL##*/tag/}"
            VERSION="${VERSION%%/*}"
            ;;
    esac
    # Fall back to the API only if the redirect could not be parsed.
    if [ -z "$VERSION" ]; then
        VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
            | grep -m1 '"tag_name"' \
            | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/')"
    fi
    [ -n "$VERSION" ] || die "could not determine the latest release tag (set VERSION=<tag> to override)"
fi
ok "Target version: $VERSION"

BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

info "Downloading ${ASSET}..."
curl -fL --progress-bar -o "${TMP_DIR}/${ASSET}" "${BASE_URL}/${ASSET}" \
    || die "failed to download ${BASE_URL}/${ASSET}"

# Verify checksum when the .sha256 sidecar is available and sha256sum exists.
if curl -fsSL -o "${TMP_DIR}/${ASSET}.sha256" "${BASE_URL}/${ASSET}.sha256" 2>/dev/null; then
    if command -v sha256sum >/dev/null 2>&1; then
        info "Verifying checksum..."
        ( cd "$TMP_DIR" && sha256sum -c "${ASSET}.sha256" >/dev/null ) \
            || die "checksum verification failed for ${ASSET}"
        ok "Checksum verified"
    else
        warn "sha256sum not found; skipping checksum verification"
    fi
else
    warn "checksum file not published; skipping verification"
fi

# ── 3. Extract into the install directory ──────────────────────────────────────
mkdir -p "$INSTALL_DIR"
INSTALL_DIR="$(cd "$INSTALL_DIR" && pwd)"   # normalise to an absolute path
info "Extracting into ${INSTALL_DIR}..."
tar -xzf "${TMP_DIR}/${ASSET}" -C "$INSTALL_DIR"
chmod +x "${INSTALL_DIR}/sing-box-easy"
[ -f "${INSTALL_DIR}/dist/index.html" ] || die "package is missing the bundled frontend (dist/index.html)"
ok "Extracted sing-box-easy + frontend"

# ── 4. Resolve the sing-box config.json path ───────────────────────────────────
if [ -z "$SINGBOX_CONFIG" ]; then
    if [ -f "$DEFAULT_SINGBOX_CONFIG" ]; then
        SINGBOX_CONFIG="$DEFAULT_SINGBOX_CONFIG"
        ok "Found sing-box config: $SINGBOX_CONFIG"
    else
        warn "sing-box config not found at $DEFAULT_SINGBOX_CONFIG"
        if [ ! -t 0 ]; then
            die "no config found and shell is non-interactive; set SINGBOX_CONFIG=<path> and re-run"
        fi
        # Prompt until the user supplies a path to an existing file.
        while true; do
            printf "Enter the full path to your sing-box config.json: "
            read -r SINGBOX_CONFIG
            [ -n "$SINGBOX_CONFIG" ] || { warn "path cannot be empty"; continue; }
            if [ -f "$SINGBOX_CONFIG" ]; then
                ok "Using sing-box config: $SINGBOX_CONFIG"
                break
            fi
            warn "no file at '$SINGBOX_CONFIG' — try again (Ctrl-C to abort)"
        done
    fi
else
    ok "Using sing-box config from SINGBOX_CONFIG: $SINGBOX_CONFIG"
fi

# Keep the database next to the sing-box config so all state lives together.
SINGBOX_DIR="$(dirname "$SINGBOX_CONFIG")"
DB_PATH="${SINGBOX_DIR}/sing-box-easy.db"

# ── 5. Generate app.yml (preserve an existing one) ──────────────────────────────
# An upgrade must not clobber a user's tuned config. If app.yml already exists we
# keep it untouched and only honour its port for the smoke test below; otherwise
# we generate a fresh one from the detected defaults.
APP_YML="${INSTALL_DIR}/app.yml"
if [ -f "$APP_YML" ]; then
    # Best-effort: read the configured port so the smoke test hits the right
    # address. Strips everything but digits from the first `port:` line; falls
    # back to the current $PORT when nothing parseable is found.
    existing_port="$(grep -m1 -E '^[[:space:]]*port:[[:space:]]*' "$APP_YML" 2>/dev/null | tr -dc '0-9')"
    if [ -n "$existing_port" ]; then
        PORT="$existing_port"
    fi
    ok "Existing app.yml found — keeping it unchanged: $APP_YML (port=${PORT})"
else
    IS_FIRST_INSTALL=true
    info "Writing ${APP_YML}..."
    cat > "$APP_YML" <<EOF
# sing-box-easy server config — generated by scripts/install.sh
server:
  # HTTP server port. Ensure this port is open in your cloud provider security group and system firewall.
  # HTTP 服务器端口。请确保在云服务商安全组和系统防火墙中开放此端口以允许外部访问。
  port: "${PORT}"

sing_box:
  config_path: "${SINGBOX_CONFIG}"
  binary_path: "sing-box"
  database_path: "${DB_PATH}"

log:
  level: "info"
EOF
    ok "Wrote app.yml (port=${PORT}, config=${SINGBOX_CONFIG})"
fi

# ── 6. Launch the service ───────────────────────────────────────────────────────
# Prefer systemd; fall back to a background nohup process.
# Installing a unit requires root (directly or via sudo).
can_privileged=false
if [ "$(id -u)" -eq 0 ] || [ -n "$SUDO" ]; then
    can_privileged=true
fi

have_systemd=false
if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
    if [ "$can_privileged" = "true" ]; then
        have_systemd=true
    else
        warn "systemd detected but no root/sudo access; will start in the background instead"
    fi
fi

start_with_nohup() {
    local log="${INSTALL_DIR}/sing-box-easy.log"
    # Stop any previous instance launched from this directory.
    pkill -f "${INSTALL_DIR}/sing-box-easy" 2>/dev/null || true
    local args=(-c "$APP_YML")
    if [ -n "$ADMIN_USER" ]; then
        args+=("-admin_user=$ADMIN_USER")
    fi
    if [ -n "$ADMIN_PASS" ]; then
        args+=("-admin_pass=$ADMIN_PASS")
    fi
    ( cd "$INSTALL_DIR" && nohup ./sing-box-easy "${args[@]}" >"$log" 2>&1 & )
    ok "Started in background (log: $log)"
    echo "  Stop with: pkill -f '${INSTALL_DIR}/sing-box-easy'"
}

if [ "$have_systemd" = "true" ]; then
    info "Installing systemd service '${SERVICE_NAME}'..."
    UNIT_PATH="/etc/systemd/system/${SERVICE_NAME}.service"
    local exec_start="${INSTALL_DIR}/sing-box-easy -c ${APP_YML}"
    if [ -n "$ADMIN_USER" ]; then
        exec_start="$exec_start -admin_user=${ADMIN_USER}"
    fi
    if [ -n "$ADMIN_PASS" ]; then
        exec_start="$exec_start -admin_pass=${ADMIN_PASS}"
    fi
    # WorkingDirectory must be the install dir: the server resolves the frontend
    # from ./dist relative to its working directory.
    $SUDO tee "$UNIT_PATH" >/dev/null <<EOF
[Unit]
Description=sing-box-easy management service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}
ExecStart=${exec_start}
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
    $SUDO systemctl daemon-reload
    # Enable on boot, then restart explicitly. On an upgrade the service is
    # already running the previous binary; `enable --now` would leave that old
    # process in place, so `restart` is required to pick up the new binary.
    # `restart` also starts the unit when it is currently stopped (fresh install).
    $SUDO systemctl enable "${SERVICE_NAME}.service" >/dev/null 2>&1 || true
    info "Restarting ${SERVICE_NAME} service..."
    $SUDO systemctl restart "${SERVICE_NAME}.service"
    ok "Service enabled and (re)started"
    echo "  Status:  ${SUDO} systemctl status ${SERVICE_NAME}"
    echo "  Restart: ${SUDO} systemctl restart ${SERVICE_NAME}"
    echo "  Logs:    ${SUDO} journalctl -u ${SERVICE_NAME} -f"
else
    warn "systemd not detected; starting in the background instead"
    start_with_nohup
fi

# ── 7. Smoke test ───────────────────────────────────────────────────────────────
info "Waiting for the HTTP server to come up..."
SMOKE_URL="http://127.0.0.1:${PORT}/api/1.12.12/init/status"
ATTEMPTS=10
for i in $(seq 1 "$ATTEMPTS"); do
    if RESP="$(curl -fsS --max-time 3 "$SMOKE_URL" 2>/dev/null)" && echo "$RESP" | grep -q '"code":0'; then
        ok "Service is up: $SMOKE_URL -> $RESP"
        echo ""
        echo -e "${GREEN}=== Install complete ===${NC}"
        echo "Open: http://$(hostname -I 2>/dev/null | awk '{print $1}'):${PORT}/  (or http://127.0.0.1:${PORT}/)"
        warn "Note: If you cannot access the dashboard from outside, verify that port ${PORT} is open in your cloud provider's firewall/security group."
        if [ "$IS_FIRST_INSTALL" = "true" ]; then
            echo ""
            echo -e "Default administrator credentials:"
            echo -e "  Username: ${YELLOW}${ADMIN_USER:-admin}${NC}"
            echo -e "  Password: ${YELLOW}${ADMIN_PASS:-admin}${NC}"
            echo -e "${YELLOW}Please change this password immediately after your first login!${NC}"
        fi
        exit 0
    fi
    sleep 1
done

warn "Smoke test did not confirm a healthy service within ${ATTEMPTS}s."
if [ "$have_systemd" = "true" ]; then
    echo "Inspect logs with: ${SUDO} journalctl -u ${SERVICE_NAME} -e"
else
    echo "Inspect logs with: tail -n 50 ${INSTALL_DIR}/sing-box-easy.log"
fi
exit 1
