#!/bin/sh
# build-ipk.sh — assemble an OpenWrt .ipk for sing-box-easy without the
# OpenWrt SDK. An ipk is a gzipped tar containing debian-binary,
# control.tar.gz and data.tar.gz, which plain tar/gzip can produce.
#
# Usage:
#   build-ipk.sh <binary> <version> <openwrt-arch> <output-dir>
#
#   binary        path to the (already cross-compiled) sing-box-easy binary
#   version       package version, without leading "v" (e.g. 1.4.0)
#   openwrt-arch  opkg Architecture string (x86_64, aarch64_generic,
#                 arm_cortex-a7, mipsel_24kc, ...)
#   output-dir    directory the finished ipk is written into
#
# The frontend is embedded in the binary (app/webui via go:embed), so the
# package ships no dist directory.
set -eu

BINARY="$1"
VERSION="$2"
ARCH="$3"
OUTDIR="$4"

[ -f "$BINARY" ] || { echo "binary not found: $BINARY" >&2; exit 1; }
case "$VERSION" in
	v*) echo "version must not include the leading 'v': $VERSION" >&2; exit 1 ;;
esac

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

# ── data.tar.gz: the installed file tree ─────────────────────────────────────
DATA="$WORK/data"
mkdir -p "$DATA/usr/bin" "$DATA/etc/init.d" "$DATA/etc/sing-box-easy" "$DATA/etc/config"

install -m 755 "$BINARY" "$DATA/usr/bin/sing-box-easy"
install -m 755 "$SCRIPT_DIR/files/sing-box-easy.init" "$DATA/etc/init.d/sing-box-easy"
install -m 644 "$REPO_ROOT/app.yml.example" "$DATA/etc/sing-box-easy/app.yml.example"

# Ship app.yml itself, not just the example.
#
# It is listed in conffiles, and opkg checksums every conffile during install.
# When the file was created by postinst instead of being in the package, that
# checksum step failed noisily:
#
#   file_sha256sum_alloc: Failed to open file /etc/sing-box-easy/app.yml
#
# The install still worked, but the error is alarming and pointless. Shipping
# the file makes conffiles behave as intended: opkg now preserves an edited
# app.yml across upgrades instead of tracking a file it never saw.
install -m 644 "$REPO_ROOT/app.yml.example" "$DATA/etc/sing-box-easy/app.yml"
install -m 644 "$SCRIPT_DIR/files/uci-sing-box-easy" "$DATA/etc/config/sing-box-easy"

# ── LuCI entry (OpenWrt only) ────────────────────────────────────────────────
# A menu item under Services pointing at the panel, so it is reachable the way
# every other OpenWrt service is rather than only by typing host:8080.
#
# These files are inert anywhere LuCI is absent — they are plain data under
# /usr/share and /www that nothing reads unless luci-base is installed — so the
# package stays dependency-free and still installs on a bare OpenWrt.
mkdir -p "$DATA/usr/share/luci/menu.d" \
         "$DATA/usr/share/rpcd/acl.d" \
         "$DATA/www/luci-static/resources/view/sing-box-easy"

install -m 644 "$SCRIPT_DIR/files/luci/menu.d/luci-app-sing-box-easy.json" \
	"$DATA/usr/share/luci/menu.d/luci-app-sing-box-easy.json"
install -m 644 "$SCRIPT_DIR/files/luci/acl.d/luci-app-sing-box-easy.json" \
	"$DATA/usr/share/rpcd/acl.d/luci-app-sing-box-easy.json"
install -m 644 "$SCRIPT_DIR/files/luci/view/sing-box-easy/panel.js" \
	"$DATA/www/luci-static/resources/view/sing-box-easy/panel.js"

# ── control.tar.gz: package metadata + maintainer scripts ────────────────────
CONTROL="$WORK/control"
mkdir -p "$CONTROL"

INSTALLED_SIZE=$(wc -c < "$DATA/usr/bin/sing-box-easy" | tr -d ' ')

cat > "$CONTROL/control" <<EOF
Package: sing-box-easy
Version: $VERSION
Architecture: $ARCH
Maintainer: SealinGp
Section: net
Priority: optional
Depends: libc
Installed-Size: $INSTALLED_SIZE
Description: Web panel for managing sing-box configurations and lifecycle.
 RESTful API + embedded web UI for sing-box config management,
 subscriptions, and service control. Install sing-box itself via
 'opkg install sing-box' or from the panel.
EOF

# Keep the operator's edits across upgrades. Both files are shipped in
# data.tar.gz so opkg can checksum them at install time.
cat > "$CONTROL/conffiles" <<EOF
/etc/sing-box-easy/app.yml
/etc/config/sing-box-easy
EOF

# Seed app.yml from the example on first install, then enable + start the
# panel. Starting the panel does NOT start sing-box itself.
cat > "$CONTROL/postinst" <<'EOF'
#!/bin/sh
# app.yml ships in the package now; this only covers an upgrade from a build
# that did not include it.
[ -f /etc/sing-box-easy/app.yml ] || cp /etc/sing-box-easy/app.yml.example /etc/sing-box-easy/app.yml

if [ -z "${IPKG_INSTROOT:-}" ]; then
	# Point the LuCI menu entry at the port the panel actually listens on.
	# app.yml is the single source of truth; UCI only mirrors it for the view.
	if [ -x /sbin/uci ] || command -v uci >/dev/null 2>&1; then
		port=$(sed -n 's/^[[:space:]]*port:[[:space:]]*"\{0,1\}\([0-9]\{1,\}\)"\{0,1\}[[:space:]]*$/\1/p' \
			/etc/sing-box-easy/app.yml 2>/dev/null | head -n 1)
		if [ -n "$port" ]; then
			uci -q set sing-box-easy.main=sing-box-easy
			uci -q set sing-box-easy.main.port="$port"
			uci -q commit sing-box-easy
		fi
	fi

	# LuCI caches its menu tree; without this the new entry only appears after
	# the cache expires or the router reboots.
	rm -f /tmp/luci-indexcache* /tmp/luci-modulecache/* 2>/dev/null

	/etc/init.d/sing-box-easy enable
	/etc/init.d/sing-box-easy start
fi
exit 0
EOF

cat > "$CONTROL/prerm" <<'EOF'
#!/bin/sh
if [ -z "${IPKG_INSTROOT:-}" ]; then
	/etc/init.d/sing-box-easy stop || true
	/etc/init.d/sing-box-easy disable || true
fi
exit 0
EOF

chmod 755 "$CONTROL/postinst" "$CONTROL/prerm"

# ── assemble the ipk ─────────────────────────────────────────────────────────
echo "2.0" > "$WORK/debian-binary"

# --owner/--group=0 requires GNU tar (present on the CI runners this runs on).
tar -C "$CONTROL" --owner=0 --group=0 -czf "$WORK/control.tar.gz" .
tar -C "$DATA" --owner=0 --group=0 -czf "$WORK/data.tar.gz" .

mkdir -p "$OUTDIR"
IPK="$OUTDIR/sing-box-easy_${VERSION}_${ARCH}.ipk"
tar -C "$WORK" --owner=0 --group=0 -czf "$IPK" ./debian-binary ./control.tar.gz ./data.tar.gz

echo "built: $IPK"
