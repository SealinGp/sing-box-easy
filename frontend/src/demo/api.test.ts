import { afterAll, expect, test } from 'bun:test'
import { apiService } from '../services/api'
import { installDemoApi } from './api'
import { members, proxies } from './data'

const original = { get: apiService.get, post: apiService.post, put: apiService.put, delete: apiService.delete }
afterAll(() => Object.assign(apiService, original))

test('static demo serves copies, simulates local changes and rejects unsupported operations', async () => {
  installDemoApi()
  const response = await apiService.get<{ data: typeof proxies }>('/runtime/proxies')
  response.data.data.groups.length = 0
  expect(proxies.groups.length).toBe(3)
  await apiService.put('/runtime/proxies/selection', { group: '🚀 代理', name: members[1]!.name })
  expect(proxies.groups[0]!.now).toBe(members[1]!.name)
  await apiService.put('/runtime/proxies/selection', { group: '自动选择', name: members[1]!.name })
  expect(proxies.groups[1]!.now).toBe(members[0]!.name)
  await apiService.put('/runtime/proxies/auto-selection', { group: '🚀 代理', enabled: true, interval_seconds: 300 })
  const automated = await apiService.get<{ data: typeof proxies }>('/runtime/proxies')
  expect(automated.data.data.groups[0]!.auto_selection?.reason).toBe('retained')
  await apiService.put('/runtime/proxies/selection', { group: '🚀 代理', name: members[0]!.name })
  expect(proxies.groups[0]!.auto_selection?.enabled).toBe(false)
  await expect(apiService.put('/runtime/proxies/auto-selection', { group: '自动选择', enabled: true })).rejects.toThrow('Static demo')
  const quality = { samples: 100, max_delay_ms: 400, good_percent: 95, fair_percent: 50 }
  await apiService.put('/settings', { node_quality: quality })
  const settings = await apiService.get<{ data: { node_quality: typeof quality } }>('/settings')
  expect(settings.data.data.node_quality).toEqual(quality)
  await expect(apiService.post('/subscriptions/sub_demo1/update')).rejects.toThrow('Static demo')
  await expect(apiService.delete('/subscriptions/sub_demo1')).rejects.toThrow('Static demo')
  await expect(apiService.get('/unknown')).rejects.toThrow('Static demo')
  installDemoApi()
  const reset = await apiService.get<{ data: { node_quality: typeof quality } }>('/settings')
  expect(reset.data.data.node_quality.good_percent).toBe(97)
})
