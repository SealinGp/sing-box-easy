/**
 * Outbound field curation — the editorial layer over the generated inventory.
 *
 * Mirrors `inboundFields.ts` and `dnsServerFields.ts`; the generic machinery
 * lives in `optionSchema.ts`.
 *
 * WHAT THE OLD FORM COULD EDIT
 * ────────────────────────────
 * Ten scalar fields, one list, and the dialer subtree — for all 17 types it
 * offered. Everything protocol-specific beyond `server`/`port`/`method`/
 * `password`/`uuid` was unreachable: `tls`, `transport`, `multiplex`, VLESS
 * `flow`, VMess `alter_id`/`security`, Hysteria `up_mbps`/`down_mbps`/`obfs`,
 * TUIC `congestion_control`, WireGuard `peers`/`private_key`, SSH
 * `user`/`private_key`, ShadowTLS `version`. `socks` and `http` got no
 * `username`/`password` at all despite matching the "needs a server" predicate.
 *
 * `method` was a free-text input, so a typo produced a config that saved and
 * then failed to start. It is a fixed vocabulary here, as on the inbound side.
 *
 * PROPORTIONALITY
 * ───────────────
 * Most outbounds are not written here. On the config this was built against,
 * 165 of 176 come from subscription parsers and the node-rules engine; the form
 * owns the ~11 groups and terminal outbounds. So the group editing below is the
 * part that carries the weight, and the protocol curation mostly exists so that
 * a hand-made or imported outbound can be corrected without editing JSON.
 *
 * DEPRECATED TYPES ARE GATED, NOT HIDDEN
 * ──────────────────────────────────────
 * `dns` and `wireguard` are retired upstream — the generated
 * OUTBOUND_TYPE_NOTES carries the versions, and the UI compares them against
 * the INSTALLED sing-box. They stay listed so an existing config using one can
 * still be opened and fixed.
 */
import {
  OUTBOUND_INVENTORY,
  OUTBOUND_TYPE_NOTES,
  type OutboundFieldKey,
  type OutboundTypeName,
} from './outboundInventory.generated'
import { createSchema, type FieldCuration } from './optionSchema'
import { NETWORK_STRATEGIES } from './vocabularies'

/** Every field name across every outbound type — so the shared map cannot hold a typo. */
type AnyOutboundFieldKey = {
  [T in OutboundTypeName]: OutboundFieldKey<T>
}[OutboundTypeName]

/**
 * Shadowsocks ciphers. Same list the inbound form uses; sing-box accepts the
 * same set on both sides, and a typo here is a startup failure rather than a
 * validation error.
 */
const SHADOWSOCKS_METHODS = [
  '2022-blake3-aes-128-gcm',
  '2022-blake3-aes-256-gcm',
  '2022-blake3-chacha20-poly1305',
  'none',
  'aes-128-gcm',
  'aes-192-gcm',
  'aes-256-gcm',
  'chacha20-ietf-poly1305',
  'xchacha20-ietf-poly1305',
] as const

/**
 * Curation applying to any type possessing the field.
 *
 * Keyed by field name and intersected with each type's real inventory rather
 * than spread into it: a spread bypasses TypeScript's excess-property check, so
 * `server` could be spread into `block` — which has no fields at all — and fail
 * silently.
 */
const SHARED: Partial<Record<AnyOutboundFieldKey, FieldCuration>> = {
  server: { tier: 'core', order: 10, placeholder: 'example.com or 1.2.3.4' },
  server_port: { tier: 'core', order: 20, control: 'number', placeholder: '443' },

  password: { tier: 'core', order: 30, control: 'password' },
  uuid: { tier: 'core', order: 31 },
  method: { tier: 'core', order: 32, control: 'select', options: SHADOWSOCKS_METHODS },

  // socks/http carry these and the old form offered neither, so a proxy
  // needing auth could not be configured at all.
  username: { tier: 'typical', order: 40 },

  detour: { tier: 'typical', order: 50, control: 'outbound' },

  tls: { tier: 'advanced', order: 100, control: 'json' },
  transport: { tier: 'advanced', order: 110, control: 'json' },
  multiplex: { tier: 'advanced', order: 120, control: 'json' },
  network: { tier: 'advanced', order: 130, control: 'select', options: ['tcp', 'udp'] },
  domain_resolver: { tier: 'advanced', order: 140, control: 'json' },
  connect_timeout: { tier: 'advanced', order: 150 },

  // Dialer enums. Curated as selects rather than left to the inferred text /
  // chips control because the vocabularies are small, closed, and unguessable —
  // and because these three shipped as NUMBER inputs until the generator learned
  // that option.NetworkStrategy and option.InterfaceType marshal as names. See
  // schemas/vocabularies.ts.
  network_strategy: {
    tier: 'advanced',
    order: 160,
    control: 'select',
    options: NETWORK_STRATEGIES,
  },
  network_type: { tier: 'advanced', order: 170, control: 'chips' },
  fallback_network_type: { tier: 'advanced', order: 180, control: 'chips' },
}

/**
 * Per-type curation. Overrides SHARED for the same key.
 *
 * Typed as a mapped type over `OutboundTypeName`, so each entry is checked
 * against that type's own field keys — naming a field the type does not have is
 * a compile error, not an input that never binds.
 */
const BY_TYPE: {
  [T in OutboundTypeName]: Partial<Record<OutboundFieldKey<T>, FieldCuration>>
} = {
  // Terminal behaviours. `block` and `dns` have no options at all — the
  // inventory is empty and the editor shows its empty state.
  direct: {
    // Deprecated since 1.11, rejected at config decode by 1.13. The generated
    // inventory carries the versions; the UI gates on the installed binary.
    override_address: { tier: 'advanced', order: 200 },
    override_port: { tier: 'advanced', order: 210, control: 'number' },
  },
  block: {},
  dns: {},

  socks: {
    version: { tier: 'advanced', order: 100, control: 'select', options: ['4', '4a', '5'] },
    udp_over_tcp: { tier: 'advanced', order: 110, control: 'json' },
  },
  http: {
    path: { tier: 'advanced', order: 100 },
    headers: { tier: 'advanced', order: 110, control: 'json' },
  },

  shadowsocks: {
    plugin: { tier: 'advanced', order: 100 },
    plugin_opts: { tier: 'advanced', order: 110 },
    udp_over_tcp: { tier: 'advanced', order: 120, control: 'json' },
  },

  vmess: {
    alter_id: { tier: 'typical', order: 41, control: 'number' },
    security: {
      tier: 'advanced',
      order: 100,
      control: 'select',
      options: ['auto', 'none', 'zero', 'aes-128-gcm', 'chacha20-poly1305', 'aes-128-ctr'],
    },
    packet_encoding: { tier: 'advanced', order: 110, control: 'select', options: ['', 'packetaddr', 'xudp'] },
    global_padding: { tier: 'advanced', order: 120 },
    authenticated_length: { tier: 'advanced', order: 130 },
  },

  vless: {
    flow: { tier: 'typical', order: 41, control: 'select', options: ['', 'xtls-rprx-vision'] },
    packet_encoding: { tier: 'advanced', order: 100, control: 'select', options: ['', 'packetaddr', 'xudp'] },
  },

  trojan: {},

  hysteria: {
    // Hysteria v1 authenticates with auth_str and REQUIRES both bandwidths;
    // the old form offered none of the three.
    auth_str: { tier: 'core', order: 30, control: 'password' },
    up_mbps: { tier: 'core', order: 33, control: 'number' },
    down_mbps: { tier: 'core', order: 34, control: 'number' },
    obfs: { tier: 'advanced', order: 100 },
    disable_mtu_discovery: { tier: 'advanced', order: 110 },
  },

  hysteria2: {
    up_mbps: { tier: 'typical', order: 41, control: 'number' },
    down_mbps: { tier: 'typical', order: 42, control: 'number' },
    obfs: { tier: 'advanced', order: 100, control: 'json' },
    brutal_debug: { tier: 'advanced', order: 110 },
  },

  tuic: {
    congestion_control: {
      tier: 'advanced',
      order: 100,
      control: 'select',
      options: ['cubic', 'new_reno', 'bbr'],
    },
    udp_relay_mode: { tier: 'advanced', order: 110, control: 'select', options: ['native', 'quic'] },
    zero_rtt_handshake: { tier: 'advanced', order: 120 },
    heartbeat: { tier: 'advanced', order: 130 },
  },

  shadowtls: {
    version: { tier: 'core', order: 33, control: 'number', default: 3, placeholder: '3' },
  },

  anytls: {
    idle_session_timeout: { tier: 'advanced', order: 100 },
    idle_session_check_interval: { tier: 'advanced', order: 110 },
    min_idle_session: { tier: 'advanced', order: 120, control: 'number' },
  },

  shadowsocksr: {
    protocol: { tier: 'core', order: 33 },
    obfs: { tier: 'core', order: 34 },
    protocol_param: { tier: 'advanced', order: 100 },
    obfs_param: { tier: 'advanced', order: 110 },
  },

  wireguard: {
    private_key: { tier: 'core', order: 30, control: 'password' },
    peer_public_key: { tier: 'core', order: 31, control: 'password' },
    local_address: { tier: 'core', order: 32, control: 'chips', placeholder: '10.0.0.2/32' },
    pre_shared_key: { tier: 'advanced', order: 100, control: 'password' },
    peers: { tier: 'advanced', order: 110, control: 'json' },
    mtu: { tier: 'advanced', order: 120, control: 'number' },
    reserved: { tier: 'advanced', order: 130, control: 'chips' },
  },

  ssh: {
    user: { tier: 'core', order: 30 },
    private_key: { tier: 'typical', order: 40, control: 'password' },
    private_key_path: { tier: 'advanced', order: 100 },
    private_key_passphrase: { tier: 'advanced', order: 110, control: 'password' },
    host_key: { tier: 'advanced', order: 120, control: 'chips' },
    client_version: { tier: 'advanced', order: 130 },
  },

  tor: {
    executable_path: { tier: 'advanced', order: 100 },
    data_directory: { tier: 'advanced', order: 110 },
    torrc: { tier: 'advanced', order: 120, control: 'json' },
    extra_args: { tier: 'advanced', order: 130, control: 'chips' },
  },

  // ── Groups ───────────────────────────────────────────────────────────────
  // These do not dial: they pick among other outbounds by tag. `outbounds` is
  // the whole point of the type, and sing-box refuses to start a group with an
  // empty list ("missing tags") — at START, past where `sing-box check` looks.
  selector: {
    outbounds: { tier: 'core', order: 10, control: 'outbound-list' },
    // Must be one of `outbounds`; sing-box errors at start with
    // "default outbound not found" otherwise. Rendered as a picker over the
    // current selection rather than free text — see SchemaFieldControl.
    default: { tier: 'typical', order: 20, control: 'outbound-member' },
    interrupt_exist_connections: { tier: 'advanced', order: 100 },
  },

  urltest: {
    outbounds: { tier: 'core', order: 10, control: 'outbound-list' },
    url: { tier: 'typical', order: 20, placeholder: 'https://www.gstatic.com/generate_204' },
    interval: { tier: 'typical', order: 30, placeholder: '3m' },
    // Writable by the node-rules engine but not, until now, by this form.
    tolerance: { tier: 'advanced', order: 100, control: 'number' },
    idle_timeout: { tier: 'advanced', order: 110 },
    interrupt_exist_connections: { tier: 'advanced', order: 120 },
  },
}

const schema = createSchema<OutboundTypeName>({
  inventory: OUTBOUND_INVENTORY,
  labelPrefix: 'outbounds.form.fields',
  shared: SHARED as Record<string, FieldCuration>,
  byType: BY_TYPE as Partial<Record<OutboundTypeName, Record<string, FieldCuration>>>,
})

export const resolveOutboundFields = schema.resolveFields
export const pruneForType = schema.pruneForType
export const applyTypeDefaults = schema.applyTypeDefaults
export const isOutboundType = schema.isKnownType

/**
 * Types that group other outbounds rather than dialing one.
 *
 * Derived from the inventory rather than hand-listed. The old form carried
 * three copies of this classification (a predicate, a badge colour, a save
 * branch) and the backend a fourth in `config.IsOutboundGroupType`.
 */
export const OUTBOUND_GROUP_TYPES = (
  Object.keys(OUTBOUND_INVENTORY) as OutboundTypeName[]
).filter((type) => 'outbounds' in OUTBOUND_INVENTORY[type])

/** Types that dial an upstream and therefore need a server address. */
export const OUTBOUND_TYPES_NEEDING_SERVER = (
  Object.keys(OUTBOUND_INVENTORY) as OutboundTypeName[]
).filter((type) => 'server' in OUTBOUND_INVENTORY[type])

export { OUTBOUND_INVENTORY, OUTBOUND_TYPE_NOTES }
export type { OutboundTypeName }
export { OUTBOUND_TYPE_NAMES } from './outboundInventory.generated'
