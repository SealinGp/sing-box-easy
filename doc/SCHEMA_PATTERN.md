# Schema-Driven Option Forms

How the Inbound, DNS Server and Outbound dialogs know what fields exist, which
ones matter, and which ones the installed sing-box will actually accept.

> **Status**: implemented for 3 of 5 option domains. `endpoint` and `service`
> remain. The multi-version plan in §6 is designed but **not built**.

---

## 1. The problem

sing-box options are polymorphic: a `type` string selects a struct, and the rest
of the object is that struct's fields. Nothing in JSON says which fields belong
to which type, so a UI has to know — and the only way to know is to copy the
information out of sing-box.

Every copy drifted:

| Copy | What went wrong |
|---|---|
| `v-if` blocks in each modal | Inbounds covered 3 of 16 types. A `trojan` inbound got no `users` field, so one created through the panel could authenticate nobody. |
| `frontend/src/types/inbound.ts` | 424 hand-transcribed lines. Missing `anytls`, which the backend had registered since 1.12. |
| Hardcoded type dropdowns | Outbounds offered 17 while the registry built 20. `anytls`, `dns`, `shadowsocksr` were unreachable. |
| Capability predicates | `needsServerAddress` and `supportsDetour` in `DNSServers.vue` were byte-identical lists under different names. Group classification existed in four places. |
| `registry.go` itself | Spelled HTTP/3 DNS `"http3"`; sing-box's constant is `"h3"`. A valid config could not be parsed, and a saved one could not start. |

The authoritative list already exists in this repo — `app/pkg/config/registry.go`
maps each type to a real `option.*Options` struct from the pinned sing-box. The
pattern below reads those structs instead of copying them.

---

## 2. The four layers

```
sing-box option structs  (the dependency, pinned in go.mod)
        │  reflect + go/ast
        ▼
cmd/gen-option-schema                          ← Go, run by `go generate`
        │  emits
        ▼
schemas/*Inventory.generated.ts                ← WHICH fields exist, what shape
        │  typed against
        ▼
schemas/{inbound,dnsServer,outbound}Fields.ts  ← WHICH fields matter (editorial)
        │  bound by createSchema()
        ▼
SchemaFieldsEditor.vue + SchemaFieldControl.vue ← renders, domain-agnostic
```

### 2.1 Generator — `cmd/gen-option-schema`

Reflects over the structs `registry.go` constructs and emits one inventory per
domain. Driven by a table:

```go
func domains(registry *config.Registry) []domain {
    return []domain{
        {Name: "Inbound",   Types: config.InboundTypes,  Create: registry.CreateInboundOptions,  ...},
        {Name: "Outbound",  Types: config.OutboundTypes, Create: registry.CreateOutboundOptions, ...},
        {Name: "DNSServer", Types: config.DNSTypes,      Create: registry.CreateDNSOptions,      ...},
    }
}
```

It handles:

- **Embedded structs** — flattened the way `encoding/json` does, so
  `ListenOptions` / `DialerOptions` contribute their fields to every type that
  embeds them. Name collisions resolve shallowest-wins, matching the encoder.
- **Wrapper types** — `badoption.Duration` is an `int64` that serializes as
  `"5m"`; `badoption.Addr` is a struct that must render as one input. Named
  types are checked before `reflect.Kind`. Both the `badoption` wrappers **and**
  the stdlib types they wrap appear in the structs — `ListenOptions` uses
  `*badoption.Addr` while `TunInboundOptions` uses a bare `netip.Prefix`.
- **Deprecation** — from three sources, see §4.
- **Field-less types** — `block` and `dns` outbounds are `option.StubOptions`,
  i.e. `struct{}`. A domain lists these in `FieldlessTypes` so the
  "reflected to zero fields" guard (which catches a broken registry) does not
  fire, while a silently-empty struct still fails loudly.

Output is alphabetical per type, so regenerating after an unrelated dependency
change produces a diff containing only what actually changed.

### 2.2 Inventory — `schemas/*Inventory.generated.ts`

Data only. Says nothing about presentation.

```ts
export const OUTBOUND_INVENTORY = {
  shadowsocks: {
    method:   { kind: 'string' },
    server:   { kind: 'string' },
    // deprecated, with the versions sing-box itself records
    domain_strategy: { kind: 'string', deprecated: true, since: '1.11.0', removed: '1.14.0' },
  },
} as const satisfies Record<string, Record<string, OptionFieldInfo>>

export type OutboundFieldKey<T extends OutboundTypeName> =
  Extract<keyof (typeof OUTBOUND_INVENTORY)[T], string>
```

That last type is the point. Curation is keyed by it, so naming a field
sing-box does not have is a `vue-tsc` error rather than an input that never
binds:

```
error TS2820: Type '"set_system_proxxy"' is not assignable to type
              'InboundFieldKey<"mixed">'. Did you mean '"set_system_proxy"'?
```

### 2.3 Curation — `schemas/*Fields.ts`

The only hand-maintained layer. Tiers, controls, labels, ordering, defaults.

```ts
const BY_TYPE: { [T in OutboundTypeName]: Partial<Record<OutboundFieldKey<T>, FieldCuration>> } = {
  selector: {
    outbounds: { tier: 'core',    order: 10, control: 'outbound-list' },
    default:   { tier: 'typical', order: 20, control: 'outbound-member' },
  },
}
```

Shared curation is **intersected** with each type's real inventory, never spread
into it — a spread bypasses TypeScript's excess-property check, so `server`
could be spread into `block` and fail silently.

### 2.4 Renderer

`SchemaFieldsEditor.vue` takes an already-resolved field list and never learns
which domain it is drawing. Visibility comes from `useOptionalFields`, the same
composable behind the route- and DNS-rule condition forms.

---

## 3. The tier model

**core / typical / advanced — not required / optional.** This is the design
decision most likely to be "corrected" by someone who has not hit the problem.

sing-box marks almost nothing required. For a `mixed` inbound the docs mark
exactly one field Required (`listen`); `users` is explicitly *"No authentication
required if empty"* and `set_system_proxy` is optional. A form built on
required-ness would render a single address box and hide the two fields anyone
opens the dialog to set.

What each doc page *shows* is the type's **characteristic** fields, which is a
different question from what sing-box will reject.

| Tier | Behaviour |
|---|---|
| `core` | Always rendered, no remove control |
| `typical` | Rendered by default, removable while empty — what the doc page's example shows |
| `advanced` | Behind the "add field" row, or auto-shown when a loaded config uses it |

Genuine required-ness is enforced on save, per domain, in
`utils/inboundRequiredFields.ts`, `dns_validation.go`, and `validateOutbound`.

**Anything uncurated resolves to `advanced`** rather than being dropped. That is
what makes curation safe to leave incomplete: a field added by a future sing-box
is reachable immediately, just not promoted. Labels fall back to a humanized
JSON key (`tcp_fast_open` → "TCP Fast Open"), so the locale files only need
entries where a human label beats the field's own name.

---

## 4. Deprecation and the version gate

### 4.1 Three sources, because none is complete

| Source | Covers | Why needed |
|---|---|---|
| `// Deprecated:` doc comments, read with `go/ast` | Most retired fields | `reflect` discards comments. A hand-written list got 3 of 8 tun fields wrong — it missed `inet4_route_exclude_address`, `inet6_route_exclude_address` and `endpoint_independent_nat`. |
| `experimental/deprecated/constants.go`, parsed | Fields **and whole types**, with versions | The only source that says *which version removes it*. |
| `domain.DocDeprecated` in the generator | Docs-only retirements | The sniff family moved to route rules in 1.12 while `option.InboundOptions` still declares it plainly. |

The table's entries are identifiers (`"wireguard-outbound"`), not field paths.
Mapping them onto the inventory is `cmd/gen-option-schema/versions.go`, and that
table was built by finding each `deprecated.Report(...)` **call site** — the
names mislead. `"special-outbounds"` reads like it covers `block` and `dns`, but
`option/outbound.go` reports it for `dns` only, and 1.13.11 still accepts a
`block` outbound. Recheck on a dependency bump:

```bash
grep -rn 'deprecated\.Option' "$(go env GOMODCACHE)/github.com/sagernet/sing-box@<ver>"
```

### 4.2 Why a gate exists at all

**The inventory describes the pinned library. The operator runs a different
binary.** This repo pins sing-box 1.12.12; the machine it was developed on runs
1.13.11.

Drift in the *additive* direction is harmless — a field a newer sing-box added
is a knob the form does not offer. The other direction is not. Probed against a
real 1.13.11 with `sing-box check`:

```
sniff (+ the legacy inbound family)   → config decode fails
direct outbound override_address/port → config decode fails
tun inet4_route_address               → inbound init fails
dns outbound                          → config decode fails
wireguard outbound                    → outbound init fails
inbound detour                        → ACCEPTED  (survived into 1.13)
```

The sniff family sat in the inbound form's own amber "deprecated" row, one click
each. `sing-box check` runs before any write, so nothing was ever at risk of
corruption — the cost was a save that could not succeed and an opaque upstream
error for something the UI had just suggested.

Note the last two rows: `wireguard` and `shadowsocksr` outbounds **parse fine
and fail at init**. `sing-box check` passing proves nothing about them.

### 4.3 What the gate does

`useSingBoxVersion` reads `/system/info`; `isRetired` / `isDeprecatedIn` compare.

| Condition | Behaviour |
|---|---|
| `removed` ≤ installed | Withheld from the "add" row, with a line saying how many and why |
| `removed` ≤ installed, but **present in the loaded config** | Still rendered, hard-flagged — hiding it would mean the operator cannot see or clear the thing breaking their config |
| `since` ≤ installed | Shown, flagged with the version |
| version undetected | Everything shown — i.e. the pre-gate behaviour |

An undetected version deliberately hides nothing. Guessing "probably removed"
would blank fields for someone whose binary we merely failed to detect, which is
worse than showing one too many.

`compareVersions` strips any prerelease suffix before comparing. Without that,
`1.13.0-beta.1` split into four segments and sorted **above** `1.13.0` while
`1.13.0-rc` sorted equal to it — same shape, different answer. A prerelease of X
carries X's removals, so both must gate like X.

---

## 5. Recipes

### Add a type to an existing domain

1. Add a case to the matching `Create*Options` in `app/pkg/config/registry.go`.
   Use the `C.Type*` / `C.DNSType*` constants, never a string literal — that is
   how `"http3"` diverged from `"h3"`.
2. Add it to `config.InboundTypes` / `DNSTypes` / `OutboundTypes`.
3. `go generate ./app/pkg/config/`
4. Optionally curate in the domain's `*Fields.ts`. Uncurated is valid.

`Test*TypesAreRegistered` fails if the list and the registry disagree. The
reverse — a registry case nobody listed — is **not** detectable, because a Go
type switch cannot be enumerated at runtime. Adding a type means touching both.

### Add a domain (`endpoint`, `service`)

1. `app/pkg/config/<domain>_types.go` — exported list + `IsKnown*Type`, mirroring
   `outbound_types.go`.
2. A `Test*TypesAreRegistered` + round-trip test.
3. A row in `domains()` in `cmd/gen-option-schema/main.go`.
4. `schemas/<domain>Fields.ts` calling `createSchema`.
5. Point the view at `<SchemaFieldsEditor>`.

### Commands

```bash
go generate ./app/pkg/config/     # regenerate all inventories
go test ./app/pkg/config/         # registry/list consistency + round-trips
cd frontend && bun test           # schema decision rules
cd frontend && bun run build      # vue-tsc catches curation naming a dead field
```

---

## 6. Future: multi-version field support

**Not built.** Designed here so the next person does not have to re-derive it.

### 6.1 The goal

Today one inventory describes the pinned library, and §4 gates the dangerous
direction. The goal is for the form to show *the fields the installed sing-box
actually has* — including ones newer than the pin.

### 6.2 The constraint that shapes everything

**Go cannot import two versions of one module into a single binary.** So the
generator cannot simply loop over versions in-process. It has to be run once per
version, each in its own module context.

### 6.3 Proposed mechanics

```
for each supported version V:
    tmp=$(mktemp -d)
    cd $tmp && go mod init schemagen
    go get github.com/sagernet/sing-box@vV
    go run <generator> -version vV -out <repo>/frontend/src/schemas/vV/
```

The generator already isolates everything version-dependent behind
`moduleDir()` / `singBoxSubdir()`, so it needs a flag for the module version and
an output prefix — not a rewrite.

Complications to expect:

- **`registry.go` must compile against each version.** It is the list of types,
  and a type removed upstream breaks the build for that version. Either keep a
  per-version registry shim, or reflect over sing-box's *own* registry rather
  than ours.
- **Version matrix policy.** Which versions ship? Probably minors only
  (`1.11`, `1.12`, `1.13`), resolved by nearest-not-greater against the detected
  patch version.
- **Bundle size.** Three domains × N versions of ~500-line files. Almost all of
  it is identical between adjacent minors — the measured 1.12→1.13 drift is ~10
  added fields and 1 removed across all three domains. Ship a **baseline plus
  per-version deltas**, not N full inventories.
- **Unknown versions.** A binary newer than anything generated for should fall
  back to the newest inventory plus the existing gate, not to nothing.

### 6.4 Migration path

The current design is deliberately on this path rather than in its way:

| Today | Becomes |
|---|---|
| `INVENTORY` per domain | `INVENTORY[version]`, or baseline + deltas |
| `resolveFields(type)` | `resolveFields(type, version)` |
| `isRetired(note, version)` | Unchanged — still needed for versions with no generated inventory |
| Curation `*Fields.ts` | **Unchanged.** Keyed by field name, version-independent |

Curation is the expensive layer and it does not move. `createSchema` already
takes the inventory as a parameter, so the version axis threads through one
call site per domain.

### 6.5 Is it worth it?

Measured 1.12.12 → 1.13.12 across all three domains:

| Direction | Count | Consequence |
|---|---|---|
| Fields **added** in 1.13 | ~10 distinct | Form does not offer them — a missing knob |
| Fields **removed** in 1.13 | 1 (inbound `detour`, and it survived in practice) | Gated by §4 |
| Types removed at runtime | 2 (`wireguard`, `shadowsocksr` outbounds) | Gated by §4 |

The gate already covers the whole dangerous direction. Multi-version generation
buys ~10 optional dial knobs at the cost of a build matrix. **Revisit when the
supported version range widens, or when a minor lands that changes protocol
fields rather than dial fields** — every protocol option struct was byte-identical
between 1.12.12 and 1.13.12, and that is why this is deferrable.

A cheaper intermediate step, if the pin is bumped: nothing about the gate assumes
the pin is *older* than the binary. Bumping to 1.13 and keeping the gate would
serve 1.13 hosts fully and 1.12 hosts with the same withholding behaviour,
inverted.

---

## 7. Traps

Each of these cost real time.

1. **A field-less struct is not an absent one.** `option.StubOptions` is
   `struct{}`; guarding on "zero fields" without an allowlist rejects `block`
   and `dns` outbounds.
2. **`false` counts as empty.** With a `value !== undefined && value !== ''`
   test, `set_system_proxy: false` reads as filled, pinning the switch visible
   **and** un-removable, since removal is only offered for empty fields. Numbers
   are the opposite: `0` is meaningful (`listen_port: 0` means "pick one").
3. **Removing a field must delete the key**, not blank it. Writing `""` persists
   it as an explicit setting and makes it permanently un-removable next time.
4. **Type change must prune, not wipe and not carry.** Inbounds/DNS carried
   stale fields forward (sing-box decodes strictly and rejects them); outbounds
   wiped the whole model, discarding shared dial fields the operator had already
   filled in. `pruneForType` keeps the intersection.
5. **Some constraints only fire at start, not at `check`.** A selector with no
   members (`missing tags`) or a `default` naming a non-member
   (`default outbound not found`) both pass `sing-box check` and then refuse to
   start. Client and server validation are the only places to catch them.
6. **`option.DNSServerOptions` and `option.Outbound` cannot be bound
   reflectively.** Every real field lives behind an `Options any` that only
   `UnmarshalJSONContext` populates. `c.Bind` leaves it nil and the config then
   fails to marshal — which broke DNS server add/edit entirely until it was
   found.
7. **Some outbounds are not owned by the form.** `noderules` →
   `BuildGroupOutbounds` rebuilds any outbound whose tag matches a Filter or
   Group name, discarding form edits on the next apply. `GET /outbounds` returns
   `managed_tags` and the dialog warns.
8. **Verify the artifact, not the command output.** `bun --cwd frontend run build`
   printed `✓ built` without rebuilding, which nearly produced a "the feature
   does not work" conclusion. Grep the built bundle for a string you just added.

---

## 8. File reference

| Path | Role |
|---|---|
| `app/pkg/config/registry.go` | Type → option struct. The source of truth. |
| `app/pkg/config/{inbound,dns,outbound}_types.go` | Exported type lists + classification helpers |
| `cmd/gen-option-schema/main.go` | Generator: domains table, reflection, emission |
| `cmd/gen-option-schema/versions.go` | sing-box deprecation table → inventory mapping |
| `frontend/src/schemas/optionSchema.ts` | Domain-agnostic: tiers, visibility, prune, defaults, version helpers |
| `frontend/src/schemas/*Inventory.generated.ts` | Generated. **Do not edit.** |
| `frontend/src/schemas/*Fields.ts` | Curation — the only hand-maintained layer |
| `frontend/src/schemas/*Fields.test.ts` | Decision rules, not data |
| `frontend/src/components/SchemaFieldsEditor.vue` | Tier rendering, add/remove rows |
| `frontend/src/components/SchemaFieldControl.vue` | One field → one control |
| `frontend/src/composables/useSingBoxVersion.ts` | Installed binary version |

Composite controls, for fields with no generic editor:
`UsersEditor.vue` (inbound `users`), `HostsEditor.vue` (DNS `predefined`),
`JsonField.vue` (anything uncurated and object-shaped).

Current coverage: **Inbound** 17 types, **DNSServer** 11, **Outbound** 20.
Remaining: `endpoint`, `service`.
