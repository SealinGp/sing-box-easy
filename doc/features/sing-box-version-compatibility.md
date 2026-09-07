# sing-box Version Compatibility

> **Status**: accepted; core compatibility and DNS editor implemented
>
> **Target core versions**: 1.12.12, 1.13.x, and 1.14.x
> **Primary decision**: the installed sing-box binary is the authority for
> configuration validity; sing-box-easy must not require the entire document to
> decode through one pinned version of `option.Options`.

### Implementation progress

- [x] Phase 1: installed-core validation, version detection, capability API,
  and structured validation stages.
- [x] Phase 2: raw active-config read/save, raw history retrieval, and raw-safe
  rollback.
- [x] Phase 3: subscription updates mutate only recognized outbounds while
  preserving newer DNS/route sections and unknown outbound records.
- [ ] Phase 4: migrate every remaining structured editor to section-local,
  raw-preserving adapters.
  - [x] DNS document, server, hosts, and rule editors, including 1.14
    `evaluate`/`respond` actions.
  - [x] Log and experimental (`clash_api`, `cache_file`, `v2ray_api`) editors.
  - [x] Inbound editor and route rules, rule sets, cascade references, and
    final-policy editor.
  - [ ] Outbound and node-rule editors.
- [ ] Phase 5: upgrade the optional typed helper dependency after the raw seams
  are established; this does not change the installed sing-box binary.
- [ ] Phase 6: migrate the full frontend to `/api/v1` and document removal of
  the legacy route. Core and raw-config endpoints are already published there.

The frontend now consumes the core capability response for DNS action gating.
It keeps using the legacy API prefix for feature endpoints until those routes
are all available under `/api/v1`.

---

## 1. Summary

sing-box-easy currently compiles against `github.com/sagernet/sing-box
v1.12.12`. Every configuration request is decoded into that version's
`option.Options` before the installed `sing-box` executable is asked to check
it. Consequently, a valid 1.14 configuration containing a DNS action such as
`evaluate` is rejected by the panel even when the installed 1.14 binary accepts
the same file.

The proposed design separates configuration storage and validation from the Go
option types used by editor helpers:

1. Store and transport the complete configuration as raw JSON.
2. Validate JSON syntax in sing-box-easy, then delegate semantic validation to
   the installed sing-box binary.
3. Detect the installed core version and publish a capability document for the
   frontend.
4. Patch only the section owned by an operation, preserving unknown fields in
   every other section.
5. Keep typed sing-box option structs as optional adapters for structured
   editing and protocol conversion, not as the persistence format or final
   validator.

This supports known 1.12, 1.13, and 1.14 configurations and also allows newer
unknown fields to survive read, save, subscription update, and rollback flows.

## 2. Problem statement

### 2.1 Current failure

The current validation flow is:

```text
HTTP request body
        │
        ▼
Hertz binds JSON into config.SingBoxConfig
        │
        ▼
SingBoxConfig.UnmarshalJSON uses the compiled option.Options version
        │
        ▼
Manager.ValidateConfig writes a staging file
        │
        ▼
installed sing-box check -c <staging-file>
```

The request never reaches `sing-box check` when the embedded decoder rejects a
new field or discriminator. For example, the embedded 1.12.12 decoder reports:

```text
dns.rules[1]: unknown DNS rule action: evaluate
```

The installed binary version and the compiled Go dependency are independent.
Restarting sing-box-easy does not align them; sing-box-easy must be rebuilt to
change its embedded dependency.

### 2.2 Affected operations

This is not limited to `POST /config/validate`. The following paths currently
depend on decoding the full document into `SingBoxConfig`:

- reading and returning the active configuration;
- saving a complete configuration;
- validating before service start, restart, or reload;
- updating subscription outbounds;
- editing inbounds, outbounds, DNS, route rules, and experimental options;
- creating and restoring configuration history;
- DNS, route, and subscription probes that inspect the configuration.

A raw validation endpoint alone would make one request succeed while the other
operations remained incompatible.

### 2.3 Why one newer Go dependency is insufficient

Upgrading the Go dependency to 1.14 is useful, but it does not create durable
multi-version support:

- newer option structs may remove or reinterpret deprecated fields;
- older installed binaries must still reject features they cannot run;
- future core fields would again be rejected by an older panel build;
- decoding and re-encoding a document can silently remove fields unknown to
  the embedded version;
- several versions of the same Go module path cannot be selected dynamically
  as ordinary runtime adapters.

The official migration guide records configuration changes in both 1.12 and
1.14, including DNS server formats and response matching:
<https://sing-box.sagernet.org/migration/>.

## 3. Goals

- Support installed sing-box versions 1.12.12, 1.13.x, and 1.14.x.
- Accept a configuration whenever the installed core accepts it.
- Reject a version-specific feature with an actionable compatibility error.
- Preserve unknown fields and types during unrelated edits.
- Keep atomic validation, version history, and rollback behavior.
- Let structured forms expose only capabilities supported by the installed
  core.
- Keep the raw JSON editor available for fields the panel does not understand.
- Make subscription refreshes safe for newer configuration documents.
- Test compatibility using actual released sing-box binaries.

## 4. Non goals

- Reimplement the complete sing-box validator in sing-box-easy.
- Translate every configuration feature between versions automatically.
- Emulate 1.14 DNS racing on a 1.12 or 1.13 core.
- Guarantee that a configuration valid for one core version behaves identically
  on another version.
- Load multiple versions of the upstream Go module dynamically.
- Automatically downgrade a configuration when the sing-box binary is
  downgraded.

## 5. Design principles

### 5.1 Installed core is authoritative

Only the installed `sing-box check` command decides whether the complete
configuration can run on that host. Panel-side checks may improve error
messages, but must not reject syntax solely because an embedded library does
not recognize it.

### 5.2 Preserve before understanding

An operation that edits `outbounds` does not need to understand every DNS,
service, endpoint, or experimental option. Unknown data must survive an
unrelated edit.

### 5.3 Capabilities not API path versions

`/api/1.12.12` currently looks like a promise about the installed core. The
panel protocol and core configuration version are separate concerns. A stable
panel API should publish the detected core version and explicit capabilities.

### 5.4 No silent degradation

When a requested feature cannot be represented on the installed core, the
panel must reject the edit before saving. It must not silently convert a DNS
race to one resolver or discard an unknown field.

## 6. Proposed architecture

```text
                         ┌─────────────────────────┐
HTTP and frontend        │ Core capabilities       │
                         │ version and feature set │
                         └────────────┬────────────┘
                                      │
                                      ▼
┌───────────────────┐       ┌─────────────────────┐
│ Raw config store  │──────▶│ Config document     │
│ bytes and history │       │ raw JSON sections   │
└─────────┬─────────┘       └──────────┬──────────┘
          │                            │
          │                            ├── typed editor adapters
          │                            ├── subscription patcher
          │                            ├── route and DNS inspectors
          │                            └── unknown field preservation
          │
          ▼
┌─────────────────────────────────────────────────┐
│ Installed core adapter                          │
│ sing-box version                                │
│ sing-box check -c <staging-file>                │
└─────────────────────────────────────────────────┘
```

### 6.1 Core adapter

The version-dependent process interaction should live behind one small
interface:

```go
type CoreVersion struct {
	Major int
	Minor int
	Patch int
	Raw   string
}

type CoreAdapter interface {
	Version(ctx context.Context) (CoreVersion, error)
	Validate(ctx context.Context, configPath string) error
}
```

The production adapter invokes the same configured binary path used by the
service controller. Tests use a fake adapter or released binaries.

Validation errors should identify their stage:

```json
{
  "code": 4,
  "data": {
    "stage": "core_check",
    "core_version": "1.13.0"
  },
  "msg": "dns action evaluate requires sing-box >= 1.14.0"
}
```

Expected stages are `json_syntax`, `panel_guard`, `staging_write`, and
`core_check`.

### 6.2 Raw configuration document

The persistence representation should not embed `option.Options`:

```go
type ConfigDocument struct {
	Raw []byte
}
```

For section-level access, decode only the top-level object:

```go
type RawSections map[string]json.RawMessage
```

This representation intentionally permits unknown top-level sections. Section
adapters decode only the data they own.

The document module should expose a small interface:

```go
type ConfigStore interface {
	Read(ctx context.Context) (ConfigDocument, error)
	Validate(ctx context.Context, document ConfigDocument) error
	Save(ctx context.Context, document ConfigDocument) error
	Update(ctx context.Context, mutation Mutation) error
}
```

`Save` retains the current staging-file, binary-check, snapshot, and atomic
rename guarantees. `Update` reads once, applies one raw-preserving mutation,
validates the result, then saves it atomically.

### 6.3 Raw preserving mutations

Each mutation declares the section it owns:

```go
type Mutation interface {
	Apply(document ConfigDocument) (ConfigDocument, error)
}
```

Examples:

- subscription refresh owns selected entries in `outbounds`;
- DNS server editing owns one entry in `dns.servers`;
- DNS rule editing owns one entry in `dns.rules`;
- route rule editing owns one entry in `route.rules`;
- log editing owns `log`;
- service lifecycle does not mutate the document.

An outbound envelope can preserve fields not understood by the panel:

```go
type RawOutbound struct {
	Type string `json:"type"`
	Tag  string `json:"tag"`
	Raw  map[string]json.RawMessage
}
```

When changing one outbound, merge the known changed fields into its raw object
instead of reconstructing the whole outbound from a typed struct.

### 6.4 Typed adapters

The newest supported `option` package may remain useful for:

- parsing subscription links into known outbound shapes;
- generating structured form inventories;
- normalizing fields owned by a specific editor;
- route and DNS probe helpers where the input feature is supported.

These adapters are conveniences. Failure to decode through a typed adapter
must not prevent raw configuration retrieval, validation, version history, or
rollback.

### 6.5 Capability provider

At startup and after a core install or update, the backend runs `sing-box
version`, parses the semantic version, and computes capabilities.

```go
type Capabilities struct {
	CoreVersion      string `json:"core_version"`
	DNSEvaluate      bool   `json:"dns_evaluate"`
	DNSRespond       bool   `json:"dns_respond"`
	DNSRace          bool   `json:"dns_race"`
	DNSMatchResponse bool   `json:"dns_match_response"`
	DNSOptimistic    bool   `json:"dns_optimistic"`
}
```

Initial capability matrix:

| Feature | 1.12.12 | 1.13.x | 1.14.x |
|---|:---:|:---:|:---:|
| DNS `route` | Yes | Yes | Yes |
| DNS `route-options` | Yes | Yes | Yes |
| DNS `reject` | Yes | Yes | Yes |
| DNS `predefined` | Yes | Yes | Yes |
| DNS `evaluate` | No | No | Yes |
| DNS `respond` | No | No | Yes |
| DNS `match_response` | No | No | Yes |
| DNS `race` | No | No | Yes |
| Optimistic DNS cache | No | No | Yes |

The 1.14 DNS actions are documented at
<https://sing-box.sagernet.org/configuration/dns/rule_action/>.

Capabilities should be tested against the exact minimum release that introduced
a feature. They must not be inferred from the panel API path.

## 7. API changes

### 7.1 Core information

Add:

```text
GET /api/v1/core
```

Example response:

```json
{
  "code": 0,
  "data": {
    "version": "1.14.0",
    "supported": true,
    "minimum": "1.12.12",
    "maximum_tested": "1.14.x",
    "capabilities": {
      "dns_evaluate": true,
      "dns_respond": true,
      "dns_match_response": true,
      "dns_race": true,
      "dns_optimistic": true
    }
  },
  "msg": "success"
}
```

### 7.2 Raw configuration retrieval

`GET /api/v1/config` returns the JSON document without decoding it through
`option.Options`. The response may still use the standard envelope, provided
the configuration object remains structurally unchanged.

For diagnostics and exact export, an additional endpoint may return the raw
document directly:

```text
GET /api/v1/config/raw
Content-Type: application/json
```

### 7.3 Validation

`POST /api/v1/config/validate` accepts raw JSON and performs:

1. body size enforcement;
2. JSON syntax and top-level object checks;
3. security guards owned by the panel, if any;
4. staging-file creation with mode `0600`;
5. `sing-box check` using the installed core;
6. staging-file cleanup.

It does not bind the body to `SingBoxConfig`.

### 7.4 Saving

`PUT /api/v1/config` follows the existing safe order:

```text
receive raw document
  → syntax check
  → write staging file
  → installed core validation
  → snapshot active config
  → atomic rename
```

The save must fail closed if the proposed document does not validate. Existing
baseline failures may be reported separately, but must not permit a newly
invalid document without explicit recovery semantics.

### 7.5 Compatibility alias

Keep `/api/1.12.12` temporarily as an alias for existing frontends. New
development should use `/api/v1`. Deprecation of the old path is a panel API
migration and must not depend on the installed core version.

## 8. Frontend behavior

The frontend loads core capabilities before rendering structured configuration
forms.

For an unsupported option:

- do not offer it when creating a new item;
- preserve it when it already exists in raw JSON;
- show it as read-only or route the user to the raw editor;
- explain the minimum required core version;
- never remove it merely because the structured form cannot render it.

Example:

```text
DNS action "evaluate" requires sing-box 1.14.0 or newer.
Installed core: 1.13.0.
```

The frontend-generated schema should add capability metadata rather than
assuming every reflected field is universally available:

```ts
interface FeatureRequirement {
  since?: string
  deprecatedSince?: string
  removedSince?: string
}
```

## 9. Version behavior

### 9.1 Core 1.12.12

- Accept configurations validated by the 1.12.12 binary.
- Offer only the DNS actions supported by that core.
- Reject `evaluate`, `respond`, `race`, `match_response`, and optimistic DNS
  cache before saving, with minimum-version guidance.
- Preserve unknown raw fields during unrelated edits, but do not claim they can
  run on this core.

### 9.2 Core 1.13.x

- Use the same DNS action capability set as 1.12 unless an exact 1.13 release
  proves otherwise.
- Let the installed binary decide support for all other fields.
- Preserve deprecated fields even when a form no longer offers them.

### 9.3 Core 1.14.x

- Expose `evaluate`, `respond`, `race`, `match_response`, per-action DNS
  timeouts, and optimistic DNS cache.
- Use the installed 1.14 binary for complete validation.
- Surface migration errors for removed or incompatible legacy behavior instead
  of rewriting it silently. The 1.14 migration notes document DNS response
  matching and query-type behavior changes:
  <https://sing-box.sagernet.org/migration/#1140>.

### 9.4 Unsupported future versions

For a core newer than the maximum tested version:

- raw read, raw validate, raw save, version history, rollback, and service
  lifecycle remain available;
- structured editors operate only on known capabilities;
- the UI displays `newer than tested`, not `unsupported`, when the installed
  binary accepts the configuration;
- unknown content is preserved.

## 10. Migration plan

### Phase 1 Raw validation and core detection

- Add the `CoreAdapter` interface.
- Detect and expose the installed core version.
- Change `/config/validate` to accept raw JSON.
- Delegate semantic validation directly to the installed binary.
- Return structured validation stages and core version.

This fixes the immediate validation discrepancy but does not yet make typed
mutation paths safe.

### Phase 2 Raw storage and history

- Make active config reads return a `ConfigDocument`.
- Store historical snapshots as raw bytes.
- Validate and save raw documents atomically.
- Keep rollback independent of typed decoding.

After this phase, configuration retrieval, complete save, history, rollback,
and lifecycle validation work across supported core versions.

### Phase 3 Subscription safe mutations

- Replace full `SingBoxConfig` decoding in subscription updates.
- Decode and patch only the outbound array.
- Preserve unknown outbound types and fields.
- Validate the complete candidate document with the installed binary.
- Restart sing-box only after a successful save.

This phase is required before subscription refresh can be considered safe on a
1.14 configuration.

### Phase 4 Structured editor adapters

Migrate in order of operational risk:

1. DNS rules and servers;
2. route rules and rule sets;
3. outbounds and inbounds;
4. experimental options;
5. remaining inspectors and probes.

Each migrated editor must preserve unknown siblings and reject unsupported
edits with an explicit version requirement.

### Phase 5 Dependency upgrade

- Upgrade the helper dependency to sing-box 1.14.
- Update registries and schema generation for new actions and fields.
- Keep raw storage and installed-core validation unchanged.
- Treat the dependency as an adapter rather than restoring typed full-document
  decoding.

### Phase 6 API transition

- Publish `/api/v1`.
- Keep `/api/1.12.12` as a compatibility alias for one documented transition
  period.
- Update the frontend and API documentation.
- Add deprecation response metadata for the old path.

## 11. Test strategy

### 11.1 Released binary matrix

CI downloads and caches official binaries for:

- 1.12.12;
- the selected latest supported 1.13 patch;
- 1.14.0;
- optionally the latest 1.14 patch.

Each fixture is checked by the corresponding real executable.

| Fixture | 1.12.12 | 1.13.x | 1.14.x |
|---|:---:|:---:|:---:|
| Basic modern config | Pass | Pass | Pass |
| DNS route action | Pass | Pass | Pass |
| DNS evaluate and race | Reject | Reject | Pass |
| Optimistic DNS cache | Reject | Reject | Pass |
| Unknown future top-level field | Core decides | Core decides | Core decides |

Tests must assert both the verdict and the reported validation stage.

### 11.2 Preservation tests

For every mutation:

1. start with known fields plus sentinel unknown fields;
2. perform one structured edit;
3. compare untouched sections semantically and, where practical, byte-for-byte;
4. assert that sentinel fields remain;
5. validate the result with the target core binary.

Required regression cases include:

- a subscription update preserves 1.14 DNS actions;
- a DNS server edit preserves an unknown DNS rule action;
- a route edit preserves unknown services and endpoints;
- config history restores the exact raw document;
- a failed binary check never replaces the active config;
- a core downgrade reports incompatible features before restart.

### 11.3 Capability tests

- parse stable, prerelease, and vendor-suffixed version output;
- map each supported release to the expected features;
- reject malformed version output without guessing;
- refresh capabilities after core installation;
- ensure frontend controls follow the backend capability document.

## 12. Error handling

Errors should answer three questions:

1. Which stage failed?
2. Which core version made the decision?
3. What can the operator do next?

Example:

```json
{
  "code": 4,
  "data": {
    "stage": "core_check",
    "core_version": "1.13.0",
    "minimum_required": "1.14.0",
    "path": "dns.rules[1].action"
  },
  "msg": "DNS action evaluate is not supported by the installed core"
}
```

Raw core output should be retained for diagnostics but sanitized before being
logged or returned when it may contain configuration values.

## 13. Security requirements

- Never include the submitted configuration in validation logs.
- Redact authentication headers, proxy passwords, UUIDs, private keys, and
  subscription URLs.
- Create staging files with mode `0600`.
- Keep staging files beside the active configuration for atomic rename, but use
  a name that sing-box directory mode will not merge.
- Remove validation-only staging files on both success and failure.
- Apply request-body size limits before reading the full document.
- Do not expose raw configuration endpoints to unauthenticated users.
- Ensure version and validation commands cannot accept user-controlled command
  arguments or binary paths.

## 14. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Raw JSON reduces compile-time checking | Installed-core validation remains mandatory before save or lifecycle changes. |
| A partial editor removes unknown data | Use raw-preserving merge operations and sentinel preservation tests. |
| Binary version changes after capability detection | Refresh capabilities before save and after installation; the final binary check remains authoritative. |
| Newer core accepts fields the UI cannot display | Preserve them and keep the raw editor available. |
| Older core rejects a document after downgrade | Detect the downgrade and report incompatible paths before restart. |
| Validation command leaks secrets | Sanitize errors and never log submitted JSON. |
| Compatibility aliases become permanent | Publish a removal milestone and telemetry for old API usage. |

## 15. Rejected alternatives

### Compile only against 1.14 option types

This fixes the immediate `evaluate` decoding failure but does not preserve
future fields or guarantee correct validation for an older installed binary.
It remains useful as Phase 5, not as the compatibility architecture.

### Import one Go module version per core version

The upstream packages use the same module and import paths and are not designed
as runtime plugins. Aliasing or vendoring complete copies would multiply
registries, security updates, and tests without solving future-version field
preservation.

### Validate raw JSON but keep typed storage

This fixes only `POST /config/validate`. Reading, saving, subscription updates,
and restart validation would continue to fail or lose fields.

### Maintain a complete panel-side validator

This duplicates sing-box and will drift. Panel guards should cover only
application-owned invariants; the installed core should validate core
semantics.

### Silently downgrade unsupported features

Many features have no equivalent on older cores. For example, a 1.14 DNS race
cannot be reduced to a single 1.13 DNS server without changing availability and
privacy behavior. Reject with a clear requirement instead.

## 16. Acceptance criteria

The feature is complete when all of the following are true:

- A 1.14 DNS race configuration passes the panel validator when the installed
  core is 1.14.
- The same configuration is rejected with a minimum-version message when the
  installed core is 1.12 or 1.13.
- A valid 1.12 configuration can be read, saved, restarted, versioned, and
  rolled back while running core 1.12.12.
- Subscription refresh on a 1.14 document preserves DNS `evaluate`, `respond`,
  `race`, and `match_response` fields.
- Unknown fields in untouched sections survive every structured edit.
- The frontend does not offer unsupported features for the detected core.
- Raw editing and installed-core validation continue to work for a core newer
  than the maximum tested version.
- Failed validation never changes the active configuration.
- The compatibility matrix runs in CI against real released binaries.

## 17. Review questions

1. Should `/api/v1/config` return the configuration inside the standard
   response envelope, or should the raw endpoint be the primary interface?
2. Should unsupported existing fields be read-only in structured forms or make
   the entire item raw-editor-only?
3. Which exact 1.13 patch should define the supported test target?
4. Should a core downgrade be blocked while the active configuration requires a
   newer version?
5. Is semantic preservation sufficient for untouched sections, or should exact
   byte preservation be required where no mutation occurs?
6. How long should `/api/1.12.12` remain as a compatibility alias?

## 18. Implementation starting points

The first implementation slice should touch only:

- `app/pkg/config`: raw document, core adapter, staging validation, and storage;
- `app/routes/v1_12_12/config_handler.go`: raw validation request path;
- `app/pkg/service`: shared installed-core adapter;
- `frontend/src/services/config.ts`: raw validation and capability query;
- focused backend tests for 1.14 `evaluate` acceptance and 1.13 rejection.

Do not begin by rewriting every structured handler. Establish the raw document
and installed-core validation seam first, then migrate callers incrementally.
