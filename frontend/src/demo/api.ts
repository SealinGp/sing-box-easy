import { apiService } from '../services/api'
import { DEFAULT_NODE_QUALITY } from '../utils/nodeQuality'
import { members, outbounds, probeHistory, proxies, rules, servers, subscriptions, now } from './data'

const readonly = '静态演示：此操作需在安装后使用。 / Static demo: install the app to use this action.'
/** Install before mounting any real product component. No request reaches a backend. */
export function installDemoApi() {
  let settings = { node_quality: { ...DEFAULT_NODE_QUALITY } }
  const latest = () => Object.fromEntries(subscriptions.map(s => [s.id, probeHistory(s.id).at(-1)]))
  function read(url: string): unknown {
    const { pathname: path, searchParams: query } = new URL(url, 'https://demo.invalid')
    if (path === '/runtime/proxies') return proxies
    if (path === '/runtime/proxies/history') return { points: members.find(m => m.name === query.get('name'))?.history ?? [] }
    if (path === '/settings') return settings
    if (path === '/service/status') return { status: 'running' }
    if (path === '/outbounds') return { outbounds, endpoints: [], managed_tags: [] }
    if (path === '/subscriptions') return { subscriptions }
    if (path.startsWith('/subscriptions/')) return subscriptions.find(s => s.id === path.split('/')[2])
    if (path === '/subscription-probes/status') return { status: { running: true, in_flight: false, interval: '10m', last_run: new Date(now).toISOString() },
      latest: latest(), sample_count: 200, node_sample_count: 524, retention: 30, max_points: 500, timeout_ms: 5000, interval_secs: 600 }
    if (path.startsWith('/subscription-probes/')) {
      const [, , id, action] = path.split('/')
      const points = probeHistory(id!)
      if (action === 'history') return { id, range: query.get('range') ?? '24h', bucket_seconds: 0, points }
      if (action === 'nodes') return { available: true, at: new Date(now).toISOString(), sample: points.at(-1),
        results: members.filter(m => m.name.endsWith(id!)).map(m => ({ tag: m.name, delay_ms: m.delay, error: m.delay ? '' : 'timeout' })) }
    }
    if (path === '/settings/subscription-info-keywords') return { keywords: ['剩余流量', '到期时间'], defaults: ['剩余流量', '到期时间'], using_defaults: true }
    if (path === '/route/rules') return { rules }
    if (path === '/route/rule-sets' || path === '/templates/rule-sets') return { rule_sets: [{ tag: 'geosite-cn', type: 'remote', format: 'binary', url: 'https://example.com/geosite-cn.srs' }] }
    if (path === '/route/final') return { final: '🚀 代理', auto_detect_interface: true, default_domain_resolver: '国内 DNS' }
    if (path === '/dns/servers') return { servers }
    if (path === '/dns') return { servers, rules: [], final: '国内 DNS' }
    if (path === '/core') return { installed: true, version: '1.13.0', capabilities: {} }
    if (path === '/inbounds') return { inbounds: [] }
    if (path === '/config') return { outbounds, route: { rules, final: '🚀 代理' }, dns: { servers } }
    throw new Error(readonly)
  }
  function write(url: string, payload: unknown): unknown {
    const body = payload as Record<string, unknown> | undefined
    if (url === '/settings' && body?.node_quality) {
      settings = structuredClone(body) as typeof settings
      return settings
    }
    if (url === '/runtime/proxies/selection') {
      const group = proxies.groups.find(g => g.name === body?.group)
      if (group?.switchable && group.members.some(m => m.name === body?.name)) group.now = String(body?.name)
      return {}
    }
    if (url === '/runtime/proxies/delay') {
      const member = members.find(m => m.name === body?.name)
      if (!member) throw new Error(readonly)
      member.history?.push({ time: new Date().toISOString(), delay: member.delay, success: member.delay > 0, source: 'manual' })
      return { name: member.name, delay: member.delay }
    }
    if (url === '/runtime/proxies/group-delay') return { group: body?.name, delays: Object.fromEntries(members.map(m => [m.name, m.delay])) }
    throw new Error(readonly)
  }
  const wrap = <T>(data: unknown) => ({ data: { code: 0, msg: 'demo', data: structuredClone(data) } as T })
  apiService.get = async <T>(url: string) => wrap<T>(read(url))
  apiService.post = async <T>(url: string, payload?: unknown) => wrap<T>(write(url, payload))
  apiService.put = async <T>(url: string, payload?: unknown) => wrap<T>(write(url, payload))
  apiService.delete = async () => { throw new Error(readonly) }
}
