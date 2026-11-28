#!/usr/bin/env bash
# Interactive Debian/Ubuntu x86_64/arm64 installer. Also works with curl | bash.
# VERSION or argument 1 pins the panel release. NONINTERACTIVE=1 uses env values.
# INSTALL_DIR, PORT, SINGBOX_CONFIG, ADMIN_USER, ADMIN_PASS, SINGBOX_VERSION
# (a version or "skip") can prefill the wizard. Existing app.yml is kept by default.
set -euo pipefail

info() { printf '\n==> %s\n' "$*"; }
warn() { printf '! %s\n' "$*" >&2; }
die() { warn "$*"; exit 1; }
fetch() { curl --retry 2 --connect-timeout 15 --max-time 180 -fsSL "$@"; }
ask() {
    local variable=$1 label=$2 fallback=$3 reply
    if [ "$INTERACTIVE" = true ]; then
        printf '%s [%s]: ' "$label" "$fallback" >&3
        IFS= read -r reply <&3 || die '输入中断，尚未应用安装。'
    else reply=''; fi
    printf -v "$variable" '%s' "${reply:-$fallback}"
}
secret() {
    local variable=$1 reply
    TTY_STATE=$(stty -g <&3)
    stty -echo <&3
    printf '%s: ' "$2" >&3
    if ! IFS= read -r reply <&3; then
        stty "$TTY_STATE" <&3; TTY_STATE=''; die '输入中断。'
    fi
    stty "$TTY_STATE" <&3; TTY_STATE=''
    printf '\n' >&3
    printf -v "$variable" '%s' "$reply"
}
# JSON quoted strings are valid YAML scalars, including quotes and backslashes.
yaml_string() { python3 -c 'import json,sys; print(json.dumps(sys.argv[1],ensure_ascii=False))' "$1"; }
valid_path() { [[ $1 == /* && $1 != *$'\n'* && $1 != *$'\r'* ]]; }
valid_version() { [[ $1 =~ ^v?[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$ ]]; }
# Check both address families, including listeners bound to a particular interface.
# /proc checks catch non-wildcard listeners; bind catches races and reservations.
free_port() {
    python3 - "$1" <<'PY'
import socket, sys, random, errno
requested = int(sys.argv[1])
used = set()
for path in ('/proc/net/tcp', '/proc/net/tcp6'):
    try:
        for line in open(path).readlines()[1:]:
            fields = line.split()
            if fields[3] == '0A': used.add(int(fields[1].split(':')[1], 16))
    except FileNotFoundError: pass
for _ in range(100):
    port = requested or random.SystemRandom().randint(20000, 60000)
    if not 1 <= port <= 65535 or port in used:
        if requested: break
        continue
    sockets = []
    try:
        for family, address in ((socket.AF_INET, '0.0.0.0'), (socket.AF_INET6, '::')):
            try: sock = socket.socket(family)
            except OSError:
                if family == socket.AF_INET6: continue
                raise
            sockets.append(sock)
            if family == socket.AF_INET6: sock.setsockopt(socket.IPPROTO_IPV6, socket.IPV6_V6ONLY, 1)
            try: sock.bind((address, port))
            except OSError as error:
                if family == socket.AF_INET6 and error.errno in (errno.EAFNOSUPPORT, errno.EADDRNOTAVAIL): continue
                raise
        print(port)
        sys.exit(0)
    except OSError:
        if requested: break
    finally:
        for sock in sockets: sock.close()
sys.exit(1)
PY
}
choose_port() {
    local suggested candidate
    suggested=${PORT:-$(free_port 0)}
    while :; do
        ask candidate '面板监听端口（回车使用推荐的空闲端口）' "$suggested"
        if [[ $candidate =~ ^[0-9]{1,5}$ ]] && free_port "$candidate" >/dev/null; then
            PORT=$((10#$candidate)); return
        fi
        [ "$INTERACTIVE" = true ] || die "端口无效或已占用：$candidate"
        warn '请输入 1–65535 之间的未占用端口。'
        suggested=$(free_port 0)
    done
}
choose_credentials() {
    ask ADMIN_USER '管理员用户名' "${ADMIN_USER:-admin}"
    [ -n "$ADMIN_USER" ] || die '用户名不能为空。'
    if [ -z "${ADMIN_PASS:-}" ]; then
        [ "$INTERACTIVE" = true ] || die '非交互首次安装需要 ADMIN_PASS；不会使用默认密码。'
        local confirmation
        while :; do
            secret ADMIN_PASS '管理员密码（输入不显示）'
            secret confirmation '再次输入密码'
            if [ -n "$ADMIN_PASS" ] && [ "$ADMIN_PASS" = "$confirmation" ]; then break; fi
            warn '密码不能为空，两次输入必须一致，请重试。'
        done
    fi
}
choose_core() {
    local choice index=0 tag label installed=''
    CORE_VERSION=${SINGBOX_VERSION:-}
    command -v sing-box >/dev/null 2>&1 && installed=$(sing-box version 2>/dev/null | head -n1) || true
    [ -z "$installed" ] || printf '已检测到：%s\n' "$installed"
    if [ -n "$CORE_VERSION" ]; then
        [ "$CORE_VERSION" = skip ] || valid_version "$CORE_VERSION" || die 'SINGBOX_VERSION 格式无效。'
        return
    fi
    if [ "$INTERACTIVE" != true ]; then CORE_VERSION=skip; return; fi
    local versions=()
    if fetch 'https://api.github.com/repos/SagerNet/sing-box/releases?per_page=20' > "$TMP_DIR/releases.json"; then
        while IFS= read -r tag; do versions+=("$tag"); done < <(python3 - "$TMP_DIR/releases.json" <<'PY'
import json, sys, re
for release in json.load(open(sys.argv[1])):
    tag = release.get('tag_name', '')
    if not release.get('draft') and re.fullmatch(r'v?\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?', tag): print(tag)
PY
        )
    else warn '版本列表获取失败；可以输入准确版本，或稍后在设置页安装。'; fi
    printf '0) 跳过%s\n' "${installed:+，保留当前 sing-box}"
    for tag in "${versions[@]}"; do
        index=$((index + 1)); label='稳定版'; [[ $tag != *-* ]] || label='预发布版'
        printf '%s) %s · %s\n' "$index" "$tag" "$label"
    done
    printf '也可以输入完整版本号，例如 1.14.1。\n'
    while :; do
        local recommended=0
        if [ "${ROLE:-keep}" != keep ]; then
            for ((index=0;index<${#versions[@]};index++)); do
                if [[ ${versions[$index]} != *-* ]]; then recommended=$((index+1)); break; fi
            done
        fi
        ask choice '选择 sing-box 版本（0 跳过）' "$recommended"
        if [ "$choice" = 0 ]; then CORE_VERSION=skip; break; fi
        if [[ $choice =~ ^[0-9]{1,2}$ ]] && ((10#$choice >= 1 && 10#$choice <= ${#versions[@]})); then
            CORE_VERSION=${versions[$((10#$choice - 1))]}; break
        fi
        if valid_version "$choice"; then CORE_VERSION=$choice; break; fi
        warn '请输入列表编号、完整版本号或 0。'
    done
}
choose_role() {
    local choice
    ROLE=${DEPLOYMENT_ROLE:-keep}
    if [ "$KEEP_CONFIG" = true ]; then ROLE=keep; return; fi
    ask choice '设备用途：1 出口节点服务器；2 客户端/局域网网关；3 仅面板/保留配置' "${DEPLOYMENT_ROLE:-3}"
    case "${choice:-3}" in 1|exit) ROLE=exit;; 2|client) ROLE=client;; 3|keep) ROLE=keep;; *) die '无效用途。';; esac
    if [ "$ROLE" = exit ]; then
        ask PROXY_PORT 'Shadowsocks 2022 入站端口（0 自动选择 TCP/UDP 空闲端口）' "${PROXY_PORT:-0}"
        [[ $PROXY_PORT =~ ^[0-9]{1,5}$ ]] && [ "$PROXY_PORT" -le 65535 ] || die '入站端口无效。'
        info '将生成随机协议密码，Google DNS，出站直连；需要放行入站 TCP/UDP 端口。'
    elif [ "$ROLE" = client ]; then
        ask SUBSCRIPTION_URL '订阅 URL 或单个节点链接（本次导入）' "${SUBSCRIPTION_URL:-}"
        [ -n "$SUBSCRIPTION_URL" ] || die '客户端配置需要订阅或节点链接。'
        ask LAN_ADDRESS '为局域网提供 DNS？输入本机内网 IP；留空仅本机 TUN' "${LAN_ADDRESS:-}"
        info 'TUN 启用后接管本机流量。网关模式还需配置设备网关/DNS及系统 IP 转发；不会自动改防火墙。'
    fi
    if [ "$ROLE" != keep ] && [ -f "$SINGBOX_CONFIG" ]; then
        warn '生成用途配置将替换现有 sing-box 配置，应用前会备份。'
    fi
}
apply_profile() {
    [ "$ROLE" != keep ] || return 0
    printf '%s' "${SUBSCRIPTION_URL:-}" > "$TMP_DIR/subscription"
    python3 - "$ROLE" "${PROXY_PORT:-0}" "${LAN_ADDRESS:-}" "$TMP_DIR/subscription" > "$TMP_DIR/profile-request" <<'PYREQUEST'
import json,sys
print(json.dumps({'role':sys.argv[1],'port':int(sys.argv[2]),'lan_address':sys.argv[3],'subscription':open(sys.argv[4]).read()}))
PYREQUEST
    $SUDO "$INSTALL_DIR/sing-box-easy" -generate_profile -profile_output "$TMP_DIR/profile.json" < "$TMP_DIR/profile-request"
    if command -v sing-box >/dev/null; then $SUDO sing-box check -c "$TMP_DIR/profile.json"; fi
    if [ -f "$SINGBOX_CONFIG" ]; then $SUDO cp -p "$SINGBOX_CONFIG" "$SINGBOX_CONFIG.backup.$(date +%Y%m%d%H%M%S)"; fi
    $SUDO install -m 600 "$TMP_DIR/profile.json" "$SINGBOX_CONFIG"
    info "用途配置已写入 $SINGBOX_CONFIG；协议密码可在面板入站页查看和复制。"
    if [ "$CORE_VERSION" != skip ] && command -v systemctl >/dev/null && [ -d /run/systemd/system ]; then
        local start_default=y answer
        [ "$ROLE" != client ] || start_default=n
        ask answer '现在启动 sing-box？客户端 TUN 会改变本机路由' "$start_default"
        case "$answer" in y|Y|yes)
            $SUDO mkdir -p /etc/systemd/system/sing-box.service.d
            printf '[Service]\nExecStart=\nExecStart=/usr/bin/sing-box run -c %s\n' "$(unit_quote "$SINGBOX_CONFIG" command)" > "$TMP_DIR/core-unit"
            $SUDO install -m 644 "$TMP_DIR/core-unit" /etc/systemd/system/sing-box.service.d/panel-config.conf
            $SUDO systemctl daemon-reload
            $SUDO systemctl enable sing-box
            $SUDO systemctl restart sing-box;;
        esac
    fi
}
configure() {
    info '1/5 安装位置'
    ask INSTALL_DIR '面板安装目录' "${INSTALL_DIR:-$PWD}"
    valid_path "$INSTALL_DIR" || die '安装目录必须是绝对路径，且不能包含换行。'
    APP_YML="$INSTALL_DIR/app.yml"
    KEEP_CONFIG=false
    if [ -f "$APP_YML" ]; then
        local choice
        ask choice '已有 app.yml：1 保留配置和账号；2 重新配置（旧文件会备份，已有账号不重置）' 1
        case "$choice" in 1) KEEP_CONFIG=true;; 2) :;; *) die '无效选项。';; esac
    fi
    if [ "$KEEP_CONFIG" = true ]; then
        PORT=$($SUDO python3 - "$APP_YML" <<'PY'
import re,sys
text=open(sys.argv[1]).read()
m=re.search(r'^\s*port:\s*["\x27]?(\d+)',text,re.M)
if not m: sys.exit('无法读取已有 app.yml 的端口，请检查配置。')
print(m[1])
PY
        )
        info "2/5 保留已有配置，监听端口 $PORT"
        info '3/5 保留已有管理员账号和密码'
    else
        info '2/5 面板与 sing-box 配置'
        choose_port
        ask SINGBOX_CONFIG 'sing-box 配置文件路径（已有文件原样保留；新路径可在面板初始化）' "${SINGBOX_CONFIG:-/etc/sing-box/config.json}"
        valid_path "$SINGBOX_CONFIG" || die '配置文件必须使用绝对路径，且不能包含换行。'
        DB_PATH="$(dirname "$SINGBOX_CONFIG")/sing-box-easy.db"
        if [ -f "$SINGBOX_CONFIG" ]; then info '将使用现有 sing-box 配置，不覆盖内容。'
        else info '该配置尚不存在，安装后在面板初始化向导中创建。'; fi
        if [ -f "$DB_PATH" ]; then
            info '3/5 检测到已有数据库，保留账号；如需改密请在面板操作。'
            ADMIN_USER=''; ADMIN_PASS=''
        else
            info '3/5 创建初始管理员'
            choose_credentials
        fi
    fi
    choose_role
    info '4/5 sing-box 内核（可跳过）'
    choose_core
    local host
    host=$(hostname -I 2>/dev/null | awk '{print $1}') || host=''
    host=${host:-127.0.0.1}; [[ $host != *:* ]] || host="[$host]"
    SETTINGS_URL="http://${host}:${PORT}/dashboard/settings"
    [ "$CORE_VERSION" != skip ] || info "已跳过 sing-box 安装。稍后打开 $SETTINGS_URL 安装。"
    info '5/5 确认安装'
    printf '目录：%s\n端口：%s\n配置：%s\nsing-box：%s\n' "$INSTALL_DIR" "$PORT" "$APP_YML" "$CORE_VERSION"
    local answer
    ask answer '开始安装？y 开始，n 退出' y
    case "$answer" in y|Y|yes) :;; *) info '已取消，未修改安装配置。'; exit 0;; esac
}
write_config() {
    [ "$KEEP_CONFIG" != true ] || return 0
    if [ -f "$APP_YML" ]; then $SUDO cp -p "$APP_YML" "$APP_YML.backup.$(date +%Y%m%d%H%M%S)"; fi
    cat > "$TMP_DIR/app.yml" <<YAML
# Generated by the sing-box-easy installation wizard.
server:
  port: $(yaml_string "$PORT")
  auth: "enabled"
sing_box:
  config_path: $(yaml_string "$SINGBOX_CONFIG")
  binary_path: "sing-box"
  database_path: $(yaml_string "$DB_PATH")
log:
  level: "info"
YAML
    $SUDO mkdir -p "$(dirname "$SINGBOX_CONFIG")"
    $SUDO install -m 600 "$TMP_DIR/app.yml" "$APP_YML"
}
write_launcher() {
    # Keep credentials out of the unit file and shell interpolation. The seed
    # file is removed once startup succeeds; existing accounts are never reset.
    local credentials="$INSTALL_DIR/.bootstrap-password"
    if [ "$KEEP_CONFIG" != true ] && [ -n "${ADMIN_PASS:-}" ]; then
        printf '%s' "$ADMIN_PASS" > "$TMP_DIR/admin"
        $SUDO install -m 600 "$TMP_DIR/admin" "$credentials"
    fi
    {
        printf '#!/usr/bin/env bash\nset -euo pipefail\n'
        printf 'cd %q\n' "$INSTALL_DIR"
        printf 'args=(-c %q)\n' "$APP_YML"
        printf 'seed_user=%q\n' "${ADMIN_USER:-admin}"
        cat <<'SH'
unset ADMIN_PASS
if [ -f .bootstrap-password ]; then
    args+=("-admin_user=$seed_user" "-admin_pass_file=$PWD/.bootstrap-password")
fi
exec ./sing-box-easy "${args[@]}"
SH
    } > "$TMP_DIR/start-panel"
    $SUDO install -m 700 "$TMP_DIR/start-panel" "$INSTALL_DIR/start-panel"
}
unit_quote() {
    local value=$1
    value=${value//\\/\\\\}; value=${value//\"/\\\"}; value=${value//%/%%}
    if [ "${2:-}" = command ]; then value=${value//\$/\$\$}; fi
    printf '"%s"' "$value"
}
start_panel() {
    if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
        cat > "$TMP_DIR/unit" <<UNIT
[Unit]
Description=sing-box-easy management service
After=network-online.target
Wants=network-online.target
[Service]
Type=simple
WorkingDirectory=$(unit_quote "$INSTALL_DIR")
ExecStart=$(unit_quote "$INSTALL_DIR/start-panel" command)
Restart=on-failure
RestartSec=3
[Install]
WantedBy=multi-user.target
UNIT
        $SUDO install -m 644 "$TMP_DIR/unit" /etc/systemd/system/sing-box-easy.service
        $SUDO systemctl daemon-reload
        $SUDO systemctl enable sing-box-easy.service
        $SUDO systemctl restart sing-box-easy.service
    else
        if ! free_port "$PORT" >/dev/null; then die "端口 $PORT 已占用，无法启动后台服务；请先停止旧面板。"; fi
        warn '未检测到 systemd，使用后台进程（不会自动开机启动）。'
        $SUDO bash -c 'nohup "$1/start-panel" > "$1/sing-box-easy.log" 2>&1 & echo $! > "$1/sing-box-easy.pid"' _ "$INSTALL_DIR"
    fi
}
main() {
    umask 077
    local VERSION=${1:-${VERSION:-}} archive=sing-box-easy-linux-amd64.tar.gz repo=SealinGp/sing-box-easy
    SUDO=''; INTERACTIVE=false
    if [ "${NONINTERACTIVE:-0}" != 1 ] && { exec 3<>/dev/tty; } 2>/dev/null; then INTERACTIVE=true; fi
    [ "$(uname -s)" = Linux ] || die '目前支持 Debian/Ubuntu Linux。'
    case "$(uname -m)" in x86_64|amd64) :;; aarch64|arm64) archive=sing-box-easy-linux-arm64.tar.gz;; *) die '目前支持 x86_64 / arm64 架构。';; esac
    local distro
    distro=$( . /etc/os-release; printf '%s %s' "${ID:-}" "${ID_LIKE:-}" )
    case " $distro " in *' debian '*|*' ubuntu '*) :;; *) die '目前仅支持 Debian/Ubuntu。';; esac
    for tool in curl tar python3 sha256sum; do command -v "$tool" >/dev/null || die "缺少依赖 $tool，请先安装。"; done
    if [ "$(id -u)" != 0 ]; then command -v sudo >/dev/null || die '需要 root 或 sudo 权限。'; SUDO=sudo; sudo -v; fi
    TMP_DIR=$(mktemp -d)
    trap '[ -z "${TTY_STATE:-}" ] || stty "$TTY_STATE" <&3; rm -rf "$TMP_DIR"' EXIT
    configure
    if [ -z "$VERSION" ]; then
        local latest
        latest=$(fetch -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest")
        VERSION=${latest##*/tag/}
    fi
    valid_version "$VERSION" || die '无法确定面板版本，请显式传入 vX.Y.Z。'
    local url="https://github.com/$repo/releases/download/$VERSION"
    info "下载面板 $VERSION"
    fetch "$url/$archive" -o "$TMP_DIR/$archive"
    fetch "$url/$archive.sha256" -o "$TMP_DIR/checksum"
    local expected actual
    expected=$(awk 'NR==1 {print $1}' "$TMP_DIR/checksum")
    actual=$(sha256sum "$TMP_DIR/$archive" | awk '{print $1}')
    [[ $expected =~ ^[a-fA-F0-9]{64}$ && $actual = "$expected" ]] || die '安装包校验失败。'
    mkdir "$TMP_DIR/package"
    tar -xzf "$TMP_DIR/$archive" -C "$TMP_DIR/package"
    [ -f "$TMP_DIR/package/sing-box-easy" ] && [ -f "$TMP_DIR/package/dist/index.html" ] || die '安装包缺少程序或前端。'
    if [ "$KEEP_CONFIG" != true ] && [ -n "${ADMIN_PASS:-}" ]; then
        "$TMP_DIR/package/sing-box-easy" -h 2>&1 | grep -q admin_pass_file || die '该面板版本不支持密码文件，请安装 v1.19.0 或更新版本。'
    fi
    if [ "$CORE_VERSION" != skip ]; then
        info "安装 sing-box ${CORE_VERSION#v}"
        fetch https://sing-box.app/install.sh -o "$TMP_DIR/install-core.sh"
        $SUDO sh "$TMP_DIR/install-core.sh" --version "${CORE_VERSION#v}"
        sing-box version
    fi
    if [ ! -d "$INSTALL_DIR" ]; then $SUDO install -d -m 755 "$INSTALL_DIR"; fi
    if [ -f "$INSTALL_DIR/sing-box-easy.pid" ]; then
        local old_pid old_exe
        old_pid=$($SUDO cat "$INSTALL_DIR/sing-box-easy.pid")
        if [[ $old_pid =~ ^[0-9]+$ ]] && [ "$old_pid" -gt 1 ]; then
            old_exe=$($SUDO readlink "/proc/$old_pid/exe" 2>/dev/null || true)
            if [ "$old_exe" = "$INSTALL_DIR/sing-box-easy" ]; then
                $SUDO kill "$old_pid"
                for attempt in {1..20}; do $SUDO kill -0 "$old_pid" 2>/dev/null || break; sleep 1; done
            fi
        fi
    fi
    $SUDO install -m 755 "$TMP_DIR/package/sing-box-easy" "$INSTALL_DIR/sing-box-easy.new"
    $SUDO mv -f "$INSTALL_DIR/sing-box-easy.new" "$INSTALL_DIR/sing-box-easy"
    $SUDO cp -R "$TMP_DIR/package/dist" "$INSTALL_DIR/"
    write_config
    write_launcher
    apply_profile
    start_panel
    info '等待面板启动'
    local attempt
    for attempt in {1..20}; do
        if curl -fsS --max-time 2 "http://127.0.0.1:$PORT/api/v1/auth/status" | grep -Eq '"code"[[:space:]]*:[[:space:]]*0'; then
            $SUDO rm -f "$INSTALL_DIR/.bootstrap-password"
            info "安装完成：${SETTINGS_URL%/dashboard/settings}/"
            [ "$CORE_VERSION" != skip ] || printf '安装 sing-box：%s\n' "$SETTINGS_URL"
            printf '管理员密码不会显示；使用刚才设置的密码登录。已有数据库则使用原账号。\n'
            return
        fi
        sleep 1
    done
    die "未确认面板启动成功。检查 systemctl status sing-box-easy 或 $INSTALL_DIR/sing-box-easy.log。"
}
# Allow isolated function tests; execute normally both from a file and curl | bash.
if [ -z "${BASH_SOURCE[0]:-}" ] || [ "${BASH_SOURCE[0]}" = "$0" ]; then main "$@"; fi
