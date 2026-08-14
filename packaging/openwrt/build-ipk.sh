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
mkdir -p "$DATA/usr/bin" "$DATA/etc/init.d" "$DATA/etc/sing-box-easy"

install -m 755 "$BINARY" "$DATA/usr/bin/sing-box-easy"
install -m 755 "$SCRIPT_DIR/files/sing-box-easy.init" "$DATA/etc/init.d/sing-box-easy"
install -m 644 "$REPO_ROOT/app.yml.example" "$DATA/etc/sing-box-easy/app.yml.example"

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

# Keep the operator's app.yml across upgrades.
cat > "$CONTROL/conffiles" <<EOF
/etc/sing-box-easy/app.yml
EOF

# Seed app.yml from the example on first install, then enable + start the
# panel. Starting the panel does NOT start sing-box itself.
cat > "$CONTROL/postinst" <<'EOF'
#!/bin/sh
[ -f /etc/sing-box-easy/app.yml ] || cp /etc/sing-box-easy/app.yml.example /etc/sing-box-easy/app.yml
if [ -z "${IPKG_INSTROOT:-}" ]; then
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
