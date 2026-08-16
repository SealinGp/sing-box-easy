# Schema-Driven Option Forms

How the Inbound, DNS Server, Outbound, DNS Rule and Route Rule dialogs know what
fields exist, which ones matter, and which ones the installed sing-box will
actually accept.

> **Status**: implemented for 6 domains — inbound, outbound, DNS server, DNS rule
> action, route rule action, route rule matcher. `endpoint` and `service`
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
| DNS rule action lists | Three copies. `types/dns.ts` said `'route' \| 'return' \| 'reject'` — `return` is not a sing-box action and `route-options`/`predefined` were missing; `DNSRules.vue` carried a second, correct list; a third `switch` decided which fields to strip on save. |
| Route rule matchers | 9 of 37 rendered, and two of the nine (`geosite`, `geoip`) had been REMOVED from sing-box since 1.12.0 — promoted as curated dropdowns whose every value produced a rule that could not start. |
| Route rule actions | 9 of ~45 fields across 6 of the 7 actions; `direct` was not offered at all. The ten dial options were shown only under `route-options`, though sing-box accepts them on `route` too. |
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

### 4.1a The docs site is not a version

`sing-box.sagernet.org` serves **dev-next**. It is not the documentation for any
release, and it is not a substitute for reading the option structs.

Measured while building the DNS rule action domain. The docs page for
`dns/rule_action` lists a `route` action with `speculative`, `timeout`,
`disable_optimistic_cache` and `remove_client_subnet`. Against actual tags:

| Field | 1.12.12 (pin) | 1.13.0 | 1.13.11 | 1.14.0-beta.1 | docs |
|---|---|---|---|---|---|
| server, strategy, disable_cache, rewrite_ttl, client_subnet | ✓ | ✓ | ✓ | ✓ | ✓ |
| tag, speculative, timeout, disable_optimistic_cache | — | — | — | ✓ | ✓ |
| remove_client_subnet | — | — | — | **—** | ✓ |

So one documented field exists in **no release at all**, and the rest are a whole
minor further out than the page implies. Curating from that page would have
produced a form offering four fields every installable binary rejects — the
precise failure §4.2 exists to prevent, arrived at from the opposite direction.

```bash
# What to do instead, for any tag:
curl -s https://raw.githubusercontent.com/SagerNet/sing-box/<tag>/option/rule_action.go
```

### 4.1b A removal can predate the pin

`geosite`, `geoip` and `source_geoip` on a route or DNS rule were **removed in
1.12.0** — the version this repo pins. They still exist as Go fields, so they
parse; sing-box then refuses to build the rule:

```
$ sing-box check          # 1.13.11
geosite database is deprecated in sing-box 1.8.0 and removed in sing-box 1.12.0
```

The route form promoted `geosite` and `geoip` as the primary content matchers,
with curated dropdowns of 12 and 7 options. All 19 values produced a rule that
could not start.

This is the gate working in a direction §4.2 does not describe: `removed` is at
or below the PINNED library, not merely below the installed binary, so the field
is withheld from every host. Nothing about `isRetired` needed changing — the
entry simply had to exist in `versions.go`, and sing-box does not report these
through `deprecated.Report` at all, so nothing found them automatically.

Worth checking for the remaining domains on any dependency bump:

```bash
grep -rn 'removed in sing-box' "$(go env GOMODCACHE)/github.com/sagernet/sing-box@<ver>"
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

### When the discriminator is not `type`

DNS rule actions key off `action`, because a DNS rule already uses `type` for
something else — the matcher shape (`default` / `logical`), which varies
independently of what the rule does once it matches. `createSchema` takes a
`typeKey` for this; binding it wrong turns a logical rule into an unparseable
one on the first action change.

That domain is also the first whose record the schema only **partly owns**. A
DNS rule is one flat JSON object holding both the action's fields and every
match condition. `pruneForType` is an allowlist and would delete the entire
matcher, so there is a denylist sibling:

| | Keeps | Use for |
|---|---|---|
| `pruneForType` | identity keys + this type's fields | a record the schema fully describes (inbound, outbound, DNS server) |
| `pruneForeignFields` | everything except keys another type in this inventory owns | a record the schema shares with something else (DNS rule) |

The denylist is the safer default where it applies: a key no inventory has heard
of is kept **by construction**. The hand-written allowlist it replaced
(`EDITED_FIELDS` in `DNSRules.vue`) destroyed any field nobody remembered to
list — including `no_drop`, which was listed but never rendered, so it was read
into nothing and written back as nothing.

The route rule form had neither. No allowlist, no denylist, nothing: the whole
in-memory object went to the server, so switching the action shipped the
previous action's fields. Against a running panel, the two outcomes were

```
POST {"action":"reject","outbound":"direct"}   -> 200, outbound SILENTLY DROPPED
POST {"action":"sniff","outbound":"direct",…}  -> 400 unknown field "outbound"
```

silent data loss or a hard error, depending on which pair of actions.

### Render one domain as several sections

A domain whose fields fall into groups the operator reasons about separately —
route rule matchers are the only case so far — adds a `group` to its curation
and filters the resolved list per section.

**`SchemaFieldsEditor` knows nothing about groups.** It already takes an
already-resolved `fields` array, so grouping is just filtering: several
instances bound to the same record, each handed its own subset. Each keeps its
own `useOptionalFields` and its own add-row, which is what a grouped form wants
anyway.

That was a deliberate choice over teaching the editor about sections. The route
matcher grouping is not decoration — it encodes that CONTENT matchers are an
alternative to a rule set (combining them is the AND trap) while CONTEXT
matchers narrow either one and must never be folded away. That reasoning belongs
in the component that documents it, not in machinery with one user.

An uncurated field has no group. `resolveMatcherFields` files those under
CONTEXT rather than CONTENT, because the safe failure is a matcher that shows
when it need not; a matcher hidden by the rule-set choice would silently
disappear from a rule that uses it.

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

Re-measured for DNS rule actions, 1.12.12 → 1.13.11: **zero drift.** The four
action structs are field-for-field identical across the entire 1.13 line, so the
generated inventory is complete for every binary an operator can install today
and the §4 gate has nothing to withhold. The first real movement is
1.14.0-beta.1, which adds 6 fields across `route` and `route-options`.

That is the trigger §6.5 was waiting for — 1.14 changes *action* fields, not dial
fields — but it is still a beta, so the pin-only inventory stands. Revisit when
1.14 ships stable.

Measured 1.12.12 → 1.13.12 across the first three domains:

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
9. **A Go type's underlying kind can lie about its wire format.** `option.DNSRCode`
   is an `int` that marshals as `"NXDOMAIN"`; `option.DNSRecordOptions` is a
   struct embedding the `dns.RR` interface that marshals as a one-line RR string.
   Classified by `reflect.Kind` they became a number spinner and a JSON textarea.
   Both needed `namedKinds` entries. When adding a domain, marshal one fully
   populated value of every type and read the JSON — that is what
   `TestDNSRuleActionRoundTrip` does.
10. **`badoption.Listable[T]` collapses a single entry to a bare scalar.** A
    `predefined` rule with one answer marshals as `"answer": "a. 3600 IN A ..."`,
    not a one-element array. A chips control bound to that renders one chip per
    *character*. The conditions form already coerced scalar → array for
    `domain`/`rule_set`; anything list-shaped needs the same treatment.
11. **A field can be declared in a context that refuses it.** `DirectActionOptions`
    IS `DialerOptions`, so `detour` reflects onto the `direct` route action — and
    sing-box answers "detour is not available in the current context".
    `domain.ExcludedFields` drops those. Omitting is safe here in a way it is not
    for a deprecated field: a config carrying one cannot decode at all, so none
    can be holding a value the form would then hide.
12. **A Go type's underlying kind can lie about its VOCABULARY too, not just its
    shape.** `option.NetworkStrategy` and `option.InterfaceType` are `uint8`
    enums marshalling as names. They shipped as `number` and `list of number` in
    the released inbound and outbound inventories, so those forms rendered number
    spinners for `default|fallback|hybrid` and `wifi|cellular|ethernet|other`.
13. **Two fields can look like one.** `port` is `Listable[uint16]`; `port_range`
    is `Listable[string]` and its separator is a COLON. The route form had a
    single `port` field that kept range syntax as a string and advertised
    `8080-8090` in its placeholder — wrong field and wrong separator, rejected
    twice over.
14. **Defaults belong on create, never on open.** `openEditRuleModal` seeded
    `rcode: 'NXDOMAIN'` for a missing value, but an absent rcode means `NOERROR` —
    so opening a rule and pressing Update silently changed what it did.
    `applyTypeDefaults` is called on create and on a type change only, and this
    is why.

---

## 8. File reference

| Path | Role |
|---|---|
| `app/pkg/config/registry.go` | Type → option struct. The source of truth. |
| `app/pkg/config/{inbound,dns,outbound,dns_rule_action,route_rule}_types.go` | Exported type lists + classification helpers |
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

Dependent controls, whose vocabulary is the live config rather than a constant:
`outbound` / `outbound-list` / `outbound-member` (outbound tags) and
`dns-server` (DNS server tags). All resolve their options in
`SchemaFieldControl.vue`, and all keep a value that no longer resolves visible
and flagged rather than dropping it — otherwise the next save looks like the
operator cleared it.

Current coverage: **Inbound** 17 types, **DNSServer** 11, **Outbound** 20,
**DNSRuleAction** 4, **RouteRuleAction** 7, **RouteRuleMatcher** 1 (37 fields).
Remaining: `endpoint`, `service`.

`RouteRuleMatcher` is the one domain that is not polymorphic — `RawDefaultRule`
is a single flat struct — so it generates one entry and reuses the generator's
shape only to avoid a second code path.
