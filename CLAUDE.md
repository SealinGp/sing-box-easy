# CLAUDE.md
claude --resume dd747f67-5dd5-4ea0-b4af-9bf313c2a125

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

sing-box-easy is a RESTful API service for managing sing-box configurations and controlling the sing-box proxy service. It provides:
- Full sing-box configuration management (Inbound, Outbound, DNS, Route, etc.)
- Service lifecycle control (start, stop, restart)
- Subscription management with auto-update
- Multi-protocol node parsing (shadowsocks, vmess, trojan, vless)
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

**Always use `bun` for the frontend** (install, scripts, and package management) — do NOT use `npm`/`npx`/`yarn`/`pnpm`.

```bash
cd frontend

# Install dependencies
bun install

# Dev server
bun run dev

# Build for production (runs vue-tsc type-check + vite build)
bun run build

# Preview production build
bun run preview

# Run a one-off binary (npx equivalent)
bunx <tool>
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
- `app/pkg/appconfig/` - Application configuration loading (app.yml), including `github.oauth_client_id` / `github.proxy`
- `app/pkg/githubauth/` - GitHub sign-in via OAuth Device Flow (RFC 8628): no client secret, no callback URL, so it works at any self-hosted address. `device.go` speaks the protocol; `manager.go` owns login sessions and persists the issued token through the `TokenStore` interface (satisfied by `settings.ManagerXORM`). The device code never leaves the server
- `app/pkg/config/` - sing-box config management with validation and rollback
- `app/pkg/configversion/` - DB-backed config version history store (XORM) + retention cleaner (used by `config.Manager` snapshots and the rollback endpoints)
- `app/pkg/settings/` - Application settings persistence (XORM); served via `GET/PUT /settings`
- `app/pkg/service/` - sing-box service lifecycle control (start/stop/restart). A `Backend` interface (`backend.go`) abstracts the init system, detected in order: `systemd` (systemctl + journald), `procd` (OpenWrt `/etc/init.d/sing-box` + logread), `process` (direct spawn/signal). The `Controller` owns config validation and delegates lifecycle to the backend
- `app/webui/` - frontend bundle embedded into the binary via `go:embed` (release builds copy the vite dist in before compiling; a committed placeholder covers dev builds). SPA serving prefers an on-disk `./dist`, falling back to the embedded bundle — single-binary installs (OpenWrt ipk) ship no dist directory
- `app/pkg/openwrtnet/` - OpenWrt host-network integration applied around the sing-box lifecycle. `DerivePlan` reads the config (as generic JSON — `Inbound.Options` is a registry-populated `any`) for a tun inbound and for the inbound named by a `hijack-dns` route rule; `Manager.Apply` then creates a `sing-box-easy` firewall zone for the tun device (`masq=0` on purpose: sing-box dials its own outbounds, so NAT would erase the client IP that route rules match on) and repoints dnsmasq at sing-box's DNS port, recording the previous upstreams. It also exempts the names a `hosts` DNS server maps to a private address from dnsmasq's rebind protection (`rebind_domain`) — once dnsmasq forwards to sing-box those answers arrive from an "upstream" and `stop-dns-rebind` strips them, so the client sees NOERROR with zero records for a name sing-box resolved correctly. Every dhcp edit is staged and committed together so dnsmasq restarts once. `Manager.Revert` restores the upstreams and removes only the exemptions it added. State lives in `/etc/sing-box/.openwrt-net-state.json` so it survives a panel restart. Every operation is a no-op off OpenWrt. Wired into `service.Controller`: apply **before** `backend.Start` (the tun device appears with sing-box, and until its zone exists fw4 resets everything coming back off it), revert **after** `backend.Stop`
- `app/pkg/ruleset/` - Evaluates `route.rule_set` **without a running sing-box**. Remote sets are read from the same bbolt cache file sing-box populates (`experimental.cache_file`), local ones from their path; both are decoded with sing-box's own `common/srs`. bbolt gives writers an exclusive flock, so a read-only open blocks while sing-box runs — the file is then copied and the copy read. Matching is reimplemented in `matcher.go` rather than delegated to `route/rule`, which would drag the whole sing-box runtime (sing-tun, gvisor, tfo-go, quic-go) into a panel that otherwise depends on `option` + `constant` alone; each rule carries its upstream line reference. Verdicts are **tri-state** — "could not read geoip-cn" and "not in geoip-cn" route differently, and a two-state matcher has to guess
- `app/pkg/routeprobe/` - The route simulator behind `POST /route/probe`. Walks `route.rules` the way `route/route.go` does and reports where a destination *would* go, before any traffic exists
- `app/pkg/process/` - Process discovery and signaling helpers (pgrep/SIGTERM/SIGHUP)
- `app/pkg/database/` - SQLite database management, migrations, and JSON import
- `app/pkg/subscription/` - Subscription CRUD + cron AutoUpdater (database-backed)
- `app/pkg/noderules/` - Outbound Node Rules: Filters (keyword/tag-matched node buckets) + Groups (sets of Filters). Pure matcher + XORM manager; `BuildSpecs`/`config.BuildGroupOutbounds` materialize selector/urltest groups on each subscription update. The matcher runs over a `NodePool` with two halves: `Endpoints` (real exit nodes — one matching nothing falls through to the fallback Filter) and `OptIn` (`config.OptInTags`, today the `direct` outbounds). An opt-in tag joins a Filter only when a matcher names it and NEVER reaches the fallback — a urltest that quietly acquired `direct` elects it on every probe (zero latency wins) and silently routes everything unproxied. `POST /node-rules/preview` returns them as `optional` so the UI's node pickers can offer them. **A `code` matcher sees only the display name** (`displayName` in `tagparts.go`), never the whole tag: a tag ends in the node's endpoint, so matching all of it let the SERVER's hostname vote — every node on `s4.usghq.ps1ksydn.com` matched US through `usghq`, so Hong Kong and Taiwan joined a US-only filter and its urltest then sent that traffic out through Hong Kong. ASCII synonyms additionally require word boundaries, because a two-letter code is a substring of ordinary words (`us` in Belarus/Cyprus, `in` in Singapore/link, `de` in Sweden); CJK synonyms and emoji flags have no boundaries and stay plain substrings. `keyword`/`emoji` matchers still see the FULL tag — operators paste whole tags as excludes to pin one node, and widening those would silently re-admit the node they were written to remove
- `app/pkg/initstate/` - Initialization state management (database-backed)
- `app/pkg/sublink/` - Node parsing and subscription fetching. The canonical feed is a base64 list of proxy URIs, but two other shapes are common enough that rejecting them loses a working subscription, so `foreign.go` handles them **after** base64 decoding fails (a well-formed feed never reaches it): a Clash/Mihomo YAML profile, and a plain-text URI list. Fetching also carries a **User-Agent fallback**: the request goes out under a neutral `sbe-fetcher/1.0` because many panels content-negotiate a client-native config on a known client name — but some gate the other way and answer 404 to everything not on a client whitelist, so a 4xx (and only a 4xx — a 5xx is the panel failing at a request it accepted) is retried once as `v2rayN/6.23`
- `app/pkg/sublink/clash/` - Imports the `proxies:` list of a Clash/Mihomo profile as nodes. Everything else in the profile (rules, proxy-groups, DNS) is ignored on purpose — sing-box-easy owns routing itself. Proxies are kept as raw maps rather than typed structs: the key set differs per proxy type and providers add vendor keys freely, so the accessors in `value.go` do the narrowing. A type sing-box has no outbound for is returned in a `Skipped` list and logged, not swallowed — a subscription that silently imports 40 of 50 nodes reads as a panel bug. TLS is forced on for the TLS-framed protocols (trojan/hysteria2/anytls), where Clash omits `tls: true` because there is nothing to toggle and a missing block would dial plaintext. Two decode details matter: a list entry that is not a mapping makes yaml.v3 return a `*yaml.TypeError` **after** decoding every sibling it could, so that error is tolerated (the bad rows become `Skipped`) while a real syntax error stays fatal; and tags are made unique here, because an outbound tag is a primary key to sing-box — a repeated provider name fails `sing-box check` and rolls back the whole config update, naming the tag but neither of the feed entries that collided
- `app/pkg/installer/` - sing-box and dashboard installation (task-based). `buildInstallCommand` is platform-aware: official install script (curl) on Debian/other Linux, `opkg install sing-box` on OpenWrt (pinned versions download the static release tarball with BusyBox wget)
- `app/pkg/appupdate/` - sing-box-easy self-update: GitHub release listing (cached), tarball download + sha256 verification, atomic binary/`dist` swap with rollback, then `execve` restart. Binary-only packages (embedded frontend) skip the dist swap. On OpenWrt ipk installs (detected via `/usr/lib/opkg/status`) tarball self-update is refused — updates go through opkg instead. Running version comes from the `-X ...appupdate.Version` ldflag stamped by `.github/workflows/release.yml`, falling back to a `.sing-box-easy-version` file written on each update. GitHub calls are authenticated with the token issued by `githubauth` sign-in (or the `GITHUB_TOKEN` env var) — 60 req/h anonymous vs 5000 req/h authenticated — and use ETag/`If-None-Match` conditional requests, which return 304 and cost no rate-limit quota
- `app/pkg/logger/` - Zap logger setup + Hertz logger adapter

**Protocol Parsers** (`app/pkg/sublink/protocol/`):
- Factory pattern for creating protocol-specific parsers
- Currently supports: Shadowsocks (ss://), VMess (vmess://), Trojan (trojan://), VLESS (vless://), Hysteria2 (hysteria2://), AnyTLS (anytls://)
- Each parser implements `node.SubNodeParser` interface
- `password@host:port` style links (hysteria2, anytls) share `parseUserinfoURI` in `helpers.go`. The order it enforces is the point: the path is stripped from the **authority**, after the `@` split, never from the whole link — these passwords are often raw base64, and a `/` inside one used to truncate away the `@host:port` that followed, so the node silently vanished from an otherwise valid subscription
- New protocols: Add parser file + register in `ppMap` in `protocol.go`

**HTTP Layer**:
- Framework: CloudWeGo Hertz (high-performance HTTP framework)
- Routes: `app/routes/v1_12_12/` - All API handlers for v1.12.12
- API prefix: `/api/1.12.12/`
- Route registration: `routes.go` in v1_12_12 package

### Configuration Safety Mechanism

All config modifications follow a safe workflow (implemented in `config.Manager`):
1. Write to a staging file `.config_new.tmp` (same directory as `config.json`, so
   the final `os.Rename` is atomic on one filesystem). The name intentionally
   does NOT end in `.json` so a sing-box running in directory mode
   (`sing-box -C <dir>`) never merges it and aborts on duplicate tags.
2. Validate using `sing-box check` command
3. On success:
   - Snapshot the current `config.json` into the DB-backed version history
     (see "Config version history" — there is no more `config.old.json` file)
   - Atomically rename the staging file over `config.json`
4. On failure: Keep original config, remove the staging file
5. Rollback available via `/api/1.12.12/config/rollback` (or to a specific
   version via `/api/1.12.12/config/versions/:id/rollback`)

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

### Subscription outbound tags

A node imported from a subscription is tagged `<name> <fingerprint> | <subID>`:
the provider's display name, 8 hex characters of md5 over `server:port`, and the
owning subscription's ID. Only the middle part is machine-made — it exists to
keep tags unique when a provider ships two nodes with the same name, which
sing-box requires (a duplicate tag fails `sing-box check`).

The endpoint used to be spelled out (`… s4.usghq.ps1ksydn.com:37219 | sub_x`).
Hashing it shortens tags by ~40% on a real feed (53 → 31 characters average) and
removes a hostname that a `code` matcher was reading as evidence of the node's
country. The cost is stated rather than hidden: you can no longer read a node's
endpoint out of `config.json`, so identifying one means hashing the candidate
endpoint (`config.FingerprintEndpointKey`) and grepping for it. md5 is an
identity function here, not a security primitive.

Two compatibility paths keep the format change from being destructive:

- `fingerprintLegacyTag` (auto_updater.go) converts an old tag and re-matches
  it, so a refresh migrates the format as **renames**, not delete-plus-add. The
  difference is not cosmetic: a delete drops the node from every selector and
  urltest that references it (`renameMap` in `applyChanges` carries a rename
  across, a deletion does not). Measured on a real config: 34 updated, 0 added,
  0 deleted, and every group kept its members.
- `legacyTagValue` (noderules/tagparts.go) converts a saved matcher/exclude
  value that IS a full old-format tag. An exclude that matches nothing fails
  silently — it re-admits exactly the node an operator removed on purpose.

Every path that mints a node tag uses this shape — the subscription updater, the
batch add behind `POST /outbounds/batch`, and the single add behind
`POST /outbounds` — via `config.OutboundTagCandidates`, whose first element is
the tag to write and whose rest are the shapes an older build produced. The
"already exists" check on both add paths tests every candidate, so re-pasting
links that were added before the change is still a skip rather than a second
copy under a new tag. A bare display name is deliberately not a candidate: two
providers legitimately ship a "香港 01".

Route rules that name a node tag directly are NOT rewritten; they normally name
a group, which is stable.

### Subscription official link

A subscription carries `official_url` — the provider's own site, where an
operator tops up or renews. It is auto-filled on refresh from whichever source
the feed offers (`app/pkg/subscription/official_url.go`): the
`profile-web-page-url` response header first, then an info entry whose label
says so ("官网：https://…"). `siteLabelKeywords` is deliberately narrower than
`DefaultInfoLabelKeywords` — a 客服 (support) entry is metadata and often holds
a URL, but it is not where "top up" should go.

Two rules matter:

- **Fill only while empty** (`officialURLToPersist`). Providers move domains and
  mirrors keep reporting the old one, so an operator who corrects the link by
  hand must be able to trust that the next refresh leaves it alone.
- **http(s) only, enforced twice.** The value is third-party text that ends up
  in an `href`, so `javascript:`/`data:` must never reach the DOM.
  `NormalizeOfficialURL` guards the write (handler + refresh) and
  `frontend/src/utils/safeExternalUrl.ts` guards the render — not redundancy: a
  row written before the rule existed, or by any other API client, still reaches
  the render path. Both promote a bare domain to https, and the frontend adds
  `rel="noopener noreferrer"` since the target page is provider-chosen.

The name in `Subscriptions.vue` and `SubscriptionsOverviewCard.vue` becomes that
link when one is known, and stays plain text otherwise.

### Subscription info nodes

Account metadata (traffic left, expiry, reset countdown) arrives through three
provider-dependent channels, handled in `app/pkg/subscription/info_node.go`:

1. the standard `Subscription-Userinfo` response header (`parseUserinfo`);
2. "info nodes" pointed at a loopback address (`isLoopbackNode`);
3. "info nodes" that are ordinary, fully-routable entries — usually clones of the
   first real node — distinguished only by their display name (`isInfoLabel`),
   e.g. 良心云's `剩余流量：4.7 TB`.

Case 3 requires **both** a `<label><colon><value>` shape (fullwidth `：` or ASCII
`:`) and a keyword inside the label, so region names like `🇭🇰香港：01` are never
swallowed. The keyword list is operator-editable (Subscriptions page → "Info
Keywords" → `PUT /settings/subscription-info-keywords`, stored as a JSON array in
`settings` under `subscription_info_keywords`); an empty override falls back to
`subscription.DefaultInfoLabelKeywords`. The updater reads it per refresh via the
`InfoKeywordsProvider` interface, so edits apply without a restart. Info nodes are
kept out of `config.json`: they are not usable exits and their names change every
refresh, which would otherwise churn the outbound list.

### Frontend Architecture

- **Framework**: Vue 3 + TypeScript + Vite
- **Routing**: Vue Router with `/init` wizard and `/dashboard` main app. A global `beforeEach` guard hits `/init/status` and redirects to `/init` until initialization is marked complete.
- **State**: Pinia stores in `src/stores/` (currently `dns`, `outbounds`, `route`)
- **i18n**: `vue-i18n` setup in `src/i18n/`; reusable logic in `src/composables/`; app-wide plugins registered in `src/plugins/`; PrimeVue "Volt" theme components in `src/volt/`
- **Styling**: Tailwind CSS v4 + DaisyUI utility classes; PrimeVue + HeadlessUI for components; Heroicons for icons
- **API Client**: Axios wrapper in `src/services/api.ts` (`baseURL: /api/1.12.12`), with one service module per domain (`config.ts`, `dns.ts`, `outbound.ts`, …)
- **Editor**: Monaco (via `monaco-editor-vue3`) — kept in its own chunk by `vite.config.ts`
- **Dev proxy**: `vite.config.ts` proxies `/api/*` to `http://localhost:5100` — match this with `server.port` in `bin/app.yml` when changing dev ports
- **Components**: Reusable UI components in `src/components/`
- **Navigation shell**: two layouts share one `menuItems` tree — `Sidebar.vue` everywhere, `Topbar.vue` on OpenWrt (LuCI already owns the left edge of the screen there). `Dashboard.vue` picks between them via `useDeployment`'s `isOpenWrt`, which is resolved by the router guard before the view mounts so the correct layout renders on first paint. Shared chrome (version + update badge, live service dot, signed-in user, logout) lives in `useNavChrome`
- **Views**:
  - `views/InitWizard.vue` - Multi-step initialization (steps in `views/init-steps/`)
  - `views/Dashboard.vue` - Main dashboard with nested routes
  - `views/dashboard/` - Feature-specific views (Config, DNS, Inbounds, Outbounds, Route, Experimental, Subscriptions, Log, Overview)

## Key Development Patterns

### Schema-driven forms (generated field inventories)

The inbound, DNS server and outbound dialogs are schema-driven, not hand-written
per type. Four layers:

1. **`cmd/gen-option-schema`** reflects over the `option.*` structs that
   `app/pkg/config/registry.go` constructs and emits one inventory per domain —
   `frontend/src/schemas/{inbound,dnsServer,outbound}Inventory.generated.ts`.
   Deprecation is recovered by parsing sing-box's `// Deprecated:` doc comments
   with `go/ast`, since `reflect` cannot see comments. Regenerate with
   `go generate ./app/pkg/config/`. Adding the next domain (outbounds,
   endpoints, services) is a row in `domains()` plus a curation file.
2. **`frontend/src/schemas/optionSchema.ts`** holds everything domain-agnostic:
   the tier model, visibility, pruning, defaults. Domains bind it with
   `createSchema`.
3. **`inboundFields.ts` / `dnsServerFields.ts` / `outboundFields.ts`** are pure curation — which
   fields to promote, what control to render, what to call them. Typed against
   the generated keys, so naming a field sing-box does not have is a `vue-tsc`
   error rather than an input that never binds.
4. **`SchemaFieldsEditor.vue`** renders it for all three domains, reusing
   `useOptionalFields` (the composable behind the route- and DNS-rule condition
   forms).

Things that are easy to get wrong:

- Tiers are **core / typical / advanced**, not required/optional. sing-box marks
  almost nothing Required — for `mixed` only `listen` — so tiering follows what
  each doc page's *example* shows. Genuine required-ness lives in
  `utils/inboundRequiredFields.ts` and `routes/v1_12_12/dns_validation.go`, and
  runs on save.
- Anything **uncurated resolves to `advanced`** rather than being dropped, so a
  field added by a future sing-box is reachable immediately. Field labels fall
  back to a humanized JSON key, so the locale files only need entries where a
  human label beats the field's own name.
- **The inventory describes the PINNED library, not the installed binary.** The
  repo pins sing-box 1.12.12; a host may run 1.13.x. Drift in the *additive*
  direction is harmless (a knob the form does not offer), but a field or type a
  newer sing-box REMOVED is rejected outright — `sniff` fails config decode on
  1.13, and it used to sit in the form's own "deprecated" row. The generator
  records `since`/`removed` from sing-box's `experimental/deprecated` constants
  and the UI gates against `/system/info` (`useSingBoxVersion` + `isRetired`).
  Mapping a deprecation entry onto fields is hand-maintained in
  `cmd/gen-option-schema/versions.go`, built from the `deprecated.Report(...)`
  call sites — recheck those on a dependency bump.
- **A full per-version inventory is deliberately NOT built.** Go cannot import
  two versions of one module, so it would need the generator run once per
  version in a throwaway module. Measured 1.12->1.13 drift is ~10 added fields
  and one removed, so the gate above covers the dangerous direction at a
  fraction of the cost. Revisit if the supported version range widens.
- **HTTP/3 DNS is `h3`, never `http3`.** sing-box's constant is
  `C.DNSTypeHTTP3 = "h3"` and its transport registers under that name. The
  registry spelled it `http3`, which broke both directions: a valid config could
  not be parsed, and a config the panel saved could not start.
  `TestDNSTypeHTTP3IsH3` pins it.
- **DNS server handlers must not use `c.Bind`.** `option.DNSServerOptions`
  defines only `UnmarshalJSONContext`; every real field lives behind an
  `Options any` that a reflective bind leaves nil, so the config then fails to
  marshal. Use `bindDNSServer` (`dns_validation.go`). The DNS *rule* handlers
  already carry the same fix.

- **Outbound groups.** `selector`/`urltest` carry an `outbounds` list of other
  tags, edited by the `outbound-list` control (self excluded — a group listing
  itself hangs at start). A selector's `default` uses `outbound-member`, a
  picker over that group's *current* members: sing-box fails at START with
  "default outbound not found" otherwise, which `sing-box check` does not catch.
- **Some outbounds are not owned by the form.** `noderules` ->
  `BuildGroupOutbounds` rebuilds any outbound whose tag matches a Filter or
  Group name, discarding form edits on the next apply. `GET /outbounds` returns
  `managed_tags` (from `config.ManagedOutboundTags`) and the dialog warns.

Adding a type means: add it to `config.InboundTypes` / `DNSTypes` /
`OutboundTypes`, add a case to the matching `Create*Options`, regenerate, then
optionally curate. The `Test*TypesAreRegistered` tests fail if a list and the
registry disagree. A type whose option struct legitimately has no fields
(`block`/`dns` outbounds are `option.StubOptions`) must be listed in the
domain's `FieldlessTypes`, so a silently-empty struct still fails loudly.

Schema logic is tested with `bun test` (`frontend/src/schemas/*.test.ts`).

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
- `/dns`, `/dns/servers`, `/dns/hosts`, `/dns/rules`. `PUT /dns/rules` (collection, no index) **reorders**, same contract as `PUT /route/rules` below — an `{"order": [...]}` permutation of the current indices. Sending indices rather than rules matters twice as much here: a DNS rule is polymorphic, so re-uploading one would mean the client reproducing sing-box's own decode
- `/dns/probe` (POST) — resolve a domain the way sing-box does and explain the routing. Backed by `app/pkg/dnsprobe/`, which keeps three sources separate: the **live answer** comes from sing-box's own Clash API `/dns/query` (so hosts, `predefined` actions and FakeIP all apply — no `dig`, which BusyBox lacks); **attribution** is reconstructed offline and flagged `exact: false` whenever a `rule_set` or other runtime-only condition sits ahead of the decision; **logged_matches** is sing-box's own `dns: match[N]` debug line, which is authoritative. Note `N` is `2*index+1`, not the rule index (`dns/router.go`) — verified against sing-box 1.13.11. The log window is derived by diffing snapshots because only the systemd backend honours `TailLogs`' cursor
- `/dns/probe/stream` (POST, **SSE**) — the same probe, reported phase by phase via `dnsprobe.RunStaged`. The phases are where the time actually is: a live query over the Clash API, a fixed 250ms `logSettleDelay`, then (with `compare_servers`) one query to every configured resolver. Streamed, the rule ladder is on screen before the live query returns. **What is deliberately NOT streamed is the per-rule verdict** — `dnsprobe.Attribute` walks every rule in one synchronous pass, so pacing the rungs server-side would mean sleeping between them, and a tool whose job is explaining timing must not fake its own. The client paces them and labels it as a replay (`useRuleSequencer`)
- `/route/probe` (POST) — **pre-flight** routing: where a destination *would* go, before a connection exists. The complement to the Clash API's `/connections`, which only reports connections that already happened — the wrong half of the loop when the question is "did my config edit do what I meant, or should I roll it back?". Backed by `app/pkg/routeprobe/` + `app/pkg/ruleset/`. Three things it gets right that a naive reading of the rule struct does not: the walk is a **state machine** (only route/reject/hijack-dns terminate; `sniff` and `resolve` match and then change what the rules below them see — and `direct` does NOT terminate, which looks like an upstream oversight but is mirrored, since a diagnostic that silently corrects the engine lies about the config in force); **group semantics** follow `route/rule/rule_default.go` — `rule_set` is an AND item whose own tags are OR'd, while `ip_is_private` and `geosite` join the destination-address group and are therefore OR'd with the domain matchers; and all three fall-through cases are reported separately (`route.final`, first outbound, implicit direct), because collapsing them into "final" sends people hunting for a key that is not in their config. Verdicts stay **tri-state** to the top: a rule that could not be decided clears `exact` and is counted, so a prediction below an undecidable rule is presented as the guess it is. The handler resolves names through sing-box's own DNS router and reads the live `clash_mode`, both optional — on the production config that turns 3 undecidable rules into 1, and naming the sniffed `protocol` turns the last one into an exact answer
- `/inbounds`
- `/route/rules`, `/route/rule-sets`, `/route/final`. `PUT /route/rules` (collection, no index) **reorders**: the body is `{"order": [...]}`, a strict permutation of the current indices, and the rule bodies never leave the server. It sits on the collection path because `/route/rules/:index` already owns the next path segment and a static sibling there collides in the Hertz router. Both endpoints are driven by `frontend/src/composables/useDragReorder.ts` — the reorder mode on the Routing Rules card and the DNS Rules table (drag handle, or arrow keys on a focused handle). Reorder mode is a **batch edit**: the list sorts live during each drag, but nothing is written until Save, which sends ONE composed permutation (Cancel restores the list as it was on entry). Rows must be keyed by `reorder.keyAt(i)`, not by array index, or the FLIP animation (`transition` prop on `<List>` / `<Table>`) has nothing to animate
- `/log`
- `/experimental/{clash-api,cache-file,v2ray-api}`
- `/service/{status,start,stop,restart}`
- `/service/logs` (GET) — the backlog window, and the fallback feed. `/service/logs/stream` (GET, **SSE**) — the same log, pushed. The unary call is seeded first and returns a journald cursor the stream resumes from, so the handover drops nothing and repeats nothing — the one case polling could never get right, since only the systemd backend implements the cursor and the others re-served the same window every tick. Backed by `Backend.FollowLogs` (`app/pkg/service/follow.go`): `journalctl -f` on systemd, `logread -f` on procd, and a seek-based poll for a plain `log.output` file. **The two exec-based followers hold a child process open for the life of the connection**, so the handler cancels its context on every return path and `startCommand` kills the child in a `defer` — an ignored write error is one orphaned `journalctl` per closed browser tab. A backend with no log source returns `(nil, nil)`, not an error, and the client falls back to polling
- `/subscriptions` + `/subscriptions/:id/update`. The body carries `official_url` (see "Subscription official link"); a value that is not an http(s) link is rejected with `CodeBadRequest` rather than stored, because it is served back to every client of this API and rendered as a link
- `/node-rules` (full ruleset), `/node-rules/{apply,preview,keywords,templates}`, `/node-rules/templates/:id/apply`, `/node-rules/filters[/:id]`, `/node-rules/groups[/:id]` — Outbound Node Rules CRUD + dry-run/apply
- `/scheduler/{status,start,stop,trigger,jobs}` — cron auto-updater control
- `/install`, `/install/task/:task_id`, `/install/status`, `/update`
- `/dashboard/{download,upload}`, `/dashboard/task/:task_id`, `/dashboard/status`
- `/version`, `/version/releases`, `/version/task/:task_id` (read), `/version/update` (admin-only) — app self-update from GitHub releases. `GET /version` includes a `self_update` object (`method`: `tarball`/`opkg`, `automatic`, `architecture`, `feed_provides`, `feed_known`) so the UI offers an action that can actually work
- `/version/prepare-package` (admin-only) — for opkg-managed installs: downloads and sha256-verifies the arch-matched `.ipk` into `/tmp` and returns the install command, **without installing**. Reuses the update Task machinery (poll `/version/task/:id`); the finished task carries a `plan`. The panel deliberately stops short of running opkg because our ipk's `prerm` stops `sing-box-easy` itself — an install driven from inside the process would kill the process group mid-transaction
- `/init/{status,complete,reset}`
- `/templates/rule-sets`
- `/settings` (GET/PUT) — application settings (`config_versions_keep`). Keys listed in `settings.SecretKeys` (the GitHub token) are stripped from GET responses and are **not writable here** — the credential is only ever issued by device-flow sign-in
- `/settings/subscription-info-keywords` (GET/PUT) — the labels that mark a feed entry as account metadata instead of a proxy node (see "Subscription info nodes"). PUT with `{"keywords": []}` clears the override and restores `subscription.DefaultInfoLabelKeywords`. Lives under `/settings` because `/subscriptions` already owns a `:id` wildcard at that path position, which a static sibling would collide with in the Hertz router
- `/github/auth/status` (read), `/github/auth/device` POST/GET/DELETE `:session_id`, `/github/auth` DELETE (admin-only) — GitHub sign-in via OAuth device flow
- `/auth/status` (public) — whether this deployment requires login, plus `system_type` (`openwrt`/`debian`/`unknown`). `server.auth` in app.yml: `auto` (default; login disabled on OpenWrt, enabled elsewhere) / `enabled` / `disabled`. When disabled, `AuthMiddleware` injects a synthetic admin user and the frontend hides login/profile/user management. Deliberately the only host detail exposed pre-auth — the frontend needs it to pick a navigation layout
- `/system/logs` (GET) + `/system/logs/stream` (GET, **SSE**) — the panel's OWN log, the second tab on the Logs page. Same wire shape as the `/service/logs` pair so the viewer swaps a URL instead of branching. Backed by `app/pkg/applog`, an in-process ring buffer filled through a registered zap sink (`applog://ring` in `logger.Init`'s OutputPaths) — **not** journald/logread, because the panel writes only to stdout and where that lands depends on the packaging (journald under an unknown unit name, syslog on procd, or an unrecoverable terminal on a manual install). A ring works identically everywhere and spawns nothing, so unlike the sing-box follower it cannot orphan a child. The cost is stated in the UI rather than hidden: the buffer is the life of the process, so a restart — including a self-update's execve — starts it empty, and a panel that crashed cannot serve the log explaining why. `Ring.Append` sends to subscribers **under the same exclusive lock** the teardown closes them under; releasing between snapshot and send let a closing subscriber be sent to, which panics
- `/system/info` — host details for the Settings "About" card: arch, CPU cores, kernel, distribution, hostname, service backend, the sing-box + sing-box-easy versions, and `disks` (free space on the filesystems the panel writes to). Collected by `app/pkg/sysinfo/`, which degrades every field to a zero value rather than failing. `disks` exists because a full filesystem is otherwise invisible: SQLite cannot create its rollback journal and reports `SQLITE_CANTOPEN` ("unable to open database file (14)"), which names the database and never the disk — reads keep working, so it reads as a permissions bug on the `.db` file. `sysinfo.CollectDisks` statfs's each path and resolves the mount point/device from `/proc/self/mounts` (longest-prefix match, octal-unescaped), collapsing paths that share a filesystem. `database.AnnotateWriteError` wraps the same diagnosis onto write failures, but only after statfs confirms it, so a genuine permission error is never mislabelled. This matters most on OpenWrt, where the root overlay is often a small loop image while terabytes sit mounted elsewhere

## Testing Notes

- Protocol parsers have test files (e.g., `trojan_test.go`); only Shadowsocks/VMess/Trojan parsing is covered today.
- Use `go test -v` for verbose output.
- Test node link formats inside the parser tests serve as documentation for accepted URI shapes.
- Frontend logic is tested with `bun test` (`frontend/src/**/*.test.ts`) — the schema curation files, and `types/ruleLadder.test.ts`. Components themselves are untested; add Vue Test Utils if that changes.

## Response Envelope Reference

Defined in `app/routes/v1_12_12/handler.go`. All new v1.12.12 endpoints must use these helpers; HTTP status is always 200 and clients branch on `code`.

**Streaming endpoints are the one exception to "one envelope per response"** — and only to the *shape*, not to the rule. An SSE response cannot carry a single envelope, so **each event carries its own**: `data: {"code":0,"data":{…},"msg":""}`. Clients branch on `code` exactly as they do for a unary call, which is what makes a mid-stream failure reportable instead of an unexplained disconnect. Framing lives in `app/routes/v1_12_12/sse.go`; note `formatSSEFrame` emits one `data:` line per line of payload, because a raw newline inside the JSON (sing-box logs multi-line messages) would otherwise end the frame early and the client would silently drop the event.

Clients read these with `fetch()` + `ReadableStream` (`frontend/src/services/stream.ts`), **never `EventSource`**: it cannot set headers, and auth here is `Authorization: Bearer` from localStorage. Passing the token as a query parameter instead would write a live session credential into every access log and `Referer` on the path.

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
