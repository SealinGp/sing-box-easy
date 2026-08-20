#!/bin/bash
# Local dev orchestrator for sing-box-easy — PID-tracked, idempotent.
#
# Each service is launched as a background process. The wrapper PID is written
# to /tmp/sing-box-easy-<name>.pid; stdout+stderr go to /tmp/sing-box-easy-<name>.log.
# Status, stop, and restart operate via those PID files, so re-running `up` is safe.
#
# Usage:
#   scripts/dev-local.sh                       # starts backend and frontend in background (daemon mode)
#   scripts/dev-local.sh up                    # same
#   scripts/dev-local.sh up --logs             # starts services and tails logs
#   scripts/dev-local.sh down                  # stops all services
#   scripts/dev-local.sh status                # displays service table
#   scripts/dev-local.sh restart [<name>]      # restarts one or all services
#   scripts/dev-local.sh logs [<name>] [-f]    # tails logs of one or all services
#
# Services: backend (go run . -c bin/app.yml, port 5100)
#           frontend (bun run dev / vite, port 5179)
#
# Requires: lsof, go (backend), bun (frontend). `sing-box` on PATH is needed for
# config validation inside the panel, but not to boot the dev server.

set -euo pipefail

# ---- Paths --------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
FRONTEND_DIR="$REPO_ROOT/frontend"

PID_PREFIX=/tmp/sing-box-easy

# Backend port must match server.port in bin/app.yml.
# Frontend port must match server.port in frontend/vite.config.ts (strictPort).
BACKEND_PORT=5100
FRONTEND_PORT=5179

SERVICES=(backend frontend)

# ---- Logging helpers ----------------------------------------------------
log()   { printf '\033[1;34m[dev]\033[0m %s\n' "$*"; }
warn()  { printf '\033[1;33m[dev]\033[0m %s\n' "$*" >&2; }
die()   { printf '\033[1;31m[dev]\033[0m %s\n' "$*" >&2; exit 1; }
require() { command -v "$1" >/dev/null 2>&1 || die "missing dependency: $1"; }

pidfile() { echo "$PID_PREFIX-$1.pid"; }
logfile() { echo "$PID_PREFIX-$1.log"; }

# ---- Service registry ---------------------------------------------------
# Port to monitor for each service ("" = no port check).
service_port() {
  case "$1" in
    backend)  echo "$BACKEND_PORT" ;;
    frontend) echo "$FRONTEND_PORT" ;;
    *)        echo "" ;;
  esac
}

service_dir() {
  case "$1" in
    backend)  echo "$REPO_ROOT" ;;
    frontend) echo "$FRONTEND_DIR" ;;
  esac
}

service_cmd() {
  case "$1" in
    backend)  echo "bash ./dev.sh" ;;
    frontend) echo "bash ./dev.sh" ;;
  esac
}

# ---- Process inspection -------------------------------------------------
is_running() {
  local pf
  pf=$(pidfile "$1")
  [ -f "$pf" ] || return 1
  local pid
  pid=$(cat "$pf" 2>/dev/null || true)
  [ -n "$pid" ] || return 1
  kill -0 "$pid" 2>/dev/null
}

# Print PIDs (one per line) listening on the given TCP port.
port_listeners() {
  local port="$1"
  lsof -nP -iTCP:"$port" -sTCP:LISTEN -t 2>/dev/null || true
}

# Recursively kill a process tree (children first, then root).
# `go run` spawns the compiled binary as a child, so killing only the wrapper
# would leave the server holding port 5100.
kill_tree() {
  local pid="$1" sig="${2:-TERM}"
  [ -n "$pid" ] || return 0
  kill -0 "$pid" 2>/dev/null || return 0
  local children
  children=$(pgrep -P "$pid" 2>/dev/null || true)
  for c in $children; do
    kill_tree "$c" "$sig"
  done
  kill "-$sig" "$pid" 2>/dev/null || true
}

# Returns 0 if `candidate` equals `root` or is anywhere in `root`'s subtree.
# Used to credit a port-listener PID back to the wrapper PID we tracked.
in_pid_tree() {
  local root="$1" candidate="$2"
  [ "$root" = "$candidate" ] && return 0
  local children
  children=$(pgrep -P "$root" 2>/dev/null || true)
  for c in $children; do
    in_pid_tree "$c" "$candidate" && return 0
  done
  return 1
}

# ---- Start / stop -------------------------------------------------------
start_service() {
  local name="$1"
  local pf lf dir cmd port
  pf=$(pidfile "$name")
  lf=$(logfile "$name")
  dir=$(service_dir "$name")
  cmd=$(service_cmd "$name")
  port=$(service_port "$name")

  if is_running "$name"; then
    log "$name already running (pid $(cat "$pf"))"
    return 0
  fi

  # Stale pidfile from a crashed run.
  rm -f "$pf"

  # Refuse to start if the port is held by an untracked process.
  if [ -n "$port" ]; then
    local owners
    owners=$(port_listeners "$port")
    if [ -n "$owners" ]; then
      warn "$name: port $port already held by PID(s): $(echo $owners) — skipping start"
      warn "  -> run '$0 down' to clean, or kill manually"
      return 1
    fi
  fi

  log "starting $name (logs: $lf)"
  # Subshell `(...)` runs in background via `&`. `exec bash -c` replaces the
  # subshell so $! is the PID of the running command.
  (
    cd "$dir"
    exec bash -c "$cmd"
  ) >"$lf" 2>&1 &
  echo $! >"$pf"

  # Quick liveness check — if it died within 500ms, surface it now.
  sleep 0.5
  if ! is_running "$name"; then
    warn "$name died immediately — see $lf"
    rm -f "$pf"
    return 1
  fi
}

stop_service() {
  local name="$1"
  local pf port
  pf=$(pidfile "$name")
  port=$(service_port "$name")

  if [ -f "$pf" ]; then
    local pid
    pid=$(cat "$pf" 2>/dev/null || true)
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
      log "stopping $name (pid $pid)"
      kill_tree "$pid" TERM
      sleep 1
      kill_tree "$pid" KILL
    fi
    rm -f "$pf"
  fi

  # Belt-and-suspenders: kill anything still holding the service's port,
  # even if it was started outside this script.
  if [ -n "$port" ]; then
    local owners
    owners=$(port_listeners "$port")
    if [ -n "$owners" ]; then
      warn "$name: port $port still held by $(echo $owners) — killing"
      for p in $owners; do kill_tree "$p" TERM; done
      sleep 1
      owners=$(port_listeners "$port")
      for p in $owners; do kill_tree "$p" KILL; done
    fi
  fi
}

# ---- Status table -------------------------------------------------------
status_one() {
  local name="$1"
  local pf lf pid status port port_field
  pf=$(pidfile "$name")
  lf=$(logfile "$name")
  port=$(service_port "$name")

  if is_running "$name"; then
    pid=$(cat "$pf")
    status="running"
  elif [ -f "$pf" ]; then
    pid=$(cat "$pf" 2>/dev/null || echo "-")
    status="stale"
  else
    pid="-"
    status="stopped"
  fi

  if [ -n "$port" ]; then
    local owner_list
    owner_list=$(port_listeners "$port")
    if [ -z "$owner_list" ]; then
      port_field="$port (free)"
      # Backend compiles before it listens, so "no-port" is normal right after up.
      [ "$status" = "running" ] && status="no-port"
    else
      # Walk our wrapper's tree: anyone in it counts as "ours".
      local ours_count=0 foreign=""
      for owner in $owner_list; do
        if [ "$status" = "running" ] && in_pid_tree "$pid" "$owner"; then
          ours_count=$((ours_count + 1))
        else
          foreign="${foreign:+$foreign,}$owner"
        fi
      done
      if [ -z "$foreign" ]; then
        port_field="$port"
      elif [ "$ours_count" -gt 0 ]; then
        port_field="$port (+foreign: $foreign)"
      else
        port_field="$port (held by $foreign)"
        [ "$status" = "stopped" ] && status="port-held"
      fi
    fi
  else
    port_field="-"
  fi

  printf "  %-10s %-8s %-26s %-12s %s\n" "$name" "$pid" "$port_field" "$status" "$lf"
}

status_table() {
  printf "  %-10s %-8s %-26s %-12s %s\n" "NAME" "PID" "PORT" "STATUS" "LOG"
  printf "  %-10s %-8s %-26s %-12s %s\n" "----" "---" "----" "------" "---"
  for s in "${SERVICES[@]}"; do status_one "$s"; done
}

# ---- Subcommands --------------------------------------------------------
cmd_up() {
  require lsof
  require go
  require bun

  local tail_logs=0
  while [ $# -gt 0 ]; do
    case "$1" in
      --logs|-l) tail_logs=1; shift ;;
      *) die "unknown up flag: $1" ;;
    esac
  done

  [ -f "$REPO_ROOT/bin/app.yml" ] || warn "bin/app.yml missing — backend will fail to boot"
  [ -d "$FRONTEND_DIR/node_modules" ] || warn "frontend deps missing — run: (cd frontend && bun install)"

  if is_running backend || is_running frontend; then
    warn "stack is already running"
    status_table
    return 0
  fi

  # `|| true`: one service failing (e.g. its port is held) must not abort the
  # whole `up` under `set -e` — the status table below reports what came up.
  start_service backend || true
  start_service frontend || true

  echo
  status_table
  echo
  echo "  Frontend  : http://localhost:$FRONTEND_PORT   (proxies /api -> :$BACKEND_PORT)"
  echo "  Backend   : http://localhost:$BACKEND_PORT/api/1.12.12"
  echo
  echo "  Logs   : $0 logs <name> [-f] (or run without args to tail all)"
  echo "  Status : $0 status"
  echo "  Stop   : $0 down"
  echo

  if [ "$tail_logs" = "1" ]; then
    log "Tailing logs of all services. Press Ctrl+C to exit tailing (servers will remain running)."
    echo "--------------------------------------------------------------------------------"
    cmd_logs
  fi
}

cmd_down() {
  for s in frontend backend; do
    stop_service "$s"
  done
  log "stack stopped"
}

cmd_restart() {
  local target="${1:-}"
  if [ -z "$target" ]; then
    cmd_down
    cmd_up
    return
  fi

  local known=0
  for s in "${SERVICES[@]}"; do [ "$s" = "$target" ] && known=1; done
  [ "$known" = "1" ] || die "unknown service: $target (services: ${SERVICES[*]})"

  stop_service "$target"
  start_service "$target"
}

cmd_logs() {
  local name="${1:-}"
  if [ -z "$name" ]; then
    local logfiles=()
    for s in "${SERVICES[@]}"; do
      local lf
      lf=$(logfile "$s")
      [ -f "$lf" ] && logfiles+=("$lf")
    done
    if [ ${#logfiles[@]} -gt 0 ]; then
      tail -f "${logfiles[@]}"
    else
      die "no log files found to tail"
    fi
  else
    local lf
    lf=$(logfile "$name")
    [ -f "$lf" ] || die "no log file: $lf"
    shift || true
    if [ "${1:-}" = "-f" ] || [ -z "${1:-}" ]; then
      tail -f "$lf"
    else
      tail -100 "$lf"
    fi
  fi
}

cmd_status() {
  status_table
}

# ---- Dispatch -----------------------------------------------------------
case "${1:-up}" in
  up|"")                 shift || true; cmd_up "$@" ;;
  down|stop|--stop)      shift || true; cmd_down "$@" ;;
  status|--status|ps)    cmd_status ;;
  restart)               shift || true; cmd_restart "${1:-}" ;;
  logs|log)              shift || true; cmd_logs "$@" ;;
  -h|--help|help)        echo "Usage: $0 {up|down|status|restart|logs}" ;;
  *)                     die "unknown command: $1 (try up|down|status|restart|logs)" ;;
esac
