/**
 * Human labels for sing-box field names.
 *
 * The generated inventory carries ~250 distinct field names across the inbound
 * types. Hand-translating all of them into en + zh would be a large, mostly
 * mechanical edit whose main effect is that the day sing-box adds a field, the
 * form renders the raw i18n path — `inbounds.form.fields.tcp_fast_open` — as
 * the label, because vue-i18n falls back to the key.
 *
 * So translations are opt-in: a field gets an entry under
 * `inbounds.form.fields.*` when a human label genuinely adds meaning over its
 * own name (`listen` → "Listen Address", `up_mbps` → "Upload Bandwidth"), and
 * everything else is humanized from the JSON key at render time. That keeps the
 * locale files to the fields worth explaining, and guarantees a new sing-box
 * field is legible on day one without touching this repo.
 */

/**
 * Segments that should not be title-cased into "Tcp" / "Uuid".
 *
 * Keyed lowercase, valued as the form to render. Only initialisms belong here —
 * ordinary words are handled by the generic path.
 */
const INITIALISMS: Record<string, string> = {
  tcp: 'TCP',
  udp: 'UDP',
  ip: 'IP',
  ipv4: 'IPv4',
  ipv6: 'IPv6',
  inet4: 'IPv4',
  inet6: 'IPv6',
  dns: 'DNS',
  tls: 'TLS',
  sni: 'SNI',
  mtu: 'MTU',
  nat: 'NAT',
  uuid: 'UUID',
  url: 'URL',
  id: 'ID',
  gso: 'GSO',
  mbps: 'Mbps',
  ttl: 'TTL',
  acme: 'ACME',
  ech: 'ECH',
  alpn: 'ALPN',
  rtt: 'RTT',
  os: 'OS',
  api: 'API',
  http: 'HTTP',
  https: 'HTTPS',
  quic: 'QUIC',
  ssh: 'SSH',
  uid: 'UID',
  netns: 'NetNS',
  iproute2: 'iproute2',
}

/**
 * "tcp_fast_open" → "TCP Fast Open", "up_mbps" → "Up Mbps".
 *
 * Also handles the one camelCase field name sing-box uses (`alterId`), which
 * would otherwise render as a single run-on word.
 */
export function humanizeFieldName(key: string): string {
  return key
    .replace(/([a-z0-9])([A-Z])/g, '$1_$2')
    .split(/[_\-.]/)
    .filter(Boolean)
    .map((segment) => {
      const lower = segment.toLowerCase()
      if (INITIALISMS[lower]) return INITIALISMS[lower]
      return lower.charAt(0).toUpperCase() + lower.slice(1)
    })
    .join(' ')
}
