/**
 * Domain-agnostic half of the schema-driven forms.
 *
 * sing-box has several families of polymorphic options — inbounds, DNS
 * transports, and (still to come) outbounds, endpoints, services. Each is a
 * `type` string selecting a struct whose fields the UI must render, and each
 * gets a generated inventory from `cmd/gen-option-schema`. Everything that does
 * NOT vary between those families lives here: the tier model, the visibility
 * rules, pruning, defaults.
 *
 * A domain then supplies only data — a curation map — and binds it with
 * `createSchema`.
 *
 * THREE TIERS, NOT "REQUIRED" / "OPTIONAL"
 * ────────────────────────────────────────
 * The obvious model does not work, because sing-box marks almost nothing
 * required. For a `mixed` inbound the docs mark exactly one field Required
 * (`listen`); `users` is explicitly "No authentication required if empty". A
 * form built on required-ness would render one address box and hide the fields
 * people open the dialog to set.
 *
 * What each doc page *shows* is the type's characteristic fields, which is a
 * different question from what sing-box will reject. So:
 *
 *   core      always rendered, never removable
 *   typical   rendered by default, removable while empty — what the doc page's
 *             example for this type shows
 *   advanced  hidden behind the "add field" row until asked for, or until a
 *             loaded config turns out to use it
 *
 * Genuine required-ness is enforced on save, per domain.
 *
 * ANYTHING UNCURATED IS `advanced`
 * ────────────────────────────────
 * Fields absent from a curation map are not dropped — they resolve to advanced
 * automatically. That is what makes curation safe to leave incomplete: a field
 * added by a future sing-box is reachable immediately, just not promoted.
 */

/** Field shapes the generator can produce. Mirrors cmd/gen-option-schema. */
export type OptionFieldKind =
  | 'address'
  | 'boolean'
  | 'cidr'
  | 'duration'
  | 'json'
  | 'list'
  | 'number'
  | 'object'
  | 'string'

/**
 * When sing-box retired something, from its own deprecation table.
 *
 * `removed` is the load-bearing one. The inventory describes the sing-box
 * LIBRARY this repo pins, but the operator runs whatever binary is installed —
 * routinely a newer one. A field or type whose `removed` version is at or below
 * the installed binary is not "discouraged", it is REJECTED: probed against
 * 1.13.11, `sniff` fails config decode outright and a `wireguard` outbound
 * fails at init. See `isRetiredIn` and cmd/gen-option-schema/versions.go.
 */
export interface OptionVersionNote {
  /** sing-box version that deprecated it, e.g. "1.11.0". */
  readonly since?: string
  /** sing-box version that removes it, e.g. "1.13.0". */
  readonly removed?: string
  /** Upstream migration guide, when the table names one. */
  readonly link?: string
}

export interface OptionFieldInfo extends OptionVersionNote {
  readonly kind: OptionFieldKind
  /** Element kind, for kind: 'list'. */
  readonly item?: OptionFieldKind
  /**
   * Still accepted by the pinned sing-box, but retired upstream. Kept so an
   * existing config that uses the field can still be edited rather than having
   * it silently dropped on save — but see `removed`, which decides whether the
   * INSTALLED binary will actually take it.
   */
  readonly deprecated?: true
}

export type FieldTier = 'core' | 'typical' | 'advanced'

/**
 * The control to render. Mostly inferred from the generated `kind`; named in
 * curation only when the inferred one would be wrong — `method` is a string in
 * Go but a fixed vocabulary in practice, and `users` is a list of objects that
 * deserves a real editor rather than a JSON textarea.
 */
export type ControlKind =
  | 'text'
  | 'number'
  | 'switch'
  | 'select'
  | 'chips'
  | 'password'
  | 'users'
  | 'hosts'
  /**
   * A `detour` — pick an outbound by tag, plus an empty entry meaning direct.
   * Its own kind rather than 'select' because the vocabulary is the live
   * config's outbound list, not a constant the schema can carry.
   */
  | 'outbound'
  /**
   * A group's `outbounds` — several outbound tags, self excluded. Distinct
   * from 'outbound' (one tag, empty meaning direct) because the vocabulary,
   * the cardinality and the self-reference rule all differ: a group listing
   * itself is a startup hang, not a config error.
   */
  | 'outbound-list'
  /**
   * One of the tags this record's OWN `outbounds` list currently holds — a
   * selector's `default`. Dependent rather than fixed: sing-box errors at start
   * with "default outbound not found" if it names something not in the group,
   * and that is past where `sing-box check` looks.
   */
  | 'outbound-member'
  | 'json'

export interface FieldCuration {
  tier: FieldTier
  /** Lower sorts first within a tier. Uncurated fields sort last, alphabetically. */
  order?: number
  control?: ControlKind
  /** Fixed vocabulary for `control: 'select'`. */
  options?: readonly string[]
  /** Seeded when the field is first shown on a NEW record. Never on edit. */
  default?: unknown
  /** Defaults to `<labelPrefix>.<key>`. */
  labelKey?: string
  hintKey?: string
  placeholder?: string
}

export interface ResolvedField extends OptionVersionNote {
  key: string
  tier: FieldTier
  control: ControlKind
  kind: OptionFieldKind
  item?: OptionFieldKind
  options?: readonly string[]
  default?: unknown
  labelKey: string
  hintKey?: string
  placeholder?: string
  deprecated: boolean
}

type Inventory = Record<string, Record<string, OptionFieldInfo>>

/** Control to use when curation does not name one. */
function inferControl(info: OptionFieldInfo): ControlKind {
  switch (info.kind) {
    case 'boolean':
      return 'switch'
    case 'number':
      return 'number'
    case 'object':
      return 'json'
    case 'list':
      // A list of objects has no generic editor; raw JSON is honest about that.
      return info.item === 'object' ? 'json' : 'chips'
    default:
      // string, address, cidr, duration — all single-line text. Duration keeps
      // its sing-box spelling ("5m", "300ms") rather than becoming a number.
      return 'text'
  }
}

const TIER_ORDER: Record<FieldTier, number> = { core: 0, typical: 1, advanced: 2 }
const UNCURATED_ORDER = 1_000

/**
 * Whether a value counts as set, for "should this field be visible" and "may it
 * be removed".
 *
 * `false` deliberately counts as EMPTY. With the usual
 * `value !== undefined && value !== ''` test, a `set_system_proxy: false` reads
 * as filled, which pins the switch permanently visible and — since removal is
 * only offered for empty fields — permanently un-removable. An unticked switch
 * is the absence of a setting, and sing-box agrees: every boolean option is
 * `omitempty`, so `false` and absent serialize identically.
 *
 * Numbers are the opposite case: `0` is meaningful (`alterId: 0`,
 * `listen_port: 0` meaning "pick one"), so it counts as set.
 */
export function isFieldFilled(value: unknown): boolean {
  if (value === undefined || value === null) return false
  if (Array.isArray(value)) return value.length > 0
  if (typeof value === 'boolean') return value
  if (typeof value === 'string') return value !== ''
  if (typeof value === 'object') return Object.keys(value).length > 0
  return true
}

/** Remove a field, deleting the key rather than blanking it. */
export function withoutField(
  record: Readonly<Record<string, unknown>>,
  key: string,
): Record<string, unknown> {
  const { [key]: _removed, ...rest } = record
  return rest
}

export interface SchemaOptions<TypeName extends string> {
  inventory: Inventory
  /** i18n path stem; a field's label defaults to `<labelPrefix>.<key>`. */
  labelPrefix: string
  /** Curation applying to any type possessing the field. */
  shared?: Record<string, FieldCuration>
  /** Per-type curation, overriding `shared` for the same key. */
  byType?: Partial<Record<TypeName, Record<string, FieldCuration>>>
  /**
   * Keys that live on the wrapper rather than in any type's option struct and
   * therefore appear in no inventory. They survive pruning.
   */
  identityKeys?: readonly string[]
}

/**
 * Binds the generic logic to one domain's inventory and curation.
 *
 * Callers get back functions that no longer mention the inventory, so the
 * components consuming them stay domain-agnostic too.
 */
export function createSchema<TypeName extends string>(options: SchemaOptions<TypeName>) {
  const {
    inventory,
    labelPrefix,
    shared = {},
    byType = {} as Partial<Record<TypeName, Record<string, FieldCuration>>>,
    identityKeys = ['tag', 'type'],
  } = options

  /**
   * The full field list for one type, tier-resolved and sorted.
   *
   * Every field in the inventory comes back — including deprecated ones and
   * ones nobody curated. Callers decide what to render; nothing is filtered
   * here, because a field omitted at this layer would be one the operator can
   * never reach, even in a config that already uses it.
   */
  function resolveFields(type: TypeName): ResolvedField[] {
    const fields = inventory[type] as Record<string, OptionFieldInfo> | undefined
    if (!fields) return []

    const curationForType = (byType[type] ?? {}) as Record<string, FieldCuration>

    const resolved = Object.entries(fields).map(([key, info]) => {
      const curation = curationForType[key] ?? shared[key]

      // A deprecated field is never promoted, whatever the curation says: it
      // can still be edited when a loaded config uses it, but it is not
      // offered up.
      const tier: FieldTier = info.deprecated ? 'advanced' : (curation?.tier ?? 'advanced')

      return {
        key,
        tier,
        control: curation?.control ?? inferControl(info),
        kind: info.kind,
        item: info.item,
        options: curation?.options,
        default: curation?.default,
        labelKey: curation?.labelKey ?? `${labelPrefix}.${key}`,
        hintKey: curation?.hintKey,
        placeholder: curation?.placeholder,
        deprecated: info.deprecated === true,
        since: info.since,
        removed: info.removed,
        _order: curation?.order ?? UNCURATED_ORDER,
      }
    })

    resolved.sort((a, b) => {
      const tierDiff = TIER_ORDER[a.tier] - TIER_ORDER[b.tier]
      if (tierDiff !== 0) return tierDiff
      if (a._order !== b._order) return a._order - b._order
      return a.key.localeCompare(b.key)
    })

    return resolved.map(({ _order, ...field }) => field)
  }

  /**
   * Drop the previous type's fields when the operator switches type.
   *
   * Without this, building a shadowsocks inbound and then switching to trojan
   * carries `method` and `password` into the saved payload. sing-box decodes
   * options strictly and rejects unknown keys, so the save fails with an error
   * naming a field the form no longer shows.
   */
  function pruneForType(
    record: Readonly<Record<string, unknown>>,
    type: TypeName,
  ): Record<string, unknown> {
    const allowed = (inventory[type] ?? {}) as Record<string, OptionFieldInfo>
    const next: Record<string, unknown> = {}

    for (const [key, value] of Object.entries(record)) {
      if (identityKeys.includes(key) || key in allowed) next[key] = value
    }
    next.type = type
    return next
  }

  /**
   * Seed defaults for a NEW record of this type.
   *
   * Only fills fields that are absent, and only core/typical ones — an advanced
   * field the operator has not asked for should not appear pre-filled. Never
   * called when opening an existing record: writing a default into a config
   * that deliberately omitted the key would change behaviour on open, before
   * the operator touched anything.
   */
  function applyTypeDefaults(
    record: Readonly<Record<string, unknown>>,
    type: TypeName,
  ): Record<string, unknown> {
    const next: Record<string, unknown> = { ...record, type }

    for (const field of resolveFields(type)) {
      if (field.tier === 'advanced') continue
      if (field.default === undefined) continue
      if (next[field.key] !== undefined) continue
      next[field.key] = Array.isArray(field.default) ? [...field.default] : field.default
    }

    return next
  }

  function isKnownType(value: unknown): value is TypeName {
    return typeof value === 'string' && value in inventory
  }

  return { resolveFields, pruneForType, applyTypeDefaults, isKnownType }
}

// ── Version gating ───────────────────────────────────────────────────────────
//
// The inventory describes the sing-box LIBRARY this repo pins; the operator
// runs whatever binary is installed, routinely a newer one. Fields a newer
// sing-box ADDED are simply missing from the form — a lost knob, no worse. The
// dangerous direction is the other one: offering something the installed binary
// REJECTS.
//
// Measured against 1.13.11 while this was written, with the panel pinned to the
// 1.12.12 library: `sniff` fails config decode outright, and it was offered in
// the inbound form's own "deprecated" row. `sing-box check` catches it before
// anything is written, so the result is a failed save with an opaque upstream
// error rather than a broken config — but the UI should not have suggested it.

/**
 * Compare two dotted version strings. Returns <0, 0, >0 like a comparator.
 *
 * Deliberately tolerant: sing-box versions carry suffixes ("1.13.0-beta.1",
 * "1.12.12"), and a missing or unparseable segment sorts as 0 rather than
 * throwing. Only the numeric prefix of each segment is read, so "0-beta.1"
 * compares as 0 — a prerelease of X counts as X, which is the conservative
 * answer for "has this been removed yet".
 */
export function compareVersions(a: string, b: string): number {
  const parse = (v: string) =>
    v
      .replace(/^v/, '')
      // Drop any prerelease/build suffix before splitting. Without this,
      // "1.13.0-beta.1" splits into four segments and sorts ABOVE "1.13.0",
      // while "1.13.0-rc" sorts equal to it — same shape, different answer.
      .split(/[-+]/)[0]!
      .split('.')
      .map((part) => parseInt(part, 10) || 0)

  const left = parse(a)
  const right = parse(b)
  const length = Math.max(left.length, right.length)

  for (let i = 0; i < length; i++) {
    const diff = (left[i] ?? 0) - (right[i] ?? 0)
    if (diff !== 0) return diff
  }
  return 0
}

/**
 * Whether the installed sing-box has actually removed this field or type.
 *
 * Unknown version -> false. Guessing "probably removed" would hide fields from
 * someone whose binary we simply failed to detect, which is worse than showing
 * one too many.
 */
export function isRetired(note: OptionVersionNote, installedVersion: string | undefined): boolean {
  if (!installedVersion || !note.removed) return false
  return compareVersions(installedVersion, note.removed) >= 0
}

/** Whether the installed sing-box deprecates it but still accepts it. */
export function isDeprecatedIn(
  note: OptionVersionNote,
  installedVersion: string | undefined,
): boolean {
  if (!note.since) return false
  if (isRetired(note, installedVersion)) return false
  if (!installedVersion) return true
  return compareVersions(installedVersion, note.since) >= 0
}
