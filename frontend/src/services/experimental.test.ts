import { expect, test } from 'bun:test'
import { ExperimentalService } from './experimental'
import type { ApiService } from './api'

test('inactive cache duration cannot invalidate saving and the draft is retained', async () => {
  const requests: unknown[] = []
  const api = { put: async (_url: string, payload: unknown) => { requests.push(payload); return { data: {} } } } as unknown as ApiService
  const service = new ExperimentalService(api)
  const draft = { enabled: true, store_rdrc: false, rdrc_timeout: 'unfinished' }
  await service.updateCacheFile(draft)
  expect(requests[0]).not.toHaveProperty('rdrc_timeout')
  expect(draft.rdrc_timeout).toBe('unfinished')
  await service.updateCacheFile({ ...draft, enabled: false, store_rdrc: true })
  expect(requests[1]).not.toHaveProperty('rdrc_timeout')
  await service.updateCacheFile({ ...draft, store_rdrc: true, rdrc_timeout: '7d' })
  expect(requests[2]).toHaveProperty('rdrc_timeout', '7d')
})
