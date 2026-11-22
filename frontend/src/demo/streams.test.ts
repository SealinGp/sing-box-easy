import { expect, test } from 'bun:test'
import { connectionFrame, demoFetch, trafficFrame } from './streams'
import { lookupNetwork, measureImage } from './browserNetwork'
import { ipProviders } from '../utils/browserNetwork'

test('overview streams provide initial traffic history and close on abort', async () => {
  const controller = new AbortController()
  const response = await demoFetch('/api/v1/runtime/connections/stream?rows=0', { signal: controller.signal })
  controller.abort()
  const text = await response.text()
  expect(text.match(/event: frame/g)?.length).toBe(60)
  const frame = JSON.parse(text.split('data: ')[1]!.split('\n')[0]!).data
  expect(frame.totals.connections).toBeGreaterThan(0)
  expect(frame.totals.download_total).toBeGreaterThan(0)
  const flow = await demoFetch('/api/v1/traffic/flow/stream', { signal: AbortSignal.abort() })
  expect(await flow.text()).toBe('')
  await expect(demoFetch('https://example.com/unknown')).rejects.toThrow('Static demo')
})

test('mock flow and totals agree and browser measurements stay synthetic', async () => {
  const at = 1700000000000
  const traffic = trafficFrame(at), connections = connectionFrame(at)
  expect(traffic.totals.down).toBe(connections.totals.down_rate)
  expect(traffic.exits[0]!.tag).toBe('自动选择')
  const signal = new AbortController().signal
  expect(await measureImage('https://example.com/favicon.ico', signal)).toBeGreaterThan(0)
  expect((await lookupNetwork(ipProviders[0], signal)).ip).toBe('192.0.2.10')
  expect(await measureImage('https://example.com/favicon.ico', AbortSignal.abort())).toBeNull()
})
