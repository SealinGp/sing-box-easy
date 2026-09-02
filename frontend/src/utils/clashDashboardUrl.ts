/**
 * Turning `experimental.clash_api` into a link a browser can actually open.
 *
 * Extracted from ClashAPISettings.vue when the Overview card started offering
 * the same link: two copies of these rules would drift, and the rules are the
 * whole value — the naive answer (link to `external_controller`) is wrong in
 * three separate ways.
 *
 * THE LINK CANNOT TARGET THE CLASH API ROOT WHEN A SECRET IS SET
 * ──────────────────────────────────────────────────────────────
 * sing-box guards every API route with `Authorization: Bearer <secret>`, and an
 * <a href> navigation cannot send headers — so `http://host:port` answers
 * `{"message":"Unauthorized"}`. The `?token=` query parameter does not help:
 * sing-box honours it only for WebSocket upgrades
 * (experimental/clashapi/server.go, the `Upgrade == "websocket"` branch).
 *
 * What IS reachable is the dashboard. sing-box mounts `/ui/*` in a separate
 * router group with NO authentication middleware, so it loads without a header
 * — and the dashboard then takes the secret from the URL and replays it as a
 * proper Authorization header on its own XHR calls.
 */

/** Hosts sing-box can bind but a browser cannot usefully dial. */
const UNROUTABLE_HOSTS = new Set(['', '0.0.0.0', '::', '[::]'])

export interface ControllerEndpoint {
  host: string
  port: string
}

/** Splits `host:port`, `:port`, or `[::1]:port` into parts. */
export function parseListenAddress(value: string): ControllerEndpoint {
  const bracketed = value.match(/^\[(.*)\]:(\d+)$/)
  if (bracketed) return { host: `[${bracketed[1]}]`, port: bracketed[2] ?? '' }
  const lastColon = value.lastIndexOf(':')
  if (lastColon === -1) return { host: value, port: '' }
  return { host: value.slice(0, lastColon), port: value.slice(lastColon + 1) }
}

/**
 * The address a browser should dial for a given `external_controller`.
 *
 * A wildcard bind (`0.0.0.0`, `::`) is reachable for sing-box but meaningless
 * to the browser, so it falls back to the host this page was loaded from — the
 * same substitution Inbounds.vue makes when it builds a client config.
 *
 * `fallbackHost` is a parameter rather than a `window` read so this stays
 * testable; callers pass `window.location.hostname`.
 */
export function resolveControllerEndpoint(
  externalController: string | undefined,
  fallbackHost: string,
): ControllerEndpoint | null {
  const raw = externalController?.trim()
  if (!raw) return null
  const { host, port } = parseListenAddress(raw)
  if (!port) return null
  return {
    host: UNROUTABLE_HOSTS.has(host) ? fallbackHost || '127.0.0.1' : host,
    port,
  }
}

export interface ClashLinkInput {
  external_controller?: string
  external_ui?: string
  secret?: string
}

export function hasDashboard(clash: ClashLinkInput): boolean {
  return !!clash.external_ui?.trim()
}

/**
 * The URL to open, or `''` when no working link exists.
 *
 * With a dashboard installed it targets `/ui/` and hands the backend details
 * over twice: both dashboards this project ships read them from the URL, but
 * they disagree on where — zashboard documents
 * `#/setup?hostname=…&port=…&secret=…` while yacd reads a plain query string.
 * Emitting both costs nothing and means the link auto-connects on either.
 */
export function clashDashboardHref(clash: ClashLinkInput, fallbackHost: string): string {
  const endpoint = resolveControllerEndpoint(clash.external_controller, fallbackHost)
  if (!endpoint) return ''

  const origin = `http://${endpoint.host}:${endpoint.port}`
  const secret = clash.secret ?? ''

  // No dashboard to route through: the bare root works only without a secret.
  if (!hasDashboard(clash)) return secret ? '' : origin

  const params = new URLSearchParams({ hostname: endpoint.host, port: endpoint.port })
  if (secret) params.set('secret', secret)
  // URLSearchParams encodes a space as `+`, which is only correct in a query
  // string. The same text is reused inside the hash fragment, where parsers
  // typically run decodeURIComponent and leave `+` as a literal plus — so a
  // secret containing a space would arrive corrupted. `%20` decodes correctly
  // in both positions.
  const query = params.toString().replace(/\+/g, '%20')

  return `${origin}/ui/?${query}#/setup?${query}`
}

/**
 * True when the controller is configured and secured but no dashboard exists to
 * open — the one case where no link is rendered at all, because every URL we
 * could produce would 401.
 */
export function isLinkBlockedBySecret(clash: ClashLinkInput, fallbackHost: string): boolean {
  return (
    !!resolveControllerEndpoint(clash.external_controller, fallbackHost) &&
    !hasDashboard(clash) &&
    !!clash.secret
  )
}
