/**
 * DNS server field curation — the editorial layer over the generated inventory.
 *
 * Mirrors `inboundFields.ts`; the generic machinery lives in `optionSchema.ts`.
 * See that file for the tier model and why uncurated fields stay reachable.
 *
 * WHAT THE OLD FORM COULD EDIT
 * ────────────────────────────
 * Almost nothing. `DNSServers.vue` had three capability predicates driving a
 * handful of `v-if` blocks, so a `tls` server could set server/port/detour and
 * NOT its `tls` block (server_name, insecure, alpn); `https` could not set
 * `method` or `headers`; and `local` rendered nothing at all beyond tag and
 * type. No dialer option — bind_interface, connect_timeout, domain_resolver,
 * fallback_delay — was reachable for any type.
 *
 * Two of those predicates, `needsServerAddress` and `supportsDetour`, were
 * byte-identical lists under different names. Both are now the inventory: a
 * type has `server` if its option struct embeds DNSServerAddressOptions, and
 * `detour` if it embeds DialerOptions, which is exactly what the generator
 * reflects.
 *
 * THERE IS NO LEGACY SHAPE HERE
 * ─────────────────────────────
 * sing-box parses an untyped `{tag, address: "tls://1.1.1.1"}` server into
 * LegacyDNSServerOptions and immediately upgrades it to a typed one
 * (option/dns.go:165-172), and the panel reads config through that same path.
 * So a legacy server is already typed by the time the UI sees it, and the form
 * never needs to render an `address` field. `TestLegacyDNSServerUpgrades` pins
 * this.
 */
import {
  DNS_SERVER_INVENTORY,
  type DNSServerFieldKey,
  type DNSServerTypeName,
} from './dnsServerInventory.generated'
import { createSchema, type FieldCuration } from './optionSchema'
import { NETWORK_STRATEGIES } from './vocabularies'

/** Every field name across every DNS server type — so the shared map cannot hold a typo. */
type AnyDNSServerFieldKey = {
  [T in DNSServerTypeName]: DNSServerFieldKey<T>
}[DNSServerTypeName]

/**
 * Curation applying to any type possessing the field.
 *
 * Keyed by field name and intersected with each type's real inventory rather
 * than spread into it: a spread bypasses TypeScript's excess-property check, so
 * `server` could be spread into `local` — which has no such field — and fail
 * silently.
 */
const SHARED: Partial<Record<AnyDNSServerFieldKey, FieldCuration>> = {
  // The upstream address. Present only on the remote transports, via the
  // embedded DNSServerAddressOptions — which is precisely the old
  // `needsServerAddress` predicate, now derived rather than hand-listed.
  server: { tier: 'core', order: 10, placeholder: '1.1.1.1' },
  server_port: { tier: 'typical', order: 20, control: 'number', placeholder: '53' },

  // Which outbound carries the query. Empty means direct. Worth promoting:
  // routing DNS through a proxy is the single most common reason to edit a
  // server after creating it.
  detour: { tier: 'typical', order: 30, control: 'outbound' },

  tls: { tier: 'advanced', order: 100, control: 'json' },
  domain_resolver: { tier: 'advanced', order: 110, control: 'json' },
  connect_timeout: { tier: 'advanced', order: 120 },
  fallback_delay: { tier: 'advanced', order: 130 },

  // See the matching block in outboundFields.ts: these reach every type that
  // embeds DialerOptions, and all three shipped as number inputs for what are
  // string enums.
  network_strategy: {
    tier: 'advanced',
    order: 140,
    control: 'select',
    options: NETWORK_STRATEGIES,
  },
  network_type: { tier: 'advanced', order: 150, control: 'chips' },
  fallback_network_type: { tier: 'advanced', order: 160, control: 'chips' },
}

/**
 * Per-type curation. Overrides SHARED for the same key.
 *
 * Typed as a mapped type over `DNSServerTypeName`, so each entry is checked
 * against that type's own field keys — naming a field the type does not have is
 * a compile error, not an input that never binds.
 */
const BY_TYPE: {
  [T in DNSServerTypeName]: Partial<Record<DNSServerFieldKey<T>, FieldCuration>>
} = {
  udp: {},
  tcp: {},
  tls: {},
  quic: {},

  https: {
    // sing-box defaults to /dns-query when omitted, but every provider
    // documents its own path, so it is worth showing rather than hiding.
    path: { tier: 'typical', order: 40, placeholder: '/dns-query' },
    method: { tier: 'advanced', order: 100, control: 'select', options: ['GET', 'POST'] },
    headers: { tier: 'advanced', order: 110, control: 'json' },
  },
  h3: {
    path: { tier: 'typical', order: 40, placeholder: '/dns-query' },
    method: { tier: 'advanced', order: 100, control: 'select', options: ['GET', 'POST'] },
    headers: { tier: 'advanced', order: 110, control: 'json' },
  },

  // `local` resolves through the host's own resolver and has no upstream
  // address at all — every field it owns is a dialer option. It is therefore
  // the one type with no core or typical field, and the editor shows an
  // explicit empty state rather than a bare "add field" row.
  local: {},

  hosts: {
    predefined: { tier: 'core', order: 10, control: 'hosts' },
    path: { tier: 'typical', order: 20, control: 'chips', placeholder: '/etc/hosts' },
  },

  fakeip: {
    // sing-box's documented defaults. Seeded because a fakeip server with no
    // range is valid to parse and useless to run.
    inet4_range: { tier: 'core', order: 10, default: '198.18.0.0/15', placeholder: '198.18.0.0/15' },
    inet6_range: { tier: 'core', order: 20, default: 'fc00::/18', placeholder: 'fc00::/18' },
  },

  dhcp: {
    // Optional on purpose: sing-box auto-detects the interface when empty, so
    // this is a narrowing knob rather than a required field.
    interface: { tier: 'typical', order: 10, placeholder: 'auto' },
  },

  tailscale: {
    endpoint: { tier: 'core', order: 10, placeholder: 'ts-endpoint' },
    accept_default_resolvers: { tier: 'typical', order: 20 },
  },
}

const schema = createSchema<DNSServerTypeName>({
  inventory: DNS_SERVER_INVENTORY,
  labelPrefix: 'dns.servers.form.fields',
  shared: SHARED as Record<string, FieldCuration>,
  byType: BY_TYPE as Partial<Record<DNSServerTypeName, Record<string, FieldCuration>>>,
})

export const resolveDNSServerFields = schema.resolveFields
export const pruneForType = schema.pruneForType
export const applyTypeDefaults = schema.applyTypeDefaults
export const isDNSServerType = schema.isKnownType

/**
 * Types that cannot work without an upstream address.
 *
 * Derived from the inventory rather than hand-listed, so it cannot drift from
 * what sing-box actually accepts — this replaces the `needsServerAddress`
 * constant array.
 */
export const DNS_TYPES_NEEDING_SERVER = (
  Object.keys(DNS_SERVER_INVENTORY) as DNSServerTypeName[]
).filter((type) => 'server' in DNS_SERVER_INVENTORY[type])

export { DNS_SERVER_INVENTORY }
export type { DNSServerTypeName }
export { DNS_SERVER_TYPE_NAMES } from './dnsServerInventory.generated'
