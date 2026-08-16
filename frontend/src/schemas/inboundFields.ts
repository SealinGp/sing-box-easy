/**
 * Inbound field curation — the editorial layer over the generated inventory.
 *
 * `inboundInventory.generated.ts` says which fields sing-box accepts for each
 * inbound type and what shape they are. This says which ones matter, what
 * control to render, and what to call them. The generic machinery — tiers,
 * visibility, pruning, defaults — lives in `optionSchema.ts` and is shared with
 * the DNS server form; see that file for why the tiers are core/typical/advanced
 * rather than required/optional, and why uncurated fields stay reachable.
 */
import {
  INBOUND_INVENTORY,
  type InboundFieldKey,
  type InboundTypeName,
} from './inboundInventory.generated'
import { createSchema, type ControlKind, type FieldCuration } from './optionSchema'

/** Every field name across every inbound type — so the shared map cannot hold a typo. */
type AnyInboundFieldKey = {
  [T in InboundTypeName]: InboundFieldKey<T>
}[InboundTypeName]

const SHARED: Partial<Record<AnyInboundFieldKey, FieldCuration>> = {
  listen: { tier: 'core', order: 10, default: '127.0.0.1', placeholder: '127.0.0.1' },
  listen_port: { tier: 'core', order: 20, control: 'number', placeholder: '1080' },

  users: { tier: 'typical', order: 30, control: 'users' },

  // Shown early when present, because a proxy that terminates TLS is usually
  // configured around it. Still advanced: most inbounds do not use it.
  tls: { tier: 'advanced', order: 40, control: 'json' },

  network: { tier: 'advanced', order: 50, control: 'select', options: ['tcp', 'udp'] },
  detour: { tier: 'advanced', order: 60, control: 'outbound' },

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

const schema = createSchema<InboundTypeName>({
  inventory: INBOUND_INVENTORY,
  labelPrefix: 'inbounds.form.fields',
  shared: SHARED as Record<string, FieldCuration>,
  byType: BY_TYPE as Partial<Record<InboundTypeName, Record<string, FieldCuration>>>,
})

export const resolveInboundFields = schema.resolveFields
export const pruneForType = schema.pruneForType
export const applyTypeDefaults = schema.applyTypeDefaults
export const isInboundType = schema.isKnownType

export { isFieldFilled, withoutField } from './optionSchema'
export type {
  ControlKind,
  FieldCuration,
  FieldTier,
  OptionFieldKind as InboundFieldKind,
  ResolvedField,
} from './optionSchema'

export { INBOUND_INVENTORY }
export type { InboundTypeName }
export { INBOUND_TYPE_NAMES } from './inboundInventory.generated'
