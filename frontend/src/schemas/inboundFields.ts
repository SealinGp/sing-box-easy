/**
 * Editorial layer over the generated inbound field inventory.
 *
 * `inboundInventory.generated.ts` says which fields sing-box accepts for each
 * inbound type and what shape they are. It deliberately says nothing about
 * which ones matter — that judgement is here, and it is the only part a human
 * maintains.
 *
 * THREE TIERS, NOT "REQUIRED" / "OPTIONAL"
 * ────────────────────────────────────────
 * The obvious model — show required fields, hide optional ones — does not work,
 * because sing-box marks almost nothing required. For a `mixed` inbound the
 * docs mark exactly one field Required (`listen`); `users` is explicitly
 * "No authentication required if empty" and `set_system_proxy` is optional too.
 * A form built on required-ness would render a single address box and hide the
 * two fields anyone actually opens the dialog to set.
 *
 * What each doc page *shows* is the type's characteristic fields, which is a
 * different question from what sing-box will reject. So:
 *
 *   core      always rendered, never removable — the field the type cannot work
 *             without (`listen`, and the port for anything that binds one).
 *   typical   rendered by default, removable while empty — what the sing-box doc
 *             page for this type puts in its example.
 *   advanced  hidden behind the "add field" row until asked for, or until a
 *             loaded config turns out to use it.
 *
 * Genuine required-ness stays where it already lived, in
 * `utils/inboundRequiredFields.ts`, and is enforced on save.
 *
 * ANYTHING UNCURATED IS `advanced`
 * ────────────────────────────────
 * Fields absent from the maps below are not dropped — they resolve to the
 * advanced tier automatically. That is what makes this file safe to leave
 * incomplete: a sing-box upgrade that adds a field makes it reachable in the UI
 * immediately, just not promoted. Nothing has to be exhaustive except the
 * things we want to *promote*.
 */
import {
  INBOUND_INVENTORY,
  type InboundFieldInfo,
  type InboundFieldKind,
  type InboundFieldKey,
  type InboundTypeName,
} from './inboundInventory.generated'

export type FieldTier = 'core' | 'typical' | 'advanced'

/**
 * The control to render. Mostly inferred from the generated `kind`; named here
 * only when the inferred one would be wrong — `method` is a string in Go but a
 * fixed vocabulary in practice, and `users` is a list of objects that deserves
 * a real editor rather than a JSON textarea.
 */
export type ControlKind =
  | 'text'
  | 'number'
  | 'switch'
  | 'select'
  | 'chips'
  | 'password'
  | 'users'
  | 'json'

export interface FieldCuration {
  tier: FieldTier
  /** Lower sorts first within a tier. Uncurated fields sort last, alphabetically. */
  order?: number
  control?: ControlKind
  /** Fixed vocabulary for `control: 'select'`. */
  options?: readonly string[]
  /** Seeded when the field is first shown on a NEW inbound. Never on edit. */
  default?: unknown
  /** Defaults to `inbounds.form.fields.<key>`. */
  labelKey?: string
  hintKey?: string
  placeholder?: string
}

/** Every field name across every inbound type — so the shared map cannot hold a typo. */
type AnyInboundFieldKey = {
  [T in InboundTypeName]: InboundFieldKey<T>
}[InboundTypeName]

/**
 * Curation that applies to any type possessing the field.
 *
 * Keyed by field name rather than spread into each type on purpose: a spread
 * would bypass TypeScript's excess-property check, so `listen_port` could be
 * spread into `tun` — which has no such field — and fail silently. Resolution
 * intersects this with the type's real inventory instead, so a shared entry
 * simply does not apply where the field does not exist.
 */
const SHARED: Partial<Record<AnyInboundFieldKey, FieldCuration>> = {
  listen: { tier: 'core', order: 10, default: '127.0.0.1', placeholder: '127.0.0.1' },
  listen_port: { tier: 'core', order: 20, control: 'number', placeholder: '1080' },

  users: { tier: 'typical', order: 30, control: 'users' },

  // Shown early when present, because a proxy that terminates TLS is usually
  // configured around it. Still advanced: most inbounds do not use it.
  tls: { tier: 'advanced', order: 40, control: 'json' },

  network: { tier: 'advanced', order: 50, control: 'select', options: ['tcp', 'udp'] },
  detour: { tier: 'advanced', order: 60 },

  transport: { tier: 'advanced', order: 210, control: 'json' },
  multiplex: { tier: 'advanced', order: 220, control: 'json' },
  domain_resolver: { tier: 'advanced', order: 230, control: 'json' },
}

/**
 * Per-type curation. Overrides SHARED for the same key.
 *
 * Typed as a mapped type over `InboundTypeName`, so each entry is checked
 * against that type's own field keys — naming a field the type does not have is
 * a compile error, not an input that never binds.
 */
const BY_TYPE: { [T in InboundTypeName]: Partial<Record<InboundFieldKey<T>, FieldCuration>> } = {
  tun: {
    // tun has no listen/listen_port at all — the inventory omits them, which is
    // why the old `v-if="type !== 'tun'"` special case is no longer needed.
    address: {
      tier: 'core',
      order: 10,
      control: 'chips',
      default: ['172.19.0.1/30'],
      placeholder: '172.19.0.1/30',
    },
    auto_route: { tier: 'typical', order: 20, default: true },
    stack: { tier: 'typical', order: 30, control: 'select', options: ['mixed', 'system', 'gvisor'] },
    strict_route: { tier: 'typical', order: 40 },
    interface_name: { tier: 'advanced', order: 50, placeholder: 'tun0' },
    mtu: { tier: 'advanced', order: 60, placeholder: '9000' },
    route_address: { tier: 'advanced', order: 70, control: 'chips' },
    route_exclude_address: { tier: 'advanced', order: 80, control: 'chips' },
  },

  redirect: {},
  tproxy: {},

  direct: {
    override_address: { tier: 'typical', order: 30 },
    override_port: { tier: 'typical', order: 40, control: 'number' },
  },

  mixed: {
    set_system_proxy: { tier: 'typical', order: 40 },
  },
  http: {
    set_system_proxy: { tier: 'typical', order: 40 },
  },
  socks: {},

  shadowsocks: {
    method: {
      tier: 'core',
      order: 25,
      control: 'select',
      options: [
        '2022-blake3-aes-128-gcm',
        '2022-blake3-aes-256-gcm',
        '2022-blake3-chacha20-poly1305',
        'none',
        'aes-128-gcm',
        'aes-192-gcm',
        'aes-256-gcm',
        'chacha20-ietf-poly1305',
        'xchacha20-ietf-poly1305',
      ],
      default: '2022-blake3-aes-128-gcm',
    },
    password: { tier: 'core', order: 26, control: 'password' },
    // Multi-user and relay are alternative shapes to the single password above;
    // both stay advanced so the common single-user server is not buried.
    users: { tier: 'advanced', order: 100, control: 'users' },
    destinations: { tier: 'advanced', order: 110, control: 'json' },
    managed: { tier: 'advanced', order: 120 },
  },

  vmess: {},
  vless: {},

  trojan: {
    fallback: { tier: 'advanced', order: 100, control: 'json' },
    fallback_for_alpn: { tier: 'advanced', order: 110, control: 'json' },
  },

  naive: {},

  hysteria: {
    up_mbps: { tier: 'typical', order: 40, control: 'number' },
    down_mbps: { tier: 'typical', order: 50, control: 'number' },
    obfs: { tier: 'advanced', order: 100 },
  },

  hysteria2: {
    up_mbps: { tier: 'typical', order: 40, control: 'number' },
    down_mbps: { tier: 'typical', order: 50, control: 'number' },
    obfs: { tier: 'advanced', order: 100, control: 'json' },
    masquerade: { tier: 'advanced', order: 110 },
    ignore_client_bandwidth: { tier: 'advanced', order: 120 },
  },

  tuic: {
    congestion_control: {
      tier: 'advanced',
      order: 100,
      control: 'select',
      options: ['cubic', 'new_reno', 'bbr'],
    },
    auth_timeout: { tier: 'advanced', order: 110 },
    zero_rtt_handshake: { tier: 'advanced', order: 120 },
    heartbeat: { tier: 'advanced', order: 130 },
  },

  shadowtls: {
    version: { tier: 'core', order: 25, control: 'number', default: 3, placeholder: '3' },
    handshake: { tier: 'core', order: 27, control: 'json' },
    password: { tier: 'advanced', order: 100, control: 'password' },
    strict_mode: { tier: 'advanced', order: 110 },
    wildcard_sni: { tier: 'advanced', order: 120 },
  },

  anytls: {
    padding_scheme: { tier: 'advanced', order: 100, control: 'chips' },
  },
}

/**
 * The sub-fields of one entry in a `users` array, per inbound type.
 *
 * Every type spells this differently and the differences are not cosmetic:
 * mixed/http/socks/naive carry sing's own `auth.User`, which has NO json tags
 * at all — Go emits the Go field names, and its decoder matches case
 * insensitively, so the documented lowercase `username`/`password` is what we
 * write. The rest use purpose-built structs with real tags.
 */
export interface UserFieldSpec {
  key: string
  control: ControlKind
  /** At least one identity field must be filled for the entry to be meaningful. */
  identity?: boolean
}

const AUTH_USER: UserFieldSpec[] = [
  { key: 'username', control: 'text', identity: true },
  { key: 'password', control: 'password' },
]

const NAME_PASSWORD: UserFieldSpec[] = [
  { key: 'name', control: 'text' },
  { key: 'password', control: 'password', identity: true },
]

export const USER_FIELDS: Partial<Record<InboundTypeName, UserFieldSpec[]>> = {
  mixed: AUTH_USER,
  http: AUTH_USER,
  socks: AUTH_USER,
  naive: AUTH_USER,

  shadowsocks: NAME_PASSWORD,
  trojan: NAME_PASSWORD,
  hysteria2: NAME_PASSWORD,
  anytls: NAME_PASSWORD,
  shadowtls: NAME_PASSWORD,

  vmess: [
    { key: 'name', control: 'text' },
    { key: 'uuid', control: 'text', identity: true },
    { key: 'alterId', control: 'number' },
  ],
  vless: [
    { key: 'name', control: 'text' },
    { key: 'uuid', control: 'text', identity: true },
    { key: 'flow', control: 'select' },
  ],
  tuic: [
    { key: 'name', control: 'text' },
    { key: 'uuid', control: 'text', identity: true },
    { key: 'password', control: 'password' },
  ],
  hysteria: [
    { key: 'name', control: 'text' },
    { key: 'auth_str', control: 'password', identity: true },
  ],
}

/** VLESS `flow` accepts one documented value, or empty for none. */
export const VLESS_FLOW_OPTIONS = ['', 'xtls-rprx-vision'] as const

export interface ResolvedField {
  key: string
  tier: FieldTier
  control: ControlKind
  kind: InboundFieldKind
  /** Element kind, when kind is 'list'. */
  item?: InboundFieldKind
  options?: readonly string[]
  default?: unknown
  labelKey: string
  hintKey?: string
  placeholder?: string
  deprecated: boolean
}

/** Control to use when curation does not name one. */
function inferControl(info: InboundFieldInfo): ControlKind {
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
 * The full field list for one inbound type, tier-resolved and sorted.
 *
 * Every field in the inventory comes back — including deprecated ones and ones
 * nobody curated. Callers decide what to render; nothing is filtered away here,
 * because a field omitted at this layer would be a field the operator can never
 * reach or even see in a config they already have.
 */
export function resolveInboundFields(type: InboundTypeName): ResolvedField[] {
  const inventory = INBOUND_INVENTORY[type] as Record<string, InboundFieldInfo>
  const byType = BY_TYPE[type] as Partial<Record<string, FieldCuration>>

  const resolved = Object.entries(inventory).map(([key, info]) => {
    const curation = byType[key] ?? SHARED[key as AnyInboundFieldKey]

    // A deprecated field is never promoted, whatever the curation says: it can
    // still be edited when a loaded config uses it, but it is not offered up.
    const tier: FieldTier = info.deprecated ? 'advanced' : (curation?.tier ?? 'advanced')

    return {
      key,
      tier,
      control: curation?.control ?? inferControl(info),
      kind: info.kind,
      item: info.item,
      options: curation?.options,
      default: curation?.default,
      labelKey: curation?.labelKey ?? `inbounds.form.fields.${key}`,
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
 * Whether a value counts as set, for "should this field be visible" and "may it
 * be removed".
 *
 * `false` deliberately counts as EMPTY. This is the trap the route-rule matcher
 * form does not hit because it has no boolean matchers: with the usual
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

/**
 * Drop the previous type's fields when the operator switches type.
 *
 * Without this, building a shadowsocks inbound and then switching to trojan
 * carries `method` and `password` along into the saved payload. sing-box
 * decodes options strictly and rejects unknown keys, so the save fails with an
 * error naming a field the form no longer shows.
 *
 * `tag` and `type` survive because they live on the inbound wrapper rather than
 * in any type's option struct, so they appear in no inventory.
 */
export function pruneForType(
  inbound: Readonly<Record<string, unknown>>,
  type: InboundTypeName,
): Record<string, unknown> {
  const allowed = INBOUND_INVENTORY[type] as Record<string, InboundFieldInfo>
  const next: Record<string, unknown> = {}

  for (const [key, value] of Object.entries(inbound)) {
    if (key === 'tag' || key === 'type' || key in allowed) {
      next[key] = value
    }
  }
  next.type = type
  return next
}

/**
 * Seed defaults for a NEW inbound of this type.
 *
 * Only fills fields that are absent, and only core/typical ones — an advanced
 * field the operator has not asked for should not appear pre-filled. Never
 * called when opening an existing inbound: writing a default into a config that
 * deliberately omitted the key would change behaviour on open, before the
 * operator touched anything.
 */
export function applyTypeDefaults(
  inbound: Readonly<Record<string, unknown>>,
  type: InboundTypeName,
): Record<string, unknown> {
  const next: Record<string, unknown> = { ...inbound, type }

  for (const field of resolveInboundFields(type)) {
    if (field.tier === 'advanced') continue
    if (field.default === undefined) continue
    if (next[field.key] !== undefined) continue
    next[field.key] = Array.isArray(field.default) ? [...field.default] : field.default
  }

  return next
}

/** Remove a field, deleting the key rather than blanking it. */
export function withoutField(
  inbound: Readonly<Record<string, unknown>>,
  key: string,
): Record<string, unknown> {
  const { [key]: _removed, ...rest } = inbound
  return rest
}

export { INBOUND_INVENTORY }
export type { InboundTypeName, InboundFieldKind }
export { INBOUND_TYPE_NAMES } from './inboundInventory.generated'
