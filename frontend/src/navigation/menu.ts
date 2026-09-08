import type { Component } from 'vue'
import { ChartBarIcon, ArrowDownTrayIcon, ArrowUpTrayIcon, GlobeAltIcon, MapIcon,
  DocumentTextIcon, BeakerIcon, QueueListIcon, UsersIcon, Cog6ToothIcon,
  ArrowPathIcon, AdjustmentsHorizontalIcon } from '@heroicons/vue/24/outline'

export interface MenuItem {
  name: string
  icon: Component
  path: string
  keywords?: string
}
export interface MenuGroup {
  id: string
  name: string
  items: MenuItem[]
}

export function createMenu(t: (key: string) => string, authEnabled: boolean): MenuGroup[] {
  const item = (key: string, icon: Component, path: string, keywords = ''): MenuItem =>
    ({ name: t(key), icon, path: '/dashboard/' + path, keywords })
  return [
    { id: 'monitor', name: t('nav.monitor'), items: [
      item('nav.overview', ChartBarIcon, 'overview', 'status service traffic 流量 状态'),
      item('nav.logs', QueueListIcon, 'logs', 'logs events 日志'),
    ] },
    { id: 'connections', name: t('nav.connections'), items: [
      item('nav.inbounds', ArrowDownTrayIcon, 'inbounds', 'listen tun 入站'),
      item('nav.outbounds', ArrowUpTrayIcon, 'outbounds/list', 'nodes proxies 出站 节点'),
      item('nav.subscriptions', ArrowPathIcon, 'outbounds/subscriptions', 'subscription feeds update 订阅'),
      item('nav.nodeRules', AdjustmentsHorizontalIcon, 'outbounds/node-rules', 'filters groups 节点规则 分组'),
    ] },
    { id: 'network', name: t('nav.network'), items: [
      item('nav.dns', GlobeAltIcon, 'dns', 'dns resolver 域名'),
      item('nav.route', MapIcon, 'route', 'routing rules 路由'),
    ] },
    { id: 'system', name: t('nav.system'), items: [
      item('nav.config', DocumentTextIcon, 'config', 'json configuration history 配置'),
      item('nav.experimental', BeakerIcon, 'experimental', 'clash v2ray cache api 实验'),
      item('nav.settings', Cog6ToothIcon, 'settings', 'update language github 设置'),
      ...(authEnabled ? [item('nav.users', UsersIcon, 'users', 'accounts 用户')] : []),
    ] },
  ]
}

export function isMenuActive(path: string, currentPath: string): boolean {
  return currentPath === path || currentPath.startsWith(path + '/')
}

export function searchMenu(groups: MenuGroup[], query: string) {
  const terms = query.toLocaleLowerCase().trim().split(/\s+/).filter(Boolean)
  return groups.flatMap(group => group.items.map(item => ({ ...item, group: group.name })))
    .filter(item => terms.every(term =>
      [item.name, item.group, item.path, item.keywords].join(' ').toLocaleLowerCase().includes(term)))
}
