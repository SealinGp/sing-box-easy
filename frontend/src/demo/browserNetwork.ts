/** Vite demo alias: keep the real cards, replace their browser measurements. */
export { LATENCY_ROUNDS, latencyTargets, latencyStats, ipProviders } from '../utils/browserNetwork.ts'
import type { IPProvider, NetworkInfo } from '../utils/browserNetwork.ts'
let round = 0
export async function measureImage(url: string, signal: AbortSignal): Promise<number | null> {
  if (signal.aborted) return null
  const base = url.includes('bdimg') ? 35 : url.includes('cloudflare') ? 110 : url.includes('github') ? 160 : 210
  return base + (++round * 17 % 53)
}
export async function lookupNetwork(provider: IPProvider, signal: AbortSignal): Promise<NetworkInfo> {
  if (signal.aborted) throw new DOMException('Aborted', 'AbortError')
  return { ip: provider.id === 'ipip' ? '192.0.2.10' : '203.0.113.25',
    location: provider.id === 'ipip' ? '示例地区 · Example Region' : '示例出口 · Example Exit',
    organization: 'Example Network · MOCK' }
}
