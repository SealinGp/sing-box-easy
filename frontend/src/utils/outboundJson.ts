import { isOutboundType } from '../schemas/outboundFields'

/** Preserve native options; only remove the legacy copy helper's VMess label. */
export function parseOutboundJson(input: string): Record<string, unknown> {
  const value: unknown = JSON.parse(input)
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('outboundJson.objectRequired')
  const node = { ...value } as Record<string, unknown>
  if (typeof node.type !== 'string' || !isOutboundType(node.type)) throw new Error('outboundJson.typeRequired')
  if ('listen' in node || 'listen_port' in node || 'users' in node) throw new Error('outboundJson.inbound')
  if (!['socks', 'http', 'naive'].includes(node.type)) delete node.username
  if (!node.tag) node.tag = `${node.type}-${node.server || 'out'}`
  return node
}
