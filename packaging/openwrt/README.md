# OpenWrt packaging

`build-ipk.sh` assembles a `.ipk` for sing-box-easy without the OpenWrt SDK.
The release workflow runs it for every published architecture; you can also
run it locally against a cross-compiled binary:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o build/sing-box-easy ./main.go
sh packaging/openwrt/build-ipk.sh build/sing-box-easy 1.4.0 aarch64_generic out/
```

## Install on the router

```sh
opkg install ./sing-box-easy_<version>_<arch>.ipk
```

The package:

- installs the single binary to `/usr/bin/sing-box-easy` (web UI embedded)
- installs a procd service `/etc/init.d/sing-box-easy` (enabled + started on
  install; manages the panel only, **not** sing-box itself)
- seeds `/etc/sing-box-easy/app.yml` from the bundled example on first
  install and keeps it across upgrades (conffile)

**Authentication note:** with the default `server.auth: "auto"` the panel
skips login on OpenWrt — anyone who can reach the panel's port has admin
access. Keep the port LAN-only (the default firewall does not expose it to
WAN), or set `server.auth: "enabled"` in `/etc/sing-box-easy/app.yml` to
require login like on other platforms.

Install sing-box itself with `opkg install sing-box` (OpenWrt ≥ 23.05
packages feed) or from the panel's init wizard.

## Architectures

| Release asset arch | opkg Architecture | Typical devices |
|---|---|---|
| `amd64` | `x86_64` | x86 boxes / VMs |
| `arm64` | `aarch64_generic` | R2S/R4S, RPi 4, most ARMv8 routers |
| `arm` (GOARM=7) | `arm_cortex-a7` | 32-bit ARMv7 routers |

If your device reports a more specific arch string (e.g.
`aarch64_cortex-a53`), the binary still runs — install with
`opkg install --force-architecture <ipk>`.

**mips/mipsle (MT7621-class) is not supported**: the pure-Go SQLite driver
(`modernc.org/sqlite`) has no mips port, and those devices are generally too
RAM-constrained for this panel anyway.

## Requirements

- ~25 MB free space on the overlay (or install to USB/extroot)
- 128 MB+ RAM recommended
- If OpenClash / another Clash instance runs on the same router, make sure
  only one of them owns the TUN interface / firewall rules at a time.
