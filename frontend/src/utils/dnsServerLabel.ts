import type { DNSServer } from '../types/api'

/**
 * Human-readable labels for a DNS server, used wherever one is picked by tag.
 *
 * A bare tag ("dns_router") says nothing about where the query actually goes,
 * which is exactly what someone wiring up a DNS rule needs to know. These
 * helpers add the address and the transport, so the option reads
 * "dns_router — 192.168.9.2:53 (udp)".
 */

// The DNS server union has a different address shape per type (`server` for the
// remote transports, `address` for legacy, none at all for local/hosts/fakeip).
// Reading through one optional-property view keeps the helpers free of a
// type-guard ladder that would have to be updated for every new server type.
type AddressableServer = {
  server?: string
  server_port?: number
  address?: string
  interface?: string
  inet4_range?: string
}

/**
 * Where this server sends queries, or '' when the type has no address at all
 * (local, hosts, fakeip without a configured range).
 *
 * The port is shown only when the config actually sets one — printing an
 * implied 53/853/443 would claim something the config does not say. IPv6 hosts
 * are bracketed so "::1:53" can't be misread.
 */
export function dnsServerAddress(server: DNSServer): string {
  const s = server as AddressableServer

  if (s.server) {
    const host = s.server.includes(':') ? `[${s.server}]` : s.server
    return s.server_port ? `${host}:${s.server_port}` : host
  }
  // Legacy servers carry a full URL-ish address ("tls://8.8.8.8").
  if (s.address) return s.address
  if (server.type === 'dhcp' && s.interface) return s.interface
  if (server.type === 'fakeip' && s.inet4_range) return s.inet4_range
  return ''
}

/**
 * Option label for a DNS server select: "<tag> — <address> (<type>)", falling
 * back to "<tag> (<type>)" for the address-less types.
 */
export function dnsServerOptionLabel(server: DNSServer): string {
  const type = server.type || 'legacy'
  const address = dnsServerAddress(server)
  return address ? `${server.tag} — ${address} (${type})` : `${server.tag} (${type})`
}
