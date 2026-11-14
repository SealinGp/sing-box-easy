/** Browser-path measurements, deliberately independent of the managed server. */
export const latencyTargets = [
  { name: 'Baidu', url: 'https://apps.bdimg.com/favicon.ico' },
  { name: 'Cloudflare', url: 'https://www.cloudflare.com/favicon.ico' },
  { name: 'GitHub', url: 'https://github.githubassets.com/favicon.ico' },
  { name: 'YouTube', url: 'https://yt3.ggpht.com/favicon.ico' },
] as const
export const LATENCY_ROUNDS = 10

export function latencyStats(samples: readonly (number | null)[]) {
  const values = samples.filter((v): v is number => v !== null && Number.isFinite(v) && v >= 0)
  return values.length ? {
    min: Math.min(...values),
    avg: Math.round(values.reduce((a, b) => a + b, 0) / values.length),
    max: Math.max(...values),
  } : null
}

// Image loading avoids CORS response-reading restrictions. This is elapsed
// HTTP/image time (including connection setup), not ICMP or core proxy delay.
export function measureImage(url: string, signal: AbortSignal, timeout = 5000): Promise<number | null> {
  return new Promise(resolve => {
    if (signal.aborted) { resolve(null); return }
    const img = new Image()
    const start = performance.now()
    const finish = (value: number | null) => {
      clearTimeout(timer)
      img.onload = img.onerror = null
      signal.removeEventListener('abort', abort)
      img.removeAttribute('src')
      resolve(value)
    }
    const abort = () => finish(null)
    const timer = setTimeout(abort, timeout)
    signal.addEventListener('abort', abort, { once: true })
    img.onload = () => finish(Math.max(1, Math.round(performance.now() - start)))
    img.onerror = abort
    img.referrerPolicy = 'no-referrer'
    img.src = `${url}?_=${Date.now()}-${Math.random().toString(36).slice(2)}`
  })
}

export const ipProviders = [
  { id: 'ipip', name: 'IPIP.net', url: 'https://myip.ipip.net/json' },
  { id: 'ipsb', name: 'IP.sb', url: 'https://api.ip.sb/geoip' },
] as const
export type IPProvider = typeof ipProviders[number]
export interface NetworkInfo { ip: string; location: string; organization: string }

export function parseNetworkInfo(provider: IPProvider['id'], value: unknown): NetworkInfo {
  if (!value || typeof value !== 'object') throw new Error('Invalid response')
  const body = value as Record<string, unknown>
  const text = (v: unknown) => typeof v === 'string' ? v : ''
  let result: NetworkInfo
  if (provider === 'ipip') {
    const data = body.data as Record<string, unknown> | undefined
    if (body.ret !== 'ok' || !data) throw new Error('Lookup failed')
    const location = Array.isArray(data.location) ? data.location.map(text) : []
    result = { ip: text(data.ip), location: [...new Set(location.slice(0, 3).filter(Boolean))].join(' · '), organization: location.slice(3).filter(Boolean).join(' · ') }
  } else {
    result = { ip: text(body.ip), location: [...new Set([body.country, body.region, body.city].map(text).filter(Boolean))].join(' · '), organization: text(body.organization) || text(body.asn_organization) }
  }
  // Validate IP syntax through the URL parser, without accepting a hostname.
  const ipv4 = /^(\d{1,3}\.){3}\d{1,3}$/.test(result.ip) && result.ip.split('.').every(v => Number(v) <= 255)
  let ipv6 = false
  if (result.ip.includes(':')) {
    try { ipv6 = new URL(`http://[${result.ip}]/`).hostname.startsWith('[') } catch { /* invalid */ }
  }
  if (!ipv4 && !ipv6) throw new Error('Invalid IP')
  return result
}

export async function lookupNetwork(provider: IPProvider, signal: AbortSignal): Promise<NetworkInfo> {
  const response = await fetch(provider.url, { signal, cache: 'no-store', credentials: 'omit', referrerPolicy: 'no-referrer' })
  if (!response.ok) throw new Error(`HTTP ${response.status}`)
  return parseNetworkInfo(provider.id, await response.json())
}
