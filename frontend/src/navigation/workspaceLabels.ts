import { createMenu } from './menu'
const children: Record<string, string> = {
  'dns/groups': 'lab.resolverGroups', 'dns/servers': 'dns.tabs.servers',
  'dns/rules': 'dns.tabs.rules', 'dns/settings': 'dns.tabs.settings',
  'route/policies': 'lab.trafficPolicies', 'route/rules': 'route.tabs.rules',
  'route/rule-sets': 'route.tabs.ruleSets', 'route/final-policy': 'route.tabs.finalPolicy',
  'diagnostics/dual-stack': 'diagnostics.tabs.dualStack', 'diagnostics/dns': 'diagnostics.tabs.dns',
  'diagnostics/route': 'diagnostics.tabs.route', 'experimental/cache-file': 'experimental.tabs.cacheFile',
  'experimental/clash-api': 'experimental.tabs.clashApi', 'experimental/v2ray-api': 'experimental.tabs.v2rayApi',
  'lab/functionality': 'nav.labFunctionality', 'lab/theme': 'nav.labTheme',
  profile: 'nav.profile', log: 'nav.logs',
}
export function workspaceLabel(path: string, t: (key: string) => string) {
  const key = children[path.replace('/dashboard/', '')]
  if (path === '/dashboard/dns/settings') return t('nav.dns') + ' · ' + t('dns.tabs.settings')
  if (path.startsWith('/dashboard/diagnostics/') && key) return t('nav.diagnostics') + ' · ' + t(key)
  if (key) return t(key)
  return createMenu(t).flatMap(group => group.items).find(item => item.path === path)?.name ?? path.split('/').pop() ?? path
}
