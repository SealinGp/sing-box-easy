import { runtimeService, serviceControlService } from '../services'
import type { RuntimeMember, RuntimeProxies } from '../types/runtime'
import { Code, type BasicResponse } from '../types/api'

/** Opt-in Vite demo only. Never imported by a production build. */
export function installNodeQualityPreview() {
  const start = Date.now() - 100 * 60000
  const member = (name: string, delays: number[]): RuntimeMember => ({
    name, type: 'VLESS', group: false, delay: delays.at(-1) ?? 0,
    history: delays.map((delay, i) => ({ time: new Date(start + i * 10 * 60000).toISOString(), delay, source: i % 3 ? 'subscription' : 'manual' })),
  })
  const members = [
    member('香港 A', [120, 180, 0, 210, 600, 260, 180, 300, 200, 160]),
    member('新加坡 B', [120, 680, 0, 210, 900, 260, 180, 0, 200, 460]),
    member('日本 C', [900, 680, 0, 210, 900, 760, 180, 0, 800, 460]),
    member('少样本 D', [120, 180, 0, 210]),
    member('少样本 E', [120, 500]),
    member('新节点 F', []),
  ]
  const view: RuntimeProxies = {
    selection_persisted: true,
    groups: [
      { name: '演示 · 节点质量', type: 'Selector', now: members[0]!.name, delay: 160, switchable: true, members },
      { name: '演示 · 自动选择', type: 'URLTest', now: members[0]!.name, delay: 160, switchable: false, members },
    ],
  }
  const ok = <T>(data: T): BasicResponse<T> => ({ code: Code.Success, data, msg: 'mock' })
  serviceControlService.getServiceStatus = async () => ok({ status: 'running' })
  runtimeService.getProxies = async () => ok(structuredClone(view))
  runtimeService.getNodeHistory = async name => ok({ points: structuredClone(members.find(item => item.name === name)?.history ?? []) })
  runtimeService.select = async (group, name) => {
    const target = view.groups.find(item => item.name === group)
    if (target?.switchable) target.now = name
    return ok({})
  }
  runtimeService.testNode = async name => ok({ name, delay: members.find(item => item.name === name)?.delay ?? 0 })
  runtimeService.testGroup = async group => ok({ group, delays: Object.fromEntries(members.filter(item => item.delay > 0).map(item => [item.name, item.delay])) })

  // Keep the example discoverable while reviewing either real page.
  const banner = document.createElement('aside')
  banner.setAttribute('aria-label', 'Mock preview')
  banner.style.cssText = 'position:fixed;right:16px;bottom:16px;z-index:9000;max-width:calc(100vw - 32px);padding:10px 14px;border:1px solid #d97706;border-radius:12px;background:#fffbeb;color:#78350f;font:12px/1.6 system-ui;box-shadow:0 3px 15px #0002'
  banner.append(document.createTextNode('MOCK 节点 · 切换/测速仅模拟 · 设置保存到本地开发实例'))
  const links = document.createElement('div')
  for (const [text, path] of [['查看代理', '/dashboard/proxies?mock=node-quality'], ['调整质量标准', '/dashboard/settings?mock=node-quality#node-quality'], ['退出演示', '/dashboard/proxies']] as const) {
    const link = document.createElement('a')
    link.textContent = text
    link.href = path!
    link.style.cssText = 'display:inline-block;margin-right:14px;text-decoration:underline'
    links.append(link)
  }
  banner.append(links)
  document.body.append(banner)
}
