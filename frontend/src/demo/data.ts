import type { DNSServer, RouteRule, Subscription } from '../types/api'
import type { RuntimeMember, RuntimeProxies } from '../types/runtime'
import type { ProbePoint } from '../types/subprobe'

export const now = Date.now()
export const subscriptions: Subscription[] = ['示例网络 · 亚太', '示例网络 · 环球'].map((name, i) => ({
  id: `sub_demo${i + 1}`, name, url: `https://example.com/subscription/${i + 1}`,
  auto_update: true, update_interval: '24h', last_update: new Date(now - 1800000).toISOString(),
  probe_enabled: true, info: [{ key: 'used', value: `${i ? 414 : 132} GB` }, { key: 'remaining', value: `${i ? 86 : 368} GB` }, { key: 'total', value: '500 GB' },
    { key: 'expires', value: new Date(now + 45 * 86400000).toISOString().slice(0, 10) }],
}))
const names = ['🇭🇰 香港 01 · 高速专线', '🇯🇵 日本 02 · 东京', '🇸🇬 新加坡 03', '🇺🇸 美国 01 · 洛杉矶', '🇩🇪 德国 02 · 法兰克福', '🇬🇧 英国 03 · 伦敦']
export const members: RuntimeMember[] = names.map((name, n) => {
  const count = n === 5 ? 24 : 100
  const history = Array.from({ length: count }, (_, i) => {
    const success = i % 37 !== n
    const delay = success ? Math.round(90 + n * 62 + Math.sin(i * .37 + n) * 55 + (i % 19 === 0 ? 260 : 0)) : 0
    return { time: new Date(now - (count - i) * 600000).toISOString(), delay, success, source: i % 9 ? 'subscription' : 'manual' }
  })
  return { name: `${name} | sub_demo${n < 3 ? 1 : 2}`, type: n % 2 ? 'Trojan' : 'VLESS', group: false,
    delay: history.at(-1)!.delay, history }
})
export const proxies: RuntimeProxies = { selection_persisted: true, groups: [
  { name: '🚀 代理', type: 'Selector', switchable: true, now: members[0]!.name, delay: members[0]!.delay, members },
  { name: '自动选择', type: 'URLTest', switchable: false, now: members[0]!.name, delay: members[0]!.delay, members },
  { name: '🤖 AI', type: 'Selector', switchable: true, now: members[3]!.name, delay: members[3]!.delay, members: members.slice(2) },
] }
export const outbounds = [
  { tag: 'direct', type: 'direct' },
  ...proxies.groups.map(g => ({ tag: g.name, type: g.switchable ? 'selector' : 'urltest', outbounds: g.members.map(m => m.name) })),
  ...members.map(m => ({ tag: m.name, type: m.type.toLowerCase(), server: 'node.example.com', server_port: 443 })),
]
export const rules: RouteRule[] = [
  { protocol: ['dns'], action: 'hijack-dns' },
  { ip_is_private: true, action: 'route', outbound: 'direct' },
  { domain_suffix: ['example.org', 'example.net'], action: 'route', outbound: '🤖 AI' },
  { network: ['tcp'], port: [443], domain_suffix: ['media.example.com'], action: 'route', outbound: '🚀 代理' },
  { domain_keyword: ['ads', 'tracking'], action: 'reject' },
  { rule_set: ['geosite-cn'], action: 'route', outbound: 'direct' },
  { network: ['tcp', 'udp'], action: 'route', outbound: '自动选择' },
]
export const servers: DNSServer[] = [
  { tag: '国内 DNS', type: 'https', server: 'dns.alidns.com', server_port: 443, path: '/dns-query' },
  { tag: '海外 DNS', type: 'https', server: 'dns.google', server_port: 443, path: '/dns-query', detour: '🚀 代理' },
  { tag: '备用 DNS', type: 'tls', server: '1.1.1.1', server_port: 853 },
]
export function probeHistory(id: string): ProbePoint[] {
  const nodes = members.filter(m => m.name.endsWith(id))
  return Array.from({ length: 100 }, (_, i) => {
    const samples = nodes.map(m => m.history?.[i - (100 - (m.history?.length ?? 0))]).filter(s => s && s.success)
    const delays = samples.map(s => s!.delay)
    return { at: new Date(now - (100 - i) * 600000).toISOString(), total: nodes.length, reachable: delays.length,
      avg_ms: delays.length ? Math.round(delays.reduce((a, b) => a + b, 0) / delays.length) : 0,
      min_ms: delays.length ? Math.min(...delays) : 0, max_ms: delays.length ? Math.max(...delays) : 0 }
  })
}
