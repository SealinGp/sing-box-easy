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

export interface OptionFieldInfo {
  readonly kind: OptionFieldKind
  /** Element kind, for kind: 'list'. */
  readonly item?: OptionFieldKind
  /**
   * Still accepted by sing-box, but documented as deprecated. Kept so an
   * existing config that uses the field can still be edited rather than having
   * it silently dropped on save.
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

export interface ResolvedField {
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
